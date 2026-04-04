//go:build !windows

package terminal

import "strings"

// hasPathPrefix checks if s starts with a recognized Unix path prefix.
func hasPathPrefix(s string) bool {
	return strings.HasPrefix(s, "/") ||
		strings.HasPrefix(s, "./") ||
		strings.HasPrefix(s, "../") ||
		strings.HasPrefix(s, "~/")
}
