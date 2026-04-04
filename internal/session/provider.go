package session

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/ccdevkit/ccbox/internal/constants"
)

// SessionFile represents a file written by TempDirProvider, mapping a host path
// to where it should appear inside the container.
type SessionFile struct {
	HostPath      string // Absolute path on the host
	ContainerPath string // Where the file should appear in the container
	ReadOnly      bool   // Whether the mount should be read-only
}

// TempDirProvider implements SessionFileWriter by writing files to a host temp
// directory. The orchestrator reads Files after session setup to generate
// container bind mounts.
type TempDirProvider struct {
	Dir   string        // /tmp/ccbox-{sessionID}/
	Files []SessionFile // Populated by WriteFile
}

// NewTempDirProvider creates a temp directory for the given session and returns
// a provider ready to write files into it.
func NewTempDirProvider(sessionID string) (*TempDirProvider, error) {
	dir, err := os.MkdirTemp("", fmt.Sprintf("%s%s-", constants.TempDirPrefix, sessionID))
	if err != nil {
		return nil, fmt.Errorf("create temp dir: %w", err)
	}
	return &TempDirProvider{
		Dir:   dir,
		Files: []SessionFile{},
	}, nil
}

// WriteFile slugifies containerPath to derive a host filename, writes data to
// {Dir}/{slug}, and appends to Files. If readOnly is true, the file will be
// mounted read-only in the container.
func (p *TempDirProvider) WriteFile(containerPath string, data []byte, readOnly bool) error {
	slug := slugify(containerPath)
	hostPath := filepath.Join(p.Dir, slug)

	if err := os.WriteFile(hostPath, data, 0o600); err != nil {
		return fmt.Errorf("write session file %s: %w", slug, err)
	}

	p.Files = append(p.Files, SessionFile{
		HostPath:      hostPath,
		ContainerPath: containerPath,
		ReadOnly:      readOnly,
	})
	return nil
}

// Cleanup removes the temp directory and all files in it.
func (p *TempDirProvider) Cleanup() error {
	return os.RemoveAll(p.Dir)
}

// slugify converts an absolute container path to a flat filename by replacing
// path separators with underscores. E.g. "/opt/ccbox/settings.json" → "_opt_ccbox_settings.json".
func slugify(path string) string {
	return strings.ReplaceAll(path, "/", "_")
}
