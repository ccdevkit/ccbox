# Contract: Settings Manager API

**Package**: `internal/claude/settings`

## Interface

```go
// ClaudeSettingsManager discovers, merges, and manages Claude Code settings.
type ClaudeSettingsManager struct { ... }

// FS abstracts filesystem operations for testability.
type FS interface {
    ReadFile(path string) ([]byte, error)
    Stat(path string) (os.FileInfo, error)
}

// NewClaudeSettingsManager creates a manager that discovers and merges
// settings from standard locations using the provided filesystem and paths.
func NewClaudeSettingsManager(fs FS, homeDir string, projectDir string) (*ClaudeSettingsManager, error)

// Set sets a top-level key in the merged settings.
// Panics if called after Finalize().
func (m *ClaudeSettingsManager) Set(key string, value interface{})

// SetDeep sets a nested key using dot notation (e.g., "hooks.PreToolUse").
// Creates intermediate maps as needed.
// Panics if called after Finalize().
func (m *ClaudeSettingsManager) SetDeep(path string, value interface{})

// MergeHooks merges hook entries from the registry into the settings,
// respecting before/after ordering relative to existing user hooks.
// Panics if called after Finalize().
func (m *ClaudeSettingsManager) MergeHooks(hooks map[string][]MatcherGroup, order map[string]map[string]Order)

// Finalize writes the merged settings to a session file and returns
// the CLI args to add (--settings <path> --setting-sources "").
func (m *ClaudeSettingsManager) Finalize(fw session.SessionFileWriter) (cliArgs []string, err error)
```

## Discovery Order (lowest → highest precedence)

1. `{homeDir}/.claude/settings.json`
2. `{homeDir}/.claude/settings.local.json`
3. `{projectDir}/.claude/settings.json`
4. `{projectDir}/.claude/settings.local.json`

Files that don't exist are skipped. Malformed JSON files are skipped with a logged warning.

## Merge Rules

- Top-level keys: higher-precedence file wins (simple overwrite)
- `hooks` key: matcher groups are merged, not overwritten. User hooks within a matcher group are preserved; ccbox hooks are inserted before or after based on order.
- Module-injected settings (`Set`/`SetDeep`) are applied after file merge, so they have highest precedence for non-hook keys.

## Finalize Behavior

1. Marshal merged settings to JSON
2. Write to session file via `SessionFileWriter.WriteFile(containerPath, data, readOnly)`
3. Return `["--settings", containerPath, "--setting-sources", ""]`
4. Mark manager as finalized (subsequent Set/SetDeep calls panic)
