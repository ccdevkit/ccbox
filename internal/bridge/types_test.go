package bridge

import (
	"encoding/json"
	"testing"

	"github.com/ccdevkit/ccbox/internal/constants"
)

func TestExecRequestMarshalRoundTrip(t *testing.T) {
	req := ExecRequest{
		Type:    constants.ExecRequestType,
		Command: "ls -la",
		Cwd:     "/home/claude",
	}

	data, err := json.Marshal(req)
	if err != nil {
		t.Fatalf("marshal ExecRequest: %v", err)
	}

	var got ExecRequest
	if err := json.Unmarshal(data, &got); err != nil {
		t.Fatalf("unmarshal ExecRequest: %v", err)
	}

	if got != req {
		t.Errorf("round-trip mismatch: got %+v, want %+v", got, req)
	}
}

func TestExecRequestUnmarshalFromJSON(t *testing.T) {
	raw := `{"type":"exec","command":"echo hello","cwd":"/tmp"}`

	var req ExecRequest
	if err := json.Unmarshal([]byte(raw), &req); err != nil {
		t.Fatalf("unmarshal ExecRequest: %v", err)
	}

	if req.Type != constants.ExecRequestType {
		t.Errorf("Type = %q, want %q", req.Type, constants.ExecRequestType)
	}
	if req.Command != "echo hello" {
		t.Errorf("Command = %q, want %q", req.Command, "echo hello")
	}
	if req.Cwd != "/tmp" {
		t.Errorf("Cwd = %q, want %q", req.Cwd, "/tmp")
	}
}

func TestLogRequestMarshalRoundTrip(t *testing.T) {
	req := LogRequest{
		Type:    constants.LogRequestType,
		Message: "[ccptproxy] started",
	}

	data, err := json.Marshal(req)
	if err != nil {
		t.Fatalf("marshal LogRequest: %v", err)
	}

	var got LogRequest
	if err := json.Unmarshal(data, &got); err != nil {
		t.Fatalf("unmarshal LogRequest: %v", err)
	}

	if got != req {
		t.Errorf("round-trip mismatch: got %+v, want %+v", got, req)
	}
}

func TestLogRequestUnmarshalFromJSON(t *testing.T) {
	raw := `{"type":"log","message":"[ccdebug] test message"}`

	var req LogRequest
	if err := json.Unmarshal([]byte(raw), &req); err != nil {
		t.Fatalf("unmarshal LogRequest: %v", err)
	}

	if req.Type != constants.LogRequestType {
		t.Errorf("Type = %q, want %q", req.Type, constants.LogRequestType)
	}
	if req.Message != "[ccdebug] test message" {
		t.Errorf("Message = %q, want %q", req.Message, "[ccdebug] test message")
	}
}
