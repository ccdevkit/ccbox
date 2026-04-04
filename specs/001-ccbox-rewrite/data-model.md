# Data Model: ccbox

**Feature**: ccbox — Docker-Sandboxed Claude Code Runner
**Date**: 2026-03-25

## Entities

### ContainerSpec

The specification for launching a Docker container. Built by the orchestrator and consumed by `docker.RunContainer()`.

```go
type ContainerSpec struct {
    ImageName string        // e.g., "ccbox-local:0.2.0-2.1.16"
    Mounts    []Mount       // Bind mounts (host→container path pairs)
    Env       []EnvVar      // Environment variables (some marked secret)
    Ports     []PortMapping // Host→container port mappings
    Args      []string      // Arguments passed to the container entrypoint
    Command   string        // Override entrypoint command (optional)
    WorkDir   string        // Container working directory
}

type Mount struct {
    Host      string // Host path
    Container string // Container path
    ReadOnly  bool   // ro or rw
}

type EnvVar struct {
    Key    string
    Value  string
    Secret bool // If true, value is redacted in debug logs
}

type PortMapping struct {
    Host      int
    Container int
}
```

**Relationships**: Built by the orchestrator from `ClaudeRunSpec` (args, env vars) + session providers (`TempDirProvider.Files` for session file mounts, `DockerBindMountProvider.Passthroughs` for file passthrough mounts) + bridge/TCP port + image name.

---

### ClaudeRunSpec

The docker-agnostic description of a claude CLI invocation. Produced by `claude.BuildRunSpec()`. Contains only what's needed to invoke the claude binary — no Docker concepts (mounts, images, ports). File needs are communicated separately via `Session.AddFilePassthrough()` calls during `BuildRunSpec`.

```go
// in internal/claude
type ClaudeRunSpec struct {
    Args []string  // CLI args to pass to claude binary (e.g., ["-p", "hello", "--append-system-prompt-file", "/opt/ccbox/ccbox-system-prompt.md"])
    Env  []EnvVar  // Environment variables needed by claude
}

type EnvVar struct {
    Key    string
    Value  string
    Secret bool // If true, value is redacted in debug logs
}
```

**Note**: `claude.EnvVar` is deliberately separate from `docker.EnvVar` to avoid coupling `internal/claude` to `internal/docker`. The orchestrator maps `claude.EnvVar` → `docker.EnvVar` when assembling `ContainerSpec`. The two types are structurally identical.

**Relationships**: Built by `claude.BuildRunSpec()`. Consumed by the orchestrator (`main.go`) which combines it with session provider data to build `docker.ContainerSpec`. The claude package also registers file passthroughs on the session during `BuildRunSpec` — CWD (rw), `~/.claude/` (rw), and file args from ParsedArgs (ro, with arg path rewriting).

---

### ParsedArgs

The fully deserialized CLI input. Produced by `args.Parse(os.Args[1:], fs)`. Consumed by the orchestrator (`main.go`) and passed to `claude.BuildRunSpec`, settings merge, and subcommand dispatch.

```go
type ParsedArgs struct {
    // ccbox flags
    Passthrough []string  // Command prefixes to route to host
    ClaudePath  string    // Path to claude CLI on host
    Use         string    // Pinned Claude Code version
    Verbose     bool      // Enable debug logging
    LogFile     string    // Debug log file path
    Version     bool      // Print version and exit
    Help        bool      // Print help and exit
    Subcommand  string    // "update", "clean", or "" (default run)

    // Claude args — typed so the caller knows which need mounts
    ClaudeArgs  []ClaudeArg
}

type ClaudeArg struct {
    Value  string // The original argument string
    IsFile bool   // true = this arg is a path to an existing file (needs mount)
}
```

**File detection**: `Parse` uses a `FileSystem` interface for testability:

```go
type FileSystem interface {
    Stat(path string) (os.FileInfo, error)
}
```

Candidates are identified via: (1) semantic awareness — flags known to take file values (e.g., `--system-prompt-file`) have their values checked, (2) heuristics — args that look like paths (start with `/`, `./`, `../`, `~/`), (3) confirmation — `FileSystem.Stat()` verifies existence. Non-existent paths get `IsFile: false`.

**Relationships**: Consumed by `claude.BuildRunSpec` (which registers file passthroughs on the session for `ClaudeArg` entries where `IsFile: true`, rewrites their paths to container paths in the returned `Args`, and builds the full CLI args and env vars), `main.go` (subcommand routing, settings merge).

---

### Settings

User-configurable settings loaded from `.ccbox/settings.{json,yaml,yml}` files. Loading is delegated to `ccdevkit/common/settings.Load()` — see [settings contract](contracts/settings.md).

```go
type Settings struct {
    Passthrough []string `yaml:"passthrough"` // Command prefixes to route to host
    ClaudePath  string   `yaml:"claudePath"`  // Path to claude CLI on host
    Verbose     bool     `yaml:"verbose"`     // Enable debug logging
    LogFile     string   `yaml:"logFile"`     // Debug log file path
}
```

**Tags**: Uses `yaml` tags only (not mixed `json`+`yaml`) to comply with `common/settings.ErrMixedTags` constraint. File format detection is by extension, not struct tags.

**Discovery & Merge**: Handled entirely by `common/settings`. See [settings contract](contracts/settings.md) for details.

---

### ProxyConfig

Configuration written via `Session.FileWriter` and mounted into the container for ccptproxy to read. The container path is determined by the `SessionFileWriter.WriteFile(containerPath)` argument (e.g., `/opt/ccbox/ccbox-proxy.json`).

**Container path**: `/opt/ccbox/ccbox-proxy.json` (written by host via session file writer, mounted into container).

```go
type ProxyConfig struct {
    HostAddress string   `json:"hostAddress"` // "host.docker.internal:PORT"
    Passthrough []string `json:"passthrough"` // Command prefixes to intercept
    Verbose     bool     `json:"verbose"`     // Enable debug logging in proxy
}
```

**Lifecycle**: Created per-session by the orchestrator, mounted read-only in the container.

---

### SessionFileWriter (Interface)

Abstraction for making files available inside the container. The orchestrator creates the appropriate implementation and passes it to session setup functions.

```go
type SessionFileWriter interface {
    WriteFile(containerPath string, data []byte) error
}
```

**Purpose**: Decouples file provisioning strategy from session logic. Currently implemented by `TempDirProvider` (writes to host temp dir, files mounted into container). Designed to be replaced by a FUSE-based implementation in the future (files served directly inside the container, no host-side writes).

**Lifecycle**: Created by the orchestrator in `main.go`, injected into `Session`. Consumers (e.g., `claude` package, orchestrator) use `Session.FileWriter` to make files available in the container.

---

### TempDirProvider

Implements `SessionFileWriter` by writing files to a host temp directory. The orchestrator is responsible for mounting these files into the container.

```go
type TempDirProvider struct {
    Dir   string        // /tmp/ccbox-{sessionID}/ (host side)
    Files []SessionFile // Populated by WriteFile
}

// Note: "SessionFile" here refers to ccbox-internal configuration files (e.g., settings.json,
// ccbox-proxy.json), NOT Claude Code's own session files managed via ~/.claude/.
type SessionFile struct {
    HostPath      string // Absolute path on the host
    ContainerPath string // Where the file should appear in the container
}
```

**Relationships**: Created per-session. `WriteFile` slugifies the container path to derive the host filename (e.g., `/opt/ccbox/settings.json` → `_opt_ccbox_settings.json`), writes to `{Dir}/{slug}`, and appends to `Files`. The orchestrator reads `Files` after session setup to generate container mounts.

**Cleanup**: `Dir` removed when ccbox exits.

---

### FilePassthroughProvider (Interface)

Abstraction for registering host files/directories that need to be available inside the container. Unlike `SessionFileWriter` (which creates files that don't exist on the host), file passthroughs reference existing host paths and provide two-way access (edits in the container reflect on the host, unless read-only).

```go
type FilePassthroughProvider interface {
    AddPassthrough(hostPath, containerPath string, readOnly bool) error
}

type FilePassthrough struct {
    HostPath      string // Absolute path on the host
    ContainerPath string // Where the file/dir should appear in the container
    ReadOnly      bool   // If true, container cannot modify the file
}
```

**Purpose**: Decouples file passthrough registration from mount implementation. Consumers (e.g., `claude` package) call `session.AddFilePassthrough()` to declare what host files/directories the session needs, without knowing how they'll be materialized (bind mounts, 9p, virtio-fs, etc.).

**Distinction from SessionFileWriter**: A session file is a file that **doesn't exist on the host** — it's written by a consumer (e.g., `settings.json` content generated in memory). A file passthrough is a file/directory that **already exists on the host** and needs to be accessible inside the container, with optional two-way synchronization.

**Lifecycle**: Created by the orchestrator in `main.go`, injected into `Session`. Consumers call `session.AddFilePassthrough()` to register host paths.

---

### DockerBindMountProvider

Implements `FilePassthroughProvider` by storing passthrough registrations. The orchestrator reads the accumulated passthroughs after session setup and converts them to `docker.Mount` entries.

```go
type DockerBindMountProvider struct {
    Passthroughs []FilePassthrough // Populated by AddPassthrough
}
```

**Method**: `AddPassthrough` appends a `FilePassthrough` entry to `Passthroughs`.

**Relationships**: Created per-session. Implements `FilePassthroughProvider`. The orchestrator holds a reference to the concrete `DockerBindMountProvider` type and reads `Passthroughs` directly to generate `docker.Mount` entries (no type assertion needed — orchestrator created it).

---

### Session

Ephemeral session state. Not related to Claude Code sessions.

```go
type Session struct {
    ID              string                  // UUID v4
    FileWriter      SessionFileWriter       // For session files (don't exist on host)
    FilePassthrough FilePassthroughProvider  // For host file/dir passthroughs (exist on host)
}

// Convenience method — delegates to FilePassthrough.AddPassthrough
func (s *Session) AddFilePassthrough(hostPath, containerPath string, readOnly bool) error
```

**Session files** (written via `Session.FileWriter` — files that don't exist on the host):
- `settings.json` — Claude Code settings override (bypassPermissions) — written by `claude` package
- `ccbox-system-prompt.md` — Injected system prompt (when command passthrough configured) — written by `claude` package
- `ccbox-proxy.json` — ProxyConfig for ccptproxy — written by orchestrator

**File passthroughs** (registered via `Session.AddFilePassthrough` — host files/dirs made available in container):
- CWD mount (rw) — registered by `claude` package, identity-path mapping
- `~/.claude/` mount (rw) — registered by `claude` package
- File args from ParsedArgs (ro) — registered by `claude` package, with arg path rewriting to container path

---

### ExecRequest (Wire Type)

Request from container ccptproxy to host TCP server to execute a command.

**Flow**: `ccptproxy` (container) → `bridge/server.go` (host)

```go
type ExecRequest struct {
    Type    string `json:"type"`    // Always "exec"
    Command string `json:"command"` // Shell command to execute
    Cwd     string `json:"cwd"`    // Working directory for execution
}
```

**Wire format**: JSON + newline, followed by TCP write-side close (CloseWrite).
**Response**: `{exit_code}\n{combined stdout+stderr}`, then TCP close.

---

### LogRequest (Wire Type)

Fire-and-forget log message from container to host.

**Flow**: `ccdebug` (container) → `bridge/server.go` (host)

```go
type LogRequest struct {
    Type    string `json:"type"`    // Always "log"
    Message string `json:"message"` // "[source] message text"
}
```

**Wire format**: JSON + newline, then TCP close. No response.

---

### ClipboardMessage (Wire Type)

Binary message from host to container clipboard daemon.

```
[4 bytes: length (big-endian uint32)] [N bytes: PNG data]
```

**Response**: `[1 byte: 0x00 success | 0x01 error]`
**Constraints**: Maximum payload 50 MB. Wire format is always PNG; the host-side syncer accepts common image formats (PNG, JPEG, GIF, WebP, BMP, TIFF) and transcodes to PNG before transport. Animated formats (GIF, WebP) are silently flattened to the first frame.

---

## State Transitions

### Docker Image Lifecycle

```
No image → EnsureLocalImage() → Cached image
Cached image → Version change detected → Rebuild (remove old auto-update, build new)
Cached image → ccbox clean → Removed (except latest auto-update)
Pinned image (--use) → ccbox clean → Removed
Pinned image (--use) → Auto-update → NOT removed (only auto-update images are auto-cleaned)
```

### Session Lifecycle

```
ccbox start → NewTempDirProvider() → /tmp/ccbox-{UUID}/ created
           → NewDockerBindMountProvider()
           → NewSession(fileWriter, filePassthrough) → Session{ID, FileWriter, FilePassthrough}
           → claude.New(sess) → writes settings.json via FileWriter
           → if command passthrough:
               cmdpassthrough.WriteProxyConfig(sess, cfg)
               c.SetPassthroughEnabled(commands) → writes system prompt via FileWriter
           → c.BuildRunSpec(parsedArgs, settings) → ClaudeRunSpec{Args, Env}
               (also registers CWD, ~/.claude/, file args as file passthroughs on session)
           → orchestrator builds docker.ContainerSpec from:
               ClaudeRunSpec.Env → ContainerSpec.Env
               ClaudeRunSpec.Args → ContainerSpec.Args
               TempDirProvider.Files → session file mounts (ro)
               DockerBindMountProvider.Passthroughs → file passthrough mounts
               + image name, ports, working dir
Container exits → provider.Cleanup() → TempDir removed
```

### Container Lifecycle

```
ccbox start → CaptureToken() + GetClaudeVersion() [parallel]
            → EnsureLocalImage() [if needed]
            → BuildRunSpec() → ClaudeRunSpec
            → orchestrator assembles docker.ContainerSpec
            → docker run --rm [-it|-i]
Container running → PTY bridge active, TCP server listening
User exits Claude / container exits → --rm auto-removes container
                                    → ccbox propagates exit code
```

## Validation Rules

| Entity | Field | Rule |
|--------|-------|------|
| Settings | Passthrough | Each entry must be a non-empty string |
| Settings | ClaudePath | If set, must be a valid path; validated at use time |
| ProxyConfig | HostAddress | Must be `host.docker.internal:{port}` format |
| ProxyConfig | Passthrough | Must be non-empty if proxy config exists |
| Session | ID | Must be valid UUID v4 |
| ExecRequest | Type | Must equal "exec" |
| ExecRequest | Command | Must be non-empty |
| LogRequest | Type | Must equal "log" |
| LogRequest | Message | Must be non-empty |
| ClipboardMessage | Length | Must be > 0 and ≤ 50MB |
| ContainerSpec | ImageName | Must be non-empty |
| ContainerSpec | Mounts | Must include CWD mount at identity path (registered by `claude` package via `session.AddFilePassthrough`; orchestrator converts to `ContainerSpec.Mounts`) |

## Implementation Notes (from Analysis Report)

### Constants to Define (internal/constants/constants.go)

The following constants should be defined during T003 implementation:

- **SystemPromptContainerPath**: `/opt/ccbox/ccbox-system-prompt.md` — stable container path for the injected system prompt file (referenced by `--append-system-prompt-file` CLI arg). Used by `claude` package when writing system prompt via FileWriter.
- **BaseImageRegistry**: `ghcr.io/ccdevkit/ccbox-base` — base Docker image registry path. Used by `docker/image.go` for `EnsureLocalImage`.
- **ContainerUserUID**: `1001` — UID for the `claude` user in the container (avoids conflict with Node.js UID 1000).
