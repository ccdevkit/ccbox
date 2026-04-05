package hooks

import (
	"encoding/json"
	"errors"
	"testing"
)

func TestPreToolUseHandler_Invoke(t *testing.T) {
	handler := PreToolUseHandler{
		Matcher: "Bash",
		Order:   OrderBefore,
		Fn: func(input *PreToolUseInput) (*PreToolUseOutput, error) {
			if input.ToolName != "Bash" {
				t.Errorf("expected tool_name Bash, got %s", input.ToolName)
			}
			if input.SessionID != "sess-123" {
				t.Errorf("expected session_id sess-123, got %s", input.SessionID)
			}
			return &PreToolUseOutput{
				HookSpecificOutput: &PreToolUseSpecificOutput{
					PermissionDecision: "allow",
				},
			}, nil
		},
	}

	raw := json.RawMessage(`{
		"session_id": "sess-123",
		"hook_event_name": "PreToolUse",
		"tool_name": "Bash",
		"tool_input": {"command": "ls"}
	}`)

	result, err := handler.invoke(raw)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result.ExitCode != 0 {
		t.Errorf("expected exit code 0, got %d", result.ExitCode)
	}

	var output PreToolUseOutput
	if err := json.Unmarshal(result.Stdout, &output); err != nil {
		t.Fatalf("failed to unmarshal stdout: %v", err)
	}
	if output.HookSpecificOutput == nil {
		t.Fatal("expected HookSpecificOutput to be non-nil")
	}
	if output.HookSpecificOutput.PermissionDecision != "allow" {
		t.Errorf("expected permissionDecision allow, got %s", output.HookSpecificOutput.PermissionDecision)
	}
}

func TestSessionStartHandler_InterfaceMethods(t *testing.T) {
	handler := SessionStartHandler{
		Matcher: "test-pattern",
		Order:   OrderAfter,
		Fn: func(input *SessionStartInput) (*SessionStartOutput, error) {
			return &SessionStartOutput{}, nil
		},
	}

	if handler.EventName() != SessionStart {
		t.Errorf("expected EventName SessionStart, got %s", handler.EventName())
	}
	if handler.MatcherPattern() != "test-pattern" {
		t.Errorf("expected MatcherPattern test-pattern, got %s", handler.MatcherPattern())
	}
	if handler.HandlerOrder() != OrderAfter {
		t.Errorf("expected HandlerOrder after, got %s", handler.HandlerOrder())
	}
}

func TestHandler_InvokeNilOutput(t *testing.T) {
	handler := SessionEndHandler{
		Order: OrderAfter,
		Fn: func(input *SessionEndInput) (*SessionEndOutput, error) {
			return nil, nil
		},
	}

	raw := json.RawMessage(`{"session_id": "s1", "hook_event_name": "SessionEnd"}`)
	result, err := handler.invoke(raw)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result.ExitCode != 0 {
		t.Errorf("expected exit code 0, got %d", result.ExitCode)
	}
	if len(result.Stdout) != 0 {
		t.Errorf("expected empty stdout for nil output, got %s", result.Stdout)
	}
}

func TestHandler_InvokeUnmarshalError(t *testing.T) {
	handler := PreToolUseHandler{
		Fn: func(input *PreToolUseInput) (*PreToolUseOutput, error) {
			t.Fatal("Fn should not be called on unmarshal error")
			return nil, nil
		},
	}

	raw := json.RawMessage(`{not valid json}`)
	_, err := handler.invoke(raw)
	if err == nil {
		t.Fatal("expected error for invalid JSON")
	}
}

func TestHandler_InvokeFnError(t *testing.T) {
	handler := NotificationHandler{
		Fn: func(input *NotificationInput) (*NotificationOutput, error) {
			return nil, errors.New("handler failed")
		},
	}

	raw := json.RawMessage(`{"session_id": "s1", "hook_event_name": "Notification", "message": "hi"}`)
	_, err := handler.invoke(raw)
	if err == nil {
		t.Fatal("expected error from Fn")
	}
	if err.Error() != "handler failed" {
		t.Errorf("expected 'handler failed', got %q", err.Error())
	}
}

func TestAllHandlers_ImplementInterface(t *testing.T) {
	// Compile-time check that all 26 handler types implement HookHandler.
	var _ HookHandler = SessionStartHandler{}
	var _ HookHandler = SessionEndHandler{}
	var _ HookHandler = InstructionsLoadedHandler{}
	var _ HookHandler = UserPromptSubmitHandler{}
	var _ HookHandler = PreToolUseHandler{}
	var _ HookHandler = PostToolUseHandler{}
	var _ HookHandler = PostToolUseFailureHandler{}
	var _ HookHandler = PermissionRequestHandler{}
	var _ HookHandler = PermissionDeniedHandler{}
	var _ HookHandler = SubagentStartHandler{}
	var _ HookHandler = SubagentStopHandler{}
	var _ HookHandler = TaskCreatedHandler{}
	var _ HookHandler = TaskCompletedHandler{}
	var _ HookHandler = TeammateIdleHandler{}
	var _ HookHandler = StopHandler{}
	var _ HookHandler = StopFailureHandler{}
	var _ HookHandler = PreCompactHandler{}
	var _ HookHandler = PostCompactHandler{}
	var _ HookHandler = FileChangedHandler{}
	var _ HookHandler = CwdChangedHandler{}
	var _ HookHandler = ConfigChangeHandler{}
	var _ HookHandler = WorktreeCreateHandler{}
	var _ HookHandler = WorktreeRemoveHandler{}
	var _ HookHandler = ElicitationHandler{}
	var _ HookHandler = ElicitationResultHandler{}
	var _ HookHandler = NotificationHandler{}
}

func TestPreToolUseOutput_ToResult_WithSpecificOutput(t *testing.T) {
	output := &PreToolUseOutput{
		HookOutputBase: HookOutputBase{
			SuppressOutput: true,
		},
		HookSpecificOutput: &PreToolUseSpecificOutput{
			PermissionDecision: "deny",
		},
	}

	result, err := output.toResult()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	var decoded PreToolUseOutput
	if err := json.Unmarshal(result.Stdout, &decoded); err != nil {
		t.Fatalf("failed to unmarshal: %v", err)
	}
	if !decoded.SuppressOutput {
		t.Error("expected suppressOutput true")
	}
	if decoded.HookSpecificOutput == nil || decoded.HookSpecificOutput.PermissionDecision != "deny" {
		t.Error("expected permissionDecision deny")
	}
}

func TestHookOutputBase_ToResult_Nil(t *testing.T) {
	var o *HookOutputBase
	result, err := o.toResult()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result.ExitCode != 0 {
		t.Errorf("expected exit code 0, got %d", result.ExitCode)
	}
	if len(result.Stdout) != 0 {
		t.Errorf("expected empty stdout, got %s", result.Stdout)
	}
}
