//go:build windows

package terminal

import (
	"os/exec"

	gopty "github.com/aymanbagabas/go-pty"
)

// WindowsPTY implements docker.PTY using go-pty (ConPTY) for Windows.
//
// TDD exemption: thin platform wrapper delegating to go-pty;
// tested via integration tests, not unit tests (Principle VII amendment).
type WindowsPTY struct {
	pty gopty.Pty
}

// Start launches the command attached to a new Windows pseudo-console.
func (p *WindowsPTY) Start(cmd *exec.Cmd) error {
	pp, err := gopty.New()
	if err != nil {
		return err
	}
	p.pty = pp

	c := pp.Command(cmd.Path, cmd.Args[1:]...)
	c.Env = cmd.Env
	c.Dir = cmd.Dir
	return c.Start()
}

// Read reads from the PTY output pipe.
func (p *WindowsPTY) Read(buf []byte) (int, error) {
	return p.pty.Read(buf)
}

// Write writes to the PTY input pipe.
func (p *WindowsPTY) Write(buf []byte) (int, error) {
	return p.pty.Write(buf)
}

// Resize sets the PTY window size.
func (p *WindowsPTY) Resize(rows, cols uint16) error {
	return p.pty.Resize(int(cols), int(rows))
}

// Close closes the PTY and its associated pipes.
func (p *WindowsPTY) Close() error {
	return p.pty.Close()
}
