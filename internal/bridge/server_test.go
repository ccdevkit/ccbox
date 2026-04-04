package bridge

import (
	"bytes"
	"fmt"
	"net"
	"testing"

	"github.com/ccdevkit/ccbox/internal/logger"
)

func TestServerExecRequest(t *testing.T) {
	var gotReq ExecRequest
	execHandler := func(req ExecRequest) (int, []byte) {
		gotReq = req
		return 0, []byte("hello world")
	}
	logHandler := func(req LogRequest) {}

	srv := NewServer(execHandler, logHandler)
	port, err := srv.Start()
	if err != nil {
		t.Fatalf("Start: %v", err)
	}
	defer srv.Stop()

	conn, err := net.Dial("tcp", fmt.Sprintf("127.0.0.1:%d", port))
	if err != nil {
		t.Fatalf("Dial: %v", err)
	}

	msg := `{"type":"exec","command":"echo hello","cwd":"/tmp"}` + "\n"
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

	if gotReq.Command != "echo hello" {
		t.Errorf("handler got Command = %q, want %q", gotReq.Command, "echo hello")
	}
	if gotReq.Cwd != "/tmp" {
		t.Errorf("handler got Cwd = %q, want %q", gotReq.Cwd, "/tmp")
	}

	want := "0\nhello world"
	if resp != want {
		t.Errorf("response = %q, want %q", resp, want)
	}
}

func TestServerLogRequest(t *testing.T) {
	done := make(chan LogRequest, 1)
	execHandler := func(req ExecRequest) (int, []byte) {
		return 0, nil
	}
	logHandler := func(req LogRequest) {
		done <- req
	}

	srv := NewServer(execHandler, logHandler)
	port, err := srv.Start()
	if err != nil {
		t.Fatalf("Start: %v", err)
	}
	defer srv.Stop()

	conn, err := net.Dial("tcp", fmt.Sprintf("127.0.0.1:%d", port))
	if err != nil {
		t.Fatalf("Dial: %v", err)
	}

	msg := `{"type":"log","message":"[test] hello"}` + "\n"
	if _, err := conn.Write([]byte(msg)); err != nil {
		t.Fatalf("Write: %v", err)
	}
	conn.Close()

	got := <-done
	if got.Message != "[test] hello" {
		t.Errorf("handler got Message = %q, want %q", got.Message, "[test] hello")
	}
}

func TestServerUnknownTypeSilentlyDropped(t *testing.T) {
	execHandler := func(req ExecRequest) (int, []byte) {
		t.Error("exec handler should not be called for unknown type")
		return 0, nil
	}
	logHandler := func(req LogRequest) {
		t.Error("log handler should not be called for unknown type")
	}

	srv := NewServer(execHandler, logHandler)
	port, err := srv.Start()
	if err != nil {
		t.Fatalf("Start: %v", err)
	}
	defer srv.Stop()

	conn, err := net.Dial("tcp", fmt.Sprintf("127.0.0.1:%d", port))
	if err != nil {
		t.Fatalf("Dial: %v", err)
	}

	msg := `{"type":"unknown","data":"stuff"}` + "\n"
	if _, err := conn.Write([]byte(msg)); err != nil {
		t.Fatalf("Write: %v", err)
	}
	conn.(*net.TCPConn).CloseWrite()

	// Server should close the connection without sending anything.
	buf := make([]byte, 4096)
	n, err := readAll(conn, buf)
	conn.Close()
	if err != nil {
		t.Fatalf("readAll: %v", err)
	}
	if n != 0 {
		t.Errorf("expected no response for unknown type, got %d bytes: %q", n, string(buf[:n]))
	}
}

func TestServerMalformedJSONSilentlyDropped(t *testing.T) {
	execHandler := func(req ExecRequest) (int, []byte) {
		t.Error("exec handler should not be called for malformed JSON")
		return 0, nil
	}
	logHandler := func(req LogRequest) {
		t.Error("log handler should not be called for malformed JSON")
	}

	srv := NewServer(execHandler, logHandler)
	port, err := srv.Start()
	if err != nil {
		t.Fatalf("Start: %v", err)
	}
	defer srv.Stop()

	conn, err := net.Dial("tcp", fmt.Sprintf("127.0.0.1:%d", port))
	if err != nil {
		t.Fatalf("Dial: %v", err)
	}

	msg := "this is not json at all\n"
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
	if n != 0 {
		t.Errorf("expected no response for malformed JSON, got %d bytes: %q", n, string(buf[:n]))
	}
}

func TestNewLogHandlerDisplaysContainerPrefix(t *testing.T) {
	// Create a logger that writes to a buffer so we can inspect output.
	var buf bytes.Buffer
	log := logger.NewWithWriter(&buf)

	handler := NewLogHandler(log)
	handler(LogRequest{Message: "starting up"})

	got := buf.String()
	want := "[container] starting up\n"
	if got != want {
		t.Errorf("log output = %q, want %q", got, want)
	}
}

// readAll reads from conn until EOF, returning bytes read.
func readAll(conn net.Conn, buf []byte) (int, error) {
	total := 0
	for {
		n, err := conn.Read(buf[total:])
		total += n
		if err != nil {
			if err.Error() == "EOF" {
				return total, nil
			}
			// net.OpError wrapping io.EOF
			if netErr, ok := err.(*net.OpError); ok && netErr.Err.Error() == "EOF" {
				return total, nil
			}
			return total, nil
		}
	}
}
