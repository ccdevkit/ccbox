# Tasks: Passthrough Command Permissions

**Input**: Design documents from `/specs/002-passthrough-permissions/`
**Prerequisites**: plan.md (required), spec.md (required for user stories), research.md, data-model.md, contracts/

**TDD**: Every task that produces code MUST follow Red-Green-Refactor. Tests are NOT separate tasks — each task includes writing the failing test, making it pass, and refactoring. This is non-negotiable per the project constitution (Principle VII).

**Task Granularity**: Each task MUST be small enough that the full TDD cycle (write failing test → implement → refactor) is a single coherent unit of work. If a task feels too large, split it. A good task produces one tested behavior.

**Organization**: Tasks are grouped by user story to enable independent implementation and testing of each story.

## Format: `[ID] [P?] [Story] Description`

- **[P]**: Can run in parallel (different files, no dependencies)
- **[Story]**: Which user story this task belongs to (e.g., US1, US2, US3)
- Include exact file paths in descriptions

## Path Conventions

- Go project: `internal/` for business logic, `cmd/` for CLI entry points
- Tests co-located with source: `*_test.go` next to `*.go`

---

## Phase 1: Setup

**Purpose**: Create the `internal/permissions` package skeleton and shared types

- [ ] T001 Create `internal/permissions/` package directory and types file with `Effect`, `MatchResult`, `PermissionsConfig`, `CommandPermission`, `Rule`, `PatternOrArray` types in `internal/permissions/types.go`
- [ ] T002 Implement `PatternOrArray` custom YAML/JSON unmarshaling (string or []string) in `internal/permissions/types.go` + `internal/permissions/types_test.go` (TDD: test unmarshal string → single; unmarshal array → multiple; unmarshal invalid → error)

---

## Phase 2: Foundational — Pattern Parser

**Purpose**: The pattern parser is the core engine that ALL user stories depend on. Must be complete and extensively tested before any rule evaluation can work.

**⚠️ CRITICAL**: No user story work can begin until pattern parsing is fully tested.

- [ ] T003 Implement pattern tokenizer: whitespace normalization (strip leading/trailing, collapse multi-whitespace) and `$` extraction (must be final character after stripping, error if mid-pattern) in `internal/permissions/pattern.go` + `internal/permissions/pattern_test.go` (TDD: test "push --force$" / "push --force $" / "push --force           $" all equivalent; test "push        --force" == "push --force"; test "push $ --force" → error)
- [ ] T004 Implement pattern parsing for literal tokens in `internal/permissions/pattern.go` + `internal/permissions/pattern_test.go` (TDD: test "pull" → single literal element; test "push origin main" → three literals)
- [ ] T005 [P] Implement pattern parsing for `*` (single-arg wildcard) and `**` (cross-arg wildcard) in `internal/permissions/pattern.go` + `internal/permissions/pattern_test.go` (TDD: test "*" → wildcard; test "**" → doubleWildcard; test "--*" → literal with embedded wildcard; test "pre*fix" → literal with embedded wildcard; test "*suffix" → literal with embedded wildcard; test "push **" → literal + doubleWildcard)
- [ ] T006 [P] Implement pattern parsing for `.` (single char wildcard) in `internal/permissions/pattern.go` + `internal/permissions/pattern_test.go` (TDD: test "." → singleChar; test "v." → literal-with-dot; test "a.b" → literal-with-embedded-dot; test ".v" → dot-prefixed literal; test ".." → two singleChars)
- [ ] T007 [P] Implement pattern parsing for `/regex/` and `/regex/**` in `internal/permissions/pattern.go` + `internal/permissions/pattern_test.go` (TDD: test "/^https/" → regex element; test "/^https/**" → regexMulti; test "/invalid[/" → error with pattern context; test "/pattern\\/with\\/slashes/" → regex with escaped slashes)
- [ ] T008 [P] Implement pattern parsing for `~` (non-positional) modifier in `internal/permissions/pattern.go` + `internal/permissions/pattern_test.go` (TDD: test "~--force" → literal with NonPositional=true; test "push ~--force" → literal + non-positional literal)
- [ ] T009 [P] Implement pattern parsing for `?` (optional) modifier in `internal/permissions/pattern.go` + `internal/permissions/pattern_test.go` (TDD: test "origin?" → literal with Optional=true; test "pull origin?" → literal + optional literal)
- [ ] T010 [P] Implement pattern parsing for quoted strings (`""` and `''`) and `\` escape in `internal/permissions/pattern.go` + `internal/permissions/pattern_test.go` (TDD: test `"my file"` → quoted element; test `'my file'` → quoted element; test `\*` → escaped literal asterisk; test `\.` → escaped literal dot; test unclosed double quote → error; test unclosed single quote → error)
- [ ] T011 Implement pattern parsing for `()` grouping in `internal/permissions/pattern.go` + `internal/permissions/pattern_test.go` (TDD: test "(origin main)?" → group with Optional=true containing two elements; test "~(-n 0)" → non-positional group containing two elements; test "(origin main)" without modifier → group equivalent to bare elements; test unclosed paren → error; depends on T008 for `~` and T009 for `?` support)

**Checkpoint**: Pattern parser complete — all token types parse correctly with full test coverage.

---

## Phase 3: Foundational — Pattern Matching Engine

**Purpose**: The matching engine evaluates compiled patterns against actual command arguments. Depends on parser being complete.

- [ ] T012 Implement literal matching: exact positional arg matching in `internal/permissions/match.go` + `internal/permissions/match_test.go` (TDD: test "pull" matches ["pull"] → true; test "pull" matches ["push"] → false)
- [ ] T013 Implement prefix matching (default) and exact-match ($) in `internal/permissions/match.go` + `internal/permissions/match_test.go` (TDD: test "status" matches ["status", "--short"] → true; test "status$" matches ["status", "--short"] → false; test "status$" matches ["status"] → true)
- [ ] T014 Implement `*` wildcard matching within a single arg in `internal/permissions/match.go` + `internal/permissions/match_test.go` (TDD: test "*" matches ["anything"] → true; test "--*" matches ["--verbose"] → true; test "--*" matches ["verbose"] → false; test "pre*fix" matches ["prefix"] → true; test "pre*fix" matches ["pre-blah-fix"] → true; test "*suffix" matches ["my-suffix"] → true; test "*" does NOT match across args)
- [ ] T015 Implement `**` double wildcard matching across args in `internal/permissions/match.go` + `internal/permissions/match_test.go` (TDD: test "push **" matches ["push", "origin", "main"] → true; test "**" matches any args → true; test "push **" matches ["push"] → true (zero trailing args); test "**" matches [] → true)
- [ ] T016 [P] Implement `.` single-char matching in `internal/permissions/match.go` + `internal/permissions/match_test.go` (TDD: test "." matches ["x"] → true; test "." matches ["xx"] → false; test "v." matches ["v1"] → true; test "a.b" matches ["axb"] → true; test "a.b" matches ["ab"] → false; test ".v" matches ["xv"] → true)
- [ ] T017 [P] Implement `/regex/` matching against single arg in `internal/permissions/match.go` + `internal/permissions/match_test.go` (TDD: test `/^https?:\/\//` matches ["https://github.com"] → true; test `/^https?:\/\//` matches ["ftp://server"] → false)
- [ ] T018 [P] Implement `/regex/**` matching across multiple args in `internal/permissions/match.go` + `internal/permissions/match_test.go` (TDD: test `/--force|--hard/**` matches ["--force"] → true; test `/--force|--hard/**` matches ["origin", "--force", "main"] → true)
- [ ] T019 Implement `~` non-positional matching in `internal/permissions/match.go` + `internal/permissions/match_test.go` (TDD: test "push ~--force" matches ["push", "--force", "origin"] → true; test "push ~--force" matches ["push", "origin", "--force", "main"] → true; test "push ~--force" matches ["push", "origin", "main"] → false)
- [ ] T020 Implement `?` optional matching in `internal/permissions/match.go` + `internal/permissions/match_test.go` (TDD: test "pull origin?" matches ["pull"] → true; test "pull origin?" matches ["pull", "origin"] → true; test "pull origin?" matches ["pull", "upstream"] → false)
- [ ] T021 Implement `()` group matching in `internal/permissions/match.go` + `internal/permissions/match_test.go` (TDD: test "(origin main)?" matches ["origin", "main"] → true; test "(origin main)?" matches [] → true with prefix matching; test "(origin main)?$" matches [] → true; test "~(-n 0)" matches ["cmd", "foo", "-n", "0", "bar"] → true; test "~(-n 0)" matches ["cmd", "foo"] → false)
- [ ] T022 Implement quoted string exact matching in `internal/permissions/match.go` + `internal/permissions/match_test.go` (TDD: test `"my file"` matches ["my file"] → true; test `'my file'` matches ["my file"] → true; test `"my file"` matches ["my"] → false; test `"v."` matches ["v."] → true and does NOT glob-match ["v1"])
- [ ] T023 Implement `\` escape matching in `internal/permissions/match.go` + `internal/permissions/match_test.go` (TDD: test `\*` matches ["*"] → true; test `\*` matches ["foo"] → false)

**Checkpoint**: Full pattern matching engine complete. Every syntax element can be parsed and matched.

---

## Phase 4: User Story 1 — Define Allowed Passthrough Commands in Permissions File (Priority: P1) 🎯 MVP

**Goal**: Users can declare passthrough commands in `.ccbox/permissions.{json,yml,yaml}` and have them merged with CLI flags.

**Independent Test**: Create a permissions file with allowed commands, verify ccbox enables passthrough for exactly those commands plus any from CLI flags.

Each task below includes TDD: write failing test → implement → refactor.

- [ ] T024 [US1] Implement `permissions.Load()` to discover and parse `.ccbox/permissions.{json,yml,yaml}` via `common/settings.Load` in `internal/permissions/config.go` + `internal/permissions/config_test.go` (TDD: test valid YAML → PermissionsConfig with correct commands; test valid JSON → same; test no file → nil config, nil error)
- [ ] T025 [US1] Implement `permissions.Load()` validation: malformed file → error, invalid effect → error, empty command name → error in `internal/permissions/config.go` + `internal/permissions/config_test.go` (TDD: test malformed YAML → error with file context; test effect "block" → error; test empty key → error)
- [ ] T026 [US1] Implement `permissions.NewChecker()` with CLI merge logic: CLI commands get implicit `allow **` first rule, file rules appended after in `internal/permissions/checker.go` + `internal/permissions/checker_test.go` (TDD: test CLI pt:git + file deny rules → allow ** prepended; test CLI pt:git no file → allow ** only; test file only → file rules as-is; test CLI + file null command → unrestricted)
- [ ] T027 [US1] Implement `Checker.Commands()` returning deduplicated command names in `internal/permissions/checker.go` + `internal/permissions/checker_test.go` (TDD: test CLI + file commands → all present, deduplicated; test file only → file commands; test CLI only → CLI commands)
- [ ] T028 [US1] Implement `NewChecker()` pattern validation at construction: invalid regex → error, `$` mid-pattern → error in `internal/permissions/checker.go` + `internal/permissions/checker_test.go` (TDD: test rule with `/[/` pattern → error identifying the pattern; test rule with "push $ --force" → error)
- [ ] T029 [US1] Implement backward-compatible no-permissions-file path: when `Load()` returns nil, `NewChecker(nil, cliPassthrough)` creates allow-all entries for CLI commands in `internal/permissions/checker.go` + `internal/permissions/checker_test.go` (TDD: test nil config + CLI pt:git → git allowed with allow **; test nil config + no CLI → empty checker, Commands() returns [])

**Checkpoint**: Config loading, CLI merge, and Checker construction are fully tested. Ready for rule evaluation.

---

## Phase 5: User Story 2 — Cascading Permission Rules with Pattern Matching (Priority: P1)

**Goal**: Users configure ordered pattern/effect rules per command; last matching rule wins.

**Independent Test**: Configure cascading rules for `git` (deny **, allow pull), verify `git pull` succeeds and `git push` is denied.

Each task below includes TDD: write failing test → implement → refactor.

- [ ] T030 [US2] Implement `Checker.Check()` core: split command string into name + args, look up command, evaluate rules top-to-bottom with last-match-wins in `internal/permissions/checker.go` + `internal/permissions/checker_test.go` (TDD: test [deny **, allow pull] → "git pull" allowed, "git push" denied; test [allow **, deny "push ~--force"] → "git push --force origin" denied, "git push origin" allowed)
- [ ] T031 [US2] Implement `Check()` for commands with no rules (null/empty value) → allow all in `internal/permissions/checker.go` + `internal/permissions/checker_test.go` (TDD: test command with nil rules → all subcommands allowed; test command with empty rules array → all allowed)
- [ ] T032 [US2] Implement `Check()` with `$` exact-match patterns in `internal/permissions/checker.go` + `internal/permissions/checker_test.go` (TDD: test rule "status$" → "git status" allowed, "git status --short" denied because $ prevents prefix match)
- [ ] T033 [US2] Implement `Check()` with regex patterns in `internal/permissions/checker.go` + `internal/permissions/checker_test.go` (TDD: test rule `clone /https?:\/\//` → "git clone https://github.com/repo" allowed; "git clone git@github.com:repo" denied)
- [ ] T034 [US2] Implement `Check()` for command not in permissions → denied in `internal/permissions/checker.go` + `internal/permissions/checker_test.go` (TDD: test checker with git only → "npm install" → MatchResult{Allowed: false, Reason: "command not configured"})
- [ ] T035 [US2] Implement `Check()` with array patterns (pattern: ["a", "b"] shorthand) in `internal/permissions/checker.go` + `internal/permissions/checker_test.go` (TDD: test rule {pattern: ["build", "test", "lint"], effect: allow} → "make build" allowed, "make deploy" denied)

**Checkpoint**: Full cascading rule evaluation works. All acceptance scenarios from US2 pass.

---

## Phase 6: User Story 3 — Default Effect When No Rule Matches (Priority: P2)

**Goal**: When rules exist but none match, the command is denied (fail-closed).

**Independent Test**: Define a single allow rule, verify commands not matching it are denied.

Each task below includes TDD: write failing test → implement → refactor.

- [ ] T036 [US3] Implement fail-closed default in `Checker.Check()`: rules exist but no rule matches → deny in `internal/permissions/checker.go` + `internal/permissions/checker_test.go` (TDD: test [allow "status"] → "git push" denied with "no matching rule"; test [allow "status"] → "git status" allowed)
- [ ] T037 [US3] Verify no-rules vs has-rules distinction: no rules → allow all, has rules but no match → deny in `internal/permissions/checker.go` + `internal/permissions/checker_test.go` (TDD: test command with nil rules → any subcommand allowed; test command with [deny "**"] → no rule can ever allow; test only deny rules → everything denied)

**Checkpoint**: Fail-closed behavior verified. Security posture correct.

---

## Phase 7: User Story 4 — Hierarchical Permissions File Discovery (Priority: P2)

**Goal**: Permissions files discovered using the same hierarchical walk as settings files.

**Independent Test**: Place permissions files at two directory levels, verify correct merge behavior.

Each task below includes TDD: write failing test → implement → refactor.

- [ ] T038 [US4] Verify hierarchical discovery via `common/settings.Load` in `internal/permissions/config_test.go` (TDD: test parent permissions with `git` + child permissions with `npm` → both present; test parent and child with overlapping `git` → child's rules take precedence. Uses temp directories with `.ccbox/permissions.yaml` at each level.)

**Checkpoint**: Hierarchical merge works correctly. `common/settings` integration verified.

---

## Phase 8: User Story 5 — Clear Denial Messages (Priority: P3)

**Goal**: Denied commands produce clear messages explaining which rule blocked the command.

**Independent Test**: Run a denied command, verify the error message includes the command, rule, reason, and guidance.

Each task below includes TDD: write failing test → implement → refactor.

- [ ] T039 [US5] Implement denial message for rule-based deny: include command, blocking rule pattern, and configured `reason` in `MatchResult.Reason` in `internal/permissions/checker.go` + `internal/permissions/checker_test.go` (TDD: test deny rule with reason "Force push is destructive" → message includes the reason; test deny rule without reason → message includes the blocking pattern)
- [ ] T040 [US5] Implement denial message for no-match default deny: include command and list of available patterns in `MatchResult.Reason` in `internal/permissions/checker.go` + `internal/permissions/checker_test.go` (TDD: test no-match with patterns ["status", "pull"] → message says no rule matched and lists available patterns)

**Checkpoint**: All denial messages are clear and actionable.

---

## Phase 9: Integration — Wire Permissions into ccbox

**Purpose**: Connect the isolated `internal/permissions` package to the host-side enforcement point.

- [ ] T041 Implement permission-aware exec handler wrapper in `internal/cmdpassthrough/exec.go` + `internal/cmdpassthrough/exec_test.go` (TDD: test wrapper with denied command → returns exit code 1 + denial message without executing; test wrapper with allowed command → delegates to HandleExec; test wrapper with nil checker → delegates to HandleExec unconditionally for backward compat)
- [ ] T042 Wire permissions loading and checker creation into `cmd/ccbox/orchestrate.go`: load permissions config, create checker, pass to bridge server via wrapped exec handler (TDD: test orchestration with permissions file → checker created and wired; test orchestration without permissions file → backward-compatible behavior)
- [ ] T043 Update `Checker.Commands()` integration: ensure command list from checker is used for container shim creation and proxy config in `cmd/ccbox/orchestrate.go` (TDD: test that commands from both permissions file and CLI flags produce shims in container)

**Checkpoint**: End-to-end flow works: permissions file → load → validate → wire → enforce on exec.

---

## Phase 10: Polish & Cross-Cutting Concerns

**Purpose**: Edge cases, backward compatibility verification, and cleanup.

- [ ] T044 [P] Add edge case tests for empty permissions file (no commands added), only-deny rules (all denied), pattern with `**` and fewer args than expected (no match, no error) in `internal/permissions/checker_test.go`
- [ ] T045 [P] Run full test suite and verify all existing tests still pass (backward compatibility: no permissions file + CLI -pt:git → all git commands work as before)
- [ ] T046 Run quickstart.md validation: verify the example configuration in quickstart.md parses and evaluates correctly

---

## Dependencies & Execution Order

### Phase Dependencies

- **Phase 1 (Setup)**: No dependencies — start immediately
- **Phase 2 (Pattern Parser)**: Depends on Phase 1 (types)
- **Phase 3 (Pattern Matching)**: Depends on Phase 2 (parser)
- **Phase 4 (US1)**: Depends on Phase 2 (parser for validation)
- **Phase 5 (US2)**: Depends on Phase 3 (matching) + Phase 4 (config loading)
- **Phase 6 (US3)**: Depends on Phase 5 (Check() exists)
- **Phase 7 (US4)**: Depends on Phase 4 (config loading)
- **Phase 8 (US5)**: Depends on Phase 5 (Check() exists)
- **Phase 9 (Integration)**: Depends on Phase 5, 6, 8
- **Phase 10 (Polish)**: Depends on Phase 9

### User Story Dependencies

- **US1 (P1)**: Depends on Foundational (Phase 2) — no dependency on other stories
- **US2 (P1)**: Depends on US1 (needs config loading) + Phase 3 (needs matching engine)
- **US3 (P2)**: Depends on US2 (needs Check() to add default deny behavior)
- **US4 (P2)**: Depends on US1 (needs config loading) — independent of US2/US3
- **US5 (P3)**: Depends on US2 (needs Check() to format messages)

### Parallel Opportunities

**Within Phase 2** (after T003, T004):
- T005, T006, T007, T008, T009, T010 can all run in parallel (different token types, independent parse logic)

**Within Phase 3** (after T012-T015):
- T016, T017, T018 can run in parallel (different match types)

**Between Phases**:
- Phase 4 (US1) can start after Phase 2 completes — does not need Phase 3
- Phase 7 (US4) can start as soon as Phase 4 completes — independent of US2/US3
- Phase 8 (US5) can start as soon as Phase 5 completes

---

## Execution Protocol (Agent Teams)

**⚠️ CRITICAL**: This section defines HOW tasks are executed. Follow this protocol exactly.

### Step 1: Create the Team

Use `TeamCreate` to create a team for the feature implementation:

```
TeamCreate({
  team_name: "passthrough-permissions-impl",
  description: "Implementing Passthrough Command Permissions per tasks.md"
})
```

### Step 2: Create Tasks

Use `TaskCreate` to transform every task from this file into a team task. Each task description MUST include:

- The exact task ID and description from this file
- The TDD requirement: "Follow Red-Green-Refactor: write a failing test first, implement the minimum to pass, then refactor while green."
- File paths to create/modify
- Dependencies (which task IDs must complete first)

### Step 3: Spawn Teammates

**ONE TEAMMATE PER TASK. NO EXCEPTIONS.**

- Spawn a teammate using the `Agent` tool with `team_name` set to the team name.
- Assign exactly ONE task to each teammate via `TaskUpdate` (set `owner`).
- For tasks marked `[P]` with no unresolved dependencies, spawn teammates in parallel.
- For sequential tasks, wait for the blocking task's teammate to finish before spawning the next.

### Step 4: Teammate Lifecycle

Each teammate MUST:

1. Read its assigned task via `TaskGet`
2. Execute the task following Red-Green-Refactor
3. Mark the task as `completed` via `TaskUpdate`
4. **Shut down immediately** — send a shutdown acknowledgment and terminate

**Context rot prevention**: A teammate MUST NOT be reused for a second task. Once a teammate completes its task and shuts down, spawn a **new** teammate for the next task. This ensures every task starts with a fresh context window.

### Step 5: Orchestration Loop

The team lead (you) orchestrates:

```
while uncompleted tasks exist:
  1. Check TaskList for completed tasks
  2. For each newly completed task:
     - Verify the teammate has shut down
     - Check if any blocked tasks are now unblocked
  3. For each unblocked task without an owner:
     - Spawn a NEW teammate
     - Assign the task
  4. At each phase checkpoint:
     - Verify all phase tasks are complete
     - Run the full test suite
     - Only proceed to next phase if green
```

### Step 6: Cleanup

After all tasks are complete:

1. Run the full test suite one final time
2. Shut down any remaining teammates via `SendMessage` with `shutdown_request`
3. Clean up the team via `TeamDelete`

---

## Implementation Strategy

### MVP First (User Story 1 + 2 Only)

1. Complete Phase 1: Setup (types)
2. Complete Phase 2: Pattern Parser (foundational)
3. Complete Phase 3: Pattern Matching Engine (foundational)
4. Complete Phase 4: User Story 1 (config loading + CLI merge)
5. Complete Phase 5: User Story 2 (cascading rules + Check())
6. **STOP and VALIDATE**: Run full test suite, test US1+US2 independently
7. Wire integration (Phase 9) for a working end-to-end demo

### Incremental Delivery

1. Setup + Foundational (Phases 1-3) → Pattern engine ready
2. Add US1 (Phase 4) → Config loading works → All tests green
3. Add US2 (Phase 5) → Cascading rules work → All tests green → **MVP!**
4. Add US3 (Phase 6) → Fail-closed default verified → All tests green
5. Add US4 (Phase 7) → Hierarchical discovery works → All tests green
6. Add US5 (Phase 8) → Clear denial messages → All tests green
7. Integration (Phase 9) → End-to-end wiring → All tests green
8. Polish (Phase 10) → Edge cases, backward compat → **Done!**

### Parallel Team Strategy

Using the agent team protocol above:

1. Team lead completes Setup (Phase 1) sequentially
2. Phase 2: After T003-T004, spawn 6 parallel teammates for T005-T010
3. Phase 3: After T012-T015, spawn 3 parallel teammates for T016-T018
4. Phase 4 (US1) can start once Phase 2 is done — run alongside Phase 3 if staffed
5. Phase 7 (US4) can start once Phase 4 is done — run alongside Phase 5
6. Phase 8 (US5) can start once Phase 5 is done — run alongside Phase 6

---

## Notes

- [P] tasks = different files, no dependencies — can be assigned to parallel teammates
- [Story] label maps task to specific user story for traceability
- Each user story should be independently completable and testable
- Every code task includes TDD — there are no separate "write tests" tasks
- Tasks should be small: one tested behavior per task
- Commit after each task completes (teammate responsibility)
- Stop at any checkpoint to validate story independently
- One teammate per task, always — no reuse, no context rot
- The `internal/permissions` package has ZERO dependencies on the rest of ccbox — it is a pure logic package tested entirely in isolation
