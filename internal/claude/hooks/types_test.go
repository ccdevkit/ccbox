package hooks

import (
	"encoding/json"
	"testing"
)

func TestHookInputBase_UnmarshalAllFields(t *testing.T) {
	raw := `{
		"session_id": "sess-123",
		"transcript_path": "/tmp/transcript.json",
		"cwd": "/home/user/project",
		"permission_mode": "default",
		"hook_event_name": "PreToolUse",
		"agent_id": "agent-456",
		"agent_type": "subagent"
	}`

	var input HookInputBase
	if err := json.Unmarshal([]byte(raw), &input); err != nil {
		t.Fatalf("unmarshal failed: %v", err)
	}

	if input.SessionID != "sess-123" {
		t.Errorf("SessionID = %q, want %q", input.SessionID, "sess-123")
	}
	if input.TranscriptPath != "/tmp/transcript.json" {
		t.Errorf("TranscriptPath = %q, want %q", input.TranscriptPath, "/tmp/transcript.json")
	}
	if input.CWD != "/home/user/project" {
		t.Errorf("CWD = %q, want %q", input.CWD, "/home/user/project")
	}
	if input.PermissionMode != "default" {
		t.Errorf("PermissionMode = %q, want %q", input.PermissionMode, "default")
	}
	if input.HookEventName != "PreToolUse" {
		t.Errorf("HookEventName = %q, want %q", input.HookEventName, "PreToolUse")
	}
	if input.AgentID != "agent-456" {
		t.Errorf("AgentID = %q, want %q", input.AgentID, "agent-456")
	}
	if input.AgentType != "subagent" {
		t.Errorf("AgentType = %q, want %q", input.AgentType, "subagent")
	}
}

func TestHookInputBase_UnmarshalRequiredOnly(t *testing.T) {
	raw := `{
		"session_id": "sess-789",
		"transcript_path": "/tmp/t.json",
		"cwd": "/projects/foo",
		"permission_mode": "plan",
		"hook_event_name": "PostToolUse"
	}`

	var input HookInputBase
	if err := json.Unmarshal([]byte(raw), &input); err != nil {
		t.Fatalf("unmarshal failed: %v", err)
	}

	if input.SessionID != "sess-789" {
		t.Errorf("SessionID = %q, want %q", input.SessionID, "sess-789")
	}
	if input.HookEventName != "PostToolUse" {
		t.Errorf("HookEventName = %q, want %q", input.HookEventName, "PostToolUse")
	}
	if input.AgentID != "" {
		t.Errorf("AgentID = %q, want empty", input.AgentID)
	}
	if input.AgentType != "" {
		t.Errorf("AgentType = %q, want empty", input.AgentType)
	}
}

func TestHookOutputBase_MarshalContinueFalse(t *testing.T) {
	cont := false
	output := HookOutputBase{
		Continue:   &cont,
		StopReason: "policy violation",
	}

	data, err := json.Marshal(output)
	if err != nil {
		t.Fatalf("marshal failed: %v", err)
	}

	var m map[string]any
	if err := json.Unmarshal(data, &m); err != nil {
		t.Fatalf("re-unmarshal failed: %v", err)
	}

	if m["continue"] != false {
		t.Errorf("continue = %v, want false", m["continue"])
	}
	if m["stopReason"] != "policy violation" {
		t.Errorf("stopReason = %v, want %q", m["stopReason"], "policy violation")
	}
}

func TestHookOutputBase_MarshalNoContinue(t *testing.T) {
	output := HookOutputBase{
		SystemMessage: "hello",
	}

	data, err := json.Marshal(output)
	if err != nil {
		t.Fatalf("marshal failed: %v", err)
	}

	var m map[string]any
	if err := json.Unmarshal(data, &m); err != nil {
		t.Fatalf("re-unmarshal failed: %v", err)
	}

	if _, ok := m["continue"]; ok {
		t.Error("continue should be omitted when nil")
	}
	if m["systemMessage"] != "hello" {
		t.Errorf("systemMessage = %v, want %q", m["systemMessage"], "hello")
	}
}

func TestHandlerResult_Constructable(t *testing.T) {
	result := HandlerResult{
		ExitCode: 1,
		Stdout:   []byte("output"),
		Stderr:   []byte("error"),
	}

	if result.ExitCode != 1 {
		t.Errorf("ExitCode = %d, want 1", result.ExitCode)
	}
	if string(result.Stdout) != "output" {
		t.Errorf("Stdout = %q, want %q", result.Stdout, "output")
	}
	if string(result.Stderr) != "error" {
		t.Errorf("Stderr = %q, want %q", result.Stderr, "error")
	}
}
