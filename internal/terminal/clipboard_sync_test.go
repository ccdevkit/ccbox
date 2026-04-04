package terminal

import (
	"bytes"
	"encoding/binary"
	"fmt"
	"image"
	"image/color"
	"image/gif"
	"image/jpeg"
	"image/png"
	"io"
	"net"
	"strings"
	"testing"

	"github.com/ccdevkit/ccbox/internal/constants"
	"golang.org/x/image/bmp"
	"golang.org/x/image/tiff"
)

// mockClipboardReader implements ClipboardReader for tests.
type mockClipboardReader struct {
	data []byte
	err  error
}

func (m *mockClipboardReader) ReadImage() ([]byte, error) {
	return m.data, m.err
}

// makeTestImage creates a 2x2 RGBA image with distinct pixel colors.
func makeTestImage() *image.RGBA {
	img := image.NewRGBA(image.Rect(0, 0, 2, 2))
	img.Set(0, 0, color.RGBA{R: 255, A: 255})
	img.Set(1, 0, color.RGBA{G: 255, A: 255})
	img.Set(0, 1, color.RGBA{B: 255, A: 255})
	img.Set(1, 1, color.RGBA{R: 255, G: 255, B: 255, A: 255})
	return img
}

// encodeJPEG encodes an image as JPEG bytes.
func encodeJPEG(img image.Image) []byte {
	var buf bytes.Buffer
	_ = jpeg.Encode(&buf, img, nil)
	return buf.Bytes()
}

// testEncodePNG encodes an image as PNG bytes for test data.
func testEncodePNG(img image.Image) []byte {
	var buf bytes.Buffer
	_ = png.Encode(&buf, img)
	return buf.Bytes()
}

// encodeGIF encodes an image as a single-frame GIF.
func encodeGIF(img image.Image) []byte {
	bounds := img.Bounds()
	palettedImg := image.NewPaletted(bounds, color.Palette{
		color.RGBA{R: 255, A: 255},
		color.RGBA{G: 255, A: 255},
		color.RGBA{B: 255, A: 255},
		color.RGBA{R: 255, G: 255, B: 255, A: 255},
	})
	for y := bounds.Min.Y; y < bounds.Max.Y; y++ {
		for x := bounds.Min.X; x < bounds.Max.X; x++ {
			palettedImg.Set(x, y, img.At(x, y))
		}
	}
	var buf bytes.Buffer
	_ = gif.Encode(&buf, palettedImg, nil)
	return buf.Bytes()
}

// encodeAnimatedGIF creates a 2-frame animated GIF. Frame 0 is red, frame 1 is blue.
func encodeAnimatedGIF() []byte {
	palette := color.Palette{
		color.RGBA{R: 255, A: 255},
		color.RGBA{B: 255, A: 255},
	}

	frame0 := image.NewPaletted(image.Rect(0, 0, 2, 2), palette)
	for y := 0; y < 2; y++ {
		for x := 0; x < 2; x++ {
			frame0.SetColorIndex(x, y, 0) // red
		}
	}

	frame1 := image.NewPaletted(image.Rect(0, 0, 2, 2), palette)
	for y := 0; y < 2; y++ {
		for x := 0; x < 2; x++ {
			frame1.SetColorIndex(x, y, 1) // blue
		}
	}

	g := &gif.GIF{
		Image: []*image.Paletted{frame0, frame1},
		Delay: []int{100, 100},
	}

	var buf bytes.Buffer
	_ = gif.EncodeAll(&buf, g)
	return buf.Bytes()
}

// encodeBMP encodes an image as BMP bytes.
func encodeBMP(img image.Image) []byte {
	var buf bytes.Buffer
	_ = bmp.Encode(&buf, img)
	return buf.Bytes()
}

// encodeTIFF encodes an image as TIFF bytes.
func encodeTIFF(img image.Image) []byte {
	var buf bytes.Buffer
	_ = tiff.Encode(&buf, img, nil)
	return buf.Bytes()
}

// testWebPData returns a valid 2x2 lossless WebP image (generated via cwebp).
// Since golang.org/x/image/webp only provides a decoder, we embed pre-encoded bytes.
func testWebPData() []byte {
	return []byte{
		0x52, 0x49, 0x46, 0x46, 0x30, 0x00, 0x00, 0x00, 0x57, 0x45, 0x42, 0x50,
		0x56, 0x50, 0x38, 0x4c, 0x23, 0x00, 0x00, 0x00, 0x2f, 0x01, 0x40, 0x00,
		0x00, 0x1f, 0x20, 0x10, 0x20, 0x38, 0x77, 0x6e, 0x43, 0x40, 0x50, 0x74,
		0xdd, 0x72, 0x02, 0x01, 0x82, 0x73, 0xe7, 0xe6, 0x3f, 0xf0, 0xc9, 0x51,
		0xc1, 0x0d, 0x18, 0x22, 0xfa, 0x1f, 0x02, 0x00,
	}
}

func TestTranscodeToPNG(t *testing.T) {
	baseImg := makeTestImage()

	tests := []struct {
		name string
		data []byte
	}{
		{"PNG passthrough", testEncodePNG(baseImg)},
		{"JPEG to PNG", encodeJPEG(baseImg)},
		{"GIF to PNG", encodeGIF(baseImg)},
		{"BMP to PNG", encodeBMP(baseImg)},
		{"TIFF to PNG", encodeTIFF(baseImg)},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := TranscodeToPNG(tt.data)
			if err != nil {
				t.Fatalf("TranscodeToPNG() error: %v", err)
			}

			// Verify the result is valid PNG by decoding it.
			_, format, err := image.Decode(bytes.NewReader(result))
			if err != nil {
				t.Fatalf("result is not a valid image: %v", err)
			}
			if format != "png" {
				t.Errorf("expected png format, got %s", format)
			}
		})
	}
}

func TestTranscodeToPNG_WebP(t *testing.T) {
	data := testWebPData()

	result, err := TranscodeToPNG(data)
	if err != nil {
		t.Fatalf("TranscodeToPNG() error: %v", err)
	}

	decoded, format, err := image.Decode(bytes.NewReader(result))
	if err != nil {
		t.Fatalf("result is not a valid image: %v", err)
	}
	if format != "png" {
		t.Errorf("expected png format, got %s", format)
	}

	// Verify dimensions match the source (2x2).
	bounds := decoded.Bounds()
	if bounds.Dx() != 2 || bounds.Dy() != 2 {
		t.Errorf("expected 2x2 image, got %dx%d", bounds.Dx(), bounds.Dy())
	}
}

func TestTranscodeToPNG_AnimatedGIF(t *testing.T) {
	data := encodeAnimatedGIF()

	result, err := TranscodeToPNG(data)
	if err != nil {
		t.Fatalf("TranscodeToPNG() error: %v", err)
	}

	// Verify valid PNG output.
	decoded, format, err := image.Decode(bytes.NewReader(result))
	if err != nil {
		t.Fatalf("result is not a valid image: %v", err)
	}
	if format != "png" {
		t.Errorf("expected png format, got %s", format)
	}

	// Verify it used the first frame (all red pixels).
	bounds := decoded.Bounds()
	for y := bounds.Min.Y; y < bounds.Max.Y; y++ {
		for x := bounds.Min.X; x < bounds.Max.X; x++ {
			r, g, b, a := decoded.At(x, y).RGBA()
			// First frame is all red (palette index 0 = red).
			if r == 0 && g == 0 && b == 0 {
				t.Errorf("pixel (%d,%d) is black, expected red from first frame", x, y)
			}
			_ = a
		}
	}
}

func TestTranscodeToPNG_InvalidData(t *testing.T) {
	_, err := TranscodeToPNG([]byte("not an image"))
	if err == nil {
		t.Fatal("expected error for invalid image data, got nil")
	}
}

func TestTranscodeToPNG_EmptyData(t *testing.T) {
	_, err := TranscodeToPNG([]byte{})
	if err == nil {
		t.Fatal("expected error for empty data, got nil")
	}
}

// startTestClipServer starts a TCP server that reads the clipboard protocol
// message and responds with the given status byte. It returns the received
// length prefix and payload via channels.
func startTestClipServer(t *testing.T, status byte) (addr string, gotLen chan uint32, gotPayload chan []byte) {
	t.Helper()
	ln, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatalf("listen: %v", err)
	}
	t.Cleanup(func() { ln.Close() })

	gotLen = make(chan uint32, 1)
	gotPayload = make(chan []byte, 1)

	go func() {
		conn, err := ln.Accept()
		if err != nil {
			return
		}
		defer conn.Close()

		// Read 4-byte length prefix.
		lenBuf := make([]byte, 4)
		if _, err := io.ReadFull(conn, lenBuf); err != nil {
			return
		}
		length := binary.BigEndian.Uint32(lenBuf)
		gotLen <- length

		// Read payload.
		payload := make([]byte, length)
		if _, err := io.ReadFull(conn, payload); err != nil {
			return
		}
		gotPayload <- payload

		// Send status response.
		conn.Write([]byte{status})
	}()

	return ln.Addr().String(), gotLen, gotPayload
}

func TestTCPClipboardSyncer_Sync_SendsCorrectLengthAndPayload(t *testing.T) {
	// Create a small PNG for the mock reader to return.
	img := makeTestImage()
	pngData := testEncodePNG(img)

	addr, gotLen, gotPayload := startTestClipServer(t, constants.ClipboardStatusSuccess)

	syncer := &TCPClipboardSyncer{
		Address: addr,
		Reader:  &mockClipboardReader{data: pngData},
	}

	if err := syncer.Sync(); err != nil {
		t.Fatalf("Sync() error: %v", err)
	}

	length := <-gotLen
	payload := <-gotPayload

	// Length prefix must match payload size.
	if length != uint32(len(payload)) {
		t.Errorf("length prefix %d != actual payload length %d", length, len(payload))
	}

	// Payload must be valid PNG.
	_, format, err := image.Decode(bytes.NewReader(payload))
	if err != nil {
		t.Fatalf("received payload is not valid image: %v", err)
	}
	if format != "png" {
		t.Errorf("expected png format, got %s", format)
	}
}

func TestTCPClipboardSyncer_Sync_ErrorStatusReturnsError(t *testing.T) {
	img := makeTestImage()
	pngData := testEncodePNG(img)

	addr, _, _ := startTestClipServer(t, constants.ClipboardStatusError)

	syncer := &TCPClipboardSyncer{
		Address: addr,
		Reader:  &mockClipboardReader{data: pngData},
	}

	err := syncer.Sync()
	if err == nil {
		t.Fatal("expected error for error status byte, got nil")
	}
	if !strings.Contains(err.Error(), "error status") {
		t.Errorf("expected error message to contain 'error status', got: %v", err)
	}
}

func TestTCPClipboardSyncer_Sync_PayloadExceeding50MBRejected(t *testing.T) {
	// To test the size check without generating a truly massive PNG (which
	// would be slow and memory-intensive), we use syncWithMaxPayload directly
	// with a lowered threshold.
	img := makeTestImage()
	pngData := testEncodePNG(img)
	reader := &mockClipboardReader{data: pngData}

	syncer := &TCPClipboardSyncer{
		Address: "127.0.0.1:0", // won't be reached
		Reader:  reader,
	}

	// Use the internal helper with a max payload of 1 byte to trigger the check.
	err := syncer.syncWithMaxPayload(1)
	if err == nil {
		t.Fatal("expected error for payload exceeding max, got nil")
	}
	if !strings.Contains(err.Error(), "exceeds maximum") {
		t.Errorf("expected error message to contain 'exceeds maximum', got: %v", err)
	}
}

func TestTCPClipboardSyncer_Sync_ReaderError(t *testing.T) {
	syncer := &TCPClipboardSyncer{
		Address: "127.0.0.1:0",
		Reader:  &mockClipboardReader{err: fmt.Errorf("no clipboard data")},
	}

	err := syncer.Sync()
	if err == nil {
		t.Fatal("expected error when reader fails, got nil")
	}
	if !strings.Contains(err.Error(), "reading clipboard image") {
		t.Errorf("expected error about reading clipboard, got: %v", err)
	}
}
