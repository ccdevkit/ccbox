# CLI Interface Contract: ccbox

## Command Syntax

```
ccbox [ccbox-flags] [-- [claude-args]]
ccbox update
ccbox clean
ccbox --version
ccbox --help
```

The `--` separator is **required** to pass arguments to Claude Code. Without it, all arguments are parsed as ccbox flags.

## ccbox Flags

| Flag | Short | Type | Default | Description |
|------|-------|------|---------|-------------|
| `--passthrough` | `-pt:CMD` | string (repeatable) | `[]` | Command prefix to route to host. Can use `-pt:git` or `--passthrough git` syntax. |
| `--claudePath` | `-c` | string | `claude` | Path to the `claude` CLI on the host. |
| `--use` | — | string | (auto-detect) | Pin Claude Code version inside the container. |
| `--verbose` | `-v` | bool | `false` | Enable debug logging to stderr. |
| `--log` | — | string | `""` | Write debug log to file (implicitly enables verbose). |
| `--version` | — | bool | — | Print ccbox version and exit. |
| `--help` | `-h` | bool | — | Print help text and exit. |

## Subcommands

### `ccbox update`

Runs `claude update` on the host (not in container), then rebuilds the local Docker image.

**Exit code**: 0 on success, non-zero on failure.

### `ccbox clean`

Removes all ccbox-managed Docker images except the latest auto-update image.

**Exit code**: 0 on success, non-zero on failure.

## Passthrough Flag Syntax

Two equivalent forms:
- Prefix syntax: `-pt:git`, `-pt:docker`, `-pt:gh`
- Long form: `--passthrough git`, `--passthrough docker`

Each entry is a single command name. If the first word of an invoked command matches an entry, the entire command is routed to the host.

Multiple passthroughs can be specified: `ccbox -pt:git -pt:docker -- -p "hello"`

## Exit Codes

ccbox propagates the container's exit code as its own exit code. Additional exit codes:

| Code | Meaning |
|------|---------|
| 0 | Success |
| 1 | General error (Docker not running, auth failure, etc.) |
| * | Container exit code (pass-through from Claude Code) |

## Environment Variables Read

| Variable | Purpose |
|----------|---------|
| `TERM` | Forwarded to container for terminal capability |
| `COLORTERM` | Forwarded to container for color support |
| `HOME` | Used for `~/.claude/` and `~/.ccbox/` paths |

## Examples

```bash
# Interactive session
ccbox

# One-shot prompt
ccbox -- -p "list files"

# With passthrough
ccbox -pt:git -pt:docker -- -p "run git status"

# Pin Claude version
ccbox --use 2.1.16 -- -p "hello"

# Debug logging
ccbox -v -- -p "hello"
ccbox --log /tmp/debug.log -- -p "hello"

# Update Claude and rebuild image
ccbox update

# Clean old Docker images
ccbox clean
```
