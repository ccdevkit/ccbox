package settings

import (
	"os"
	"path/filepath"
	"testing"
)

func TestMergeClaudeSettings_LocalOverridesGlobal(t *testing.T) {
	host := map[string]interface{}{
		"bypassPermissions": false,
		"theme":             "dark",
	}
	overrides := map[string]interface{}{
		"bypassPermissions": true,
	}

	merged := MergeClaudeSettings(host, overrides)

	if merged["bypassPermissions"] != true {
		t.Errorf("bypassPermissions = %v, want true", merged["bypassPermissions"])
	}
	if merged["theme"] != "dark" {
		t.Errorf("theme = %v, want dark", merged["theme"])
	}
}

func TestMergeClaudeSettings_PreservesUserCustomSettings(t *testing.T) {
	host := map[string]interface{}{
		"customSetting":     "userValue",
		"allowedTools":      []interface{}{"tool1", "tool2"},
		"bypassPermissions": false,
	}
	overrides := map[string]interface{}{
		"bypassPermissions": true,
		"newSetting":        "override",
	}

	merged := MergeClaudeSettings(host, overrides)

	if merged["customSetting"] != "userValue" {
		t.Errorf("customSetting = %v, want userValue", merged["customSetting"])
	}
	if merged["bypassPermissions"] != true {
		t.Errorf("bypassPermissions = %v, want true", merged["bypassPermissions"])
	}
	if merged["newSetting"] != "override" {
		t.Errorf("newSetting = %v, want override", merged["newSetting"])
	}
}

func TestMergeClaudeSettings_NilInputs(t *testing.T) {
	merged := MergeClaudeSettings(nil, map[string]interface{}{"key": "val"})
	if merged["key"] != "val" {
		t.Errorf("key = %v, want val", merged["key"])
	}

	merged = MergeClaudeSettings(map[string]interface{}{"key": "val"}, nil)
	if merged["key"] != "val" {
		t.Errorf("key = %v, want val", merged["key"])
	}
}

func TestReadClaudeSettings_MissingFileReturnsEmptyMap(t *testing.T) {
	result, err := ReadClaudeSettings("/nonexistent/path/settings.json")
	if err != nil {
		t.Fatalf("ReadClaudeSettings() returned error: %v", err)
	}
	if len(result) != 0 {
		t.Errorf("expected empty map, got %v", result)
	}
}

func TestReadClaudeSettings_ValidFile(t *testing.T) {
	tmp := t.TempDir()
	path := filepath.Join(tmp, "settings.json")
	content := []byte(`{"bypassPermissions":false,"theme":"dark"}`)
	if err := os.WriteFile(path, content, 0o644); err != nil {
		t.Fatal(err)
	}

	result, err := ReadClaudeSettings(path)
	if err != nil {
		t.Fatalf("ReadClaudeSettings() returned error: %v", err)
	}
	if result["bypassPermissions"] != false {
		t.Errorf("bypassPermissions = %v, want false", result["bypassPermissions"])
	}
	if result["theme"] != "dark" {
		t.Errorf("theme = %v, want dark", result["theme"])
	}
}

func TestReadClaudeSettings_MalformedJSON(t *testing.T) {
	tmp := t.TempDir()
	path := filepath.Join(tmp, "settings.json")
	if err := os.WriteFile(path, []byte("{bad json"), 0o644); err != nil {
		t.Fatal(err)
	}

	_, err := ReadClaudeSettings(path)
	if err == nil {
		t.Fatal("expected error for malformed JSON, got nil")
	}
}
