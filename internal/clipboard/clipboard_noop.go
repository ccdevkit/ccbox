//go:build noclipboard

// Package clipboard provides host clipboard access for reading image data.
//
// TDD Exemption (Principle VII): Thin platform wrapper with no business logic.
// This no-op build always returns ErrNotSupported.
package clipboard

import "errors"

// ErrNotSupported is returned when clipboard access is not available.
var ErrNotSupported = errors.New("clipboard: not supported on this platform")

// Init is a no-op when built with the noclipboard tag.
func Init() error { return ErrNotSupported }

// ReadImage returns ErrNotSupported when built with the noclipboard tag.
func ReadImage() ([]byte, error) {
	return nil, ErrNotSupported
}

// Reader adapts the package-level ReadImage function to the
// terminal.ClipboardReader interface.
type Reader struct{}

// ReadImage returns ErrNotSupported when built with the noclipboard tag.
func (Reader) ReadImage() ([]byte, error) { return ReadImage() }
