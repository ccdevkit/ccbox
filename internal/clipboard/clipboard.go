//go:build !noclipboard

// Package clipboard provides host clipboard access for reading image data.
//
// TDD Exemption (Principle VII): Thin platform wrapper around golang.design/x/clipboard
// with no business logic. Testing requires a live display server and system clipboard,
// making automated tests impractical and brittle.
package clipboard

import (
	"fmt"

	"golang.design/x/clipboard"
)

// Init initializes the clipboard library. Must be called once before
// ReadImage, from the main goroutine (macOS Cocoa requires main thread).
// Returns an error if initialization fails (e.g. no display server, no CGO).
func Init() (err error) {
	defer func() {
		if r := recover(); r != nil {
			err = fmt.Errorf("clipboard not available: %v", r)
		}
	}()
	return clipboard.Init()
}

// ReadImage reads image data from the system clipboard.
// On macOS, this reads directly from NSPasteboard supporting both PNG and
// TIFF formats (the library's FmtImage only supports PNG, but most macOS
// clipboard operations produce TIFF). On other platforms, falls back to
// golang.design/x/clipboard.
// Returns PNG-encoded bytes, or nil if no image data is present.
func ReadImage() ([]byte, error) {
	return readImagePlatform()
}

// Reader adapts the package-level ReadImage function to the
// terminal.ClipboardReader interface.
type Reader struct{}

// ReadImage reads image data from the system clipboard.
func (Reader) ReadImage() ([]byte, error) { return ReadImage() }
