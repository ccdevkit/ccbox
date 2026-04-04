package main

import (
	"encoding/json"
	"net"
	"time"

	"github.com/ccdevkit/ccbox/internal/bridge"
)

// SendLog connects to the host TCP server and sends a fire-and-forget LogRequest.
// It uses a 2-second dial timeout and closes the connection after sending.
func SendLog(address string, message string) error {
	conn, err := net.DialTimeout("tcp", address, 2*time.Second)
	if err != nil {
		return err
	}
	defer conn.Close()

	req := bridge.LogRequest{
		Type:    "log",
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
