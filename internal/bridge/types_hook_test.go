package bridge

import (
	"encoding/json"
	"testing"

	"github.com/ccdevkit/ccbox/internal/constants"
)

func TestHookRequestMarshalRoundTrip(t *testing.T) {
	input := json.RawMessage(`{"session_id":"s1","hook_event_name":"PreToolUse","tool_name":"Bash"}`)
	req := HookRequest{
		Type:  constants.HookRequestType,
		Event: "PreToolUse",
		Input: input,
	}

	data, err := json.Marshal(req)
	if err != nil {
		t.Fatalf("marshal HookRequest: %v", err)
	}

	var got HookRequest
	if err := json.Unmarshal(data, &got); err != nil {
		t.Fatalf("unmarshal HookRequest: %v", err)
	}

	if got.Type != req.Type {
		t.Errorf("Type = %q, want %q", got.Type, req.Type)
	}
	if got.Event != req.Event {
		t.Errorf("Event = %q, want %q", got.Event, req.Event)
	}
	if string(got.Input) != string(req.Input) {
		t.Errorf("Input = %s, want %s", got.Input, req.Input)
	}
}

func TestHookRequestUnmarshalFromJSON(t *testing.T) {
	raw := `{"type":"hook","event":"SessionStart","input":{"session_id":"abc","hook_event_name":"SessionStart"}}`

	var req HookRequest
	if err := json.Unmarshal([]byte(raw), &req); err != nil {
		t.Fatalf("unmarshal HookRequest: %v", err)
	}

	if req.Type != constants.HookRequestType {
		t.Errorf("Type = %q, want %q", req.Type, constants.HookRequestType)
	}
	if req.Event != "SessionStart" {
		t.Errorf("Event = %q, want %q", req.Event, "SessionStart")
	}
	if req.Input == nil {
		t.Fatal("Input is nil, want non-nil")
	}

	// Verify the raw input preserves the original JSON
	var inputMap map[string]string
	if err := json.Unmarshal(req.Input, &inputMap); err != nil {
		t.Fatalf("unmarshal Input: %v", err)
	}
	if inputMap["session_id"] != "abc" {
		t.Errorf("Input session_id = %q, want %q", inputMap["session_id"], "abc")
	}
}

func TestHookResponseMarshalRoundTrip(t *testing.T) {
	resp := HookResponse{
		ExitCode: 0,
		Stdout:   `{"continue":true}`,
		Stderr:   "",
	}

	data, err := json.Marshal(resp)
	if err != nil {
		t.Fatalf("marshal HookResponse: %v", err)
	}

	var got HookResponse
	if err := json.Unmarshal(data, &got); err != nil {
		t.Fatalf("unmarshal HookResponse: %v", err)
	}

	if got.ExitCode != resp.ExitCode {
		t.Errorf("ExitCode = %d, want %d", got.ExitCode, resp.ExitCode)
	}
	if got.Stdout != resp.Stdout {
		t.Errorf("Stdout = %q, want %q", got.Stdout, resp.Stdout)
	}
	if got.Stderr != resp.Stderr {
		t.Errorf("Stderr = %q, want %q", got.Stderr, resp.Stderr)
	}
}

func TestHookResponseUnmarshalFromJSON(t *testing.T) {
	raw := `{"exit_code":2,"stdout":"","stderr":"hook denied the action"}`

	var resp HookResponse
	if err := json.Unmarshal([]byte(raw), &resp); err != nil {
		t.Fatalf("unmarshal HookResponse: %v", err)
	}

	if resp.ExitCode != 2 {
		t.Errorf("ExitCode = %d, want %d", resp.ExitCode, 2)
	}
	if resp.Stdout != "" {
		t.Errorf("Stdout = %q, want empty", resp.Stdout)
	}
	if resp.Stderr != "hook denied the action" {
		t.Errorf("Stderr = %q, want %q", resp.Stderr, "hook denied the action")
	}
}
