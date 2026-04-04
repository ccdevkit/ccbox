package logger

import (
	"fmt"
	"io"
	"os"
	"strings"
	"sync"
)

// Logger provides debug logging with contextual prefixes, optional file output,
// and secret redaction.
type Logger struct {
	verbose bool
	out     io.Writer // stderr or multi-writer
	file    *os.File  // non-nil when logging to file
	secrets []string
	mu      sync.Mutex
}

// New creates a Logger. When logFile is non-empty, output is written to that
// file and verbose is implicitly enabled. When verbose is true (or implied by
// logFile), output goes to stderr. When both are off, all output is discarded.
func New(verbose bool, logFile string) (*Logger, error) {
	l := &Logger{}

	if logFile != "" {
		f, err := os.Create(logFile)
		if err != nil {
			return nil, fmt.Errorf("logger: open log file: %w", err)
		}
		l.file = f
		l.verbose = true
		l.out = f
		return l, nil
	}

	l.verbose = verbose
	if verbose {
		l.out = os.Stderr
	}

	return l, nil
}

// NewWithWriter creates a verbose Logger that writes to the given writer.
// This is primarily useful for testing.
func NewWithWriter(w io.Writer) *Logger {
	return &Logger{
		verbose: true,
		out:     w,
	}
}

// Debug writes a formatted log line with a contextual prefix (e.g., "[docker] message").
// It is a no-op when the logger is not verbose.
func (l *Logger) Debug(prefix, format string, args ...interface{}) {
	if !l.verbose {
		return
	}

	msg := fmt.Sprintf(format, args...)
	line := fmt.Sprintf("[%s] %s\n", prefix, msg)

	l.mu.Lock()
	defer l.mu.Unlock()

	for _, s := range l.secrets {
		line = strings.ReplaceAll(line, s, "[REDACTED]")
	}

	fmt.Fprint(l.out, line)
}

// RegisterSecret adds a value that will be masked as [REDACTED] in all future
// log output. Empty strings are ignored.
func (l *Logger) RegisterSecret(secret string) {
	if secret == "" {
		return
	}
	l.mu.Lock()
	defer l.mu.Unlock()
	l.secrets = append(l.secrets, secret)
}

// Close releases resources. It is safe to call on a logger with no open file.
func (l *Logger) Close() error {
	if l.file != nil {
		return l.file.Close()
	}
	return nil
}
