//go:build windows

package terminal

import "strings"

// hasPathPrefix checks if s starts with a recognized path prefix.
// On Windows this includes drive-letter paths and backslash-relative paths
// in addition to Unix prefixes (for WSL/Git Bash compatibility).
func hasPathPrefix(s string) bool {
	// Unix prefixes (WSL / Git Bash compatibility).
	if strings.HasPrefix(s, "/") ||
		strings.HasPrefix(s, "./") ||
		strings.HasPrefix(s, "../") ||
		strings.HasPrefix(s, "~/") {
		return true
	}

	// Windows backslash-relative paths.
	if strings.HasPrefix(s, `.\`) || strings.HasPrefix(s, `..\`) {
		return true
	}

	// Drive letter paths: e.g. C:\, D:/, E:\Users
	if len(s) >= 3 {
		ch := s[0]
		if (ch >= 'A' && ch <= 'Z' || ch >= 'a' && ch <= 'z') &&
			s[1] == ':' &&
			(s[2] == '\\' || s[2] == '/') {
			return true
		}
	}

	return false
}
