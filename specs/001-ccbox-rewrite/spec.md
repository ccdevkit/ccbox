# Feature Specification: ccbox — Docker-Sandboxed Claude Code Runner

**Feature Branch**: `001-ccbox-rewrite`
**Created**: 2026-03-25
**Status**: Draft
**Input**: User description: "Rewrite ccbox — a CLI tool that runs Claude Code inside a Docker container with auto-accept (YOLO mode) enabled, providing safety-net sandboxing, credential forwarding, clipboard bridging, and passthrough."

## User Scenarios & Testing *(mandatory)*

### User Story 1 - Run Claude Code in a Docker Container (Priority: P1)

A developer wants to run Claude Code autonomously without being prompted for permissions, while keeping their host machine safe from unintended file deletions or destructive commands. They type `ccbox` (or `ccbox -p "hello"`) and ccbox transparently launches Claude Code inside a Docker container with `bypassPermissions` enabled. The developer interacts with Claude Code exactly as they would natively — all arguments, flags, and session management work identically.

**Why this priority**: This is the core value proposition. Without containerized execution and permission bypassing, ccbox has no reason to exist. Every other feature depends on this working correctly.

**Independent Test**: Can be fully tested by running `ccbox -p "list files in current directory"` and verifying that Claude Code executes inside a container, produces output, and the host filesystem is not modified outside the mounted working directory.

**Acceptance Scenarios**:

1. **Given** Docker is running and Claude Code CLI is installed and authenticated, **When** the user runs `ccbox -p "hello"`, **Then** Claude Code runs inside a Docker container with `bypassPermissions` enabled, produces a response, and exits with the same exit code as Claude Code.
2. **Given** Docker is running, **When** the user runs `ccbox` with no arguments, **Then** an interactive Claude Code session starts inside the container with full terminal support (colors, resizing).
3. **Given** the user's current working directory is `/home/user/project`, **When** ccbox starts, **Then** the container mounts that directory at the same path and Claude Code operates on the same files.
4. **Given** Docker is not running, **When** the user runs `ccbox`, **Then** a clear error message is shown indicating Docker must be running.
5. **Given** Claude Code is not installed or not authenticated, **When** the user runs `ccbox`, **Then** a clear error message is shown indicating the prerequisite.

---

### User Story 2 - Credential Forwarding (Priority: P1)

A developer expects ccbox to "just work" with their existing Claude Code authentication. They should not need to manually configure tokens or re-authenticate inside the container. ccbox automatically captures the authentication credential from the host's authenticated `claude` CLI and injects it into the container environment.

**Why this priority**: Without authentication, Claude Code cannot function inside the container. This is a P1 prerequisite alongside containerized execution.

**Independent Test**: Can be fully tested by running `ccbox -p "who am I"` and verifying Claude Code responds successfully, confirming authentication was forwarded.

**Acceptance Scenarios**:

1. **Given** the host `claude` CLI is authenticated, **When** ccbox starts, **Then** the authentication credential is extracted from the host CLI and made available inside the container.
2. **Given** the host `claude` CLI is authenticated, **When** Claude Code runs inside the container, **Then** it authenticates successfully without any user interaction.
3. **Given** credential forwarding is active, **When** debug logging is enabled, **Then** the authentication credential value is redacted in all log output.

---

### User Story 3 - Command Passthrough to Host (Priority: P2)

A developer needs certain commands (like `git`, `docker`, `gh`) to run on the host machine rather than inside the container, because those commands depend on host-side credentials (SSH keys, Docker socket, GitHub auth). The developer configures passthrough commands via CLI flags (`-pt:git`) or a settings file, and matching commands are transparently routed from the container to the host.

**Why this priority**: Many real-world workflows require git operations with host SSH keys or Docker commands with host daemon access. Without passthrough, ccbox would be unusable for most development tasks involving version control or container management.

**Independent Test**: Can be fully tested by running `ccbox -pt:git -- -p "run git status"` and verifying the output reflects the host's git state (including branches, remotes) rather than the container's.

**Acceptance Scenarios**:

1. **Given** passthrough is configured for `git`, **When** Claude Code inside the container executes `git status`, **Then** the command runs on the host and the output is returned to Claude Code.
2. **Given** passthrough is configured for `git`, **When** Claude Code executes `gitk`, **Then** the command runs inside the container (matching requires the first word of the command to exactly equal the passthrough entry: `git` matches `git` and `git status` but not `gitk`).
3. **Given** passthrough is configured via both CLI flags and settings file, **When** ccbox starts, **Then** the passthrough lists are merged (appended), not replaced.
4. **Given** passthrough commands are configured, **When** a passthrough command runs on the host, **Then** the output includes a note: `[NOTE: This command was run on the host machine]`.
5. **Given** passthrough commands are configured, **When** Claude Code starts, **Then** a system prompt is injected informing Claude that certain commands run on the host.

---

### User Story 4 - Clipboard and Image Paste (Priority: P2)

A developer wants to paste images from their clipboard into a Claude Code session running inside ccbox, just as they would with native Claude Code. When they press Ctrl+V / Cmd+V with image data on the clipboard, ccbox bridges it into the container. They can also drag-drop or paste image file paths, which are rewritten to container-accessible paths.

**Why this priority**: Image input is a key Claude Code workflow for UI debugging, screenshot analysis, and visual context. Without clipboard bridging, this workflow breaks inside the container.

**Independent Test**: Can be fully tested by copying an image to the clipboard, pressing Ctrl+V in a ccbox session, and verifying Claude Code receives the image data.

**Acceptance Scenarios**:

1. **Given** the user has image data on their clipboard, **When** they press Ctrl+V in a ccbox session, **Then** the image is bridged into the container and made available to Claude Code.
2. **Given** the user pastes a file path like `./screenshot.png`, **When** the path points to an existing image, **Then** the file is copied to a shared bridge directory and the path is rewritten to the container-side path.
3. **Given** a pasted path is shell-escaped (e.g., `file\ name.png`), **When** ccbox processes it, **Then** the escaping is handled correctly and the file is bridged.
4. **Given** the user is on ARM64 Linux or ARM64 Windows, **When** they attempt clipboard paste, **Then** clipboard image paste is unavailable but file drag-drop still works.
5. **Given** the user pastes a URL (`https://example.com/image.png`), **When** ccbox processes input, **Then** the URL is NOT treated as a file path.

---

### User Story 5 - Project and Global Configuration (Priority: P2)

A developer wants to configure ccbox settings (like passthrough commands and claude CLI path) persistently rather than passing flags every time. They create a `.ccbox/settings.json` (or YAML) file in their project or home directory, and ccbox picks it up automatically with a clear precedence order.

**Why this priority**: Persistent configuration eliminates repetitive flag passing and enables team-wide defaults via checked-in project settings.

**Independent Test**: Can be fully tested by creating `.ccbox/settings.json` with `{"passthrough": ["git"]}`, running `ccbox -- -p "run git status"`, and verifying git runs on the host.

**Acceptance Scenarios**:

1. **Given** a `.ccbox/settings.json` exists in the project directory with `{"passthrough": ["git"]}`, **When** ccbox starts without CLI passthrough flags, **Then** `git` commands are routed to the host.
2. **Given** both project and global settings exist, **When** they define conflicting `claudePath` values, **Then** the project-level value takes precedence.
3. **Given** settings files exist at multiple levels, **When** they define `passthrough` arrays, **Then** the arrays are merged (appended) across all levels.
4. **Given** a settings file contains invalid JSON/YAML, **When** ccbox starts, **Then** the invalid file is silently ignored and other settings sources still apply.

---

### User Story 6 - Version Pinning (Priority: P3)

A developer wants to use a specific version of Claude Code inside the container, rather than whatever is installed on the host. They pass `--use 2.1.16` and ccbox builds/uses an image with that exact version.

**Why this priority**: Useful for reproducibility and debugging but not required for core functionality.

**Independent Test**: Can be fully tested by running `ccbox --use 2.1.16 -- --version` and verifying the reported Claude Code version matches.

**Acceptance Scenarios**:

1. **Given** the user passes `--use 2.1.16`, **When** ccbox starts, **Then** the container runs Claude Code version 2.1.16 regardless of the host-installed version.
2. **Given** the user does not pass `--use`, **When** ccbox starts, **Then** the container uses the same Claude Code version as the host CLI.

---

### User Story 7 - Debug Logging (Priority: P3)

A developer is troubleshooting a ccbox issue and wants to see what Docker commands are being run, how the container is configured, and what communication is happening between host and container. They enable verbose mode with `-v` or log to a file with `--log`.

**Why this priority**: Essential for troubleshooting but not for normal usage.

**Independent Test**: Can be fully tested by running `ccbox -v -- -p "hello"` and verifying debug output appears on stderr with contextual prefixes.

**Acceptance Scenarios**:

1. **Given** the user passes `-v`, **When** ccbox runs, **Then** debug messages are written to stderr with contextual prefixes.
2. **Given** the user passes `--log /tmp/debug.log`, **When** ccbox runs, **Then** debug messages are written to that file and verbose mode is implicitly enabled.
3. **Given** verbose mode is enabled, **When** credential-related data appears in logs, **Then** authentication credentials and other secrets are redacted.
4. **Given** verbose mode is enabled, **When** container-side log messages are produced, **Then** they are forwarded to the host and displayed with a `[container]` prefix.

---

### User Story 8 - Update Command (Priority: P3)

A developer wants to update Claude Code and have ccbox automatically rebuild its Docker image to match. They run `ccbox update` and ccbox handles both the host-side update and image rebuild.

**Why this priority**: Convenience feature that simplifies the upgrade workflow but can be done manually.

**Independent Test**: Can be fully tested by running `ccbox update` and verifying both the host CLI is updated and a new Docker image is built.

**Acceptance Scenarios**:

1. **Given** the user runs `ccbox update`, **When** the command executes, **Then** `claude update` runs on the host (not inside a container) and the local Docker image is rebuilt with the new version.

---

### Edge Cases

- What happens when the Docker daemon stops mid-session? The container terminates and ccbox should propagate the error exit code.
- How does ccbox handle the user's Docker being out of disk space? The image build fails with a clear error.
- What happens when the mounted working directory has restrictive permissions? The unprivileged container user may not be able to read/write files — this is expected and mirrors host permission behavior.
- What happens when the host Claude Code version changes between ccbox invocations? ccbox detects the version mismatch and rebuilds the local Docker image.
- What happens when multiple ccbox sessions run concurrently? Each session gets its own container and ephemeral session ID; there should be no interference.
- What happens when a passthrough command is not found on the host? The error output and non-zero exit code are returned to the container process identically to a local command failure — no special handling or annotation.
- What happens when the network drops mid-session? ccbox does not intervene; network resilience is handled by Claude Code inside the container.
- What happens when the `--` separator is omitted? All arguments are parsed as ccbox flags. To pass arguments to Claude Code, the `--` separator is required. For example, `ccbox -pt:git` configures passthrough but passes nothing to Claude; `ccbox -- -p "Hello"` passes `-p "Hello"` to Claude.
- What happens with the `--append-system-prompt` or `--append-system-prompt-file` flags? These are Claude Code flags (passed after `--`). ccbox makes referenced files available to the container via FR-022's semantic arg parsing, and always injects its own system prompt additively per FR-024.
- Does ccbox intercept text clipboard data? No — only image data is bridged from the host clipboard. Text clipboard is not intercepted.

## Requirements *(mandatory)*

### Functional Requirements

*Note: Gaps in FR numbering (e.g., FR-019, FR-023) reflect removed requirements — see [clarifications.md](clarifications.md) for rationale.*

- **FR-001**: System MUST run Claude Code inside a Docker container with `bypassPermissions` mode enabled, with no interactive permission prompts.
- **FR-002**: System MUST mount the user's current working directory into the container at the same path, so all file paths work identically inside and outside the container.
- **FR-003**: System MUST automatically extract the authentication credential from the host's authenticated `claude` CLI and make it available inside the container.
- **FR-004**: System MUST forward all `claude` CLI arguments and flags transparently to Claude Code inside the container.
- **FR-005**: System MUST propagate the container's exit code as ccbox's own exit code.
- **FR-006**: System MUST support a `--` separator to distinguish ccbox-specific flags from Claude Code arguments. Without `--`, all arguments are parsed as ccbox flags. The `--` separator is required to pass any arguments or flags to Claude Code.
- **FR-007**: System MUST support command passthrough, routing specified command prefixes from the container to the host for execution.
- **FR-008**: Passthrough matching MUST compare the command name against the passthrough list. A match occurs when the command exactly equals a passthrough entry (e.g., passthrough entry `git` matches the command `git` invoked as `git status`, but does not match the command `gitk`). Passthrough entries are single command names only — argument-level filtering is out of scope.
- **FR-008a**: Passthrough commands MUST execute on the host in the container process's current working directory at the time the command is invoked (not the static launch directory).
- **FR-009**: Passthrough lists from CLI flags and settings files MUST be merged (appended), not replaced.
- **FR-010**: System MUST bridge host clipboard image data into the container when the user presses Ctrl+V / Cmd+V, making it available to Claude Code. The host-side syncer MUST accept common image formats (PNG, JPEG, GIF, WebP, BMP, TIFF) and transcode to PNG before transport. Animated formats are silently flattened to the first frame.
- **FR-011**: System MUST detect image file paths in pasted input, copy the files to a shared bridge directory, and rewrite paths to container-accessible locations. Paths MUST have a path prefix (`/`, `./`, `../`, `~/`) on Unix, or (`C:\`, `.\`, `..\`) on Windows (build-tag gated) — bare filenames MUST NOT be detected. Supported image extensions: `.png`, `.jpg`, `.jpeg`, `.gif`, `.webp`. Multiple image paths in a single paste MUST be supported.
- **FR-012**: System MUST support project-level (`.ccbox/settings.json`) and global (`~/.ccbox/settings.json`) configuration files in JSON or YAML format, delegating discovery, parsing, and merge to `ccdevkit/common/settings.Load()`. Project-level settings MUST be discovered by walking up from the current working directory to the root; the closest file takes highest precedence. Errors reading or parsing settings files MUST be silently ignored. Implementation note: per Constitution Principle III, each silently-ignored error MUST include an explicit inline code comment justifying the decision (e.g., 'settings are optional; missing/invalid file is a valid state').
- **FR-013**: Settings precedence MUST follow: CLI flags > project settings > global settings > defaults. Primitive values (e.g., `claudePath`) at higher precedence replace lower ones. Objects are merged recursively (field-level merge). Arrays (e.g., `passthrough`) are appended across all levels.
- **FR-014**: System MUST cache a reusable container image and rebuild it only when ccbox or the Claude Code version changes.
- **FR-015**: System MUST automatically build the local Docker image on first use and rebuild it when ccbox or Claude Code is upgraded.
- **FR-016**: System MUST support version pinning via `--use <version>` to override automatic version detection.
- **FR-017**: System MUST support debug logging to stderr (`-v`) and to a file (`--log <path>`), with secrets redacted. `--log` MUST implicitly enable verbose mode.
- **FR-018**: System MUST intercept the `update` command and run it on the host, then rebuild the local Docker image.
- **FR-020**: System MUST run the container as an unprivileged (non-root) user and remove the container automatically after the session ends.
- **FR-021**: System MUST support PTY forwarding so terminal features (colors, resizing) work transparently.
- **FR-022**: System MUST semantically parse Claude CLI arguments to identify which arguments expect a file path (e.g., `--system-prompt-file <path>`, `--resume <path>`). For each such argument, if the referenced file exists on disk, the system MUST make it available to the container at the same path. Non-existent paths MUST be silently skipped. Path detection is semantic (based on the argument's expected type), not heuristic (no pattern-matching on path separators or file extensions).
**Architecture note — passthrough terminology**: This specification uses 'passthrough' in two distinct senses: (1) *Command passthrough* (FR-007, FR-008): routing container commands to the host for execution via the TCP bridge (`internal/cmdpassthrough` package); (2) *File passthrough* (FR-022): making host files and directories available inside the container via bind mounts (`session/FilePassthroughProvider`). These are separate mechanisms with different implementations.

- **FR-024**: System MUST inject a system prompt when passthrough commands are configured, informing Claude Code about the container environment and which commands run on the host. This injection MUST occur regardless of whether the user also passes `--append-system-prompt` or `--append-system-prompt-file` — ccbox's prompt is always additive.
- **FR-025**: System MUST be installable via `npm install -g @ccdevkit/ccbox`, with the npm package downloading a pre-built native binary for the user's platform.
- **FR-026**: System MUST support macOS (ARM64, x64), Linux (x64, ARM64), and Windows (x64, ARM64) platforms.
- **FR-027**: System MUST forward terminal capability information to the container so that color and formatting support is preserved.
- **FR-028**: System MUST mount `~/.claude/` read-write into the container so Claude Code sessions persist across invocations.
- **FR-029**: System MUST pre-set onboarding and permission acceptance flags in the container's Claude project config.
- **FR-030**: The passthrough proxy MUST be the only execution channel from the container to the host. The container MUST NOT have a mounted Docker socket, SSH back-channel, or any other mechanism to execute commands on the host outside of explicitly configured passthrough prefixes.
- **FR-031**: System MUST support a `--version` flag that prints the ccbox version and exits with code 0.
- **FR-032**: System MUST support a `--help` / `-h` flag that displays ccbox help text and exits with code 0.
- **FR-033**: The base Docker image MUST support both `linux/amd64` and `linux/arm64` architectures.
- **FR-034**: System MUST support a `--claudePath <path>` / `-c` CLI flag to specify the path to the `claude` CLI executable on the host, defaulting to `claude` in PATH.
- **FR-035**: System MUST forward Unix signals (SIGINT, SIGTERM, SIGHUP) received by the host ccbox process to the container process, wait for the container to exit, and propagate the container's exit code.
- **FR-036**: When a passthrough command fails on the host (including command-not-found), the proxy MUST return the error output and non-zero exit code to the container process identically to a local command failure — no additional error-specific annotation or special handling beyond the standard `[NOTE: This command was run on the host machine]` that accompanies all passthrough output.
- **FR-037**: When a new auto-update image is built (via version detection or `ccbox update`), the previous auto-update image MUST be automatically removed. Images created via `--use` (pinned) MUST NOT be auto-removed.
- **FR-038**: System MUST support a `ccbox clean` subcommand that removes all ccbox-managed Docker images except the latest auto-update image.

### Key Entities

- **Docker Container**: Ephemeral execution environment running Claude Code with bypassPermissions. Created per session, removed on exit.
- **Base Image**: Pre-built multi-architecture Docker image (`linux/amd64`, `linux/arm64`) containing common development tools and a runtime compatible with Claude Code. Pulled from a container registry.
- **Local Image**: Locally-built Docker image layered on top of the base image, containing a specific Claude Code version.
- **Authentication Credential**: Credential extracted from the host CLI and injected into the container (currently an OAuth token, but may support API keys in future). Must never appear in logs unredacted.
- **Settings File**: JSON or YAML configuration file (`.ccbox/settings.json`) discovered by walking up from the current directory. Supports project-level and global-level configuration.
- **Passthrough Command**: A command prefix configured to route execution from the container to the host, enabling host-side credential and daemon access.
- **Clipboard Bridge**: A mechanism that forwards host clipboard image data into the container, making it available to Claude Code.
- **Session ID**: An ephemeral identifier generated by ccbox for organizing temporary configuration files. Independent from Claude Code's session system. Cleaned up when the session ends.

## Success Criteria *(mandatory)*

### Measurable Outcomes

- **SC-001**: Users can start an interactive Claude Code session via ccbox in under 5 seconds on subsequent runs (after the first-time image build).
- **SC-002**: All Claude Code CLI flags and arguments work identically through ccbox as they do natively, with zero behavioral differences visible to the user (excluding known limitations).
- **SC-003**: Users can paste clipboard images into a ccbox session and Claude Code receives them within 2 seconds on supported platforms.
- **SC-004**: Passthrough commands return results to Claude Code indistinguishably from commands run inside the container, except for the host execution note.
- **SC-005**: ccbox can be installed from npm and used for a basic prompt within 3 minutes on a machine meeting prerequisites.
- **SC-006**: Session-related flags (`-c`, `-r`, `--session-id`) are forwarded to Claude Code and behave identically to native invocations — sessions persist across ccbox invocations via the mounted `~/.claude/` directory.
- **SC-007**: Users can run ccbox on all six supported platform/architecture combinations (macOS ARM64/x64, Linux x64/ARM64, Windows x64/ARM64) with consistent behavior within documented platform limitations.

## Assumptions

- ccbox does not manage Claude Code sessions. All session state is owned by Claude Code via the `~/.claude/` mount; session-related flags are forwarded transparently.
- Docker is installed and the Docker daemon is running on the user's machine.
- The Claude Code CLI (`claude`) is installed and authenticated on the host.
- npm is available for installation.
- The user's machine has sufficient disk space for Docker images (base image + local image).
- Network access is available for pulling the base Docker image on first use.
- The container has network access (ccbox is a safety net, not a security sandbox).
- The user trusts the repositories they are working with (ccbox does not protect against deliberately malicious agents).
- ARM64 Linux and ARM64 Windows users accept that clipboard image paste is unavailable; file drag-drop is the alternative.
- The first invocation of ccbox (or after a version upgrade) requires building a local Docker image, which is slower than subsequent cached runs.
- Auto-update images are automatically cleaned up when a newer auto-update image is built. Pinned images (created via `--use`) are retained until explicitly removed via `ccbox clean`.
- The base image registry is hardcoded and not configurable.
