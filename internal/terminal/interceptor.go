package terminal

import (
	"bytes"
	"io"
	"os"
	"path/filepath"
	"strings"
	"sync"
)

const ctrlV = 0x16

var (
	pasteStart = "\x1b[200~"
	pasteEnd   = "\x1b[201~"
)

// imageExtensions lists the supported image file extensions for path rewriting.
var imageExtensions = map[string]bool{
	".png":  true,
	".jpg":  true,
	".jpeg": true,
	".gif":  true,
	".webp": true,
}

// FileBridger copies a host file to the bridge directory and returns the
// container-side path.
type FileBridger interface {
	CopyFileToBridge(srcPath string) (containerPath string, err error)
}

// Interceptor wraps an io.Reader, scanning for Ctrl+V (0x16) to trigger
// clipboard sync and for bracketed paste sequences to rewrite image file
// paths. All non-special bytes are forwarded unchanged.
//
// This is a single-pass design: each Read processes bytes inline without
// holding data back between reads, which is critical for terminal escape
// sequences to pass through without corruption.
type Interceptor struct {
	source  io.Reader
	syncer  ClipboardSyncer
	bridger FileBridger
	Debug   DebugFunc

	mu        sync.Mutex
	buffer    bytes.Buffer // overflow from previous processing
	inPaste   bool
	pasteData bytes.Buffer

	// WorkDir is used to resolve relative paths. If empty, os.Getwd() is used.
	WorkDir string
}

func (i *Interceptor) debug(format string, args ...any) {
	if i.Debug != nil {
		i.Debug("interceptor", format, args...)
	}
}

// NewInterceptor creates an Interceptor that reads from source, triggers
// syncer.Sync() when Ctrl+V is detected, and rewrites image file paths
// within bracketed paste sequences using bridger. Either syncer or bridger
// may be nil to disable that feature.
func NewInterceptor(source io.Reader, syncer ClipboardSyncer, bridger FileBridger) *Interceptor {
	return &Interceptor{
		source:  source,
		syncer:  syncer,
		bridger: bridger,
	}
}

// Read implements io.Reader.
func (i *Interceptor) Read(p []byte) (int, error) {
	i.mu.Lock()
	defer i.mu.Unlock()

	// Return buffered overflow from previous processing first.
	if i.buffer.Len() > 0 {
		return i.buffer.Read(p)
	}

	// Read from source.
	n, err := i.source.Read(p)
	if n == 0 {
		return n, err
	}

	// Process the data in a single pass.
	processed := i.process(p[:n])

	// Return processed data, buffering overflow.
	copy(p, processed)
	if len(processed) > len(p) {
		i.buffer.Write(processed[len(p):])
		return len(p), err
	}
	return len(processed), err
}

// process handles Ctrl+V detection and bracketed paste processing in a
// single pass over the data. Terminal escape sequences pass through
// immediately without being held between reads.
func (i *Interceptor) process(data []byte) []byte {
	var result bytes.Buffer

	for idx := 0; idx < len(data); idx++ {
		b := data[idx]
		remaining := string(data[idx:])

		// Check for bracketed paste start.
		if strings.HasPrefix(remaining, pasteStart) {
			i.inPaste = true
			i.pasteData.Reset()
			result.WriteString(pasteStart)
			idx += len(pasteStart) - 1
			continue
		}

		// Check for bracketed paste end.
		if strings.HasPrefix(remaining, pasteEnd) {
			i.inPaste = false
			processed := i.processPastedContent(i.pasteData.Bytes())
			result.Write(processed)
			result.WriteString(pasteEnd)
			idx += len(pasteEnd) - 1
			continue
		}

		// Accumulate paste content.
		if i.inPaste {
			i.pasteData.WriteByte(b)
			continue
		}

		// Check for Ctrl+V outside of bracketed paste.
		// Sync synchronously BEFORE forwarding the byte so the container
		// clipboard is populated by the time Claude Code reads it.
		if b == ctrlV && i.syncer != nil {
			i.debug("Ctrl+V detected (0x16), syncing clipboard before forwarding")
			if err := i.syncer.Sync(); err != nil {
				i.debug("Sync error: %v", err)
			}
		}

		result.WriteByte(b)
	}

	// If still in paste mode (paste spans multiple reads), flush accumulated
	// paste data so terminal I/O isn't blocked.
	if i.inPaste {
		result.Write(i.pasteData.Bytes())
		i.pasteData.Reset()
	}

	return result.Bytes()
}

// processPastedContent scans paste content for image file paths and rewrites them.
func (i *Interceptor) processPastedContent(content []byte) []byte {
	if i.bridger == nil {
		return content
	}

	s := string(content)
	tokens := tokenizePasteContent(s)

	var result strings.Builder
	for _, tok := range tokens {
		if tok.isPath {
			resolved := i.resolveAndRewrite(tok.raw)
			if resolved != "" {
				result.WriteString(resolved)
				continue
			}
		}
		result.WriteString(tok.raw)
	}

	return []byte(result.String())
}

type pasteToken struct {
	raw    string
	isPath bool
}

// tokenizePasteContent splits paste text into tokens, identifying potential
// file paths. A token is a contiguous run of non-whitespace characters,
// possibly including shell escapes (backslash-space) or surrounding quotes.
func tokenizePasteContent(s string) []pasteToken {
	var tokens []pasteToken
	i := 0

	for i < len(s) {
		// Skip whitespace — preserve it as non-path tokens.
		if s[i] == ' ' || s[i] == '\t' || s[i] == '\n' || s[i] == '\r' {
			j := i
			for j < len(s) && (s[j] == ' ' || s[j] == '\t' || s[j] == '\n' || s[j] == '\r') {
				j++
			}
			tokens = append(tokens, pasteToken{raw: s[i:j], isPath: false})
			i = j
			continue
		}

		// Check for quoted strings.
		if s[i] == '\'' || s[i] == '"' {
			quote := s[i]
			j := i + 1
			for j < len(s) && s[j] != quote {
				j++
			}
			if j < len(s) {
				inner := s[i+1 : j]
				raw := s[i : j+1]
				tokens = append(tokens, pasteToken{raw: raw, isPath: isPathCandidate(inner)})
				i = j + 1
				continue
			}
		}

		// Unquoted token: handle backslash-escaped spaces.
		j := i
		for j < len(s) {
			if s[j] == '\\' && j+1 < len(s) {
				j += 2
				continue
			}
			if s[j] == ' ' || s[j] == '\t' || s[j] == '\n' || s[j] == '\r' {
				break
			}
			j++
		}

		raw := s[i:j]
		unescaped := unescapeShell(raw)
		tokens = append(tokens, pasteToken{raw: raw, isPath: isPathCandidate(unescaped)})
		i = j
	}

	return tokens
}

// isPathCandidate checks if a string looks like a file path with a recognized
// prefix and image extension. URLs are excluded.
func isPathCandidate(s string) bool {
	if strings.HasPrefix(s, "http://") || strings.HasPrefix(s, "https://") {
		return false
	}
	if !hasPathPrefix(s) {
		return false
	}
	ext := strings.ToLower(filepath.Ext(s))
	return imageExtensions[ext]
}

// unescapeShell removes backslash escapes from a string.
func unescapeShell(s string) string {
	var b strings.Builder
	i := 0
	for i < len(s) {
		if s[i] == '\\' && i+1 < len(s) {
			b.WriteByte(s[i+1])
			i += 2
			continue
		}
		b.WriteByte(s[i])
		i++
	}
	return b.String()
}

// resolveAndRewrite resolves a path token, checks if it's an existing image
// file, copies it to the bridge directory, and returns the container path.
// Returns empty string if the path should not be rewritten.
func (i *Interceptor) resolveAndRewrite(raw string) string {
	// Strip quotes if present.
	path := raw
	if len(path) >= 2 && (path[0] == '\'' || path[0] == '"') && path[len(path)-1] == path[0] {
		path = path[1 : len(path)-1]
	}

	path = unescapeShell(path)

	// Expand ~/ to home directory.
	if strings.HasPrefix(path, "~/") {
		home, err := os.UserHomeDir()
		if err != nil {
			return ""
		}
		path = filepath.Join(home, path[2:])
	}

	// Resolve relative paths.
	if !filepath.IsAbs(path) {
		workDir := i.WorkDir
		if workDir == "" {
			var err error
			workDir, err = os.Getwd()
			if err != nil {
				return ""
			}
		}
		path = filepath.Join(workDir, path)
	}

	path = filepath.Clean(path)

	if _, err := os.Stat(path); err != nil {
		return ""
	}

	containerPath, err := i.bridger.CopyFileToBridge(path)
	if err != nil {
		return ""
	}

	return containerPath
}
