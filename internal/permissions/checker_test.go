package permissions

import (
	"strings"
	"testing"
)

func TestNewChecker_CLIAndFileWithDenyRules(t *testing.T) {
	config := &PermissionsConfig{
		Passthrough: map[string]*CommandPermission{
			"git": {
				Rules: []Rule{
					{
						Pattern: PatternOrArray{Values: []string{"push"}},
						Effect:  "deny",
						Reason:  "no pushing",
					},
				},
			},
		},
	}
	checker, err := NewChecker(config, []string{"git"})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	rules := checker.snapshot().commands["git"]
	if rules == nil {
		t.Fatal("expected non-nil rules for git")
	}
	if len(rules) != 2 {
		t.Fatalf("expected 2 rules, got %d", len(rules))
	}
	// First rule: CLI implicit allow **
	if rules[0].Effect != EffectAllow {
		t.Errorf("first rule effect = %q, want %q", rules[0].Effect, EffectAllow)
	}
	if rules[0].Pattern.Elements[0].Type != ElementDoubleWildcard {
		t.Errorf("first rule pattern element type = %q, want %q", rules[0].Pattern.Elements[0].Type, ElementDoubleWildcard)
	}
	// Second rule: file deny rule
	if rules[1].Effect != EffectDeny {
		t.Errorf("second rule effect = %q, want %q", rules[1].Effect, EffectDeny)
	}
	if rules[1].Reason != "no pushing" {
		t.Errorf("second rule reason = %q, want %q", rules[1].Reason, "no pushing")
	}
}

func TestNewChecker_CLIOnlyNoFileEntry(t *testing.T) {
	checker, err := NewChecker(nil, []string{"git"})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	rules := checker.snapshot().commands["git"]
	if rules == nil {
		t.Fatal("expected non-nil rules for git")
	}
	if len(rules) != 1 {
		t.Fatalf("expected 1 rule, got %d", len(rules))
	}
	if rules[0].Effect != EffectAllow {
		t.Errorf("rule effect = %q, want %q", rules[0].Effect, EffectAllow)
	}
	if rules[0].Pattern.Elements[0].Type != ElementDoubleWildcard {
		t.Errorf("rule pattern element type = %q, want %q", rules[0].Pattern.Elements[0].Type, ElementDoubleWildcard)
	}
}

func TestNewChecker_FileOnlyNoCliCommands(t *testing.T) {
	config := &PermissionsConfig{
		Passthrough: map[string]*CommandPermission{
			"docker": {
				Rules: []Rule{
					{
						Pattern: PatternOrArray{Values: []string{"build"}},
						Effect:  "allow",
					},
					{
						Pattern: PatternOrArray{Values: []string{"rm"}},
						Effect:  "deny",
						Reason:  "no removing",
					},
				},
			},
		},
	}
	checker, err := NewChecker(config, nil)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	rules := checker.snapshot().commands["docker"]
	if rules == nil {
		t.Fatal("expected non-nil rules for docker")
	}
	if len(rules) != 2 {
		t.Fatalf("expected 2 rules, got %d", len(rules))
	}
	// No prepended allow ** — file rules as-is
	if rules[0].Effect != EffectAllow {
		t.Errorf("first rule effect = %q, want %q", rules[0].Effect, EffectAllow)
	}
	if rules[0].Pattern.Elements[0].Value != "build" {
		t.Errorf("first rule pattern = %q, want %q", rules[0].Pattern.Elements[0].Value, "build")
	}
	if rules[1].Effect != EffectDeny {
		t.Errorf("second rule effect = %q, want %q", rules[1].Effect, EffectDeny)
	}
}

func TestNewChecker_CLIAndFileNullCommand(t *testing.T) {
	config := &PermissionsConfig{
		Passthrough: map[string]*CommandPermission{
			"git": nil, // null = unrestricted
		},
	}
	checker, err := NewChecker(config, []string{"git"})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	rules, exists := checker.snapshot().commands["git"]
	if !exists {
		t.Fatal("expected git to exist in commands")
	}
	if rules != nil {
		t.Fatalf("expected nil rules (unrestricted) for null command, got %d rules", len(rules))
	}
}

func TestNewChecker_NilConfigNilCLI(t *testing.T) {
	checker, err := NewChecker(nil, nil)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(checker.snapshot().commands) != 0 {
		t.Errorf("expected empty commands, got %d", len(checker.snapshot().commands))
	}
}

func TestNewChecker_InvalidPatternReturnsError(t *testing.T) {
	config := &PermissionsConfig{
		Passthrough: map[string]*CommandPermission{
			"git": {
				Rules: []Rule{
					{
						Pattern: PatternOrArray{Values: []string{"/[invalid/"}},
						Effect:  "allow",
					},
				},
			},
		},
	}
	_, err := NewChecker(config, nil)
	if err == nil {
		t.Fatal("expected error for invalid pattern")
	}
}

func TestNewChecker_HasCommand(t *testing.T) {
	checker, err := NewChecker(nil, []string{"git"})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !checker.HasCommand("git") {
		t.Error("expected HasCommand(git) = true")
	}
	if checker.HasCommand("docker") {
		t.Error("expected HasCommand(docker) = false")
	}
}

// --- T027: Commands() deduplicated list ---

func TestCommands_CLIAndFileDeduplicatedSorted(t *testing.T) {
	config := &PermissionsConfig{
		Passthrough: map[string]*CommandPermission{
			"git": {
				Rules: []Rule{
					{Pattern: PatternOrArray{Values: []string{"push"}}, Effect: "deny"},
				},
			},
			"docker": {
				Rules: []Rule{
					{Pattern: PatternOrArray{Values: []string{"build"}}, Effect: "allow"},
				},
			},
		},
	}
	// CLI includes "git" (overlap) and "npm" (new).
	checker, err := NewChecker(config, []string{"git", "npm"})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	got := checker.Commands()
	want := []string{"docker", "git", "npm"}
	if len(got) != len(want) {
		t.Fatalf("Commands() = %v, want %v", got, want)
	}
	for i := range want {
		if got[i] != want[i] {
			t.Errorf("Commands()[%d] = %q, want %q", i, got[i], want[i])
		}
	}
}

func TestCommands_FileOnly(t *testing.T) {
	config := &PermissionsConfig{
		Passthrough: map[string]*CommandPermission{
			"docker": {
				Rules: []Rule{
					{Pattern: PatternOrArray{Values: []string{"build"}}, Effect: "allow"},
				},
			},
			"make": {
				Rules: []Rule{
					{Pattern: PatternOrArray{Values: []string{"test"}}, Effect: "allow"},
				},
			},
		},
	}
	checker, err := NewChecker(config, nil)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	got := checker.Commands()
	want := []string{"docker", "make"}
	if len(got) != len(want) {
		t.Fatalf("Commands() = %v, want %v", got, want)
	}
	for i := range want {
		if got[i] != want[i] {
			t.Errorf("Commands()[%d] = %q, want %q", i, got[i], want[i])
		}
	}
}

func TestCommands_CLIOnly(t *testing.T) {
	checker, err := NewChecker(nil, []string{"npm", "git"})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	got := checker.Commands()
	want := []string{"git", "npm"}
	if len(got) != len(want) {
		t.Fatalf("Commands() = %v, want %v", got, want)
	}
	for i := range want {
		if got[i] != want[i] {
			t.Errorf("Commands()[%d] = %q, want %q", i, got[i], want[i])
		}
	}
}

// --- T028: Pattern validation at construction ---

func TestNewChecker_InvalidRegexIdentifiesPattern(t *testing.T) {
	config := &PermissionsConfig{
		Passthrough: map[string]*CommandPermission{
			"git": {
				Rules: []Rule{
					{Pattern: PatternOrArray{Values: []string{"/[/"}}, Effect: "allow"},
				},
			},
		},
	}
	_, err := NewChecker(config, nil)
	if err == nil {
		t.Fatal("expected error for invalid regex pattern")
	}
	if !strings.Contains(err.Error(), "git") {
		t.Errorf("error should identify command %q, got: %v", "git", err)
	}
	if !strings.Contains(err.Error(), "/[/") {
		t.Errorf("error should identify pattern, got: %v", err)
	}
}

func TestNewChecker_DollarMidPatternError(t *testing.T) {
	config := &PermissionsConfig{
		Passthrough: map[string]*CommandPermission{
			"git": {
				Rules: []Rule{
					{Pattern: PatternOrArray{Values: []string{"push $ --force"}}, Effect: "allow"},
				},
			},
		},
	}
	_, err := NewChecker(config, nil)
	if err == nil {
		t.Fatal("expected error for $ mid-pattern")
	}
	if !strings.Contains(err.Error(), "$") {
		t.Errorf("error should mention $, got: %v", err)
	}
}

// --- T029: Backward-compatible no-permissions-file path ---

func TestNewChecker_NilConfigWithCLI_CommandsReturnsCLI(t *testing.T) {
	checker, err := NewChecker(nil, []string{"git"})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !checker.HasCommand("git") {
		t.Error("expected HasCommand(git) = true")
	}
	// Should have allow ** rule.
	rules := checker.snapshot().commands["git"]
	if len(rules) != 1 {
		t.Fatalf("expected 1 rule, got %d", len(rules))
	}
	if rules[0].Effect != EffectAllow {
		t.Errorf("rule effect = %q, want %q", rules[0].Effect, EffectAllow)
	}
	if rules[0].Pattern.Elements[0].Type != ElementDoubleWildcard {
		t.Errorf("rule pattern type = %q, want %q", rules[0].Pattern.Elements[0].Type, ElementDoubleWildcard)
	}
	got := checker.Commands()
	if len(got) != 1 || got[0] != "git" {
		t.Errorf("Commands() = %v, want [git]", got)
	}
}

// --- T030: Check() core — last-match-wins ---

func TestCheck_LastMatchWins_DenyThenAllow(t *testing.T) {
	// [deny **, allow pull] → "git pull" allowed, "git push" denied
	config := &PermissionsConfig{
		Passthrough: map[string]*CommandPermission{
			"git": {
				Rules: []Rule{
					{Pattern: PatternOrArray{Values: []string{"**"}}, Effect: "deny"},
					{Pattern: PatternOrArray{Values: []string{"pull"}}, Effect: "allow"},
				},
			},
		},
	}
	checker, err := NewChecker(config, nil)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	r := checker.Check("git pull")
	if !r.Allowed {
		t.Errorf("expected 'git pull' allowed, got denied: %s", r.Reason)
	}
	r = checker.Check("git push")
	if r.Allowed {
		t.Error("expected 'git push' denied, got allowed")
	}
}

func TestCheck_LastMatchWins_AllowThenDeny(t *testing.T) {
	// [allow **, deny "push ~--force"] → "git push --force origin" denied, "git push origin" allowed
	config := &PermissionsConfig{
		Passthrough: map[string]*CommandPermission{
			"git": {
				Rules: []Rule{
					{Pattern: PatternOrArray{Values: []string{"**"}}, Effect: "allow"},
					{Pattern: PatternOrArray{Values: []string{"push ~--force"}}, Effect: "deny"},
				},
			},
		},
	}
	checker, err := NewChecker(config, nil)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	r := checker.Check("git push --force origin")
	if r.Allowed {
		t.Error("expected 'git push --force origin' denied, got allowed")
	}
	r = checker.Check("git push origin")
	if !r.Allowed {
		t.Errorf("expected 'git push origin' allowed, got denied: %s", r.Reason)
	}
}

// --- T031: Check() for commands with no rules ---

func TestCheck_NilRules_AllSubcommandsAllowed(t *testing.T) {
	config := &PermissionsConfig{
		Passthrough: map[string]*CommandPermission{
			"git": nil,
		},
	}
	checker, err := NewChecker(config, nil)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	r := checker.Check("git status")
	if !r.Allowed {
		t.Errorf("expected allowed, got denied: %s", r.Reason)
	}
	if r.Reason != "unrestricted" {
		t.Errorf("expected reason 'unrestricted', got %q", r.Reason)
	}
}

func TestCheck_EmptyRules_AllSubcommandsAllowed(t *testing.T) {
	config := &PermissionsConfig{
		Passthrough: map[string]*CommandPermission{
			"git": {Rules: []Rule{}},
		},
	}
	checker, err := NewChecker(config, nil)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	r := checker.Check("git anything")
	if !r.Allowed {
		t.Errorf("expected allowed, got denied: %s", r.Reason)
	}
	if r.Reason != "unrestricted" {
		t.Errorf("expected reason 'unrestricted', got %q", r.Reason)
	}
}

// --- T032: Check() with $ exact-match patterns ---

func TestCheck_ExactMatch_StatusDollar(t *testing.T) {
	config := &PermissionsConfig{
		Passthrough: map[string]*CommandPermission{
			"git": {
				Rules: []Rule{
					{Pattern: PatternOrArray{Values: []string{"status$"}}, Effect: "allow"},
				},
			},
		},
	}
	checker, err := NewChecker(config, nil)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	r := checker.Check("git status")
	if !r.Allowed {
		t.Errorf("expected 'git status' allowed, got denied: %s", r.Reason)
	}
	r = checker.Check("git status --short")
	if r.Allowed {
		t.Error("expected 'git status --short' denied, got allowed")
	}
}

// --- T033: Check() with regex patterns ---

func TestCheck_RegexPattern_CloneHTTPS(t *testing.T) {
	config := &PermissionsConfig{
		Passthrough: map[string]*CommandPermission{
			"git": {
				Rules: []Rule{
					{Pattern: PatternOrArray{Values: []string{`clone /^https?:\/\//`}}, Effect: "allow"},
				},
			},
		},
	}
	checker, err := NewChecker(config, nil)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	r := checker.Check("git clone https://github.com/repo")
	if !r.Allowed {
		t.Errorf("expected HTTPS clone allowed, got denied: %s", r.Reason)
	}
	r = checker.Check("git clone git@github.com:repo")
	if r.Allowed {
		t.Error("expected SSH clone denied, got allowed")
	}
}

// --- T034: Check() for command not in permissions ---

func TestCheck_CommandNotConfigured(t *testing.T) {
	config := &PermissionsConfig{
		Passthrough: map[string]*CommandPermission{
			"git": {
				Rules: []Rule{
					{Pattern: PatternOrArray{Values: []string{"**"}}, Effect: "allow"},
				},
			},
		},
	}
	checker, err := NewChecker(config, nil)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	r := checker.Check("npm install")
	if r.Allowed {
		t.Error("expected 'npm install' denied, got allowed")
	}
	if !strings.Contains(r.Reason, "command not configured") {
		t.Errorf("expected 'command not configured' in reason, got %q", r.Reason)
	}
}

// --- T035: Check() with array patterns ---

func TestCheck_ArrayPattern(t *testing.T) {
	config := &PermissionsConfig{
		Passthrough: map[string]*CommandPermission{
			"make": {
				Rules: []Rule{
					{Pattern: PatternOrArray{Values: []string{"build", "test", "lint"}}, Effect: "allow"},
				},
			},
		},
	}
	checker, err := NewChecker(config, nil)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	r := checker.Check("make build")
	if !r.Allowed {
		t.Errorf("expected 'make build' allowed, got denied: %s", r.Reason)
	}
	r = checker.Check("make deploy")
	if r.Allowed {
		t.Error("expected 'make deploy' denied, got allowed")
	}
}

// --- T036: Fail-closed default in Check() ---

func TestCheck_FailClosed_NoMatchingRule(t *testing.T) {
	config := &PermissionsConfig{
		Passthrough: map[string]*CommandPermission{
			"git": {
				Rules: []Rule{
					{Pattern: PatternOrArray{Values: []string{"status"}}, Effect: "allow"},
				},
			},
		},
	}
	checker, err := NewChecker(config, nil)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	r := checker.Check("git push")
	if r.Allowed {
		t.Error("expected 'git push' denied (fail-closed), got allowed")
	}
	if !strings.Contains(r.Reason, "no matching rule") {
		t.Errorf("expected 'no matching rule' in reason, got %q", r.Reason)
	}
	r = checker.Check("git status")
	if !r.Allowed {
		t.Errorf("expected 'git status' allowed, got denied: %s", r.Reason)
	}
}

// --- T037: No-rules vs has-rules distinction ---

func TestCheck_NilRules_AnySubcommandAllowed(t *testing.T) {
	config := &PermissionsConfig{
		Passthrough: map[string]*CommandPermission{
			"git": nil,
		},
	}
	checker, err := NewChecker(config, nil)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	r := checker.Check("git anything-goes")
	if !r.Allowed {
		t.Errorf("expected allowed for nil rules, got denied: %s", r.Reason)
	}
}

func TestCheck_DenyAll_EverythingDenied(t *testing.T) {
	config := &PermissionsConfig{
		Passthrough: map[string]*CommandPermission{
			"git": {
				Rules: []Rule{
					{Pattern: PatternOrArray{Values: []string{"**"}}, Effect: "deny"},
				},
			},
		},
	}
	checker, err := NewChecker(config, nil)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	r := checker.Check("git status")
	if r.Allowed {
		t.Error("expected denied with deny ** rule, got allowed")
	}
}

func TestCheck_OnlyDenyRules_EverythingDenied(t *testing.T) {
	config := &PermissionsConfig{
		Passthrough: map[string]*CommandPermission{
			"git": {
				Rules: []Rule{
					{Pattern: PatternOrArray{Values: []string{"push"}}, Effect: "deny"},
					{Pattern: PatternOrArray{Values: []string{"pull"}}, Effect: "deny"},
				},
			},
		},
	}
	checker, err := NewChecker(config, nil)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	// "push" matches deny rule → denied
	r := checker.Check("git push")
	if r.Allowed {
		t.Error("expected 'git push' denied")
	}
	// "status" matches no rule → fail-closed
	r = checker.Check("git status")
	if r.Allowed {
		t.Error("expected 'git status' denied (fail-closed)")
	}
}

// --- T039: Denial message for rule-based deny ---

func TestCheck_DenyWithReason_IncludesReason(t *testing.T) {
	config := &PermissionsConfig{
		Passthrough: map[string]*CommandPermission{
			"git": {
				Rules: []Rule{
					{Pattern: PatternOrArray{Values: []string{"push ~--force"}}, Effect: "deny", Reason: "Force push is destructive"},
				},
			},
		},
	}
	checker, err := NewChecker(config, nil)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	r := checker.Check("git push --force origin")
	if r.Allowed {
		t.Fatal("expected denied")
	}
	if r.Reason != "Force push is destructive" {
		t.Errorf("expected reason 'Force push is destructive', got %q", r.Reason)
	}
}

func TestCheck_DenyWithoutReason_IncludesPattern(t *testing.T) {
	config := &PermissionsConfig{
		Passthrough: map[string]*CommandPermission{
			"git": {
				Rules: []Rule{
					{Pattern: PatternOrArray{Values: []string{"push"}}, Effect: "deny"},
				},
			},
		},
	}
	checker, err := NewChecker(config, nil)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	r := checker.Check("git push origin")
	if r.Allowed {
		t.Fatal("expected denied")
	}
	if !strings.Contains(r.Reason, "push") {
		t.Errorf("expected pattern 'push' in reason, got %q", r.Reason)
	}
}

// --- T040: Denial message for no-match default deny ---

func TestCheck_NoMatch_ListsAvailablePatterns(t *testing.T) {
	config := &PermissionsConfig{
		Passthrough: map[string]*CommandPermission{
			"git": {
				Rules: []Rule{
					{Pattern: PatternOrArray{Values: []string{"status"}}, Effect: "allow"},
					{Pattern: PatternOrArray{Values: []string{"pull"}}, Effect: "allow"},
				},
			},
		},
	}
	checker, err := NewChecker(config, nil)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	r := checker.Check("git push")
	if r.Allowed {
		t.Fatal("expected denied")
	}
	if !strings.Contains(r.Reason, "no matching rule") {
		t.Errorf("expected 'no matching rule' in reason, got %q", r.Reason)
	}
	if !strings.Contains(r.Reason, "status") {
		t.Errorf("expected 'status' pattern listed in reason, got %q", r.Reason)
	}
	if !strings.Contains(r.Reason, "pull") {
		t.Errorf("expected 'pull' pattern listed in reason, got %q", r.Reason)
	}
}

// --- T044: Edge case tests ---

func TestNewChecker_EmptyPassthroughMap_NothingAllowed(t *testing.T) {
	config := &PermissionsConfig{
		Passthrough: map[string]*CommandPermission{},
	}
	checker, err := NewChecker(config, nil)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	got := checker.Commands()
	if len(got) != 0 {
		t.Errorf("Commands() = %v, want []", got)
	}
	r := checker.Check("anything")
	if r.Allowed {
		t.Error("expected 'anything' denied with empty passthrough map, got allowed")
	}
	if !strings.Contains(r.Reason, "command not configured") {
		t.Errorf("expected 'command not configured' in reason, got %q", r.Reason)
	}
}

func TestCheck_OnlyDenyRules_AllDenied(t *testing.T) {
	config := &PermissionsConfig{
		Passthrough: map[string]*CommandPermission{
			"git": {
				Rules: []Rule{
					{Pattern: PatternOrArray{Values: []string{"**"}}, Effect: "deny"},
				},
			},
		},
	}
	checker, err := NewChecker(config, nil)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	for _, cmd := range []string{"git status", "git push", "git pull --rebase", "git log --oneline"} {
		r := checker.Check(cmd)
		if r.Allowed {
			t.Errorf("expected %q denied with only deny ** rule, got allowed", cmd)
		}
	}
}

func TestCheck_DoubleWildcardWithTrailingLiteral_FewerArgs(t *testing.T) {
	config := &PermissionsConfig{
		Passthrough: map[string]*CommandPermission{
			"git": {
				Rules: []Rule{
					{Pattern: PatternOrArray{Values: []string{"push ** main"}}, Effect: "allow"},
				},
			},
		},
	}
	checker, err := NewChecker(config, nil)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	// "git push" → args=["push"], pattern needs "push", then ** consuming zero+, then "main"
	// With only ["push"], ** consumes zero args but "main" has nothing to match → denied.
	r := checker.Check("git push")
	if r.Allowed {
		t.Error("expected 'git push' denied when pattern is 'push ** main' (not enough args)")
	}
	// Verify it works when "main" is present.
	r = checker.Check("git push origin main")
	if !r.Allowed {
		t.Errorf("expected 'git push origin main' allowed, got denied: %s", r.Reason)
	}
}

func TestNewChecker_NilConfigNoCLI_EmptyCommands(t *testing.T) {
	checker, err := NewChecker(nil, nil)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(checker.snapshot().commands) != 0 {
		t.Errorf("expected empty commands, got %d", len(checker.snapshot().commands))
	}
	got := checker.Commands()
	if len(got) != 0 {
		t.Errorf("Commands() = %v, want []", got)
	}
}
