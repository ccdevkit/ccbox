# ccbox Technical Specification

## 1. Tech Stack

### Language & Runtime

- **Host binary**: Go 1.24 (toolchain go1.24.5)
- **Container runtime**: Node.js 22 (slim) — required because Claude Code is a Node.js application
- **Container OS**: Debian-based (node:22-slim)
- **Build system**: GNU Make (local), GitHub Actions (CI/CD)

### Go Dependencies

| Module | Version | Purpose |
|--------|---------|---------|
| `github.com/creack/pty` | v1.1.24 | Unix PTY allocation and control |
| `github.com/aymanbagabas/go-pty` | v0.2.2 | Windows PTY allocation |
| `github.com/google/uuid` | v1.6.0 | Session UUID generation |
| `github.com/ccdevkit/common` | v0.1.0 | Shared settings library (file discovery, YAML/JSON merge) |
| `golang.design/x/clipboard` | v0.7.0 | Cross-platform clipboard access (requires CGO on macOS/Linux) |
| `golang.org/x/term` | v0.39.0 | Terminal raw mode, size detection |

Indirect: `golang.org/x/crypto`, `golang.org/x/sys`, `golang.org/x/image`, `golang.org/x/mobile`, `golang.org/x/exp`, `github.com/u-root/u-root`, `gopkg.in/yaml.v3`.

### Container System Packages

git, curl, ca-certificates, netcat-openbsd, gosu, openssh-client, jq, ripgrep, make, build-essential, python3, vim-tiny, xvfb, xclip.

---

## 2. Architecture Overview

ccbox is a transparent wrapper around Claude Code that runs it inside a Docker container with `bypassPermissions` enabled. The architecture has three layers:

```
┌─────────────────────────────────────────────────────────────────┐
│                         HOST MACHINE                            │
│                                                                 │
│  ┌──────────────┐    ┌───────────────┐    ┌──────────────────┐  │
│  │ ccbox CLI   │    │ TCP Server    │    │ Clipboard Syncer │  │
│  │ (main.go)    │    │ (hostexec)    │    │ (stdin package)  │  │
│  │              │    │               │    │                  │  │
│  │ - arg parse  │    │ - exec cmds   │    │ - reads host     │  │
│  │ - token cap  │    │ - fwd logs    │    │   clipboard      │  │
│  │ - image mgmt │    │ - statusline  │    │ - sends PNG data │  │
│  │ - PTY bridge │    │               │    │   to container   │  │
│  └──────┬───────┘    └───────▲───────┘    └────────┬─────────┘  │
│         │                    │                     │             │
│         │ docker run         │ TCP (127.0.0.1:N)   │ TCP (N)     │
│─────────┼────────────────────┼─────────────────────┼─────────────│
│         │                    │                     │             │
│  ┌──────▼────────────────────┼─────────────────────▼───────────┐ │
│  │                    DOCKER CONTAINER                         │ │
│  │                                                             │ │
│  │  ┌──────────┐  ┌──────────┐  ┌─────────┐  ┌────────────┐  │ │
│  │  │ Claude   │  │ ccproxy  │  │ ccdebug │  │ ccclipd    │  │ │
│  │  │ Code     │  │          │  │         │  │            │  │ │
│  │  │          │  │ hijacks  │  │ fwd log │  │ recv PNG → │  │ │
│  │  │ runs w/  │  │ cmds →   │  │ to host │  │ xclip      │  │ │
│  │  │ bypass   │  │ host TCP │  │         │  │            │  │ │
│  │  └──────────┘  └──────────┘  └─────────┘  └────────────┘  │ │
│  │                                                             │ │
│  │  entrypoint.sh: gosu → ccproxy --setup → Xvfb → ccclipd   │ │
│  └─────────────────────────────────────────────────────────────┘ │
└─────────────────────────────────────────────────────────────────┘
```

### Component Interactions

1. **ccbox CLI** captures the OAuth token, detects the Claude version, ensures the Docker image exists, generates session config files, and launches the container with a PTY bridge.
2. **TCP Server** (hostexec) runs on localhost and handles three message types from the container: command execution, log forwarding, and statusline rendering.
3. **Stdin Interceptor** sits between the terminal and the container PTY, detecting Ctrl+V and pasted image paths to bridge clipboard/files into the container.
4. **ccproxy** inside the container intercepts configured commands via PATH hijacking and forwards them to the host TCP server.
5. **ccclipd** receives clipboard PNG data over TCP and writes it to the container's X11 clipboard via xclip.
6. **ccdebug** forwards container-side log messages to the host TCP server for display.

---

## 3. File and Directory Structure

```
ccbox/
├── cmd/
│   ├── ccbox/
│   │   └── main.go              # Host CLI entry point: arg parsing, orchestration
│   ├── ccproxy/
│   │   ├── main.go              # Container command proxy: --setup and --exec modes
│   │   ├── proxy.go             # TCP exec request sender
│   │   ├── config.go            # Proxy config reader (ccbox-proxy.json)
│   │   ├── hijacker.go          # Hijacker script generator
│   │   ├── logging.go           # TCP log sender
│   │   └── matcher/
│   │       ├── matcher.go       # Matcher interface
│   │       ├── prefix.go        # PrefixMatcher implementation
│   │       └── prefix_test.go   # Matcher tests (21 cases)
│   ├── ccclipd/
│   │   └── main.go              # Container clipboard daemon
│   └── ccdebug/
│       └── main.go              # Container debug log forwarder
├── internal/
│   ├── args/
│   │   ├── args.go              # Claude arg processing, path detection, mount generation
│   │   └── args_test.go         # Path detection and mount tests
│   ├── clipboard/
│   │   └── clipboard.go         # Host clipboard access (wraps golang.design/x/clipboard)
│   ├── claude/
│   │   └── claude.go            # OAuth capture, container spec builder
│   ├── constants/
│   │   └── constants.go         # All shared constants (paths, env vars, defaults)
│   ├── docker/
│   │   ├── docker.go            # Container execution: RunSpec(), ContainerSpec
│   │   ├── pty.go               # PTY interface
│   │   ├── pty_unix.go          # Unix PTY (creack/pty)
│   │   ├── pty_windows.go       # Windows PTY (go-pty)
│   │   ├── resize_unix.go       # SIGWINCH-based resize
│   │   └── resize_windows.go    # Polling-based resize
│   ├── hostexec/
│   │   ├── server.go            # Host TCP server: exec, log, statusline handlers
│   │   ├── server_test.go       # Path rewriting tests
│   │   ├── settings.go          # Claude settings.json reader/merger
│   │   └── settings_test.go     # Settings tests
│   ├── image/
│   │   └── image.go             # Docker image lifecycle (version detect, build, cache)
│   ├── protocol/
│   │   └── protocol.go          # Wire types: ExecRequest, LogRequest
│   ├── session/
│   │   └── session.go           # Ephemeral session: UUID, temp dir, config file writers
│   ├── settings/
│   │   └── settings.go          # ccbox settings loader (.ccbox/settings.{json,yaml,yml})
│   └── stdin/
│       ├── interceptor.go       # Stdin proxy: Ctrl+V, bracketed paste, file bridging
│       ├── interceptor_test.go  # Interceptor tests
│       └── clipboard_sync.go    # TCPClipboardSyncer and NoOp implementations
├── Dockerfile                   # Multi-stage: Go binaries + Node.js runtime
├── entrypoint.sh                # Container init: gosu, ccproxy setup, Xvfb, ccclipd
├── Makefile                     # Local dev: build, install, docker, run
├── npm/
│   ├── package.json             # @ccdevkit/ccbox npm wrapper
│   ├── install.js               # Postinstall: downloads Go binary from GitHub Releases
│   └── bin/
│       └── ccbox               # Node.js shim that spawns the Go binary
├── docs/
│   ├── model-selection.md       # Design doc: OAuth token vs subscription tier
│   └── session-id-handling.md   # Design doc: session ID decoupling
└── .github/
    └── workflows/
        └── release.yml          # CI/CD: build, Docker push, GitHub Release, npm publish
```

---

## 4. Data Flows

### 4.1 Startup Flow

```
1. Parse CLI args
   ├── splitArgs("--" separator) → ccbox flags vs claude args
   ├── extractPassthroughArgs() → -pt:/-passthrough: prefixes
   └── flag.Parse() → remaining ccbox flags

2. Load settings
   └── Walk .ccbox/settings.{json,yaml,yml} from cwd to root
       └── Merge: CLI flags > closer file > farther file > defaults
           (Primitives replaced, objects merged recursively, arrays appended)

3. Parallel goroutines:
   ├── CaptureToken()
   │   ├── Start ephemeral HTTP server on 127.0.0.1:0
   │   ├── Launch `claude` with ANTHROPIC_BASE_URL pointing to local server
   │   ├── Capture Authorization header from first request
   │   └── Kill claude process, return token
   └── GetClaudeVersion()
       ├── Run `claude --version`
       └── Parse "X.Y.Z (Claude Code)" → "X.Y.Z"

4. EnsureLocalImage(baseVersion, claudeVersion)
   ├── Check docker image inspect ccbox-local:{base}-{claude}
   ├── If missing: docker build (FROM base, install Claude CLI)
   └── Return image name

5. Create session
   ├── Generate UUID, create temp dir
   └── Write config files:
       ├── settings.json (bypassPermissions + statusline nc command)
       ├── ccbox-proxy.json (host address, passthrough list, verbose)
       └── system-prompt.md (if passthrough configured)

6. Setup clipboard (if CGO available)
   ├── Initialize clipboard library
   ├── Find free port
   └── Create TCPClipboardSyncer

7. Start TCP server on 127.0.0.1:0

8. Build ContainerSpec
   ├── Mounts: ~/.claude/, ~/.ccbox/.claude.json, cwd, config files
   ├── Env: CLAUDE_CODE_OAUTH_TOKEN, TERM, COLORTERM, CCBOX_CLIP_PORT, DISPLAY
   ├── Ports: clipboard port mapping
   └── Args: --settings, --append-system-prompt[-file], user args

9. docker.RunSpec()
   ├── Detect TTY → choose execution path:
   │   ├── PTY mode (TTY + interceptor): raw terminal, stdin interceptor, PTY bridge, resize signals
   │   ├── Interceptor mode (no TTY + interceptor): stdin interceptor without PTY
   │   └── Simple mode (no interceptor): direct stdin/stdout passthrough
   ├── Build docker run command (-it or -i based on TTY)
   └── Propagate container exit code via ExitError type
```

### 4.2 Command Execution (Passthrough)

```
Claude Code runs "git status" inside container
  │
  ▼
/opt/ccbox/bin/git (hijacker script, first in PATH)
  │ Quotes args, calls: exec ccproxy --exec "git 'status'"
  ▼
ccproxy --exec "git 'status'"
  │ Loads ccbox-proxy.json
  │ PrefixMatcher: "git status" matches pattern "git"
  ▼
ProxyToHost(hostAddress, "git status")
  │ TCP connect → host.docker.internal:PORT
  │ Send: {"type":"exec","command":"git status","cwd":"/path"}\n
  │ CloseWrite()
  ▼
Host TCP Server (handleExec)
  │ sh -c "git status" (runs in host cwd)
  │ Capture combined output
  │ Reply: "0\ngit output..."
  ▼
ccproxy reads response
  │ Print "[NOTE: This command was run on the host machine]"
  │ Print command output
  │ Exit with host exit code
  ▼
Claude Code sees git output
```

### 4.3 Clipboard Sync (Ctrl+V)

```
User presses Ctrl+V in terminal
  │
  ▼
Stdin Interceptor detects byte 0x16
  │ Calls TCPClipboardSyncer.Sync()
  ▼
clipboard.ReadImage() → PNG bytes from host clipboard
  │
  ▼
TCP connect → localhost:CLIP_PORT (mapped to container)
  │ Send: [4-byte length (big-endian uint32)] [PNG data]
  ▼
ccclipd (inside container) receives connection
  │ Read length prefix, validate (>0, ≤50MB)
  │ Read PNG data via io.ReadFull
  │ Pipe to: xclip -selection clipboard -t image/png -i
  │ Reply: 0x00 (success)
  ▼
Interceptor forwards 0x16 byte to PTY unchanged
Claude Code can now access the image from clipboard
```

### 4.4 Statusline Bridge

```
Claude Code statusline output
  │ Piped through: nc -N host.docker.internal:PORT
  ▼
Host TCP Server receives raw bytes (not JSON-typed)
  │ Parse failure for exec/log → falls through to handleStatusline
  │ rewriteContainerPaths(): /home/claude → actual host home
  │ GetMergedSettings(): read ~/.claude/settings.json + cwd/.claude/settings.json
  │ Execute user's statusLine.command with rewritten data on stdin
  │ Return command output
  ▼
nc sends output back to Claude Code
```

### 4.5 Image Path Paste (Drag-and-Drop / Paste)

```
User pastes "/Users/me/screenshot.png" into terminal
  │
  ▼
Terminal sends: \x1B[200~ /Users/me/screenshot.png \x1B[201~
  │ (bracketed paste sequence)
  ▼
Stdin Interceptor buffers paste content
  │ processPastedContent():
  │   looksLikeImagePath("/Users/me/screenshot.png") → true
  │   os.Stat() confirms file exists
  │   bridgeFile(): copy to ~/.ccbox-bridge/screenshot.png
  │   Rewrite path → /home/claude/.ccbox-bridge/screenshot.png
  ▼
Modified paste forwarded to container PTY
Claude Code sees a valid in-container path
```

---

## 5. Protocol Design

### 5.1 Host TCP Server Protocol

**Transport**: TCP over IPv4, `127.0.0.1:0` (dynamic port).

**Connection model**: One-shot. Each message opens a new TCP connection, sends the request, optionally receives a response, and closes. No persistent connections or multiplexing.

**Message discrimination**: Try-parse strategy on the first line:
1. Parse as JSON with `type: "exec"` → exec handler
2. Parse as JSON with `type: "log"` → log handler
3. Anything else → statusline handler (raw bytes)

#### Exec Request

```
Container → Host:
  {"type":"exec","command":"<shell command>","cwd":"<directory>"}\n
  <TCP write-side close (CloseWrite)>

Host → Container:
  <exit_code>\n
  <combined stdout+stderr bytes>
  <TCP close>
```

- Commands execute via `sh -c` (Unix) or `cmd /C` (Windows)
- Working directory from request used, falls back to server cwd
- No stdin piping — commands are non-interactive

#### Log Request

```
Container → Host:
  {"type":"log","message":"[source] message text"}\n
  <TCP close>

No response.
```

- Fire-and-forget, connection closed after sending
- 2-second dial timeout, failures silently swallowed

#### Statusline Data

```
Container → Host:
  <raw JSON bytes from Claude Code's statusline>\n
  <possibly more bytes>

Host → Container:
  <output from user's statusline command>
  <TCP close>
```

- No JSON envelope — raw bytes discriminated by parse failure
- Path rewriting applied: `/home/claude` → actual host home dir

### 5.2 Clipboard Protocol

**Transport**: TCP, port mapped identically on host and container.

```
Host → Container:
  [4 bytes: length (big-endian uint32)] [N bytes: PNG data]

Container → Host:
  [1 byte: status (0x00 = success, 0x01 = error)]
```

- Maximum payload: 50 MB
- PNG-only, no content type negotiation
- Connection-per-request model

### 5.3 Wire Format Types

```go
// internal/protocol/protocol.go
type ExecRequest struct {
    Type    string `json:"type"`    // "exec"
    Command string `json:"command"`
    Cwd     string `json:"cwd"`
}

type LogRequest struct {
    Type    string `json:"type"`    // "log"
    Message string `json:"message"`
}
```

---

## 6. Docker Integration

### 6.1 Two-Tier Image Strategy

**Base image** (remote):
- Registry: `ghcr.io/ccdevkit/ccbox-base:{baseVersion}`
- Contains: OS, system packages, ccproxy, ccdebug, ccclipd, gosu, Xvfb, xclip
- Does NOT contain Claude CLI
- Multi-platform: `linux/amd64` and `linux/arm64`

**Local image** (built on-demand):
- Name: `ccbox-local:{baseVersion}-{claudeVersion}`
- Layers Claude CLI on top of the base image
- Generated Dockerfile passed via stdin (no temp files):
  ```dockerfile
  FROM ghcr.io/ccdevkit/ccbox-base:{baseVersion}
  USER root
  RUN curl -fsSL https://claude.ai/install.sh | bash -s -- {claudeVersion}
  RUN cp /root/.local/bin/claude /home/claude/.local/bin/claude \
      && chown claude:claude /home/claude/.local/bin/claude
  ```
- If `LocalImageName` is empty, falls back to using the base image directly (`ghcr.io/ccdevkit/ccbox-base:{version}`)
- Rebuilt when either ccbox version or Claude CLI version changes
- Cached locally — reused instantly when unchanged
- Uses Docker's default layer caching (no `--no-cache` or `--pull`) — mutable tags could result in stale layers

### 6.2 Dockerfile (Multi-Stage Build)

**Stage 1 — Builder** (`golang:1.24-alpine`):
- Compiles three static Go binaries with `CGO_ENABLED=0`:
  - `ccproxy` — command proxy
  - `ccdebug` — debug log forwarder
  - `ccclipd` — clipboard daemon

**Stage 2 — Runtime** (`node:22-slim`):
- Installs system packages (git, curl, jq, ripgrep, make, build-essential, python3, etc.)
- Installs X11 packages (xvfb, xclip) for clipboard support
- Creates `claude` user with UID 1001 (avoids conflict with node user UID 1000)
- Sets PATH: `/opt/ccbox/bin` (hijacker) → `/home/claude/.local/bin` (Claude CLI) → system PATH
- Copies Go binaries from builder stage
- Sets entrypoint to `entrypoint.sh`

### 6.3 Container Spec (`ContainerSpec`)

```go
type ContainerSpec struct {
    ImageName string
    Mounts    []Mount
    Env       []EnvVar
    Ports     []PortMapping
    Args      []string
    Command   string
    WorkDir   string
}
```

**Volume mounts**:

| Host Path | Container Path | Mode | Purpose |
|-----------|---------------|------|---------|
| `~/.ccbox/.claude.json` | `/home/claude/.claude.json` | rw | Onboarding flags, oauthAccount |
| `~/.claude/` | `/home/claude/.claude/` | rw | Full Claude config directory |
| `<cwd>` | `<cwd>` (identity mount) | rw | Working directory at same path |
| Session settings.json | `/tmp/ccbox-settings.json` | ro | Claude Code settings override |
| Session proxy config | `/tmp/ccbox-proxy.json` | ro | ccproxy configuration |
| `~/.ccbox-bridge/` | `/home/claude/.ccbox-bridge/` | rw | File bridge directory (clipboard images, dragged files) |
| System prompt file | `/home/claude/.ccbox-bridge/ccbox-system-prompt.md` | ro | Injected system prompt |
| Extra path args | Varies | Varies | Paths detected in claude args |

**Identity path mapping**: CWD is mounted at the same absolute path inside the container, so all path references work identically on both sides.

**Environment variables injected**:

| Variable | Value | Secret |
|----------|-------|--------|
| `CLAUDE_CODE_OAUTH_TOKEN` | Captured OAuth token | Yes |
| `TERM` | From host | No |
| `COLORTERM` | From host | No |
| `CCBOX_CLIP_PORT` | Dynamic free port | No |
| `DISPLAY` | `:99` | No |

**Container flags**: `docker run --rm [-it|-i]` — always auto-removed, interactive, optionally with TTY.

### 6.4 Entrypoint Script (`entrypoint.sh`)

```
1. Running as root:
   ├── Create /tmp/.X11-unix with sticky bit (1777)
   ├── Set CCBOX_NEEDS_SETUP=1
   └── exec gosu claude "$0" "$@"  (re-exec as claude user)

2. Running as claude user:
   ├── If CCBOX_NEEDS_SETUP && /tmp/ccbox-proxy.json exists:
   │   └── ccproxy --setup  (create hijacker scripts in /opt/ccbox/bin)
   ├── If CCBOX_CLIP_PORT set:
   │   ├── Start Xvfb :99 -screen 0 1024x768x24
   │   ├── export DISPLAY=:99
   │   └── Start ccclipd (background, piped through ccdebug)
   └── exec "$@"  (run claude with provided args)
```

### 6.5 Container Lifecycle

```
Host                                    Container
─────                                   ─────────
1. CaptureToken() gets OAuth token
2. GetClaudeVersion() detects version
3. EnsureLocalImage() builds if needed
4. Session writes config files
5. Start TCP server (127.0.0.1:0)
6. BuildContainerSpec()
7. RunSpec() → docker run --rm ...      → entrypoint.sh starts
                                        → gosu switches to claude user
                                        → ccproxy --setup (if proxy config)
                                        → Xvfb + ccclipd (if clipboard)
                                        → exec claude <args>
8. PTY bridges stdin/stdout             ↔ Claude CLI runs interactively
9. Container exits                      → --rm auto-removes container
10. Session cleanup (rm temp dir)
```

---

## 7. Build System and CI/CD

### 7.1 Local Build (Makefile)

| Target | Description |
|--------|-------------|
| `build` | `go build ./cmd/ccbox` → `bin/ccbox` |
| `install` | Copy binary to `/usr/local/bin/ccbox` |
| `uninstall` | Remove installed binary |
| `clean` | Remove `bin/` directory |
| `docker` | Build base image locally as `ghcr.io/ccdevkit/ccbox-base:dev` |
| `run` | Build then run with `$(ARGS)` |

Local builds do not inject version via ldflags. No test or lint targets.

### 7.2 CI/CD Pipeline (GitHub Actions)

**Trigger**: Push to `release/*` branches. Version extracted from branch name.

**Job dependency chain**:
```
version
  ├── build-docker (Buildx + QEMU, multi-platform)
  ├── build-binaries-native (4 matrix entries, CGO_ENABLED=1)
  └── build-binaries-cross (2 matrix entries, CGO_ENABLED=0)
        └── create-release (GitHub Release + artifacts)
              └── publish-npm (OIDC trusted publishing)
```

**Native builds** (CGO_ENABLED=1): macOS ARM64, macOS AMD64, Linux AMD64, Windows AMD64.
- Build flags: `-ldflags="-s -w -X main.Version=${VERSION}"`
- Linux requires `libx11-dev` for clipboard CGO.

**Cross-compiled builds** (CGO_ENABLED=0): Linux ARM64, Windows ARM64.
- No clipboard support on these platforms.

**Docker image**: Multi-platform (`linux/amd64`, `linux/arm64`), pushed to `ghcr.io/ccdevkit/ccbox-base:{version}` and `:latest`.

**npm publish**: Sets version via `npm version`, publishes `@ccdevkit/ccbox` with `--provenance` (OIDC, no NPM_TOKEN needed).

**Pipeline permissions**: `contents: write` (releases), `packages: write` (ghcr.io), `id-token: write` (OIDC).

### 7.3 npm Distribution

The npm package `@ccdevkit/ccbox` is a binary wrapper:

1. `postinstall` hook runs `install.js`
2. Detects platform/arch, maps to Go GOOS/GOARCH
3. Downloads `ccbox-{goos}-{goarch}.tar.gz` from GitHub Releases
4. Extracts to `npm/bin/`, sets executable permissions
5. Runtime shim (`npm/bin/ccbox`) spawns the Go binary with inherited stdio

### 7.4 Versioning

- Branch-driven: `release/X.Y.Z` → version `X.Y.Z`
- Injected at build time via `-X main.Version=${VERSION}`
- Synchronized across: GitHub Release tag (`v0.1.0`), Docker tag, npm version, binary self-report
- Default version: `"dev"` when not injected

---

## 8. Cross-Platform Considerations

### 8.1 PTY Abstraction

```go
type PTY interface {
    io.ReadWriteCloser
    Resize(rows, cols uint16) error
    Wait() error
}
```

| Aspect | Unix | Windows |
|--------|------|---------|
| Library | `creack/pty` | `aymanbagabas/go-pty` |
| Build tag | `!windows` | `windows` |
| Resize detection | `SIGWINCH` signal | Polling (250ms interval) |
| Resize application | `pty.InheritSize` | `pty.Resize(cols, rows)` — note swapped parameter order |
| Resize latency | Immediate | Up to 250ms |

### 8.2 Shell Command Execution

```go
func shellCommand(command string) *exec.Cmd  // in hostexec/server.go
```

- Unix: `sh -c "<command>"`
- Windows: `cmd /C "<command>"`

### 8.3 Clipboard Support

| Platform | Architecture | Clipboard Support | Mechanism |
|----------|-------------|-------------------|-----------|
| macOS | ARM64 | Yes | NSPasteboard (CGO) |
| macOS | AMD64 | Yes | NSPasteboard (CGO) |
| Linux | AMD64 | Yes | X11/xclip (CGO) |
| Linux | ARM64 | No | Cross-compiled, no CGO |
| Windows | AMD64 | Yes | Win32 API (CGO) |
| Windows | ARM64 | No | Cross-compiled, no CGO |

The `clipboard.Init()` function wraps `golang.design/x/clipboard` in a `defer/recover` block to gracefully handle panics when built without CGO.

### 8.4 Path Handling

- `looksLikePath()` detects Unix paths (`/`, `./`, `../`, `~/`, containing `/`, file extensions) and Windows paths (`C:\`, `.\` — Windows build only)
- Shell escape handling for special characters in filenames: spaces, parentheses, quotes, backslashes
- Tilde expansion to home directory

---

## 9. Internal Design Decisions and Trade-offs

### 9.1 Identity Path Mapping

CWD is mounted at the same absolute path inside the container (`{Host: cwd, Container: cwd}`). This eliminates path translation complexity — any absolute path works identically in both environments. The trade-off is that it requires the container filesystem to accommodate arbitrary host paths.

### 9.2 OAuth Token Capture via Fake HTTP Server

Rather than parsing config files or using Claude Code internals, ccbox starts an ephemeral HTTP server, launches Claude with `ANTHROPIC_BASE_URL` pointing to it, and captures the `Authorization` header. This is resilient to auth implementation changes but:
- Requires spawning a Claude process that's immediately killed
- Adds startup latency
- Token captures `subscriptionType: null`, causing Max subscribers to get Sonnet instead of Opus (documented in `docs/model-selection.md`)

### 9.3 Session Decoupling

ccbox's session UUID is purely for temp directory namespacing. It never passes `--session-id` to Claude Code, avoiding conflicts with Claude's own session management (`-c`/`--continue`, `-r`/`--resume`). The trade-off is inability to correlate ccbox debug logs with Claude sessions.

### 9.4 Hijacker Pattern (PATH Manipulation)

Command interception uses individual shell scripts in `/opt/ccbox/bin/` (prepended to PATH) rather than shell aliases or functions. This is more robust — it survives subshells, works with `exec`, and is compatible with any calling convention. The trade-off is PATH order dependency (if PATH is modified, hijacking breaks silently).

### 9.5 Prefix Matching Over Regex

Passthrough patterns use simple word-boundary-aware prefix matching (`"git"` matches `"git status"` but not `"gitk"`). Less flexible than regex but more predictable and harder to misconfigure.

### 9.6 Per-Request TCP Connections

Each exec/log/statusline message opens a new TCP connection. No persistent connections or connection pooling. Simple and stateless, but adds connection overhead for each operation.

### 9.7 No Explicit Docker Pull

The image system relies on Docker's implicit pull during `docker build` (via the `FROM` directive). Simplifies code but means pull failures surface as build errors rather than with a clear download-specific error message.

### 9.8 Passthrough Array Merge

The `ccdevkit/common` settings library appends arrays rather than replacing them. Passthrough commands from multiple settings files and CLI flags accumulate. A project adds its own passthroughs without losing global ones.

### 9.9 bypassPermissions Always Set

The generated container settings always include `permissions.defaultMode: "bypassPermissions"`. This is the core value proposition — the container sandbox replaces the permission prompt system.

---

## 10. Security Model

### 10.1 Container Isolation

- Claude Code runs inside a Docker container as an unprivileged user (`claude`, UID 1001)
- `gosu` handles root-to-user privilege transition (not `su`/`sudo`, avoiding TTY issues)
- Container is `--rm` (auto-removed on exit)
- The container replaces Claude Code's built-in permission system with `bypassPermissions` mode, relying on Docker isolation instead

### 10.2 Network Security

- Host TCP server binds to `127.0.0.1` only — not exposed to external network
- No authentication on the TCP protocol — security relies entirely on localhost binding
- Container reaches host via `host.docker.internal` (Docker's built-in DNS)
- Dynamic port allocation prevents port conflicts

### 10.3 Command Execution

- The exec handler runs arbitrary shell commands on the host via `sh -c`, inheriting the host user's full permissions
- Security relies on the container-side client (ccproxy) only sending authorized commands based on the passthrough configuration
- The host server performs no command filtering, validation, or sandboxing
- The passthrough list is configured by the user, not by external input

### 10.4 Secret Handling

- OAuth token is marked as `Secret: true` in `EnvVar`, causing it to be redacted (`***`) in debug logs
- A parallel `logArgs` slice is maintained for debug output, ensuring secrets never appear in logs
- Token is passed as a Docker environment variable (`CLAUDE_CODE_OAUTH_TOKEN`)

### 10.5 File System

- The working directory is mounted read-write (required for Claude Code to modify files)
- `~/.claude/` is mounted read-write (required for Claude state persistence)
- Generated config files are mounted read-only where possible
- Bridge directory (`~/.ccbox-bridge/`) is read-write for file transfer

### 10.6 Clipboard

- Clipboard daemon has no authentication — relies on Docker network isolation
- Maximum payload size enforced (50 MB) to prevent resource exhaustion
- PNG-only, no arbitrary code execution through clipboard data

### 10.7 Trust Boundaries

```
Trusted:
├── Host machine (full user permissions)
├── ccbox CLI (runs as host user)
├── TCP server (localhost-only, host user permissions)
└── Configuration files (user-written or user-generated)

Partially trusted:
├── Docker container (isolated but with mounted volumes)
├── Claude Code (runs with bypassPermissions inside container)
└── ccproxy (forwards commands based on configuration)

External:
├── Claude API (authenticated via captured OAuth token)
├── Docker Hub / ghcr.io (image source)
└── claude.ai/install.sh (Claude CLI installer)
```

### 10.8 Known Limitations

- Container has read-write access to `~/.claude/` and the working directory — a compromised container process could modify these
- OAuth token is extracted via a short-lived fake server but is then stored as a container environment variable
- The `bypassPermissions` mode disables all Claude Code safety prompts — the container boundary is the only protection
- PATH manipulation for hijacking can be defeated if PATH is modified inside the container
- The statusline path rewriter uses `strings.HasPrefix("/home/claude")`, which would also match paths like `/home/claudette`
