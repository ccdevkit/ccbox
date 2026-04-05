package hooks

import (
	"encoding/json"
	"testing"
)

func TestStopInput_JSONRoundTrip(t *testing.T) {
	input := StopInput{
		HookInputBase: HookInputBase{
			SessionID:      "sess-100",
			TranscriptPath: "/tmp/transcript.json",
			CWD:            "/home/user/project",
			PermissionMode: "default",
			HookEventName:  "Stop",
		},
		Reason: "user_requested",
	}

	data, err := json.Marshal(input)
	if err != nil {
		t.Fatalf("marshal: %v", err)
	}

	var got StopInput
	if err := json.Unmarshal(data, &got); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}

	if got.SessionID != input.SessionID {
		t.Errorf("SessionID = %q, want %q", got.SessionID, input.SessionID)
	}
	if got.HookEventName != input.HookEventName {
		t.Errorf("HookEventName = %q, want %q", got.HookEventName, input.HookEventName)
	}
	if got.Reason != input.Reason {
		t.Errorf("Reason = %q, want %q", got.Reason, input.Reason)
	}
}

func TestStopFailureInput_JSONRoundTrip(t *testing.T) {
	input := StopFailureInput{
		HookInputBase: HookInputBase{
			SessionID:      "sess-101",
			TranscriptPath: "/tmp/transcript2.json",
			CWD:            "/home/user/project",
			PermissionMode: "plan",
			HookEventName:  "StopFailure",
		},
		Error: "context_limit_exceeded",
	}

	data, err := json.Marshal(input)
	if err != nil {
		t.Fatalf("marshal: %v", err)
	}

	var got StopFailureInput
	if err := json.Unmarshal(data, &got); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}

	if got.SessionID != input.SessionID {
		t.Errorf("SessionID = %q, want %q", got.SessionID, input.SessionID)
	}
	if got.HookEventName != input.HookEventName {
		t.Errorf("HookEventName = %q, want %q", got.HookEventName, input.HookEventName)
	}
	if got.Error != input.Error {
		t.Errorf("Error = %q, want %q", got.Error, input.Error)
	}
}

func TestPreCompactInput_JSONRoundTrip(t *testing.T) {
	input := PreCompactInput{
		HookInputBase: HookInputBase{
			SessionID:      "sess-102",
			TranscriptPath: "/tmp/transcript3.json",
			CWD:            "/home/user/project",
			PermissionMode: "default",
			HookEventName:  "PreCompact",
			AgentID:        "agent-5",
			AgentType:      "subagent",
		},
	}

	data, err := json.Marshal(input)
	if err != nil {
		t.Fatalf("marshal: %v", err)
	}

	var got PreCompactInput
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
}
