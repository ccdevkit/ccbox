package terminal

import (
	"bytes"
	"encoding/binary"
	"fmt"
	"image"
	"image/gif"
	"image/png"
	"io"
	"net"

	// Register image format decoders.
	_ "image/jpeg"

	_ "golang.org/x/image/bmp"
	_ "golang.org/x/image/tiff"
	_ "golang.org/x/image/webp"

	"github.com/ccdevkit/ccbox/internal/constants"
)

// ClipboardReader reads image data from the host clipboard.
type ClipboardReader interface {
	ReadImage() ([]byte, error)
}

// ClipboardSyncer syncs the host clipboard to the container.
type ClipboardSyncer interface {
	Sync() error
}

// NoOpClipboardSyncer is used on platforms without clipboard support.
// TDD exemption: trivial no-op with zero branching logic (Principle VII).
type NoOpClipboardSyncer struct{}

// Sync is a no-op that always returns nil.
func (n *NoOpClipboardSyncer) Sync() error { return nil }

// DebugFunc is a function for debug logging.
type DebugFunc func(prefix, format string, args ...any)

// TCPClipboardSyncer reads the host clipboard image, transcodes to PNG,
// and sends it to the container clipboard daemon via TCP.
type TCPClipboardSyncer struct {
	Address string
	Reader  ClipboardReader
	Debug   DebugFunc
}

func (s *TCPClipboardSyncer) debug(format string, args ...any) {
	if s.Debug != nil {
		s.Debug("clipboard-sync", format, args...)
	}
}

// Sync reads the clipboard image, transcodes to PNG, sends it to the container
// clipboard daemon at s.Address, and reads a 1-byte status response.
func (s *TCPClipboardSyncer) Sync() error {
	return s.syncWithMaxPayload(constants.MaxClipboardPayload)
}

func (s *TCPClipboardSyncer) syncWithMaxPayload(maxPayload int) error {
	s.debug("Sync triggered, reading clipboard image...")
	imgData, err := s.Reader.ReadImage()
	if err != nil {
		s.debug("ReadImage error: %v", err)
		return fmt.Errorf("reading clipboard image: %w", err)
	}
	if imgData == nil {
		s.debug("ReadImage returned nil (no image on clipboard)")
		return nil
	}
	s.debug("ReadImage returned %d bytes", len(imgData))

	pngData, err := TranscodeToPNG(imgData)
	if err != nil {
		s.debug("TranscodeToPNG error: %v", err)
		return fmt.Errorf("transcoding to PNG: %w", err)
	}
	s.debug("Transcoded to %d bytes PNG", len(pngData))

	if len(pngData) > maxPayload {
		s.debug("payload too large: %d > %d", len(pngData), maxPayload)
		return fmt.Errorf("clipboard payload %d bytes exceeds maximum %d bytes", len(pngData), maxPayload)
	}

	s.debug("connecting to %s", s.Address)
	conn, err := net.Dial("tcp", s.Address)
	if err != nil {
		s.debug("TCP connect error: %v", err)
		return fmt.Errorf("connecting to clipboard daemon: %w", err)
	}
	defer conn.Close()
	s.debug("connected")

	// Send 4-byte big-endian length prefix.
	lenBuf := make([]byte, 4)
	binary.BigEndian.PutUint32(lenBuf, uint32(len(pngData)))
	if _, err := conn.Write(lenBuf); err != nil {
		s.debug("write length error: %v", err)
		return fmt.Errorf("writing length prefix: %w", err)
	}

	// Send PNG payload.
	if _, err := conn.Write(pngData); err != nil {
		s.debug("write payload error: %v", err)
		return fmt.Errorf("writing PNG payload: %w", err)
	}

	s.debug("payload sent, reading status response")

	// Read 1-byte status response.
	statusBuf := make([]byte, 1)
	if _, err := io.ReadFull(conn, statusBuf); err != nil {
		s.debug("read status error: %v", err)
		return fmt.Errorf("reading status response: %w", err)
	}

	if statusBuf[0] != constants.ClipboardStatusSuccess {
		s.debug("daemon returned error status 0x%02x", statusBuf[0])
		return fmt.Errorf("clipboard daemon returned error status 0x%02x", statusBuf[0])
	}

	s.debug("sync complete, daemon returned success")
	return nil
}

// TranscodeToPNG decodes image data in any supported format (PNG, JPEG, GIF,
// WebP, BMP, TIFF) and re-encodes it as PNG. For animated GIFs, only the
// first frame is used.
func TranscodeToPNG(data []byte) ([]byte, error) {
	if len(data) == 0 {
		return nil, fmt.Errorf("empty image data")
	}

	r := bytes.NewReader(data)

	// Try animated GIF first to handle multi-frame extraction.
	if isGIF(data) {
		g, err := gif.DecodeAll(r)
		if err != nil {
			return nil, fmt.Errorf("decoding GIF: %w", err)
		}
		if len(g.Image) == 0 {
			return nil, fmt.Errorf("GIF contains no frames")
		}
		return encodePNG(g.Image[0])
	}

	img, _, err := image.Decode(r)
	if err != nil {
		return nil, fmt.Errorf("decoding image: %w", err)
	}

	return encodePNG(img)
}

func isGIF(data []byte) bool {
	return len(data) >= 3 && string(data[:3]) == "GIF"
}

func encodePNG(img image.Image) ([]byte, error) {
	var buf bytes.Buffer
	if err := png.Encode(&buf, img); err != nil {
		return nil, fmt.Errorf("encoding PNG: %w", err)
	}
	return buf.Bytes(), nil
}
