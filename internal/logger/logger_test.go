package logger

import (
	"bytes"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestDebug_VerboseWritesToStderr(t *testing.T) {
	var buf bytes.Buffer
	l := &Logger{
		verbose: true,
		out:     &buf,
	}

	l.Debug("docker", "starting container %s", "test-123")

	got := buf.String()
	if !strings.Contains(got, "[docker]") {
		t.Errorf("expected prefix [docker], got %q", got)
	}
	if !strings.Contains(got, "starting container test-123") {
		t.Errorf("expected formatted message, got %q", got)
	}
}

func TestDebug_NotVerboseDiscardsOutput(t *testing.T) {
	var buf bytes.Buffer
	l := &Logger{
		verbose: false,
		out:     &buf,
	}

	l.Debug("docker", "should not appear")

	if buf.Len() != 0 {
		t.Errorf("expected no output, got %q", buf.String())
	}
}

func TestNew_LogFileEnablesVerbose(t *testing.T) {
	dir := t.TempDir()
	logPath := filepath.Join(dir, "debug.log")

	l, err := New(false, logPath)
	if err != nil {
		t.Fatalf("New() error: %v", err)
	}
	defer l.Close()

	if !l.verbose {
		t.Error("expected --log to implicitly enable verbose")
	}
}

func TestNew_LogFileWritesToFile(t *testing.T) {
	dir := t.TempDir()
	logPath := filepath.Join(dir, "debug.log")

	l, err := New(false, logPath)
	if err != nil {
		t.Fatalf("New() error: %v", err)
	}

	l.Debug("session", "created %s", "abc-123")
	l.Close()

	data, err := os.ReadFile(logPath)
	if err != nil {
		t.Fatalf("ReadFile error: %v", err)
	}
	got := string(data)
	if !strings.Contains(got, "[session]") {
		t.Errorf("expected prefix [session] in log file, got %q", got)
	}
	if !strings.Contains(got, "created abc-123") {
		t.Errorf("expected message in log file, got %q", got)
	}
}

func TestNew_VerboseNoFile(t *testing.T) {
	l, err := New(true, "")
	if err != nil {
		t.Fatalf("New() error: %v", err)
	}
	defer l.Close()

	if !l.verbose {
		t.Error("expected verbose=true")
	}
}

func TestNew_NoVerboseNoFile(t *testing.T) {
	l, err := New(false, "")
	if err != nil {
		t.Fatalf("New() error: %v", err)
	}
	defer l.Close()

	if l.verbose {
		t.Error("expected verbose=false")
	}
}

func TestRegisterSecret_RedactsInOutput(t *testing.T) {
	var buf bytes.Buffer
	l := &Logger{
		verbose: true,
		out:     &buf,
	}

	l.RegisterSecret("sk-ant-abc123")
	l.Debug("auth", "token is sk-ant-abc123")

	got := buf.String()
	if strings.Contains(got, "sk-ant-abc123") {
		t.Errorf("expected secret to be redacted, got %q", got)
	}
	if !strings.Contains(got, "[REDACTED]") {
		t.Errorf("expected [REDACTED] placeholder, got %q", got)
	}
}

func TestRegisterSecret_RedactsInFileOutput(t *testing.T) {
	dir := t.TempDir()
	logPath := filepath.Join(dir, "debug.log")

	l, err := New(false, logPath)
	if err != nil {
		t.Fatalf("New() error: %v", err)
	}

	l.RegisterSecret("my-secret-token")
	l.Debug("auth", "using my-secret-token for login")
	l.Close()

	data, err := os.ReadFile(logPath)
	if err != nil {
		t.Fatalf("ReadFile error: %v", err)
	}
	got := string(data)
	if strings.Contains(got, "my-secret-token") {
		t.Errorf("expected secret redacted in file, got %q", got)
	}
}

func TestRegisterSecret_EmptyStringIgnored(t *testing.T) {
	var buf bytes.Buffer
	l := &Logger{
		verbose: true,
		out:     &buf,
	}

	l.RegisterSecret("")
	l.Debug("test", "hello world")

	got := buf.String()
	if !strings.Contains(got, "hello world") {
		t.Errorf("expected normal output, got %q", got)
	}
}

func TestClose_NilFileIsNoOp(t *testing.T) {
	l, err := New(true, "")
	if err != nil {
		t.Fatalf("New() error: %v", err)
	}
	if err := l.Close(); err != nil {
		t.Errorf("Close() on no-file logger should not error, got %v", err)
	}
}
