package terminal

import (
	"os"
	"path/filepath"
	"testing"
)

func TestFileBridge_CopyFileToBridge(t *testing.T) {
	// Create a temp source file.
	srcDir := t.TempDir()
	srcPath := filepath.Join(srcDir, "test.png")
	content := []byte("fake-png-data")
	if err := os.WriteFile(srcPath, content, 0644); err != nil {
		t.Fatal(err)
	}

	// Create bridge directory.
	bridgeDir := t.TempDir()

	b := &FileBridge{
		HostDir:      bridgeDir,
		ContainerDir: "/home/claude/.ccbox-bridge",
	}

	containerPath, err := b.CopyFileToBridge(srcPath)
	if err != nil {
		t.Fatalf("CopyFileToBridge() error: %v", err)
	}

	// Verify container path.
	if want := "/home/claude/.ccbox-bridge/test.png"; containerPath != want {
		t.Errorf("container path = %q, want %q", containerPath, want)
	}

	// Verify file was copied.
	got, err := os.ReadFile(filepath.Join(bridgeDir, "test.png"))
	if err != nil {
		t.Fatalf("reading copied file: %v", err)
	}
	if string(got) != string(content) {
		t.Errorf("copied content = %q, want %q", got, content)
	}
}

func TestFileBridge_CopyFileToBridge_SourceNotFound(t *testing.T) {
	b := &FileBridge{
		HostDir:      t.TempDir(),
		ContainerDir: "/home/claude/.ccbox-bridge",
	}

	_, err := b.CopyFileToBridge("/nonexistent/file.png")
	if err == nil {
		t.Fatal("expected error for nonexistent source file")
	}
}
