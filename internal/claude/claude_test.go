package claude

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/ccdevkit/ccbox/internal/args"
	"github.com/ccdevkit/ccbox/internal/constants"
	"github.com/ccdevkit/ccbox/internal/session"
	"github.com/ccdevkit/ccbox/internal/settings"
)

// mockFilePassthrough records AddPassthrough calls for verification.
type mockFilePassthrough struct {
	calls []session.FilePassthrough
}

func (m *mockFilePassthrough) AddPassthrough(hostPath, containerPath string, readOnly bool) error {
	m.calls = append(m.calls, session.FilePassthrough{
		HostPath:      hostPath,
		ContainerPath: containerPath,
		ReadOnly:      readOnly,
	})
	return nil
}

// writtenFile records a single WriteFile call.
type writtenFile struct {
	containerPath string
	data          []byte
}

// mockFileWriter satisfies session.SessionFileWriter and records all writes.
type mockFileWriter struct {
	files []writtenFile
}

func (m *mockFileWriter) WriteFile(containerPath string, data []byte, readOnly bool) error {
	m.files = append(m.files, writtenFile{containerPath: containerPath, data: append([]byte(nil), data...)})
	return nil
}

func newTestSession() (*session.Session, *mockFilePassthrough) {
	fp := &mockFilePassthrough{}
	sess := &session.Session{
		ID:              "test-session-id",
		FileWriter:      &mockFileWriter{},
		FilePassthrough: fp,
	}
	return sess, fp
}

func newTestSessionWithWriter() (*session.Session, *mockFileWriter, *mockFilePassthrough) {
	fw := &mockFileWriter{}
	fp := &mockFilePassthrough{}
	sess := &session.Session{
		ID:              "test-session-id",
		FileWriter:      fw,
		FilePassthrough: fp,
	}
	return sess, fw, fp
}

func TestBuildRunSpec_ArgsIncludesAllClaudeArgs(t *testing.T) {
	sess, _ := newTestSession()
	c := &Claude{Session: sess, Token: "test-token"}

	parsed := &args.ParsedArgs{
		ClaudeArgs: []args.ClaudeArg{
			{Value: "-p", IsFile: false},
			{Value: "hello world", IsFile: false},
			{Value: "--verbose", IsFile: false},
		},
	}

	spec, err := c.BuildRunSpec(parsed, &settings.Settings{})
	if err != nil {
		t.Fatalf("BuildRunSpec() error: %v", err)
	}

	expected := []string{"-p", "hello world", "--verbose"}
	if len(spec.Args) != len(expected) {
		t.Fatalf("Args length = %d, want %d", len(spec.Args), len(expected))
	}
	for i, want := range expected {
		if spec.Args[i] != want {
			t.Errorf("Args[%d] = %q, want %q", i, spec.Args[i], want)
		}
	}
}

func TestBuildRunSpec_EnvIncludesTermAndColorTerm(t *testing.T) {
	sess, _ := newTestSession()
	c := &Claude{Session: sess, Token: "test-token"}

	t.Setenv("TERM", "xterm-256color")
	t.Setenv("COLORTERM", "truecolor")

	spec, err := c.BuildRunSpec(&args.ParsedArgs{}, &settings.Settings{})
	if err != nil {
		t.Fatalf("BuildRunSpec() error: %v", err)
	}

	envMap := make(map[string]EnvVar)
	for _, e := range spec.Env {
		envMap[e.Key] = e
	}

	if ev, ok := envMap["TERM"]; !ok {
		t.Error("TERM not found in Env")
	} else if ev.Value != "xterm-256color" {
		t.Errorf("TERM = %q, want %q", ev.Value, "xterm-256color")
	} else if ev.Secret {
		t.Error("TERM should not be secret")
	}

	if ev, ok := envMap["COLORTERM"]; !ok {
		t.Error("COLORTERM not found in Env")
	} else if ev.Value != "truecolor" {
		t.Errorf("COLORTERM = %q, want %q", ev.Value, "truecolor")
	}
}

func TestBuildRunSpec_EnvIncludesOAuthTokenAsSecret(t *testing.T) {
	sess, _ := newTestSession()
	c := &Claude{Session: sess, Token: "sk-ant-secret123"}

	spec, err := c.BuildRunSpec(&args.ParsedArgs{}, &settings.Settings{})
	if err != nil {
		t.Fatalf("BuildRunSpec() error: %v", err)
	}

	envMap := make(map[string]EnvVar)
	for _, e := range spec.Env {
		envMap[e.Key] = e
	}

	ev, ok := envMap["CLAUDE_CODE_OAUTH_TOKEN"]
	if !ok {
		t.Fatal("CLAUDE_CODE_OAUTH_TOKEN not found in Env")
	}
	if ev.Value != "sk-ant-secret123" {
		t.Errorf("token value = %q, want %q", ev.Value, "sk-ant-secret123")
	}
	if !ev.Secret {
		t.Error("OAuth token should be marked Secret")
	}
}

func TestBuildRunSpec_RegistersCWDAndClaudeDirPassthroughs(t *testing.T) {
	sess, fp := newTestSession()
	c := &Claude{Session: sess, Token: "test-token"}

	spec, err := c.BuildRunSpec(&args.ParsedArgs{}, &settings.Settings{})
	if err != nil {
		t.Fatalf("BuildRunSpec() error: %v", err)
	}
	_ = spec

	cwd, err := os.Getwd()
	if err != nil {
		t.Fatalf("os.Getwd() error: %v", err)
	}

	home, err := os.UserHomeDir()
	if err != nil {
		t.Fatalf("os.UserHomeDir() error: %v", err)
	}
	claudeDir := filepath.Join(home, ".claude")

	// Check CWD mount (rw, identity path)
	foundCWD := false
	foundClaude := false
	for _, pt := range fp.calls {
		if pt.HostPath == cwd && pt.ContainerPath == cwd && !pt.ReadOnly {
			foundCWD = true
		}
		if pt.HostPath == claudeDir && pt.ContainerPath == "/home/claude/.claude" && !pt.ReadOnly {
			foundClaude = true
		}
	}

	if !foundCWD {
		t.Errorf("CWD passthrough not registered; calls: %+v", fp.calls)
	}
	if !foundClaude {
		t.Errorf("~/.claude/ passthrough not registered; calls: %+v", fp.calls)
	}
}

func TestBuildRunSpec_FileArgPathsRewrittenToContainerPaths(t *testing.T) {
	sess, fp := newTestSession()
	c := &Claude{Session: sess, Token: "test-token"}

	parsed := &args.ParsedArgs{
		ClaudeArgs: []args.ClaudeArg{
			{Value: "--system-prompt-file", IsFile: false},
			{Value: "/home/user/prompt.md", IsFile: true},
			{Value: "-p", IsFile: false},
			{Value: "hello", IsFile: false},
		},
	}

	spec, err := c.BuildRunSpec(parsed, &settings.Settings{})
	if err != nil {
		t.Fatalf("BuildRunSpec() error: %v", err)
	}

	// The file arg should be rewritten to container path
	expected := []string{"--system-prompt-file", "/home/claude/prompt.md", "-p", "hello"}
	if len(spec.Args) != len(expected) {
		t.Fatalf("Args length = %d, want %d; got %v", len(spec.Args), len(expected), spec.Args)
	}
	for i, want := range expected {
		if spec.Args[i] != want {
			t.Errorf("Args[%d] = %q, want %q", i, spec.Args[i], want)
		}
	}

	// Verify file passthrough registered (ro)
	foundFile := false
	for _, pt := range fp.calls {
		if pt.HostPath == "/home/user/prompt.md" && pt.ContainerPath == "/home/claude/prompt.md" && pt.ReadOnly {
			foundFile = true
		}
	}
	if !foundFile {
		t.Errorf("file arg passthrough not registered; calls: %+v", fp.calls)
	}
}

func TestBuildRunSpec_NonFileArgsPassThroughUnchanged(t *testing.T) {
	sess, _ := newTestSession()
	c := &Claude{Session: sess, Token: "test-token"}

	parsed := &args.ParsedArgs{
		ClaudeArgs: []args.ClaudeArg{
			{Value: "-p", IsFile: false},
			{Value: "some prompt text", IsFile: false},
			{Value: "--model", IsFile: false},
			{Value: "claude-opus-4-6", IsFile: false},
		},
	}

	spec, err := c.BuildRunSpec(parsed, &settings.Settings{})
	if err != nil {
		t.Fatalf("BuildRunSpec() error: %v", err)
	}

	expected := []string{"-p", "some prompt text", "--model", "claude-opus-4-6"}
	if len(spec.Args) != len(expected) {
		t.Fatalf("Args length = %d, want %d", len(spec.Args), len(expected))
	}
	for i, want := range expected {
		if spec.Args[i] != want {
			t.Errorf("Args[%d] = %q, want %q", i, spec.Args[i], want)
		}
	}
}

func TestNew_WritesSettingsJSON(t *testing.T) {
	sess, fw, _ := newTestSessionWithWriter()

	_, err := New(sess)
	if err != nil {
		t.Fatalf("New() error: %v", err)
	}

	var found *writtenFile
	for i := range fw.files {
		if fw.files[i].containerPath == "/home/claude/.claude/settings.json" {
			found = &fw.files[i]
			break
		}
	}
	if found == nil {
		t.Fatalf("settings.json not written; files: %+v", fw.files)
	}

	got := string(found.data)
	for _, want := range []string{`"allowedTools"`, `"enableAllProjectMcpServers"`, `"bypassPermissions"`} {
		if !strings.Contains(got, want) {
			t.Errorf("settings.json missing %s; got: %s", want, got)
		}
	}
}

func TestNew_MountsClaudeJSON(t *testing.T) {
	// Remove cached file so ensureClaudeJSON creates a fresh one.
	home, _ := os.UserHomeDir()
	hostPath := filepath.Join(home, constants.SettingsDirName, ".claude.json")
	os.Remove(hostPath)
	defer os.Remove(hostPath)

	sess, _, fp := newTestSessionWithWriter()

	_, err := New(sess)
	if err != nil {
		t.Fatalf("New() error: %v", err)
	}

	// .claude.json should be registered as a rw file passthrough.
	var found bool
	for _, c := range fp.calls {
		if c.ContainerPath == "/home/claude/.claude.json" && !c.ReadOnly {
			found = true
			break
		}
	}
	if !found {
		t.Fatalf(".claude.json not mounted as rw passthrough; calls: %+v", fp.calls)
	}

	// Verify the file exists on disk at ~/.ccbox/.claude.json.
	data, err := os.ReadFile(hostPath)
	if err != nil {
		t.Fatalf("~/.ccbox/.claude.json not found: %v", err)
	}
	got := string(data)
	for _, want := range []string{`"hasCompletedOnboarding"`, `"bypassPermissionsModeAccepted"`} {
		if !strings.Contains(got, want) {
			t.Errorf(".claude.json missing %s; got: %s", want, got)
		}
	}
}

func TestSetPassthroughEnabled_WritesSystemPrompt(t *testing.T) {
	sess, fw, _ := newTestSessionWithWriter()
	c := &Claude{Session: sess}

	c.SetPassthroughEnabled([]string{"git", "npm", "docker"})

	var found *writtenFile
	for i := range fw.files {
		if fw.files[i].containerPath == constants.SystemPromptContainerPath {
			found = &fw.files[i]
			break
		}
	}
	if found == nil {
		t.Fatalf("system prompt not written; files: %+v", fw.files)
	}

	got := string(found.data)
	for _, cmd := range []string{"git", "npm", "docker"} {
		if !strings.Contains(got, cmd) {
			t.Errorf("system prompt missing command %q; got: %s", cmd, got)
		}
	}
}

func TestSetPassthroughEnabled_EmptyCommandsIsNoOp(t *testing.T) {
	sess, fw, _ := newTestSessionWithWriter()
	c := &Claude{Session: sess}

	c.SetPassthroughEnabled(nil)
	if len(fw.files) != 0 {
		t.Errorf("expected no files written for nil commands, got %d", len(fw.files))
	}

	c.SetPassthroughEnabled([]string{})
	if len(fw.files) != 0 {
		t.Errorf("expected no files written for empty commands, got %d", len(fw.files))
	}
}

func TestWriteSystemPrompt_ProducesMarkdownWithCommands(t *testing.T) {
	fw := &mockFileWriter{}

	commands := []string{"git", "npm", "docker"}
	err := writeSystemPrompt(fw, commands)
	if err != nil {
		t.Fatalf("writeSystemPrompt() error: %v", err)
	}

	if len(fw.files) != 1 {
		t.Fatalf("expected 1 file written, got %d", len(fw.files))
	}

	if fw.files[0].containerPath != constants.SystemPromptContainerPath {
		t.Errorf("containerPath = %q, want %q", fw.files[0].containerPath, constants.SystemPromptContainerPath)
	}

	got := string(fw.files[0].data)
	for _, cmd := range commands {
		if !strings.Contains(got, cmd) {
			t.Errorf("system prompt missing command %q; got: %s", cmd, got)
		}
	}
}
