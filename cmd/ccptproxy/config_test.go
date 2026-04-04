package main

import (
	"os"
	"path/filepath"
	"testing"
)

func TestReadConfig_UnmarshalsHostAddressAndPassthrough(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "ccbox-proxy.json")

	data := `{"hostAddress":"192.168.1.10:9100","passthrough":["cat","ls","grep"],"verbose":true}`
	if err := os.WriteFile(path, []byte(data), 0644); err != nil {
		t.Fatal(err)
	}

	cfg, err := ReadConfig(path)
	if err != nil {
		t.Fatalf("ReadConfig returned error: %v", err)
	}

	if cfg.HostAddress != "192.168.1.10:9100" {
		t.Errorf("HostAddress = %q, want %q", cfg.HostAddress, "192.168.1.10:9100")
	}
	if len(cfg.Passthrough) != 3 {
		t.Fatalf("Passthrough length = %d, want 3", len(cfg.Passthrough))
	}
	if cfg.Passthrough[0] != "cat" || cfg.Passthrough[1] != "ls" || cfg.Passthrough[2] != "grep" {
		t.Errorf("Passthrough = %v, want [cat ls grep]", cfg.Passthrough)
	}
	if !cfg.Verbose {
		t.Error("Verbose = false, want true")
	}
}

func TestReadConfig_FileNotFound(t *testing.T) {
	_, err := ReadConfig("/nonexistent/path/ccbox-proxy.json")
	if err == nil {
		t.Fatal("expected error for missing file, got nil")
	}
}

func TestReadConfig_InvalidJSON(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "ccbox-proxy.json")

	if err := os.WriteFile(path, []byte(`{invalid`), 0644); err != nil {
		t.Fatal(err)
	}

	_, err := ReadConfig(path)
	if err == nil {
		t.Fatal("expected error for invalid JSON, got nil")
	}
}
