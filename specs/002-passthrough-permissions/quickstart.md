# Quickstart: Passthrough Command Permissions

**Feature Branch**: `002-passthrough-permissions`

## What This Feature Does

Adds a permissions system for passthrough commands so users can control exactly which command+argument combinations Claude is allowed to execute on the host. Configured via `.ccbox/permissions.{json,yml,yaml}` with cascading allow/deny rules evaluated on the host side.

## Example Configuration

```yaml
# .ccbox/permissions.yaml
passthrough:
  # Allow all git commands except force push
  git:
    rules:
      - pattern: "**"
        effect: allow
      - pattern: "push ~--force"
        effect: deny
        reason: "Force push is destructive — use regular push"

  # Only allow safe npm commands
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

  # Allow docker with no restrictions
  docker:
```

## How It Works

1. User creates `.ccbox/permissions.yaml` in their project (or home dir)
2. On startup, ccbox loads and validates all patterns (fails fast on errors)
3. Permissions merge with CLI `-pt` flags (CLI adds implicit "allow all" as first rule)
4. When Claude runs a passthrough command, the host-side bridge checks permissions
5. Last matching rule wins — denied commands get a clear error message

## Key Concepts

- **Last-match-wins**: Rules evaluated top-to-bottom, last match determines outcome
- **Fail-closed**: If rules exist but none match, command is denied
- **Prefix matching**: `status` matches `status --short` (disable with `$`)
- **Host enforcement**: Permissions checked on trusted host, not in container

## Development Entry Points

- `internal/permissions/` — New package: config loading, pattern parsing, rule evaluation
- `internal/cmdpassthrough/exec.go` — Enforcement hook wraps `HandleExec`
- `cmd/ccbox/orchestrate.go` — Load permissions config and wire into bridge
- `internal/settings/settings.go` — Permissions config type (separate from settings)
