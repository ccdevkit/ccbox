//go:build !darwin && !noclipboard

package clipboard

import "golang.design/x/clipboard"

// readImagePlatform reads image data using golang.design/x/clipboard.
// Returns PNG-encoded bytes, or nil if no image data is present.
func readImagePlatform() ([]byte, error) {
	data := clipboard.Read(clipboard.FmtImage)
	if len(data) == 0 {
		return nil, nil
	}
	return data, nil
}
