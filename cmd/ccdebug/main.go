// ccdebug reads debug log lines from stdin and forwards each as a LogRequest
// to the host TCP bridge server. It exits when stdin is closed.
//
// TDD exemption (Principle VII amendment): ccdebug is a thin I/O forwarder with
// no branching logic beyond error handling. Testing would require mocking stdin
// and a TCP server, adding complexity disproportionate to the trivial logic.
package main

import (
	"bufio"
	"encoding/json"
	"fmt"
	"net"
	"os"
	"time"

	"github.com/ccdevkit/ccbox/internal/bridge"
	"github.com/ccdevkit/ccbox/internal/constants"
)

func main() {
	port := os.Getenv(constants.EnvCCBoxTCPPort)
	if port == "" {
		fmt.Fprintf(os.Stderr, "ccdebug: %s not set\n", constants.EnvCCBoxTCPPort)
		os.Exit(1)
	}
	address := constants.DockerHostname + ":" + port

	scanner := bufio.NewScanner(os.Stdin)
	for scanner.Scan() {
		line := scanner.Text()
		if err := sendLog(address, line); err != nil {
			fmt.Fprintf(os.Stderr, "ccdebug: send failed: %v\n", err)
		}
	}

	if err := scanner.Err(); err != nil {
		fmt.Fprintf(os.Stderr, "ccdebug: read error: %v\n", err)
		os.Exit(1)
	}
}

// sendLog connects to the host TCP server and sends a fire-and-forget LogRequest.
func sendLog(address string, message string) error {
	conn, err := net.DialTimeout("tcp", address, time.Duration(constants.LogDialTimeoutSec)*time.Second)
	if err != nil {
		return err
	}
	defer conn.Close()

	req := bridge.LogRequest{
		Type:    constants.LogRequestType,
		Message: message,
	}

	data, err := json.Marshal(req)
	if err != nil {
		return err
	}

	data = append(data, '\n')
	_, err = conn.Write(data)
	return err
}
