package claude

import (
	"fmt"
	"net"
	"net/http"
	"strings"
	"time"
)

// DefaultCaptureTimeout is the maximum time to wait for the token capture.
const DefaultCaptureTimeout = 30 * time.Second

// CaptureProcess represents a running process that can be killed and waited on.
type CaptureProcess interface {
	Kill() error
	Wait() error
}

// CommandRunner allows injecting test doubles for command execution.
type CommandRunner interface {
	Start(name string, args []string, env []string) (CaptureProcess, error)
}

// RequestLogger is an optional callback invoked for each HTTP request received
// during token capture, useful for debugging auth flow issues.
type RequestLogger func(method, path, authHeader string)

// CaptureToken starts an ephemeral HTTP server on localhost, runs the claude CLI
// with ANTHROPIC_BASE_URL pointing to it, and captures the OAuth token from the
// Authorization header of the first request. The claude process is killed
// immediately once the token is captured.
// An optional RequestLogger can be provided via CaptureTokenWithLogger.
func CaptureToken(claudePath string, runner CommandRunner) (string, error) {
	return captureToken(claudePath, runner, nil)
}

// CaptureTokenWithLogger is like CaptureToken but logs each incoming request.
func CaptureTokenWithLogger(claudePath string, runner CommandRunner, reqLog RequestLogger) (string, error) {
	return captureToken(claudePath, runner, reqLog)
}

func captureToken(claudePath string, runner CommandRunner, reqLog RequestLogger) (string, error) {
	listener, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		return "", fmt.Errorf("failed to start listener: %w", err)
	}
	defer listener.Close()

	port := listener.Addr().(*net.TCPAddr).Port
	baseURL := fmt.Sprintf("http://127.0.0.1:%d", port)

	tokenCh := make(chan string, 1)

	server := &http.Server{
		Handler: http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			auth := r.Header.Get("Authorization")
			if reqLog != nil {
				masked := auth
				if strings.HasPrefix(auth, "Bearer ") && len(auth) > len("Bearer ")+6 {
					t := auth[len("Bearer "):]
					masked = "Bearer " + t[:3] + "***" + t[len(t)-3:]
				}
				reqLog(r.Method, r.URL.Path, masked)
			}
			if !strings.HasPrefix(auth, "Bearer ") || len(auth) <= len("Bearer ") {
				// Respond 200 to health checks so Claude proceeds to API calls.
				w.WriteHeader(http.StatusOK)
				return
			}
			token := strings.TrimPrefix(auth, "Bearer ")
			select {
			case tokenCh <- token:
			default:
			}
		}),
	}
	defer server.Close()

	go server.Serve(listener)

	env := []string{"ANTHROPIC_BASE_URL=" + baseURL}
	args := []string{"-p", "hello", "--setting-sources", ""}
	proc, err := runner.Start(claudePath, args, env)
	if err != nil {
		return "", fmt.Errorf("failed to start claude: %w", err)
	}

	// Wait for either: token captured, process exit, or timeout.
	errCh := make(chan error, 1)
	go func() {
		errCh <- proc.Wait()
	}()

	select {
	case token := <-tokenCh:
		proc.Kill()
		proc.Wait()
		return token, nil
	case err := <-errCh:
		if err != nil {
			return "", fmt.Errorf("claude process failed: %w", err)
		}
		return "", fmt.Errorf("claude process exited without sending a request")
	case <-time.After(DefaultCaptureTimeout):
		proc.Kill()
		proc.Wait()
		return "", fmt.Errorf("token capture timed out after %v", DefaultCaptureTimeout)
	}
}

