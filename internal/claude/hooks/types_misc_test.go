package hooks

import (
	"encoding/json"
	"testing"
)

func TestFileChangedInputRoundTrip(t *testing.T) {
	input := FileChangedInput{
		HookInputBase: HookInputBase{
			SessionID:     "sess-1",
			HookEventName: string(FileChanged),
		},
		FilePath: "/tmp/foo.txt",
	}
	data, err := json.Marshal(input)
	if err != nil {
		t.Fatalf("marshal: %v", err)
	}
	var got FileChangedInput
	if err := json.Unmarshal(data, &got); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	if got.FilePath != "/tmp/foo.txt" {
		t.Errorf("FilePath = %q, want %q", got.FilePath, "/tmp/foo.txt")
	}
	if got.SessionID != "sess-1" {
		t.Errorf("SessionID = %q, want %q", got.SessionID, "sess-1")
	}
}

func TestCwdChangedInputRoundTrip(t *testing.T) {
	input := CwdChangedInput{
		HookInputBase: HookInputBase{
			SessionID:     "sess-2",
			HookEventName: string(CwdChanged),
		},
		OldCwd: "/old/path",
		NewCwd: "/new/path",
	}
	data, err := json.Marshal(input)
	if err != nil {
		t.Fatalf("marshal: %v", err)
	}
	var got CwdChangedInput
	if err := json.Unmarshal(data, &got); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	if got.OldCwd != "/old/path" {
		t.Errorf("OldCwd = %q, want %q", got.OldCwd, "/old/path")
	}
	if got.NewCwd != "/new/path" {
		t.Errorf("NewCwd = %q, want %q", got.NewCwd, "/new/path")
	}
}

func TestNotificationInputRoundTrip(t *testing.T) {
	input := NotificationInput{
		HookInputBase: HookInputBase{
			SessionID:     "sess-3",
			HookEventName: string(Notification),
		},
		Message: "deploy complete",
	}
	data, err := json.Marshal(input)
	if err != nil {
		t.Fatalf("marshal: %v", err)
	}
	var got NotificationInput
	if err := json.Unmarshal(data, &got); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	if got.Message != "deploy complete" {
		t.Errorf("Message = %q, want %q", got.Message, "deploy complete")
	}
}

func TestMiscTypesBaseFieldsAccessible(t *testing.T) {
	fc := FileChangedInput{
		HookInputBase: HookInputBase{
			SessionID:      "s1",
			TranscriptPath: "/t",
			CWD:            "/cwd",
			PermissionMode: "default",
			HookEventName:  "FileChanged",
			AgentID:        "a1",
			AgentType:      "main",
		},
		FilePath: "/f",
	}
	if fc.SessionID != "s1" {
		t.Error("embedded SessionID not accessible")
	}
	if fc.AgentID != "a1" {
		t.Error("embedded AgentID not accessible")
	}

	cwd := CwdChangedInput{}
	cwd.SessionID = "s2"
	if cwd.SessionID != "s2" {
		t.Error("embedded field not settable on CwdChangedInput")
	}

	cfg := ConfigChangeInput{}
	cfg.SessionID = "s3"
	if cfg.SessionID != "s3" {
		t.Error("embedded field not settable on ConfigChangeInput")
	}

	wc := WorktreeCreateInput{}
	wc.SessionID = "s4"
	if wc.SessionID != "s4" {
		t.Error("embedded field not settable on WorktreeCreateInput")
	}

	wr := WorktreeRemoveInput{}
	wr.SessionID = "s5"
	if wr.SessionID != "s5" {
		t.Error("embedded field not settable on WorktreeRemoveInput")
	}

	el := ElicitationInput{}
	el.SessionID = "s6"
	if el.SessionID != "s6" {
		t.Error("embedded field not settable on ElicitationInput")
	}

	er := ElicitationResultInput{}
	er.SessionID = "s7"
	if er.SessionID != "s7" {
		t.Error("embedded field not settable on ElicitationResultInput")
	}

	n := NotificationInput{}
	n.SessionID = "s8"
	if n.SessionID != "s8" {
		t.Error("embedded field not settable on NotificationInput")
	}
}
