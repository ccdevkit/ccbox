package session

import (
	"fmt"
	"testing"

	"github.com/google/uuid"
)

// mockFileWriter implements SessionFileWriter for testing.
type mockFileWriter struct {
	written map[string][]byte
}

func (m *mockFileWriter) WriteFile(containerPath string, data []byte, readOnly bool) error {
	m.written[containerPath] = data
	return nil
}

// mockPassthroughProvider implements FilePassthroughProvider for testing.
type mockPassthroughProvider struct {
	calls []passthroughCall
}

type passthroughCall struct {
	hostPath      string
	containerPath string
	readOnly      bool
}

func (m *mockPassthroughProvider) AddPassthrough(hostPath, containerPath string, readOnly bool) error {
	m.calls = append(m.calls, passthroughCall{hostPath, containerPath, readOnly})
	return nil
}

// errorPassthroughProvider returns an error from AddPassthrough.
type errorPassthroughProvider struct{}

func (e *errorPassthroughProvider) AddPassthrough(_, _ string, _ bool) error {
	return fmt.Errorf("passthrough error")
}

func TestNewSession_ValidUUID(t *testing.T) {
	fw := &mockFileWriter{written: make(map[string][]byte)}
	fp := &mockPassthroughProvider{}

	s := NewSession(fw, fp)

	if s == nil {
		t.Fatal("NewSession returned nil")
	}

	// ID must be a valid UUID v4.
	parsed, err := uuid.Parse(s.ID)
	if err != nil {
		t.Fatalf("ID is not a valid UUID: %v", err)
	}
	if parsed.Version() != 4 {
		t.Fatalf("expected UUID v4, got v%d", parsed.Version())
	}
}

func TestNewSession_StoresInjectedDependencies(t *testing.T) {
	fw := &mockFileWriter{written: make(map[string][]byte)}
	fp := &mockPassthroughProvider{}

	s := NewSession(fw, fp)

	if s.FileWriter != fw {
		t.Error("FileWriter not stored correctly")
	}
	if s.FilePassthrough != fp {
		t.Error("FilePassthrough not stored correctly")
	}
}

func TestAddFilePassthrough_DelegatesToProvider(t *testing.T) {
	fw := &mockFileWriter{written: make(map[string][]byte)}
	fp := &mockPassthroughProvider{}
	s := NewSession(fw, fp)

	err := s.AddFilePassthrough("/host/path", "/container/path", true)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if len(fp.calls) != 1 {
		t.Fatalf("expected 1 call, got %d", len(fp.calls))
	}
	c := fp.calls[0]
	if c.hostPath != "/host/path" {
		t.Errorf("hostPath = %q, want %q", c.hostPath, "/host/path")
	}
	if c.containerPath != "/container/path" {
		t.Errorf("containerPath = %q, want %q", c.containerPath, "/container/path")
	}
	if !c.readOnly {
		t.Error("readOnly = false, want true")
	}
}

func TestAddFilePassthrough_PropagatesError(t *testing.T) {
	fw := &mockFileWriter{written: make(map[string][]byte)}
	ep := &errorPassthroughProvider{}
	s := NewSession(fw, ep)

	err := s.AddFilePassthrough("/a", "/b", false)
	if err == nil {
		t.Fatal("expected error, got nil")
	}
}
