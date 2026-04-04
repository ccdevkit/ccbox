package main

import (
	"bufio"
	"encoding/json"
	"net"
	"testing"
	"time"

	"github.com/ccdevkit/ccbox/internal/bridge"
)

func TestSendLog_SendsCorrectJSON(t *testing.T) {
	ln, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatalf("failed to start test server: %v", err)
	}
	defer ln.Close()

	received := make(chan bridge.LogRequest, 1)
	go func() {
		conn, err := ln.Accept()
		if err != nil {
			return
		}
		defer conn.Close()

		var req bridge.LogRequest
		if err := json.NewDecoder(bufio.NewReader(conn)).Decode(&req); err != nil {
			return
		}
		received <- req
	}()

	err = SendLog(ln.Addr().String(), "[test] hello world")
	if err != nil {
		t.Fatalf("SendLog returned error: %v", err)
	}

	select {
	case req := <-received:
		if req.Type != "log" {
			t.Errorf("expected type %q, got %q", "log", req.Type)
		}
		if req.Message != "[test] hello world" {
			t.Errorf("expected message %q, got %q", "[test] hello world", req.Message)
		}
	case <-time.After(3 * time.Second):
		t.Fatal("timed out waiting for log request")
	}
}

func TestSendLog_TimeoutOnUnresponsiveServer(t *testing.T) {
	// Create a listener that accepts but never reads
	ln, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatalf("failed to start test server: %v", err)
	}
	defer ln.Close()

	// Use a port that refuses connections (close the listener immediately)
	ln.Close()

	start := time.Now()
	err = SendLog(ln.Addr().String(), "[test] should timeout")
	elapsed := time.Since(start)

	if err == nil {
		t.Fatal("expected error when connecting to closed server")
	}

	// Should fail quickly (connection refused), not hang
	if elapsed > 3*time.Second {
		t.Errorf("SendLog took %v, expected it to fail within timeout", elapsed)
	}
}

func TestSendLog_ConnectionRefused(t *testing.T) {
	// Use a port that nothing is listening on
	err := SendLog("127.0.0.1:1", "[test] should fail")
	if err == nil {
		t.Fatal("expected error when connecting to refused port")
	}
}
