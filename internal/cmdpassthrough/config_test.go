package cmdpassthrough

import (
	"encoding/json"
	"testing"

	"github.com/ccdevkit/ccbox/internal/constants"
	"github.com/ccdevkit/ccbox/internal/session"
)

// mockFileWriter captures calls to WriteFile for test assertions.
type mockFileWriter struct {
	path string
	data []byte
}

func (m *mockFileWriter) WriteFile(containerPath string, data []byte, readOnly bool) error {
	m.path = containerPath
	m.data = data
	return nil
}

// mockFilePassthrough is a no-op FilePassthroughProvider for tests.
type mockFilePassthrough struct{}

func (m *mockFilePassthrough) AddPassthrough(hostPath, containerPath string, readOnly bool) error {
	return nil
}

func TestWriteProxyConfig_WritesCorrectJSON(t *testing.T) {
	fw := &mockFileWriter{}
	sess := session.NewSession(fw, &mockFilePassthrough{})

	config := ProxyConfig{
		HostAddress: "host.docker.internal:9876",
		Passthrough: []string{"git", "gh", "npm"},
		Verbose:     false,
	}

	if err := WriteProxyConfig(sess, config); err != nil {
		t.Fatalf("WriteProxyConfig returned error: %v", err)
	}

	// Verify it wrote to the correct container path.
	if fw.path != constants.ProxyConfigContainerPath {
		t.Errorf("wrote to %q, want %q", fw.path, constants.ProxyConfigContainerPath)
	}

	// Verify the JSON content round-trips correctly.
	var got ProxyConfig
	if err := json.Unmarshal(fw.data, &got); err != nil {
		t.Fatalf("failed to unmarshal written data: %v", err)
	}

	if got.HostAddress != config.HostAddress {
		t.Errorf("hostAddress = %q, want %q", got.HostAddress, config.HostAddress)
	}
	if len(got.Passthrough) != len(config.Passthrough) {
		t.Fatalf("passthrough length = %d, want %d", len(got.Passthrough), len(config.Passthrough))
	}
	for i, cmd := range config.Passthrough {
		if got.Passthrough[i] != cmd {
			t.Errorf("passthrough[%d] = %q, want %q", i, got.Passthrough[i], cmd)
		}
	}
	if got.Verbose != config.Verbose {
		t.Errorf("verbose = %v, want %v", got.Verbose, config.Verbose)
	}
}

func TestWriteProxyConfig_VerboseTrue(t *testing.T) {
	fw := &mockFileWriter{}
	sess := session.NewSession(fw, &mockFilePassthrough{})

	config := ProxyConfig{
		HostAddress: "localhost:1234",
		Passthrough: []string{"docker"},
		Verbose:     true,
	}

	if err := WriteProxyConfig(sess, config); err != nil {
		t.Fatalf("WriteProxyConfig returned error: %v", err)
	}

	var got ProxyConfig
	if err := json.Unmarshal(fw.data, &got); err != nil {
		t.Fatalf("failed to unmarshal written data: %v", err)
	}

	if !got.Verbose {
		t.Error("verbose = false, want true")
	}
}

func TestWriteProxyConfig_EmptyPassthrough(t *testing.T) {
	fw := &mockFileWriter{}
	sess := session.NewSession(fw, &mockFilePassthrough{})

	config := ProxyConfig{
		HostAddress: "host.docker.internal:5555",
		Passthrough: []string{},
	}

	if err := WriteProxyConfig(sess, config); err != nil {
		t.Fatalf("WriteProxyConfig returned error: %v", err)
	}

	var got ProxyConfig
	if err := json.Unmarshal(fw.data, &got); err != nil {
		t.Fatalf("failed to unmarshal written data: %v", err)
	}

	if len(got.Passthrough) != 0 {
		t.Errorf("passthrough length = %d, want 0", len(got.Passthrough))
	}
}
