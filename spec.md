# ccbox Product Specification

## 1. Overview

ccbox is a CLI tool that runs [Claude Code](https://docs.anthropic.com/en/docs/claude-code) inside a Docker container with auto-accept (YOLO mode) enabled. It acts as a transparent wrapper around the `claude` CLI: users invoke `ccbox` exactly as they would `claude`, and ccbox handles containerization, credential forwarding, and permission bypassing automatically.

### Problem Statement

Claude Code's interactive permission prompts slow down autonomous workflows. Running Claude Code with `--dangerously-skip-permissions` on the host exposes the user's machine to uncontrolled file and command execution. ccbox solves this by running Claude Code inside a Docker container where permission bypassing is safe — the container provides a boundary that limits the blast radius of unintended actions.

### Safety Model

ccbox is a **safety net, not a sandbox**. It protects against overzealous agents making destructive mistakes (deleting files, running dangerous commands), but it does not provide full security isolation against a deliberately malicious agent. The container has network access, mounted volumes, and forwarded credentials. Users should only use ccbox with repositories they trust.

---

## 2. Installation

### Prerequisites

- **Docker** must be installed and running
- **Claude Code CLI** (`claude`) must be installed and authenticated on the host machine

### Install via npm

```bash
npm install -g @ccdevkit/ccbox
```

This installs the `ccbox` command globally. The npm package downloads a pre-built native binary for the user's platform at install time.

### Supported Platforms

| Platform | Architecture | Clipboard Support |
|----------|-------------|-------------------|
| macOS    | ARM64 (Apple Silicon) | Yes |
| macOS    | x64 (Intel) | Yes |
| Linux    | x64 | Yes |
| Linux    | ARM64 | No |
| Windows  | x64 | Yes |
| Windows  | ARM64 | No |

ARM64 Linux and Windows builds lack clipboard image support because the clipboard library requires native compilation (CGO), which is not available for cross-compiled builds. File drag-drop still works on all platforms. Users needing clipboard support on ARM64 can build from source on a native ARM64 machine with CGO enabled.

---

## 3. CLI Usage

### Basic Syntax

```
ccbox [claude-args]
ccbox [ccbox-flags] -- [claude-args]
```

When no `--` separator is present, all arguments are passed directly to `claude` inside the container. The `--` separator allows ccbox-specific flags to be specified before the claude arguments.

### Common Usage Patterns

```bash
ccbox                          # Interactive mode
ccbox -p "hello"               # One-shot prompt
ccbox -c                       # Continue previous session
ccbox -r                       # Resume session picker
ccbox -- --help                # Show claude's help
ccbox --help                   # Show ccbox's help
```

All `claude` CLI flags and arguments are supported and forwarded to Claude Code inside the container.

### Exit Behavior

ccbox propagates the container's exit code. If Claude Code exits with code 1, `ccbox` exits with code 1. This makes ccbox compatible with scripting and CI pipelines. `ccbox --help` and `ccbox --version` exit with code 0.

---

## 4. ccbox-Specific Flags

ccbox flags must appear before the `--` separator.

| Flag | Short | Default | Description |
|------|-------|---------|-------------|
| `--verbose` | `-v` | `false` | Enable debug logging to stderr |
| `--log <path>` | | `""` | Write debug logs to a file (implies `--verbose`) |
| `--claudePath <path>` | `-c` | `claude` | Path to the claude CLI executable on the host |
| `--use <version>` | | | Use a specific Claude Code version (e.g., `2.1.16`) |
| `--passthrough:<cmd>` | `-pt:<cmd>` | | Run commands matching `<cmd>` prefix on the host (repeatable) |
| `--version` | | | Print ccbox version and exit |
| `--help` | `-h` | | Show ccbox help text |

### Flag Examples

```bash
ccbox -v -- -p "hello"              # Debug mode, prompt to claude
ccbox --log /tmp/debug.log -- -c    # Log to file, continue session
ccbox -pt:git -- -p "git status"    # Run git on host
ccbox -pt:git -pt:docker --         # Multiple passthroughs
ccbox --use 2.1.16 --               # Pin Claude Code version
ccbox -c /usr/local/bin/claude --   # Custom claude path
```

---

## 5. Features

### 5.1 Docker Sandboxing

ccbox automatically runs Claude Code inside a Docker container with `bypassPermissions` mode enabled. This means Claude Code can execute any tool (bash commands, file edits, etc.) without interactive permission prompts.

**Container behavior:**
- The container runs as an unprivileged user (`claude`, UID 1001)
- The user's current working directory is mounted into the container at the same path, so all file paths work identically inside and outside the container
- The container is automatically removed after the session ends (`--rm`)
- Terminal features (colors, resizing) work transparently through PTY forwarding

**Docker image management:**
- ccbox uses a two-tier image strategy: a pre-built base image from `ghcr.io/ccdevkit/ccbox-base` and a locally-built image that layers in the specific Claude Code version
- The local image is automatically built on first use and cached. It is rebuilt whenever ccbox or the Claude CLI is upgraded
- The local image tag format is `ccbox-local:{ccboxVersion}-{claudeVersion}`
- The base image supports both `linux/amd64` and `linux/arm64` architectures

**Container environment:**
- Includes common development tools: git, curl, jq, ripgrep, make, build-essential, python3, vim-tiny, openssh-client
- Node.js 22 runtime (required by Claude Code)
- X11 virtual framebuffer (Xvfb) and xclip for clipboard support

### 5.2 Credential Forwarding

ccbox automatically captures the user's OAuth token from the host's authenticated `claude` CLI and injects it into the container. No manual token configuration is required.

The following credentials and configuration are forwarded:
- OAuth authentication token (extracted from the host CLI at startup)
- Claude configuration directory (`~/.claude/`) is mounted read-write
- Claude project config (`.claude.json`) is mounted, with onboarding and permission acceptance flags pre-set

### 5.3 Command Passthrough

By default, all commands that Claude Code invokes run inside the container. Some commands need to run on the host machine — for example, `git` (to use host SSH keys and credentials), `docker` (to access the host Docker daemon), or `gh` (to use host GitHub authentication).

The passthrough feature routes specified commands from the container to the host for execution.

**Usage:**
```bash
# Via CLI flags
ccbox -pt:git -pt:docker -- -p "build and push the image"

# Via settings file
# .ccbox/settings.json
{
  "passthrough": ["git", "docker", "gh"]
}
```

**How it works:**
- Commands are matched by prefix: `-pt:git` matches `git`, `git status`, `git push origin main`, but NOT `gitk` or `github-cli`
- Multiple passthrough patterns can be specified
- Multi-word patterns are supported: `-pt:"npm publish"` matches `npm publish` but not `npm install`
- When a command is routed to the host, the output includes a note: `[NOTE: This command was run on the host machine]`
- The host command runs in the same working directory as the container process
- When passthrough commands are configured, a system prompt is injected to inform Claude Code that certain commands run on the host and there may be environment differences

**Merging behavior:** Passthrough lists from CLI flags and settings files are merged (appended), not replaced. A project can add its own passthroughs without losing global ones.

### 5.4 Clipboard and Image Support

ccbox bridges the host clipboard into the container, allowing users to paste images from their clipboard into Claude Code sessions.

**Clipboard paste (Ctrl+V / Cmd+V):**
- When the user presses Ctrl+V, ccbox reads any image data from the host clipboard
- The image (PNG format) is sent over TCP to a clipboard daemon inside the container
- The daemon writes the image to the container's X11 clipboard via `xclip`
- Claude Code can then reference the pasted image
- This is transparent to the user — clipboard paste "just works" as it does with native Claude Code

**File drag-drop and path pasting:**
- When the user pastes or drags a file path into the terminal, ccbox detects image file paths
- Paths must have a path prefix (`/`, `./`, `../`, `~/`) — bare filenames are not detected
- URLs (`http://`, `https://`) are explicitly excluded from path detection
- Single-path detection is tried first; multi-path splitting is used as a fallback
- Supported image extensions: `.png`, `.jpg`, `.jpeg`, `.gif`, `.webp`
- The file is copied from the host into a shared bridge directory accessible to the container
- The pasted path is rewritten to the container-side path so Claude Code can access the file
- Shell-escaped paths (e.g., `file\ name.png`) are handled correctly
- Multiple image paths in a single paste are supported

**Limitations:**
- Clipboard image support is unavailable on ARM64 Linux and ARM64 Windows (see Platform Support)
- File drag-drop works on all platforms (it only requires path rewriting, not clipboard access)
- Only image data is synced from clipboard; text clipboard is not intercepted

### 5.5 Version Pinning

The `--use <version>` flag allows specifying an exact Claude Code version to use inside the container, bypassing automatic detection from the host CLI.

```bash
ccbox --use 2.1.16 --
```

This is useful for:
- Testing with a specific Claude Code version
- Working when the host CLI version detection fails
- Reproducing behavior with a known version

### 5.6 Update Command

The `update` command is intercepted and run on the host instead of inside the container:

```bash
ccbox update
```

This runs `claude update` on the host, then automatically rebuilds the local Docker image with the updated Claude Code version.

### 5.7 Debug Logging

Debug logging provides visibility into ccbox's operations, including Docker commands, container startup, and host-container communication.

```bash
ccbox -v -- -p "hello"              # Log to stderr
ccbox --log /tmp/debug.log --       # Log to file (implies -v)
```

When verbose mode is enabled:
- Host-side debug messages are prefixed with context
- Container-side messages (from proxy, clipboard daemon, entrypoint) are forwarded to the host and displayed with `[container]` prefix
- Secret values (like OAuth tokens) are redacted in log output

### 5.8 Statusline Support

ccbox proxies Claude Code's statusline feature across the container boundary. If the user has a `statusLine.command` configured in their Claude Code settings (`~/.claude/settings.json` or `.claude/settings.json`), ccbox will:

1. Capture statusline data from Claude Code inside the container
2. Rewrite container paths to host paths in the data
3. Execute the user's configured statusline command on the host with the rewritten data

This allows custom status displays (like terminal status bars) to work transparently with ccbox.

### 5.9 System Prompt Injection

ccbox supports Claude Code's `--append-system-prompt` and `--append-system-prompt-file` flags. When passthrough commands are configured, ccbox also injects its own system prompt to inform Claude Code about the container environment and which commands run on the host.

If the user passes `--append-system-prompt` directly, ccbox does not inject its own, preserving the user's value.

File paths referenced by `--append-system-prompt-file` are automatically mounted into the container as read-only.

### 5.10 Path Handling

ccbox automatically detects filesystem paths in Claude arguments and mounts them into the container:

- Absolute paths (`/usr/local/bin`)
- Relative paths (`./config.yaml`, `../Makefile`)
- Tilde paths (`~/Documents`)
- Paths containing `/` (`src/main.go`)
- Paths with file extensions (`config.json`)

Bare `.`, `..`, and `~` without a path component are NOT treated as paths.

Detected paths that exist on disk are bind-mounted into the container at the same path, so absolute paths work identically inside and outside the container. Non-existent paths are silently skipped.

---

## 6. Configuration

### 6.1 Settings File

ccbox supports project-level and global settings files.

**File locations:**
- Project: `.ccbox/settings.json` (or `.yaml`/`.yml`) in the current directory or any ancestor directory
- Global: `~/.ccbox/settings.json` (or `.yaml`/`.yml`)

Settings files are discovered by walking up from the current working directory to the root. The closest file to the current directory takes highest precedence.

**Supported formats:** JSON, YAML (`.json`, `.yaml`, `.yml`)

### 6.2 Available Settings

| Setting | Type | Description |
|---------|------|-------------|
| `claudePath` | string | Path to the claude CLI executable (default: `claude` in PATH) |
| `passthrough` | string[] | List of command prefixes to run on the host instead of in the container |

### 6.3 Settings Priority

Settings are resolved with the following priority (highest to lowest):

1. **CLI flags** (`-c`, `-pt:`, etc.)
2. **Project settings** (`.ccbox/settings.json` closest to cwd)
3. **Global settings** (`~/.ccbox/settings.json`)
4. **Defaults** (`claudePath` = `"claude"`)

**Merge behavior:**
- Primitive values (e.g., `claudePath`): higher-precedence values replace lower ones
- Objects: merged recursively (field-level merge)
- Arrays (e.g., `passthrough`): **appended** across all levels, not replaced. This means a project can add its own passthrough commands without losing globally configured ones

Errors reading or parsing settings files are silently ignored — an invalid file in one location does not prevent loading from other locations.

### 6.4 Example Configurations

**Minimal (just passthrough git):**
```json
{
  "passthrough": ["git"]
}
```

**Advanced:**
```json
{
  "claudePath": "/usr/local/bin/claude",
  "passthrough": ["git", "docker", "gh", "npm"]
}
```

**YAML format:**
```yaml
claudePath: /usr/local/bin/claude
passthrough:
  - git
  - docker
  - gh
```

---

## 7. Session Management

ccbox does not manage Claude Code sessions. All session-related flags (`-c`/`--continue`, `-r`/`--resume`, `--session-id`) are passed directly to Claude Code inside the container.

Claude Code's session state is persisted in `~/.claude/`, which is mounted read-write into the container. This means sessions persist across ccbox invocations — continuing and resuming sessions works as expected.

ccbox generates its own ephemeral session ID internally, used only for organizing temporary configuration files. This ID is completely independent from Claude Code's session system and is cleaned up when the session ends.

---

## 8. Model Selection

Claude Code's model selection works through ccbox. Users can pass `--model` to select a specific model:

```bash
ccbox -- --model opus
```

**Known limitation:** Users with Claude Max subscriptions may receive Sonnet instead of Opus by default when using ccbox. This occurs because the OAuth token forwarding does not preserve subscription tier information (which is stored in the macOS Keychain on the host, inaccessible from Docker). The workaround is to explicitly pass `--model opus`.

---

## 9. Environment

### Environment Variables Forwarded to Container

| Variable | Purpose |
|----------|---------|
| `TERM` | Terminal type for color support |
| `COLORTERM` | Extended color support |
| `CLAUDE_CODE_OAUTH_TOKEN` | Authentication (captured automatically) |

### Container Environment

| Variable | Value | Purpose |
|----------|-------|---------|
| `DISPLAY` | `:99` | X11 display for clipboard support (Xvfb) |
| `CCBOX_CLIP_PORT` | Dynamic | Clipboard daemon communication port |

---

## 10. Requirements

- **Docker**: Must be installed and the Docker daemon must be running. The Docker base image supports both `linux/amd64` and `linux/arm64`.
- **Claude Code CLI**: Must be installed and authenticated on the host machine. ccbox detects the installed version and builds a matching container image.
- **npm** (for installation): Required to install ccbox via the npm package. Alternatively, the Go binary can be built from source.

---

## 11. Limitations and Known Issues

1. **Not a security sandbox**: ccbox provides a safety net against accidental damage, but does not provide full isolation. The container has network access, mounted volumes, and forwarded credentials.

2. **Clipboard on ARM64**: Clipboard image paste (Ctrl+V) is unavailable on ARM64 Linux and ARM64 Windows due to CGO cross-compilation constraints. File drag-drop still works.

3. **Model selection for Max subscribers**: Max subscription users may get Sonnet instead of Opus by default. Use `--model opus` explicitly as a workaround.

4. **No old image cleanup**: When ccbox or Claude Code is upgraded, new local Docker images are built but old ones are not automatically removed. Users should periodically run `docker image prune` to clean up stale `ccbox-local` images.

5. **First-run build time**: The first invocation of ccbox (or after a version upgrade) requires building a local Docker image, which involves downloading and installing Claude Code. Subsequent runs use the cached image and start quickly.

6. **`-c` flag ambiguity**: The `-c` short flag is used by ccbox for `--claudePath`. Since ccbox flags require the `--` separator, `ccbox -c` (without `--`) is treated as Claude's `--continue` flag. With `--`, it becomes ccbox's `--claudePath`: `ccbox -c /path/to/claude --`.

7. **Host command execution**: Passthrough commands run via `sh -c` on the host with the host user's full permissions. There is no sandboxing or filtering of passthrough command execution.

8. **Container path prefix matching**: The statusline path rewriting uses prefix matching on `/home/claude`, which means paths starting with `/home/claudette` (or similar) would also be incorrectly rewritten. This is an edge case unlikely to occur in practice.

9. **Image registry not configurable**: The Docker base image registry (`ghcr.io/ccdevkit/ccbox-base`) is hardcoded. There is no mechanism to override the registry via flags, environment variables, or config files.
