package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net"
	"os"
	"testing"
	"time"

	"github.com/ccdevkit/ccbox/internal/bridge"
	"github.com/ccdevkit/ccbox/internal/constants"
)

// startTestServer starts a TCP server that accepts connections in a loop.
// Log requests (from debugLog) are silently consumed. Hook requests get the
// given HookResponse. Returns the listener (caller must close).
func startTestServer(t *testing.T, resp bridge.HookResponse) net.Listener {
	t.Helper()
	ln, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatalf("failed to start test server: %v", err)
	}

	go func() {
		for {
			conn, err := ln.Accept()
			if err != nil {
				return // listener closed
			}
			go func(c net.Conn) {
				defer c.Close()

				reqBytes, err := io.ReadAll(c)
				if err != nil {
					return
				}

				// Peek at the type field to distinguish log vs hook requests
				var envelope struct {
					Type string `json:"type"`
				}
				if json.Unmarshal(reqBytes, &envelope) != nil {
					return
				}

				// Silently consume log requests from debugLog
				if envelope.Type == constants.LogRequestType {
					return
				}

				// Hook request — send response
				respBytes, _ := json.Marshal(resp)
				c.Write(respBytes)
			}(conn)
		}
	}()

	return ln
}

func TestRunSuccess(t *testing.T) {
	// Save and restore timeouts
	origDial, origResp := dialTimeout, responseTimeout
	defer func() { dialTimeout, responseTimeout = origDial, origResp }()
	dialTimeout = 2 * time.Second
	responseTimeout = 5 * time.Second

	wantStdout := `{"decision":"approve"}`
	resp := bridge.HookResponse{
		ExitCode: 0,
		Stdout:   wantStdout,
		Stderr:   "",
	}

	ln := startTestServer(t, resp)
	defer ln.Close()

	// Extract port from listener address
	_, port, _ := net.SplitHostPort(ln.Addr().String())

	// Set env vars for the run function
	os.Setenv("DOCKER_HOSTNAME", "127.0.0.1")
	os.Setenv(constants.EnvCCBoxTCPPort, port)
	defer os.Unsetenv("DOCKER_HOSTNAME")
	defer os.Unsetenv(constants.EnvCCBoxTCPPort)

	// Prepare hook input
	hookInput := map[string]interface{}{
		"hook_event_name": "PreToolUse",
		"tool_name":       "Bash",
		"tool_input":      map[string]string{"command": "ls"},
	}
	inputBytes, _ := json.Marshal(hookInput)

	var stdout, stderr bytes.Buffer
	exitCode := run(bytes.NewReader(inputBytes), &stdout, &stderr)

	if exitCode != 0 {
		t.Errorf("expected exit code 0, got %d; stderr: %s", exitCode, stderr.String())
	}
	if stdout.String() != wantStdout {
		t.Errorf("expected stdout %q, got %q", wantStdout, stdout.String())
	}
	if stderr.String() != "" {
		t.Errorf("expected empty stderr, got %q", stderr.String())
	}
}

func TestRunNonZeroExitCode(t *testing.T) {
	origDial, origResp := dialTimeout, responseTimeout
	defer func() { dialTimeout, responseTimeout = origDial, origResp }()
	dialTimeout = 2 * time.Second
	responseTimeout = 5 * time.Second

	resp := bridge.HookResponse{
		ExitCode: 2,
		Stdout:   "",
		Stderr:   "blocked by policy",
	}

	ln := startTestServer(t, resp)
	defer ln.Close()

	_, port, _ := net.SplitHostPort(ln.Addr().String())
	os.Setenv("DOCKER_HOSTNAME", "127.0.0.1")
	os.Setenv(constants.EnvCCBoxTCPPort, port)
	defer os.Unsetenv("DOCKER_HOSTNAME")
	defer os.Unsetenv(constants.EnvCCBoxTCPPort)

	hookInput := map[string]interface{}{
		"hook_event_name": "PreToolUse",
		"tool_name":       "Bash",
	}
	inputBytes, _ := json.Marshal(hookInput)

	var stdout, stderr bytes.Buffer
	exitCode := run(bytes.NewReader(inputBytes), &stdout, &stderr)

	if exitCode != 2 {
		t.Errorf("expected exit code 2, got %d", exitCode)
	}
	if stderr.String() != "blocked by policy" {
		t.Errorf("expected stderr %q, got %q", "blocked by policy", stderr.String())
	}
	if stdout.String() != "" {
		t.Errorf("expected empty stdout, got %q", stdout.String())
	}
}

func TestRunConnectionFailed(t *testing.T) {
	origDial, origResp := dialTimeout, responseTimeout
	defer func() { dialTimeout, responseTimeout = origDial, origResp }()
	dialTimeout = 100 * time.Millisecond
	responseTimeout = 100 * time.Millisecond

	// Use a port that nothing listens on
	os.Setenv("DOCKER_HOSTNAME", "127.0.0.1")
	os.Setenv(constants.EnvCCBoxTCPPort, "1")
	defer os.Unsetenv("DOCKER_HOSTNAME")
	defer os.Unsetenv(constants.EnvCCBoxTCPPort)

	hookInput := `{"hook_event_name":"PreToolUse"}`

	var stdout, stderr bytes.Buffer
	exitCode := run(bytes.NewReader([]byte(hookInput)), &stdout, &stderr)

	if exitCode != 1 {
		t.Errorf("expected exit code 1, got %d", exitCode)
	}
	got := stderr.String()
	if got == "" {
		t.Fatal("expected error on stderr, got empty")
	}
	wantPrefix := "cchookproxy: connection failed:"
	if !bytes.Contains([]byte(got), []byte(wantPrefix)) {
		t.Errorf("expected stderr to contain %q, got %q", wantPrefix, got)
	}
}

func TestRunMissingPort(t *testing.T) {
	os.Setenv("DOCKER_HOSTNAME", "127.0.0.1")
	os.Unsetenv(constants.EnvCCBoxTCPPort)

	hookInput := `{"hook_event_name":"PreToolUse"}`

	var stdout, stderr bytes.Buffer
	exitCode := run(bytes.NewReader([]byte(hookInput)), &stdout, &stderr)

	if exitCode != 1 {
		t.Errorf("expected exit code 1, got %d", exitCode)
	}
	want := fmt.Sprintf("cchookproxy: %s not set\n", constants.EnvCCBoxTCPPort)
	if stderr.String() != want {
		t.Errorf("expected stderr %q, got %q", want, stderr.String())
	}
}

func TestRunResponseTimeout(t *testing.T) {
	origDial, origResp := dialTimeout, responseTimeout
	defer func() { dialTimeout, responseTimeout = origDial, origResp }()
	dialTimeout = 2 * time.Second
	responseTimeout = 100 * time.Millisecond

	// Start a server that accepts connections in a loop.
	// Log requests are consumed; hook requests hang (no response).
	ln, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatalf("failed to start test server: %v", err)
	}
	defer ln.Close()

	go func() {
		for {
			conn, err := ln.Accept()
			if err != nil {
				return
			}
			go func(c net.Conn) {
				defer c.Close()
				reqBytes, _ := io.ReadAll(c)
				var envelope struct {
					Type string `json:"type"`
				}
				if json.Unmarshal(reqBytes, &envelope) == nil && envelope.Type == constants.LogRequestType {
					return // consume debug log
				}
				// Hold connection open without responding — forces client read timeout
				time.Sleep(5 * time.Second)
			}(conn)
		}
	}()

	_, port, _ := net.SplitHostPort(ln.Addr().String())
	os.Setenv("DOCKER_HOSTNAME", "127.0.0.1")
	os.Setenv(constants.EnvCCBoxTCPPort, port)
	defer os.Unsetenv("DOCKER_HOSTNAME")
	defer os.Unsetenv(constants.EnvCCBoxTCPPort)

	hookInput := `{"hook_event_name":"PreToolUse"}`

	var stdout, stderr bytes.Buffer
	exitCode := run(bytes.NewReader([]byte(hookInput)), &stdout, &stderr)

	if exitCode != 1 {
		t.Errorf("expected exit code 1, got %d", exitCode)
	}
	if stderr.String() != "cchookproxy: response timeout\n" {
		t.Errorf("expected 'response timeout' stderr, got %q", stderr.String())
	}
}

func TestRunInvalidResponse(t *testing.T) {
	origDial, origResp := dialTimeout, responseTimeout
	defer func() { dialTimeout, responseTimeout = origDial, origResp }()
	dialTimeout = 2 * time.Second
	responseTimeout = 5 * time.Second

	// Start a server that sends garbage for hook requests
	ln, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatalf("failed to start test server: %v", err)
	}
	defer ln.Close()

	go func() {
		for {
			conn, err := ln.Accept()
			if err != nil {
				return
			}
			go func(c net.Conn) {
				defer c.Close()
				reqBytes, _ := io.ReadAll(c)
				var envelope struct {
					Type string `json:"type"`
				}
				if json.Unmarshal(reqBytes, &envelope) == nil && envelope.Type == constants.LogRequestType {
					return // consume debug log
				}
				c.Write([]byte("not json"))
			}(conn)
		}
	}()

	_, port, _ := net.SplitHostPort(ln.Addr().String())
	os.Setenv("DOCKER_HOSTNAME", "127.0.0.1")
	os.Setenv(constants.EnvCCBoxTCPPort, port)
	defer os.Unsetenv("DOCKER_HOSTNAME")
	defer os.Unsetenv(constants.EnvCCBoxTCPPort)

	hookInput := `{"hook_event_name":"PreToolUse"}`

	var stdout, stderr bytes.Buffer
	exitCode := run(bytes.NewReader([]byte(hookInput)), &stdout, &stderr)

	if exitCode != 1 {
		t.Errorf("expected exit code 1, got %d", exitCode)
	}
	if stderr.String() != "cchookproxy: invalid response\n" {
		t.Errorf("expected 'invalid response' stderr, got %q", stderr.String())
	}
}
