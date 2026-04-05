# Data Model: Hook Integration

**Branch**: `003-hook-integration` | **Date**: 2026-04-05

## Entities

### HookEvent (enum)

Represents all Claude Code hook event names.

| Value | Category |
|-------|----------|
| `SessionStart` | Lifecycle |
| `SessionEnd` | Lifecycle |
| `InstructionsLoaded` | Instruction |
| `UserPromptSubmit` | Prompt |
| `PreToolUse` | Tool |
| `PostToolUse` | Tool |
| `PostToolUseFailure` | Tool |
| `PermissionRequest` | Tool |
| `PermissionDenied` | Tool |
| `SubagentStart` | Agent |
| `SubagentStop` | Agent |
| `TaskCreated` | Task |
| `TaskCompleted` | Task |
| `TeammateIdle` | Task |
| `Stop` | Workflow |
| `StopFailure` | Workflow |
| `PreCompact` | Compact |
| `PostCompact` | Compact |
| `FileChanged` | File/Config |
| `CwdChanged` | File/Config |
| `ConfigChange` | File/Config |
| `WorktreeCreate` | Worktree |
| `WorktreeRemove` | Worktree |
| `Elicitation` | MCP |
| `ElicitationResult` | MCP |
| `Notification` | Other |

### HookInputBase

Common fields present in every hook event's stdin JSON.

| Field | Type | Required | Description |
|-------|------|----------|-------------|
| `session_id` | string | yes | Current session ID |
| `transcript_path` | string | yes | Path to transcript file |
| `cwd` | string | yes | Current working directory |
| `permission_mode` | string | yes | Active permission mode |
| `hook_event_name` | string | yes | Event name that triggered this hook |
| `agent_id` | string | no | Agent ID if in subagent context |
| `agent_type` | string | no | Agent type if in subagent context |

### HookOutputBase

Common fields for hook stdout JSON responses.

| Field | Type | Required | Default | Description |
|-------|------|----------|---------|-------------|
| `continue` | bool | no | true | false stops the agentic loop |
| `stopReason` | string | no | "" | Reason shown when continue=false |
| `suppressOutput` | bool | no | false | Suppress tool output from context |
| `systemMessage` | string | no | "" | Message shown to user |

### HookHandler (Interface)

Implemented by per-event handler structs (e.g., `PreToolUseHandler`, `SessionStartHandler`).
The event name is implicit from the concrete type — no chance of mismatch.

| Method | Returns | Description |
|--------|---------|-------------|
| `EventName()` | HookEvent | The event this handler responds to |
| `MatcherPattern()` | string | Regex pattern to narrow invocations (empty = match all) |
| `HandlerOrder()` | Order | `before` or `after` relative to user hooks |
| `invoke(json.RawMessage)` | `(*HandlerResult, error)` | Unmarshal input, call typed Fn, marshal output |

### Per-Event Handler Struct (e.g., PreToolUseHandler)

Each of the 26 events gets a concrete struct implementing HookHandler.

| Field | Type | Required | Description |
|-------|------|----------|-------------|
| `Matcher` | string | no | Regex pattern (empty = match all) |
| `Order` | Order | yes | `before` or `after` |
| `Fn` | `func(*PreToolUseInput) (*PreToolUseOutput, error)` | yes | Strongly-typed handler callback |

### HandlerResult (Wire-level)

The internal wire representation returned by `invoke()`. Callers never construct this directly — it's produced by the typed output's `toResult()` method.

| Field | Type | Required | Description |
|-------|------|----------|-------------|
| `ExitCode` | int | yes | 0 = success, 2 = block, other = non-blocking error |
| `Stdout` | []byte | no | JSON output (parsed by Claude Code when exit 0) |
| `Stderr` | []byte | no | Error message (used by Claude Code when exit 2) |

### HookRequest (Wire)

TCP bridge message from container proxy to host.

| Field | Type | Description |
|-------|------|-------------|
| `type` | string | Always `"hook"` |
| `event` | string | Hook event name |
| `input` | json.RawMessage | Raw hook input JSON from stdin |

### HookResponse (Wire)

TCP bridge response from host to container proxy.

| Field | Type | Description |
|-------|------|-------------|
| `exit_code` | int | Exit code for proxy to return |
| `stdout` | string | Stdout content for proxy to write |
| `stderr` | string | Stderr content for proxy to write |

### MatcherGroup (Settings)

A matcher group within the settings.json hooks configuration.

| Field | Type | Required | Description |
|-------|------|----------|-------------|
| `matcher` | string | no | Regex pattern; empty/omitted = match all |
| `hooks` | []HookEntry | yes | Array of hook handler configurations |

### HookEntry (Settings)

A single hook handler entry in the settings.json hooks array.

| Field | Type | Required | Description |
|-------|------|----------|-------------|
| `type` | string | yes | Always `"command"` for ccbox-injected hooks |
| `command` | string | yes | Path to container proxy binary |
| `timeout` | int | no | Timeout in seconds (default from Claude Code) |

### SettingsManager

Manages Claude Code settings discovery, merging, and modification.

| Field | Type | Description |
|-------|------|-------------|
| `merged` | map[string]interface{} | Combined settings from all sources |
| `finalized` | bool | Whether Finalize() has been called |

**State transitions**: `New()` → modifications via `Set()`/`SetDeep()` → `Finalize()` → settings file written, CLI args registered.

### Settings File Precedence (lowest → highest)

1. `~/.claude/settings.json` (user)
2. `~/.claude/settings.local.json` (user local)
3. `.claude/settings.json` (project)
4. `.claude/settings.local.json` (project local)
5. ccbox module modifications (additive, applied after merge)

## Relationships

```
HookHandler (registration) ──registers──▶ HookRegistry
                                              │
                                              ▼ (at finalize time)
                                        SettingsManager
                                              │
                                    ┌─────────┴─────────┐
                                    ▼                   ▼
                             settings.json        CLI args
                          (hooks entries)     (--settings, --setting-sources "")
                                    │
                                    ▼
                              Container
                                    │
                    ┌───────────────┼───────────────┐
                    ▼               ▼               ▼
              Claude Code    cchookproxy      ccptproxy
              (fires hooks)  (TCP→host)    (existing proxy)
                    │               │
                    ▼               ▼
              Hook stdin      TCP bridge
              to proxy        (host server)
                                    │
                                    ▼
                              HookHandler.Fn
                              (host-side)
```
