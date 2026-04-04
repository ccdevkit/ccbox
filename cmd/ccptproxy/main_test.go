package main

import (
	"os"
	"path/filepath"
	"testing"
)

func TestRunSetup_NoConfigFile_ExitsCleanly(t *testing.T) {
	// When the config file doesn't exist, the --setup path should be a no-op.
	// This is tested at the main() level via the os.Stat guard, but we verify
	// that ReadConfig itself returns an error for a missing file to confirm
	// the guard is necessary.
	_, err := ReadConfig("/nonexistent/path/ccbox-proxy.json")
	if err == nil {
		t.Fatal("expected error for missing config file")
	}
}

func TestRunSetup_GeneratesHijackerScripts(t *testing.T) {
	// Point ContainerShimsDir at a temp dir for this test
	shimsDir := filepath.Join(t.TempDir(), "shims")

	// Temporarily override the shims dir constant via runSetupWithDir
	dir := t.TempDir()
	configPath := filepath.Join(dir, "ccbox-proxy.json")
	err := os.WriteFile(configPath, []byte(`{"hostAddress":"127.0.0.1:9000","passthrough":["git","docker"],"verbose":false}`), 0644)
	if err != nil {
		t.Fatal(err)
	}

	err = runSetupWithDir(configPath, shimsDir)
	if err != nil {
		t.Fatalf("runSetup returned error: %v", err)
	}

	// Verify hijacker scripts were created for each passthrough command
	for _, cmd := range []string{"git", "docker"} {
		path := filepath.Join(shimsDir, cmd)
		info, err := os.Stat(path)
		if err != nil {
			t.Errorf("hijacker script for %q not found: %v", cmd, err)
			continue
		}
		if info.Mode().Perm()&0111 == 0 {
			t.Errorf("hijacker script for %q is not executable", cmd)
		}
	}
}

func TestRunExec_RoutesMatchedCommand(t *testing.T) {
	// We need to test that --exec mode:
	// 1. Reads config
	// 2. Matches command via CommandMatcher
	// 3. Calls SendExec with the matched command
	// 4. Prepends the NOTE to output
	// 5. Returns the remote exit code

	// For this test, we inject a fake sender
	dir := t.TempDir()
	configPath := filepath.Join(dir, "ccbox-proxy.json")
	err := os.WriteFile(configPath, []byte(`{"hostAddress":"127.0.0.1:9000","passthrough":["git","docker"],"verbose":false}`), 0644)
	if err != nil {
		t.Fatal(err)
	}

	var capturedAddr, capturedCmd, capturedCwd string
	fakeSender := func(address, command, cwd string) (int, []byte, error) {
		capturedAddr = address
		capturedCmd = command
		capturedCwd = cwd
		return 0, []byte("fake output"), nil
	}

	stdout, exitCode, err := runExec(configPath, "git status", fakeSender)
	if err != nil {
		t.Fatalf("runExec returned error: %v", err)
	}

	if capturedAddr != "127.0.0.1:9000" {
		t.Errorf("expected address 127.0.0.1:9000, got %q", capturedAddr)
	}
	if capturedCmd != "git status" {
		t.Errorf("expected command %q, got %q", "git status", capturedCmd)
	}
	if capturedCwd == "" {
		t.Error("expected non-empty cwd")
	}
	if exitCode != 0 {
		t.Errorf("expected exit code 0, got %d", exitCode)
	}

	// Verify NOTE is prepended
	expected := "[NOTE: This command was run on the host machine]\nfake output"
	if string(stdout) != expected {
		t.Errorf("expected output %q, got %q", expected, string(stdout))
	}
}

func TestRunExec_OutputPrependedWithNote(t *testing.T) {
	dir := t.TempDir()
	configPath := filepath.Join(dir, "ccbox-proxy.json")
	err := os.WriteFile(configPath, []byte(`{"hostAddress":"127.0.0.1:9000","passthrough":["git"],"verbose":false}`), 0644)
	if err != nil {
		t.Fatal(err)
	}

	fakeSender := func(address, command, cwd string) (int, []byte, error) {
		return 42, []byte("error output"), nil
	}

	stdout, exitCode, err := runExec(configPath, "git push", fakeSender)
	if err != nil {
		t.Fatalf("runExec returned error: %v", err)
	}

	if exitCode != 42 {
		t.Errorf("expected exit code 42, got %d", exitCode)
	}

	expected := "[NOTE: This command was run on the host machine]\nerror output"
	if string(stdout) != expected {
		t.Errorf("expected output %q, got %q", expected, string(stdout))
	}
}

func TestRunExec_UnmatchedCommand(t *testing.T) {
	dir := t.TempDir()
	configPath := filepath.Join(dir, "ccbox-proxy.json")
	err := os.WriteFile(configPath, []byte(`{"hostAddress":"127.0.0.1:9000","passthrough":["git"],"verbose":false}`), 0644)
	if err != nil {
		t.Fatal(err)
	}

	fakeSender := func(address, command, cwd string) (int, []byte, error) {
		t.Fatal("sender should not be called for unmatched command")
		return 0, nil, nil
	}

	_, _, err = runExec(configPath, "ls -la", fakeSender)
	if err == nil {
		t.Fatal("expected error for unmatched command")
	}
}
