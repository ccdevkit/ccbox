//go:build !windows

package docker

import (
	"os"
	"os/signal"
	"syscall"

	"github.com/creack/pty"
)

// handleResize listens for SIGWINCH and resizes the PTY to match the host
// terminal. Returns a cleanup function that stops listening.
func handleResize(p PTY) func() {
	up, ok := p.(*unixPTY)
	if !ok {
		return func() {}
	}

	ch := make(chan os.Signal, 1)
	signal.Notify(ch, syscall.SIGWINCH)
	go func() {
		for range ch {
			_ = pty.InheritSize(os.Stdin, up.File())
		}
	}()
	ch <- syscall.SIGWINCH // Initial resize
	return func() { signal.Stop(ch); close(ch) }
}
