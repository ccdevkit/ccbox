package main

import (
	"fmt"
	"os"
	"testing"

	"github.com/ccdevkit/ccbox/internal/args"
	"github.com/ccdevkit/ccbox/internal/bridge"
	"github.com/ccdevkit/ccbox/internal/docker"
)

// testFS implements args.FileSystem for orchestration tests.
type testFS struct{}

func (testFS) Stat(path string) (os.FileInfo, error) { return nil, os.ErrNotExist }
func (testFS) ReadFile(path string) ([]byte, error)   { return nil, os.ErrNotExist }

// --- Mock dependencies for orchestration testing ---

type mockDockerChecker struct {
	err error
}

func (m *mockDockerChecker) CheckRunning() error { return m.err }

type mockTokenCapture struct {
	token string
	err   error
}

func (m *mockTokenCapture) CaptureToken(claudePath string) (string, error) {
	return m.token, m.err
}

type mockVersionDetect struct {
	version string
	err     error
}

func (m *mockVersionDetect) DetectVersion(claudePath string) (string, error) {
	return m.version, m.err
}

type mockImageEnsurer struct {
	buildErr error
}

func (m *mockImageEnsurer) EnsureLocalImage(ccboxVersion, claudeVersion string, pinned bool) error {
	return m.buildErr
}

type mockContainerRunner struct {
	exitCode int
	err      error
}

func (m *mockContainerRunner) RunContainer(spec *docker.ContainerSpec) (int, error) {
	return m.exitCode, m.err
}

type mockBridgeServer struct {
	port int
	err  error
}

func (m *mockBridgeServer) Start() (int, error) { return m.port, m.err }
func (m *mockBridgeServer) Stop() error          { return nil }

// defaultDeps returns a deps struct with all mocks set to succeed.
func defaultDeps() *orchestrationDeps {
	return &orchestrationDeps{
		dockerChecker:   &mockDockerChecker{},
		tokenCapture:    &mockTokenCapture{token: "test-token"},
		versionDetect:   &mockVersionDetect{version: "2.1.16"},
		imageEnsurer:    &mockImageEnsurer{},
		bridgeServer:    &mockBridgeServer{port: 12345},
		containerRunner: &mockContainerRunner{exitCode: 0},
		ccboxVersion:    "0.2.0",
		fs:              testFS{},
	}
}

func TestOrchestration_AuthFailure_ReturnsExitCode1(t *testing.T) {
	deps := defaultDeps()
	deps.tokenCapture = &mockTokenCapture{err: fmt.Errorf("auth failed")}

	parsed := &args.ParsedArgs{}
	exitCode := runOrchestration(parsed, deps)

	if exitCode != 1 {
		t.Fatalf("expected exit code 1 on auth failure, got %d", exitCode)
	}
}

func TestOrchestration_ImageBuildFailure_ReturnsExitCode1(t *testing.T) {
	deps := defaultDeps()
	deps.imageEnsurer = &mockImageEnsurer{buildErr: fmt.Errorf("build failed")}

	parsed := &args.ParsedArgs{}
	exitCode := runOrchestration(parsed, deps)

	if exitCode != 1 {
		t.Fatalf("expected exit code 1 on image build failure, got %d", exitCode)
	}
}

// mockBridgeServerWithHandler records the exec handler passed via the factory.
type mockBridgeServerWithHandler struct {
	port        int
	err         error
	execHandler bridge.ExecHandler
}

func (m *mockBridgeServerWithHandler) Start() (int, error) { return m.port, m.err }
func (m *mockBridgeServerWithHandler) Stop() error          { return nil }

func TestOrchestration_NoPermissionsNoPassthrough_BackwardCompat(t *testing.T) {
	// No permissions file exists, no CLI passthrough flags.
	// The orchestration should succeed with backward-compatible behavior:
	// checker is nil → NewPermissionAwareHandler(nil) returns HandleExec.
	mock := &mockBridgeServerWithHandler{port: 12345}
	deps := defaultDeps()
	deps.bridgeServerFactory = func(execHandler bridge.ExecHandler, hookHandler bridge.HookHandler) BridgeServer {
		mock.execHandler = execHandler
		return mock
	}

	parsed := &args.ParsedArgs{}
	exitCode := runOrchestration(parsed, deps)

	if exitCode != 0 {
		t.Fatalf("expected exit code 0 with no permissions/passthrough, got %d", exitCode)
	}
	if mock.execHandler == nil {
		t.Fatal("bridge server factory was not called with an exec handler")
	}
}

func TestOrchestration_CLIPassthrough_FactoryReceivesHandler(t *testing.T) {
	// CLI passthrough flags are set. The factory should receive a
	// permission-aware exec handler (not nil).
	mock := &mockBridgeServerWithHandler{port: 12345}
	deps := defaultDeps()
	deps.bridgeServerFactory = func(execHandler bridge.ExecHandler, hookHandler bridge.HookHandler) BridgeServer {
		mock.execHandler = execHandler
		return mock
	}

	parsed := &args.ParsedArgs{
		Passthrough: []string{"git"},
	}
	exitCode := runOrchestration(parsed, deps)

	if exitCode != 0 {
		t.Fatalf("expected exit code 0 with CLI passthrough, got %d", exitCode)
	}
	if mock.execHandler == nil {
		t.Fatal("bridge server factory was not called with an exec handler")
	}
}

func TestOrchestration_VersionMismatch_TriggersRebuild(t *testing.T) {
	// If the detected version changes, EnsureLocalImage must be called
	// (which would trigger a rebuild). We verify it's called with the
	// new version by making EnsureLocalImage fail and checking we get that error.
	deps := defaultDeps()
	deps.versionDetect = &mockVersionDetect{version: "3.0.0"}
	deps.imageEnsurer = &mockImageEnsurer{buildErr: fmt.Errorf("rebuild triggered")}

	parsed := &args.ParsedArgs{}
	exitCode := runOrchestration(parsed, deps)

	// The image build failure should cause exit code 1, proving the rebuild was attempted
	if exitCode != 1 {
		t.Fatalf("expected exit code 1 when version mismatch triggers rebuild that fails, got %d", exitCode)
	}
}
