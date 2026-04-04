package claude

import (
	"fmt"
	"net/http"
	"strings"
	"testing"
)

// mockProcess implements CaptureProcess for tests.
type mockProcess struct {
	waitErr error
	killed  bool
	waitCh  chan struct{} // if set, Wait blocks until closed
}

func (p *mockProcess) Kill() error {
	p.killed = true
	if p.waitCh != nil {
		select {
		case <-p.waitCh:
		default:
			close(p.waitCh)
		}
	}
	return nil
}

func (p *mockProcess) Wait() error {
	if p.waitCh != nil {
		<-p.waitCh
	}
	return p.waitErr
}

// mockRunner starts a mock process that makes an HTTP request with an Authorization header.
type mockRunner struct {
	token string
}

func (m *mockRunner) Start(name string, args []string, env []string) (CaptureProcess, error) {
	// Extract the base URL from the env vars.
	var baseURL string
	for _, e := range env {
		if strings.HasPrefix(e, "ANTHROPIC_BASE_URL=") {
			baseURL = strings.TrimPrefix(e, "ANTHROPIC_BASE_URL=")
			break
		}
	}
	if baseURL == "" {
		return nil, fmt.Errorf("ANTHROPIC_BASE_URL not set")
	}

	// Process blocks until killed (like real claude would).
	proc := &mockProcess{waitCh: make(chan struct{})}

	// Simulate claude making an API request with an auth header.
	go func() {
		req, _ := http.NewRequest("POST", baseURL+"/v1/messages", nil)
		req.Header.Set("Authorization", "Bearer "+m.token)
		client := &http.Client{}
		resp, err := client.Do(req)
		if err == nil {
			resp.Body.Close()
		}
	}()

	return proc, nil
}

func TestCaptureToken_Success(t *testing.T) {
	expectedToken := "sk-ant-test-token-abc123"
	runner := &mockRunner{token: expectedToken}

	token, err := CaptureToken("/usr/bin/claude", runner)
	if err != nil {
		t.Fatalf("CaptureToken returned error: %v", err)
	}
	if token != expectedToken {
		t.Errorf("got token %q, want %q", token, expectedToken)
	}
}

// failRunner returns a start error.
type failRunner struct{}

func (f *failRunner) Start(name string, args []string, env []string) (CaptureProcess, error) {
	return nil, fmt.Errorf("command failed: exit status 1")
}

func TestCaptureToken_RunnerError(t *testing.T) {
	runner := &failRunner{}

	_, err := CaptureToken("/usr/bin/claude", runner)
	if err == nil {
		t.Fatal("expected error, got nil")
	}
}

// exitRunner starts a process that exits immediately without making a request.
type exitRunner struct{}

func (e *exitRunner) Start(name string, args []string, env []string) (CaptureProcess, error) {
	return &mockProcess{}, nil
}

func TestCaptureToken_ProcessExitedWithoutRequest(t *testing.T) {
	runner := &exitRunner{}

	_, err := CaptureToken("/usr/bin/claude", runner)
	if err == nil {
		t.Fatal("expected error, got nil")
	}
	if got := err.Error(); got != "claude process exited without sending a request" {
		t.Errorf("unexpected error: %s", got)
	}
}
