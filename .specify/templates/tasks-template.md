---

description: "Task list template for feature implementation"
---

# Tasks: [FEATURE NAME]

**Input**: Design documents from `/specs/[###-feature-name]/`
**Prerequisites**: plan.md (required), spec.md (required for user stories), research.md, data-model.md, contracts/

**TDD**: Every task that produces code MUST follow Red-Green-Refactor. Tests are NOT separate tasks — each task includes writing the failing test, making it pass, and refactoring. This is non-negotiable per the project constitution (Principle VII).

**Task Granularity**: Each task MUST be small enough that the full TDD cycle (write failing test → implement → refactor) is a single coherent unit of work. If a task feels too large, split it. A good task produces one tested behavior.

**Organization**: Tasks are grouped by user story to enable independent implementation and testing of each story.

## Format: `[ID] [P?] [Story] Description`

- **[P]**: Can run in parallel (different files, no dependencies)
- **[Story]**: Which user story this task belongs to (e.g., US1, US2, US3)
- Include exact file paths in descriptions

## Path Conventions

- **Single project**: `src/`, `tests/` at repository root
- **Web app**: `backend/src/`, `frontend/src/`
- **Mobile**: `api/src/`, `ios/src/` or `android/src/`
- Paths shown below assume single project - adjust based on plan.md structure

<!--
  ============================================================================
  IMPORTANT: The tasks below are SAMPLE TASKS for illustration purposes only.

  The /speckit.tasks command MUST replace these with actual tasks based on:
  - User stories from spec.md (with their priorities P1, P2, P3...)
  - Feature requirements from plan.md
  - Entities from data-model.md
  - Endpoints from contracts/

  Tasks MUST be organized by user story so each story can be:
  - Implemented independently
  - Tested independently
  - Delivered as an MVP increment

  Each task that produces code MUST include the TDD cycle:
  - Write a failing test for the behavior
  - Implement the minimum code to pass
  - Refactor while green

  DO NOT create separate "write tests" tasks. Tests are part of every task.
  DO NOT keep these sample tasks in the generated tasks.md file.
  ============================================================================
-->

## Phase 1: Setup (Shared Infrastructure)

**Purpose**: Project initialization and basic structure

- [ ] T001 Create project structure per implementation plan
- [ ] T002 Initialize [language] project with [framework] dependencies
- [ ] T003 [P] Configure linting and formatting tools

---

## Phase 2: Foundational (Blocking Prerequisites)

**Purpose**: Core infrastructure that MUST be complete before ANY user story can be implemented

**⚠️ CRITICAL**: No user story work can begin until this phase is complete

Examples of foundational tasks (adjust based on your project):

- [ ] T004 Setup database schema and migrations framework
- [ ] T005 [P] Implement authentication/authorization framework (TDD: test auth flow → implement → refactor)
- [ ] T006 [P] Setup API routing and middleware structure (TDD: test routing → implement → refactor)
- [ ] T007 Create base models/entities that all stories depend on (TDD: test model behavior → implement → refactor)
- [ ] T008 Configure error handling and logging infrastructure
- [ ] T009 Setup environment configuration management

**Checkpoint**: Foundation ready - user story implementation can now begin in parallel

---

## Phase 3: User Story 1 - [Title] (Priority: P1) 🎯 MVP

**Goal**: [Brief description of what this story delivers]

**Independent Test**: [How to verify this story works on its own]

Each task below includes TDD: write failing test → implement → refactor.

- [ ] T010 [P] [US1] Create [Entity1] model with validation in src/models/[entity1].py + tests/test_[entity1].py
- [ ] T011 [P] [US1] Create [Entity2] model with validation in src/models/[entity2].py + tests/test_[entity2].py
- [ ] T012 [US1] Implement [Service] core operation in src/services/[service].py + tests/test_[service].py (depends on T010, T011)
- [ ] T013 [US1] Implement [Service] error handling in src/services/[service].py + tests/test_[service].py
- [ ] T014 [US1] Implement [endpoint] happy path in src/[location]/[file].py + tests/test_[file].py
- [ ] T015 [US1] Implement [endpoint] validation and error responses in src/[location]/[file].py + tests/test_[file].py
- [ ] T016 [P] [US1] Integration test for [user journey] in tests/integration/test_[name].py

**Checkpoint**: At this point, User Story 1 should be fully functional and testable independently

---

## Phase 4: User Story 2 - [Title] (Priority: P2)

**Goal**: [Brief description of what this story delivers]

**Independent Test**: [How to verify this story works on its own]

Each task below includes TDD: write failing test → implement → refactor.

- [ ] T017 [P] [US2] Create [Entity] model with validation in src/models/[entity].py + tests/test_[entity].py
- [ ] T018 [US2] Implement [Service] core operation in src/services/[service].py + tests/test_[service].py
- [ ] T019 [US2] Implement [endpoint] in src/[location]/[file].py + tests/test_[file].py
- [ ] T020 [US2] Integrate with User Story 1 components (if needed)

**Checkpoint**: At this point, User Stories 1 AND 2 should both work independently

---

## Phase 5: User Story 3 - [Title] (Priority: P3)

**Goal**: [Brief description of what this story delivers]

**Independent Test**: [How to verify this story works on its own]

Each task below includes TDD: write failing test → implement → refactor.

- [ ] T021 [P] [US3] Create [Entity] model with validation in src/models/[entity].py + tests/test_[entity].py
- [ ] T022 [US3] Implement [Service] core operation in src/services/[service].py + tests/test_[service].py
- [ ] T023 [US3] Implement [endpoint] in src/[location]/[file].py + tests/test_[file].py

**Checkpoint**: All user stories should now be independently functional

---

[Add more user story phases as needed, following the same pattern]

---

## Phase N: Polish & Cross-Cutting Concerns

**Purpose**: Improvements that affect multiple user stories

- [ ] TXXX [P] Documentation updates in docs/
- [ ] TXXX Code cleanup and refactoring (all tests MUST remain green)
- [ ] TXXX Performance optimization across all stories
- [ ] TXXX Security hardening
- [ ] TXXX Run quickstart.md validation

---

## Dependencies & Execution Order

### Phase Dependencies

- **Setup (Phase 1)**: No dependencies - can start immediately
- **Foundational (Phase 2)**: Depends on Setup completion - BLOCKS all user stories
- **User Stories (Phase 3+)**: All depend on Foundational phase completion
  - User stories can then proceed in parallel (if staffed)
  - Or sequentially in priority order (P1 → P2 → P3)
- **Polish (Final Phase)**: Depends on all desired user stories being complete

### User Story Dependencies

- **User Story 1 (P1)**: Can start after Foundational (Phase 2) - No dependencies on other stories
- **User Story 2 (P2)**: Can start after Foundational (Phase 2) - May integrate with US1 but should be independently testable
- **User Story 3 (P3)**: Can start after Foundational (Phase 2) - May integrate with US1/US2 but should be independently testable

### Within Each User Story

- Each task follows Red-Green-Refactor: failing test → minimum implementation → refactor
- Models before services
- Services before endpoints
- Core implementation before integration
- Story complete before moving to next priority

### Parallel Opportunities

- All Setup tasks marked [P] can run in parallel
- All Foundational tasks marked [P] can run in parallel (within Phase 2)
- Once Foundational phase completes, all user stories can start in parallel
- Tasks within a story marked [P] can run in parallel
- Different user stories can be worked on in parallel by different teammates

---

## Execution Protocol (Agent Teams)

**⚠️ CRITICAL**: This section defines HOW tasks are executed. Follow this protocol exactly.

### Step 1: Create the Team

Use `TeamCreate` to create a team for the feature implementation:

```
TeamCreate({
  team_name: "[feature-name]-impl",
  description: "Implementing [FEATURE NAME] per tasks.md"
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
2. Complete Phase 2: Foundational (CRITICAL - blocks all stories)
3. Complete Phase 3: User Story 1
4. **STOP and VALIDATE**: Run full test suite, test User Story 1 independently
5. Deploy/demo if ready

### Incremental Delivery

1. Complete Setup + Foundational → Foundation ready
2. Add User Story 1 → All tests green → Deploy/Demo (MVP!)
3. Add User Story 2 → All tests green → Deploy/Demo
4. Add User Story 3 → All tests green → Deploy/Demo
5. Each story adds value without breaking previous stories

### Parallel Team Strategy

Using the agent team protocol above:

1. Team lead completes Setup + Foundational (sequential teammates)
2. Once Foundational is done, spawn parallel teammates:
   - Teammate A: First task of User Story 1
   - Teammate B: First task of User Story 2
   - Teammate C: First task of User Story 3
3. As each teammate finishes and shuts down, spawn new teammates for the next tasks in each story
4. Stories complete and integrate independently

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
- Avoid: vague tasks, same file conflicts, cross-story dependencies that break independence
