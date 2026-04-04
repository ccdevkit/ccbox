# Research: ccbox Rewrite

**Feature**: ccbox — Docker-Sandboxed Claude Code Runner
**Date**: 2026-03-25

## Research Questions & Findings

### R-001: OAuth Token Capture Strategy

**Decision**: Ephemeral HTTP server that captures the Authorization header from Claude CLI.

**Rationale**: The ccyolo implementation proves this approach works reliably. Starting a fake HTTP server on `127.0.0.1:0`, launching `claude` with `ANTHROPIC_BASE_URL` pointing to it, and capturing the Authorization header from the first request is resilient to auth implementation changes in Claude Code.

**Alternatives considered**:
- Parsing Claude config files directly — fragile, ties to internal file formats
- Prompting user for token — poor UX, security concern
- Using Claude Code API keys — different auth model, not compatible with all subscription tiers

**Known limitation**: Token captures `subscriptionType: null`, causing Max subscribers to get Sonnet instead of Opus.

---

### R-002: PTY Bridge Architecture

**Decision**: Platform-abstracted PTY interface with build tags for Unix (creack/pty) and Windows (go-pty).

**Rationale**: Docker requires TTY allocation for interactive sessions. The PTY bridge must sit between the host terminal and the container's stdin/stdout. Unix and Windows have fundamentally different PTY APIs requiring separate implementations.

**Alternatives considered**:
- Direct `docker run -it` passthrough — doesn't support stdin interception for clipboard
- SSH tunnel into container — adds complexity, latency, auth overhead
- Named pipes — not cross-platform

**Implementation pattern** (from ccyolo):
```go
type PTY interface {
    io.ReadWriteCloser
    Resize(rows, cols uint16) error
    Wait() error
}
```
- Unix: SIGWINCH signal handler for immediate resize
- Windows: 250ms polling loop for resize detection

---

### R-003: Command Passthrough Security Model

**Decision**: PATH manipulation via hijacker scripts in `/opt/ccbox/bin/` (prepended to PATH), with prefix-based command matching.

**Rationale**: Shell scripts in PATH are more robust than aliases or functions — they survive subshells, work with `exec`, and are compatible with any calling convention. Prefix matching with word-boundary awareness (`"git"` matches `"git status"` but not `"gitk"`) prevents surprising behavior.

**Alternatives considered**:
- Shell aliases — don't survive subshells
- LD_PRELOAD — fragile, platform-specific, no Windows support
- Filesystem FUSE mount — extreme complexity for simple command routing

**Security note**: The host TCP server performs no command filtering. Security relies entirely on:
1. Localhost-only binding (127.0.0.1)
2. Container-side ccptproxy only sending configured commands
3. User-defined passthrough list

---

### R-004: Clipboard Bridge Mechanism

**Decision**: Two-channel approach: (1) TCP-based PNG transfer for Ctrl+V clipboard images, (2) file bridging via shared directory for pasted image paths.

**Rationale**: Docker containers can't access host clipboard directly. The TCP channel handles clipboard image data (requires CGO and platform-specific APIs), while the file bridge handles drag-drop and path paste scenarios (works everywhere).

**Alternatives considered**:
- X11 forwarding — adds X11 dependency to host, Windows incompatible
- Shared clipboard file — polling latency, race conditions
- gRPC streaming — overkill for single-shot image transfer

**Protocol**: Binary length-prefix format: `[4-byte big-endian uint32 length][PNG data]` → `[1-byte status (0x00 success)]`. Max 50MB.

---

### R-005: Container Image Strategy

**Decision**: Two-tier image system: remote base image (OS + tools) + locally-built overlay (Claude CLI version).

**Rationale**: The base image changes infrequently and contains ~500MB of system packages. The Claude CLI overlay is ~50MB and changes with each Claude Code release. Separating them means version upgrades only rebuild the small overlay layer.

**Alternatives considered**:
- Single monolithic image — every Claude version change rebuilds everything
- Volume-mount Claude CLI from host — breaks container isolation, path differences
- Sidecar container — unnecessary complexity

**Image naming**: `ccbox-local:{baseVersion}-{claudeVersion}` for auto-built images.

---

### R-006: Settings Discovery and Merge

**Decision**: Delegate to `ccdevkit/common/settings.Load(".ccbox/settings", cfg, nil)`. The `internal/settings` package is a thin wrapper defining the `Settings` struct — no custom discovery, parsing, or merge logic. Uses `yaml` struct tags only (matching ccyolo pattern) to avoid `common/settings.ErrMixedTags`.

**Rationale**: `ccdevkit/common/settings` already implements the exact walk-from-CWD-to-root discovery, extension priority (`.json` > `.yaml` > `.yml`), and merge semantics (primitives replace, arrays append, objects merge recursively). Reimplementing this would violate Constitution Principle V (No Over-Engineering).

**Merge precedence**: CLI flags (via `Options.AdditionalFiles`) > closest .ccbox/settings > ... > farthest .ccbox/settings > defaults

**Alternatives considered**:
- Custom settings implementation — duplicates proven common library, maintenance burden
- Single global config — no project-level customization
- Environment variables only — not persistent, hard to share with team
- XDG config directories — non-standard on macOS/Windows

---

### R-007: Session Management

**Decision**: Ephemeral UUID-based sessions for temp directory namespacing. Never pass `--session-id` to Claude Code.

**Rationale**: ccbox sessions are purely for organizing temporary config files (settings.json, proxy config, system prompt). Claude Code manages its own sessions via `-c`/`-r`/`--session-id` through the mounted `~/.claude/` directory.

**Alternatives considered**:
- Correlating ccbox and Claude sessions — creates coupling, complicates session flags
- Using PID as session ID — not unique across restarts
- No session concept — temp files could collide in concurrent usage

---

### R-008: Identity Path Mapping

**Decision**: Mount CWD at the same absolute path inside the container.

**Rationale**: Eliminates all path translation complexity. Any absolute path in Claude Code output or arguments works identically on both sides. The trade-off (container filesystem must accommodate arbitrary host paths) is acceptable since Docker handles this transparently.

**Alternatives considered**:
- Fixed mount point (e.g., /workspace) — requires bidirectional path rewriting everywhere
- Symlink-based mapping — fragile, security concerns
- Bind mount with path translation layer — adds complexity to every path operation

---

### R-009: Three Execution Paths

**Decision**: Three distinct container execution paths based on TTY availability and interceptor configuration.

**Rationale** (from ccyolo analysis):
1. **PTY mode** (TTY + interceptor): Full terminal with stdin interception for clipboard. Uses raw terminal mode, PTY bridge, resize signal handling.
2. **Interceptor mode** (no TTY + interceptor): Pipe-based stdin with interception but no terminal. For piped/scripted usage.
3. **Simple mode** (no interceptor): Direct docker stdin/stdout passthrough. Minimal overhead when no clipboard needed.

**Alternatives considered**:
- Single execution path with conditional logic — harder to reason about, more edge cases
- Always use PTY — breaks piped/scripted usage

---

### R-010: Wire Protocol Design

**Decision**: Per-request TCP connections with JSON-envelope discrimination for exec/log.

**Rationale**: One-shot TCP connections are simple, stateless, and avoid connection management complexity.

**Alternatives considered**:
- Persistent connections with multiplexing — complex, unnecessary for single-user
- Unix domain sockets — not available inside Docker for host communication
- HTTP — overhead for simple request/response
- gRPC — dependency overhead, complexity

---

### R-011: Auto-Update Image Cleanup (FR-037)

**Decision**: When a new auto-update image is built, the previous auto-update image is removed. Pinned images (from `--use`) are retained.

**Rationale**: Prevents unbounded disk usage from accumulated Docker images as Claude Code versions change. Pinned images are explicitly retained because the user chose that specific version.

**Implementation**: Track image origin (auto vs pinned) and remove previous auto image during `EnsureLocalImage()`.

---

### R-012: Clean Subcommand (FR-038)

**Decision**: `ccbox clean` removes all ccbox-managed Docker images except the latest auto-update image.

**Rationale**: Provides a manual cleanup mechanism for users who have accumulated pinned images or want to reclaim disk space.

**Implementation**: List images matching `ccbox-local:*` pattern, identify the latest auto-update, remove all others.
