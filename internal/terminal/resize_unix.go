//go:build !windows

package terminal

import (
	"context"
	"os"
	"os/signal"
	"syscall"

	"github.com/ccdevkit/ccbox/internal/docker"
	"golang.org/x/term"
)

// WatchResize listens for SIGWINCH signals and resizes the PTY to match the
// current terminal dimensions. It blocks until ctx is cancelled.
//
// TDD exemption: OS signal handler with no injectable seam; behaviour is
// verified via manual and integration testing (Principle VII amendment).
func WatchResize(ctx context.Context, pty docker.PTY) error {
	sigCh := make(chan os.Signal, 1)
	signal.Notify(sigCh, syscall.SIGWINCH)
	defer signal.Stop(sigCh)

	for {
		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-sigCh:
			cols, rows, err := term.GetSize(int(os.Stdout.Fd()))
			if err != nil {
				continue
			}
			_ = pty.Resize(uint16(rows), uint16(cols))
		}
	}
}
