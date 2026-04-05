package docker

import (
	"bytes"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"strings"
)

// CLICmdRunner abstracts command execution for CLIImageManager testability.
type CLICmdRunner interface {
	Output(name string, args ...string) ([]byte, error)
	Run(name string, args ...string) error
	RunWithStdin(name string, stdin string, args ...string) error
}

// ExecCLICmdRunner implements CLICmdRunner using real exec.Command.
type ExecCLICmdRunner struct{}

func (ExecCLICmdRunner) Output(name string, args ...string) ([]byte, error) {
	return exec.Command(name, args...).Output()
}

func (ExecCLICmdRunner) Run(name string, args ...string) error {
	return exec.Command(name, args...).Run()
}

func (ExecCLICmdRunner) RunWithStdin(name string, stdin string, args ...string) error {
	cmd := exec.Command(name, args...)
	cmd.Stdin = strings.NewReader(stdin)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	return cmd.Run()
}

// CLIImageManager implements ImageManager by shelling out to the docker CLI.
type CLIImageManager struct {
	Runner CLICmdRunner
}

// NewCLIImageManager returns a CLIImageManager using real exec commands.
func NewCLIImageManager() *CLIImageManager {
	return &CLIImageManager{Runner: ExecCLICmdRunner{}}
}

// ImageExists checks if a Docker image exists locally via docker image inspect.
func (m *CLIImageManager) ImageExists(imageName string) (bool, error) {
	err := m.Runner.Run("docker", "image", "inspect", imageName)
	if err != nil {
		var exitErr *exec.ExitError
		if errors.As(err, &exitErr) {
			return false, nil
		}
		return false, fmt.Errorf("inspecting image %s: %w", imageName, err)
	}
	return true, nil
}

// BuildImage builds a Docker image from the given Dockerfile content via stdin.
func (m *CLIImageManager) BuildImage(imageName string, dockerfile string, context string) error {
	err := m.Runner.RunWithStdin("docker", dockerfile, "build", "-t", imageName, "-f", "-", context)
	if err != nil {
		return fmt.Errorf("building image %s: %w", imageName, err)
	}
	return nil
}

// RemoveImage removes a Docker image via docker rmi.
func (m *CLIImageManager) RemoveImage(imageName string) error {
	err := m.Runner.Run("docker", "rmi", imageName)
	if err != nil {
		return fmt.Errorf("removing image %s: %w", imageName, err)
	}
	return nil
}

// ListContainersForImage returns container IDs (running or stopped) using the given image.
func (m *CLIImageManager) ListContainersForImage(imageName string) ([]string, error) {
	out, err := m.Runner.Output("docker", "ps", "-a", "--filter", "ancestor="+imageName, "--format", "{{.ID}}")
	if err != nil {
		return nil, fmt.Errorf("listing containers for image %s: %w", imageName, err)
	}
	trimmed := bytes.TrimSpace(out)
	if len(trimmed) == 0 {
		return nil, nil
	}
	lines := strings.Split(string(trimmed), "\n")
	var result []string
	for _, line := range lines {
		line = strings.TrimSpace(line)
		if line != "" {
			result = append(result, line)
		}
	}
	return result, nil
}

// StopAndRemoveContainer stops and removes a container by ID.
// The rm call uses -f to handle containers that were already removed by --rm.
func (m *CLIImageManager) StopAndRemoveContainer(containerID string) error {
	if err := m.Runner.Run("docker", "stop", containerID); err != nil {
		return fmt.Errorf("stopping container %s: %w", containerID, err)
	}
	// Use rm -f: the container may already be gone if it was started with --rm.
	_ = m.Runner.Run("docker", "rm", containerID)
	return nil
}

// ListImages lists Docker images matching the given prefix.
// Returns image names in the format "repository:tag".
func (m *CLIImageManager) ListImages(prefix string) ([]string, error) {
	out, err := m.Runner.Output("docker", "images", "--filter", "reference="+prefix+":*", "--format", "{{.Repository}}:{{.Tag}}")
	if err != nil {
		return nil, fmt.Errorf("listing images with prefix %s: %w", prefix, err)
	}

	trimmed := bytes.TrimSpace(out)
	if len(trimmed) == 0 {
		return nil, nil
	}

	lines := strings.Split(string(trimmed), "\n")
	var result []string
	for _, line := range lines {
		line = strings.TrimSpace(line)
		if line != "" {
			result = append(result, line)
		}
	}
	return result, nil
}
