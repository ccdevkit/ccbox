package session

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/ccdevkit/ccbox/internal/constants"
)

func TestNewTempDirProvider_CreatesTempDir(t *testing.T) {
	p, err := NewTempDirProvider("test-session-id")
	if err != nil {
		t.Fatalf("NewTempDirProvider error: %v", err)
	}
	defer p.Cleanup()

	// Dir must exist on disk.
	info, err := os.Stat(p.Dir)
	if err != nil {
		t.Fatalf("Dir does not exist: %v", err)
	}
	if !info.IsDir() {
		t.Fatalf("Dir is not a directory")
	}

	// Dir must contain the TempDirPrefix and session ID.
	base := filepath.Base(p.Dir)
	if !strings.HasPrefix(base, constants.TempDirPrefix) {
		t.Errorf("Dir base %q does not start with %q", base, constants.TempDirPrefix)
	}
	if !strings.Contains(base, "test-session-id") {
		t.Errorf("Dir base %q does not contain session ID", base)
	}

	// Files slice must be initialized and empty.
	if p.Files == nil {
		t.Fatal("Files slice is nil")
	}
	if len(p.Files) != 0 {
		t.Fatalf("Files slice is not empty: %d", len(p.Files))
	}
}

func TestWriteFile_CreatesFileAndAppendsToFiles(t *testing.T) {
	p, err := NewTempDirProvider("write-test")
	if err != nil {
		t.Fatalf("NewTempDirProvider error: %v", err)
	}
	defer p.Cleanup()

	data := []byte(`{"hello":"world"}`)
	containerPath := "/opt/ccbox/settings.json"

	err = p.WriteFile(containerPath, data, true)
	if err != nil {
		t.Fatalf("WriteFile error: %v", err)
	}

	// Files slice should have one entry.
	if len(p.Files) != 1 {
		t.Fatalf("expected 1 file, got %d", len(p.Files))
	}

	sf := p.Files[0]
	if sf.ContainerPath != containerPath {
		t.Errorf("ContainerPath = %q, want %q", sf.ContainerPath, containerPath)
	}

	// Host file must exist with correct content.
	got, err := os.ReadFile(sf.HostPath)
	if err != nil {
		t.Fatalf("cannot read host file: %v", err)
	}
	if string(got) != string(data) {
		t.Errorf("file content = %q, want %q", got, data)
	}

	// Host filename should be the slugified container path.
	expectedSlug := "_opt_ccbox_settings.json"
	if filepath.Base(sf.HostPath) != expectedSlug {
		t.Errorf("host filename = %q, want %q", filepath.Base(sf.HostPath), expectedSlug)
	}
}

func TestWriteFile_MultipleFiles(t *testing.T) {
	p, err := NewTempDirProvider("multi-test")
	if err != nil {
		t.Fatalf("NewTempDirProvider error: %v", err)
	}
	defer p.Cleanup()

	err = p.WriteFile("/opt/ccbox/a.json", []byte("a"), true)
	if err != nil {
		t.Fatalf("WriteFile error: %v", err)
	}
	err = p.WriteFile("/opt/ccbox/b.json", []byte("b"), false)
	if err != nil {
		t.Fatalf("WriteFile error: %v", err)
	}

	if len(p.Files) != 2 {
		t.Fatalf("expected 2 files, got %d", len(p.Files))
	}
}

func TestCleanup_RemovesDir(t *testing.T) {
	p, err := NewTempDirProvider("cleanup-test")
	if err != nil {
		t.Fatalf("NewTempDirProvider error: %v", err)
	}

	dir := p.Dir

	// Write a file so the directory is non-empty.
	err = p.WriteFile("/opt/ccbox/test.json", []byte("test"), true)
	if err != nil {
		t.Fatalf("WriteFile error: %v", err)
	}

	err = p.Cleanup()
	if err != nil {
		t.Fatalf("Cleanup error: %v", err)
	}

	// Dir must no longer exist.
	_, err = os.Stat(dir)
	if !os.IsNotExist(err) {
		t.Fatalf("expected dir to be removed, got err: %v", err)
	}
}

func TestTempDirProvider_ImplementsSessionFileWriter(t *testing.T) {
	p, err := NewTempDirProvider("interface-test")
	if err != nil {
		t.Fatalf("NewTempDirProvider error: %v", err)
	}
	defer p.Cleanup()

	// Compile-time check that *TempDirProvider implements SessionFileWriter.
	var _ SessionFileWriter = p
}
