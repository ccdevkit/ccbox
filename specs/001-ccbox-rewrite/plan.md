# Implementation Plan: ccbox — Docker-Sandboxed Claude Code Runner

**Branch**: `001-ccbox-rewrite` | **Date**: 2026-03-25 | **Spec**: [spec.md](spec.md)
**Input**: Feature specification from `/specs/001-ccbox-rewrite/spec.md`

## Summary

ccbox is a CLI tool that transparently runs Claude Code inside a Docker container with `bypassPermissions` enabled. The host binary orchestrates authentication credential capture, Docker image management, PTY bridging, command passthrough via TCP, and clipboard/image bridging. Written in Go 1.24 with platform-specific PTY and clipboard support via build tags.

## Technical Context

**Language/Version**: Go 1.24 (toolchain go1.24.5)
**Primary Dependencies**: `creack/pty` (Unix PTY), `aymanbagabas/go-pty` (Windows PTY), `google/uuid`, `ccdevkit/common` (settings library), `golang.design/x/clipboard`, `golang.org/x/term`, `golang.org/x/image` (WebP/BMP/TIFF decoding for clipboard transcode); GIF decoding uses stdlib `image/gif` (first frame only)
**Storage**: N/A (ephemeral sessions via temp directories, Docker images cached locally)
**Testing**: `go test` with table-driven tests, `t.TempDir()` for filesystem isolation, interface mocks for clipboard
**Target Platform**: macOS (ARM64, x64), Linux (x64, ARM64), Windows (x64, ARM64)
**Project Type**: CLI tool
**Performance Goals**: <5s startup on cached runs (SC-001), <2s clipboard bridge (SC-003)
**Constraints**: Must work with CGO_ENABLED=0 on ARM64 Linux/Windows (no clipboard), Docker required
**Scale/Scope**: Single-user CLI, ~38 Go source files across 11 internal packages + 4 cmd packages

## Constitution Check

_GATE: Must pass before Phase 0 research. Re-check after Phase 1 design._

| Principle | Status | Notes |
|-----------|--------|-------|
| I. Simplicity Over Cleverness | PASS | Architecture follows ccyolo patterns — no novel abstractions |
| II. Explicit Over Implicit | PASS | Clear function signatures, typed parameters, context in errors |
| III. Fail Fast, Fail Clearly | PASS | Input validation at boundaries (Docker check, auth check, settings parse) |
| IV. Single Responsibility | PASS | Each package owns one domain (14 packages, each with clear scope) |
| V. No Over-Engineering | PASS | Rewrite follows proven ccyolo patterns, no speculative features |
| VI. Test What Matters | PASS | Table-driven tests for logic, skip platform-specific wrappers |
| VII. Red-Green-Refactor TDD | PASS | TDD order defined per component, skip reasons documented |

No violations. No entries needed in Complexity Tracking.

## Project Structure

### Documentation (this feature)

```text
specs/001-ccbox-rewrite/
├── plan.md              # This file
├── research.md          # Phase 0: decisions and rationale
├── data-model.md        # Phase 1: entities and structures
├── quickstart.md        # Phase 1: getting started guide
├── contracts/           # Phase 1: interface contracts
│   ├── cli.md           # CLI interface contract
│   ├── tcp-protocol.md  # Host TCP server wire protocol
│   ├── clipboard-protocol.md  # Clipboard bridge protocol
│   └── settings.md      # Settings file format
└── tasks.md             # Phase 2 output (/speckit.tasks)
```

### Source Code (repository root)

```text
cmd/
├── ccbox/
│   └── main.go              # Host CLI entry point: arg parsing, orchestration, --version (build-time ldflags injection), --help
├── ccptproxy/
│   ├── main.go              # Container command proxy: --setup and --exec modes
│   ├── proxy.go             # TCP exec request sender
│   ├── config.go            # Proxy config reader (ccbox-proxy.json)
│   ├── hijacker.go          # Hijacker script generator
│   ├── logging.go           # TCP log sender
│   └── matcher/
│       └── matcher.go       # CommandMatcher: exact first-word matching
├── ccclipd/
│   └── main.go              # Container clipboard daemon
└── ccdebug/
    └── main.go              # Container debug log forwarder

internal/
├── args/
│   └── args.go              # CLI argument parsing: Parse(args, fs) → ParsedArgs with typed ClaudeArg (string or file)
├── bridge/
│   ├── types.go             # Wire types: ExecRequest, LogRequest
│   └── server.go            # Host TCP server: listener, connection routing, log handler (implemented in Phase 5/US3, co-located with types)
├── claude/
│   ├── claude.go            # Claude struct: New(sess), SetPassthroughEnabled, BuildRunSpec (single invocation management)
│   ├── auth.go              # CaptureToken: standalone OAuth credential capture function
│   ├── version.go           # DetectVersion: standalone version detection function
│   ├── redact.go            # RedactToken: standalone token redaction utility
│   ├── types.go             # ClaudeRunSpec and EnvVar types (docker-agnostic)
│   └── session_files.go     # Claude-specific session file writers: writeSettings, writeSystemPrompt (called internally by Claude methods)
├── clipboard/
│   └── clipboard.go         # Host clipboard access (wraps golang.design/x/clipboard)
├── constants/
│   └── constants.go         # All shared constants (paths, env vars, defaults)
├── docker/
│   ├── docker.go            # Container execution: RunContainer(), ContainerSpec, PTY interface, CheckRunning()
│   └── image.go             # Docker image lifecycle: LocalImageName, EnsureLocalImage, version comparison
├── cmdpassthrough/
│   ├── exec.go              # Host command execution: exec handler, container path rewriting
│   ├── config.go            # WriteProxyConfig via Session.FileWriter
│   └── merge.go             # Merge CLI flags + settings command passthrough lists
├── session/
│   ├── session.go           # Session struct (UUID + FileWriter + FilePassthroughProvider) and NewSession
│   ├── provider.go          # SessionFileWriter interface + TempDirProvider implementation
│   └── passthrough.go       # FilePassthroughProvider interface + DockerBindMountProvider implementation
├── logger/
│   └── logger.go            # Debug logger with contextual prefixes, stderr/file output, secret redaction
├── settings/
│   ├── settings.go          # ccbox settings loader (.ccbox/settings.{json,yaml,yml})
│   └── claude_settings.go   # Claude settings.json reader/merger (MergeSettings)
└── terminal/
    ├── pty.go               # PTY implementations: Unix (creack/pty), Windows (go-pty)
    ├── pty_unix.go           # Unix PTY bridge (build tag: !windows)
    ├── pty_windows.go        # Windows PTY bridge (build tag: windows)
    ├── resize_unix.go        # SIGWINCH-based resize
    ├── resize_windows.go     # Polling-based resize
    ├── interceptor.go        # Stdin proxy: Ctrl+V, bracketed paste, file bridging
    └── clipboard_sync.go     # TCPClipboardSyncer and NoOp implementations

Dockerfile                    # Multi-stage: Go binaries + Node.js runtime; images named ccbox-local:{ccbox_version}-{claude_version} (e.g., ccbox-local:0.2.0-2.1.16)
entrypoint.sh                 # Container init: gosu (drop root to unprivileged user per FR-020), ccptproxy setup, Xvfb, ccclipd
Makefile                      # Local dev: build, install, docker, run
npm/
├── package.json              # @ccdevkit/ccbox npm wrapper
├── install.js                # Postinstall: downloads Go binary from GitHub Releases
└── bin/
    └── ccbox                 # Node.js shim that spawns the Go binary
.github/
└── workflows/
    └── release.yml           # CI/CD: build, Docker push, GitHub Release, npm publish
```

**Structure Decision**: Standard Go project layout with `cmd/` for CLI entry points and `internal/` for all business logic. Four binaries: `ccbox` (host), `ccptproxy` (container proxy), `ccclipd` (container clipboard daemon), `ccdebug` (container debug forwarder). Eleven internal packages, each with a single clear responsibility. `docker` defines the PTY interface (consumer-side); `terminal` provides implementations — wired via interface in `main.go`, no import cycle. `bridge` accepts handler functions via DI, keeping `cmdpassthrough` decoupled. `session` owns the session lifecycle (`Session` struct with UUID + `FileWriter` + `FilePassthroughProvider`) and defines both the `SessionFileWriter` interface (with `TempDirProvider` for session files that don't exist on the host) and the `FilePassthroughProvider` interface (with `DockerBindMountProvider` for host file/dir mounts); orchestrator creates both providers, injects them into `Session`, and passes `Session` to consumers. `claude` package has a stateful struct (`claude.New(sess)`) for managing a single `claude` invocation — it accumulates configuration (command passthrough commands), writes session files immediately as state changes, registers file passthroughs on the session (CWD, `~/.claude/`, file args), and produces a docker-agnostic `ClaudeRunSpec` via `BuildRunSpec()`. The orchestrator (main.go) converts ClaudeRunSpec to ContainerSpec inline — mapping Args, Env, and session file/passthrough Mounts into the docker-specific struct. OAuth credential capture (`CaptureToken`), version detection (`DetectVersion`), and token redaction (`RedactToken`) are standalone utility functions in the same package. `cmdpassthrough` encapsulates all host-side command passthrough functionality: exec handling, proxy config writing, and CLI/settings merge. The `update` subcommand shells out to `claude update` via `exec.Command`, captures exit code, then calls `EnsureLocalImage` with force-rebuild.

**Execution Paths**: TTY detection in `RunContainer` determines one of three execution paths: (1) **PTY mode** (TTY detected + interceptor): raw terminal mode, stdin interceptor for clipboard paste detection, PTY bridge between host and container, resize signal forwarding — uses `docker run -it`; (2) **Interceptor mode** (no TTY + interceptor): stdin interceptor active for paste detection but no PTY or raw terminal — uses `docker run -i`; (3) **Simple mode** (no interceptor): direct stdin/stdout passthrough with no clipboard support — uses `docker run -i`.

**Container Environment Variables**:

| Variable | Source | Purpose |
|----------|--------|---------|
| CLAUDE_CODE_OAUTH_TOKEN | CaptureToken (auth.go) | OAuth credential for Claude CLI authentication |
| TERM | Host env | Terminal type forwarding (FR-027) |
| COLORTERM | Host env | Color support forwarding (FR-027) |
| CCBOX_TCP_PORT | Host TCP server | Dynamic port for command passthrough bridge |
| CCBOX_CLIP_PORT | Host config | Clipboard daemon communication port |
| DISPLAY | Hardcoded `:99` | Xvfb virtual display for clipboard access |

The host TCP server binds to `127.0.0.1:0` (OS-assigned port); the actual port is passed to the container via `CCBOX_TCP_PORT`.

**Privilege Model**: entrypoint.sh uses `gosu` to drop from root to an unprivileged user before launching the Claude CLI (per FR-020).

**Docker Image Naming**: Docker images follow the naming convention `ccbox-local:{ccbox_version}-{claude_version}` (e.g., `ccbox-local:0.2.0-2.1.16`). Base image registry: `ghcr.io/ccdevkit/ccbox-base:{version}`.

## Test Strategy

| Component | Test Type | First Red Test | TDD | Skip Reason |
|---|---|---|---|---|
| matcher/matcher.go | Unit (go test) | `CommandMatcher.Matches("git", "git status") == true; Matches("git", "gitk") == false` | Yes | — |
| bridge/types.go | Unit (go test) | `ExecRequest JSON round-trip preserves fields` | Yes | — |
| settings/settings.go | Unit (go test) | `Load returns defaults when no files exist` | Yes | Thin wrapper over common/settings; test wiring only |
| settings/settings.go (merge) | Unit (go test) | `CLI flags override project settings; arrays append not replace` | Yes | — |
| settings/claude_settings.go | Unit (go test) | `MergeSettings local overrides global primitives` | Yes | — |
| args/args.go (Parse) | Unit (go test) | `Parse(["--", "-p", "hello"], fs) splits on separator and returns (*ParsedArgs, error); error returned for invalid input` | Yes | Uses FileSystem interface for testability |
| args/args.go (ClaudeArg) | Unit (go test) | `Parse(["-p", "--system-prompt-file", "/tmp/f.md"], fs) marks /tmp/f.md as IsFile:true` | Yes | — |
| session/passthrough.go | Unit (go test) | `DockerBindMountProvider.AddPassthrough records FilePassthrough with correct fields` | Yes | — |
| terminal/interceptor.go | Unit (go test) | `Read() passes non-Ctrl+V data through unchanged` | Yes | — |
| terminal/interceptor.go (paste) | Unit (go test) | `Detects bracketed paste and rewrites image paths (plural); bare filename NOT rewritten; URL not treated as path` | Yes | — |
| terminal/clipboard_sync.go | Unit (go test) | `TCPClipboardSyncer.Sync() sends length-prefixed PNG; reads status byte; rejects >50MB payload` | Yes | — |
| terminal/clipboard_sync.go (transcode) | Unit (go test) | `Each input format (JPEG, GIF, WebP, BMP, TIFF) transcodes to valid PNG; animated GIF flattens to first frame` | Yes | — |
| cmdpassthrough/exec.go | Unit (go test) | `ExecHandler uses CWD from ExecRequest, not static launch dir; rewriteContainerPaths replaces /home/claude/ (trailing slash to avoid matching /home/claudette) with host home; response format is {exit_code}\n{output_bytes}` | Yes | — |
| cmdpassthrough/config.go | Unit (go test) | `WriteProxyConfig calls FileWriter with correct JSON` | Yes | — |
| cmdpassthrough/merge.go | Unit (go test) | `Merge appends CLI and settings command passthrough lists` | Yes | — |
| claude/claude.go | Unit (go test) | `New(sess) returns Claude with session` | Yes | — |
| claude/auth.go | Unit (go test) | `CaptureToken starts server, receives Authorization header` | Yes | — |
| claude/version.go | Unit (go test) | `DetectVersion extracts version from claude --version output` | Yes | — |
| claude/redact.go | Unit (go test) | `RedactToken masks token values in log strings` | Yes | — |
| claude/claude.go (spec) | Unit (go test) | `BuildRunSpec returns correct Args and Env; registers file passthroughs on session; Env includes TERM/COLORTERM forwarding per FR-027` | Yes | — |
| claude/claude.go (passthrough) | Unit (go test) | `SetPassthroughEnabled writes system prompt via session FileWriter` | Yes | — |
| claude/session_files.go | Unit (go test) | `writeSettings calls FileWriter with bypassPermissions content` | Yes | — |
| session/session.go | Unit (go test) | `NewSession returns valid UUID and injects FileWriter + FilePassthroughProvider` | Yes | — |
| session/provider.go | Unit (go test) | `TempDirProvider.WriteFile creates file and records SessionFile` | Yes | — |
| ccptproxy/hijacker.go | Unit (go test) | `GenerateHijacker creates valid shell script for command` | Yes | — |
| ccptproxy/config.go | Unit (go test) | `ReadConfig unmarshals ccbox-proxy.json` | Yes | — |
| ccptproxy/main.go | Unit (go test) | `--setup generates hijacker scripts; --exec routes matched command to TCP sender; exec output prepended with NOTE annotation` | Yes | — |
| docker/image.go | Unit (go test) | `LocalImageName formats "{base}-{claude}" tag` | Yes | — |
| docker/image.go (cleanup) | Unit (go test) | `EnsureLocalImage removes previous auto-update image on rebuild; pinned images are not removed` | Yes | — |
| docker/image.go (clean) | Unit (go test) | `CleanImages removes all ccbox-managed images except latest auto-update` | Yes | — |
| docker/image.go (pinning) | Unit (go test) | `Pinned image name differs from auto-update; pinned images not auto-removed` | Yes | — |
| logger/logger.go | Unit (go test) | `Verbose output goes to stderr; --log writes to file and enables verbose; secrets redacted` | Yes | — |
| docker/docker.go (CheckRunning) | Unit (go test) | `CheckRunning returns error when docker not available` | Yes | — |
| docker/docker.go (spec) | Unit (go test) | `RunContainer converts ContainerSpec to correct docker args; returns container exit code per FR-005; signal forwarding relays SIGINT/SIGTERM to container per FR-035` | Yes | — |
| constants/constants.go | — | — | No | Pure constants, no logic |
| terminal/pty_unix.go | — | — | No | Platform-specific wrapper around creack/pty |
| terminal/pty_windows.go | — | — | No | Platform-specific wrapper around go-pty |
| terminal/resize_*.go | — | — | No | Platform-specific signal/polling, thin wrapper |
| clipboard/clipboard.go | — | — | No | Thin wrapper around golang.design/x/clipboard |
| cmd/ccbox/main.go | — | — | No | Orchestration wiring, tested via integration |
| cmd/ccclipd/main.go | — | — | No | Platform-specific daemon, TCP + xclip piping |
| cmd/ccdebug/main.go | — | — | No | Trivial log forwarder |
| Dockerfile | — | — | No | Infrastructure config, no testable logic |
| entrypoint.sh | — | — | No | Shell script, tested via integration |
| Makefile | — | — | No | Build config, no testable logic |
| npm/* | — | — | No | Distribution packaging, no Go logic |

**Red-green-refactor sequence**: Task generation (`/speckit.tasks`) MUST interleave "write failing test" steps before their corresponding implementation steps, not group all tests at the end.

## Complexity Tracking

| Item | Principle | Justification |
|------|-----------|---------------|
| PTY interface (`internal/docker/docker.go`) | V (No Over-Engineering: interfaces need two consumers) | PTY interface defined at the consumer (`docker`) per Go convention. `terminal` package provides platform-specific implementations via build tags. `docker.RunContainer()` consumes the interface without knowing the platform. |
| ContainerSpec structs (`internal/docker/docker.go`) | VII (TDD: all new code) | Pure data types with no logic or branching. Exempt per Principle VI: "trivial getters, standard library functions, and code with no branching logic SHOULD NOT be tested." Tested indirectly via orchestrator assembly using `claude.BuildRunSpec` output (T017, T025). |
| ClaudeRunSpec structs (`internal/claude/types.go`) | VII (TDD: all new code) | Pure data types with no logic or branching. Same exemption as ContainerSpec. Tested indirectly via `claude.BuildRunSpec` (T017). |
| PTY interface definition (`internal/docker/docker.go`) | VII (TDD: all new code) | Interface-only definition with no implementation logic. No behavior to test. |
| Unix PTY bridge (`internal/terminal/pty_unix.go`) | VII (TDD: all new code) | Thin platform-specific wrapper around `creack/pty`. No branching logic — delegates directly to library. Tested via integration. |
| Windows PTY bridge (`internal/terminal/pty_windows.go`) | VII (TDD: all new code) | Thin platform-specific wrapper around `go-pty`. No branching logic — delegates directly to library. Tested via integration. |
| Unix resize handler (`internal/terminal/resize_unix.go`) | VII (TDD: all new code) | Signal handler wrapper (SIGWINCH). Platform-specific OS interaction, not unit-testable. Tested via integration. |
| Windows resize handler (`internal/terminal/resize_windows.go`) | VII (TDD: all new code) | Polling-based resize detection. Platform-specific OS interaction, not unit-testable. Tested via integration. |
| Dockerfile + entrypoint.sh | VII (TDD: all new code) | Infrastructure configuration files, not Go code. No unit-testable logic. Validated via integration testing. |
| Host clipboard wrapper (`internal/clipboard/clipboard.go`) | VII (TDD: all new code) | Thin wrapper around `golang.design/x/clipboard` with build-tag gated NoOp. No branching logic beyond build tags. Platform-specific, not unit-testable. |
| ccclipd daemon (`cmd/ccclipd/main.go`) | VII (TDD: all new code) | Platform-specific daemon (TCP + xclip piping). Requires container environment with xclip. Tested via integration. |
| ccdebug forwarder (`cmd/ccdebug/main.go`) | VII (TDD: all new code) | Trivial log forwarder with no branching logic. TCP dial + write only. |
| Claude struct split (`internal/claude/`) | IV (Single Responsibility) | Claude struct manages a single invocation only. OAuth capture (`CaptureToken`), version detection (`DetectVersion`), and token redaction (`RedactToken`) extracted as standalone utility functions — each is a distinct concern used independently by the orchestrator. |
| T001 (module init + directory scaffolding) | VII (TDD: all new code) | No Go logic, no test needed. |
| Subcommand dispatch (`cmd/ccbox/main.go`) | VII (TDD: all new code) | Simple switch on `ParsedArgs.Subcommand` with no complex logic — tested via integration. |
| Matcher interface removed (`cmd/ccptproxy/matcher/`) | V (No Over-Engineering: interfaces need two consumers) | `CommandMatcher` is a concrete type. Matcher interface removed — only one consumer (ccptproxy routing). Table-driven testing works directly against concrete type. |
