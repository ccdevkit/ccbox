package main

import (
	"bufio"
	"encoding/json"
	"fmt"
	"net"
	"strconv"
	"strings"
	"time"

	"github.com/ccdevkit/ccbox/internal/bridge"
)

// SendExec connects to the host TCP server at address, sends an ExecRequest,
// and reads back the exit code and combined output.
func SendExec(address string, command string, cwd string) (exitCode int, output []byte, err error) {
	conn, err := net.DialTimeout("tcp", address, 2*time.Second)
	if err != nil {
		return 0, nil, err
	}
	defer conn.Close()

	req := bridge.ExecRequest{
		Type:    "exec",
		Command: command,
		Cwd:     cwd,
	}

	data, err := json.Marshal(req)
	if err != nil {
		return 0, nil, err
	}

	data = append(data, '\n')
	if _, err := conn.Write(data); err != nil {
		return 0, nil, err
	}

	// Signal end-of-request by closing the write side
	tcpConn, ok := conn.(*net.TCPConn)
	if !ok {
		return 0, nil, fmt.Errorf("connection is not TCP")
	}
	if err := tcpConn.CloseWrite(); err != nil {
		return 0, nil, err
	}

	// Read response: first line is exit code, rest is output bytes
	reader := bufio.NewReader(conn)
	exitLine, err := reader.ReadString('\n')
	if err != nil {
		return 0, nil, fmt.Errorf("reading exit code: %w", err)
	}

	exitCode, err = strconv.Atoi(strings.TrimRight(exitLine, "\n"))
	if err != nil {
		return 0, nil, fmt.Errorf("parsing exit code %q: %w", exitLine, err)
	}

	// Read remaining bytes as output
	var out []byte
	buf := make([]byte, 4096)
	for {
		n, readErr := reader.Read(buf)
		if n > 0 {
			out = append(out, buf[:n]...)
		}
		if readErr != nil {
			break
		}
	}

	return exitCode, out, nil
}
