package settings

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"
)

type mockFS struct {
	files map[string][]byte
}

func (m mockFS) ReadFile(path string) ([]byte, error) {
	data, ok := m.files[path]
	if !ok {
		return nil, os.ErrNotExist
	}
	return data, nil
}

func (m mockFS) Stat(path string) (os.FileInfo, error) {
	_, ok := m.files[path]
	if !ok {
		return nil, os.ErrNotExist
	}
	return nil, nil
}

func mustJSON(t *testing.T, v interface{}) []byte {
	t.Helper()
	data, err := json.Marshal(v)
	if err != nil {
		t.Fatal(err)
	}
	return data
}

func TestNewClaudeSettingsManager_AllFourFiles(t *testing.T) {
	home := "/fakehome"
	project := "/fakeproject"

	fs := mockFS{files: map[string][]byte{
		filepath.Join(home, ".claude", "settings.json"):          mustJSON(t, map[string]interface{}{"a": "user", "b": "user"}),
		filepath.Join(home, ".claude", "settings.local.json"):    mustJSON(t, map[string]interface{}{"b": "user-local", "c": "user-local"}),
		filepath.Join(project, ".claude", "settings.json"):       mustJSON(t, map[string]interface{}{"c": "project", "d": "project"}),
		filepath.Join(project, ".claude", "settings.local.json"): mustJSON(t, map[string]interface{}{"d": "project-local", "e": "project-local"}),
	}}

	mgr, err := NewClaudeSettingsManager(fs, home, project)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	merged := mgr.Merged()
	// a: only in user settings
	assertValue(t, merged, "a", "user")
	// b: user overridden by user-local
	assertValue(t, merged, "b", "user-local")
	// c: user-local overridden by project
	assertValue(t, merged, "c", "project")
	// d: project overridden by project-local
	assertValue(t, merged, "d", "project-local")
	// e: only in project-local
	assertValue(t, merged, "e", "project-local")
}

func TestNewClaudeSettingsManager_OnlyUserLevel(t *testing.T) {
	home := "/fakehome"
	project := "/fakeproject"

	fs := mockFS{files: map[string][]byte{
		filepath.Join(home, ".claude", "settings.json"): mustJSON(t, map[string]interface{}{"key": "value"}),
	}}

	mgr, err := NewClaudeSettingsManager(fs, home, project)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	merged := mgr.Merged()
	assertValue(t, merged, "key", "value")
}

func TestNewClaudeSettingsManager_NoFiles(t *testing.T) {
	fs := mockFS{files: map[string][]byte{}}

	mgr, err := NewClaudeSettingsManager(fs, "/fakehome", "/fakeproject")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	merged := mgr.Merged()
	if len(merged) != 0 {
		t.Fatalf("expected empty merged settings, got %v", merged)
	}
}

func TestNewClaudeSettingsManager_Precedence(t *testing.T) {
	home := "/fakehome"
	project := "/fakeproject"

	// All four files set the same key — project-local should win
	fs := mockFS{files: map[string][]byte{
		filepath.Join(home, ".claude", "settings.json"):          mustJSON(t, map[string]interface{}{"key": "1-user"}),
		filepath.Join(home, ".claude", "settings.local.json"):    mustJSON(t, map[string]interface{}{"key": "2-user-local"}),
		filepath.Join(project, ".claude", "settings.json"):       mustJSON(t, map[string]interface{}{"key": "3-project"}),
		filepath.Join(project, ".claude", "settings.local.json"): mustJSON(t, map[string]interface{}{"key": "4-project-local"}),
	}}

	mgr, err := NewClaudeSettingsManager(fs, home, project)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	assertValue(t, mgr.Merged(), "key", "4-project-local")
}

func TestNewClaudeSettingsManager_ProjectOverridesUserForSameKey(t *testing.T) {
	home := "/fakehome"
	project := "/fakeproject"

	fs := mockFS{files: map[string][]byte{
		filepath.Join(home, ".claude", "settings.json"): mustJSON(t, map[string]interface{}{
			"allowedTools": []string{"Read"},
			"userOnly":     "present",
		}),
		filepath.Join(project, ".claude", "settings.json"): mustJSON(t, map[string]interface{}{
			"allowedTools": []string{"Write"},
			"projectOnly":  "present",
		}),
	}}

	mgr, err := NewClaudeSettingsManager(fs, home, project)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	merged := mgr.Merged()

	// allowedTools: project-level should override user-level
	tools, ok := merged["allowedTools"]
	if !ok {
		t.Fatal("allowedTools not found in merged settings")
	}
	arr, ok := tools.([]interface{})
	if !ok {
		t.Fatalf("allowedTools: expected []interface{}, got %T", tools)
	}
	if len(arr) != 1 || arr[0] != "Write" {
		t.Errorf("allowedTools: expected [Write], got %v", arr)
	}

	// Keys unique to each file should both be present
	assertValue(t, merged, "userOnly", "present")
	assertValue(t, merged, "projectOnly", "present")
}

func TestNewClaudeSettingsManager_MalformedJSONSkipped(t *testing.T) {
	home := "/fakehome"
	project := "/fakeproject"

	fs := mockFS{files: map[string][]byte{
		filepath.Join(home, ".claude", "settings.json"):       []byte(`{"validKey": "from-user"}`),
		filepath.Join(home, ".claude", "settings.local.json"): []byte(`{invalid json!!!}`),
		filepath.Join(project, ".claude", "settings.json"):    []byte(`{"projectKey": "from-project"}`),
	}}

	mgr, err := NewClaudeSettingsManager(fs, home, project)
	if err != nil {
		t.Fatalf("expected no error for malformed JSON, got: %v", err)
	}

	merged := mgr.Merged()
	assertValue(t, merged, "validKey", "from-user")
	assertValue(t, merged, "projectKey", "from-project")

	// Malformed file should not contribute any keys
	if len(merged) != 2 {
		t.Errorf("expected exactly 2 keys in merged settings, got %d: %v", len(merged), merged)
	}
}

func TestSet_TopLevelKey(t *testing.T) {
	fs := mockFS{files: map[string][]byte{}}
	mgr, err := NewClaudeSettingsManager(fs, "/fakehome", "/fakeproject")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	mgr.Set("myKey", "myValue")

	merged := mgr.Merged()
	assertValue(t, merged, "myKey", "myValue")
}

func TestSet_OverridesExisting(t *testing.T) {
	fs := mockFS{files: map[string][]byte{
		"/fakehome/.claude/settings.json": mustJSON(t, map[string]interface{}{"key": "original"}),
	}}
	mgr, err := NewClaudeSettingsManager(fs, "/fakehome", "/fakeproject")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	mgr.Set("key", "overridden")

	assertValue(t, mgr.Merged(), "key", "overridden")
}

func TestSetDeep_CreatesNestedStructure(t *testing.T) {
	fs := mockFS{files: map[string][]byte{}}
	mgr, err := NewClaudeSettingsManager(fs, "/fakehome", "/fakeproject")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	mgr.SetDeep("a.b.c", "deep-value")

	merged := mgr.Merged()
	a, ok := merged["a"].(map[string]interface{})
	if !ok {
		t.Fatalf("expected 'a' to be map, got %T", merged["a"])
	}
	b, ok := a["b"].(map[string]interface{})
	if !ok {
		t.Fatalf("expected 'a.b' to be map, got %T", a["b"])
	}
	c, ok := b["c"].(string)
	if !ok {
		t.Fatalf("expected 'a.b.c' to be string, got %T", b["c"])
	}
	if c != "deep-value" {
		t.Errorf("a.b.c = %q, want %q", c, "deep-value")
	}
}

func TestSetDeep_SingleSegment(t *testing.T) {
	fs := mockFS{files: map[string][]byte{}}
	mgr, err := NewClaudeSettingsManager(fs, "/fakehome", "/fakeproject")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	mgr.SetDeep("topLevel", "value")

	assertValue(t, mgr.Merged(), "topLevel", "value")
}

func TestSetDeep_PreservesExistingNestedKeys(t *testing.T) {
	fs := mockFS{files: map[string][]byte{
		"/fakehome/.claude/settings.json": mustJSON(t, map[string]interface{}{
			"hooks": map[string]interface{}{
				"PreToolUse": "existing",
			},
		}),
	}}
	mgr, err := NewClaudeSettingsManager(fs, "/fakehome", "/fakeproject")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	mgr.SetDeep("hooks.PostToolUse", "new")

	merged := mgr.Merged()
	hooks, ok := merged["hooks"].(map[string]interface{})
	if !ok {
		t.Fatalf("expected 'hooks' to be map, got %T", merged["hooks"])
	}
	if hooks["PreToolUse"] != "existing" {
		t.Errorf("PreToolUse = %v, want %q", hooks["PreToolUse"], "existing")
	}
	if hooks["PostToolUse"] != "new" {
		t.Errorf("PostToolUse = %v, want %q", hooks["PostToolUse"], "new")
	}
}

func TestCaptureAndReplaceHooks_CapturesUserHooks(t *testing.T) {
	fs := mockFS{files: map[string][]byte{
		"/fakehome/.claude/settings.json": mustJSON(t, map[string]interface{}{
			"hooks": map[string]interface{}{
				"PreToolUse": []interface{}{
					map[string]interface{}{
						"matcher": "Bash",
						"hooks": []interface{}{
							map[string]interface{}{"type": "command", "command": "/user/bash-hook"},
						},
					},
				},
			},
		}),
	}}

	mgr, err := NewClaudeSettingsManager(fs, "/fakehome", "/fakeproject")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	captured := mgr.CaptureAndReplaceHooks("/ccbox/proxy", map[string]bool{})

	hooks, ok := captured["PreToolUse"]
	if !ok {
		t.Fatal("expected captured hooks for PreToolUse")
	}
	if len(hooks) != 1 {
		t.Fatalf("expected 1 captured hook, got %d", len(hooks))
	}
	if hooks[0].Command != "/user/bash-hook" {
		t.Errorf("command = %q, want %q", hooks[0].Command, "/user/bash-hook")
	}
	if hooks[0].Matcher != "Bash" {
		t.Errorf("matcher = %q, want %q", hooks[0].Matcher, "Bash")
	}
}

func TestCaptureAndReplaceHooks_ReplacesWithCatchAll(t *testing.T) {
	fs := mockFS{files: map[string][]byte{
		"/fakehome/.claude/settings.json": mustJSON(t, map[string]interface{}{
			"hooks": map[string]interface{}{
				"PreToolUse": []interface{}{
					map[string]interface{}{
						"matcher": "Bash",
						"hooks": []interface{}{
							map[string]interface{}{"type": "command", "command": "/user/bash-hook"},
							map[string]interface{}{"type": "command", "command": "/user/bash-hook-2"},
						},
					},
				},
			},
		}),
	}}

	mgr, err := NewClaudeSettingsManager(fs, "/fakehome", "/fakeproject")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	mgr.CaptureAndReplaceHooks("/ccbox/proxy", map[string]bool{})

	merged := mgr.Merged()
	hooksMap, ok := merged["hooks"].(map[string]interface{})
	if !ok {
		t.Fatalf("expected hooks to be map, got %T", merged["hooks"])
	}
	preToolUse, ok := hooksMap["PreToolUse"].([]interface{})
	if !ok {
		t.Fatalf("expected PreToolUse to be []interface{}, got %T", hooksMap["PreToolUse"])
	}

	// Should have exactly one catch-all matcher group
	if len(preToolUse) != 1 {
		t.Fatalf("expected 1 catch-all matcher group, got %d", len(preToolUse))
	}

	group, ok := preToolUse[0].(map[string]interface{})
	if !ok {
		t.Fatalf("expected matcher group to be map, got %T", preToolUse[0])
	}

	// Matcher should be "*" (catch-all)
	if matcher, _ := group["matcher"].(string); matcher != "*" {
		t.Errorf("matcher = %q, want %q (catch-all)", matcher, "*")
	}

	// Should have exactly one hook entry pointing to proxy
	hookEntries, ok := group["hooks"].([]interface{})
	if !ok {
		t.Fatalf("expected hooks to be []interface{}, got %T", group["hooks"])
	}
	if len(hookEntries) != 1 {
		t.Fatalf("expected 1 hook entry, got %d", len(hookEntries))
	}
	entry := hookEntries[0].(map[string]interface{})
	if entry["command"] != "/ccbox/proxy" {
		t.Errorf("command = %v, want /ccbox/proxy", entry["command"])
	}
}

func TestCaptureAndReplaceHooks_MergesRegisteredAndUserEvents(t *testing.T) {
	// User has PreToolUse hooks; ccbox registers PostToolUse handler
	fs := mockFS{files: map[string][]byte{
		"/fakehome/.claude/settings.json": mustJSON(t, map[string]interface{}{
			"hooks": map[string]interface{}{
				"PreToolUse": []interface{}{
					map[string]interface{}{
						"matcher": "",
						"hooks": []interface{}{
							map[string]interface{}{"type": "command", "command": "/user/hook"},
						},
					},
				},
			},
		}),
	}}

	mgr, err := NewClaudeSettingsManager(fs, "/fakehome", "/fakeproject")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	registeredEvents := map[string]bool{
		"PostToolUse": true,
	}

	mgr.CaptureAndReplaceHooks("/ccbox/proxy", registeredEvents)

	merged := mgr.Merged()
	hooksMap := merged["hooks"].(map[string]interface{})

	// Both events should have proxy entries
	if _, ok := hooksMap["PreToolUse"]; !ok {
		t.Error("expected PreToolUse in hooks (from user hooks)")
	}
	if _, ok := hooksMap["PostToolUse"]; !ok {
		t.Error("expected PostToolUse in hooks (from registered events)")
	}
}

func TestCaptureAndReplaceHooks_CapturesIfField(t *testing.T) {
	fs := mockFS{files: map[string][]byte{
		"/fakehome/.claude/settings.json": mustJSON(t, map[string]interface{}{
			"hooks": map[string]interface{}{
				"PreToolUse": []interface{}{
					map[string]interface{}{
						"matcher": "Bash",
						"hooks": []interface{}{
							map[string]interface{}{
								"type":    "command",
								"command": "/user/rm-hook",
								"if":      "Bash(rm *)",
							},
						},
					},
				},
			},
		}),
	}}

	mgr, err := NewClaudeSettingsManager(fs, "/fakehome", "/fakeproject")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	captured := mgr.CaptureAndReplaceHooks("/ccbox/proxy", map[string]bool{})

	hooks := captured["PreToolUse"]
	if len(hooks) != 1 {
		t.Fatalf("expected 1 captured hook, got %d", len(hooks))
	}
	if hooks[0].If != "Bash(rm *)" {
		t.Errorf("if = %q, want %q", hooks[0].If, "Bash(rm *)")
	}
}

func TestCaptureAndReplaceHooks_CapturesMatcherFromGroup(t *testing.T) {
	fs := mockFS{files: map[string][]byte{
		"/fakehome/.claude/settings.json": mustJSON(t, map[string]interface{}{
			"hooks": map[string]interface{}{
				"PreToolUse": []interface{}{
					map[string]interface{}{
						"matcher": "Edit|Write",
						"hooks": []interface{}{
							map[string]interface{}{"type": "command", "command": "/user/edit-hook"},
						},
					},
				},
			},
		}),
	}}

	mgr, err := NewClaudeSettingsManager(fs, "/fakehome", "/fakeproject")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	captured := mgr.CaptureAndReplaceHooks("/ccbox/proxy", map[string]bool{})

	hooks := captured["PreToolUse"]
	if len(hooks) != 1 {
		t.Fatalf("expected 1 captured hook, got %d", len(hooks))
	}
	if hooks[0].Matcher != "Edit|Write" {
		t.Errorf("matcher = %q, want %q", hooks[0].Matcher, "Edit|Write")
	}
}

func TestCaptureAndReplaceHooks_NoUserHooks(t *testing.T) {
	fs := mockFS{files: map[string][]byte{}}
	mgr, err := NewClaudeSettingsManager(fs, "/fakehome", "/fakeproject")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	registeredEvents := map[string]bool{
		"PreToolUse": true,
	}

	captured := mgr.CaptureAndReplaceHooks("/ccbox/proxy", registeredEvents)

	if len(captured) != 0 {
		t.Errorf("expected no captured hooks, got %d events", len(captured))
	}

	// But hooks config should still have the registered event
	merged := mgr.Merged()
	hooksMap := merged["hooks"].(map[string]interface{})
	if _, ok := hooksMap["PreToolUse"]; !ok {
		t.Error("expected PreToolUse in hooks from registered events")
	}
}

func TestCaptureAndReplaceHooks_MultipleMatcherGroups(t *testing.T) {
	fs := mockFS{files: map[string][]byte{
		"/fakehome/.claude/settings.json": mustJSON(t, map[string]interface{}{
			"hooks": map[string]interface{}{
				"PreToolUse": []interface{}{
					map[string]interface{}{
						"matcher": "",
						"hooks": []interface{}{
							map[string]interface{}{"type": "command", "command": "/user/global-hook"},
						},
					},
					map[string]interface{}{
						"matcher": "Bash",
						"hooks": []interface{}{
							map[string]interface{}{"type": "command", "command": "/user/bash-hook"},
						},
					},
				},
			},
		}),
	}}

	mgr, err := NewClaudeSettingsManager(fs, "/fakehome", "/fakeproject")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	captured := mgr.CaptureAndReplaceHooks("/ccbox/proxy", map[string]bool{})

	hooks := captured["PreToolUse"]
	if len(hooks) != 2 {
		t.Fatalf("expected 2 captured hooks (from 2 groups), got %d", len(hooks))
	}

	// Verify matchers are correct
	matchers := map[string]bool{}
	for _, h := range hooks {
		matchers[h.Matcher] = true
	}
	if !matchers[""] {
		t.Error("expected captured hook with empty matcher")
	}
	if !matchers["Bash"] {
		t.Error("expected captured hook with Bash matcher")
	}
}

func TestCaptureAndReplaceHooks_OnlyCommandHooksCaptured(t *testing.T) {
	// HTTP and prompt hooks can't be proxied as subprocesses — skip them
	fs := mockFS{files: map[string][]byte{
		"/fakehome/.claude/settings.json": mustJSON(t, map[string]interface{}{
			"hooks": map[string]interface{}{
				"PreToolUse": []interface{}{
					map[string]interface{}{
						"matcher": "",
						"hooks": []interface{}{
							map[string]interface{}{"type": "command", "command": "/user/cmd-hook"},
							map[string]interface{}{"type": "http", "url": "http://localhost:8080/hook"},
							map[string]interface{}{"type": "prompt", "prompt": "check this"},
						},
					},
				},
			},
		}),
	}}

	mgr, err := NewClaudeSettingsManager(fs, "/fakehome", "/fakeproject")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	captured := mgr.CaptureAndReplaceHooks("/ccbox/proxy", map[string]bool{})

	hooks := captured["PreToolUse"]
	if len(hooks) != 1 {
		t.Fatalf("expected 1 captured command hook (http/prompt skipped), got %d", len(hooks))
	}
	if hooks[0].Command != "/user/cmd-hook" {
		t.Errorf("command = %q, want %q", hooks[0].Command, "/user/cmd-hook")
	}
}

// mockSessionFileWriter records calls to WriteFile for test verification.
type mockSessionFileWriter struct {
	writtenPath     string
	writtenData     []byte
	writtenReadOnly bool
	err             error
}

func (m *mockSessionFileWriter) WriteFile(containerPath string, data []byte, readOnly bool) error {
	if m.err != nil {
		return m.err
	}
	m.writtenPath = containerPath
	m.writtenData = data
	m.writtenReadOnly = readOnly
	return nil
}

func TestFinalize_WritesSettingsAndReturnsArgs(t *testing.T) {
	fs := mockFS{files: map[string][]byte{}}
	mgr, err := NewClaudeSettingsManager(fs, "/fakehome", "/fakeproject")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	mgr.Set("allowedTools", []string{"Read", "Write"})

	fw := &mockSessionFileWriter{}
	args, err := mgr.Finalize(fw)
	if err != nil {
		t.Fatalf("Finalize() error: %v", err)
	}

	// Verify written path
	if fw.writtenPath != "/opt/ccbox/settings.json" {
		t.Errorf("written path = %q, want %q", fw.writtenPath, "/opt/ccbox/settings.json")
	}

	// Verify read-only
	if !fw.writtenReadOnly {
		t.Error("expected readOnly=true")
	}

	// Verify written data is valid JSON containing our key
	var parsed map[string]interface{}
	if err := json.Unmarshal(fw.writtenData, &parsed); err != nil {
		t.Fatalf("written data is not valid JSON: %v", err)
	}
	if _, ok := parsed["allowedTools"]; !ok {
		t.Error("written JSON missing 'allowedTools' key")
	}

	// Verify returned args
	expectedArgs := []string{"--settings", "/opt/ccbox/settings.json", "--setting-sources", ""}
	if len(args) != len(expectedArgs) {
		t.Fatalf("args length = %d, want %d", len(args), len(expectedArgs))
	}
	for i, a := range args {
		if a != expectedArgs[i] {
			t.Errorf("args[%d] = %q, want %q", i, a, expectedArgs[i])
		}
	}
}

func TestFinalize_DoubleCallFails(t *testing.T) {
	fs := mockFS{files: map[string][]byte{}}
	mgr, err := NewClaudeSettingsManager(fs, "/fakehome", "/fakeproject")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	fw := &mockSessionFileWriter{}
	_, err = mgr.Finalize(fw)
	if err != nil {
		t.Fatalf("first Finalize() error: %v", err)
	}

	_, err = mgr.Finalize(fw)
	if err == nil {
		t.Fatal("expected error on second Finalize() call")
	}
}

func TestFinalize_WriteError(t *testing.T) {
	fs := mockFS{files: map[string][]byte{}}
	mgr, err := NewClaudeSettingsManager(fs, "/fakehome", "/fakeproject")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	fw := &mockSessionFileWriter{err: os.ErrPermission}
	_, err = mgr.Finalize(fw)
	if err == nil {
		t.Fatal("expected error when writer fails")
	}
}

func TestSet_PanicsAfterFinalize(t *testing.T) {
	fs := mockFS{files: map[string][]byte{}}
	mgr, err := NewClaudeSettingsManager(fs, "/fakehome", "/fakeproject")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	fw := &mockSessionFileWriter{}
	if _, err := mgr.Finalize(fw); err != nil {
		t.Fatalf("Finalize() error: %v", err)
	}

	defer func() {
		if r := recover(); r == nil {
			t.Fatal("expected Set() to panic after Finalize")
		}
	}()

	mgr.Set("key", "value")
}

func TestSetDeep_PanicsAfterFinalize(t *testing.T) {
	fs := mockFS{files: map[string][]byte{}}
	mgr, err := NewClaudeSettingsManager(fs, "/fakehome", "/fakeproject")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	fw := &mockSessionFileWriter{}
	if _, err := mgr.Finalize(fw); err != nil {
		t.Fatalf("Finalize() error: %v", err)
	}

	defer func() {
		if r := recover(); r == nil {
			t.Fatal("expected SetDeep() to panic after Finalize")
		}
	}()

	mgr.SetDeep("a.b", "value")
}

func TestCaptureAndReplaceHooks_PanicsAfterFinalize(t *testing.T) {
	fs := mockFS{files: map[string][]byte{}}
	mgr, err := NewClaudeSettingsManager(fs, "/fakehome", "/fakeproject")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	fw := &mockSessionFileWriter{}
	if _, err := mgr.Finalize(fw); err != nil {
		t.Fatalf("Finalize() error: %v", err)
	}

	defer func() {
		if r := recover(); r == nil {
			t.Fatal("expected CaptureAndReplaceHooks() to panic after Finalize")
		}
	}()

	mgr.CaptureAndReplaceHooks("/ccbox/proxy", map[string]bool{})
}

// FR-011: Set() on a user-defined key overwrites it (explicit targeting by design).
func TestSet_OverwritesUserDefinedKey_FR011(t *testing.T) {
	home := "/fakehome"
	project := "/fakeproject"

	fs := mockFS{files: map[string][]byte{
		filepath.Join(home, ".claude", "settings.json"): mustJSON(t, map[string]interface{}{
			"allowedTools": []string{"Read", "Write", "Bash"},
		}),
	}}

	mgr, err := NewClaudeSettingsManager(fs, home, project)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// Verify user value is loaded
	tools := mgr.Merged()["allowedTools"]
	arr, ok := tools.([]interface{})
	if !ok {
		t.Fatalf("expected []interface{}, got %T", tools)
	}
	if len(arr) != 3 {
		t.Fatalf("expected 3 user tools, got %d", len(arr))
	}

	// Module explicitly targets the same key — overwrites by design
	mgr.Set("allowedTools", []string{"WebFetch"})

	updated := mgr.Merged()["allowedTools"]
	updatedArr, ok := updated.([]string)
	if !ok {
		t.Fatalf("expected []string after Set(), got %T", updated)
	}
	if len(updatedArr) != 1 || updatedArr[0] != "WebFetch" {
		t.Errorf("Set() should overwrite user value; got %v, want [WebFetch]", updatedArr)
	}
}

// FR-011: CaptureAndReplaceHooks captures ALL user hooks — none are lost.
func TestCaptureAndReplaceHooks_PreservesAllUserHooks_FR011(t *testing.T) {
	home := "/fakehome"
	project := "/fakeproject"

	// User has multiple hooks across two matcher groups
	fs := mockFS{files: map[string][]byte{
		filepath.Join(home, ".claude", "settings.json"): mustJSON(t, map[string]interface{}{
			"hooks": map[string]interface{}{
				"PreToolUse": []interface{}{
					map[string]interface{}{
						"matcher": "",
						"hooks": []interface{}{
							map[string]interface{}{"type": "command", "command": "/user/global-hook-1"},
							map[string]interface{}{"type": "command", "command": "/user/global-hook-2"},
						},
					},
					map[string]interface{}{
						"matcher": "Bash",
						"hooks": []interface{}{
							map[string]interface{}{"type": "command", "command": "/user/bash-hook"},
						},
					},
				},
			},
		}),
	}}

	mgr, err := NewClaudeSettingsManager(fs, home, project)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	captured := mgr.CaptureAndReplaceHooks("/ccbox/proxy", map[string]bool{})

	// All 3 user hooks should be captured
	hooks := captured["PreToolUse"]
	if len(hooks) != 3 {
		t.Fatalf("expected 3 captured hooks (2 global + 1 bash), got %d", len(hooks))
	}

	// Verify all commands present
	commands := map[string]bool{}
	for _, h := range hooks {
		commands[h.Command] = true
	}
	for _, expected := range []string{"/user/global-hook-1", "/user/global-hook-2", "/user/bash-hook"} {
		if !commands[expected] {
			t.Errorf("missing captured hook: %q", expected)
		}
	}
}

// FR-011: User settings keys NOT targeted by Set() remain unchanged after Finalize.
func TestFinalize_PreservesUntargetedUserKeys_FR011(t *testing.T) {
	home := "/fakehome"
	project := "/fakeproject"

	fs := mockFS{files: map[string][]byte{
		filepath.Join(home, ".claude", "settings.json"): mustJSON(t, map[string]interface{}{
			"allowedTools":    []string{"Read", "Write"},
			"userCustomKey":   "should-survive",
			"anotherUserPref": 42.0,
		}),
	}}

	mgr, err := NewClaudeSettingsManager(fs, home, project)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// Module only targets "allowedTools" — other keys should be untouched
	mgr.Set("allowedTools", []string{"Bash"})
	mgr.Set("ccboxInjected", "new-value")

	fw := &mockSessionFileWriter{}
	_, err = mgr.Finalize(fw)
	if err != nil {
		t.Fatalf("Finalize() error: %v", err)
	}

	// Parse the written JSON to verify user keys survived
	var written map[string]interface{}
	if err := json.Unmarshal(fw.writtenData, &written); err != nil {
		t.Fatalf("written data is not valid JSON: %v", err)
	}

	// Untargeted user keys must be preserved
	if v, ok := written["userCustomKey"]; !ok {
		t.Error("userCustomKey missing from finalized settings — user key was dropped")
	} else if v != "should-survive" {
		t.Errorf("userCustomKey = %v, want \"should-survive\"", v)
	}

	if v, ok := written["anotherUserPref"]; !ok {
		t.Error("anotherUserPref missing from finalized settings — user key was dropped")
	} else if v != 42.0 {
		t.Errorf("anotherUserPref = %v, want 42", v)
	}

	// Targeted key should reflect the module's value
	tools, ok := written["allowedTools"]
	if !ok {
		t.Fatal("allowedTools missing from finalized settings")
	}
	toolsArr, ok := tools.([]interface{})
	if !ok {
		t.Fatalf("allowedTools: expected []interface{}, got %T", tools)
	}
	if len(toolsArr) != 1 || toolsArr[0] != "Bash" {
		t.Errorf("allowedTools = %v, want [Bash]", toolsArr)
	}

	// Injected key should also be present
	if v, ok := written["ccboxInjected"]; !ok {
		t.Error("ccboxInjected missing from finalized settings")
	} else if v != "new-value" {
		t.Errorf("ccboxInjected = %v, want \"new-value\"", v)
	}
}

func assertValue(t *testing.T, m map[string]interface{}, key string, expected string) {
	t.Helper()
	val, ok := m[key]
	if !ok {
		t.Errorf("key %q not found in merged settings", key)
		return
	}
	str, ok := val.(string)
	if !ok {
		t.Errorf("key %q: expected string, got %T", key, val)
		return
	}
	if str != expected {
		t.Errorf("key %q: expected %q, got %q", key, expected, str)
	}
}
