package bridge

import (
	"bufio"
	"encoding/json"
	"fmt"
	"net"
	"sync"

	"github.com/ccdevkit/ccbox/internal/constants"
	"github.com/ccdevkit/ccbox/internal/logger"
)

// ExecHandler processes an exec request and returns an exit code and output.
type ExecHandler func(req ExecRequest) (exitCode int, output []byte)

// LogHandler processes a log request (fire-and-forget).
type LogHandler func(req LogRequest)

// NewLogHandler creates a LogHandler that forwards container log messages
// to the given logger with a "container" prefix.
func NewLogHandler(log *logger.Logger) LogHandler {
	return func(req LogRequest) {
		log.Debug("container", "%s", req.Message)
	}
}

// Server is a TCP server that accepts JSON messages from a container
// and dispatches them to exec or log handlers.
type Server struct {
	execHandler ExecHandler
	logHandler  LogHandler
	listener    net.Listener
	wg          sync.WaitGroup
	done        chan struct{}
}

// NewServer creates a new Server with the given handlers.
func NewServer(execHandler ExecHandler, logHandler LogHandler) *Server {
	return &Server{
		execHandler: execHandler,
		logHandler:  logHandler,
		done:        make(chan struct{}),
	}
}

// Start binds the server to 127.0.0.1:0 and begins accepting connections.
// It returns the OS-assigned port.
func (s *Server) Start() (int, error) {
	ln, err := net.Listen("tcp", constants.TCPServerBindAddress)
	if err != nil {
		return 0, fmt.Errorf("bridge server listen: %w", err)
	}
	s.listener = ln

	port := ln.Addr().(*net.TCPAddr).Port

	s.wg.Add(1)
	go s.acceptLoop()

	return port, nil
}

// Stop closes the listener and waits for all connections to finish.
func (s *Server) Stop() error {
	close(s.done)
	err := s.listener.Close()
	s.wg.Wait()
	return err
}

func (s *Server) acceptLoop() {
	defer s.wg.Done()
	for {
		conn, err := s.listener.Accept()
		if err != nil {
			select {
			case <-s.done:
				return
			default:
				continue
			}
		}
		s.wg.Add(1)
		go s.handleConn(conn)
	}
}

// typeEnvelope is used to peek at the "type" field of incoming JSON.
type typeEnvelope struct {
	Type string `json:"type"`
}

func (s *Server) handleConn(conn net.Conn) {
	defer s.wg.Done()
	defer conn.Close()

	scanner := bufio.NewScanner(conn)
	if !scanner.Scan() {
		return
	}
	line := scanner.Bytes()

	var envelope typeEnvelope
	if err := json.Unmarshal(line, &envelope); err != nil {
		// Malformed JSON: silently drop.
		return
	}

	switch envelope.Type {
	case constants.ExecRequestType:
		var req ExecRequest
		if err := json.Unmarshal(line, &req); err != nil {
			return
		}
		exitCode, output := s.execHandler(req)
		fmt.Fprintf(conn, "%d\n%s", exitCode, output)

	case constants.LogRequestType:
		var req LogRequest
		if err := json.Unmarshal(line, &req); err != nil {
			return
		}
		s.logHandler(req)

	default:
		// Unknown type: silently drop.
	}
}
