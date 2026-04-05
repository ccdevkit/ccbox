package hooks

import (
	"encoding/json"
	"testing"
)

func TestUserPromptSubmitInput_JSONRoundTrip(t *testing.T) {
	input := UserPromptSubmitInput{
		HookInputBase: HookInputBase{
			SessionID:     "sess-123",
			HookEventName: string(UserPromptSubmit),
		},
		Prompt: "hello world",
	}

	data, err := json.Marshal(input)
	if err != nil {
		t.Fatalf("marshal: %v", err)
	}

	var got UserPromptSubmitInput
	if err := json.Unmarshal(data, &got); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}

	if got.Prompt != "hello world" {
		t.Errorf("prompt = %q, want %q", got.Prompt, "hello world")
	}
	if got.SessionID != "sess-123" {
		t.Errorf("session_id = %q, want %q", got.SessionID, "sess-123")
	}
	if got.HookEventName != string(UserPromptSubmit) {
		t.Errorf("hook_event_name = %q, want %q", got.HookEventName, string(UserPromptSubmit))
	}
}

func TestInstructionsLoadedInput_JSONRoundTrip(t *testing.T) {
	raw := json.RawMessage(`{"system":"be helpful"}`)
	input := InstructionsLoadedInput{
		HookInputBase: HookInputBase{
			SessionID:     "sess-456",
			HookEventName: string(InstructionsLoaded),
			CWD:           "/tmp/project",
		},
		Instructions: raw,
	}

	data, err := json.Marshal(input)
	if err != nil {
		t.Fatalf("marshal: %v", err)
	}

	var got InstructionsLoadedInput
	if err := json.Unmarshal(data, &got); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}

	if string(got.Instructions) != `{"system":"be helpful"}` {
		t.Errorf("instructions = %s, want %s", got.Instructions, raw)
	}
	if got.CWD != "/tmp/project" {
		t.Errorf("cwd = %q, want %q", got.CWD, "/tmp/project")
	}
}

func TestInstructionTypes_EmbeddedBaseFields(t *testing.T) {
	// Verify embedded base fields are directly accessible
	input := UserPromptSubmitInput{}
	input.SessionID = "s1"
	input.TranscriptPath = "/tmp/transcript"
	input.PermissionMode = "default"

	if input.SessionID != "s1" {
		t.Error("SessionID not accessible via embedding")
	}
	if input.TranscriptPath != "/tmp/transcript" {
		t.Error("TranscriptPath not accessible via embedding")
	}

	output := UserPromptSubmitOutput{}
	cont := true
	output.Continue = &cont
	output.SuppressOutput = true

	if output.Continue == nil || *output.Continue != true {
		t.Error("Continue not accessible via embedding")
	}
	if !output.SuppressOutput {
		t.Error("SuppressOutput not accessible via embedding")
	}
}
