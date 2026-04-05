# Implementation Plan: Passthrough Command Permissions

**Branch**: `002-passthrough-permissions` | **Date**: 2026-04-04 | **Spec**: [spec.md](spec.md)
**Input**: Feature specification from `/specs/002-passthrough-permissions/spec.md`

## Summary

Add a cascading allow/deny permissions system for passthrough commands, configured via `.ccbox/permissions.{json,yml,yaml}` and enforced on the host side. Users define ordered pattern/effect rules per command; the last matching rule wins. Patterns support wildcards, regex, non-positional matching, and optional elements. The system merges with existing CLI `-pt` flags and fails closed on invalid config or unmatched rules.

## Technical Context

**Language/Version**: Go 1.24 (toolchain go1.24.5)
**Primary Dependencies**: `ccdevkit/common` (settings package for hierarchical file discovery), stdlib (`regexp`, `strings`, `fmt`)
**Storage**: Filesystem — `.ccbox/permissions.{json,yml,yaml}` discovered hierarchically
**Testing**: `go test` with table-driven tests per constitution
**Target Platform**: Darwin/Linux (host side), Linux container (container side unchanged)
**Project Type**: CLI tool
**Performance Goals**: N/A — human-speed command invocations
**Constraints**: Must not break existing `-pt` flag behavior when no permissions file exists
**Scale/Scope**: Single-user CLI tool, <100 rules typical

## Constitution Check

_GATE: Must pass before Phase 0 research. Re-check after Phase 1 design._

| Principle | Status | Notes |
|-----------|--------|-------|
| I. Simplicity Over Cleverness | PASS | Pattern syntax is defined by spec, implementation uses straightforward tokenizer + matcher |
| II. Explicit Over Implicit | PASS | Clear error messages, typed parameters, no global state |
| III. Fail Fast, Fail Clearly | PASS | All patterns validated at startup (FR-010), malformed config → startup error (FR-007) |
| IV. Single Responsibility | PASS | New `internal/permissions` package owns policy; `cmdpassthrough` owns execution |
| V. No Over-Engineering | PASS | Building exactly what spec requires. Pattern syntax is spec-mandated, not speculative. |
| VI. Test What Matters | PASS | Pattern matching has many input cases → table-driven tests. Integration tests for enforcement hook. |
| VII. Red-Green-Refactor TDD | PASS | All components built test-first per TDD cycle |

**Post-Phase 1 Re-check**: PASS — no violations introduced. The `internal/permissions` package is a single new domain package. No interfaces introduced (no second consumer yet). Pattern parser is the most complex component but justified by spec-mandated syntax.

## Project Structure

### Documentation (this feature)

```text
specs/002-passthrough-permissions/
├── spec.md
├── plan.md              # This file
├── research.md          # Phase 0 output
├── data-model.md        # Phase 1 output
├── quickstart.md        # Phase 1 output
├── contracts/
│   └── permissions-api.md
├── pattern-syntax-notes.md
└── tasks.md             # Phase 2 output (speckit.tasks)
```

### Source Code (repository root)

```text
internal/
├── permissions/          # NEW — permission policy engine
│   ├── types.go          # Effect, MatchResult, PermissionsConfig, CommandPermission, Rule, PatternOrArray
│   ├── types_test.go
│   ├── config.go         # Load() — permissions file discovery and parsing
│   ├── config_test.go
│   ├── pattern.go        # ArgPattern parser and tokenizer
│   ├── pattern_test.go
│   ├── match.go          # Pattern matching engine (positional, wildcard, regex, etc.)
│   ├── match_test.go
│   ├── checker.go        # Checker type, NewChecker(), Check(), CLI merge, Commands()
│   └── checker_test.go
├── cmdpassthrough/
│   ├── exec.go           # MODIFIED — add permission-aware wrapper
│   └── exec_test.go      # MODIFIED — test enforcement hook
└── bridge/
    └── (unchanged)

cmd/
└── ccbox/
    └── orchestrate.go    # MODIFIED — load permissions, wire into bridge
```

## Test Strategy

| Component | Test Type | First Red Test | TDD | Skip Reason |
|-----------|-----------|----------------|-----|-------------|
| permissions.Load() | Unit | Load valid YAML → returns parsed PermissionsConfig | Yes | — |
| permissions.Load() errors | Unit | Load malformed YAML → returns error with file context | Yes | — |
| PatternOrArray unmarshal | Unit | Unmarshal string → single pattern; array → multiple patterns | Yes | — |
| ArgPattern parser | Unit | Parse "pull" → single literal element | Yes | — |
| ArgPattern parser (wildcards) | Unit | Parse "**" → doubleWildcard element | Yes | — |
| ArgPattern parser (regex) | Unit | Parse "/^https/" → regex element | Yes | — |
| ArgPattern parser (complex) | Unit | Parse "push ~--force" → literal + non-positional | Yes | — |
| ArgPattern parser (errors) | Unit | Parse "/invalid[/" → error with pattern context | Yes | — |
| Pattern matching | Unit | Match "pull" against args ["pull"] → true | Yes | — |
| Pattern matching (prefix) | Unit | Match "status" against ["status", "--short"] → true | Yes | — |
| Pattern matching (exact $) | Unit | Match "status$" against ["status", "--short"] → false | Yes | — |
| Pattern parsing ($ whitespace) | Unit | "push --force$", "push --force $", "push --force           $" all parse to same elements with ExactMatch=true | Yes | — |
| Pattern parsing (whitespace normalization) | Unit | "push        --force" parses identically to "push --force" | Yes | — |
| Pattern parsing ($ mid-pattern error) | Unit | "push $ --force" → validation error at parse time | Yes | — |
| Pattern matching (**) | Unit | Match "push **" against ["push", "origin", "main"] → true | Yes | — |
| Pattern matching (~) | Unit | Match "push ~--force" against ["push", "--force", "origin"] → true | Yes | — |
| Pattern matching (?) | Unit | Match "pull origin?" against ["pull"] → true | Yes | — |
| Pattern matching (regex) | Unit | Match "clone /https?:\/\//" against ["clone", "https://x"] → true | Yes | — |
| Rule evaluation (last-match-wins) | Unit | deny **, allow pull → git pull allowed, git push denied | Yes | — |
| Rule evaluation (fail-closed) | Unit | allow status only → git push denied (no match) | Yes | — |
| Rule evaluation (no rules) | Unit | Command with nil rules → all allowed | Yes | — |
| CLI merge | Unit | CLI pt:git + file deny rules → allow ** prepended | Yes | — |
| CLI merge (no file) | Unit | CLI pt:git, no file entry → allow ** only | Yes | — |
| Denial messages | Unit | Denied by rule → message includes rule and reason | Yes | — |
| Denial messages (default) | Unit | Denied by no-match → message lists available patterns | Yes | — |
| Enforcement hook | Integration | Denied command → exit code + denial message returned | Yes | — |
| Orchestration wiring | Integration | Permissions loaded and checker wired into bridge | Yes | — |
| Config validation | Unit | Invalid regex → startup error | Yes | — |
| Config validation | Unit | Invalid effect → startup error | Yes | — |
| Backward compat | Integration | No permissions file + CLI -pt:git → all git commands work | Yes | — |

**Red-green-refactor sequence**: Task generation MUST interleave "write failing test" steps before their corresponding implementation steps.

## Complexity Tracking

No constitution violations — table intentionally empty.

| Violation | Why Needed | Simpler Alternative Rejected Because |
|-----------|-----------|--------------------------------------|
| — | — | — |
