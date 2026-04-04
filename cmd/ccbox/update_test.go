package main

import (
	"fmt"
	"testing"

	"github.com/ccdevkit/ccbox/internal/docker"
)

// mockUpdateRunner records calls for running the update command.
type mockUpdateRunner struct {
	err error
}

func (m *mockUpdateRunner) Run(name string, args ...string) error {
	return m.err
}

// mockVersionDetector returns a fixed version string.
type mockVersionDetector struct {
	version string
	err     error
}

func (m *mockVersionDetector) Output(name string, args ...string) ([]byte, error) {
	if m.err != nil {
		return nil, m.err
	}
	return []byte(m.version), nil
}

func TestDoUpdate_Success(t *testing.T) {
	runner := &mockUpdateRunner{}
	detector := &mockVersionDetector{version: "2.1.16"}
	mgr := newTestImageManager()

	err := doUpdate("claude", "0.2.0", runner, detector, mgr)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	// Should have built the new image
	targetImage := docker.LocalImageName("0.2.0", "2.1.16")
	if len(mgr.builtImages) != 1 || mgr.builtImages[0] != targetImage {
		t.Fatalf("expected build of %q, got %v", targetImage, mgr.builtImages)
	}
}

func TestDoUpdate_UpdateCommandFails(t *testing.T) {
	runner := &mockUpdateRunner{err: fmt.Errorf("update failed")}
	detector := &mockVersionDetector{version: "2.1.16"}
	mgr := newTestImageManager()

	err := doUpdate("claude", "0.2.0", runner, detector, mgr)
	if err == nil {
		t.Fatal("expected error, got nil")
	}
}

func TestDoUpdate_DetectVersionFails(t *testing.T) {
	runner := &mockUpdateRunner{}
	detector := &mockVersionDetector{err: fmt.Errorf("detect failed")}
	mgr := newTestImageManager()

	err := doUpdate("claude", "0.2.0", runner, detector, mgr)
	if err == nil {
		t.Fatal("expected error, got nil")
	}
}

func TestDoUpdate_RemovesOldAutoUpdateImage(t *testing.T) {
	runner := &mockUpdateRunner{}
	detector := &mockVersionDetector{version: "2.1.16"}
	mgr := newTestImageManager()
	// Old auto-update image exists
	oldImage := "ccbox-local:0.1.0-2.0.0"
	mgr.listResult = []string{oldImage}

	err := doUpdate("claude", "0.2.0", runner, detector, mgr)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	// Old image should be removed by EnsureLocalImage's FR-037 logic
	if len(mgr.removedImages) != 1 || mgr.removedImages[0] != oldImage {
		t.Fatalf("expected removal of %q, got %v", oldImage, mgr.removedImages)
	}
}

// testImageManager is a simple mock for docker.ImageManager in this package's tests.
type testImageManager struct {
	existingImages map[string]bool
	builtImages    []string
	removedImages  []string
	listResult     []string
	buildErr       error
}

func newTestImageManager() *testImageManager {
	return &testImageManager{
		existingImages: make(map[string]bool),
	}
}

func (m *testImageManager) ImageExists(imageName string) (bool, error) {
	return m.existingImages[imageName], nil
}

func (m *testImageManager) BuildImage(imageName string, dockerfile string, context string) error {
	if m.buildErr != nil {
		return m.buildErr
	}
	m.builtImages = append(m.builtImages, imageName)
	m.existingImages[imageName] = true
	return nil
}

func (m *testImageManager) RemoveImage(imageName string) error {
	m.removedImages = append(m.removedImages, imageName)
	delete(m.existingImages, imageName)
	return nil
}

func (m *testImageManager) ListImages(prefix string) ([]string, error) {
	return m.listResult, nil
}
