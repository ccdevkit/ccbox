//go:build darwin && !noclipboard

package clipboard

/*
#cgo CFLAGS: -x objective-c
#cgo LDFLAGS: -framework Foundation -framework Cocoa
#import <Foundation/Foundation.h>
#import <Cocoa/Cocoa.h>

// clipboard_read_any_image reads the first available image from the pasteboard,
// checking PNG, TIFF, and then any image type. Returns PNG-encoded bytes.
unsigned int clipboard_read_any_image(void **out) {
	NSPasteboard *pb = [NSPasteboard generalPasteboard];

	// Try PNG first (preferred).
	NSData *data = [pb dataForType:NSPasteboardTypePNG];
	if (data != nil && [data length] > 0) {
		NSUInteger siz = [data length];
		*out = malloc(siz);
		[data getBytes:*out length:siz];
		return (unsigned int)siz;
	}

	// Try TIFF (most common on macOS screenshots and copy operations).
	data = [pb dataForType:NSPasteboardTypeTIFF];
	if (data != nil && [data length] > 0) {
		// Convert TIFF to PNG.
		NSBitmapImageRep *rep = [NSBitmapImageRep imageRepWithData:data];
		if (rep == nil) {
			return 0;
		}
		NSData *pngData = [rep representationUsingType:NSBitmapImageFileTypePNG
		                                   properties:@{}];
		if (pngData == nil || [pngData length] == 0) {
			return 0;
		}
		NSUInteger siz = [pngData length];
		*out = malloc(siz);
		[pngData getBytes:*out length:siz];
		return (unsigned int)siz;
	}

	return 0;
}
*/
import "C"
import "unsafe"

// readImagePlatform reads image data from the macOS pasteboard, supporting
// both PNG and TIFF formats. Returns PNG-encoded bytes.
func readImagePlatform() ([]byte, error) {
	var data unsafe.Pointer
	n := C.clipboard_read_any_image(&data)
	if data == nil || n == 0 {
		return nil, nil
	}
	defer C.free(data)
	return C.GoBytes(data, C.int(n)), nil
}
