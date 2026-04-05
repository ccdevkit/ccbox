package hooks

import (
	"context"
	"encoding/json"
	"fmt"
	"sync"
	"sync/atomic"
	"testing"
)

// mockRunner records calls and returns preset results.
type mockRunner struct {
	mu      sync.Mutex
	calls   []mockRunCall
	results []mockRunResult
	callIdx atomic.Int32
}

type mockRunCall struct {
	Command string
	Stdin   []byte
	Env     []string
	Dir     string
}

type mockRunResult struct {
	ExitCode int
	Stdout   []byte
	Stderr   []byte
	Err      error
}

func (m *mockRunner) Run(_ context.Context, command string, stdin []byte, env []string, dir string) (int, []byte, []byte, error) {
	idx := int(m.callIdx.Add(1)) - 1
	m.mu.Lock()
	m.calls = append(m.calls, mockRunCall{Command: command, Stdin: stdin, Env: env, Dir: dir})
	m.mu.Unlock()
	if idx < len(m.results) {
		r := m.results[idx]
		return r.ExitCode, r.Stdout, r.Stderr, r.Err
	}
	return 0, nil, nil, nil
}

func bashInput(command string) json.RawMessage {
	data, _ := json.Marshal(map[string]interface{}{
		"hook_event_name": "PreToolUse",
		"tool_name":       "Bash",
		"tool_input":      map[string]interface{}{"command": command},
	})
	return data
}

func sessionStartInput() json.RawMessage {
	data, _ := json.Marshal(map[string]interface{}{
		"hook_event_name": "SessionStart",
	})
	return data
}

// --- Single hook execution ---

func TestExecuteUserHooks_SingleHook(t *testing.T) {
	runner := &mockRunner{
		results: []mockRunResult{
			{ExitCode: 0, Stdout: []byte(`{"continue":true}`)},
		},
	}
	hooks := []UserHook{
		{Command: "echo ok", Matcher: "Bash"},
	}
	input := bashInput("npm test")
	env := []string{"CLAUDE_PROJECT_DIR=/home/user/project"}

	result := ExecuteUserHooks(context.Background(), runner, nil, hooks, PreToolUse, input, env, "/work")

	if result.ExitCode != 0 {
		t.Errorf("ExitCode = %d, want 0", result.ExitCode)
	}
	if len(runner.calls) != 1 {
		t.Fatalf("expected 1 call, got %d", len(runner.calls))
	}
	if runner.calls[0].Command != "echo ok" {
		t.Errorf("command = %q, want %q", runner.calls[0].Command, "echo ok")
	}
	if runner.calls[0].Dir != "/work" {
		t.Errorf("dir = %q, want %q", runner.calls[0].Dir, "/work")
	}
}

// --- Multiple hooks parallel ---

func TestExecuteUserHooks_MultipleHooksParallel(t *testing.T) {
	runner := &mockRunner{
		results: []mockRunResult{
			{ExitCode: 0, Stdout: []byte(`{}`)},
			{ExitCode: 0, Stdout: []byte(`{}`)},
			{ExitCode: 0, Stdout: []byte(`{}`)},
		},
	}
	hooks := []UserHook{
		{Command: "hook1", Matcher: "Bash"},
		{Command: "hook2", Matcher: "Bash"},
		{Command: "hook3", Matcher: "Bash"},
	}
	input := bashInput("npm test")

	result := ExecuteUserHooks(context.Background(), runner, nil, hooks, PreToolUse, input, nil, "/work")

	if result.ExitCode != 0 {
		t.Errorf("ExitCode = %d, want 0", result.ExitCode)
	}
	if len(runner.calls) != 3 {
		t.Errorf("expected 3 calls, got %d", len(runner.calls))
	}
}

// --- Result aggregation: deny beats allow ---

func TestExecuteUserHooks_DenyBeatsAllow(t *testing.T) {
	denyJSON, _ := json.Marshal(map[string]interface{}{
		"hookSpecificOutput": map[string]interface{}{
			"hookEventName":      "PreToolUse",
			"permissionDecision": "deny",
		},
	})
	allowJSON, _ := json.Marshal(map[string]interface{}{
		"hookSpecificOutput": map[string]interface{}{
			"hookEventName":      "PreToolUse",
			"permissionDecision": "allow",
		},
	})

	runner := &mockRunner{
		results: []mockRunResult{
			{ExitCode: 0, Stdout: allowJSON},
			{ExitCode: 0, Stdout: denyJSON},
		},
	}
	hooks := []UserHook{
		{Command: "allow-hook", Matcher: "Bash"},
		{Command: "deny-hook", Matcher: "Bash"},
	}

	result := ExecuteUserHooks(context.Background(), runner, nil, hooks, PreToolUse, bashInput("test"), nil, "/work")

	if result.ExitCode != 0 {
		t.Errorf("ExitCode = %d, want 0", result.ExitCode)
	}
	decision := extractDecision(result.Stdout)
	if decision != "deny" {
		t.Errorf("decision = %q, want %q", decision, "deny")
	}
}

// --- Block (exit 2) beats error (exit 1) beats success ---

func TestExecuteUserHooks_BlockBeatsError(t *testing.T) {
	runner := &mockRunner{
		results: []mockRunResult{
			{ExitCode: 1, Stderr: []byte("some error")},
			{ExitCode: 2, Stderr: []byte("blocked!")},
		},
	}
	hooks := []UserHook{
		{Command: "error-hook", Matcher: "Bash"},
		{Command: "block-hook", Matcher: "Bash"},
	}

	result := ExecuteUserHooks(context.Background(), runner, nil, hooks, PreToolUse, bashInput("test"), nil, "/work")

	if result.ExitCode != 2 {
		t.Errorf("ExitCode = %d, want 2", result.ExitCode)
	}
}

// --- continue:false beats all ---

func TestExecuteUserHooks_ContinueFalseBeatsAll(t *testing.T) {
	stopJSON, _ := json.Marshal(map[string]interface{}{
		"continue":   false,
		"stopReason": "halted",
	})

	runner := &mockRunner{
		results: []mockRunResult{
			{ExitCode: 2, Stderr: []byte("blocked")},
			{ExitCode: 0, Stdout: stopJSON},
		},
	}
	hooks := []UserHook{
		{Command: "block-hook", Matcher: "Bash"},
		{Command: "stop-hook", Matcher: "Bash"},
	}

	result := ExecuteUserHooks(context.Background(), runner, nil, hooks, PreToolUse, bashInput("test"), nil, "/work")

	if result.ExitCode != 0 {
		t.Errorf("ExitCode = %d, want 0 (continue:false uses exit 0)", result.ExitCode)
	}
	stop, reason := extractContinueFalse(result.Stdout)
	if !stop {
		t.Error("expected continue:false in result")
	}
	if reason != "halted" {
		t.Errorf("reason = %q, want %q", reason, "halted")
	}
}

// --- Matcher filtering: non-matching hooks not executed ---

func TestExecuteUserHooks_MatcherFiltering(t *testing.T) {
	runner := &mockRunner{
		results: []mockRunResult{
			{ExitCode: 0, Stdout: []byte(`{}`)},
		},
	}
	hooks := []UserHook{
		{Command: "bash-hook", Matcher: "Bash"},
		{Command: "edit-hook", Matcher: "Edit"}, // should NOT run for Bash input
	}

	result := ExecuteUserHooks(context.Background(), runner, nil, hooks, PreToolUse, bashInput("test"), nil, "/work")

	if result.ExitCode != 0 {
		t.Errorf("ExitCode = %d, want 0", result.ExitCode)
	}
	if len(runner.calls) != 1 {
		t.Fatalf("expected 1 call (only bash-hook), got %d", len(runner.calls))
	}
	if runner.calls[0].Command != "bash-hook" {
		t.Errorf("command = %q, want %q", runner.calls[0].Command, "bash-hook")
	}
}

// --- If-field filtering ---

func TestExecuteUserHooks_IfFieldFiltering(t *testing.T) {
	runner := &mockRunner{
		results: []mockRunResult{
			{ExitCode: 0, Stdout: []byte(`{}`)},
		},
	}
	hooks := []UserHook{
		{Command: "rm-hook", Matcher: "Bash", If: "Bash(rm *)"},
		{Command: "all-hook", Matcher: "Bash"}, // no if, should run
	}

	// Input is "npm test" — rm hook should NOT match
	result := ExecuteUserHooks(context.Background(), runner, nil, hooks, PreToolUse, bashInput("npm test"), nil, "/work")

	if result.ExitCode != 0 {
		t.Errorf("ExitCode = %d, want 0", result.ExitCode)
	}
	if len(runner.calls) != 1 {
		t.Fatalf("expected 1 call (only all-hook), got %d", len(runner.calls))
	}
	if runner.calls[0].Command != "all-hook" {
		t.Errorf("command = %q, want %q", runner.calls[0].Command, "all-hook")
	}
}

// --- Non-tool event with if field: hook never fires ---

func TestExecuteUserHooks_NonToolEventWithIf(t *testing.T) {
	runner := &mockRunner{}
	hooks := []UserHook{
		{Command: "should-not-run", If: "Bash(rm *)"},
	}

	result := ExecuteUserHooks(context.Background(), runner, nil, hooks, SessionStart, sessionStartInput(), nil, "/work")

	if result.ExitCode != 0 {
		t.Errorf("ExitCode = %d, want 0", result.ExitCode)
	}
	if len(runner.calls) != 0 {
		t.Errorf("expected 0 calls, got %d", len(runner.calls))
	}
}

// --- All hooks filtered out → default success ---

func TestExecuteUserHooks_AllFilteredOut(t *testing.T) {
	runner := &mockRunner{}
	hooks := []UserHook{
		{Command: "edit-only", Matcher: "Edit"},
	}

	result := ExecuteUserHooks(context.Background(), runner, nil, hooks, PreToolUse, bashInput("test"), nil, "/work")

	if result.ExitCode != 0 {
		t.Errorf("ExitCode = %d, want 0", result.ExitCode)
	}
	if len(runner.calls) != 0 {
		t.Errorf("expected 0 calls, got %d", len(runner.calls))
	}
}

// --- Subprocess crash (runner returns error) ---

func TestExecuteUserHooks_RunnerError(t *testing.T) {
	runner := &mockRunner{
		results: []mockRunResult{
			{Err: fmt.Errorf("process failed to start")},
		},
	}
	hooks := []UserHook{
		{Command: "bad-hook", Matcher: "Bash"},
	}

	result := ExecuteUserHooks(context.Background(), runner, nil, hooks, PreToolUse, bashInput("test"), nil, "/work")

	// Runner error should produce exit code 1
	if result.ExitCode != 1 {
		t.Errorf("ExitCode = %d, want 1", result.ExitCode)
	}
}

// --- Environment variables passed correctly ---

func TestExecuteUserHooks_EnvironmentPassed(t *testing.T) {
	runner := &mockRunner{
		results: []mockRunResult{
			{ExitCode: 0},
		},
	}
	hooks := []UserHook{
		{Command: "env-hook", Matcher: "Bash"},
	}
	env := []string{"CLAUDE_PROJECT_DIR=/home/user/project", "FOO=bar"}

	ExecuteUserHooks(context.Background(), runner, nil, hooks, PreToolUse, bashInput("test"), env, "/work")

	if len(runner.calls) != 1 {
		t.Fatalf("expected 1 call, got %d", len(runner.calls))
	}
	call := runner.calls[0]
	foundProject := false
	foundFoo := false
	for _, e := range call.Env {
		if e == "CLAUDE_PROJECT_DIR=/home/user/project" {
			foundProject = true
		}
		if e == "FOO=bar" {
			foundFoo = true
		}
	}
	if !foundProject {
		t.Error("CLAUDE_PROJECT_DIR not passed to subprocess")
	}
	if !foundFoo {
		t.Error("FOO not passed to subprocess")
	}
}

// --- Empty hooks list ---

func TestExecuteUserHooks_EmptyHooksList(t *testing.T) {
	runner := &mockRunner{}
	result := ExecuteUserHooks(context.Background(), runner, nil, nil, PreToolUse, bashInput("test"), nil, "/work")

	if result.ExitCode != 0 {
		t.Errorf("ExitCode = %d, want 0", result.ExitCode)
	}
}

// --- Stdin contains the input JSON ---

func TestExecuteUserHooks_StdinIsInputJSON(t *testing.T) {
	runner := &mockRunner{
		results: []mockRunResult{{ExitCode: 0}},
	}
	hooks := []UserHook{
		{Command: "stdin-hook", Matcher: "Bash"},
	}
	input := bashInput("npm test")

	ExecuteUserHooks(context.Background(), runner, nil, hooks, PreToolUse, input, nil, "/work")

	if len(runner.calls) != 1 {
		t.Fatalf("expected 1 call, got %d", len(runner.calls))
	}
	// Verify stdin is the raw input JSON
	if string(runner.calls[0].Stdin) != string(input) {
		t.Errorf("stdin = %q, want %q", string(runner.calls[0].Stdin), string(input))
	}
}
