package bridge

import (
	"encoding/json"
	"fmt"
	"net"
	"testing"
)

func TestServerHookRequest(t *testing.T) {
	var gotReq HookRequest
	hookHandler := func(req HookRequest) HookResponse {
		gotReq = req
		return HookResponse{
			ExitCode: 0,
			Stdout:   `{"decision":"approve"}`,
			Stderr:   "",
		}
	}

	srv := NewServer(
		func(req ExecRequest) (int, []byte) { return 0, nil },
		func(req LogRequest) {},
		hookHandler,
	)
	port, err := srv.Start()
	if err != nil {
		t.Fatalf("Start: %v", err)
	}
	defer srv.Stop()

	conn, err := net.Dial("tcp", fmt.Sprintf("127.0.0.1:%d", port))
	if err != nil {
		t.Fatalf("Dial: %v", err)
	}

	msg := `{"type":"hook","event":"PreToolUse","input":{"tool":"bash"}}` + "\n"
	if _, err := conn.Write([]byte(msg)); err != nil {
		t.Fatalf("Write: %v", err)
	}
	conn.(*net.TCPConn).CloseWrite()

	buf := make([]byte, 4096)
	n, err := readAll(conn, buf)
	conn.Close()
	if err != nil {
		t.Fatalf("readAll: %v", err)
	}
	resp := string(buf[:n])

	if gotReq.Event != "PreToolUse" {
		t.Errorf("handler got Event = %q, want %q", gotReq.Event, "PreToolUse")
	}
	if string(gotReq.Input) != `{"tool":"bash"}` {
		t.Errorf("handler got Input = %s, want %s", gotReq.Input, `{"tool":"bash"}`)
	}

	want := `{"exit_code":0,"stdout":"{\"decision\":\"approve\"}","stderr":""}` + "\n"
	if resp != want {
		t.Errorf("response = %q, want %q", resp, want)
	}
}

func TestServerHookRequestExitCode2(t *testing.T) {
	hookHandler := func(req HookRequest) HookResponse {
		return HookResponse{
			ExitCode: 2,
			Stdout:   "",
			Stderr:   "action blocked by policy",
		}
	}

	srv := NewServer(
		func(req ExecRequest) (int, []byte) { return 0, nil },
		func(req LogRequest) {},
		hookHandler,
	)
	port, err := srv.Start()
	if err != nil {
		t.Fatalf("Start: %v", err)
	}
	defer srv.Stop()

	conn, err := net.Dial("tcp", fmt.Sprintf("127.0.0.1:%d", port))
	if err != nil {
		t.Fatalf("Dial: %v", err)
	}

	msg := `{"type":"hook","event":"PermissionRequest","input":{"session_id":"s2","hook_event_name":"PermissionRequest"}}` + "\n"
	if _, err := conn.Write([]byte(msg)); err != nil {
		t.Fatalf("Write: %v", err)
	}
	conn.(*net.TCPConn).CloseWrite()

	buf := make([]byte, 4096)
	n, err := readAll(conn, buf)
	conn.Close()
	if err != nil {
		t.Fatalf("readAll: %v", err)
	}

	var hookResp HookResponse
	if err := json.Unmarshal(buf[:n], &hookResp); err != nil {
		t.Fatalf("unmarshal response: %v (raw: %q)", err, string(buf[:n]))
	}

	if hookResp.ExitCode != 2 {
		t.Errorf("ExitCode = %d, want 2", hookResp.ExitCode)
	}
	if hookResp.Stderr != "action blocked by policy" {
		t.Errorf("Stderr = %q, want %q", hookResp.Stderr, "action blocked by policy")
	}
}

func TestServerHookRequestNilHandler(t *testing.T) {
	srv := NewServer(
		func(req ExecRequest) (int, []byte) { return 0, nil },
		func(req LogRequest) {},
		nil,
	)
	port, err := srv.Start()
	if err != nil {
		t.Fatalf("Start: %v", err)
	}
	defer srv.Stop()

	conn, err := net.Dial("tcp", fmt.Sprintf("127.0.0.1:%d", port))
	if err != nil {
		t.Fatalf("Dial: %v", err)
	}

	msg := `{"type":"hook","event":"PreToolUse","input":{"tool":"bash"}}` + "\n"
	if _, err := conn.Write([]byte(msg)); err != nil {
		t.Fatalf("Write: %v", err)
	}
	conn.(*net.TCPConn).CloseWrite()

	buf := make([]byte, 4096)
	n, err := readAll(conn, buf)
	conn.Close()
	if err != nil {
		t.Fatalf("readAll: %v", err)
	}
	resp := string(buf[:n])

	// Nil handler should return default success response
	want := `{"exit_code":0,"stdout":"","stderr":""}` + "\n"
	if resp != want {
		t.Errorf("response = %q, want %q", resp, want)
	}
}
