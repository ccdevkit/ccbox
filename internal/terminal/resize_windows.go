//go:build windows

package terminal

import (
	"context"
	"os"
	"time"

	"github.com/ccdevkit/ccbox/internal/docker"
	"golang.org/x/term"
)

// WatchResize polls terminal dimensions every 250ms and calls pty.Resize when
// they change. Windows lacks SIGWINCH, so polling is the only option.
//
// TDD exemption: OS-dependent polling loop with no injectable seams;
// tested via integration tests, not unit tests (Principle VII amendment).
func WatchResize(ctx context.Context, pty docker.PTY) error {
	fd := int(os.Stdout.Fd())

	cols, rows, err := term.GetSize(fd)
	if err != nil {
		return err
	}

	ticker := time.NewTicker(250 * time.Millisecond)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-ticker.C:
			newCols, newRows, err := term.GetSize(fd)
			if err != nil {
				continue
			}
			if newCols != cols || newRows != rows {
				cols, rows = newCols, newRows
				_ = pty.Resize(uint16(rows), uint16(cols))
			}
		}
	}
}
