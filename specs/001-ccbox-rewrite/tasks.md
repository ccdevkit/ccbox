# Tasks: ccbox — Docker-Sandboxed Claude Code Runner

**Input**: Design documents from `/specs/001-ccbox-rewrite/`
**Prerequisites**: plan.md, spec.md, data-model.md, research.md, quickstart.md, contracts/

**TDD**: Every task that produces code MUST follow Red-Green-Refactor. Tests are NOT separate tasks — each task includes writing the failing test, making it pass, and refactoring. This is non-negotiable per the project constitution (Principle VII).

**Task Granularity**: Each task MUST be small enough that the full TDD cycle (write failing test → implement → refactor) is a single coherent unit of work. If a task feels too large, split it. A good task produces one tested behavior.

**Organization**: Tasks are grouped by user story to enable independent implementation and testing of each story.

## Format: `[ID] [P?] [Story] Description`

- **[P]**: Can run in parallel (different files, no dependencies)
- **[Story]**: Which user story this task belongs to (e.g., US1, US2, US3)
- Include exact file paths in descriptions

---

## Phase 1: Setup (Shared Infrastructure)

**Purpose**: Project initialization, Go module, and directory structure

- [ ] T001 Initialize Go module with go.mod, install dependencies (`creack/pty`, `aymanbagabas/go-pty`, `google/uuid`, `ccdevkit/common`, `golang.design/x/clipboard`, `golang.org/x/term`, `golang.org/x/image`), create directory structure per plan.md (`cmd/ccbox/`, `cmd/ccptproxy/`, `cmd/ccptproxy/matcher/`, `cmd/ccclipd/`, `cmd/ccdebug/`, `internal/args/`, `internal/bridge/`, `internal/claude/`, `internal/clipboard/`, `internal/constants/`, `internal/docker/`, `internal/cmdpassthrough/`, `internal/logger/`, `internal/session/`, `internal/settings/`, `internal/terminal/`)
- [ ] T002 [P] Create Makefile with build, install, docker, run, clean targets per quickstart.md
- [ ] T003 [P] Create constants package with all shared constants (paths, env vars, defaults, port names, image name patterns) in internal/constants/constants.go

---

## Phase 2: Foundational (Blocking Prerequisites)

**Purpose**: Core data types and utility packages that ALL user stories depend on. MUST complete before any story work begins.

**⚠️ CRITICAL**: No user story work can begin until this phase is complete

- [ ] T004 [P] Implement wire protocol types (ExecRequest with fields: Type string, Command string, Cwd string; LogRequest with fields: Type string, Message string) with JSON marshal/unmarshal round-trip in internal/bridge/types.go + internal/bridge/types_test.go
- [ ] T005 [P] Implement CommandMatcher with exact first-word matching (FR-008: `git` matches `git status` but not `gitk`) in cmd/ccptproxy/matcher/matcher.go + cmd/ccptproxy/matcher/matcher_test.go (TDD: test Matches("git", "git status") == true, test Matches("git", "gitk") == false)
- [ ] T006 [P] Implement Settings struct and thin Load wrapper using `ccdevkit/common/settings.Load()` in internal/settings/settings.go + internal/settings/settings_test.go (TDD: test defaults when no files exist, test load from temp dir, test Settings struct fields have yaml tags and no json tags, test malformed settings file is silently ignored and defaults are returned)
- [ ] T007 [P] Implement Session struct with fields `ID string`, `FileWriter SessionFileWriter`, `FilePassthrough FilePassthroughProvider` and `NewSession(fw SessionFileWriter, fp FilePassthroughProvider)` with `AddFilePassthrough` convenience method in internal/session/session.go + internal/session/session_test.go (TDD: test NewSession returns valid UUID and stores injected FileWriter + FilePassthroughProvider, test AddFilePassthrough delegates to provider)
- [ ] T007a [P] Implement `SessionFileWriter` interface and `TempDirProvider` (creates temp dir, writes files, records host/container paths) in internal/session/provider.go + internal/session/provider_test.go (TDD: test WriteFile creates file on disk and appends to Files slice, test NewTempDirProvider creates temp dir, test Cleanup removes dir)
- [ ] T007b [P] Implement `FilePassthroughProvider` interface and `DockerBindMountProvider` (stores `[]FilePassthrough` entries, `AddPassthrough` appends to slice) in internal/session/passthrough.go + internal/session/passthrough_test.go (TDD: test AddPassthrough appends entry with correct host/container/readOnly fields, test multiple adds accumulate)
- [ ] T008 [P] Implement ContainerSpec, Mount, EnvVar, PortMapping structs, PTY interface definition, and `CheckRunning() error` (runs `docker info`, returns user-friendly error if Docker is not available) in internal/docker/docker.go + internal/docker/docker_test.go (TDD for CheckRunning only: test error message when docker command fails; no test for pure data types/interface)
- [ ] T038a [P] Implement `WriteProxyConfig(sess, config)` writing ccbox-proxy.json via `Session.FileWriter` in internal/cmdpassthrough/config.go + internal/cmdpassthrough/config_test.go (TDD: test writes correct JSON with hostAddress and passthrough list) (depends on T007a)

**Checkpoint**: Foundation ready — user story implementation can now begin

---

## Phase 3: User Story 1 — Run Claude Code in a Docker Container (Priority: P1) 🎯 MVP

**Goal**: User types `ccbox` or `ccbox -p "hello"` and Claude Code runs inside a Docker container with `bypassPermissions` enabled. Full terminal support, exit code propagation.

**Independent Test**: Run `ccbox -- -p "list files in current directory"` and verify Claude Code executes inside a container, produces output, and the host filesystem is not modified outside the mounted working directory.

Each task below includes TDD: write failing test → implement → refactor.

- [ ] T010 [P] [US1] Implement `args.Parse(args []string, fs FileSystem) (*ParsedArgs, error)` in internal/args/args.go + internal/args/args_test.go. Parse splits on `--` separator, parses ccbox flags (`--passthrough`/`-pt:CMD`, `--claudePath`, `--use`, `-v`, `--log`, `--version`, `--help`), detects subcommands (`update`, `clean`), and produces typed `[]ClaudeArg` where each arg is classified as string or file. File detection uses semantic awareness (knows `--system-prompt-file` takes a file value) + heuristics + `FileSystem.Stat()` confirmation. `FileSystem` interface abstracts filesystem for testability. (TDD: test `--` splitting, test `-pt:CMD` prefix parsing, test flag precedence, test `ClaudeArg.IsFile` set for existing file paths via mock FS, test semantic flag `--system-prompt-file` marks value as file, test non-existent paths get `IsFile:false`, test `-c /usr/local/bin/claude` sets ClaudePath)
- [x] T011 [P] [US1] ~~SUPERSEDED~~ — CWD, `~/.claude/`, and file arg mounts are now registered by the `claude` package via `session.AddFilePassthrough()` during `BuildRunSpec`. The orchestrator converts `DockerBindMountProvider.Passthroughs` → `docker.Mount` entries inline. No separate `GenerateMounts` function needed.
- [ ] T013 [P] [US1] Implement `CaptureToken` standalone function for authentication credential capture via ephemeral HTTP server (FR-003) in internal/claude/auth.go + internal/claude/auth_test.go (TDD: test capture server starts, receives Authorization header, returns token)
- [ ] T014 [P] [US1] Implement `DetectVersion` standalone function for Claude version detection from host CLI in internal/claude/version.go + internal/claude/version_test.go (TDD: test version parsing from `claude --version` output)
- [ ] T015 [P] [US1] Implement local image name formatting and version comparison in internal/docker/image.go + internal/docker/image_test.go (TDD: test `LocalImageName` formats `"{base}-{claude}"` tag)
- [ ] T016 [US1] Implement `EnsureLocalImage` with build-if-missing, version-mismatch detection, and auto-cleanup of previous auto-update image on rebuild (FR-037) in internal/docker/image.go + internal/docker/image_test.go (TDD: test image exists + matches → no build triggered, test image exists + mismatches → rebuild triggered, test image missing → build triggered, test previous auto-update image removed on rebuild per FR-037) (depends on T015)
  - **Note (I5)**: The generated Dockerfile for local image builds should be passed via stdin (`docker build -f - .`) rather than writing a temp file
  - **Note (I17)**: Docker's default layer caching is intentional — no `--no-cache` or `--pull`
- [ ] T017 [US1] Implement `BuildRunSpec(parsedArgs, settings)` returning `ClaudeRunSpec{Args, Env}` in internal/claude/claude.go + internal/claude/types.go + internal/claude/claude_test.go. `BuildRunSpec` also registers file passthroughs on the session: CWD mount (rw, identity-path), `~/.claude/` mount (rw), and file args from ParsedArgs where `IsFile:true` (ro, with arg rewriting to container path). (TDD: test Args includes all claude CLI args, test Env includes TERM/COLORTERM forwarding per FR-027 and OAuth token with Secret:true, test session.AddFilePassthrough called for CWD and ~/.claude/, test file arg paths rewritten to container paths in returned Args, test non-file args pass through unchanged) (depends on T010, T013, T007b)
- [ ] T018 [US1] Implement `claude.New(sess)` constructor that writes a session-scoped `settings.json` (bypassPermissions + onboarding/permission acceptance flags per FR-029) AND a `.claude.json` (with `has_completed_onboarding: true`, `hasCompletedOnboarding: true`, `permissions_accepted: true`) via `sess.FileWriter` immediately — these files are passed as CLI args / mounted into the container, never modifying the user's actual settings files. Internal session file helpers (`writeSettings`, `writeClaudeJson`, `writeSystemPrompt`) in internal/claude/claude.go + internal/claude/session_files.go + internal/claude/claude_test.go (TDD: test New writes settings.json with correct content via SessionFileWriter, test New writes .claude.json with onboarding flags via SessionFileWriter, test writeSystemPrompt produces markdown with command list) (depends on T007, T007a)
- [ ] T019 [P] [US1] Implement Unix PTY bridge (`pty_unix.go`) with `creack/pty` (build tag: `!windows`) in internal/terminal/pty_unix.go (no unit test — platform wrapper, tested via integration) (Include inline code comment justifying TDD exemption per Principle VII amendment)
- [ ] T020 [P] [US1] Implement Windows PTY bridge (`pty_windows.go`) with `go-pty` (build tag: `windows`) in internal/terminal/pty_windows.go (no unit test — platform wrapper) (Include inline code comment justifying TDD exemption per Principle VII amendment)
- [ ] T021 [P] [US1] Implement Unix resize handler (SIGWINCH) in internal/terminal/resize_unix.go (no unit test — signal wrapper) (Include inline code comment justifying TDD exemption per Principle VII amendment)
- [ ] T022 [P] [US1] Implement Windows resize handler (250ms polling) in internal/terminal/resize_windows.go (no unit test — polling wrapper) (Include inline code comment justifying TDD exemption per Principle VII amendment)
- [ ] T023 [US1] Implement `docker.RunContainer()` executing `docker run --rm` with PTY bridge, signal forwarding (FR-035), exit code propagation (FR-005), and unprivileged container execution (FR-020) in internal/docker/docker.go + internal/docker/docker_test.go (TDD: test ContainerSpec to docker args conversion, test `--rm` flag is present, test no Docker socket mount per FR-030, test signal forwarding per FR-035, test RunContainer returns container exit code to caller per FR-005) (depends on T008)
- [ ] T024 [US1] Implement thin subcommand dispatch wiring in cmd/ccbox/main.go: use `ParsedArgs.Subcommand` to route to `update`, `clean`, or default orchestration (depends on T010, T006)
- [ ] T025 [US1] Implement main orchestration: `docker.CheckRunning()` → parallel OAuth+Version → EnsureImage → create `TempDirProvider` + `DockerBindMountProvider` → `NewSession(fileWriter, filePassthrough)` → `claude.New(sess)` (writes settings.json) → if command passthrough: `cmdpassthrough.WriteProxyConfig(sess, cfg)` + `c.SetPassthroughEnabled(commands)` → `c.BuildRunSpec(parsedArgs, settings)` → orchestrator builds `docker.ContainerSpec` from `ClaudeRunSpec` + `TempDirProvider.Files` (session file mounts, ro) + `DockerBindMountProvider.Passthroughs` (file passthrough mounts) + image name + ports → `docker.RunContainer(spec)` in cmd/ccbox/main.go + cmd/ccbox/main_test.go (TDD for branching logic only: test version mismatch triggers rebuild, test auth failure returns exit code 1, test image build failure returns exit code 1; do NOT test simple call wiring — Docker-not-running is tested in T008) (depends on T007, T007a, T007b, T008, T013, T014, T016, T017, T018, T023, T024, T037, T038, T038a)
- [ ] T026 [US1] Create Dockerfile (multi-stage: Go binaries + Node.js runtime, multi-arch linux/amd64 + linux/arm64 per FR-033, verify no Docker socket mount per FR-030) and entrypoint.sh (container init: gosu for unprivileged user per FR-020, ccptproxy setup, Xvfb, ccclipd) (depends on T025) (Include inline code comment justifying TDD exemption per Principle VII amendment)
  - **Note (I3)**: Dockerfile MUST set `ENV PATH=/opt/ccbox/bin:/home/claude/.local/bin:$PATH`
  - **Note (I13)**: Create `claude` user with UID 1001 (not 1000) to avoid conflict with node user
  - **Note (I14)**: Install system packages: git, curl, ca-certificates, netcat-openbsd, gosu, openssh-client, jq, ripgrep, make, build-essential, python3, vim-tiny, xvfb, xclip

**Checkpoint**: User Story 1 complete — `ccbox -- -p "hello"` works end-to-end with container execution, PTY, and exit code propagation

---

## Phase 4: User Story 2 — Credential Forwarding (Priority: P1)

**Goal**: ccbox automatically captures the authentication credential from the host's authenticated `claude` CLI and injects it into the container environment. No manual re-authentication.

**Independent Test**: Run `ccbox -- -p "who am I"` and verify Claude Code responds successfully, confirming authentication was forwarded.

Each task below includes TDD: write failing test → implement → refactor.

- [ ] T027 [US2] Implement `RedactToken` standalone function for token redaction in debug log output (FR-017) and env injection with `Secret: true` flag in internal/claude/redact.go + internal/claude/redact_test.go (TDD: test token values are masked in log strings, test BuildRunSpec marks token env var as Secret:true in ClaudeRunSpec.Env) (depends on T013, T017)

**Checkpoint**: User Story 2 complete — authentication forwarding works and tokens are redacted in logs

---

## Phase 5: User Story 3 — Command Passthrough to Host (Priority: P2)

**Goal**: Commands like `git`, `docker`, `gh` configured via `-pt:CMD` or settings are transparently routed from the container to the host for execution, with host-side credential access.

**Independent Test**: Run `ccbox -pt:git -- -p "run git status"` and verify output reflects the host's git state.

Each task below includes TDD: write failing test → implement → refactor.

- [ ] T028 [P] [US3] Implement host TCP server: listener setup, connection accept, JSON type discrimination (exec vs log), and log handler (display with `[container]` prefix) in internal/bridge/server.go + internal/bridge/server_test.go (TDD: test server accepts connection, parses exec request JSON, dispatches to handler, test unknown type value is silently dropped, test malformed JSON is silently dropped)
- [ ] T029 [US3] Implement exec handler: command execution via `sh -c`, CWD from request (test dynamic CWD per FR-008a), combined stdout+stderr capture, exit code response in internal/cmdpassthrough/exec.go + internal/cmdpassthrough/exec_test.go (TDD: test successful command, test command failure passes error output and non-zero exit code through without special annotation per FR-036, test command-not-found, test CWD from request is used not static default, test response format is exactly `{exit_code}\n{output_bytes}`) (depends on T028)
- [ ] T030 [US3] Implement container path rewriting in exec output (`/home/claude` → host home) in internal/cmdpassthrough/exec.go + internal/cmdpassthrough/exec_test.go (TDD: test `rewriteContainerPaths` replaces paths correctly) (depends on T028, T029)
- [ ] T032 [P] [US3] Implement ccptproxy config reader (`ReadConfig` for ccbox-proxy.json) in cmd/ccptproxy/config.go + cmd/ccptproxy/config_test.go (TDD: test ReadConfig unmarshals host address and passthrough list)
- [ ] T033 [P] [US3] Implement hijacker script generator (creates shell script that routes command to host TCP) in cmd/ccptproxy/hijacker.go + cmd/ccptproxy/hijacker_test.go (TDD: test GenerateHijacker creates valid shell script with correct command name)
- [ ] T034 [US3] Implement ccptproxy TCP exec sender: sends ExecRequest JSON+newline, calls `conn.(*net.TCPConn).CloseWrite()` to signal end-of-request, reads exit code + output response in cmd/ccptproxy/proxy.go + cmd/ccptproxy/proxy_test.go (TDD: test sends ExecRequest JSON, test CloseWrite is called after write, test reads response with exit code and output, test response format is exactly `{exit_code}\n{output_bytes}`) (depends on T004)
- [ ] T035 [P] [US3] Implement ccptproxy TCP log sender (fire-and-forget LogRequest with 2s timeout) in cmd/ccptproxy/logging.go + cmd/ccptproxy/logging_test.go
- [ ] T036 [US3] Implement ccptproxy main with `--setup` mode (reads config, generates hijacker scripts for each passthrough command, prepends to PATH) and `--exec` mode (matches command, routes to host, prepends `[NOTE: This command was run on the host machine]\n` to exec output) in cmd/ccptproxy/main.go + cmd/ccptproxy/main_test.go (TDD: test `--setup` generates hijacker scripts for configured commands, test `--exec` routes matched command to TCP sender, test exec output is prepended with `[NOTE: This command was run on the host machine]\n`) (depends on T032, T033, T034, T005)
- [ ] T037 [US3] Implement `Claude.SetPassthroughEnabled(commands []string)`: stores commands, writes system prompt (FR-024) via `Session.FileWriter` immediately in internal/claude/claude.go + internal/claude/claude_test.go (TDD: test SetPassthroughEnabled writes system prompt containing command list, test empty commands is no-op) (depends on T018)
- [ ] T038 [US3] Implement passthrough list merge from CLI flags and settings (FR-009) in internal/cmdpassthrough/merge.go + internal/cmdpassthrough/merge_test.go — `ParsedArgs.Passthrough` provides CLI passthroughs, settings provides config passthroughs, `Merge` computes the final merged list (TDD: verify merge appends rather than replaces, test empty sources, test multiple sources) (depends on T010, T006)
**Checkpoint**: User Story 3 complete — passthrough commands route to host and execute with correct CWD

---

## Phase 6: User Story 4 — Clipboard and Image Paste (Priority: P2)

**Goal**: Users can paste images from clipboard (Ctrl+V) and drag-drop image file paths into ccbox sessions with automatic bridging into the container.

**Independent Test**: Copy image to clipboard, press Ctrl+V in ccbox session, verify Claude Code receives the image data.

Each task below includes TDD: write failing test → implement → refactor.

- [ ] T039 [P] [US4] Implement host clipboard access wrapper (read PNG from system clipboard) with build-tag gated NoOp for unsupported platforms (ARM64 Linux and ARM64 Windows, CGO_ENABLED=0) in internal/clipboard/clipboard.go (no unit test — thin platform wrapper around `golang.design/x/clipboard`) (Include inline code comment justifying TDD exemption per Principle VII amendment)
  - **Note (I18)**: Clipboard initialization MUST be wrapped in `defer/recover` to handle panics when built without CGO
- [ ] T040 [P] [US4] Implement image format transcoding: accept JPEG, GIF, WebP, BMP, TIFF and transcode to PNG (FR-010) in internal/terminal/clipboard_sync.go + internal/terminal/clipboard_sync_test.go (TDD: test each input format (JPEG, GIF, WebP, BMP, TIFF) transcodes to valid PNG via table-driven tests, test animated GIF flattens to first frame PNG)
- [ ] T041 [US4] Implement TCPClipboardSyncer: read clipboard image, transcode to PNG, send length-prefixed binary to container clipboard daemon, read 1-byte status response in internal/terminal/clipboard_sync.go + internal/terminal/clipboard_sync_test.go (TDD: test Sync sends correct 4-byte big-endian length + PNG payload, test Sync reads status byte and returns error on 0x01, test payload exceeding 50 MB is rejected with error) (depends on T040)
- [ ] T042 [US4] Implement NoOpClipboardSyncer for platforms without clipboard support in internal/terminal/clipboard_sync.go (no unit test — trivial no-op with zero branching logic, exempt per Principle VI). Depends on T041 for file ordering (shared clipboard_sync.go), not logical dependency. (Include inline code comment justifying TDD exemption per Principle VII amendment)
- [ ] T043 [US4] Implement stdin interceptor: detect Ctrl+V (byte 0x16), trigger clipboard sync, forward byte to container in internal/terminal/interceptor.go + internal/terminal/interceptor_test.go (TDD: test Read passes non-Ctrl+V data unchanged, test Ctrl+V triggers syncer) (depends on T041)
- [ ] T044 [US4] Implement bracketed paste detection and image path rewriting (FR-011): detect paths with prefix (`/`, `./`, `../`, `~/`), copy to bridge dir, rewrite to container path in internal/terminal/interceptor.go + internal/terminal/interceptor_test.go (TDD: test path detection, test shell-escaped paths, test URL not treated as path, test bare filename `screenshot.png` is NOT rewritten, test multiple paths in single paste are all rewritten, test Windows path patterns detected on Windows build) (depends on T043)
  - **Note (Decision 5, I21/I27)**: Bridge directory: create a bridge subdir in the session temp dir and register it as a file passthrough via `session.AddFilePassthrough()` (rw). Files are copied here and paths rewritten to container-side path. Auto-cleaned on exit.
  - **Note (Decision 6, I16)**: Add Windows path pattern detection (`C:\`, `.\`) via build-tagged `_windows.go` file alongside the Unix patterns
- [ ] T045 [US4] Implement ccclipd: container clipboard daemon that listens on TCP, receives length-prefixed PNG, pipes to `xclip`, writes 1-byte status response (0x00 success, 0x01 error) in cmd/ccclipd/main.go (depends on T041) (Include inline code comment justifying TDD exemption per Principle VII amendment)

**Checkpoint**: User Story 4 complete — clipboard image paste and file path bridging work in ccbox sessions

---

## Phase 7: User Story 5 — Project and Global Configuration (Priority: P2)

**Goal**: Users configure ccbox persistently via `.ccbox/settings.json` (or YAML) with project-level and global-level support and clear precedence.

**Independent Test**: Create `.ccbox/settings.json` with `{"passthrough": ["git"]}`, run `ccbox -- -p "run git status"`, verify git runs on host.

Each task below includes TDD: write failing test → implement → refactor.

- [ ] T046 [US5] Implement settings merge into CLI orchestration: load settings, merge passthrough/claudePath/verbose/logFile with `ParsedArgs` flags per precedence rules (FR-013). CLI flag overrides should be provided via `common/settings.Options.AdditionalFiles` to ensure correct merge semantics (arrays append, not replace). In cmd/ccbox/main.go + internal/settings/settings_test.go (TDD: test CLI flags override project settings, test arrays append not replace) (depends on T006, T010)
- [ ] T047 [US5] Implement Claude settings.json merge for container: read host `~/.claude/settings.json`, merge with ccbox overrides (bypassPermissions, system prompt) in internal/settings/claude_settings.go + internal/settings/claude_settings_test.go (TDD: test `MergeSettings` local overrides global primitives, preserves user custom settings) (depends on T018)

**Checkpoint**: User Story 5 complete — persistent configuration works with correct precedence

---

## Phase 8: User Story 6 — Version Pinning (Priority: P3)

**Goal**: Users can run a specific Claude Code version via `--use <version>`, independent of host-installed version.

**Independent Test**: Run `ccbox --use 2.1.16 -- --version` and verify reported version matches.

Each task below includes TDD: write failing test → implement → refactor.

- [ ] T048 [US6] Implement `--use` flag integration: override detected version, build/use image with pinned version, skip auto-cleanup for pinned images (FR-037) in internal/docker/image.go + internal/docker/image_test.go (TDD: test pinned image name differs from auto-update, test pinned images not auto-removed) (depends on T016)

**Checkpoint**: User Story 6 complete — version pinning works

---

## Phase 9: User Story 7 — Debug Logging (Priority: P3)

**Goal**: Users can enable debug logging via `-v` or `--log <path>` for troubleshooting, with secrets redacted and container logs forwarded.

**Independent Test**: Run `ccbox -v -- -p "hello"` and verify debug output on stderr with contextual prefixes.

Each task below includes TDD: write failing test → implement → refactor.

- [ ] T049 [P] [US7] Implement debug logger with contextual prefixes, stderr output, and file output modes in internal/logger/logger.go + internal/logger/logger_test.go (TDD: test verbose output goes to stderr, test --log writes to file, test --log enables verbose)
- [ ] T050 [US7] Add LogRequest handler to the T028 TCP server (internal/bridge/server.go) to receive and display container debug logs with `[container]` prefix on host stderr (depends on T028, T049)
- [ ] T051 [US7] Implement ccdebug container debug forwarder binary in cmd/ccdebug/main.go (depends on T035) (Include inline code comment justifying TDD exemption per Principle VII amendment)

**Checkpoint**: User Story 7 complete — debug logging works end-to-end

---

## Phase 10: User Story 8 — Update Command (Priority: P3)

**Goal**: `ccbox update` runs `claude update` on the host and rebuilds the Docker image.

**Independent Test**: Run `ccbox update` and verify host CLI is updated and new image is built.

Each task below includes TDD: write failing test → implement → refactor.

- [ ] T052 [US8] Implement `update` subcommand: run `claude update` on host, detect new version, rebuild local image, remove old auto-update image (FR-018, FR-037) in cmd/ccbox/main.go + internal/docker/image.go (TDD: test old auto-update image removal on rebuild) (depends on T016, T010)
- [ ] T053 [US8] Implement `clean` subcommand: list `ccbox-local:*` images, identify latest auto-update, remove all others (FR-038) in cmd/ccbox/main.go + internal/docker/image.go + internal/docker/image_test.go (TDD: test clean preserves latest auto-update image, test clean removes all other ccbox-managed images, test CleanImages function handles empty image list) (depends on T015)

**Checkpoint**: User Story 8 complete — update and clean subcommands work

---

## Phase 11: Polish & Cross-Cutting Concerns

**Purpose**: Improvements that affect multiple user stories

- [ ] T054 [P] Create npm distribution package (package.json, install.js, bin/ccbox shim) per plan.md npm/ structure (FR-025)
- [ ] T055 [P] Create GitHub Actions release workflow (.github/workflows/release.yml) for CI/CD: build matrix for all 6 platform/arch combos per FR-026 (macOS ARM64/x64, Linux x64/ARM64, Windows x64/ARM64), Docker push, GitHub Release, npm publish
  - **Note (I22)**: QEMU required for multi-arch Docker builds (linux/amd64 + linux/arm64)
  - **Note (I22)**: OIDC trusted publishing for npm (no NPM_TOKEN secret needed)
  - **Note (I22)**: Linux native builds require `libx11-dev` for CGO clipboard
  - **Note (I22)**: Native builds (CGO_ENABLED=1): macOS ARM64, macOS AMD64, Linux AMD64, Windows AMD64
  - **Note (I22)**: Cross-compiled builds (CGO_ENABLED=0): Linux ARM64, Windows ARM64 (no clipboard)
- [ ] T056 Implement `--version` and `--help` flag handlers (FR-031, FR-032) in cmd/ccbox/main.go (depends on T010)
- [ ] T057 Run quickstart.md validation: verify all commands from quickstart.md work correctly (depends on all previous phases)
- [ ] T058 [US1-8] Validate all 8 Independent Story Tests pass end-to-end: US1 (`ccbox -- -p "list files"`), US2 (`ccbox -- -p "who am I"`), US3 (`ccbox -pt:git -- -p "run git status"`), US4 (copy image → Ctrl+V → verify Claude receives), US5 (`.ccbox/settings.json` with passthrough), US6 (`ccbox --use 2.1.16 -- --version`), US7 (`ccbox -v -- -p "hello"`), US8 (`ccbox update`) (depends on all prior tasks)
- [ ] T059 [US1-8] Validate contract compliance and test strategy: cross-reference implementation against contracts (cli.md, tcp-protocol.md, clipboard-protocol.md, settings.md), verify test names match Test Strategy table in plan.md, verify Complexity Tracking exemptions have inline code comments (depends on T058)

---

## Dependencies & Execution Order

### Phase Dependencies

- **Setup (Phase 1)**: No dependencies — can start immediately
- **Foundational (Phase 2)**: Depends on Setup (T001) — BLOCKS all user stories
- **US1 (Phase 3)**: Depends on Foundational (Phase 2) completion
- **US2 (Phase 4)**: Depends on US1 T013, T017 (OAuth capture and run spec builder)
- **US3 (Phase 5)**: Depends on Foundational (T004, T005, T038a) — can run in parallel with US1 for server/proxy work
- **US4 (Phase 6)**: Depends on Foundational — can run in parallel with US1/US3
- **US5 (Phase 7)**: Depends on Foundational (T006) and US1 (T010 for args parsing)
- **US6 (Phase 8)**: Depends on US1 (T016 for image management)
- **US7 (Phase 9)**: Depends on US3 (T028 for TCP server with log handler)
- **US8 (Phase 10)**: Depends on US1 (T016, T010)
- **Polish (Phase 11)**: Depends on all user stories being complete

### User Story Dependencies

- **US1 (P1)**: Can start after Phase 2 — no other story dependencies
- **US2 (P1)**: Depends on US1's OAuth capture (T013) and run spec builder (T017)
- **US3 (P2)**: Can start after Phase 2 — protocol types and matcher are foundational
- **US4 (P2)**: Can start after Phase 2 — independent of other stories
- **US5 (P2)**: Depends on US1 args parsing (T010) for integration
- **US6 (P3)**: Depends on US1 image management (T016)
- **US7 (P3)**: Depends on US3 TCP server with log handler (T028)
- **US8 (P3)**: Depends on US1 image management (T016) and args parsing (T010)

### Within Each User Story

- Each task follows Red-Green-Refactor: failing test → minimum implementation → refactor
- Data types before logic
- Logic before integration
- Core implementation before CLI wiring

### Parallel Opportunities

**Phase 2** (all [P] tasks): T004, T005, T006, T007, T007a, T007b, T008, T038a (after T007a) — 7 active parallel tasks (T004, T005, T006, T007, T007a, T007b, T008) + T038a after T007a (T011 superseded)

**Phase 3 (US1)**: T010+T013+T014+T015 in parallel → T016 after T015 → T017 after T010+T013+T007b → T019+T020+T021+T022 in parallel → T023 after T008 → T024 after T010+T006 → T025 after T007+T007a+T007b+T008+T013+T014+T016+T017+T023+T024+T037+T038+T038a → T026 after T025

**Phase 5 (US3)**: T028+T032+T033+T035 in parallel → T029 after T028 → T030 after T029 → T034 after T004 → T036 after T032+T033+T034+T005

**Phase 6 (US4)**: T039+T040 in parallel → T041 after T040 → T042 after T041 → T043 after T041 → T044 after T043 → T045 after T041

---

## Execution Protocol (Agent Teams)

**⚠️ CRITICAL**: This section defines HOW tasks are executed. Follow this protocol exactly.

### Step 1: Create the Team

Use `TeamCreate` to create a team for the feature implementation:

```
TeamCreate({
  team_name: "ccbox-rewrite-impl",
  description: "Implementing ccbox — Docker-Sandboxed Claude Code Runner per tasks.md"
})
```

### Step 2: Create Tasks

Use `TaskCreate` to transform every task from this file into a team task. Each task description MUST include:

- The exact task ID and description from this file
- The TDD requirement: "Follow Red-Green-Refactor: write a failing test first, implement the minimum to pass, then refactor while green."
- File paths to create/modify
- Dependencies (which task IDs must complete first)

### Step 3: Spawn Teammates

**ONE TEAMMATE PER TASK. NO EXCEPTIONS.**

- Spawn a teammate using the `Agent` tool with `team_name` set to the team name.
- Assign exactly ONE task to each teammate via `TaskUpdate` (set `owner`).
- For tasks marked `[P]` with no unresolved dependencies, spawn teammates in parallel.
- For sequential tasks, wait for the blocking task's teammate to finish before spawning the next.

### Step 4: Teammate Lifecycle

Each teammate MUST:

1. Read its assigned task via `TaskGet`
2. Execute the task following Red-Green-Refactor
3. Mark the task as `completed` via `TaskUpdate`
4. **Shut down immediately** — send a shutdown acknowledgment and terminate

**Context rot prevention**: A teammate MUST NOT be reused for a second task. Once a teammate completes its task and shuts down, spawn a **new** teammate for the next task. This ensures every task starts with a fresh context window.

### Step 5: Orchestration Loop

The team lead (you) orchestrates:

```
while uncompleted tasks exist:
  1. Check TaskList for completed tasks
  2. For each newly completed task:
     - Verify the teammate has shut down
     - Check if any blocked tasks are now unblocked
  3. For each unblocked task without an owner:
     - Spawn a NEW teammate
     - Assign the task
  4. At each phase checkpoint:
     - Verify all phase tasks are complete
     - Run the full test suite
     - Only proceed to next phase if green
```

### Step 6: Cleanup

After all tasks are complete:

1. Run the full test suite one final time
2. Shut down any remaining teammates via `SendMessage` with `shutdown_request`
3. Clean up the team via `TeamDelete`

---

## Implementation Strategy

### MVP First (User Story 1 + 2 Only)

1. Complete Phase 1: Setup
2. Complete Phase 2: Foundational (CRITICAL — blocks all stories)
3. Complete Phase 3: User Story 1 (container execution)
4. Complete Phase 4: User Story 2 (credential forwarding)
5. **STOP and VALIDATE**: Run full test suite, test `ccbox -- -p "hello"` end-to-end
6. Deploy/demo if ready — this is the minimum viable product

### Incremental Delivery

1. Complete Setup + Foundational → Foundation ready
2. Add US1 + US2 → All tests green → Deploy/Demo (MVP!)
3. Add US3 (passthrough) → All tests green → Deploy/Demo
4. Add US4 (clipboard) → All tests green → Deploy/Demo
5. Add US5 (settings) → All tests green → Deploy/Demo
6. Add US6 (version pinning) → All tests green → Deploy/Demo
7. Add US7 (debug logging) → All tests green → Deploy/Demo
8. Add US8 (update/clean) → All tests green → Deploy/Demo
9. Each story adds value without breaking previous stories

### Parallel Team Strategy

Using the agent team protocol above:

1. Team lead completes Setup + Foundational (parallel teammates for [P] tasks)
2. Once Foundational is done, spawn parallel teammates:
   - Teammate A: First [P] task of User Story 1
   - Teammate B: First [P] task of User Story 3 (independent)
   - Teammate C: First [P] task of User Story 4 (independent)
3. As each teammate finishes and shuts down, spawn new teammates for the next tasks in each story
4. Stories complete and integrate independently

---

## Notes

- [P] tasks = different files, no dependencies — can be assigned to parallel teammates
- [Story] label maps task to specific user story for traceability
- Each user story should be independently completable and testable
- Every code task includes TDD — there are no separate "write tests" tasks
- Tasks should be small: one tested behavior per task
- Commit after each task completes (teammate responsibility)
- Stop at any checkpoint to validate story independently
- One teammate per task, always — no reuse, no context rot
- Avoid: vague tasks, same file conflicts, cross-story dependencies that break independence
