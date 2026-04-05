package hooks

import (
	"encoding/json"
	"testing"
)

func bypassInput(toolName, permMode string) json.RawMessage {
	data, _ := json.Marshal(map[string]interface{}{
		"hook_event_name": "PreToolUse",
		"tool_name":       toolName,
		"tool_input":      map[string]interface{}{"file_path": "/repo/.git/config"},
		"session_id":      "s1",
		"transcript_path": "/t",
		"cwd":             "/",
		"permission_mode": permMode,
	})
	return data
}

func TestBypassPermissions_WriteAllowed(t *testing.T) {
	r := NewRegistry()
	RegisterBypassPermissions(r)

	result := r.Dispatch(PreToolUse, bypassInput("Write", "bypassPermissions"))

	if result.ExitCode != 0 {
		t.Fatalf("ExitCode = %d, want 0", result.ExitCode)
	}
	decision := extractDecision(result.Stdout)
	if decision != "allow" {
		t.Errorf("decision = %q, want %q", decision, "allow")
	}
}

func TestBypassPermissions_EditAllowed(t *testing.T) {
	r := NewRegistry()
	RegisterBypassPermissions(r)

	result := r.Dispatch(PreToolUse, bypassInput("Edit", "bypassPermissions"))

	if result.ExitCode != 0 {
		t.Fatalf("ExitCode = %d, want 0", result.ExitCode)
	}
	decision := extractDecision(result.Stdout)
	if decision != "allow" {
		t.Errorf("decision = %q, want %q", decision, "allow")
	}
}

func TestBypassPermissions_NonBypassMode_NoDecision(t *testing.T) {
	r := NewRegistry()
	RegisterBypassPermissions(r)

	result := r.Dispatch(PreToolUse, bypassInput("Write", "default"))

	if result.ExitCode != 0 {
		t.Fatalf("ExitCode = %d, want 0", result.ExitCode)
	}
	// In non-bypass mode, handler returns nil (no opinion) — no allow decision
	decision := extractDecision(result.Stdout)
	if decision != "" {
		t.Errorf("decision = %q, want empty (no opinion)", decision)
	}
}

func TestBypassPermissions_OtherTool_NotMatched(t *testing.T) {
	r := NewRegistry()
	RegisterBypassPermissions(r)

	result := r.Dispatch(PreToolUse, bypassInput("Bash", "bypassPermissions"))

	if result.ExitCode != 0 {
		t.Fatalf("ExitCode = %d, want 0", result.ExitCode)
	}
	// Bash doesn't match Write|Edit matcher, so handler doesn't run
	decision := extractDecision(result.Stdout)
	if decision != "" {
		t.Errorf("decision = %q, want empty (handler not matched)", decision)
	}
}

func TestBypassPermissions_MultiEdit_Allowed(t *testing.T) {
	r := NewRegistry()
	RegisterBypassPermissions(r)

	result := r.Dispatch(PreToolUse, bypassInput("MultiEdit", "bypassPermissions"))

	if result.ExitCode != 0 {
		t.Fatalf("ExitCode = %d, want 0", result.ExitCode)
	}
	decision := extractDecision(result.Stdout)
	if decision != "allow" {
		t.Errorf("decision = %q, want %q", decision, "allow")
	}
}

func TestBypassPermissions_IsAfterOrder(t *testing.T) {
	r := NewRegistry()
	RegisterBypassPermissions(r)

	// Verify the handler is registered with after ordering
	// by checking it doesn't interfere with before-phase handlers
	beforeCalled := false
	r.Register(PreToolUseHandler{
		Matcher: "Write",
		Order:   OrderBefore,
		Fn: func(input *PreToolUseInput) (*PreToolUseOutput, error) {
			beforeCalled = true
			return nil, nil
		},
	})

	r.Dispatch(PreToolUse, bypassInput("Write", "bypassPermissions"))

	if !beforeCalled {
		t.Error("before handler should have been called first")
	}
}
