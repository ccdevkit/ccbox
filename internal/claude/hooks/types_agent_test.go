package hooks

import (
	"encoding/json"
	"testing"
)

func TestSubagentStartInput_JSONRoundTrip(t *testing.T) {
	input := SubagentStartInput{
		HookInputBase: HookInputBase{
			SessionID:     "sess-123",
			HookEventName: string(SubagentStart),
			AgentID:       "agent-1",
		},
		SubagentID:   "sub-42",
		SubagentType: "researcher",
	}

	data, err := json.Marshal(input)
	if err != nil {
		t.Fatalf("marshal: %v", err)
	}

	var got SubagentStartInput
	if err := json.Unmarshal(data, &got); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}

	if got.SubagentID != "sub-42" {
		t.Errorf("SubagentID = %q, want %q", got.SubagentID, "sub-42")
	}
	if got.SubagentType != "researcher" {
		t.Errorf("SubagentType = %q, want %q", got.SubagentType, "researcher")
	}
	if got.SessionID != "sess-123" {
		t.Errorf("SessionID = %q, want %q", got.SessionID, "sess-123")
	}
	if got.HookEventName != string(SubagentStart) {
		t.Errorf("HookEventName = %q, want %q", got.HookEventName, string(SubagentStart))
	}
}

func TestTaskCreatedInput_JSONRoundTrip(t *testing.T) {
	input := TaskCreatedInput{
		HookInputBase: HookInputBase{
			SessionID:      "sess-456",
			TranscriptPath: "/tmp/transcript.json",
			CWD:            "/home/user/project",
			PermissionMode: "auto",
			HookEventName:  string(TaskCreated),
		},
	}

	data, err := json.Marshal(input)
	if err != nil {
		t.Fatalf("marshal: %v", err)
	}

	var got TaskCreatedInput
	if err := json.Unmarshal(data, &got); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}

	if got.SessionID != "sess-456" {
		t.Errorf("SessionID = %q, want %q", got.SessionID, "sess-456")
	}
	if got.TranscriptPath != "/tmp/transcript.json" {
		t.Errorf("TranscriptPath = %q, want %q", got.TranscriptPath, "/tmp/transcript.json")
	}
	if got.CWD != "/home/user/project" {
		t.Errorf("CWD = %q, want %q", got.CWD, "/home/user/project")
	}
	if got.PermissionMode != "auto" {
		t.Errorf("PermissionMode = %q, want %q", got.PermissionMode, "auto")
	}
	if got.HookEventName != string(TaskCreated) {
		t.Errorf("HookEventName = %q, want %q", got.HookEventName, string(TaskCreated))
	}
}

func TestAgentTypes_EmbeddedBaseFields(t *testing.T) {
	// Verify embedded base fields are directly accessible on all agent types.
	start := SubagentStartInput{}
	start.SessionID = "s1"
	start.SubagentID = "sub-1"

	stop := SubagentStopInput{}
	stop.SessionID = "s2"
	stop.SubagentID = "sub-2"

	created := TaskCreatedInput{}
	created.SessionID = "s3"

	completed := TaskCompletedInput{}
	completed.SessionID = "s4"

	idle := TeammateIdleInput{}
	idle.SessionID = "s5"

	// Outputs embed HookOutputBase
	startOut := SubagentStartOutput{}
	cont := true
	startOut.Continue = &cont

	if start.SessionID != "s1" || stop.SessionID != "s2" ||
		created.SessionID != "s3" || completed.SessionID != "s4" ||
		idle.SessionID != "s5" {
		t.Error("embedded HookInputBase fields not accessible")
	}
	if startOut.Continue == nil || *startOut.Continue != true {
		t.Error("embedded HookOutputBase fields not accessible")
	}
}
