package docker

import (
	"errors"
	"fmt"
	"io"
	"os"
	"os/exec"
	"os/signal"
	"syscall"

	"golang.org/x/term"
)

// ContainerSpec is the specification for launching a Docker container.
type ContainerSpec struct {
	ImageName string
	Mounts    []Mount
	Env       []EnvVar
	Ports     []PortMapping
	Args      []string
	Command   string
	WorkDir   string

	// StdinInterceptor, when set, wraps stdin for clipboard sync and path
	// rewriting. When stdin is a TTY, a PTY bridge is used to preserve
	// terminal behaviour while intercepting input.
	StdinInterceptor io.Reader
}

// Mount represents a bind mount from host to container.
type Mount struct {
	Host      string
	Container string
	ReadOnly  bool
}

// EnvVar represents an environment variable passed to the container.
type EnvVar struct {
	Key    string
	Value  string
	Secret bool
}

// PortMapping represents a host-to-container port mapping.
type PortMapping struct {
	Host      int
	Container int
}

// PTY abstracts platform-specific pseudo-terminal operations.
type PTY interface {
	Read(p []byte) (int, error)
	Write(p []byte) (int, error)
	Resize(rows, cols uint16) error
	Close() error
	Wait() error
}

// Process represents a running process that can be signalled and waited on.
type Process interface {
	Signal(sig os.Signal) error
	Wait() error
}

// ProcessRunner abstracts process lifecycle for testability.
// Start launches a command and returns a Process handle for signal forwarding.
type ProcessRunner interface {
	Start(name string, args ...string) (Process, error)
}

// CommandRunner abstracts simple command execution for testability (used by CheckRunning).
type CommandRunner interface {
	Run(name string, args ...string) error
}

type execCommandRunner struct{}

func (e *execCommandRunner) Run(name string, args ...string) error {
	return exec.Command(name, args...).Run()
}

// CheckRunning verifies that Docker is available by running "docker info".
func CheckRunning() error {
	return checkRunningWith(&execCommandRunner{})
}

func checkRunningWith(runner CommandRunner) error {
	if err := runner.Run("docker", "info"); err != nil {
		return errors.New("Docker is not running or not installed. Please start Docker and try again.")
	}
	return nil
}

// BuildDockerArgs converts a ContainerSpec into docker run arguments.
func BuildDockerArgs(spec *ContainerSpec) []string {
	args := []string{"run", "--rm", "-it"}

	for _, m := range spec.Mounts {
		mountArg := m.Host + ":" + m.Container
		if m.ReadOnly {
			mountArg += ":ro"
		}
		args = append(args, "-v", mountArg)
	}

	for _, e := range spec.Env {
		args = append(args, "-e", e.Key+"="+e.Value)
	}

	for _, p := range spec.Ports {
		args = append(args, "-p", fmt.Sprintf("%d:%d", p.Host, p.Container))
	}

	if spec.WorkDir != "" {
		args = append(args, "-w", spec.WorkDir)
	}

	args = append(args, spec.ImageName)

	if spec.Command != "" {
		args = append(args, spec.Command)
	}

	args = append(args, spec.Args...)

	return args
}

// RunContainer starts docker run with the given ContainerSpec, forwards
// SIGINT/SIGTERM/SIGHUP to the container process (FR-035), waits for it
// to exit, and returns the container's exit code.
//
// When spec.StdinInterceptor is set and stdin is a TTY, a PTY bridge is
// used to preserve terminal behaviour while intercepting input. Otherwise
// the ProcessRunner path is used directly.
func RunContainer(spec *ContainerSpec, pty PTY, runner ProcessRunner) (int, error) {
	// PTY interceptor path: bypass ProcessRunner, manage the command directly.
	if spec.StdinInterceptor != nil && term.IsTerminal(int(os.Stdin.Fd())) {
		return runWithPTY(spec)
	}

	sigCh := make(chan os.Signal, 1)
	signal.Notify(sigCh, syscall.SIGINT, syscall.SIGTERM, syscall.SIGHUP)
	defer signal.Stop(sigCh)
	return runContainerWithSignals(spec, pty, runner, sigCh)
}

// runContainerWithSignals is the testable core of RunContainer.
// It accepts an external signal channel so tests can inject signals without
// sending real OS signals to the test process.
func runContainerWithSignals(spec *ContainerSpec, pty PTY, runner ProcessRunner, sigCh <-chan os.Signal) (int, error) {
	args := BuildDockerArgs(spec)
	proc, err := runner.Start("docker", args...)
	if err != nil {
		return -1, fmt.Errorf("starting container: %w", err)
	}

	// Forward signals to the container process (FR-035).
	done := make(chan struct{})
	go func() {
		for {
			select {
			case sig := <-sigCh:
				_ = proc.Signal(sig)
			case <-done:
				return
			}
		}
	}()

	waitErr := proc.Wait()
	close(done)

	if waitErr == nil {
		return 0, nil
	}

	var exitErr *exec.ExitError
	if errors.As(waitErr, &exitErr) {
		return exitErr.ExitCode(), nil
	}

	return -1, waitErr
}

// runWithPTY launches the docker command with a PTY, copies the stdin
// interceptor to the PTY master, and copies PTY output to stdout.
// This preserves terminal features (colors, cursor control, resize)
// while allowing clipboard sync and path rewriting.
func runWithPTY(spec *ContainerSpec) (int, error) {
	args := BuildDockerArgs(spec)
	cmd := exec.Command("docker", args...)

	p, err := newPTY(cmd)
	if err != nil {
		return -1, fmt.Errorf("starting pty: %w", err)
	}
	defer p.Close()

	// Handle terminal resize (SIGWINCH).
	cleanup := handleResize(p)
	defer cleanup()

	// Set host stdin to raw mode so keystrokes pass through unmodified.
	oldState, err := term.MakeRaw(int(os.Stdin.Fd()))
	if err == nil {
		defer term.Restore(int(os.Stdin.Fd()), oldState)
	}

	// Copy intercepted stdin → PTY master (runs clipboard sync + path rewriting).
	go func() {
		_, _ = io.Copy(p, spec.StdinInterceptor)
	}()

	// Copy PTY master output → stdout.
	_, _ = io.Copy(os.Stdout, p)

	// Wait for command to finish.
	waitErr := p.Wait()
	if waitErr == nil {
		return 0, nil
	}

	var exitErr *exec.ExitError
	if errors.As(waitErr, &exitErr) {
		return exitErr.ExitCode(), nil
	}

	return -1, waitErr
}
