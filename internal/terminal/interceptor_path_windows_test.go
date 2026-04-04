//go:build windows

package terminal

import "testing"

func TestHasPathPrefix_Windows(t *testing.T) {
	tests := []struct {
		input string
		want  bool
	}{
		// Windows drive-letter paths.
		{`C:\Users\file.png`, true},
		{"D:/photos/img.jpg", true},

		// Windows backslash-relative paths.
		{`.\screenshot.png`, true},
		{`..\img.gif`, true},

		// Unix prefixes (WSL / Git Bash compatibility).
		{"/home/user/file.png", true},
		{"./local.png", true},
		{"../parent.png", true},
		{"~/pictures/photo.png", true},

		// Bare filename — no prefix.
		{"file.png", false},

		// URL — not a path prefix.
		{"http://example.com", false},
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			if got := hasPathPrefix(tt.input); got != tt.want {
				t.Errorf("hasPathPrefix(%q) = %v, want %v", tt.input, got, tt.want)
			}
		})
	}
}
