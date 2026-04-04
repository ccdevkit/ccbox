package terminal

import (
	"bytes"
	"io"
	"os"
	"path/filepath"
	"sync"
	"testing"
)

// mockSyncer records calls to Sync.
type mockSyncer struct {
	mu       sync.Mutex
	calls    int
	syncDone chan struct{} // closed after each Sync call
}

func newMockSyncer() *mockSyncer {
	return &mockSyncer{syncDone: make(chan struct{}, 1)}
}

func (m *mockSyncer) Sync() error {
	m.mu.Lock()
	m.calls++
	m.mu.Unlock()
	select {
	case m.syncDone <- struct{}{}:
	default:
	}
	return nil
}

func (m *mockSyncer) syncCount() int {
	m.mu.Lock()
	defer m.mu.Unlock()
	return m.calls
}

// mockFileCopier records CopyFile calls and returns configured results.
type mockFileCopier struct {
	mu    sync.Mutex
	calls []copyCall
	err   error
}

type copyCall struct {
	srcPath string
}

func (m *mockFileCopier) CopyFileToBridge(srcPath string) (containerPath string, err error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.calls = append(m.calls, copyCall{srcPath: srcPath})
	if m.err != nil {
		return "", m.err
	}
	return "/home/claude/.bridge/" + filepath.Base(srcPath), nil
}

func (m *mockFileCopier) getCalls() []copyCall {
	m.mu.Lock()
	defer m.mu.Unlock()
	result := make([]copyCall, len(m.calls))
	copy(result, m.calls)
	return result
}

func TestInterceptor_PassThroughNonCtrlV(t *testing.T) {
	input := []byte("hello world\n")
	syncer := newMockSyncer()
	interceptor := NewInterceptor(bytes.NewReader(input), syncer, nil)

	out, err := io.ReadAll(interceptor)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if !bytes.Equal(out, input) {
		t.Errorf("got %q, want %q", out, input)
	}

	if syncer.syncCount() != 0 {
		t.Errorf("Sync called %d times, want 0", syncer.syncCount())
	}
}

func TestInterceptor_CtrlVTriggersSyncAndForwardsByte(t *testing.T) {
	input := []byte{'a', 0x16, 'b'}
	syncer := newMockSyncer()
	interceptor := NewInterceptor(bytes.NewReader(input), syncer, nil)

	out, err := io.ReadAll(interceptor)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if !bytes.Equal(out, input) {
		t.Errorf("got %q, want %q", out, input)
	}

	<-syncer.syncDone

	if syncer.syncCount() != 1 {
		t.Errorf("Sync called %d times, want 1", syncer.syncCount())
	}
}

// --- Paste path rewriting tests ---

func TestInterceptor_DetectsAbsolutePath(t *testing.T) {
	tmpDir := t.TempDir()
	imgPath := filepath.Join(tmpDir, "photo.png")
	if err := os.WriteFile(imgPath, []byte("fake png"), 0644); err != nil {
		t.Fatal(err)
	}

	paste := bracketedPaste(imgPath)
	copier := &mockFileCopier{}
	interceptor := NewInterceptor(bytes.NewReader(paste), nil, copier)

	out, err := io.ReadAll(interceptor)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	expected := bracketedPaste("/home/claude/.bridge/photo.png")
	if !bytes.Equal(out, expected) {
		t.Errorf("got %q, want %q", string(out), string(expected))
	}

	calls := copier.getCalls()
	if len(calls) != 1 {
		t.Fatalf("expected 1 CopyFileToBridge call, got %d", len(calls))
	}
	if calls[0].srcPath != imgPath {
		t.Errorf("copied %q, want %q", calls[0].srcPath, imgPath)
	}
}

func TestInterceptor_DetectsRelativeDotSlashPath(t *testing.T) {
	tmpDir := t.TempDir()
	imgPath := filepath.Join(tmpDir, "img.jpg")
	if err := os.WriteFile(imgPath, []byte("fake jpg"), 0644); err != nil {
		t.Fatal(err)
	}

	relPath := "./" + filepath.Base(imgPath)
	paste := bracketedPaste(relPath)
	copier := &mockFileCopier{}
	interceptor := NewInterceptor(bytes.NewReader(paste), nil, copier)
	interceptor.WorkDir = tmpDir

	out, err := io.ReadAll(interceptor)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	expected := bracketedPaste("/home/claude/.bridge/img.jpg")
	if !bytes.Equal(out, expected) {
		t.Errorf("got %q, want %q", string(out), string(expected))
	}
}

func TestInterceptor_DetectsRelativeDotDotSlashPath(t *testing.T) {
	tmpDir := t.TempDir()
	subDir := filepath.Join(tmpDir, "sub")
	if err := os.Mkdir(subDir, 0755); err != nil {
		t.Fatal(err)
	}
	imgPath := filepath.Join(tmpDir, "up.png")
	if err := os.WriteFile(imgPath, []byte("fake png"), 0644); err != nil {
		t.Fatal(err)
	}

	relPath := "../up.png"
	paste := bracketedPaste(relPath)
	copier := &mockFileCopier{}
	interceptor := NewInterceptor(bytes.NewReader(paste), nil, copier)
	interceptor.WorkDir = subDir

	out, err := io.ReadAll(interceptor)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	expected := bracketedPaste("/home/claude/.bridge/up.png")
	if !bytes.Equal(out, expected) {
		t.Errorf("got %q, want %q", string(out), string(expected))
	}
}

func TestInterceptor_DetectsTildePath(t *testing.T) {
	homeDir, err := os.UserHomeDir()
	if err != nil {
		t.Skip("cannot determine home dir")
	}

	imgPath := filepath.Join(homeDir, "test-paste-img.png")
	if err := os.WriteFile(imgPath, []byte("fake png"), 0644); err != nil {
		t.Fatal(err)
	}
	defer os.Remove(imgPath)

	paste := bracketedPaste("~/test-paste-img.png")
	copier := &mockFileCopier{}
	interceptor := NewInterceptor(bytes.NewReader(paste), nil, copier)

	out, err2 := io.ReadAll(interceptor)
	if err2 != nil {
		t.Fatalf("unexpected error: %v", err2)
	}

	expected := bracketedPaste("/home/claude/.bridge/test-paste-img.png")
	if !bytes.Equal(out, expected) {
		t.Errorf("got %q, want %q", string(out), string(expected))
	}
}

func TestInterceptor_ShellEscapedPath(t *testing.T) {
	tmpDir := t.TempDir()
	imgPath := filepath.Join(tmpDir, "my file.png")
	if err := os.WriteFile(imgPath, []byte("fake png"), 0644); err != nil {
		t.Fatal(err)
	}

	escapedPath := tmpDir + "/my\\ file.png"
	paste := bracketedPaste(escapedPath)
	copier := &mockFileCopier{}
	interceptor := NewInterceptor(bytes.NewReader(paste), nil, copier)

	out, err := io.ReadAll(interceptor)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	expected := bracketedPaste("/home/claude/.bridge/my file.png")
	if !bytes.Equal(out, expected) {
		t.Errorf("got %q, want %q", string(out), string(expected))
	}

	calls := copier.getCalls()
	if len(calls) != 1 {
		t.Fatalf("expected 1 call, got %d", len(calls))
	}
	if calls[0].srcPath != imgPath {
		t.Errorf("copied %q, want %q", calls[0].srcPath, imgPath)
	}
}

func TestInterceptor_QuotedPath(t *testing.T) {
	tmpDir := t.TempDir()
	imgPath := filepath.Join(tmpDir, "my file.png")
	if err := os.WriteFile(imgPath, []byte("fake png"), 0644); err != nil {
		t.Fatal(err)
	}

	quotedPath := "'" + imgPath + "'"
	paste := bracketedPaste(quotedPath)
	copier := &mockFileCopier{}
	interceptor := NewInterceptor(bytes.NewReader(paste), nil, copier)

	out, err := io.ReadAll(interceptor)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	expected := bracketedPaste("/home/claude/.bridge/my file.png")
	if !bytes.Equal(out, expected) {
		t.Errorf("got %q, want %q", string(out), string(expected))
	}
}

func TestInterceptor_URLNotTreatedAsPath(t *testing.T) {
	input := bracketedPaste("https://example.com/image.png")
	copier := &mockFileCopier{}
	interceptor := NewInterceptor(bytes.NewReader(input), nil, copier)

	out, err := io.ReadAll(interceptor)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if !bytes.Equal(out, input) {
		t.Errorf("got %q, want %q", string(out), string(input))
	}

	if len(copier.getCalls()) != 0 {
		t.Errorf("expected 0 CopyFileToBridge calls, got %d", len(copier.getCalls()))
	}
}

func TestInterceptor_HTTPURLNotTreatedAsPath(t *testing.T) {
	input := bracketedPaste("http://example.com/photo.jpg")
	copier := &mockFileCopier{}
	interceptor := NewInterceptor(bytes.NewReader(input), nil, copier)

	out, err := io.ReadAll(interceptor)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if !bytes.Equal(out, input) {
		t.Errorf("got %q, want %q", string(out), string(input))
	}

	if len(copier.getCalls()) != 0 {
		t.Error("expected no CopyFileToBridge calls for HTTP URL")
	}
}

func TestInterceptor_BareFilenameNotRewritten(t *testing.T) {
	input := bracketedPaste("screenshot.png")
	copier := &mockFileCopier{}
	interceptor := NewInterceptor(bytes.NewReader(input), nil, copier)

	out, err := io.ReadAll(interceptor)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if !bytes.Equal(out, input) {
		t.Errorf("got %q, want %q", string(out), string(input))
	}

	if len(copier.getCalls()) != 0 {
		t.Error("expected no CopyFileToBridge calls for bare filename")
	}
}

func TestInterceptor_MultiplePaths(t *testing.T) {
	tmpDir := t.TempDir()
	img1 := filepath.Join(tmpDir, "a.png")
	img2 := filepath.Join(tmpDir, "b.jpg")
	if err := os.WriteFile(img1, []byte("fake"), 0644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(img2, []byte("fake"), 0644); err != nil {
		t.Fatal(err)
	}

	pasteContent := img1 + " " + img2
	paste := bracketedPaste(pasteContent)
	copier := &mockFileCopier{}
	interceptor := NewInterceptor(bytes.NewReader(paste), nil, copier)

	out, err := io.ReadAll(interceptor)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	expected := bracketedPaste("/home/claude/.bridge/a.png /home/claude/.bridge/b.jpg")
	if !bytes.Equal(out, expected) {
		t.Errorf("got %q, want %q", string(out), string(expected))
	}

	calls := copier.getCalls()
	if len(calls) != 2 {
		t.Fatalf("expected 2 CopyFileToBridge calls, got %d", len(calls))
	}
}

func TestInterceptor_NonImageExtensionIgnored(t *testing.T) {
	tmpDir := t.TempDir()
	txtPath := filepath.Join(tmpDir, "notes.txt")
	if err := os.WriteFile(txtPath, []byte("text"), 0644); err != nil {
		t.Fatal(err)
	}

	paste := bracketedPaste(txtPath)
	copier := &mockFileCopier{}
	interceptor := NewInterceptor(bytes.NewReader(paste), nil, copier)

	out, err := io.ReadAll(interceptor)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if !bytes.Equal(out, paste) {
		t.Errorf("got %q, want %q", string(out), string(paste))
	}

	if len(copier.getCalls()) != 0 {
		t.Error("expected no CopyFileToBridge calls for .txt file")
	}
}

func TestInterceptor_NonexistentFileIgnored(t *testing.T) {
	paste := bracketedPaste("/nonexistent/path/photo.png")
	copier := &mockFileCopier{}
	interceptor := NewInterceptor(bytes.NewReader(paste), nil, copier)

	out, err := io.ReadAll(interceptor)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if !bytes.Equal(out, paste) {
		t.Errorf("got %q, want %q", string(out), string(paste))
	}

	if len(copier.getCalls()) != 0 {
		t.Error("expected no CopyFileToBridge calls for nonexistent file")
	}
}

func TestInterceptor_NonPasteDataPassedThrough(t *testing.T) {
	input := []byte("hello world /tmp/photo.png more text")
	copier := &mockFileCopier{}
	interceptor := NewInterceptor(bytes.NewReader(input), nil, copier)

	out, err := io.ReadAll(interceptor)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if !bytes.Equal(out, input) {
		t.Errorf("got %q, want %q", string(out), string(input))
	}

	if len(copier.getCalls()) != 0 {
		t.Error("expected no CopyFileToBridge calls for non-paste data")
	}
}

func TestInterceptor_MixedPasteAndNonPaste(t *testing.T) {
	tmpDir := t.TempDir()
	imgPath := filepath.Join(tmpDir, "pic.png")
	if err := os.WriteFile(imgPath, []byte("fake"), 0644); err != nil {
		t.Fatal(err)
	}

	var buf bytes.Buffer
	buf.WriteString("before ")
	buf.Write(bracketedPaste(imgPath))
	buf.WriteString(" after")

	copier := &mockFileCopier{}
	interceptor := NewInterceptor(bytes.NewReader(buf.Bytes()), nil, copier)

	out, err := io.ReadAll(interceptor)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	var expectedBuf bytes.Buffer
	expectedBuf.WriteString("before ")
	expectedBuf.Write(bracketedPaste("/home/claude/.bridge/pic.png"))
	expectedBuf.WriteString(" after")

	if !bytes.Equal(out, expectedBuf.Bytes()) {
		t.Errorf("got %q, want %q", string(out), string(expectedBuf.Bytes()))
	}
}

func TestInterceptor_WebPExtension(t *testing.T) {
	tmpDir := t.TempDir()
	imgPath := filepath.Join(tmpDir, "photo.webp")
	if err := os.WriteFile(imgPath, []byte("fake webp"), 0644); err != nil {
		t.Fatal(err)
	}

	paste := bracketedPaste(imgPath)
	copier := &mockFileCopier{}
	interceptor := NewInterceptor(bytes.NewReader(paste), nil, copier)

	out, err := io.ReadAll(interceptor)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	expected := bracketedPaste("/home/claude/.bridge/photo.webp")
	if !bytes.Equal(out, expected) {
		t.Errorf("got %q, want %q", string(out), string(expected))
	}
}

func TestInterceptor_GIFExtension(t *testing.T) {
	tmpDir := t.TempDir()
	imgPath := filepath.Join(tmpDir, "anim.gif")
	if err := os.WriteFile(imgPath, []byte("fake gif"), 0644); err != nil {
		t.Fatal(err)
	}

	paste := bracketedPaste(imgPath)
	copier := &mockFileCopier{}
	interceptor := NewInterceptor(bytes.NewReader(paste), nil, copier)

	out, err := io.ReadAll(interceptor)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	expected := bracketedPaste("/home/claude/.bridge/anim.gif")
	if !bytes.Equal(out, expected) {
		t.Errorf("got %q, want %q", string(out), string(expected))
	}
}

func TestInterceptor_EscapeSequencePassedThrough(t *testing.T) {
	// Terminal escape sequences like DA1 response must pass through unchanged.
	input := []byte("\x1b[?64;1;2;4;6;17;18;21;22;52c")
	interceptor := NewInterceptor(bytes.NewReader(input), nil, nil)

	out, err := io.ReadAll(interceptor)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if !bytes.Equal(out, input) {
		t.Errorf("got %v, want %v", out, input)
	}
}

// bracketedPaste wraps content in ESC[200~ ... ESC[201~ bracketed paste markers.
func bracketedPaste(content string) []byte {
	var buf bytes.Buffer
	buf.WriteString("\x1b[200~")
	buf.WriteString(content)
	buf.WriteString("\x1b[201~")
	return buf.Bytes()
}
