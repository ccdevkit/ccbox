package cmdpassthrough

import (
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"

	"github.com/ccdevkit/ccbox/internal/bridge"
)

func TestHandleExec_SuccessfulCommand(t *testing.T) {
	req := bridge.ExecRequest{
		Type:    "exec",
		Command: "echo hello",
		Cwd:     t.TempDir(),
	}

	exitCode, output := HandleExec(req)

	if exitCode != 0 {
		t.Fatalf("expected exit code 0, got %d", exitCode)
	}
	got := strings.TrimSpace(string(output))
	if got != "hello" {
		t.Fatalf("expected output %q, got %q", "hello", got)
	}
}

func TestHandleExec_CommandFailure(t *testing.T) {
	// FR-036: error output and non-zero exit code pass through without special annotation
	req := bridge.ExecRequest{
		Type:    "exec",
		Command: "echo errmsg >&2; exit 42",
		Cwd:     t.TempDir(),
	}

	exitCode, output := HandleExec(req)

	if exitCode != 42 {
		t.Fatalf("expected exit code 42, got %d", exitCode)
	}
	if !strings.Contains(string(output), "errmsg") {
		t.Fatalf("expected output to contain %q, got %q", "errmsg", string(output))
	}
}

func TestHandleExec_CommandNotFound(t *testing.T) {
	req := bridge.ExecRequest{
		Type:    "exec",
		Command: "nonexistent_command_abc123",
		Cwd:     t.TempDir(),
	}

	exitCode, output := HandleExec(req)

	if exitCode == 0 {
		t.Fatalf("expected non-zero exit code for command not found, got 0")
	}
	if len(output) == 0 {
		t.Fatalf("expected some error output for command not found")
	}
}

func TestRewriteContainerPaths_Basic(t *testing.T) {
	input := []byte("File at /home/claude/project/file.go line 10")
	got := rewriteContainerPaths(input, "/home/claude/", "/Users/brad/")
	want := "File at /Users/brad/project/file.go line 10"
	if string(got) != want {
		t.Fatalf("got %q, want %q", string(got), want)
	}
}

func TestRewriteContainerPaths_MultipleOccurrences(t *testing.T) {
	input := []byte("/home/claude/a.go and /home/claude/b.go")
	got := rewriteContainerPaths(input, "/home/claude/", "/Users/brad/")
	want := "/Users/brad/a.go and /Users/brad/b.go"
	if string(got) != want {
		t.Fatalf("got %q, want %q", string(got), want)
	}
}

func TestRewriteContainerPaths_NoMatch(t *testing.T) {
	input := []byte("no container paths here")
	got := rewriteContainerPaths(input, "/home/claude/", "/Users/brad/")
	if string(got) != "no container paths here" {
		t.Fatalf("got %q, want input unchanged", string(got))
	}
}

func TestRewriteContainerPaths_EmptyInput(t *testing.T) {
	got := rewriteContainerPaths(nil, "/home/claude/", "/Users/brad/")
	if got != nil {
		t.Fatalf("expected nil for nil input, got %q", string(got))
	}
}

func TestHandleExec_CwdFromRequest(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("sh -c not available on windows")
	}

	dir := t.TempDir()
	// Create a marker file so we can verify we're in the right directory
	marker := "testmarker.txt"
	if err := os.WriteFile(filepath.Join(dir, marker), []byte("found"), 0644); err != nil {
		t.Fatalf("failed to create marker file: %v", err)
	}

	req := bridge.ExecRequest{
		Type:    "exec",
		Command: "cat " + marker,
		Cwd:     dir,
	}

	exitCode, output := HandleExec(req)

	if exitCode != 0 {
		t.Fatalf("expected exit code 0, got %d; output: %s", exitCode, output)
	}
	got := strings.TrimSpace(string(output))
	if got != "found" {
		t.Fatalf("expected output %q, got %q", "found", got)
	}
}
