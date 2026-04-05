package main

import (
	"encoding/json"
	"fmt"
	"io"
	"net"
	"os"
	"os/exec"
	"path/filepath"
	"sync"

	"github.com/ccdevkit/ccbox/internal/args"
	"github.com/ccdevkit/ccbox/internal/bridge"
	"github.com/ccdevkit/ccbox/internal/claude"
	"github.com/ccdevkit/ccbox/internal/claude/hooks"
	"github.com/ccdevkit/ccbox/internal/clipboard"
	"github.com/ccdevkit/ccbox/internal/cmdpassthrough"
	"github.com/ccdevkit/ccbox/internal/constants"
	"github.com/ccdevkit/ccbox/internal/docker"
	"github.com/ccdevkit/ccbox/internal/logger"
	"github.com/ccdevkit/ccbox/internal/permissions"
	"github.com/ccdevkit/ccbox/internal/session"
	"github.com/ccdevkit/ccbox/internal/settings"
	"github.com/ccdevkit/ccbox/internal/terminal"
)

// --- Dependency interfaces for testability ---

// DockerChecker checks if Docker is available.
type DockerChecker interface {
	CheckRunning() error
}

// TokenCapturer captures an OAuth token from the claude CLI.
type TokenCapturer interface {
	CaptureToken(claudePath string) (string, error)
}

// VersionDetector detects the claude CLI version.
type VersionDetector interface {
	DetectVersion(claudePath string) (string, error)
}

// ImageEnsurer ensures the correct Docker image exists.
type ImageEnsurer interface {
	EnsureLocalImage(ccboxVersion, claudeVersion string, pinned bool) error
}

// ContainerRunner runs a Docker container and returns exit code.
type ContainerRunner interface {
	RunContainer(spec *docker.ContainerSpec) (int, error)
}

// ImageCleaner removes ccbox-managed Docker images.
type ImageCleaner interface {
	CleanImages() error
}

// BridgeServer manages the TCP bridge server lifecycle.
type BridgeServer interface {
	Start() (int, error)
	Stop() error
}

// BridgeServerFactory creates a BridgeServer with the given exec and hook handlers.
type BridgeServerFactory func(execHandler bridge.ExecHandler, hookHandler bridge.HookHandler) BridgeServer

// orchestrationDeps holds all injectable dependencies for the main orchestration.
type orchestrationDeps struct {
	dockerChecker      DockerChecker
	tokenCapture       TokenCapturer
	versionDetect      VersionDetector
	imageEnsurer       ImageEnsurer
	containerRunner    ContainerRunner
	bridgeServer       BridgeServer       // pre-built bridge server (used if bridgeServerFactory is nil)
	bridgeServerFactory BridgeServerFactory // factory to create bridge server with exec handler
	ccboxVersion       string
	log                *logger.Logger
	fs                 args.FileSystem
}

// runOrchestration implements the main orchestration flow.
// Returns the process exit code.
func runOrchestration(parsed *args.ParsedArgs, deps *orchestrationDeps) int {
	log := deps.log
	if log == nil {
		log, _ = logger.New(false, "")
	}

	// Step 1: Check Docker is running.
	if err := deps.dockerChecker.CheckRunning(); err != nil {
		fmt.Fprintf(os.Stderr, "ccbox: %v\n", err)
		return 1
	}

	// Step 2: Load settings and merge with CLI flags.
	cfg, err := settings.Load()
	if err != nil {
		fmt.Fprintf(os.Stderr, "ccbox: loading settings: %v\n", err)
		return 1
	}
	merged := settings.MergeWithCLI(cfg, parsed.Passthrough, parsed.ClaudePath, parsed.Verbose, parsed.LogFile)

	claudePath := merged.ClaudePath
	if claudePath == "" {
		claudePath = constants.DefaultClaudePath
	}

	// Step 3: Parallel — CaptureToken + DetectVersion.
	// If --use was specified, use that as claudeVersion; otherwise detect from host.
	var (
		token, claudeVersion string
		tokenErr, versionErr error
		wg                   sync.WaitGroup
	)

	wg.Add(1)
	go func() {
		defer wg.Done()
		token, tokenErr = deps.tokenCapture.CaptureToken(claudePath)
	}()

	if parsed.Use != "" {
		claudeVersion = parsed.Use
		log.Debug("orchestrate", "using specified claude version: %s", claudeVersion)
	} else {
		wg.Add(1)
		go func() {
			defer wg.Done()
			claudeVersion, versionErr = deps.versionDetect.DetectVersion(claudePath)
		}()
	}
	wg.Wait()

	if tokenErr != nil {
		fmt.Fprintf(os.Stderr, "ccbox: capturing auth token: %v\n", tokenErr)
		return 1
	}
	if versionErr != nil {
		fmt.Fprintf(os.Stderr, "ccbox: detecting claude version: %v\n", versionErr)
		return 1
	}

	log.RegisterSecret(token)
	log.Debug("orchestrate", "claude version: %s, token: %s", claudeVersion, maskToken(token))

	// Step 4: EnsureLocalImage.
	pinned := parsed.Use != ""
	if err := deps.imageEnsurer.EnsureLocalImage(deps.ccboxVersion, claudeVersion, pinned); err != nil {
		fmt.Fprintf(os.Stderr, "ccbox: ensuring docker image: %v\n", err)
		return 1
	}
	log.Debug("orchestrate", "image ensured")

	// Step 5: Create providers and session.
	tempDir, err := session.NewTempDirProvider("session")
	if err != nil {
		fmt.Fprintf(os.Stderr, "ccbox: creating temp dir: %v\n", err)
		return 1
	}
	defer tempDir.Cleanup()

	bindMount := session.NewDockerBindMountProvider()
	sess := session.NewSession(tempDir, bindMount)
	log.Debug("orchestrate", "session created: %s", sess.ID)

	// Step 6: claude.New — writes settings.json + .claude.json.
	c, err := claude.New(sess)
	if err != nil {
		fmt.Fprintf(os.Stderr, "ccbox: initializing claude: %v\n", err)
		return 1
	}
	c.Token = token

	// Step 6b: Create hook registry and register built-in handlers.
	registry := hooks.NewRegistry()
	hooks.RegisterBypassPermissions(registry)
	c.Registry = registry
	c.SetLogger(log)
	log.Debug("orchestrate", "claude initialized with hook registry, registered events: %v", registry.RegisteredEvents())

	// Step 7: Command passthrough setup — create live-reloading permission checker.
	// The checker automatically re-reads config files every ~1s so changes
	// to .ccbox/permissions.{json,yml,yaml} take effect without restart.
	checker, err := permissions.NewLiveChecker(cmdpassthrough.Merge(merged.Passthrough))
	if err != nil {
		fmt.Fprintf(os.Stderr, "ccbox: creating permission checker: %v\n", err)
		return 1
	}

	var ptCommands []string
	if checker != nil {
		ptCommands = checker.Commands()
	}
	log.Debug("orchestrate", "passthrough commands: %v", ptCommands)

	// Step 8: Start bridge TCP server (needed for passthrough port in proxy config).
	execHandler := cmdpassthrough.NewPermissionAwareHandler(checker)

	var srv BridgeServer
	hookHandler := registry.BridgeHandler()
	log.Debug("orchestrate", "hook handler created from registry (non-nil: %v)", hookHandler != nil)
	if deps.bridgeServerFactory != nil {
		srv = deps.bridgeServerFactory(execHandler, hookHandler)
	} else {
		srv = deps.bridgeServer
	}
	bridgePort, err := srv.Start()
	if err != nil {
		fmt.Fprintf(os.Stderr, "ccbox: starting bridge server: %v\n", err)
		return 1
	}
	defer srv.Stop()
	log.Debug("orchestrate", "bridge server started on port %d", bridgePort)

	if len(ptCommands) > 0 {
		proxyCfg := cmdpassthrough.ProxyConfig{
			HostAddress: fmt.Sprintf("%s:%d", constants.DockerHostname, bridgePort),
			Passthrough: ptCommands,
			Verbose:     merged.Verbose,
		}
		if err := cmdpassthrough.WriteProxyConfig(sess, proxyCfg); err != nil {
			fmt.Fprintf(os.Stderr, "ccbox: writing proxy config: %v\n", err)
			return 1
		}
		c.SetPassthroughEnabled(ptCommands)
	}

	// Step 8b: Setup clipboard support (stdin interception, bridge dir, clip port).
	clip := setupClipboard(log)
	if clip != nil {
		log.Debug("orchestrate", "clipboard enabled, port=%s bridge=%s", clip.clipPort, clip.bridgeDir)
	} else {
		log.Debug("orchestrate", "clipboard not available")
	}

	// Step 9: BuildRunSpec.
	runSpec, err := c.BuildRunSpec(parsed, merged, deps.fs)
	if err != nil {
		fmt.Fprintf(os.Stderr, "ccbox: building run spec: %v\n", err)
		return 1
	}
	log.Debug("orchestrate", "run spec built: args=%v, env count=%d", runSpec.Args, len(runSpec.Env))

	// Dump finalized settings hooks for debugging.
	if c.SettingsManager != nil {
		if hooksVal, ok := c.SettingsManager.Merged()["hooks"]; ok {
			if hooksJSON, err := json.Marshal(hooksVal); err == nil {
				log.Debug("orchestrate", "settings hooks config: %s", string(hooksJSON))
			}
		} else {
			log.Debug("orchestrate", "WARNING: no hooks in finalized settings")
		}
	}

	// Step 10: Build ContainerSpec.
	imageName := docker.LocalImageName(deps.ccboxVersion, claudeVersion)
	if pinned {
		imageName = docker.PinnedImageName(deps.ccboxVersion, claudeVersion)
	}

	spec := buildContainerSpec(imageName, runSpec, tempDir, bindMount, bridgePort, clip)
	log.Debug("orchestrate", "container spec: image=%s, mounts=%d, env=%d, args=%v",
		spec.ImageName, len(spec.Mounts), len(spec.Env), spec.Args)

	// Step 11: Run container.
	dockerArgs := docker.BuildDockerArgs(spec)
	log.Debug("orchestrate", "docker args: %v", dockerArgs)
	log.Debug("orchestrate", "starting container...")
	exitCode, err := deps.containerRunner.RunContainer(spec)
	if err != nil {
		fmt.Fprintf(os.Stderr, "ccbox: running container: %v\n", err)
		return 1
	}
	log.Debug("orchestrate", "container exited with code %d", exitCode)

	return exitCode
}

// buildContainerSpec assembles a docker.ContainerSpec from the various sources.
func buildContainerSpec(
	imageName string,
	runSpec *claude.ClaudeRunSpec,
	tempDir *session.TempDirProvider,
	bindMount *session.DockerBindMountProvider,
	bridgePort int,
	clip *clipboardResult,
) *docker.ContainerSpec {
	spec := &docker.ContainerSpec{
		ImageName: imageName,
	}

	// Map claude env vars to docker env vars.
	for _, ev := range runSpec.Env {
		spec.Env = append(spec.Env, docker.EnvVar{
			Key:    ev.Key,
			Value:  ev.Value,
			Secret: ev.Secret,
		})
	}

	// Add bridge port and display env vars.
	spec.Env = append(spec.Env,
		docker.EnvVar{Key: constants.EnvCCBoxTCPPort, Value: fmt.Sprintf("%d", bridgePort)},
		docker.EnvVar{Key: constants.EnvDisplay, Value: constants.DefaultDisplay},
	)

	// Clipboard support: env var, port mapping, bridge mount, stdin interceptor.
	if clip != nil {
		spec.Env = append(spec.Env,
			docker.EnvVar{Key: constants.EnvCCBoxClipPort, Value: clip.clipPort},
		)
		// Port mapping so the host can reach ccclipd inside the container.
		portNum := 0
		fmt.Sscanf(clip.clipPort, "%d", &portNum)
		if portNum > 0 {
			spec.Ports = append(spec.Ports, docker.PortMapping{
				Host:      portNum,
				Container: portNum,
			})
		}
		// Bridge directory mount for file drag-drop.
		spec.Mounts = append(spec.Mounts, docker.Mount{
			Host:      clip.bridgeDir,
			Container: constants.ContainerBridgeDir,
			ReadOnly:  false,
		})
		spec.StdinInterceptor = clip.stdinInterceptor
	}

	// Session file mounts.
	for _, f := range tempDir.Files {
		spec.Mounts = append(spec.Mounts, docker.Mount{
			Host:      f.HostPath,
			Container: f.ContainerPath,
			ReadOnly:  f.ReadOnly,
		})
	}

	// File passthrough mounts.
	for _, pt := range bindMount.Passthroughs {
		spec.Mounts = append(spec.Mounts, docker.Mount{
			Host:      pt.HostPath,
			Container: pt.ContainerPath,
			ReadOnly:  pt.ReadOnly,
		})
	}

	// Working directory and args from ClaudeRunSpec.
	spec.WorkDir = runSpec.WorkDir
	spec.Args = runSpec.Args

	return spec
}

// clipboardResult holds the outputs of setupClipboard.
type clipboardResult struct {
	stdinInterceptor io.Reader
	bridgeDir        string
	clipPort         string
}

// setupClipboard initialises clipboard support and returns the chained stdin
// interceptor, bridge directory path, and the clipboard daemon port.
func setupClipboard(log *logger.Logger) *clipboardResult {
	// Initialize clipboard library once from the main goroutine.
	// On macOS this must happen on the main thread (Cocoa requirement).
	clipboardEnabled := false
	if err := clipboard.Init(); err != nil {
		log.Debug("clipboard", "clipboard init failed (Ctrl+V disabled): %v", err)
	} else {
		clipboardEnabled = true
		log.Debug("clipboard", "clipboard initialized")
	}

	homeDir, err := os.UserHomeDir()
	if err != nil {
		log.Debug("clipboard", "cannot determine home dir: %v", err)
		return nil
	}

	// Create bridge directory for file drag-drop.
	bridgeDir := filepath.Join(homeDir, constants.BridgeDirName)
	if err := os.MkdirAll(bridgeDir, 0755); err != nil {
		log.Debug("clipboard", "cannot create bridge dir: %v", err)
		return nil
	}

	// Find a free port for the clipboard daemon.
	clipPort, err := findFreePort()
	if err != nil {
		log.Debug("clipboard", "cannot find free port: %v", err)
		return nil
	}

	// Debug adapter for terminal package.
	termDebug := func(prefix, format string, args ...any) {
		log.Debug(prefix, format, args...)
	}

	// Build the clipboard syncer (Ctrl+V image transfer) only if clipboard is available.
	var syncer terminal.ClipboardSyncer
	if clipboardEnabled {
		syncer = &terminal.TCPClipboardSyncer{
			Address: fmt.Sprintf("localhost:%s", clipPort),
			Reader:  clipboard.Reader{},
			Debug:   termDebug,
		}
		log.Debug("clipboard", "syncer address: localhost:%s", clipPort)
	}

	// Single interceptor handles both Ctrl+V clipboard sync and paste path rewriting.
	bridge := &terminal.FileBridge{
		HostDir:      bridgeDir,
		ContainerDir: constants.ContainerBridgeDir,
	}
	interceptor := terminal.NewInterceptor(os.Stdin, syncer, bridge)
	interceptor.Debug = termDebug
	log.Debug("clipboard", "interceptor created, bridge=%s", bridgeDir)

	return &clipboardResult{
		stdinInterceptor: interceptor,
		bridgeDir:        bridgeDir,
		clipPort:         clipPort,
	}
}

// findFreePort asks the OS for a free TCP port and returns it as a string.
func findFreePort() (string, error) {
	listener, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		return "", err
	}
	defer listener.Close()
	port := listener.Addr().(*net.TCPAddr).Port
	return fmt.Sprintf("%d", port), nil
}

// --- Real implementations ---

type realDockerChecker struct{}

func (realDockerChecker) CheckRunning() error { return docker.CheckRunning() }

type realTokenCapturer struct {
	runner claude.CommandRunner
	log    *logger.Logger
}

func (r *realTokenCapturer) CaptureToken(claudePath string) (string, error) {
	r.log.Debug("token", "starting capture with path: %s", claudePath)
	reqLog := func(method, path, authHeader string) {
		r.log.Debug("token", "request: %s %s auth=%q", method, path, authHeader)
	}
	token, err := claude.CaptureTokenWithLogger(claudePath, r.runner, reqLog)
	if err != nil {
		r.log.Debug("token", "capture failed: %v", err)
	} else {
		r.log.Debug("token", "capture result: %s", maskToken(token))
	}
	return token, err
}

type realVersionDetector struct {
	runner claude.VersionRunner
}

func (r *realVersionDetector) DetectVersion(claudePath string) (string, error) {
	return claude.DetectVersion(claudePath, r.runner)
}

type realImageEnsurer struct {
	mgr docker.ImageManager
}

func (r *realImageEnsurer) EnsureLocalImage(ccboxVersion, claudeVersion string, pinned bool) error {
	return docker.EnsureLocalImage(ccboxVersion, claudeVersion, pinned, r.mgr)
}

type realContainerRunner struct{}

func (realContainerRunner) RunContainer(spec *docker.ContainerSpec) (int, error) {
	return docker.RunContainer(spec, nil, &execProcessRunner{})
}

// execProcess wraps an exec.Cmd as a docker.Process for signal forwarding.
type execProcess struct {
	cmd *exec.Cmd
}

func (p *execProcess) Signal(sig os.Signal) error {
	if p.cmd.Process == nil {
		return nil
	}
	return p.cmd.Process.Signal(sig)
}

func (p *execProcess) Wait() error {
	return p.cmd.Wait()
}

// execProcessRunner implements docker.ProcessRunner using real exec.Cmd.
type execProcessRunner struct{}

func (execProcessRunner) Start(name string, args ...string) (docker.Process, error) {
	cmd := exec.Command(name, args...)
	cmd.Stdin = os.Stdin
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	if err := cmd.Start(); err != nil {
		return nil, err
	}
	return &execProcess{cmd: cmd}, nil
}

// realCLIRunner implements claude.CommandRunner for CaptureToken.
// Stdout/stderr are discarded — CaptureToken only needs the HTTP request,
// and claude's TUI output must not leak to the user's terminal.
type realCLIRunner struct{}

type realCLIProcess struct {
	cmd *exec.Cmd
}

func (p *realCLIProcess) Kill() error {
	if p.cmd.Process == nil {
		return nil
	}
	return p.cmd.Process.Kill()
}

func (p *realCLIProcess) Wait() error {
	return p.cmd.Wait()
}

func (realCLIRunner) Start(name string, cliArgs []string, env []string) (claude.CaptureProcess, error) {
	cmd := exec.Command(name, cliArgs...)
	cmd.Env = append(os.Environ(), env...)
	if err := cmd.Start(); err != nil {
		return nil, err
	}
	return &realCLIProcess{cmd: cmd}, nil
}

// maskToken returns a partially masked token for debug output:
// first 3 and last 3 characters visible, *** in the middle.
// Returns "(empty)" for empty strings and masks short tokens entirely.
func maskToken(token string) string {
	if token == "" {
		return "(empty)"
	}
	if len(token) <= 6 {
		return "***"
	}
	return token[:3] + "***" + token[len(token)-3:]
}
