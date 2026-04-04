package args

import (
	"os"
	"testing"
)

// mockFS implements FileSystem for testing.
type mockFS struct {
	files map[string]bool // path → exists
}

func (m *mockFS) Stat(path string) (os.FileInfo, error) {
	if m.files[path] {
		return nil, nil // existence is all we check
	}
	return nil, os.ErrNotExist
}

func newMockFS(paths ...string) *mockFS {
	m := &mockFS{files: make(map[string]bool)}
	for _, p := range paths {
		m.files[p] = true
	}
	return m
}

func TestParse_SplitOnDoubleDash(t *testing.T) {
	fs := newMockFS()

	result, err := Parse([]string{"--verbose", "--", "-p", "hello"}, fs)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !result.Verbose {
		t.Error("expected Verbose to be true")
	}
	if len(result.ClaudeArgs) != 2 {
		t.Fatalf("expected 2 claude args, got %d", len(result.ClaudeArgs))
	}
	if result.ClaudeArgs[0].Value != "-p" {
		t.Errorf("expected first claude arg to be '-p', got %q", result.ClaudeArgs[0].Value)
	}
	if result.ClaudeArgs[1].Value != "hello" {
		t.Errorf("expected second claude arg to be 'hello', got %q", result.ClaudeArgs[1].Value)
	}
}

func TestParse_NoDoubleDash(t *testing.T) {
	fs := newMockFS()

	result, err := Parse([]string{"--verbose"}, fs)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !result.Verbose {
		t.Error("expected Verbose to be true")
	}
	if len(result.ClaudeArgs) != 0 {
		t.Errorf("expected 0 claude args, got %d", len(result.ClaudeArgs))
	}
}

func TestParse_PtPrefixParsing(t *testing.T) {
	fs := newMockFS()

	result, err := Parse([]string{"-pt:git", "-pt:docker"}, fs)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(result.Passthrough) != 2 {
		t.Fatalf("expected 2 passthrough entries, got %d", len(result.Passthrough))
	}
	if result.Passthrough[0] != "git" {
		t.Errorf("expected first passthrough to be 'git', got %q", result.Passthrough[0])
	}
	if result.Passthrough[1] != "docker" {
		t.Errorf("expected second passthrough to be 'docker', got %q", result.Passthrough[1])
	}
}

func TestParse_PassthroughLongForm(t *testing.T) {
	fs := newMockFS()

	result, err := Parse([]string{"--passthrough", "git", "--passthrough", "docker"}, fs)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(result.Passthrough) != 2 {
		t.Fatalf("expected 2 passthrough entries, got %d", len(result.Passthrough))
	}
	if result.Passthrough[0] != "git" {
		t.Errorf("expected first passthrough to be 'git', got %q", result.Passthrough[0])
	}
	if result.Passthrough[1] != "docker" {
		t.Errorf("expected second passthrough to be 'docker', got %q", result.Passthrough[1])
	}
}

func TestParse_ClaudePath(t *testing.T) {
	fs := newMockFS()

	result, err := Parse([]string{"-c", "/usr/local/bin/claude"}, fs)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result.ClaudePath != "/usr/local/bin/claude" {
		t.Errorf("expected ClaudePath '/usr/local/bin/claude', got %q", result.ClaudePath)
	}
}

func TestParse_ClaudePathLongForm(t *testing.T) {
	fs := newMockFS()

	result, err := Parse([]string{"--claudePath", "/opt/claude"}, fs)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result.ClaudePath != "/opt/claude" {
		t.Errorf("expected ClaudePath '/opt/claude', got %q", result.ClaudePath)
	}
}

func TestParse_UseFlag(t *testing.T) {
	fs := newMockFS()

	result, err := Parse([]string{"--use", "2.1.16"}, fs)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result.Use != "2.1.16" {
		t.Errorf("expected Use '2.1.16', got %q", result.Use)
	}
}

func TestParse_LogFlag(t *testing.T) {
	fs := newMockFS()

	result, err := Parse([]string{"--log", "/tmp/debug.log"}, fs)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result.LogFile != "/tmp/debug.log" {
		t.Errorf("expected LogFile '/tmp/debug.log', got %q", result.LogFile)
	}
}

func TestParse_VersionFlag(t *testing.T) {
	fs := newMockFS()

	result, err := Parse([]string{"--version"}, fs)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !result.Version {
		t.Error("expected Version to be true")
	}
}

func TestParse_HelpFlag(t *testing.T) {
	fs := newMockFS()

	result, err := Parse([]string{"--help"}, fs)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !result.Help {
		t.Error("expected Help to be true")
	}
}

func TestParse_HelpShortFlag(t *testing.T) {
	fs := newMockFS()

	result, err := Parse([]string{"-h"}, fs)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !result.Help {
		t.Error("expected Help to be true")
	}
}

func TestParse_SubcommandUpdate(t *testing.T) {
	fs := newMockFS()

	result, err := Parse([]string{"update"}, fs)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result.Subcommand != "update" {
		t.Errorf("expected Subcommand 'update', got %q", result.Subcommand)
	}
}

func TestParse_SubcommandClean(t *testing.T) {
	fs := newMockFS()

	result, err := Parse([]string{"clean"}, fs)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result.Subcommand != "clean" {
		t.Errorf("expected Subcommand 'clean', got %q", result.Subcommand)
	}
}

func TestParse_CleanAll(t *testing.T) {
	fs := newMockFS()

	result, err := Parse([]string{"clean", "--all"}, fs)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result.Subcommand != "clean" {
		t.Errorf("expected Subcommand 'clean', got %q", result.Subcommand)
	}
	if !result.CleanAll {
		t.Error("expected CleanAll to be true")
	}
}

func TestParse_AllWithoutClean(t *testing.T) {
	fs := newMockFS()

	_, err := Parse([]string{"--all"}, fs)
	if err == nil {
		t.Fatal("expected error for --all without clean subcommand")
	}
}

func TestParse_ClaudeArgIsFile_ExistingFile(t *testing.T) {
	fs := newMockFS("/tmp/prompt.md")

	result, err := Parse([]string{"--", "--system-prompt-file", "/tmp/prompt.md"}, fs)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(result.ClaudeArgs) != 2 {
		t.Fatalf("expected 2 claude args, got %d", len(result.ClaudeArgs))
	}
	// The flag itself is not a file
	if result.ClaudeArgs[0].IsFile {
		t.Error("expected --system-prompt-file flag itself to not be IsFile")
	}
	// The value after --system-prompt-file is a file
	if !result.ClaudeArgs[1].IsFile {
		t.Error("expected /tmp/prompt.md to be IsFile:true")
	}
}

func TestParse_ClaudeArgIsFile_NonExistentPath(t *testing.T) {
	fs := newMockFS() // no files exist

	result, err := Parse([]string{"--", "--system-prompt-file", "/tmp/nonexistent.md"}, fs)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(result.ClaudeArgs) != 2 {
		t.Fatalf("expected 2 claude args, got %d", len(result.ClaudeArgs))
	}
	if result.ClaudeArgs[1].IsFile {
		t.Error("expected non-existent path to be IsFile:false")
	}
}

func TestParse_ClaudeArgIsFile_HeuristicAbsolutePath(t *testing.T) {
	fs := newMockFS("/home/user/file.txt")

	result, err := Parse([]string{"--", "-p", "/home/user/file.txt"}, fs)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(result.ClaudeArgs) != 2 {
		t.Fatalf("expected 2 claude args, got %d", len(result.ClaudeArgs))
	}
	if !result.ClaudeArgs[1].IsFile {
		t.Error("expected /home/user/file.txt to be IsFile:true (heuristic + stat)")
	}
}

func TestParse_ClaudeArgIsFile_HeuristicRelativePath(t *testing.T) {
	fs := newMockFS("./myfile.txt")

	result, err := Parse([]string{"--", "./myfile.txt"}, fs)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(result.ClaudeArgs) != 1 {
		t.Fatalf("expected 1 claude arg, got %d", len(result.ClaudeArgs))
	}
	if !result.ClaudeArgs[0].IsFile {
		t.Error("expected ./myfile.txt to be IsFile:true")
	}
}

func TestParse_ClaudeArgIsFile_HeuristicParentPath(t *testing.T) {
	fs := newMockFS("../other.txt")

	result, err := Parse([]string{"--", "../other.txt"}, fs)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !result.ClaudeArgs[0].IsFile {
		t.Error("expected ../other.txt to be IsFile:true")
	}
}

func TestParse_ClaudeArgIsFile_HeuristicTildePath(t *testing.T) {
	fs := newMockFS("~/notes.txt")

	result, err := Parse([]string{"--", "~/notes.txt"}, fs)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !result.ClaudeArgs[0].IsFile {
		t.Error("expected ~/notes.txt to be IsFile:true")
	}
}

func TestParse_ClaudeArgIsFile_NonPathString(t *testing.T) {
	fs := newMockFS()

	result, err := Parse([]string{"--", "-p", "hello world"}, fs)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result.ClaudeArgs[0].IsFile {
		t.Error("expected -p flag to not be IsFile")
	}
	if result.ClaudeArgs[1].IsFile {
		t.Error("expected 'hello world' to not be IsFile")
	}
}

func TestParse_SemanticFlagResumeMarksFile(t *testing.T) {
	fs := newMockFS("/tmp/session.json")

	result, err := Parse([]string{"--", "--resume", "/tmp/session.json"}, fs)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !result.ClaudeArgs[1].IsFile {
		t.Error("expected --resume value to be IsFile:true")
	}
}

func TestParse_MixedFlags(t *testing.T) {
	fs := newMockFS()

	result, err := Parse([]string{"-pt:git", "-v", "--claudePath", "/opt/claude", "--", "-p", "hello"}, fs)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(result.Passthrough) != 1 || result.Passthrough[0] != "git" {
		t.Errorf("expected passthrough [git], got %v", result.Passthrough)
	}
	if !result.Verbose {
		t.Error("expected Verbose to be true")
	}
	if result.ClaudePath != "/opt/claude" {
		t.Errorf("expected ClaudePath '/opt/claude', got %q", result.ClaudePath)
	}
	if len(result.ClaudeArgs) != 2 {
		t.Fatalf("expected 2 claude args, got %d", len(result.ClaudeArgs))
	}
}

func TestParse_EmptyArgs(t *testing.T) {
	fs := newMockFS()

	result, err := Parse([]string{}, fs)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result.Verbose || result.Version || result.Help {
		t.Error("expected all bool flags to be false for empty args")
	}
	if len(result.ClaudeArgs) != 0 {
		t.Error("expected no claude args")
	}
	if len(result.Passthrough) != 0 {
		t.Error("expected no passthrough entries")
	}
}

func TestParse_UnknownCcboxFlag(t *testing.T) {
	fs := newMockFS()

	_, err := Parse([]string{"--unknown"}, fs)
	if err == nil {
		t.Error("expected error for unknown flag")
	}
}

func TestParse_MissingFlagValue(t *testing.T) {
	fs := newMockFS()

	_, err := Parse([]string{"--claudePath"}, fs)
	if err == nil {
		t.Error("expected error for missing flag value")
	}
}

func TestParse_WindowsPathHeuristic(t *testing.T) {
	// Windows-style paths like C:\foo should not trigger heuristic
	fs := newMockFS()

	result, err := Parse([]string{"--", "C:\\Users\\foo"}, fs)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result.ClaudeArgs[0].IsFile {
		t.Error("expected Windows path to not trigger file heuristic")
	}
}

func TestParse_HeuristicPathNotExist(t *testing.T) {
	fs := newMockFS() // path doesn't exist

	result, err := Parse([]string{"--", "/nonexistent/path.txt"}, fs)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result.ClaudeArgs[0].IsFile {
		t.Error("expected heuristic path that doesn't exist to be IsFile:false")
	}
}

func TestParse_SubcommandWithFlags(t *testing.T) {
	fs := newMockFS()

	result, err := Parse([]string{"-v", "update"}, fs)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result.Subcommand != "update" {
		t.Errorf("expected Subcommand 'update', got %q", result.Subcommand)
	}
	if !result.Verbose {
		t.Error("expected Verbose to be true")
	}
}

func TestParse_UnknownSubcommand(t *testing.T) {
	fs := newMockFS()

	_, err := Parse([]string{"foobar"}, fs)
	if err == nil {
		t.Error("expected error for unknown positional argument")
	}
}
