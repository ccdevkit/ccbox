# ccbox Development Guidelines

Auto-generated from all feature plans. Last updated: 2026-03-25

## Active Technologies

- Go 1.24 (toolchain go1.24.5) + `creack/pty` (Unix PTY), `aymanbagabas/go-pty` (Windows PTY), `google/uuid`, `ccdevkit/common` (settings library), `golang.design/x/clipboard`, `golang.org/x/term`, `golang.org/x/image` (WebP/BMP/TIFF decoding) (001-ccbox-rewrite)

## Project Structure

```text
cmd/
├── ccbox/           # Host CLI entry point
├── ccptproxy/       # Container command proxy (--setup / --exec modes)
│   └── matcher/     # CommandMatcher: exact first-word matching
├── ccclipd/         # Container clipboard daemon
└── ccdebug/         # Container debug log forwarder

internal/
├── args/            # CLI argument parsing: Parse(args, fs) → ParsedArgs with typed ClaudeArg
├── bridge/          # Container↔host communication: wire types, TCP server, routing
├── claude/          # Claude struct: New(sess), SetPassthroughEnabled, BuildRunSpec, ClaudeRunSpec + standalone utils: CaptureToken, DetectVersion, RedactToken
├── clipboard/       # Host clipboard access (wraps golang.design/x/clipboard)
├── constants/       # Shared constants (paths, env vars, defaults)
├── docker/          # Container + image lifecycle: ContainerSpec, RunContainer, CheckRunning, images
├── cmdpassthrough/  # Command passthrough: exec handler, path rewriting, proxy config, merge
├── session/         # Session lifecycle: Session struct (UUID + FileWriter + FilePassthrough), SessionFileWriter, TempDirProvider, FilePassthroughProvider, DockerBindMountProvider
├── logger/          # Debug logger with contextual prefixes, stderr/file output, secret redaction
├── settings/        # All settings: ccbox loader + Claude settings.json merger
└── terminal/        # Terminal I/O: PTY bridge, resize, stdin interception, clipboard sync
```

## Commands

# Add commands for Go 1.24 (toolchain go1.24.5)

## Code Style

Go 1.24 (toolchain go1.24.5): Follow standard conventions

## Recent Changes

- 001-ccbox-rewrite: Added Go 1.24 (toolchain go1.24.5) + `creack/pty` (Unix PTY), `aymanbagabas/go-pty` (Windows PTY), `google/uuid`, `ccdevkit/common` (settings library), `golang.design/x/clipboard`, `golang.org/x/term`

<!-- MANUAL ADDITIONS START -->
<!-- MANUAL ADDITIONS END -->
