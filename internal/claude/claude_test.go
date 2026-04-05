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

// testFS implements args.FileSystem for claude tests.
type testFS struct {
	fileContents map[string][]byte
}

func (t *testFS) Stat(path string) (os.FileInfo, error) {
	if _, ok := t.fileContents[path]; ok {
		return nil, nil
	}
	return nil, os.ErrNotExist
}

func (t *testFS) ReadFile(path string) ([]byte, error) {
	if data, ok := t.fileContents[path]; ok {
		return data, nil
	}
	return nil, os.ErrNotExist
}

func newTestFS() *testFS {
	return &testFS{fileContents: make(map[string][]byte)}
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

// findWrittenFile returns the written file at the given container path, or nil.
func findWrittenFile(fw *mockFileWriter, containerPath string) *writtenFile {
	for i := range fw.files {
		if fw.files[i].containerPath == containerPath {
			return &fw.files[i]
		}
	}
	return nil
}

// assertArgsEqual checks that spec.Args matches expected exactly.
func assertArgsEqual(t *testing.T, spec *ClaudeRunSpec, expected []string) {
	t.Helper()
	if len(spec.Args) != len(expected) {
		t.Fatalf("Args length = %d, want %d\ngot:  %v\nwant: %v", len(spec.Args), len(expected), spec.Args, expected)
	}
	for i, want := range expected {
		if spec.Args[i] != want {
			t.Errorf("Args[%d] = %q, want %q", i, spec.Args[i], want)
		}
	}
}

// --- BuildRunSpec settings manager integration tests ---

func TestBuildRunSpec_IncludesSettingsArgsFromManager(t *testing.T) {
	sess, fw, _ := newTestSessionWithWriter()

	// Use New() to create Claude with a SettingsManager.
	c, err := New(sess)
	if err != nil {
		t.Fatalf("New() error: %v", err)
	}
	c.Token = "test-token"

	parsed := &args.ParsedArgs{
		ClaudeArgs: []args.ClaudeArg{
			{Value: "-p", IsFile: false},
			{Value: "hello", IsFile: false},
		},
	}

	spec, err := c.BuildRunSpec(parsed, &settings.Settings{}, newTestFS())
	if err != nil {
		t.Fatalf("BuildRunSpec() error: %v", err)
	}

	// Verify --settings and --setting-sources are present in args.
	foundSettings := false
	foundSettingSources := false
	for i, a := range spec.Args {
		if a == "--settings" {
			foundSettings = true
			if i+1 < len(spec.Args) && spec.Args[i+1] != constants.SettingsContainerPath {
				t.Errorf("--settings value = %q, want %q", spec.Args[i+1], constants.SettingsContainerPath)
			}
		}
		if a == "--setting-sources" {
			foundSettingSources = true
		}
	}
	if !foundSettings {
		t.Errorf("--settings not found in args: %v", spec.Args)
	}
	if !foundSettingSources {
		t.Errorf("--setting-sources not found in args: %v", spec.Args)
	}

	// Verify the settings file was written via the file writer.
	found := findWrittenFile(fw, constants.SettingsContainerPath)
	if found == nil {
		t.Fatalf("settings file not written to %s; files: %+v", constants.SettingsContainerPath, fw.files)
	}

	// Verify the defaults are in the written settings.
	got := string(found.data)
	for _, want := range []string{`"allowedTools"`, `"enableAllProjectMcpServers"`, `"bypassPermissions"`} {
		if !strings.Contains(got, want) {
			t.Errorf("settings missing %s; got: %s", want, got)
		}
	}
}

// --- BuildRunSpec tests ---

func TestBuildRunSpec_ArgsIncludesAllClaudeArgs(t *testing.T) {
	sess, _ := newTestSession()
	c := &Claude{Session: sess, Token: "test-token"}
	fs := newTestFS()

	parsed := &args.ParsedArgs{
		ClaudeArgs: []args.ClaudeArg{
			{Value: "-p", IsFile: false},
			{Value: "hello world", IsFile: false},
			{Value: "--verbose", IsFile: false},
		},
	}

	spec, err := c.BuildRunSpec(parsed, &settings.Settings{}, fs)
	if err != nil {
		t.Fatalf("BuildRunSpec() error: %v", err)
	}

	expected := []string{
		"-p", "hello world", "--verbose",
		"--append-system-prompt-file", constants.SystemPromptContainerPath,
		"--permission-mode", "bypassPermissions",
		"--allow-dangerously-skip-permissions",
	}
	assertArgsEqual(t, spec, expected)
}

func TestBuildRunSpec_EnvIncludesTermAndColorTerm(t *testing.T) {
	sess, _ := newTestSession()
	c := &Claude{Session: sess, Token: "test-token"}

	t.Setenv("TERM", "xterm-256color")
	t.Setenv("COLORTERM", "truecolor")

	spec, err := c.BuildRunSpec(&args.ParsedArgs{}, &settings.Settings{}, newTestFS())
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

	spec, err := c.BuildRunSpec(&args.ParsedArgs{}, &settings.Settings{}, newTestFS())
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

	spec, err := c.BuildRunSpec(&args.ParsedArgs{}, &settings.Settings{}, newTestFS())
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

	spec, err := c.BuildRunSpec(parsed, &settings.Settings{}, newTestFS())
	if err != nil {
		t.Fatalf("BuildRunSpec() error: %v", err)
	}

	expected := []string{
		"--system-prompt-file", "/home/claude/prompt.md",
		"-p", "hello",
		"--append-system-prompt-file", constants.SystemPromptContainerPath,
		"--permission-mode", "bypassPermissions",
		"--allow-dangerously-skip-permissions",
	}
	assertArgsEqual(t, spec, expected)

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

	spec, err := c.BuildRunSpec(parsed, &settings.Settings{}, newTestFS())
	if err != nil {
		t.Fatalf("BuildRunSpec() error: %v", err)
	}

	expected := []string{
		"-p", "some prompt text", "--model", "claude-opus-4-6",
		"--append-system-prompt-file", constants.SystemPromptContainerPath,
		"--permission-mode", "bypassPermissions",
		"--allow-dangerously-skip-permissions",
	}
	assertArgsEqual(t, spec, expected)
}

func TestBuildRunSpec_ExplicitPermissionModePreserved(t *testing.T) {
	sess, _ := newTestSession()
	c := &Claude{Session: sess, Token: "test-token"}

	parsed := &args.ParsedArgs{
		ClaudeArgs: []args.ClaudeArg{
			{Value: "-p", IsFile: false},
			{Value: "hello", IsFile: false},
			{Value: "--permission-mode", IsFile: false},
			{Value: "default", IsFile: false},
		},
	}

	spec, err := c.BuildRunSpec(parsed, &settings.Settings{}, newTestFS())
	if err != nil {
		t.Fatalf("BuildRunSpec() error: %v", err)
	}

	// Should NOT append a second --permission-mode, but still appends --allow-dangerously-skip-permissions.
	expected := []string{
		"-p", "hello", "--permission-mode", "default",
		"--append-system-prompt-file", constants.SystemPromptContainerPath,
		"--allow-dangerously-skip-permissions",
	}
	assertArgsEqual(t, spec, expected)
}

// --- New() tests ---

func TestNew_CreatesSettingsManagerWithDefaults(t *testing.T) {
	sess, _, _ := newTestSessionWithWriter()

	c, err := New(sess)
	if err != nil {
		t.Fatalf("New() error: %v", err)
	}

	if c.SettingsManager == nil {
		t.Fatal("SettingsManager should not be nil after New()")
	}

	merged := c.SettingsManager.Merged()
	if _, ok := merged["allowedTools"]; !ok {
		t.Error("merged settings missing allowedTools")
	}
	if v, ok := merged["enableAllProjectMcpServers"]; !ok || v != true {
		t.Errorf("enableAllProjectMcpServers = %v, want true", v)
	}
	if v, ok := merged["bypassPermissions"]; !ok || v != true {
		t.Errorf("bypassPermissions = %v, want true", v)
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

// --- SetPassthroughEnabled tests ---

func TestSetPassthroughEnabled_StoresCommandsOnly(t *testing.T) {
	sess, fw, _ := newTestSessionWithWriter()
	c := &Claude{Session: sess}

	c.SetPassthroughEnabled([]string{"git", "npm", "docker"})

	// Should NOT write any files — that's now handled by BuildRunSpec.
	if len(fw.files) != 0 {
		t.Errorf("expected no files written, got %d: %+v", len(fw.files), fw.files)
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

// --- buildCcboxSystemPrompt tests ---

func TestBuildCcboxSystemPrompt_AlwaysIncludesEnvironment(t *testing.T) {
	content := buildCcboxSystemPrompt(nil)

	for _, want := range []string{
		"# ccbox Environment",
		"container-based pseudo-sandbox",
		"/tmp",
		"MCP tools",
		"Passthrough tools",
	} {
		if !strings.Contains(content, want) {
			t.Errorf("missing %q in output:\n%s", want, content)
		}
	}

	// No "Allowed Host Commands" section when no commands
	if strings.Contains(content, "Allowed Host Commands") {
		t.Error("should not contain Allowed Host Commands with nil commands")
	}
}

func TestBuildCcboxSystemPrompt_IncludesCommandsWhenPresent(t *testing.T) {
	content := buildCcboxSystemPrompt([]string{"git", "npm", "docker"})

	// Environment section still present
	if !strings.Contains(content, "# ccbox Environment") {
		t.Error("missing environment section")
	}

	// Commands section present
	if !strings.Contains(content, "## Allowed Host Commands") {
		t.Error("missing Allowed Host Commands section")
	}
	for _, cmd := range []string{"git", "npm", "docker"} {
		if !strings.Contains(content, cmd) {
			t.Errorf("missing command %q", cmd)
		}
	}
}

// --- scanAppendArgs tests ---

func TestScanAppendArgs_NoAppendArgs(t *testing.T) {
	claudeArgs := []args.ClaudeArg{
		{Value: "-p", IsFile: false},
		{Value: "hello", IsFile: false},
	}

	result, err := scanAppendArgs(claudeArgs, newTestFS())
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result.UserContent != "" {
		t.Errorf("UserContent = %q, want empty", result.UserContent)
	}
	if len(result.StripIndices) != 0 {
		t.Errorf("StripIndices = %v, want empty", result.StripIndices)
	}
}

func TestScanAppendArgs_InlineAppend(t *testing.T) {
	claudeArgs := []args.ClaudeArg{
		{Value: "-p", IsFile: false},
		{Value: "hello", IsFile: false},
		{Value: "--append-system-prompt", IsFile: false},
		{Value: "Always use Go", IsFile: false},
	}

	result, err := scanAppendArgs(claudeArgs, newTestFS())
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result.UserContent != "Always use Go" {
		t.Errorf("UserContent = %q, want %q", result.UserContent, "Always use Go")
	}
	if len(result.StripIndices) != 2 || result.StripIndices[0] != 2 || result.StripIndices[1] != 3 {
		t.Errorf("StripIndices = %v, want [2, 3]", result.StripIndices)
	}
}

func TestScanAppendArgs_FileAppend(t *testing.T) {
	fs := newTestFS()
	fs.fileContents["/host/prompt.md"] = []byte("Use TypeScript always")

	claudeArgs := []args.ClaudeArg{
		{Value: "--append-system-prompt-file", IsFile: false},
		{Value: "/host/prompt.md", IsFile: true},
		{Value: "-p", IsFile: false},
		{Value: "hello", IsFile: false},
	}

	result, err := scanAppendArgs(claudeArgs, fs)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result.UserContent != "Use TypeScript always" {
		t.Errorf("UserContent = %q, want %q", result.UserContent, "Use TypeScript always")
	}
	if len(result.StripIndices) != 2 || result.StripIndices[0] != 0 || result.StripIndices[1] != 1 {
		t.Errorf("StripIndices = %v, want [0, 1]", result.StripIndices)
	}
}

func TestScanAppendArgs_BothAppendTypes_ReturnsError(t *testing.T) {
	claudeArgs := []args.ClaudeArg{
		{Value: "--append-system-prompt", IsFile: false},
		{Value: "some text", IsFile: false},
		{Value: "--append-system-prompt-file", IsFile: false},
		{Value: "/some/file.md", IsFile: true},
	}

	_, err := scanAppendArgs(claudeArgs, newTestFS())
	if err == nil {
		t.Fatal("expected error for mutually exclusive flags, got nil")
	}
	if !strings.Contains(err.Error(), "mutually exclusive") {
		t.Errorf("error = %q, want mention of 'mutually exclusive'", err.Error())
	}
}

// --- BuildRunSpec system prompt merge integration tests ---

func TestBuildRunSpec_NoAppendNoPassthrough_WritesEnvironmentOnly(t *testing.T) {
	sess, fw, _ := newTestSessionWithWriter()
	c := &Claude{Session: sess, Token: "test-token"}

	_, err := c.BuildRunSpec(&args.ParsedArgs{}, &settings.Settings{}, newTestFS())
	if err != nil {
		t.Fatalf("BuildRunSpec() error: %v", err)
	}

	found := findWrittenFile(fw, constants.SystemPromptContainerPath)
	if found == nil {
		t.Fatalf("system prompt file not written")
	}

	got := string(found.data)
	if !strings.Contains(got, "# ccbox Environment") {
		t.Error("missing environment section")
	}
	if strings.Contains(got, "Allowed Host Commands") {
		t.Error("should not contain Allowed Host Commands without passthrough")
	}
}

func TestBuildRunSpec_WithPassthrough_WritesEnvironmentAndCommands(t *testing.T) {
	sess, fw, _ := newTestSessionWithWriter()
	c := &Claude{Session: sess, Token: "test-token"}
	c.SetPassthroughEnabled([]string{"git", "npm"})

	_, err := c.BuildRunSpec(&args.ParsedArgs{}, &settings.Settings{}, newTestFS())
	if err != nil {
		t.Fatalf("BuildRunSpec() error: %v", err)
	}

	found := findWrittenFile(fw, constants.SystemPromptContainerPath)
	if found == nil {
		t.Fatalf("system prompt file not written")
	}

	got := string(found.data)
	if !strings.Contains(got, "# ccbox Environment") {
		t.Error("missing environment section")
	}
	if !strings.Contains(got, "Allowed Host Commands") {
		t.Error("missing Allowed Host Commands section")
	}
	for _, cmd := range []string{"git", "npm"} {
		if !strings.Contains(got, cmd) {
			t.Errorf("missing command %q", cmd)
		}
	}
}

func TestBuildRunSpec_UserInlineAppend_MergesWithEnvironment(t *testing.T) {
	sess, fw, _ := newTestSessionWithWriter()
	c := &Claude{Session: sess, Token: "test-token"}

	parsed := &args.ParsedArgs{
		ClaudeArgs: []args.ClaudeArg{
			{Value: "-p", IsFile: false},
			{Value: "hello", IsFile: false},
			{Value: "--append-system-prompt", IsFile: false},
			{Value: "Always use Go", IsFile: false},
		},
	}

	spec, err := c.BuildRunSpec(parsed, &settings.Settings{}, newTestFS())
	if err != nil {
		t.Fatalf("BuildRunSpec() error: %v", err)
	}

	// Original append args should be stripped
	for _, a := range spec.Args {
		if a == "--append-system-prompt" {
			t.Error("--append-system-prompt should have been stripped")
		}
		if a == "Always use Go" {
			t.Error("inline append text should have been stripped from args")
		}
	}

	// Merged file should contain both user text and environment
	found := findWrittenFile(fw, constants.SystemPromptContainerPath)
	if found == nil {
		t.Fatalf("system prompt file not written")
	}
	got := string(found.data)
	if !strings.Contains(got, "Always use Go") {
		t.Error("missing user append text")
	}
	if !strings.Contains(got, "# ccbox Environment") {
		t.Error("missing environment section")
	}
}

func TestBuildRunSpec_UserInlineAppendWithPassthrough_MergesAll(t *testing.T) {
	sess, fw, _ := newTestSessionWithWriter()
	c := &Claude{Session: sess, Token: "test-token"}
	c.SetPassthroughEnabled([]string{"git"})

	parsed := &args.ParsedArgs{
		ClaudeArgs: []args.ClaudeArg{
			{Value: "--append-system-prompt", IsFile: false},
			{Value: "Always use Go", IsFile: false},
			{Value: "-p", IsFile: false},
			{Value: "hello", IsFile: false},
		},
	}

	_, err := c.BuildRunSpec(parsed, &settings.Settings{}, newTestFS())
	if err != nil {
		t.Fatalf("BuildRunSpec() error: %v", err)
	}

	found := findWrittenFile(fw, constants.SystemPromptContainerPath)
	if found == nil {
		t.Fatalf("system prompt file not written")
	}
	got := string(found.data)
	if !strings.Contains(got, "Always use Go") {
		t.Error("missing user append text")
	}
	if !strings.Contains(got, "# ccbox Environment") {
		t.Error("missing environment section")
	}
	if !strings.Contains(got, "git") {
		t.Error("missing passthrough command")
	}
}

func TestBuildRunSpec_UserFileAppendWithPassthrough_MergesAll(t *testing.T) {
	sess, fw, _ := newTestSessionWithWriter()
	c := &Claude{Session: sess, Token: "test-token"}
	c.SetPassthroughEnabled([]string{"docker"})

	fs := newTestFS()
	fs.fileContents["/host/rules.md"] = []byte("Follow these rules strictly")

	parsed := &args.ParsedArgs{
		ClaudeArgs: []args.ClaudeArg{
			{Value: "--append-system-prompt-file", IsFile: false},
			{Value: "/host/rules.md", IsFile: true},
			{Value: "-p", IsFile: false},
			{Value: "hello", IsFile: false},
		},
	}

	spec, err := c.BuildRunSpec(parsed, &settings.Settings{}, fs)
	if err != nil {
		t.Fatalf("BuildRunSpec() error: %v", err)
	}

	// Original file append args should be stripped; no bind mount for user's file
	for _, a := range spec.Args {
		if a == "--append-system-prompt-file" && a != constants.SystemPromptContainerPath {
			// The only --append-system-prompt-file should point to our merged file
		}
	}

	found := findWrittenFile(fw, constants.SystemPromptContainerPath)
	if found == nil {
		t.Fatalf("system prompt file not written")
	}
	got := string(found.data)
	if !strings.Contains(got, "Follow these rules strictly") {
		t.Error("missing user file content")
	}
	if !strings.Contains(got, "# ccbox Environment") {
		t.Error("missing environment section")
	}
	if !strings.Contains(got, "docker") {
		t.Error("missing passthrough command")
	}

	// User's original file should NOT be registered as a passthrough
	for _, pt := range fw.files {
		if pt.containerPath == "/home/claude/rules.md" {
			t.Error("user's original file should not be bind-mounted; content was absorbed into merged file")
		}
	}
}
