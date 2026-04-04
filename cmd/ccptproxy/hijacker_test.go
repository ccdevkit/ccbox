package main

import (
	"os"
	"path/filepath"
	"testing"
)

func TestGenerateHijacker_CreatesValidScript(t *testing.T) {
	dir := t.TempDir()
	command := "git"

	if err := GenerateHijacker(dir, command); err != nil {
		t.Fatalf("GenerateHijacker() error = %v", err)
	}

	scriptPath := filepath.Join(dir, command)
	data, err := os.ReadFile(scriptPath)
	if err != nil {
		t.Fatalf("reading script: %v", err)
	}

	want := "#!/bin/sh\nexec ccptproxy --exec git \"$@\"\n"
	if got := string(data); got != want {
		t.Errorf("script content mismatch\ngot:  %q\nwant: %q", got, want)
	}
}

func TestGenerateHijacker_FileIsExecutable(t *testing.T) {
	dir := t.TempDir()

	if err := GenerateHijacker(dir, "docker"); err != nil {
		t.Fatalf("GenerateHijacker() error = %v", err)
	}

	info, err := os.Stat(filepath.Join(dir, "docker"))
	if err != nil {
		t.Fatalf("stat: %v", err)
	}

	mode := info.Mode().Perm()
	if mode&0111 == 0 {
		t.Errorf("script is not executable, mode = %o", mode)
	}
	if mode != 0755 {
		t.Errorf("expected mode 0755, got %o", mode)
	}
}
