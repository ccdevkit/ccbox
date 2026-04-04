package docker

import (
	"fmt"
	"os/exec"
	"testing"
)

// mockCLICmdRunner records calls and returns configured responses.
type mockCLICmdRunner struct {
	runErr       error
	outputBytes  []byte
	outputErr    error
	stdinErr     error
	lastRunName  string
	lastRunArgs  []string
	lastStdin    string
	lastStdinCmd string
}

func (m *mockCLICmdRunner) Run(name string, args ...string) error {
	m.lastRunName = name
	m.lastRunArgs = args
	return m.runErr
}

func (m *mockCLICmdRunner) Output(name string, args ...string) ([]byte, error) {
	m.lastRunName = name
	m.lastRunArgs = args
	return m.outputBytes, m.outputErr
}

func (m *mockCLICmdRunner) RunWithStdin(name string, stdin string, args ...string) error {
	m.lastStdinCmd = name
	m.lastStdin = stdin
	m.lastRunArgs = args
	return m.stdinErr
}

func TestCLIImageManager_ImageExists_True(t *testing.T) {
	runner := &mockCLICmdRunner{runErr: nil}
	mgr := &CLIImageManager{Runner: runner}

	exists, err := mgr.ImageExists("ccbox-local:0.2.0-2.1.16")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !exists {
		t.Fatal("expected image to exist")
	}
	if runner.lastRunName != "docker" {
		t.Fatalf("expected docker command, got %s", runner.lastRunName)
	}
}

func TestCLIImageManager_ImageExists_False(t *testing.T) {
	// docker image inspect returns non-zero exit code when image doesn't exist.
	runner := &mockCLICmdRunner{runErr: &exec.ExitError{}}
	mgr := &CLIImageManager{Runner: runner}

	exists, err := mgr.ImageExists("nonexistent:latest")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if exists {
		t.Fatal("expected image to not exist")
	}
}

func TestCLIImageManager_ImageExists_Error(t *testing.T) {
	runner := &mockCLICmdRunner{runErr: fmt.Errorf("connection refused")}
	mgr := &CLIImageManager{Runner: runner}

	_, err := mgr.ImageExists("test:latest")
	if err == nil {
		t.Fatal("expected error")
	}
}

func TestCLIImageManager_BuildImage(t *testing.T) {
	runner := &mockCLICmdRunner{stdinErr: nil}
	mgr := &CLIImageManager{Runner: runner}

	err := mgr.BuildImage("ccbox-local:0.2.0-2.1.16", "FROM node:22\n", ".")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if runner.lastStdin != "FROM node:22\n" {
		t.Fatalf("expected dockerfile in stdin, got %q", runner.lastStdin)
	}
}

func TestCLIImageManager_BuildImage_Error(t *testing.T) {
	runner := &mockCLICmdRunner{stdinErr: fmt.Errorf("build failed")}
	mgr := &CLIImageManager{Runner: runner}

	err := mgr.BuildImage("test:latest", "FROM alpine\n", ".")
	if err == nil {
		t.Fatal("expected error")
	}
}

func TestCLIImageManager_RemoveImage(t *testing.T) {
	runner := &mockCLICmdRunner{runErr: nil}
	mgr := &CLIImageManager{Runner: runner}

	err := mgr.RemoveImage("ccbox-local:old")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if runner.lastRunArgs[0] != "rmi" {
		t.Fatalf("expected rmi command, got %v", runner.lastRunArgs)
	}
}

func TestCLIImageManager_RemoveImage_Error(t *testing.T) {
	runner := &mockCLICmdRunner{runErr: fmt.Errorf("image in use")}
	mgr := &CLIImageManager{Runner: runner}

	err := mgr.RemoveImage("test:latest")
	if err == nil {
		t.Fatal("expected error")
	}
}

func TestCLIImageManager_ListImages(t *testing.T) {
	runner := &mockCLICmdRunner{
		outputBytes: []byte("ccbox-local:0.2.0-2.1.16\nccbox-local:0.1.0-2.1.15\n"),
	}
	mgr := &CLIImageManager{Runner: runner}

	images, err := mgr.ListImages("ccbox-local")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(images) != 2 {
		t.Fatalf("expected 2 images, got %d", len(images))
	}
	if images[0] != "ccbox-local:0.2.0-2.1.16" {
		t.Fatalf("unexpected first image: %s", images[0])
	}
}

func TestCLIImageManager_ListImages_Empty(t *testing.T) {
	runner := &mockCLICmdRunner{outputBytes: []byte("")}
	mgr := &CLIImageManager{Runner: runner}

	images, err := mgr.ListImages("ccbox-local")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if images != nil {
		t.Fatalf("expected nil for empty list, got %v", images)
	}
}

func TestCLIImageManager_ListImages_Error(t *testing.T) {
	runner := &mockCLICmdRunner{outputErr: fmt.Errorf("docker error")}
	mgr := &CLIImageManager{Runner: runner}

	_, err := mgr.ListImages("ccbox-local")
	if err == nil {
		t.Fatal("expected error")
	}
}
