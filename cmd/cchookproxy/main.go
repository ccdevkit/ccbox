package main

import (
	"encoding/json"
	"fmt"
	"io"
	"net"
	"os"
	"time"

	"github.com/ccdevkit/ccbox/internal/bridge"
	"github.com/ccdevkit/ccbox/internal/constants"
)

var (
	dialTimeout     = time.Duration(constants.HookDialTimeoutSec) * time.Second
	responseTimeout = time.Duration(constants.HookResponseTimeoutSec) * time.Second
)

func main() {
	os.Exit(run(os.Stdin, os.Stdout, os.Stderr))
}

// debugLog sends a log message to the host bridge via a separate TCP connection.
// Failures are silently ignored — debug logging must never break hook dispatch.
func debugLog(format string, args ...interface{}) {
	port := os.Getenv(constants.EnvCCBoxTCPPort)
	if port == "" {
		return
	}
	host := os.Getenv("DOCKER_HOSTNAME")
	if host == "" {
		host = constants.DockerHostname
	}
	address := net.JoinHostPort(host, port)

	conn, err := net.DialTimeout("tcp", address, 2*time.Second)
	if err != nil {
		return
	}
	defer conn.Close()

	msg := fmt.Sprintf(format, args...)
	req := bridge.LogRequest{
		Type:    constants.LogRequestType,
		Message: msg,
		Prefix:  "cchookproxy",
	}
	data, err := json.Marshal(req)
	if err != nil {
		return
	}
	data = append(data, '\n')
	conn.Write(data)
}

// run reads hook input from stdin, forwards to host via TCP, returns exit code.
func run(stdin io.Reader, stdout, stderr io.Writer) int {
	debugLog("started, pid=%d", os.Getpid())

	// 1. Read all stdin
	input, err := io.ReadAll(stdin)
	if err != nil {
		fmt.Fprintf(stderr, "cchookproxy: failed to read stdin: %v\n", err)
		return 1
	}
	debugLog("read %d bytes from stdin", len(input))

	// 2. Parse hook_event_name from the JSON
	var envelope struct {
		HookEventName string `json:"hook_event_name"`
		ToolName      string `json:"tool_name"`
	}
	if err := json.Unmarshal(input, &envelope); err != nil {
		fmt.Fprintf(stderr, "cchookproxy: invalid input JSON: %v\n", err)
		debugLog("invalid input JSON: %v", err)
		return 1
	}
	debugLog("event=%s tool=%s", envelope.HookEventName, envelope.ToolName)

	// 3. Dial TCP
	host := os.Getenv("DOCKER_HOSTNAME")
	if host == "" {
		host = constants.DockerHostname
	}
	port := os.Getenv(constants.EnvCCBoxTCPPort)
	if port == "" {
		fmt.Fprintf(stderr, "cchookproxy: %s not set\n", constants.EnvCCBoxTCPPort)
		debugLog("CCBOX_TCP_PORT not set")
		return 1
	}
	address := net.JoinHostPort(host, port)
	debugLog("dialing %s", address)

	conn, err := net.DialTimeout("tcp", address, dialTimeout)
	if err != nil {
		fmt.Fprintf(stderr, "cchookproxy: connection failed: %v\n", err)
		debugLog("connection failed: %v", err)
		return 1
	}
	defer conn.Close()
	debugLog("connected to bridge")

	// 4. Send HookRequest as newline-delimited JSON
	req := bridge.HookRequest{
		Type:  constants.HookRequestType,
		Event: envelope.HookEventName,
		Input: json.RawMessage(input),
	}
	reqBytes, err := json.Marshal(req)
	if err != nil {
		fmt.Fprintf(stderr, "cchookproxy: failed to marshal request: %v\n", err)
		return 1
	}
	reqBytes = append(reqBytes, '\n')
	if _, err := conn.Write(reqBytes); err != nil {
		fmt.Fprintf(stderr, "cchookproxy: failed to send request: %v\n", err)
		debugLog("failed to send request: %v", err)
		return 1
	}
	debugLog("sent %d byte request", len(reqBytes))

	// 5. Close write side of connection
	if tc, ok := conn.(*net.TCPConn); ok {
		tc.CloseWrite()
	}

	// 6. Read response with timeout
	conn.SetReadDeadline(time.Now().Add(responseTimeout))
	respBytes, err := io.ReadAll(conn)
	if err != nil {
		if netErr, ok := err.(net.Error); ok && netErr.Timeout() {
			fmt.Fprintf(stderr, "cchookproxy: response timeout\n")
			debugLog("response timeout")
			return 1
		}
		fmt.Fprintf(stderr, "cchookproxy: failed to read response: %v\n", err)
		debugLog("failed to read response: %v", err)
		return 1
	}
	debugLog("received %d byte response", len(respBytes))

	// 7. Parse HookResponse
	var resp bridge.HookResponse
	if err := json.Unmarshal(respBytes, &resp); err != nil {
		fmt.Fprintf(stderr, "cchookproxy: invalid response\n")
		debugLog("invalid response JSON: %v (raw: %s)", err, string(respBytes))
		return 1
	}
	debugLog("response: exit=%d stdout=%d bytes stderr=%d bytes", resp.ExitCode, len(resp.Stdout), len(resp.Stderr))

	// 8. Write stdout portion
	if resp.Stdout != "" {
		fmt.Fprint(stdout, resp.Stdout)
	}

	// 9. Write stderr portion
	if resp.Stderr != "" {
		fmt.Fprint(stderr, resp.Stderr)
	}

	// 10. Return exit code
	debugLog("exiting with code %d", resp.ExitCode)
	return resp.ExitCode
}
