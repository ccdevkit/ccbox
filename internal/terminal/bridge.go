package terminal

import (
	"fmt"
	"os"
	"path/filepath"
)

// FileBridge copies host files to a bridge directory and returns
// container-side paths. It implements the FileBridger interface.
type FileBridge struct {
	HostDir      string // Host-side bridge directory (e.g., ~/.ccbox-bridge).
	ContainerDir string // Container-side mount point (e.g., /home/claude/.ccbox-bridge).
}

// CopyFileToBridge copies srcPath into the bridge directory and returns the
// corresponding container-side path.
func (b *FileBridge) CopyFileToBridge(srcPath string) (string, error) {
	data, err := os.ReadFile(srcPath)
	if err != nil {
		return "", fmt.Errorf("reading source file: %w", err)
	}

	filename := filepath.Base(srcPath)
	destPath := filepath.Join(b.HostDir, filename)

	if err := os.WriteFile(destPath, data, 0644); err != nil {
		return "", fmt.Errorf("writing to bridge directory: %w", err)
	}

	return filepath.Join(b.ContainerDir, filename), nil
}
