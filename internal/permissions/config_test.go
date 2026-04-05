package permissions

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestLoad_ValidYAML(t *testing.T) {
	tmp := t.TempDir()
	dir := filepath.Join(tmp, ".ccbox")
	if err := os.MkdirAll(dir, 0o755); err != nil {
		t.Fatal(err)
	}

	yamlContent := `passthrough:
  git:
    rules:
      - pattern: "**"
        effect: allow
`
	if err := os.WriteFile(filepath.Join(dir, "permissions.yaml"), []byte(yamlContent), 0o644); err != nil {
		t.Fatal(err)
	}

	origDir, err := os.Getwd()
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { os.Chdir(origDir) })
	if err := os.Chdir(tmp); err != nil {
		t.Fatal(err)
	}

	cfg, err := Load()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if cfg == nil {
		t.Fatal("expected non-nil config")
	}
	git, ok := cfg.Passthrough["git"]
	if !ok {
		t.Fatal("expected 'git' command in passthrough")
	}
	if len(git.Rules) != 1 {
		t.Fatalf("expected 1 rule, got %d", len(git.Rules))
	}
	if git.Rules[0].Effect != "allow" {
		t.Errorf("expected effect 'allow', got %q", git.Rules[0].Effect)
	}
	if len(git.Rules[0].Pattern.Values) != 1 || git.Rules[0].Pattern.Values[0] != "**" {
		t.Errorf("expected pattern [**], got %v", git.Rules[0].Pattern.Values)
	}
}

func TestLoad_ValidJSON(t *testing.T) {
	tmp := t.TempDir()
	dir := filepath.Join(tmp, ".ccbox")
	if err := os.MkdirAll(dir, 0o755); err != nil {
		t.Fatal(err)
	}

	jsonContent := `{
  "passthrough": {
    "git": {
      "rules": [
        {"pattern": "**", "effect": "allow"}
      ]
    }
  }
}`
	if err := os.WriteFile(filepath.Join(dir, "permissions.json"), []byte(jsonContent), 0o644); err != nil {
		t.Fatal(err)
	}

	origDir, err := os.Getwd()
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { os.Chdir(origDir) })
	if err := os.Chdir(tmp); err != nil {
		t.Fatal(err)
	}

	cfg, err := Load()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if cfg == nil {
		t.Fatal("expected non-nil config")
	}
	git, ok := cfg.Passthrough["git"]
	if !ok {
		t.Fatal("expected 'git' command in passthrough")
	}
	if len(git.Rules) != 1 {
		t.Fatalf("expected 1 rule, got %d", len(git.Rules))
	}
	if git.Rules[0].Effect != "allow" {
		t.Errorf("expected effect 'allow', got %q", git.Rules[0].Effect)
	}
}

func TestLoad_MalformedYAML(t *testing.T) {
	tmp := t.TempDir()
	dir := filepath.Join(tmp, ".ccbox")
	if err := os.MkdirAll(dir, 0o755); err != nil {
		t.Fatal(err)
	}

	if err := os.WriteFile(filepath.Join(dir, "permissions.yaml"), []byte(`{{{`), 0o644); err != nil {
		t.Fatal(err)
	}

	origDir, err := os.Getwd()
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { os.Chdir(origDir) })
	if err := os.Chdir(tmp); err != nil {
		t.Fatal(err)
	}

	_, err = Load()
	if err == nil {
		t.Fatal("expected error for malformed YAML, got nil")
	}
}

func TestLoad_InvalidEffect(t *testing.T) {
	tmp := t.TempDir()
	dir := filepath.Join(tmp, ".ccbox")
	if err := os.MkdirAll(dir, 0o755); err != nil {
		t.Fatal(err)
	}

	yamlContent := `passthrough:
  git:
    rules:
      - pattern: "**"
        effect: block
`
	if err := os.WriteFile(filepath.Join(dir, "permissions.yaml"), []byte(yamlContent), 0o644); err != nil {
		t.Fatal(err)
	}

	origDir, err := os.Getwd()
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { os.Chdir(origDir) })
	if err := os.Chdir(tmp); err != nil {
		t.Fatal(err)
	}

	_, err = Load()
	if err == nil {
		t.Fatal("expected error for invalid effect, got nil")
	}
	if !strings.Contains(err.Error(), "block") {
		t.Errorf("expected error to mention 'block', got: %v", err)
	}
}

func TestLoad_EmptyCommandName(t *testing.T) {
	tmp := t.TempDir()
	dir := filepath.Join(tmp, ".ccbox")
	if err := os.MkdirAll(dir, 0o755); err != nil {
		t.Fatal(err)
	}

	jsonContent := `{
  "passthrough": {
    "": {
      "rules": [
        {"pattern": "**", "effect": "allow"}
      ]
    }
  }
}`
	if err := os.WriteFile(filepath.Join(dir, "permissions.json"), []byte(jsonContent), 0o644); err != nil {
		t.Fatal(err)
	}

	origDir, err := os.Getwd()
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { os.Chdir(origDir) })
	if err := os.Chdir(tmp); err != nil {
		t.Fatal(err)
	}

	_, err = Load()
	if err == nil {
		t.Fatal("expected error for empty command name, got nil")
	}
	if !strings.Contains(err.Error(), "empty command name") {
		t.Errorf("expected error to mention 'empty command name', got: %v", err)
	}
}

func TestLoad_HierarchicalMergeDifferentCommands(t *testing.T) {
	// Parent dir has "git", child dir has "npm" — both should be present after merge.
	parent := t.TempDir()
	child := filepath.Join(parent, "project")

	parentCfgDir := filepath.Join(parent, ".ccbox")
	childCfgDir := filepath.Join(child, ".ccbox")
	if err := os.MkdirAll(parentCfgDir, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.MkdirAll(childCfgDir, 0o755); err != nil {
		t.Fatal(err)
	}

	parentYAML := `passthrough:
  git:
    rules:
      - pattern: "**"
        effect: allow
`
	childYAML := `passthrough:
  npm:
    rules:
      - pattern: "install"
        effect: allow
`
	if err := os.WriteFile(filepath.Join(parentCfgDir, "permissions.yaml"), []byte(parentYAML), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(childCfgDir, "permissions.yaml"), []byte(childYAML), 0o644); err != nil {
		t.Fatal(err)
	}

	origDir, err := os.Getwd()
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { os.Chdir(origDir) })
	if err := os.Chdir(child); err != nil {
		t.Fatal(err)
	}

	cfg, err := Load()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if cfg == nil {
		t.Fatal("expected non-nil config")
	}
	if _, ok := cfg.Passthrough["git"]; !ok {
		t.Error("expected 'git' command from parent config")
	}
	if _, ok := cfg.Passthrough["npm"]; !ok {
		t.Error("expected 'npm' command from child config")
	}
}

func TestLoad_HierarchicalMergeOverlappingCommand(t *testing.T) {
	// Parent and child both define "git" — child's rules should take precedence.
	parent := t.TempDir()
	child := filepath.Join(parent, "project")

	parentCfgDir := filepath.Join(parent, ".ccbox")
	childCfgDir := filepath.Join(child, ".ccbox")
	if err := os.MkdirAll(parentCfgDir, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.MkdirAll(childCfgDir, 0o755); err != nil {
		t.Fatal(err)
	}

	parentYAML := `passthrough:
  git:
    rules:
      - pattern: "**"
        effect: allow
`
	childYAML := `passthrough:
  git:
    rules:
      - pattern: "pull"
        effect: allow
`
	if err := os.WriteFile(filepath.Join(parentCfgDir, "permissions.yaml"), []byte(parentYAML), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(childCfgDir, "permissions.yaml"), []byte(childYAML), 0o644); err != nil {
		t.Fatal(err)
	}

	origDir, err := os.Getwd()
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { os.Chdir(origDir) })
	if err := os.Chdir(child); err != nil {
		t.Fatal(err)
	}

	cfg, err := Load()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if cfg == nil {
		t.Fatal("expected non-nil config")
	}
	git, ok := cfg.Passthrough["git"]
	if !ok {
		t.Fatal("expected 'git' command in passthrough")
	}
	// settings.Load merges maps recursively and appends slices,
	// so both parent's and child's rules are present (parent first, child appended).
	if len(git.Rules) != 2 {
		t.Fatalf("expected 2 rules (parent + child merged), got %d", len(git.Rules))
	}
	if len(git.Rules[0].Pattern.Values) != 1 || git.Rules[0].Pattern.Values[0] != "**" {
		t.Errorf("expected parent pattern [**], got %v", git.Rules[0].Pattern.Values)
	}
	if len(git.Rules[1].Pattern.Values) != 1 || git.Rules[1].Pattern.Values[0] != "pull" {
		t.Errorf("expected child pattern [pull], got %v", git.Rules[1].Pattern.Values)
	}
}

func TestLoad_BareStringRule(t *testing.T) {
	tmp := t.TempDir()
	dir := filepath.Join(tmp, ".ccbox")
	if err := os.MkdirAll(dir, 0o755); err != nil {
		t.Fatal(err)
	}

	// Common mistake: bare string instead of {pattern:, effect:} object
	yamlContent := `passthrough:
  git:
    rules:
      - status
`
	if err := os.WriteFile(filepath.Join(dir, "permissions.yaml"), []byte(yamlContent), 0o644); err != nil {
		t.Fatal(err)
	}

	origDir, err := os.Getwd()
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { os.Chdir(origDir) })
	if err := os.Chdir(tmp); err != nil {
		t.Fatal(err)
	}

	_, err = Load()
	if err == nil {
		t.Fatal("expected error for bare string rule, got nil")
	}
	if !strings.Contains(err.Error(), "pattern") && !strings.Contains(err.Error(), "effect") {
		t.Errorf("expected error to mention 'pattern' or 'effect', got: %v", err)
	}
}

func TestLoad_RuleMissingEffect(t *testing.T) {
	tmp := t.TempDir()
	dir := filepath.Join(tmp, ".ccbox")
	if err := os.MkdirAll(dir, 0o755); err != nil {
		t.Fatal(err)
	}

	yamlContent := `passthrough:
  git:
    rules:
      - pattern: status
`
	if err := os.WriteFile(filepath.Join(dir, "permissions.yaml"), []byte(yamlContent), 0o644); err != nil {
		t.Fatal(err)
	}

	origDir, err := os.Getwd()
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { os.Chdir(origDir) })
	if err := os.Chdir(tmp); err != nil {
		t.Fatal(err)
	}

	_, err = Load()
	if err == nil {
		t.Fatal("expected error for missing effect, got nil")
	}
	if !strings.Contains(err.Error(), "effect") {
		t.Errorf("expected error to mention 'effect', got: %v", err)
	}
}

func TestLoad_RuleMissingPattern(t *testing.T) {
	tmp := t.TempDir()
	dir := filepath.Join(tmp, ".ccbox")
	if err := os.MkdirAll(dir, 0o755); err != nil {
		t.Fatal(err)
	}

	yamlContent := `passthrough:
  git:
    rules:
      - effect: allow
`
	if err := os.WriteFile(filepath.Join(dir, "permissions.yaml"), []byte(yamlContent), 0o644); err != nil {
		t.Fatal(err)
	}

	origDir, err := os.Getwd()
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { os.Chdir(origDir) })
	if err := os.Chdir(tmp); err != nil {
		t.Fatal(err)
	}

	_, err = Load()
	if err == nil {
		t.Fatal("expected error for missing pattern, got nil")
	}
	if !strings.Contains(err.Error(), "pattern") {
		t.Errorf("expected error to mention 'pattern', got: %v", err)
	}
}

func TestLoad_RulesDirectlyUnderCommand(t *testing.T) {
	tmp := t.TempDir()
	dir := filepath.Join(tmp, ".ccbox")
	if err := os.MkdirAll(dir, 0o755); err != nil {
		t.Fatal(err)
	}

	// Common mistake: rules array directly under command name without "rules:" key
	yamlContent := `passthrough:
  git:
    - effect: allow
      pattern: status
`
	if err := os.WriteFile(filepath.Join(dir, "permissions.yaml"), []byte(yamlContent), 0o644); err != nil {
		t.Fatal(err)
	}

	origDir, err := os.Getwd()
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { os.Chdir(origDir) })
	if err := os.Chdir(tmp); err != nil {
		t.Fatal(err)
	}

	_, err = Load()
	if err == nil {
		t.Fatal("expected error for rules directly under command, got nil")
	}
	if !strings.Contains(err.Error(), "rules") {
		t.Errorf("expected error to mention 'rules' key, got: %v", err)
	}
}

func TestLoad_NoFile(t *testing.T) {
	tmp := t.TempDir()

	origDir, err := os.Getwd()
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { os.Chdir(origDir) })
	if err := os.Chdir(tmp); err != nil {
		t.Fatal(err)
	}

	cfg, err := Load()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if cfg != nil {
		t.Fatalf("expected nil config, got %+v", cfg)
	}
}
