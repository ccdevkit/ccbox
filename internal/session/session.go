package session

import "github.com/google/uuid"

// SessionFileWriter writes files that will be available inside the container.
// Files written through this interface don't exist on the host beforehand.
type SessionFileWriter interface {
	WriteFile(containerPath string, data []byte, readOnly bool) error
}

// FilePassthroughProvider registers host files/directories to be made
// available inside the container.
type FilePassthroughProvider interface {
	AddPassthrough(hostPath, containerPath string, readOnly bool) error
}

// Session holds ephemeral per-run state.
type Session struct {
	ID              string
	FileWriter      SessionFileWriter
	FilePassthrough FilePassthroughProvider
}

// NewSession creates a session with a new UUID v4 and the given providers.
func NewSession(fw SessionFileWriter, fp FilePassthroughProvider) *Session {
	return &Session{
		ID:              uuid.New().String(),
		FileWriter:      fw,
		FilePassthrough: fp,
	}
}

// AddFilePassthrough delegates to the session's FilePassthroughProvider.
func (s *Session) AddFilePassthrough(hostPath, containerPath string, readOnly bool) error {
	return s.FilePassthrough.AddPassthrough(hostPath, containerPath, readOnly)
}
