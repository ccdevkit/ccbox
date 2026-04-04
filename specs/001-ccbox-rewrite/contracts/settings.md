# Settings File Contract

## Implementation

Settings loading delegates to `ccdevkit/common/settings.Load()`. The `internal/settings` package is a thin wrapper that defines the `Settings` struct and calls `common/settings.Load(".ccbox/settings", cfg, nil)`. No custom discovery, parsing, or merge logic is implemented in ccbox.

This matches the pattern established by ccyolo's `internal/settings` package.

## File Discovery

Handled entirely by `common/settings`. Walks from CWD to filesystem root, checking for `.ccbox/settings.{json,yaml,yml}` at each level. Files are loaded in order from farthest to closest. Home directory is always included in the walk (even if CWD is outside home).

First matching file at each level wins (`.json` > `.yaml` > `.yml` — only one per level).

## Merge Precedence

```
CLI flags (via Options.AdditionalFiles) > closest settings file > ... > farthest settings file > defaults
```

**Merge rules** (provided by `common/settings`):
- **Primitive values** (string, bool): Higher precedence replaces lower
- **Objects**: Merged recursively (field-level merge)
- **Arrays** (e.g., `passthrough`): Appended across all levels

## Schema

Struct uses `yaml` tags only (not both `json` and `yaml`) to comply with `common/settings` which returns `ErrMixedTags` on mixed tag sets. File format detection is by extension, not by struct tags.

### JSON Format

```json
{
  "passthrough": ["git", "docker", "gh"],
  "claudePath": "/usr/local/bin/claude",
  "verbose": false,
  "logFile": ""
}
```

### YAML Format

```yaml
passthrough:
  - git
  - docker
  - gh
claudePath: /usr/local/bin/claude
verbose: false
logFile: ""
```

## Fields

| Field | Type | Default | Merge | Description |
|-------|------|---------|-------|-------------|
| `passthrough` | string[] | `[]` | Append | Command prefixes to route to host |
| `claudePath` | string | `"claude"` | Replace | Path to claude CLI on host |
| `verbose` | bool | `false` | Replace | Enable debug logging |
| `logFile` | string | `""` | Replace | Debug log file path |

## Error Handling

Handled by `common/settings`:
- Invalid JSON/YAML files return `ErrInvalidConfig` (ccbox should treat as non-fatal)
- Missing files are silently skipped
- File permission errors are silently ignored
- All valid files in the discovery path are merged regardless of errors in other files
