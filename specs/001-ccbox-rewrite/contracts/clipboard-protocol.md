# Clipboard Bridge Protocol Contract

## Transport

- **Direction**: Host → Container
- **Port**: Dynamic, mapped identically on host and container (`CCBOX_CLIP_PORT`)
- **Connection model**: One connection per clipboard sync event

## Message Format

**Host → Container (ccclipd)**:
```
[4 bytes: length (big-endian uint32)]
[N bytes: PNG image data]
```

**Container → Host (response)**:
```
[1 byte: status]
  0x00 = success
  0x01 = error
```


## Constraints

| Constraint | Value |
|------------|-------|
| Maximum payload | 50 MB |
| Wire image format | PNG (host transcodes from JPEG, GIF, WebP, BMP, TIFF) |
| Connection model | One-per-request |

## Trigger Flow

1. User presses Ctrl+V in terminal
2. Stdin interceptor detects byte `0x16`
3. `TCPClipboardSyncer.Sync()` reads PNG from host clipboard
4. PNG sent to container via this protocol
5. ccclipd receives, validates, pipes to `xclip -selection clipboard -t image/png -i`
6. Returns status byte
7. Interceptor forwards `0x16` to container PTY
8. Claude Code reads image from container clipboard

## Platform Availability

| Platform | Architecture | Clipboard Sync |
|----------|-------------|----------------|
| macOS | ARM64 | Yes |
| macOS | x64 | Yes |
| Linux | x64 | Yes |
| Linux | ARM64 | No (CGO_ENABLED=0) |
| Windows | x64 | Yes |
| Windows | ARM64 | No (CGO_ENABLED=0) |

When clipboard sync is unavailable, `NoOpClipboardSyncer` is used (no-op on Ctrl+V). File path bridging (drag-drop) still works on all platforms.
