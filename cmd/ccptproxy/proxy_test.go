package main

import (
	"bufio"
	"encoding/json"
	"fmt"
	"net"
	"testing"
	"time"

	"github.com/ccdevkit/ccbox/internal/bridge"
)

func TestSendExec_SendsExecRequestJSON(t *testing.T) {
	ln, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatalf("failed to start test server: %v", err)
	}
	defer ln.Close()

	received := make(chan bridge.ExecRequest, 1)
	go func() {
		conn, err := ln.Accept()
		if err != nil {
			return
		}
		defer conn.Close()

		var req bridge.ExecRequest
		if err := json.NewDecoder(bufio.NewReader(conn)).Decode(&req); err != nil {
			return
		}
		received <- req

		// Send response
		fmt.Fprintf(conn, "0\nok\n")
	}()

	exitCode, _, err := SendExec(ln.Addr().String(), "echo hello", "/tmp")
	if err != nil {
		t.Fatalf("SendExec returned error: %v", err)
	}
	if exitCode != 0 {
		t.Errorf("expected exit code 0, got %d", exitCode)
	}

	select {
	case req := <-received:
		if req.Type != "exec" {
			t.Errorf("expected type %q, got %q", "exec", req.Type)
		}
		if req.Command != "echo hello" {
			t.Errorf("expected command %q, got %q", "echo hello", req.Command)
		}
		if req.Cwd != "/tmp" {
			t.Errorf("expected cwd %q, got %q", "/tmp", req.Cwd)
		}
	case <-time.After(3 * time.Second):
		t.Fatal("timed out waiting for exec request")
	}
}

func TestSendExec_CloseWriteCalledAfterWrite(t *testing.T) {
	ln, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatalf("failed to start test server: %v", err)
	}
	defer ln.Close()

	sawEOF := make(chan bool, 1)
	go func() {
		conn, err := ln.Accept()
		if err != nil {
			return
		}
		defer conn.Close()

		reader := bufio.NewReader(conn)
		// Read the JSON line
		_, err = reader.ReadBytes('\n')
		if err != nil {
			sawEOF <- false
			return
		}

		// After CloseWrite, a read should return EOF (0 bytes)
		buf := make([]byte, 1)
		_, err = reader.Read(buf)
		sawEOF <- (err != nil) // Should be EOF

		// Send response
		fmt.Fprintf(conn, "0\n")
	}()

	_, _, err = SendExec(ln.Addr().String(), "test", "/tmp")
	if err != nil {
		t.Fatalf("SendExec returned error: %v", err)
	}

	select {
	case eof := <-sawEOF:
		if !eof {
			t.Error("expected EOF after CloseWrite, but read succeeded")
		}
	case <-time.After(3 * time.Second):
		t.Fatal("timed out waiting for EOF check")
	}
}

func TestSendExec_ReadsExitCodeAndOutput(t *testing.T) {
	ln, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatalf("failed to start test server: %v", err)
	}
	defer ln.Close()

	go func() {
		conn, err := ln.Accept()
		if err != nil {
			return
		}
		defer conn.Close()

		// Drain request
		reader := bufio.NewReader(conn)
		reader.ReadBytes('\n')

		// Send response: exit code 42 with multi-line output
		fmt.Fprintf(conn, "42\nline1\nline2\n")
	}()

	exitCode, output, err := SendExec(ln.Addr().String(), "failing-cmd", "/tmp")
	if err != nil {
		t.Fatalf("SendExec returned error: %v", err)
	}
	if exitCode != 42 {
		t.Errorf("expected exit code 42, got %d", exitCode)
	}
	if string(output) != "line1\nline2\n" {
		t.Errorf("expected output %q, got %q", "line1\nline2\n", string(output))
	}
}

func TestSendExec_ResponseFormatExitCodeNewlineOutput(t *testing.T) {
	ln, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatalf("failed to start test server: %v", err)
	}
	defer ln.Close()

	go func() {
		conn, err := ln.Accept()
		if err != nil {
			return
		}
		defer conn.Close()

		reader := bufio.NewReader(conn)
		reader.ReadBytes('\n')

		// Response is exactly: {exit_code}\n{output_bytes}
		conn.Write([]byte("0\nhello world"))
	}()

	exitCode, output, err := SendExec(ln.Addr().String(), "echo hello world", "/home")
	if err != nil {
		t.Fatalf("SendExec returned error: %v", err)
	}
	if exitCode != 0 {
		t.Errorf("expected exit code 0, got %d", exitCode)
	}
	if string(output) != "hello world" {
		t.Errorf("expected output %q, got %q", "hello world", string(output))
	}
}

func TestSendExec_ConnectionRefused(t *testing.T) {
	_, _, err := SendExec("127.0.0.1:1", "echo test", "/tmp")
	if err == nil {
		t.Fatal("expected error when connecting to refused port")
	}
}
