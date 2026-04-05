package hooks

import (
	"encoding/json"
	"testing"
)

func TestPreToolUseInput_Unmarshal(t *testing.T) {
	raw := `{
		"session_id": "sess-1",
		"transcript_path": "/tmp/transcript.json",
		"cwd": "/home/user",
		"permission_mode": "default",
		"hook_event_name": "PreToolUse",
		"tool_name": "Bash",
		"tool_input": {"command": "ls -la"}
	}`

	var input PreToolUseInput
	if err := json.Unmarshal([]byte(raw), &input); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}

	if input.SessionID != "sess-1" {
		t.Errorf("SessionID = %q, want %q", input.SessionID, "sess-1")
	}
	if input.ToolName != "Bash" {
		t.Errorf("ToolName = %q, want %q", input.ToolName, "Bash")
	}
	if input.HookEventName != "PreToolUse" {
		t.Errorf("HookEventName = %q, want %q", input.HookEventName, "PreToolUse")
	}

	// tool_input should be raw JSON
	var ti map[string]string
	if err := json.Unmarshal(input.ToolInput, &ti); err != nil {
		t.Fatalf("unmarshal tool_input: %v", err)
	}
	if ti["command"] != "ls -la" {
		t.Errorf("tool_input.command = %q, want %q", ti["command"], "ls -la")
	}
}

func TestPreToolUseOutput_Marshal(t *testing.T) {
	output := PreToolUseOutput{
		HookOutputBase: HookOutputBase{
			SuppressOutput: true,
		},
		HookSpecificOutput: &PreToolUseSpecificOutput{
			PermissionDecision: "allow",
		},
	}

	data, err := json.Marshal(output)
	if err != nil {
		t.Fatalf("marshal: %v", err)
	}

	var m map[string]interface{}
	if err := json.Unmarshal(data, &m); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}

	if m["suppressOutput"] != true {
		t.Errorf("suppressOutput = %v, want true", m["suppressOutput"])
	}

	specific, ok := m["hookSpecificOutput"].(map[string]interface{})
	if !ok {
		t.Fatalf("hookSpecificOutput not a map: %T", m["hookSpecificOutput"])
	}
	if specific["permissionDecision"] != "allow" {
		t.Errorf("permissionDecision = %v, want %q", specific["permissionDecision"], "allow")
	}
}

func TestPostToolUseInput_RoundTrip(t *testing.T) {
	original := PostToolUseInput{
		HookInputBase: HookInputBase{
			SessionID:     "sess-2",
			HookEventName: "PostToolUse",
		},
		ToolName:   "Read",
		ToolInput:  json.RawMessage(`{"file_path":"/tmp/foo.txt"}`),
		ToolResult: json.RawMessage(`{"content":"hello world"}`),
	}

	data, err := json.Marshal(original)
	if err != nil {
		t.Fatalf("marshal: %v", err)
	}

	var decoded PostToolUseInput
	if err := json.Unmarshal(data, &decoded); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}

	if decoded.ToolName != "Read" {
		t.Errorf("ToolName = %q, want %q", decoded.ToolName, "Read")
	}

	var result map[string]string
	if err := json.Unmarshal(decoded.ToolResult, &result); err != nil {
		t.Fatalf("unmarshal tool_result: %v", err)
	}
	if result["content"] != "hello world" {
		t.Errorf("tool_result.content = %q, want %q", result["content"], "hello world")
	}
}

func TestToolInput_PreservedAsRawJSON(t *testing.T) {
	// Ensure complex nested JSON is preserved exactly
	complexInput := `{"nested":{"key":[1,2,3]},"flag":true}`
	raw := `{"session_id":"s","cwd":"/","permission_mode":"d","hook_event_name":"PreToolUse","tool_name":"X","tool_input":` + complexInput + `}`

	var input PreToolUseInput
	if err := json.Unmarshal([]byte(raw), &input); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}

	// Re-marshal the raw message and compare
	var original, parsed interface{}
	json.Unmarshal([]byte(complexInput), &original)
	json.Unmarshal(input.ToolInput, &parsed)

	origBytes, _ := json.Marshal(original)
	parsedBytes, _ := json.Marshal(parsed)

	if string(origBytes) != string(parsedBytes) {
		t.Errorf("tool_input not preserved:\n  got:  %s\n  want: %s", parsedBytes, origBytes)
	}
}
