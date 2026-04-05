# Contract: Permissions API (internal/permissions)

**Feature Branch**: `002-passthrough-permissions`
**Date**: 2026-04-04

## Package Interface

### Load

```go
// Load discovers and parses permissions from .ccbox/permissions.{json,yml,yaml}
// using hierarchical file walk. Returns nil config (not error) if no files found.
// Returns error if files exist but are malformed or contain invalid patterns.
func Load() (*PermissionsConfig, error)
```

### NewChecker

```go
// NewChecker creates a permission checker from a loaded config and optional
// CLI passthrough commands. CLI commands contribute an implicit allow-all
// rule as the first rule in each command's cascade.
// Compiles and validates all patterns at construction time.
// Returns error if any pattern is invalid.
func NewChecker(config *PermissionsConfig, cliPassthrough []string) (*Checker, error)
```

### Checker.Check

```go
// Check evaluates whether a command string is allowed.
// The command string is split: first word = command name, rest = arguments.
// Returns MatchResult with allowed status, reason, and matched rule.
//
// Behavior:
//   - Command not in permissions → MatchResult{Allowed: false, Reason: "command not configured"}
//   - Command with no rules → MatchResult{Allowed: true, Reason: "unrestricted"}
//   - Rules evaluated top-to-bottom, last match wins
//   - No rule matches → MatchResult{Allowed: false, Reason: "no matching rule (fail-closed)"}
func (c *Checker) Check(command string) MatchResult
```

### Checker.Commands

```go
// Commands returns the list of command names that have permission entries.
// Used to determine which commands need shims in the container.
func (c *Checker) Commands() []string
```

## Configuration Schema

### YAML

```yaml
passthrough:
  <command-name>:           # null = allow all
    rules:                  # ordered array, last match wins
      - pattern: "<pattern-string>"   # or array of strings
        effect: "allow" | "deny"
        reason: "optional denial message"
```

### JSON

```json
{
  "passthrough": {
    "<command-name>": null,
    "<command-name>": {
      "rules": [
        { "pattern": "<pattern-string>", "effect": "allow" },
        { "pattern": "<pattern-string>", "effect": "deny", "reason": "optional" }
      ]
    }
  }
}
```

## Pattern Syntax

See `pattern-syntax-notes.md` for full syntax reference.

| Token | Example | Matches |
|-------|---------|---------|
| `word` | `pull` | Exact arg "pull" |
| `*` | `--*` | Any arg starting with "--" |
| `**` | `push **` | "push" followed by any number of args |
| `.` | `v.` | Any 2-char arg starting with "v" |
| `/re/` | `/^https?:\/\//` | Arg matching regex |
| `/re/**` | `/--force\|--hard/**` | Any arg matching regex |
| `~` | `~--force` | "--force" anywhere in remaining args |
| `?` | `pull origin?` | "pull" with optional "origin" |
| `"str"` | `"my file"` | Exact literal (preserves spaces, disables globs) |
| `'str'` | `'my file'` | Same as `"str"` — single-quote variant |
| `()` | `(origin main)?` | Optional group of args |
| `$` | `status$` | Only "status" exactly (no prefix matching) |
| `\` | `\*` | Literal asterisk |

**Default**: Prefix matching enabled. Pattern `status` matches `status`, `status --short`, etc. Append `$` to require exact match.

## Error Contracts

| Condition | Behavior |
|-----------|----------|
| No permissions files found | `Load()` returns nil config, nil error |
| Malformed YAML/JSON | `Load()` returns error with file path and parse details |
| Invalid regex in pattern | `NewChecker()` returns error identifying the pattern |
| Invalid effect value | `Load()` returns error identifying the rule |
| Empty command name key | `Load()` returns error |
| Denial by rule | `Check()` returns `MatchResult{Allowed: false}` with rule's reason |
| Denial by default (no match) | `Check()` returns `MatchResult{Allowed: false}` listing available patterns |
