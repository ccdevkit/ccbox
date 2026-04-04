package claude

import (
	"fmt"
	"testing"
)

type mockVersionRunner struct {
	output []byte
	err    error
}

func (m *mockVersionRunner) Output(name string, args ...string) ([]byte, error) {
	return m.output, m.err
}

func TestDetectVersion_StandardOutput(t *testing.T) {
	runner := &mockVersionRunner{output: []byte("1.0.16\n")}
	version, err := DetectVersion("/usr/bin/claude", runner)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if version != "1.0.16" {
		t.Errorf("got %q, want %q", version, "1.0.16")
	}
}

func TestDetectVersion_PrefixedOutput(t *testing.T) {
	runner := &mockVersionRunner{output: []byte("claude v2.1.16\n")}
	version, err := DetectVersion("/usr/bin/claude", runner)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if version != "2.1.16" {
		t.Errorf("got %q, want %q", version, "2.1.16")
	}
}

func TestDetectVersion_PreReleaseVersion(t *testing.T) {
	runner := &mockVersionRunner{output: []byte("claude v2.1.16-beta.1\n")}
	version, err := DetectVersion("/usr/bin/claude", runner)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if version != "2.1.16-beta.1" {
		t.Errorf("got %q, want %q", version, "2.1.16-beta.1")
	}
}

func TestDetectVersion_CommandFailure(t *testing.T) {
	runner := &mockVersionRunner{err: fmt.Errorf("command failed: exit status 1")}
	_, err := DetectVersion("/usr/bin/claude", runner)
	if err == nil {
		t.Fatal("expected error, got nil")
	}
}

func TestDetectVersion_NoVersionInOutput(t *testing.T) {
	runner := &mockVersionRunner{output: []byte("some unexpected output\n")}
	_, err := DetectVersion("/usr/bin/claude", runner)
	if err == nil {
		t.Fatal("expected error, got nil")
	}
}

func TestDetectVersion_EmptyOutput(t *testing.T) {
	runner := &mockVersionRunner{output: []byte("")}
	_, err := DetectVersion("/usr/bin/claude", runner)
	if err == nil {
		t.Fatal("expected error, got nil")
	}
}

func TestDetectVersion_PassesCorrectArgs(t *testing.T) {
	var capturedName string
	var capturedArgs []string

	runner := &argCapturingRunner{
		output: []byte("1.0.0\n"),
		captureName: func(name string) { capturedName = name },
		captureArgs: func(args []string) { capturedArgs = args },
	}
	_, err := DetectVersion("/custom/path/claude", runner)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if capturedName != "/custom/path/claude" {
		t.Errorf("got name %q, want %q", capturedName, "/custom/path/claude")
	}
	if len(capturedArgs) != 1 || capturedArgs[0] != "--version" {
		t.Errorf("got args %v, want [--version]", capturedArgs)
	}
}

type argCapturingRunner struct {
	output      []byte
	captureName func(string)
	captureArgs func([]string)
}

func (a *argCapturingRunner) Output(name string, args ...string) ([]byte, error) {
	a.captureName(name)
	a.captureArgs(args)
	return a.output, nil
}
