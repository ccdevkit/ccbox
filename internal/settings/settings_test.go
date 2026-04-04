package settings

import (
	"os"
	"path/filepath"
	"reflect"
	"testing"
)

func TestSettingsStructHasYAMLTagsOnly(t *testing.T) {
	st := reflect.TypeOf(Settings{})
	for i := 0; i < st.NumField(); i++ {
		field := st.Field(i)
		if _, ok := field.Tag.Lookup("json"); ok {
			t.Errorf("field %s has json tag; only yaml tags are allowed", field.Name)
		}
		if _, ok := field.Tag.Lookup("yaml"); !ok {
			t.Errorf("field %s is missing yaml tag", field.Name)
		}
	}
}

func TestLoadDefaultsWhenNoFilesExist(t *testing.T) {
	// Use a temp dir with no settings files as cwd
	tmp := t.TempDir()
	origDir, err := os.Getwd()
	if err != nil {
		t.Fatal(err)
	}
	defer os.Chdir(origDir)

	if err := os.Chdir(tmp); err != nil {
		t.Fatal(err)
	}

	cfg, err := Load()
	if err != nil {
		t.Fatalf("Load() returned error: %v", err)
	}
	if cfg.ClaudePath != "" {
		t.Errorf("ClaudePath = %q, want empty string", cfg.ClaudePath)
	}
	if cfg.Verbose {
		t.Error("Verbose = true, want false")
	}
	if cfg.LogFile != "" {
		t.Errorf("LogFile = %q, want empty string", cfg.LogFile)
	}
	if len(cfg.Passthrough) != 0 {
		t.Errorf("Passthrough = %v, want empty", cfg.Passthrough)
	}
}

func TestLoadFromTempDir(t *testing.T) {
	tmp := t.TempDir()
	origDir, err := os.Getwd()
	if err != nil {
		t.Fatal(err)
	}
	defer os.Chdir(origDir)

	// Create .ccbox/settings.yaml in the temp dir
	settingsDir := filepath.Join(tmp, ".ccbox")
	if err := os.MkdirAll(settingsDir, 0o755); err != nil {
		t.Fatal(err)
	}
	content := []byte(`passthrough:
  - git
  - docker
claudePath: /usr/local/bin/claude
verbose: true
logFile: /tmp/ccbox.log
`)
	if err := os.WriteFile(filepath.Join(settingsDir, "settings.yaml"), content, 0o644); err != nil {
		t.Fatal(err)
	}

	if err := os.Chdir(tmp); err != nil {
		t.Fatal(err)
	}

	cfg, err := Load()
	if err != nil {
		t.Fatalf("Load() returned error: %v", err)
	}
	if cfg.ClaudePath != "/usr/local/bin/claude" {
		t.Errorf("ClaudePath = %q, want /usr/local/bin/claude", cfg.ClaudePath)
	}
	if !cfg.Verbose {
		t.Error("Verbose = false, want true")
	}
	if cfg.LogFile != "/tmp/ccbox.log" {
		t.Errorf("LogFile = %q, want /tmp/ccbox.log", cfg.LogFile)
	}
	expected := []string{"git", "docker"}
	if !reflect.DeepEqual(cfg.Passthrough, expected) {
		t.Errorf("Passthrough = %v, want %v", cfg.Passthrough, expected)
	}
}

func TestMergeWithCLI_FlagsOverrideProjectSettings(t *testing.T) {
	base := &Settings{
		Passthrough: []string{"git"},
		ClaudePath:  "/usr/local/bin/claude",
		Verbose:     false,
		LogFile:     "/tmp/base.log",
	}

	merged := MergeWithCLI(base, []string{"docker"}, "/opt/claude", true, "/tmp/cli.log")

	if merged.ClaudePath != "/opt/claude" {
		t.Errorf("ClaudePath = %q, want /opt/claude", merged.ClaudePath)
	}
	if !merged.Verbose {
		t.Error("Verbose = false, want true")
	}
	if merged.LogFile != "/tmp/cli.log" {
		t.Errorf("LogFile = %q, want /tmp/cli.log", merged.LogFile)
	}
}

func TestMergeWithCLI_PassthroughAppendsNotReplaces(t *testing.T) {
	base := &Settings{
		Passthrough: []string{"git", "docker"},
	}

	merged := MergeWithCLI(base, []string{"npm", "yarn"}, "", false, "")

	expected := []string{"git", "docker", "npm", "yarn"}
	if !reflect.DeepEqual(merged.Passthrough, expected) {
		t.Errorf("Passthrough = %v, want %v", merged.Passthrough, expected)
	}
}

func TestMergeWithCLI_ZeroValueCLIFlagsDoNotOverride(t *testing.T) {
	base := &Settings{
		ClaudePath: "/usr/local/bin/claude",
		Verbose:    true,
		LogFile:    "/tmp/base.log",
	}

	// Pass zero-value CLI flags — should not override base
	merged := MergeWithCLI(base, nil, "", false, "")

	if merged.ClaudePath != "/usr/local/bin/claude" {
		t.Errorf("ClaudePath = %q, want /usr/local/bin/claude", merged.ClaudePath)
	}
	if !merged.Verbose {
		t.Error("Verbose = false, want true (base should be preserved)")
	}
	if merged.LogFile != "/tmp/base.log" {
		t.Errorf("LogFile = %q, want /tmp/base.log", merged.LogFile)
	}
}

func TestMergeWithCLI_DoesNotMutateOriginal(t *testing.T) {
	base := &Settings{
		Passthrough: []string{"git"},
		ClaudePath:  "/original",
	}

	merged := MergeWithCLI(base, []string{"docker"}, "/new", true, "")

	// Original should be unchanged
	if base.ClaudePath != "/original" {
		t.Errorf("base.ClaudePath mutated to %q", base.ClaudePath)
	}
	if len(base.Passthrough) != 1 {
		t.Errorf("base.Passthrough mutated to %v", base.Passthrough)
	}

	// Merged should have the new values
	if merged.ClaudePath != "/new" {
		t.Errorf("merged.ClaudePath = %q, want /new", merged.ClaudePath)
	}
}

func TestLoadMalformedFileReturnsDefaults(t *testing.T) {
	tmp := t.TempDir()
	origDir, err := os.Getwd()
	if err != nil {
		t.Fatal(err)
	}
	defer os.Chdir(origDir)

	// Create .ccbox/settings.json with invalid JSON
	settingsDir := filepath.Join(tmp, ".ccbox")
	if err := os.MkdirAll(settingsDir, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(settingsDir, "settings.json"), []byte("{invalid json"), 0o644); err != nil {
		t.Fatal(err)
	}

	if err := os.Chdir(tmp); err != nil {
		t.Fatal(err)
	}

	cfg, err := Load()
	if err != nil {
		t.Fatalf("Load() should silently ignore malformed files, got error: %v", err)
	}
	if cfg.ClaudePath != "" {
		t.Errorf("ClaudePath = %q, want empty string", cfg.ClaudePath)
	}
	if cfg.Verbose {
		t.Error("Verbose = true, want false")
	}
}
