//go:build !windows

package terminal

import (
	"os"
	"os/exec"

	"github.com/creack/pty"
)

// UnixPTY implements docker.PTY using creack/pty for Unix platforms.
//
// TDD exemption: thin platform wrapper delegating to creack/pty;
// tested via integration tests, not unit tests (Principle VII amendment).
type UnixPTY struct {
	f *os.File
}

// Start launches the command attached to a new pseudo-terminal.
func (p *UnixPTY) Start(cmd *exec.Cmd) error {
	f, err := pty.Start(cmd)
	if err != nil {
		return err
	}
	p.f = f
	return nil
}

// Read reads from the PTY file descriptor.
func (p *UnixPTY) Read(buf []byte) (int, error) {
	return p.f.Read(buf)
}

// Write writes to the PTY file descriptor.
func (p *UnixPTY) Write(buf []byte) (int, error) {
	return p.f.Write(buf)
}

// Resize sets the PTY window size.
func (p *UnixPTY) Resize(rows, cols uint16) error {
	return pty.Setsize(p.f, &pty.Winsize{
		Rows: rows,
		Cols: cols,
	})
}

// Close closes the PTY file descriptor.
func (p *UnixPTY) Close() error {
	return p.f.Close()
}
