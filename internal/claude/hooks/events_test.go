package hooks

import "testing"

func TestHookEventConstants(t *testing.T) {
	tests := []struct {
		event HookEvent
		want  string
	}{
		// Lifecycle
		{SessionStart, "SessionStart"},
		{SessionEnd, "SessionEnd"},
		// Instruction
		{InstructionsLoaded, "InstructionsLoaded"},
		// Prompt
		{UserPromptSubmit, "UserPromptSubmit"},
		// Tool
		{PreToolUse, "PreToolUse"},
		{PostToolUse, "PostToolUse"},
		{PostToolUseFailure, "PostToolUseFailure"},
		{PermissionRequest, "PermissionRequest"},
		{PermissionDenied, "PermissionDenied"},
		// Agent/Task
		{SubagentStart, "SubagentStart"},
		{SubagentStop, "SubagentStop"},
		{TaskCreated, "TaskCreated"},
		{TaskCompleted, "TaskCompleted"},
		{TeammateIdle, "TeammateIdle"},
		// Workflow
		{Stop, "Stop"},
		{StopFailure, "StopFailure"},
		// Compact
		{PreCompact, "PreCompact"},
		{PostCompact, "PostCompact"},
		// File/Config
		{FileChanged, "FileChanged"},
		{CwdChanged, "CwdChanged"},
		{ConfigChange, "ConfigChange"},
		// Worktree
		{WorktreeCreate, "WorktreeCreate"},
		{WorktreeRemove, "WorktreeRemove"},
		// MCP
		{Elicitation, "Elicitation"},
		{ElicitationResult, "ElicitationResult"},
		// Other
		{Notification, "Notification"},
	}

	for _, tt := range tests {
		t.Run(tt.want, func(t *testing.T) {
			if got := string(tt.event); got != tt.want {
				t.Errorf("HookEvent = %q, want %q", got, tt.want)
			}
		})
	}
}

func TestHookEventCount(t *testing.T) {
	// Verify we have exactly 26 events defined
	all := []HookEvent{
		SessionStart, SessionEnd,
		InstructionsLoaded,
		UserPromptSubmit,
		PreToolUse, PostToolUse, PostToolUseFailure, PermissionRequest, PermissionDenied,
		SubagentStart, SubagentStop, TaskCreated, TaskCompleted, TeammateIdle,
		Stop, StopFailure,
		PreCompact, PostCompact,
		FileChanged, CwdChanged, ConfigChange,
		WorktreeCreate, WorktreeRemove,
		Elicitation, ElicitationResult,
		Notification,
	}
	if len(all) != 26 {
		t.Errorf("expected 26 hook events, got %d", len(all))
	}
}

func TestOrderConstants(t *testing.T) {
	tests := []struct {
		order Order
		want  string
	}{
		{OrderBefore, "before"},
		{OrderAfter, "after"},
	}

	for _, tt := range tests {
		t.Run(tt.want, func(t *testing.T) {
			if got := string(tt.order); got != tt.want {
				t.Errorf("Order = %q, want %q", got, tt.want)
			}
		})
	}
}
