package docker

import (
	"errors"
	"fmt"
	"os"
	"os/exec"
	"strings"
	"sync"
	"syscall"
	"testing"
)

type mockCommandRunner struct {
	err      error
	lastName string
	lastArgs []string
}

func (m *mockCommandRunner) Run(name string, args ...string) error {
	m.lastName = name
	m.lastArgs = args
	return m.err
}

// --- ProcessRunner mocks ---

type mockProcess struct {
	waitErr  error
	mu       sync.Mutex
	signals  []os.Signal
	waitCh   chan struct{} // closed when Wait should return
	onSignal func()       // called after each Signal() for synchronization
}

func (p *mockProcess) Signal(sig os.Signal) error {
	p.mu.Lock()
	p.signals = append(p.signals, sig)
	cb := p.onSignal
	p.mu.Unlock()
	if cb != nil {
		cb()
	}
	return nil
}

func (p *mockProcess) Wait() error {
	if p.waitCh != nil {
		<-p.waitCh
	}
	return p.waitErr
}

type mockProcessRunner struct {
	process  *mockProcess
	startErr error
	lastName string
	lastArgs []string
}

func (m *mockProcessRunner) Start(name string, args ...string) (Process, error) {
	m.lastName = name
	m.lastArgs = args
	if m.startErr != nil {
		return nil, m.startErr
	}
	return m.process, nil
}

// --- CheckRunning tests ---

func TestCheckRunning_Success(t *testing.T) {
	runner := &mockCommandRunner{err: nil}
	err := checkRunningWith(runner)
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
}

func TestCheckRunning_DockerNotAvailable(t *testing.T) {
	runner := &mockCommandRunner{err: errors.New("exec: \"docker\": executable file not found")}
	err := checkRunningWith(runner)
	if err == nil {
		t.Fatal("expected error, got nil")
	}
	expected := "Docker is not running or not installed. Please start Docker and try again."
	if err.Error() != expected {
		t.Fatalf("expected %q, got %q", expected, err.Error())
	}
}

// --- buildDockerArgs tests ---

func TestBuildDockerArgs_RmFlagPresent(t *testing.T) {
	spec := &ContainerSpec{
		ImageName: "ccbox-local:0.2.0-2.1.16",
	}
	args := BuildDockerArgs(spec)
	found := false
	for _, a := range args {
		if a == "--rm" {
			found = true
			break
		}
	}
	if !found {
		t.Fatalf("expected --rm flag in args, got %v", args)
	}
}

func TestBuildDockerArgs_BasicImage(t *testing.T) {
	spec := &ContainerSpec{
		ImageName: "ccbox-local:0.2.0-2.1.16",
	}
	args := BuildDockerArgs(spec)
	if args[len(args)-1] != "ccbox-local:0.2.0-2.1.16" {
		t.Fatalf("expected image name as last arg, got %v", args)
	}
	if args[0] != "run" {
		t.Fatalf("expected first arg to be 'run', got %v", args[0])
	}
}

func TestBuildDockerArgs_Mounts(t *testing.T) {
	spec := &ContainerSpec{
		ImageName: "test-image",
		Mounts: []Mount{
			{Host: "/host/path", Container: "/container/path", ReadOnly: false},
			{Host: "/host/ro", Container: "/container/ro", ReadOnly: true},
		},
	}
	args := BuildDockerArgs(spec)
	joined := strings.Join(args, " ")
	if !strings.Contains(joined, "-v /host/path:/container/path") {
		t.Fatalf("expected rw mount in args, got %v", args)
	}
	if !strings.Contains(joined, "-v /host/ro:/container/ro:ro") {
		t.Fatalf("expected ro mount in args, got %v", args)
	}
}

func TestBuildDockerArgs_EnvVars(t *testing.T) {
	spec := &ContainerSpec{
		ImageName: "test-image",
		Env: []EnvVar{
			{Key: "TERM", Value: "xterm-256color", Secret: false},
			{Key: "CLAUDE_CODE_OAUTH_TOKEN", Value: "secret-token", Secret: true},
		},
	}
	args := BuildDockerArgs(spec)
	joined := strings.Join(args, " ")
	if !strings.Contains(joined, "-e TERM=xterm-256color") {
		t.Fatalf("expected TERM env var in args, got %v", args)
	}
	if !strings.Contains(joined, "-e CLAUDE_CODE_OAUTH_TOKEN=secret-token") {
		t.Fatalf("expected secret env var in args, got %v", args)
	}
}

func TestBuildDockerArgs_Ports(t *testing.T) {
	spec := &ContainerSpec{
		ImageName: "test-image",
		Ports: []PortMapping{
			{Host: 8080, Container: 80},
		},
	}
	args := BuildDockerArgs(spec)
	joined := strings.Join(args, " ")
	if !strings.Contains(joined, "-p 8080:80") {
		t.Fatalf("expected port mapping in args, got %v", args)
	}
}

func TestBuildDockerArgs_WorkDir(t *testing.T) {
	spec := &ContainerSpec{
		ImageName: "test-image",
		WorkDir:   "/home/claude",
	}
	args := BuildDockerArgs(spec)
	joined := strings.Join(args, " ")
	if !strings.Contains(joined, "-w /home/claude") {
		t.Fatalf("expected workdir in args, got %v", args)
	}
}

func TestBuildDockerArgs_Command(t *testing.T) {
	spec := &ContainerSpec{
		ImageName: "test-image",
		Command:   "/bin/bash",
		Args:      []string{"-c", "echo hello"},
	}
	args := BuildDockerArgs(spec)
	imgIdx := -1
	for i, a := range args {
		if a == "test-image" {
			imgIdx = i
			break
		}
	}
	if imgIdx == -1 {
		t.Fatalf("image not found in args: %v", args)
	}
	remaining := args[imgIdx+1:]
	if len(remaining) < 3 || remaining[0] != "/bin/bash" || remaining[1] != "-c" || remaining[2] != "echo hello" {
		t.Fatalf("expected command and args after image, got %v", remaining)
	}
}

func TestBuildDockerArgs_InteractiveFlag(t *testing.T) {
	spec := &ContainerSpec{
		ImageName: "test-image",
	}
	args := BuildDockerArgs(spec)
	joined := strings.Join(args, " ")
	if !strings.Contains(joined, "-it") {
		t.Fatalf("expected -it flag in args, got %v", args)
	}
}

func TestBuildDockerArgs_NoDockerSocket(t *testing.T) {
	spec := &ContainerSpec{
		ImageName: "test-image",
		Mounts: []Mount{
			{Host: "/some/path", Container: "/container/path"},
		},
	}
	args := BuildDockerArgs(spec)
	joined := strings.Join(args, " ")
	if strings.Contains(joined, "/var/run/docker.sock") {
		t.Fatalf("docker socket mount must not be present per FR-030, got %v", args)
	}
}

// --- RunContainer tests ---

func TestRunContainer_ReturnsZeroExitCode(t *testing.T) {
	proc := &mockProcess{waitErr: nil}
	runner := &mockProcessRunner{process: proc}
	spec := &ContainerSpec{ImageName: "test-image"}

	exitCode, err := RunContainer(spec, nil, runner)
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if exitCode != 0 {
		t.Fatalf("expected exit code 0, got %d", exitCode)
	}
}

func TestRunContainer_ReturnsNonZeroExitCode(t *testing.T) {
	exitErr := &exec.ExitError{
		ProcessState: newProcessStateWithExitCode(1),
	}
	proc := &mockProcess{waitErr: exitErr}
	runner := &mockProcessRunner{process: proc}
	spec := &ContainerSpec{ImageName: "test-image"}

	exitCode, err := RunContainer(spec, nil, runner)
	if err != nil {
		t.Fatalf("expected no error for non-zero exit, got %v", err)
	}
	if exitCode != 1 {
		t.Fatalf("expected exit code 1, got %d", exitCode)
	}
}

func TestRunContainer_ReturnsErrorOnStartFailure(t *testing.T) {
	runner := &mockProcessRunner{startErr: errors.New("docker not found")}
	spec := &ContainerSpec{ImageName: "test-image"}

	_, err := RunContainer(spec, nil, runner)
	if err == nil {
		t.Fatal("expected error, got nil")
	}
}

func TestRunContainer_ReturnsErrorOnWaitFailure(t *testing.T) {
	proc := &mockProcess{waitErr: errors.New("unexpected error")}
	runner := &mockProcessRunner{process: proc}
	spec := &ContainerSpec{ImageName: "test-image"}

	_, err := RunContainer(spec, nil, runner)
	if err == nil {
		t.Fatal("expected error, got nil")
	}
}

func TestRunContainer_CallsDockerWithCorrectArgs(t *testing.T) {
	proc := &mockProcess{waitErr: nil}
	runner := &mockProcessRunner{process: proc}
	spec := &ContainerSpec{
		ImageName: "test-image",
		WorkDir:   "/work",
		Env: []EnvVar{
			{Key: "FOO", Value: "bar"},
		},
	}
	_, _ = RunContainer(spec, nil, runner)
	if runner.lastName != "docker" {
		t.Fatalf("expected command 'docker', got %q", runner.lastName)
	}
	if runner.lastArgs[0] != "run" {
		t.Fatalf("expected first arg 'run', got %q", runner.lastArgs[0])
	}
}

func TestRunContainerWithSignals_ForwardsToProcess(t *testing.T) {
	// Use a waitCh to keep the mock process "running" until we release it.
	waitCh := make(chan struct{})
	signalReceived := make(chan struct{}, 1)
	proc := &mockProcess{waitErr: nil, waitCh: waitCh, onSignal: func() {
		select {
		case signalReceived <- struct{}{}:
		default:
		}
	}}
	runner := &mockProcessRunner{process: proc}
	spec := &ContainerSpec{ImageName: "test-image"}

	// Inject a fake signal channel instead of using real OS signals.
	sigCh := make(chan os.Signal, 1)

	done := make(chan struct{})
	var exitCode int
	var runErr error
	go func() {
		exitCode, runErr = runContainerWithSignals(spec, nil, runner, sigCh)
		close(done)
	}()

	// Send SIGTERM via the injected channel.
	sigCh <- syscall.SIGTERM

	// Wait for the mock process to receive it.
	<-signalReceived

	// Verify the correct signal was forwarded.
	proc.mu.Lock()
	if len(proc.signals) != 1 {
		t.Fatalf("expected 1 signal, got %d", len(proc.signals))
	}
	if proc.signals[0] != syscall.SIGTERM {
		t.Fatalf("expected SIGTERM, got %v", proc.signals[0])
	}
	proc.mu.Unlock()

	// Let the process exit.
	close(waitCh)
	<-done

	if runErr != nil {
		t.Fatalf("unexpected error: %v", runErr)
	}
	if exitCode != 0 {
		t.Fatalf("expected exit code 0, got %d", exitCode)
	}
}

func TestRunContainerWithSignals_ForwardsMultipleSignals(t *testing.T) {
	waitCh := make(chan struct{})
	signalCount := make(chan struct{}, 2)
	proc := &mockProcess{waitErr: nil, waitCh: waitCh, onSignal: func() {
		signalCount <- struct{}{}
	}}
	runner := &mockProcessRunner{process: proc}
	spec := &ContainerSpec{ImageName: "test-image"}

	sigCh := make(chan os.Signal, 3)
	done := make(chan struct{})
	go func() {
		_, _ = runContainerWithSignals(spec, nil, runner, sigCh)
		close(done)
	}()

	// Send multiple signals and wait for each to be received.
	sigCh <- syscall.SIGINT
	<-signalCount
	sigCh <- syscall.SIGHUP
	<-signalCount

	// Release the process.
	close(waitCh)
	<-done

	proc.mu.Lock()
	defer proc.mu.Unlock()
	if len(proc.signals) != 2 {
		t.Fatalf("expected 2 signals forwarded, got %d", len(proc.signals))
	}
	if proc.signals[0] != syscall.SIGINT {
		t.Fatalf("expected first signal SIGINT, got %v", proc.signals[0])
	}
	if proc.signals[1] != syscall.SIGHUP {
		t.Fatalf("expected second signal SIGHUP, got %v", proc.signals[1])
	}
}

// newProcessStateWithExitCode creates a *os.ProcessState with the given exit code.
func newProcessStateWithExitCode(code int) *os.ProcessState {
	cmd := exec.Command("sh", "-c", fmt.Sprintf("exit %d", code))
	_ = cmd.Run()
	return cmd.ProcessState
}
