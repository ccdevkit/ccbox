<h1 align="center">ccbox</h1>

<p align="center">
Run Claude Code in Docker with auto-accept enabled. No setup, full parity with <code>claude</code> CLI.
</p>

> [!CAUTION]
> ccbox is a safety net, not a sandbox. It protects against overzealous agents doing stupid things, but it won't contain a fully compromised, malicious agent. Only use with repositories you trust.

## Install

```bash
npm install -g @ccdevkit/ccbox
```

## Usage

Use `--` to separate ccbox flags from claude flags:

```bash
ccbox                        # Interactive mode
ccbox -- -p "hello"          # One-shot prompt
ccbox -- -c                  # Continue previous session
ccbox -- -r                  # Resume session picker
```

All `claude` flags work as expected when passed after `--`.

## Why

- Works out of the box - no container setup required
- Full CLI parity with `claude`
- Runs with auto-accept enabled (YOLO mode)
- Your working directory is mounted into the container
- Your Claude credentials are passed through automatically

## ccbox flags

ccbox flags go before `--`, claude flags go after:

```bash
ccbox [ccbox-flags] -- [claude-args]
ccbox [claude-args]

ccbox -v -- -p "hello"              # Verbose mode
ccbox --log /tmp/debug.log -- -c    # Log to file
ccbox -pt:git -- -p "git status"    # Run git on host instead of container
ccbox --use 2.1.16 --               # Use specific Claude Code version
```

| Flag                                  | Description                                       |
| ------------------------------------- | ------------------------------------------------- |
| `-v`, `--verbose`                     | Enable debug logging to stderr                    |
| `--log <path>`                        | Write debug logs to file (implies -v)             |
| `-c`, `--claudePath <path>`           | Path to claude CLI (default: claude in PATH)      |
| `--use <version>`                     | Use specific Claude Code version (e.g., 2.1.16)   |
| `-pt:<cmd>`, `--passthrough:<cmd>`    | Run commands matching prefix on host (repeatable) |
| `--version`                           | Print ccbox version                               |

### Subcommands

| Command        | Description                                                      |
| -------------- | ---------------------------------------------------------------- |
| `ccbox update` | Update Claude Code on the host and rebuild the Docker image      |
| `ccbox clean`  | Remove all ccbox-managed Docker images except the latest         |

### Passthrough

By default, all commands run inside the container. This is usually fine, but some commands need to run on your host machine - things like `docker`, `gh`, or commands that need access to host resources.

Use `-pt:` to specify command prefixes that should run on the host:

```bash
ccbox -pt:git -pt:docker -- -p "build and push the image"
```

This runs `git` and `docker` commands on your host, while everything else runs in the container. You can specify `-pt:` multiple times.

## Passthrough Permissions

ccbox supports fine-grained control over which arguments are allowed for passthrough commands. Define rules in `.ccbox/permissions.{json,yml,yaml}` to allow or deny specific subcommands and flags.

### Why permissions?

Without permissions, passthrough is all-or-nothing: a command either runs on the host with no restrictions, or it doesn't run at all. Permissions let you grant least-privilege access - for example, allowing `git pull` but blocking `git push --force`.

### Permissions file structure

Create `.ccbox/permissions.yaml` (or `.json`/`.yml`) in your project or home directory. All configuration is namespaced under a top-level `passthrough` key. Each key under `passthrough` is a command name - a `null`/empty value allows all arguments, while a `rules` array applies cascading evaluation.

```yaml
passthrough:
  # No restrictions - all arguments allowed
  git:

  # Only allow install, ci, and run build
  npm:
    rules:
      - pattern: "**"
        effect: deny
      - pattern: "install"
        effect: allow
      - pattern: "ci"
        effect: allow
      - pattern: "run build"
        effect: allow

  # Allow everything, block dangerous flags
  kubectl:
    rules:
      - pattern: "**"
        effect: allow
      - pattern:
          - "delete ~--all"
          - "delete ~-A"
        effect: deny
        reason: "Bulk delete is too destructive"
```

### How rules work

Rules are evaluated **top-to-bottom** and the **last matching rule wins**. If rules are defined but none match, the command is **denied** (fail-closed).

Commands added via CLI flags (`-pt:git`) get an implicit `allow **` as the first rule. If the permissions file also defines rules for that command, the file's rules are appended after the implicit allow - so `-pt:git` combined with deny rules produces "allow all except what the file denies."

### Pattern syntax

| Token    | Example              | Matches                                        |
|----------|----------------------|------------------------------------------------|
| `word`   | `pull`               | Exact arg "pull"                               |
| `*`      | `--*`                | Any arg starting with "--"                     |
| `**`     | `push **`            | "push" followed by any number of args          |
| `.`      | `v.`                 | Any 2-char arg starting with "v"               |
| `/re/`   | `/^https?:\/\//`     | Arg matching regex                             |
| `~`      | `~--force`           | "--force" anywhere in remaining args           |
| `?`      | `pull origin?`       | "pull" with optional "origin"                  |
| `"str"`  | `"my file"`          | Exact literal (preserves spaces, disables globs) |
| `()`     | `(origin main)?`     | Optional group of args                         |
| `$`      | `status$`            | Only "status" exactly (no prefix matching)     |
| `\`      | `\*`                 | Literal asterisk                               |

By default, patterns use **prefix matching**: `status` matches `status`, `status --short`, etc. Append `$` to require an exact match.

### Permissions file discovery

Permissions files are discovered hierarchically, the same way settings files are - walking up from the current directory to root. Project-level permissions override parent/home-level permissions. Malformed permissions files cause ccbox to **refuse to start** with a clear error (fail-closed).

## Clipboard & Drag-Drop

ccbox supports pasting images from your clipboard (Ctrl+V / Cmd+V) and dragging files into the terminal, just like the native `claude` CLI.

### How it works

When you paste or drag a file, ccbox intercepts the input, copies the file into the container via a shared bridge directory, and rewrites the path so Claude sees it correctly.

### Platform support

| Platform | Clipboard (Ctrl+V) | File drag-drop |
|----------|-------------------|----------------|
| macOS (Intel & Apple Silicon) | ✅ | ✅ |
| Linux x64 | ✅ | ✅ |
| Windows x64 | ✅ | ✅ |
| Linux ARM64 | ❌ | ✅ |
| Windows ARM64 | ❌ | ✅ |

**Why no clipboard on ARM64?** Clipboard image support requires native system APIs (NSPasteboard, Win32, X11) which need CGO compilation. GitHub Actions doesn't provide native ARM64 runners for Linux or Windows, so those builds are cross-compiled without CGO. File drag-drop still works because it only requires path rewriting, not system clipboard access.

If you need clipboard support on ARM64, you can build from source on a native ARM64 machine with CGO enabled.

## Settings

ccbox can be configured using a settings file in your project or home directory. Settings files are discovered by walking up from your current directory to root.

### Settings file location

Create a settings file at `.ccbox/settings.json` (or `.yaml`/`.yml`) in your project directory or home directory:

```
your-project/
  .ccbox/
    settings.json    # Project-specific settings
```

Or in your home directory for global settings:

```
~/.ccbox/
  settings.json      # Global settings
```

### Available settings

```json
{
  "claudePath": "/path/to/claude",
  "passthrough": ["git", "docker", "gh"],
  "verbose": false,
  "logFile": ""
}
```

| Setting | Type | Description |
|---------|------|-------------|
| `claudePath` | string | Path to the claude CLI executable (default: `claude` in PATH) |
| `passthrough` | array | List of command prefixes to run on the host instead of in the container |
| `verbose` | bool | Enable debug logging (default: false) |
| `logFile` | string | Debug log file path (default: none) |

### Settings priority

Settings are merged with the following priority (highest to lowest):

1. Command-line flags
2. Project settings (`.ccbox/settings.json` in current directory or ancestors)
3. Global settings (`~/.ccbox/settings.json`)
4. Defaults

### Example configurations

**Minimal setup:**
```json
{
  "passthrough": ["git"]
}
```

**Advanced setup:**
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

## Requirements

- Docker
- `claude` must be authenticated on your machine
