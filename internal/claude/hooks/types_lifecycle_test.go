package hooks

import (
	"encoding/json"
	"testing"
)

func TestSessionStartInput_JSONRoundTrip(t *testing.T) {
	input := SessionStartInput{
		HookInputBase: HookInputBase{
			SessionID:      "sess-123",
			TranscriptPath: "/tmp/transcript.json",
			CWD:            "/home/user/project",
			PermissionMode: "default",
			HookEventName:  "SessionStart",
			AgentID:        "agent-1",
			AgentType:      "main",
		},
		Source: "cli",
		Model:  "claude-opus-4-6",
	}

	data, err := json.Marshal(input)
	if err != nil {
		t.Fatalf("marshal: %v", err)
	}

	var got SessionStartInput
	if err := json.Unmarshal(data, &got); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}

	if got.SessionID != input.SessionID {
		t.Errorf("SessionID = %q, want %q", got.SessionID, input.SessionID)
	}
	if got.TranscriptPath != input.TranscriptPath {
		t.Errorf("TranscriptPath = %q, want %q", got.TranscriptPath, input.TranscriptPath)
	}
	if got.CWD != input.CWD {
		t.Errorf("CWD = %q, want %q", got.CWD, input.CWD)
	}
	if got.PermissionMode != input.PermissionMode {
		t.Errorf("PermissionMode = %q, want %q", got.PermissionMode, input.PermissionMode)
	}
	if got.HookEventName != input.HookEventName {
		t.Errorf("HookEventName = %q, want %q", got.HookEventName, input.HookEventName)
	}
	if got.AgentID != input.AgentID {
		t.Errorf("AgentID = %q, want %q", got.AgentID, input.AgentID)
	}
	if got.AgentType != input.AgentType {
		t.Errorf("AgentType = %q, want %q", got.AgentType, input.AgentType)
	}
	if got.Source != input.Source {
		t.Errorf("Source = %q, want %q", got.Source, input.Source)
	}
	if got.Model != input.Model {
		t.Errorf("Model = %q, want %q", got.Model, input.Model)
	}
}

func TestSessionEndInput_JSONRoundTrip(t *testing.T) {
	input := SessionEndInput{
		HookInputBase: HookInputBase{
			SessionID:      "sess-456",
			TranscriptPath: "/tmp/transcript2.json",
			CWD:            "/home/user/other",
			PermissionMode: "plan",
			HookEventName:  "SessionEnd",
		},
	}

	data, err := json.Marshal(input)
	if err != nil {
		t.Fatalf("marshal: %v", err)
	}

	var got SessionEndInput
	if err := json.Unmarshal(data, &got); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}

	if got.SessionID != input.SessionID {
		t.Errorf("SessionID = %q, want %q", got.SessionID, input.SessionID)
	}
	if got.TranscriptPath != input.TranscriptPath {
		t.Errorf("TranscriptPath = %q, want %q", got.TranscriptPath, input.TranscriptPath)
	}
	if got.CWD != input.CWD {
		t.Errorf("CWD = %q, want %q", got.CWD, input.CWD)
	}
	if got.PermissionMode != input.PermissionMode {
		t.Errorf("PermissionMode = %q, want %q", got.PermissionMode, input.PermissionMode)
	}
	if got.HookEventName != input.HookEventName {
		t.Errorf("HookEventName = %q, want %q", got.HookEventName, input.HookEventName)
	}
}

func TestSessionStartInput_EmbeddedBaseFieldAccess(t *testing.T) {
	input := SessionStartInput{
		HookInputBase: HookInputBase{
			SessionID:     "sess-789",
			HookEventName: "SessionStart",
		},
		Source: "api",
		Model:  "claude-sonnet-4-6",
	}

	// Verify embedded fields are accessible directly (not via .HookInputBase)
	if input.SessionID != "sess-789" {
		t.Errorf("direct SessionID access = %q, want %q", input.SessionID, "sess-789")
	}
	if input.HookEventName != "SessionStart" {
		t.Errorf("direct HookEventName access = %q, want %q", input.HookEventName, "SessionStart")
	}
}

func TestSessionStartOutput_JSONRoundTrip(t *testing.T) {
	cont := true
	output := SessionStartOutput{
		HookOutputBase: HookOutputBase{
			Continue:      &cont,
			SystemMessage: "welcome",
		},
	}

	data, err := json.Marshal(output)
	if err != nil {
		t.Fatalf("marshal: %v", err)
	}

	var got SessionStartOutput
	if err := json.Unmarshal(data, &got); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}

	if got.Continue == nil || *got.Continue != true {
		t.Errorf("Continue = %v, want true", got.Continue)
	}
	if got.SystemMessage != "welcome" {
		t.Errorf("SystemMessage = %q, want %q", got.SystemMessage, "welcome")
	}
}
