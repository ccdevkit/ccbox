package main

import (
	"fmt"
	"os"
	"os/exec"

	"github.com/ccdevkit/ccbox/internal/claude"
	"github.com/ccdevkit/ccbox/internal/docker"
)

// UpdateRunner abstracts running the claude update command.
type UpdateRunner interface {
	Run(name string, args ...string) error
}

// execUpdateRunner runs commands using os/exec with stdout/stderr passthrough.
type execUpdateRunner struct{}

func (execUpdateRunner) Run(name string, args ...string) error {
	cmd := exec.Command(name, args...)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	return cmd.Run()
}

// execVersionRunner implements claude.VersionRunner using os/exec.
type execVersionRunner struct{}

func (execVersionRunner) Output(name string, args ...string) ([]byte, error) {
	return exec.Command(name, args...).Output()
}

// doUpdate runs the update flow: run claude update, detect new version, rebuild image.
func doUpdate(claudePath, ccboxVersion string, runner UpdateRunner, detector claude.VersionRunner, mgr docker.ImageManager) error {
	if err := runner.Run(claudePath, "update"); err != nil {
		return fmt.Errorf("claude update failed: %w", err)
	}

	newVersion, err := claude.DetectVersion(claudePath, detector)
	if err != nil {
		return fmt.Errorf("detecting claude version after update: %w", err)
	}

	if err := docker.EnsureLocalImage(ccboxVersion, newVersion, false, mgr); err != nil {
		return fmt.Errorf("rebuilding local image: %w", err)
	}

	return nil
}
