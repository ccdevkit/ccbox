// ccclipd is the container clipboard daemon. It listens on TCP for
// length-prefixed PNG data from the host and pipes it to xclip.
//
// TDD exemption (Principle VII): This is a container-only binary that
// depends on xclip and a running X server — cannot be unit tested in
// the host build environment. Integration coverage via T055.
package main

import (
	"bytes"
	"encoding/binary"
	"fmt"
	"io"
	"log"
	"net"
	"os"
	"os/exec"

	"github.com/ccdevkit/ccbox/internal/constants"
)

func main() {
	port := os.Getenv(constants.EnvCCBoxClipPort)
	if port == "" {
		log.Fatalf("ccclipd: %s not set", constants.EnvCCBoxClipPort)
	}

	addr := fmt.Sprintf("0.0.0.0:%s", port)
	ln, err := net.Listen("tcp", addr)
	if err != nil {
		log.Fatalf("ccclipd: listen %s: %v", addr, err)
	}
	log.Printf("ccclipd: listening on %s", addr)

	for {
		conn, err := ln.Accept()
		if err != nil {
			log.Printf("ccclipd: accept: %v", err)
			continue
		}
		go handleConn(conn)
	}
}

func handleConn(conn net.Conn) {
	defer conn.Close()

	// Read 4-byte big-endian length prefix.
	var length uint32
	if err := binary.Read(conn, binary.BigEndian, &length); err != nil {
		log.Printf("ccclipd: read length: %v", err)
		writeStatus(conn, constants.ClipboardStatusError)
		return
	}

	if length == 0 || length > constants.MaxClipboardPayload {
		log.Printf("ccclipd: invalid payload length: %d", length)
		writeStatus(conn, constants.ClipboardStatusError)
		return
	}

	// Read exactly N bytes of PNG data.
	pngData := make([]byte, length)
	if _, err := io.ReadFull(conn, pngData); err != nil {
		log.Printf("ccclipd: read payload: %v", err)
		writeStatus(conn, constants.ClipboardStatusError)
		return
	}

	// Pipe to xclip.
	if err := pipeToXclip(pngData); err != nil {
		log.Printf("ccclipd: xclip: %v", err)
		writeStatus(conn, constants.ClipboardStatusError)
		return
	}

	writeStatus(conn, constants.ClipboardStatusSuccess)
}

func pipeToXclip(data []byte) error {
	cmd := exec.Command("xclip", "-selection", "clipboard", "-target", "image/png", "-i")
	cmd.Stdin = bytes.NewReader(data)
	cmd.Stderr = os.Stderr
	return cmd.Run()
}

func writeStatus(conn net.Conn, status byte) {
	conn.Write([]byte{status}) //nolint:errcheck
}
