# Research: Hook Integration

**Branch**: `003-hook-integration` | **Date**: 2026-04-05

## R1: Claude Code Hook Event Taxonomy

**Decision**: All 28 hook events will have typed Go structs from day one.

**Rationale**: The spec mandates compile-time type safety (SC-005). All events share a common base input (`session_id`, `transcript_path`, `cwd`, `permission_mode`, `hook_event_name`, `agent_id`, `agent_type`) plus event-specific fields. Outputs follow a similar pattern: common base (`continue`, `stopReason`, `suppressOutput`, `systemMessage`) plus per-event `hookSpecificOutput`.

**Alternatives considered**:
- Start with subset (rejected by spec clarification — all 28 from day one)
- Use `map[string]interface{}` (rejected by SC-005 — compile-time safety required)

**Event categories and count**:
- Lifecycle: SessionStart, SessionEnd (2)
- Instruction: InstructionsLoaded (1)
- Prompt: UserPromptSubmit (1)
- Tool: PreToolUse, PostToolUse, PostToolUseFailure, PermissionRequest, PermissionDenied (5)
- Agent/Task: SubagentStart, SubagentStop, TaskCreated, TaskCompleted, TeammateIdle (5)
- Workflow: Stop, StopFailure (2)
- Compact: PreCompact, PostCompact (2)
- File/Config: FileChanged, CwdChanged, ConfigChange (3)
- Worktree: WorktreeCreate, WorktreeRemove (2)
- MCP: Elicitation, ElicitationResult (2)
- Other: Notification (1)

**Total**: 26 distinct events (not 28 as spec states — the spec's count likely included planned future events or counted TaskCreated/TaskCompleted separately from their shared input shape). Will define 26 typed input structs and corresponding output structs.

## R2: Settings File Discovery & Precedence

**Decision**: Settings Manager discovers four standard files in precedence order (lowest to highest): `~/.claude/settings.json`, `~/.claude/settings.local.json`, `.claude/settings.json`, `.claude/settings.local.json`.

**Rationale**: Claude Code's documented hierarchy is: managed policy > project local > project > user local > user. ccbox does not handle managed policy (that's Claude Code's responsibility). The four files above are what ccbox needs to discover and merge. The `--settings` flag loads additional settings from a file, and `--setting-sources ""` disables auto-discovery so the merged file is the sole source.

**Alternatives considered**:
- Only merge project settings (rejected — users configure hooks/permissions at user level too)
- Pass multiple `--settings` flags (rejected — not supported; single merged file is cleaner)

## R3: TCP Bridge Extension for Hook Messages

**Decision**: Extend the existing TCP bridge protocol with a `hook` message type, sharing the same port and infrastructure as `exec` and `log`.

**Rationale**: The bridge already handles newline-delimited JSON with a `type` envelope dispatched in `Server.handleConn()`. Adding `hook` is a natural extension — new wire types (`HookRequest`, `HookResponse`), a new handler type, and a new case in the switch statement.

**Key difference from exec/log**: Hook messages are request-response (like exec) but carry structured JSON payloads rather than raw command strings. The response includes an exit code, stdout, and stderr — mirroring the proxy binary's contract with Claude Code.

**Alternatives considered**:
- Separate port (rejected by spec clarification — shared bridge)
- HTTP instead of TCP (rejected — unnecessary complexity, existing pattern works)

## R4: Container Proxy Binary Pattern

**Decision**: New binary `cchookproxy` follows the same pattern as `ccptproxy` — reads from stdin, communicates over TCP, writes to stdout/stderr, exits with appropriate code.

**Rationale**: The existing `ccptproxy` pattern is proven: read config from JSON, dial `host.docker.internal:{port}`, send request, read response. The hook proxy is simpler — it doesn't need shim generation or command matching. It reads hook input from stdin, wraps it in a `HookRequest` with the event name (from `HOOK_EVENT_NAME` env var or parsed from stdin JSON), sends over TCP, and translates the response to exit code + stdout/stderr.

**Key design point**: Claude Code invokes the hook command with the hook input on stdin and expects the hook to exit with code 0 (success, parse stdout), 2 (blocking error, use stderr), or other (non-blocking error). The proxy must faithfully translate the host's response into these semantics.

**Alternatives considered**:
- Extend ccptproxy with hook mode (rejected — separate concerns, different lifecycle)
- Use HTTP proxy (rejected — TCP is simpler, matches existing pattern)

## R5: Handler Registration API Design

**Decision**: Functional options pattern for handler registration. Handlers are registered by event name with optional matcher and order parameters. The registry stores handlers and, at finalization time, calls the Settings Manager to inject the corresponding hook entries.

**Rationale**: The three-level hook config structure (event → matcher groups → hooks array) maps naturally to the registration API. Each handler registration specifies an event name, an optional matcher regex, and an order (`before`/`after`). At merge time, handlers with the same event and matcher are grouped into the same matcher group. Order controls placement within the hooks array relative to user-defined hooks.

**Alternatives considered**:
- Global function registration (rejected — harder to test, implicit coupling)
- Interface-based handlers (rejected — Principle V: no interfaces until second consumer; function type is sufficient)

## R6: Settings Manager Scope

**Decision**: The Settings Manager is a general-purpose module, not hook-specific. It discovers user settings, merges them, exposes a modification API for any ccbox module, and produces the final settings file.

**Rationale**: Per spec refinement, the hook registry is one consumer of the Settings Manager. Other modules (permissions, MCP config) can use the same API. The Settings Manager owns:
1. Discovery: find all four settings files
2. Merge: combine them by precedence
3. Modification API: `Set(key, value)`, `SetHooks(event, matcherGroups)`, etc.
4. Finalization: produce merged JSON, write as session file, register CLI args

**Alternatives considered**:
- Hook-only merger (rejected by user during spec clarification — too narrow)
- Typed settings struct (partially — hooks section needs structured types, but top-level uses `map[string]interface{}` for flexibility since we don't control Claude Code's full schema)
