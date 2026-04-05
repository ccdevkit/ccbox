# Tasks: Hook Integration

**Input**: Design documents from `/specs/003-hook-integration/`
**Prerequisites**: plan.md, spec.md, research.md, data-model.md, contracts/

**TDD**: Every task that produces code MUST follow Red-Green-Refactor. Tests are NOT separate tasks — each task includes writing the failing test, making it pass, and refactoring. This is non-negotiable per the project constitution (Principle VII).

**Task Granularity**: Each task MUST be small enough that the full TDD cycle (write failing test → implement → refactor) is a single coherent unit of work. If a task feels too large, split it. A good task produces one tested behavior.

**Organization**: Tasks are grouped by user story to enable independent implementation and testing of each story.

## Format: `[ID] [P?] [Story] Description`

- **[P]**: Can run in parallel (different files, no dependencies)
- **[Story]**: Which user story this task belongs to (e.g., US1, US2, US3)
- Include exact file paths in descriptions

---

## Phase 1: Setup

**Purpose**: Create package directories and foundational type definitions that multiple stories depend on.

- [x] T001 Define HookEvent enum constants for all 26 hook events in `internal/claude/hooks/events.go` + `internal/claude/hooks/events_test.go` (TDD: test that each constant's string value matches Claude Code's expected event name)
- [x] T002 Define HookInputBase and HookOutputBase structs in `internal/claude/hooks/types.go` + `internal/claude/hooks/types_test.go` (TDD: test JSON unmarshal of base fields from sample hook input)
- [x] T003 Add HookRequestType constant and proxy path constants to `internal/constants/constants.go` + `internal/constants/constants_test.go` (TDD: test constant values match expected strings)

---

## Phase 2: Foundational (Blocking Prerequisites)

**Purpose**: Typed input/output structs for all 26 events — required by both the registry (US1) and the bridge/proxy (US3).

**CRITICAL**: No user story work can begin until this phase is complete.

- [x] T004 [P] Define typed input/output structs for Lifecycle events (SessionStart, SessionEnd) in `internal/claude/hooks/types.go` + `internal/claude/hooks/types_test.go` (TDD: test JSON round-trip for SessionStartInput with event-specific fields like source, model)
- [x] T005 [P] Define typed input/output structs for Instruction and Prompt events (InstructionsLoaded, UserPromptSubmit) in `internal/claude/hooks/types.go` + `internal/claude/hooks/types_test.go` (TDD: test JSON round-trip for UserPromptSubmitInput with prompt field)
- [x] T006 [P] Define typed input/output structs for Tool events (PreToolUse, PostToolUse, PostToolUseFailure, PermissionRequest, PermissionDenied) in `internal/claude/hooks/types.go` + `internal/claude/hooks/types_test.go` (TDD: test JSON unmarshal of PreToolUseInput with tool_name and tool_input fields; test PreToolUseOutput with hookSpecificOutput containing permissionDecision)
- [x] T007 [P] Define typed input/output structs for Agent/Task events (SubagentStart, SubagentStop, TaskCreated, TaskCompleted, TeammateIdle) in `internal/claude/hooks/types.go` + `internal/claude/hooks/types_test.go` (TDD: test JSON round-trip for SubagentStartInput with subagent fields)
- [x] T008 [P] Define typed input/output structs for Workflow and Compact events (Stop, StopFailure, PreCompact, PostCompact) in `internal/claude/hooks/types.go` + `internal/claude/hooks/types_test.go` (TDD: test JSON round-trip for StopInput with reason field)
- [x] T009 [P] Define typed input/output structs for File/Config, Worktree, MCP, and Other events (FileChanged, CwdChanged, ConfigChange, WorktreeCreate, WorktreeRemove, Elicitation, ElicitationResult, Notification) in `internal/claude/hooks/types.go` + `internal/claude/hooks/types_test.go` (TDD: test JSON round-trip for FileChangedInput with file_path field)
- [x] T010 Define per-event handler structs implementing HookHandler interface for all 26 events in `internal/claude/hooks/types.go` + `internal/claude/hooks/types_test.go` (TDD: test that PreToolUseHandler.invoke() unmarshals raw JSON into typed input, calls Fn, and returns correct HandlerResult)

**Checkpoint**: All 26 typed event structs and handler types are defined and tested. Registry and bridge work can begin.

---

## Phase 3: User Story 1 — Internal Hook Handler Registration (Priority: P1) MVP

**Goal**: ccbox developers can register typed hook handlers and dispatch events to them. This is the core hook infrastructure.

**Independent Test**: Register a handler for PreToolUse, call Dispatch with sample JSON input, verify the handler receives typed input and returns a valid HandlerResult.

Each task below includes TDD: write failing test -> implement -> refactor.

- [x] T011 [US1] Implement Registry.Register() — store a single HookHandler by event name in `internal/claude/hooks/registry.go` + `internal/claude/hooks/registry_test.go` (TDD: register a PreToolUseHandler, verify it's stored and retrievable)
- [x] T012 [US1] Implement Registry.Dispatch() for single handler — find matching handler by event, invoke it, return result in `internal/claude/hooks/registry.go` + `internal/claude/hooks/registry_test.go` (TDD: register handler, dispatch matching event, verify handler's result is returned)
- [x] T013 [US1] Implement Registry.Dispatch() with no matching handler — return default success (exit 0, empty output) in `internal/claude/hooks/registry.go` + `internal/claude/hooks/registry_test.go` (TDD: dispatch event with no registered handler, verify exit 0 and empty stdout)
- [x] T014 [US1] Implement matcher filtering in Dispatch — handler with matcher "Bash" only invoked for Bash tool_name inputs in `internal/claude/hooks/registry.go` + `internal/claude/hooks/registry_test.go` (TDD: register handler with matcher "Bash", dispatch with tool_name "Bash" → invoked; dispatch with tool_name "Read" → not invoked)
- [x] T015 [US1] Implement multiple handler aggregation in Dispatch — block (exit 2) wins over allow (exit 0) in `internal/claude/hooks/registry.go` + `internal/claude/hooks/registry_test.go` (TDD: register two handlers for same event, one returns exit 0 and one returns exit 2, verify dispatch returns exit 2)
- [x] T016 [US1] Implement full decision precedence in Dispatch — deny > defer > ask > allow in `internal/claude/hooks/registry.go` + `internal/claude/hooks/registry_test.go` (TDD: register handlers returning different permission decisions for PreToolUse, verify deny wins over defer, defer wins over ask, ask wins over allow)
- [x] T017 [US1] Implement continue:false halting in Dispatch — any handler returning continue:false produces stopReason in result in `internal/claude/hooks/registry.go` + `internal/claude/hooks/registry_test.go` (TDD: register two handlers, one returns continue:false with stopReason, verify dispatch result has continue:false and stopReason regardless of other handler's output)
- [x] T018 [US1] Implement Registry.HookEntries() — generate settings.json hook configuration with correct matcher groups and before/after ordering in `internal/claude/hooks/registry.go` + `internal/claude/hooks/registry_test.go` (TDD: register handlers with different matchers and orders, verify HookEntries produces correct nested structure with proxy command)

**Checkpoint**: Hook registry is fully functional. Handlers can be registered, dispatched, and produce settings entries.

---

## Phase 4: User Story 2 — Settings Manager (Priority: P2)

**Goal**: General-purpose Settings Manager that discovers, merges, and modifies Claude Code settings. Hook registry is one consumer.

**Independent Test**: Provide mock settings files, call Set/MergeHooks/Finalize, verify output contains all user entries plus injected entries in correct precedence.

Each task below includes TDD: write failing test -> implement -> refactor.

- [x] T019 [US2] Implement settings file discovery — find and read 4 settings files by precedence in `internal/claude/settings/manager.go` + `internal/claude/settings/manager_test.go` (TDD: create mock FS with 4 files, verify NewClaudeSettingsManager reads them in correct order; test missing files are skipped)
- [x] T020 [US2] Implement settings merge by precedence — higher-precedence file wins for top-level keys in `internal/claude/settings/manager.go` + `internal/claude/settings/manager_test.go` (TDD: two files with same key, verify project-level value overrides user-level)
- [x] T021 [US2] Implement malformed JSON handling — skip file with warning, merge remaining files in `internal/claude/settings/manager.go` + `internal/claude/settings/manager_test.go` (TDD: one valid file + one malformed file, verify valid file's content present and no error returned)
- [x] T022 [US2] Implement Set() and SetDeep() — set top-level and nested keys in merged settings in `internal/claude/settings/manager.go` + `internal/claude/settings/manager_test.go` (TDD: call Set("key", value), verify it appears in merged output; call SetDeep("a.b.c", value), verify nested structure created)
- [x] T023 [US2] Implement MergeHooks() with before ordering — ccbox hook entries appear before user hooks in matcher group in `internal/claude/settings/manager.go` + `internal/claude/settings/manager_test.go` (TDD: user has existing PreToolUse hook, MergeHooks with order "before", verify ccbox entry is first in hooks array)
- [x] T024 [US2] Implement MergeHooks() with after ordering — ccbox hook entries appear after user hooks in matcher group in `internal/claude/settings/manager.go` + `internal/claude/settings/manager_test.go` (TDD: user has existing PreToolUse hook, MergeHooks with order "after", verify ccbox entry is last in hooks array)
- [x] T025 [US2] Implement Finalize() — write merged settings to session file, return CLI args in `internal/claude/settings/manager.go` + `internal/claude/settings/manager_test.go` (TDD: call Finalize, verify file written via mock SessionFileWriter, verify returned args contain --settings and --setting-sources "")
- [x] T026 [US2] Implement Finalize guard — panic on Set/SetDeep/MergeHooks after Finalize in `internal/claude/settings/manager.go` + `internal/claude/settings/manager_test.go` (TDD: call Finalize, then call Set, verify panic)
- [x] T027 [US2] Implement user settings preservation — module settings additive, not overwriting user keys in `internal/claude/settings/manager.go` + `internal/claude/settings/manager_test.go` (TDD: user has allowedTools=["X"], Set allowedTools=["Y"], verify both present or user value preserved per FR-011)
- [x] T028 [US2] Integrate Settings Manager into Claude.New() and BuildRunSpec() — replace hardcoded settingsJSON with Settings Manager in `internal/claude/claude.go` + `internal/claude/session_files.go` (TDD: verify BuildRunSpec output includes --settings and --setting-sources args; verify old hardcoded settingsJSON path is no longer used)
- [x] T029 [US2] Migrate and remove `internal/settings/claude_settings.go` — ensure all callers use new Settings Manager in `internal/claude/settings/manager.go` (TDD: verify ReadClaudeSettings and MergeClaudeSettings functionality is covered by Settings Manager tests; remove old file)

**Checkpoint**: Settings Manager is fully functional. Claude Code settings are discovered, merged, and injected via CLI args.

---

## Phase 5: User Story 3 — Container Proxy Binary for Hook Dispatch (Priority: P3)

**Goal**: Container-side proxy binary forwards hook events over TCP to host and returns responses. Bridge protocol extended with hook message type.

**Independent Test**: Run proxy binary with mock stdin, connect to mock TCP server, verify correct request forwarding and response translation.

Each task below includes TDD: write failing test -> implement -> refactor.

- [x] T030 [P] [US3] Define HookRequest and HookResponse wire types in `internal/bridge/types.go` + `internal/bridge/types_test.go` (TDD: test JSON marshal/unmarshal of HookRequest with type "hook", event name, and raw input; test HookResponse with exit_code, stdout, stderr)
- [x] T031 [US3] Add HookHandler type and hook dispatch to bridge Server in `internal/bridge/server.go` + `internal/bridge/server_test.go` (TDD: create server with hook handler, send hook request over TCP, verify handler invoked and response returned)
- [x] T032 [US3] Wire Registry.BridgeHandler() as the bridge HookHandler — connect registry dispatch to bridge server in `internal/claude/hooks/registry.go` + `internal/claude/hooks/registry_test.go` (TDD: register a handler, create bridge handler via BridgeHandler(), invoke with HookRequest, verify correct dispatch and response)
- [x] T033 [US3] Implement cchookproxy binary — read stdin, parse event, dial TCP, send request, write response, exit with code in `cmd/cchookproxy/main.go` + `cmd/cchookproxy/main_test.go` (TDD: start mock TCP server, run proxy with piped stdin containing hook JSON, verify request sent and stdout/stderr/exit code match response)
- [x] T034 [US3] Implement cchookproxy TCP failure handling — exit 1 on connection failure with stderr message in `cmd/cchookproxy/main.go` + `cmd/cchookproxy/main_test.go` (TDD: run proxy with no TCP server listening, verify exit code 1 and stderr contains "connection failed")
- [x] T035 [US3] Implement cchookproxy response timeout — exit 1 after 10 seconds with stderr message in `cmd/cchookproxy/main.go` + `cmd/cchookproxy/main_test.go` (TDD: start TCP server that never responds, run proxy, verify exits with code 1 and stderr contains "response timeout" — use short timeout in test)
- [x] T036 [US3] Implement cchookproxy exit code 2 handling — blocking response writes stderr and exits 2 in `cmd/cchookproxy/main.go` + `cmd/cchookproxy/main_test.go` (TDD: TCP server returns exit_code 2 with stderr text, verify proxy exits 2 and stderr contains the text)
- [x] T037 [US3] Update NewServer constructor to accept hook handler — `NewServer(exec, log, hook)` in `internal/bridge/server.go` + `internal/bridge/server_test.go` (TDD: verify NewServer accepts three handlers and existing exec/log tests still pass)

**Checkpoint**: Full hook pipeline works end-to-end: Claude Code fires hook → cchookproxy reads stdin → TCP to host → bridge dispatches to registry → response flows back.

---

## Phase 6: Polish & Cross-Cutting Concerns

**Purpose**: Integration, cleanup, and validation across all stories.

- [x] T038 End-to-end integration test: register hook handler, build session with Settings Manager, verify settings file contains hook config pointing to cchookproxy in `internal/claude/claude_test.go` (TDD: test full flow from handler registration through settings finalization)
- [x] T039 Run quickstart.md validation — verify code examples in `specs/003-hook-integration/quickstart.md` compile and match actual API

---

## Dependencies & Execution Order

### Phase Dependencies

- **Setup (Phase 1)**: No dependencies — can start immediately. T001, T002, T003 can run in parallel.
- **Foundational (Phase 2)**: Depends on Phase 1 (T001, T002). T004–T009 can all run in parallel. T010 depends on T004–T009.
- **User Story 1 (Phase 3)**: Depends on Phase 2 (T010). T011–T018 are sequential within the story.
- **User Story 2 (Phase 4)**: Depends on Phase 2 (T010) for MergeHooks, but T019–T022 can start after Phase 1. T023–T024 depend on US1 T018 (HookEntries). T028 depends on T025.
- **User Story 3 (Phase 5)**: T030 can start after Phase 1. T031 depends on T030. T032 depends on US1 (T018). T033–T036 depend on T030–T031. T037 depends on T031.
- **Polish (Phase 6)**: Depends on all user stories being complete.

### User Story Dependencies

- **US1 (P1)**: Can start after Foundational (Phase 2) — no dependencies on other stories
- **US2 (P2)**: Core discovery/merge (T019–T022) can start after Phase 1. Hook merging (T023–T024) needs US1's HookEntries (T018). Integration (T028–T029) needs T025.
- **US3 (P3)**: Wire types (T030) can start after Phase 1. Bridge dispatch (T031) needs T030. Registry bridge (T032) needs US1. Proxy binary (T033–T036) needs T030–T031.

### Parallel Opportunities

**Phase 1**: T001, T002, T003 — all parallel (different files)
**Phase 2**: T004, T005, T006, T007, T008, T009 — all parallel (same file but independent struct groups)
**Cross-story**: After Phase 2, US1 T011+ and US2 T019–T022 and US3 T030 can start in parallel
**Within US3**: T030 is independent of US1/US2, enabling early start

---

## Execution Protocol (Agent Teams)

**CRITICAL**: This section defines HOW tasks are executed. Follow this protocol exactly.

### Step 1: Create the Team

Use `TeamCreate` to create a team for the feature implementation:

```
TeamCreate({
  team_name: "hook-integration-impl",
  description: "Implementing Hook Integration per tasks.md"
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

### MVP First (User Story 1 Only)

1. Complete Phase 1: Setup
2. Complete Phase 2: Foundational
3. Complete Phase 3: User Story 1 — Hook Handler Registration
4. **STOP and VALIDATE**: Run full test suite, verify handlers register and dispatch correctly
5. Proceed to US2 + US3

### Incremental Delivery

1. Setup + Foundational → Type system ready
2. Add US1 → Hook registry functional → All tests green
3. Add US2 → Settings Manager functional → All tests green
4. Add US3 → Full pipeline working end-to-end → All tests green
5. Each story adds value without breaking previous stories

### Parallel Team Strategy

Using the agent team protocol above:

1. Team lead orchestrates Phase 1 (3 parallel tasks)
2. Phase 2: spawn 6 parallel teammates for T004–T009, then T010 sequentially
3. After Phase 2, spawn parallel teammates:
   - Teammate A: US1 T011 (registry)
   - Teammate B: US2 T019 (settings discovery)
   - Teammate C: US3 T030 (wire types)
4. As each teammate finishes and shuts down, spawn new teammates for the next tasks in each story
5. Stories complete and integrate independently

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
