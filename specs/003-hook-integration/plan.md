# Implementation Plan: Hook Integration

**Branch**: `003-hook-integration` | **Date**: 2026-04-05 | **Spec**: [spec.md](spec.md)
**Input**: Feature specification from `/specs/003-hook-integration/spec.md`

## Summary

ccbox needs to intercept and respond to Claude Code hook events during containerized sessions. This requires: (1) a typed hook handler registry with dispatch logic, (2) a general-purpose Settings Manager that discovers/merges/modifies Claude Code settings and injects hook configuration, (3) a container-side proxy binary (`cchookproxy`) that forwards hook events over TCP to the host, and (4) extension of the existing TCP bridge protocol with a `hook` message type.

## Technical Context

**Language/Version**: Go 1.24 (toolchain go1.24.5)
**Primary Dependencies**: stdlib (`encoding/json`, `net`, `os`, `regexp`, `strings`, `fmt`, `path/filepath`), `ccdevkit/common` (settings discovery)
**Storage**: Filesystem — session temp files, Claude Code settings JSON files
**Testing**: stdlib `testing` package, table-driven and per-case test functions, real TCP in integration tests, hand-rolled mock structs for interfaces
**Target Platform**: macOS/Linux (Darwin/Linux host, Linux container)
**Project Type**: CLI tool with container runtime
**Performance Goals**: Hook responses within 1 second for typical payloads (SC-002); 10-second proxy timeout (FR-010)
**Constraints**: No external dependencies beyond what's already in go.mod; proxy binary must be small and fast
**Scale/Scope**: 26 hook event types, internal API consumed by ccbox modules

## Constitution Check

_GATE: Must pass before Phase 0 research. Re-check after Phase 1 design._

| Principle | Status | Notes |
|-----------|--------|-------|
| I. Simplicity Over Cleverness | PASS | Direct function-based handlers, no framework abstractions |
| II. Explicit Over Implicit | PASS | Typed event names, explicit registration, clear error context |
| III. Fail Fast, Fail Clearly | PASS | Proxy exits with error codes, malformed JSON skipped with warning |
| IV. Single Responsibility | PASS | `hooks/` = registry + dispatch, `settings/` = settings management, `bridge/` = transport, `cmd/cchookproxy/` = container proxy |
| V. No Over-Engineering | PASS | Function types for handlers (no interfaces until second consumer), `map[string]interface{}` for settings flexibility |
| VI. Test What Matters | PASS | Registry dispatch, settings merge precedence, wire protocol, proxy binary behavior |
| VII. Red-Green-Refactor TDD | PASS | All components developed test-first |

**Post-Phase 1 re-check**: Design remains compliant. No interfaces introduced without consumers. Settings Manager uses `map[string]interface{}` to avoid over-typing Claude Code's full schema. Hook types use `json.RawMessage` for tool-specific inputs to avoid coupling to every tool's schema.

## Project Structure

### Documentation (this feature)

```text
specs/003-hook-integration/
├── plan.md              # This file
├── research.md          # Phase 0 output
├── data-model.md        # Phase 1 output
├── quickstart.md        # Phase 1 output
├── contracts/           # Phase 1 output
│   ├── hook-registry.md
│   ├── settings-manager.md
│   └── hook-bridge.md
└── tasks.md             # Phase 2 output (NOT created by /speckit.plan)
```

### Source Code (repository root)

```text
cmd/
├── cchookproxy/         # NEW: Container-side hook proxy binary
│   └── main.go

internal/
├── claude/
│   ├── claude.go        # MODIFIED: Integrate Settings Manager + hook registry
│   ├── session_files.go # MODIFIED: Remove hardcoded settingsJSON, delegate to Settings Manager
│   ├── hooks/           # NEW: Hook handler registry and types
│   │   ├── events.go    # Hook event enum constants
│   │   ├── types.go     # Typed input/output structs for all 26 events
│   │   ├── registry.go  # Handler registration and dispatch
│   │   └── registry_test.go
│   └── settings/        # MOVED+EXPANDED from internal/settings/claude_settings.go
│       ├── manager.go   # Settings Manager (discovery, merge, modification API, finalize)
│       └── manager_test.go
├── settings/
│   ├── settings.go      # UNCHANGED: ccbox settings (Load, MergeWithCLI)
│   └── claude_settings.go # REMOVED (migrated to internal/claude/settings/)
├── bridge/
│   ├── types.go         # MODIFIED: Add HookRequest/HookResponse
│   ├── server.go        # MODIFIED: Add hook handler dispatch
│   └── server_test.go   # MODIFIED: Add hook handler tests
└── constants/
    └── constants.go     # MODIFIED: Add HookRequestType, proxy paths
```

**Structure Decision**: Claude Code-specific packages are nested under `internal/claude/` to make the domain boundary clear. `internal/claude/hooks/` owns hook registration, dispatch, and typed event structs. `internal/claude/settings/` owns the Settings Manager (migrated and expanded from `internal/settings/claude_settings.go`). The top-level `internal/settings/` retains only ccbox-specific settings (`Settings` struct, `Load()`, `MergeWithCLI`). Container proxy follows the established `cmd/` pattern from `ccptproxy`.

## Test Strategy

| Component | Test Type | First Red Test | TDD | Skip Reason |
|-----------|-----------|----------------|-----|-------------|
| Hook event constants | Unit | Event string constants match Claude Code's expected values | Yes | — |
| Hook input/output types | Unit | JSON unmarshal of sample PreToolUse input produces correct typed struct | Yes | — |
| Hook registry (register) | Unit | Register handler, verify it's retrievable by event name | Yes | — |
| Hook registry (dispatch) | Unit | Dispatch to registered handler returns handler's result | Yes | — |
| Hook registry (matcher) | Unit | Handler with matcher "Bash" only matches Bash tool_name inputs | Yes | — |
| Hook registry (no match) | Unit | Dispatch with no registered handler returns exit 0, empty output | Yes | — |
| Hook registry (multiple) | Unit | Multiple handlers for same event: block wins over allow | Yes | — |
| Hook registry (HookEntries) | Unit | HookEntries produces correct settings.json structure with before/after ordering | Yes | — |
| Settings Manager (discovery) | Unit | Discovers and merges 4 settings files by precedence | Yes | — |
| Settings Manager (missing files) | Unit | Missing files skipped, no error | Yes | — |
| Settings Manager (malformed JSON) | Unit | Malformed file skipped with warning, other files still merged | Yes | — |
| Settings Manager (Set) | Unit | Set top-level key appears in finalized output | Yes | — |
| Settings Manager (MergeHooks before) | Unit | ccbox hook entries appear before user hooks in matcher group | Yes | — |
| Settings Manager (MergeHooks after) | Unit | ccbox hook entries appear after user hooks in matcher group | Yes | — |
| Settings Manager (Finalize) | Unit | Produces CLI args and writes session file | Yes | — |
| Settings Manager (preserves user) | Unit | User settings not overwritten by module settings (FR-011) | Yes | — |
| Bridge HookRequest wire | Unit | HookRequest marshals/unmarshals correctly | Yes | — |
| Bridge hook dispatch | Integration | Server receives hook request, invokes handler, returns response | Yes | — |
| cchookproxy (success) | Integration | Proxy reads stdin, sends TCP, writes stdout, exits 0 | Yes | — |
| cchookproxy (block) | Integration | Host returns exit 2, proxy exits 2 with stderr | Yes | — |
| cchookproxy (TCP failure) | Integration | TCP unreachable, proxy exits 1 (non-blocking) | Yes | — |
| cchookproxy (timeout) | Integration | Host doesn't respond within 10s, proxy exits 1 | Yes | — |
| Claude integration | Integration | BuildRunSpec with Settings Manager produces --settings and --setting-sources args | Yes | — |

**Red-green-refactor sequence**: Task generation (`/speckit.tasks`) MUST interleave
"write failing test" steps before their corresponding implementation steps, not
group all tests at the end.

## Complexity Tracking

No constitution violations. No complexity justifications needed.
