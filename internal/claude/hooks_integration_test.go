package claude

import (
	"context"
	"encoding/json"
	"os"
	"path/filepath"
	"testing"

	"github.com/ccdevkit/ccbox/internal/bridge"
	"github.com/ccdevkit/ccbox/internal/claude/hooks"
	"github.com/ccdevkit/ccbox/internal/claude/settings"
	"github.com/ccdevkit/ccbox/internal/constants"
)

// mockSettingsFS implements settings.FS and returns ErrNotExist for all files.
type mockSettingsFS struct{}

func (mockSettingsFS) ReadFile(string) ([]byte, error)    { return nil, os.ErrNotExist }
func (mockSettingsFS) Stat(string) (os.FileInfo, error)   { return nil, os.ErrNotExist }

func TestHookIntegration_EndToEnd(t *testing.T) {
	proxyCmd := "/opt/ccbox/bin/cchookproxy"

	// 1. Create hook registry
	registry := hooks.NewRegistry()

	// 2. Register a PreToolUse handler for "Bash"
	handlerCalled := false
	registry.Register(hooks.PreToolUseHandler{
		Matcher: "Bash",
		Order:   hooks.OrderBefore,
		Fn: func(input *hooks.PreToolUseInput) (*hooks.PreToolUseOutput, error) {
			handlerCalled = true
			if input.ToolName != "Bash" {
				t.Errorf("expected tool_name Bash, got %s", input.ToolName)
			}
			return &hooks.PreToolUseOutput{}, nil
		},
	})

	// 3. Create Settings Manager with mock FS (no real files)
	mgr, err := settings.NewClaudeSettingsManager(mockSettingsFS{}, "/home/testuser", "/project")
	if err != nil {
		t.Fatalf("NewClaudeSettingsManager() error: %v", err)
	}

	// 4. Capture user hooks and replace with catch-all proxy entries
	mgr.CaptureAndReplaceHooks(proxyCmd, registry.RegisteredEvents())

	// 5. Finalize — write settings to mock file writer
	fw := &mockFileWriter{}
	args, err := mgr.Finalize(fw)
	if err != nil {
		t.Fatalf("Finalize() error: %v", err)
	}

	// 6. Verify CLI args contain --settings and --setting-sources
	foundSettings := false
	foundSettingSources := false
	for _, a := range args {
		if a == "--settings" {
			foundSettings = true
		}
		if a == "--setting-sources" {
			foundSettingSources = true
		}
	}
	if !foundSettings {
		t.Errorf("--settings not found in args: %v", args)
	}
	if !foundSettingSources {
		t.Errorf("--setting-sources not found in args: %v", args)
	}

	// 7. Verify settings file was written and contains hook config
	written := findWrittenFile(fw, constants.SettingsContainerPath)
	if written == nil {
		t.Fatalf("settings file not written to %s", constants.SettingsContainerPath)
	}

	var settingsJSON map[string]interface{}
	if err := json.Unmarshal(written.data, &settingsJSON); err != nil {
		t.Fatalf("unmarshal settings: %v", err)
	}

	hooksRaw, ok := settingsJSON["hooks"]
	if !ok {
		t.Fatal("settings should contain hooks section")
	}
	hooksMap, ok := hooksRaw.(map[string]interface{})
	if !ok {
		t.Fatalf("hooks should be a map, got %T", hooksRaw)
	}

	preToolUseRaw, ok := hooksMap["PreToolUse"]
	if !ok {
		t.Fatal("hooks should contain PreToolUse event")
	}
	matcherGroups, ok := preToolUseRaw.([]interface{})
	if !ok {
		t.Fatalf("PreToolUse should be an array, got %T", preToolUseRaw)
	}
	if len(matcherGroups) == 0 {
		t.Fatal("PreToolUse should have at least one matcher group")
	}

	// Verify the matcher group has "*" matcher (catch-all) and hooks pointing to cchookproxy
	group0, ok := matcherGroups[0].(map[string]interface{})
	if !ok {
		t.Fatalf("matcher group should be a map, got %T", matcherGroups[0])
	}
	if matcher, _ := group0["matcher"].(string); matcher != "*" {
		t.Errorf("matcher = %q, want %q (catch-all)", matcher, "*")
	}
	groupHooks, ok := group0["hooks"].([]interface{})
	if !ok {
		t.Fatalf("hooks in matcher group should be an array, got %T", group0["hooks"])
	}
	if len(groupHooks) == 0 {
		t.Fatal("matcher group should have at least one hook entry")
	}
	entry0, ok := groupHooks[0].(map[string]interface{})
	if !ok {
		t.Fatalf("hook entry should be a map, got %T", groupHooks[0])
	}
	if entry0["type"] != "command" {
		t.Errorf("hook type = %v, want %q", entry0["type"], "command")
	}
	if entry0["command"] != proxyCmd {
		t.Errorf("hook command = %v, want %q", entry0["command"], proxyCmd)
	}

	// 8. Verify bridge handler dispatches correctly
	bridgeHandler := registry.BridgeHandler()
	resp := bridgeHandler(bridge.HookRequest{
		Type:  "hook",
		Event: "PreToolUse",
		Input: json.RawMessage(`{
			"hook_event_name": "PreToolUse",
			"tool_name": "Bash",
			"tool_input": {},
			"session_id": "test-session",
			"transcript_path": "/tmp/transcript",
			"cwd": "/project",
			"permission_mode": "default"
		}`),
	})
	if resp.ExitCode != 0 {
		t.Errorf("bridge handler exit code = %d, want 0; stderr: %s", resp.ExitCode, resp.Stderr)
	}
	if !handlerCalled {
		t.Error("PreToolUse handler was not called during bridge dispatch")
	}
}

// mockCmdRunner records calls for verification.
type mockCmdRunner struct {
	exitCode int
	stdout   []byte
	stderr   []byte
	called   bool
}

func (m *mockCmdRunner) Run(_ context.Context, command string, stdin []byte, env []string, dir string) (int, []byte, []byte, error) {
	m.called = true
	return m.exitCode, m.stdout, m.stderr, nil
}

func TestHookIntegration_UserHooksCapturedAndDispatched(t *testing.T) {
	proxyCmd := filepath.Join(constants.ContainerBinDir, constants.HookProxyBinaryName)

	// 1. Create settings with user hooks
	userSettings := map[string]interface{}{
		"hooks": map[string]interface{}{
			"PreToolUse": []interface{}{
				map[string]interface{}{
					"matcher": "Bash",
					"hooks": []interface{}{
						map[string]interface{}{
							"type":    "command",
							"command": "/user/hooks/check-bash.sh",
							"if":      "Bash(rm *)",
						},
					},
				},
			},
		},
	}

	settingsJSON, _ := json.Marshal(userSettings)
	fs := mockSettingsFSWithFiles{
		files: map[string][]byte{
			"/home/testuser/.claude/settings.json": settingsJSON,
		},
	}

	// 2. Create registry with a before handler
	registry := hooks.NewRegistry()
	beforeCalled := false
	registry.Register(hooks.PreToolUseHandler{
		Matcher: "Bash",
		Order:   hooks.OrderBefore,
		Fn: func(input *hooks.PreToolUseInput) (*hooks.PreToolUseOutput, error) {
			beforeCalled = true
			return nil, nil
		},
	})

	// 3. Use a mock command runner to avoid real subprocess
	runner := &mockCmdRunner{exitCode: 0, stdout: []byte(`{}`)}
	registry.SetCommandRunner(runner)

	// 4. Create Settings Manager and capture user hooks
	mgr, err := settings.NewClaudeSettingsManager(fs, "/home/testuser", "/project")
	if err != nil {
		t.Fatalf("NewClaudeSettingsManager() error: %v", err)
	}

	captured := mgr.CaptureAndReplaceHooks(proxyCmd, registry.RegisteredEvents())

	// 5. Verify user hooks were captured
	capturedPTU, ok := captured["PreToolUse"]
	if !ok {
		t.Fatal("expected captured PreToolUse hooks")
	}
	if len(capturedPTU) != 1 {
		t.Fatalf("expected 1 captured hook, got %d", len(capturedPTU))
	}
	if capturedPTU[0].Command != "/user/hooks/check-bash.sh" {
		t.Errorf("captured command = %q, want %q", capturedPTU[0].Command, "/user/hooks/check-bash.sh")
	}
	if capturedPTU[0].If != "Bash(rm *)" {
		t.Errorf("captured if = %q, want %q", capturedPTU[0].If, "Bash(rm *)")
	}

	// 6. Feed captured hooks to registry
	for eventName, ch := range captured {
		userHooks := make([]hooks.UserHook, len(ch))
		for i, h := range ch {
			userHooks[i] = hooks.UserHook{Command: h.Command, Matcher: h.Matcher, If: h.If}
		}
		registry.SetUserHooks(hooks.HookEvent(eventName), userHooks)
	}
	registry.SetProjectDir("/project")

	// 7. Dispatch a "rm" command — user hook should match (if: Bash(rm *))
	bridgeHandler := registry.BridgeHandler()
	resp := bridgeHandler(bridge.HookRequest{
		Type:  "hook",
		Event: "PreToolUse",
		Input: json.RawMessage(`{
			"hook_event_name": "PreToolUse",
			"tool_name": "Bash",
			"tool_input": {"command": "rm -rf /tmp/test"},
			"session_id": "test",
			"transcript_path": "/t",
			"cwd": "/project",
			"permission_mode": "default"
		}`),
	})
	if resp.ExitCode != 0 {
		t.Errorf("exit code = %d, want 0", resp.ExitCode)
	}
	if !beforeCalled {
		t.Error("before handler should be called")
	}
	if !runner.called {
		t.Error("user hook should be executed for rm command")
	}

	// 8. Dispatch a "npm test" command — user hook should NOT match (if: Bash(rm *))
	runner.called = false
	resp = bridgeHandler(bridge.HookRequest{
		Type:  "hook",
		Event: "PreToolUse",
		Input: json.RawMessage(`{
			"hook_event_name": "PreToolUse",
			"tool_name": "Bash",
			"tool_input": {"command": "npm test"},
			"session_id": "test",
			"transcript_path": "/t",
			"cwd": "/project",
			"permission_mode": "default"
		}`),
	})
	if resp.ExitCode != 0 {
		t.Errorf("exit code = %d, want 0", resp.ExitCode)
	}
	if runner.called {
		t.Error("user hook should NOT be executed for npm test (if: Bash(rm *) doesn't match)")
	}
}

// mockSettingsFSWithFiles provides files for settings discovery.
type mockSettingsFSWithFiles struct {
	files map[string][]byte
}

func (m mockSettingsFSWithFiles) ReadFile(path string) ([]byte, error) {
	data, ok := m.files[path]
	if !ok {
		return nil, os.ErrNotExist
	}
	return data, nil
}

func (m mockSettingsFSWithFiles) Stat(path string) (os.FileInfo, error) {
	_, ok := m.files[path]
	if !ok {
		return nil, os.ErrNotExist
	}
	return nil, nil
}
