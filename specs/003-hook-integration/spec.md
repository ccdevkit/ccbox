# Feature Specification: Hook Integration

**Feature Branch**: `003-hook-integration`  
**Created**: 2026-04-05  
**Status**: Draft  
**Input**: User description: "Hook integration for Claude Code hooks via TCP bridge, settings merging, and internal handler registration API"

## User Scenarios & Testing *(mandatory)*

### User Story 1 - Internal Hook Handler Registration (Priority: P1)

A ccbox developer registers an internal hook handler (e.g., for `PreToolUse`) so that ccbox can intercept and respond to Claude Code hook events during a containerized session. The handler is invoked whenever the corresponding hook event fires inside the container, and the handler's response controls Claude Code's behavior (e.g., allowing, denying, or modifying a tool call). When registering, the developer may optionally specify a matcher pattern (regex) to narrow which invocations trigger the handler (e.g., only `Bash` tool calls for a `PreToolUse` handler).

**Why this priority**: This is the foundational capability that all other hook-based features depend on. Without an internal API for registering handlers, no hook-driven behavior can be built.

**Independent Test**: Can be tested by registering a handler for a known event, simulating a hook invocation over TCP, and verifying the handler receives the correct typed input and returns a valid typed response.

**Acceptance Scenarios**:

1. **Given** a ccbox session is being constructed, **When** a developer registers a hook handler for `PreToolUse` with order `before`, **Then** the handler is stored and retrievable by event name, and its ordering is recorded as `before`.
2. **Given** a hook handler is registered for `PostToolUse`, **When** the container proxy binary sends a hook request for that event over TCP, **Then** the handler is invoked with the correct strongly-typed input and its response is returned to the proxy.
3. **Given** no handler is registered for a given event, **When** the container proxy sends a hook request for that event, **Then** a default success response is returned (exit 0, no JSON output).

---

### User Story 2 - Settings Manager (Priority: P2)

ccbox provides a general-purpose Settings Manager that serves as ccbox's internal API for controlling Claude Code's settings. It discovers all relevant Claude Code settings files from the host filesystem (user-level and project-level, including `.local` variants), merges them by precedence, and exposes an API for other ccbox modules to modify the merged settings programmatically. After all modules have applied their modifications, the Settings Manager produces a final merged settings file, registers it as a session file, and adds `--settings <path>` and `--setting-sources ""` to the Claude CLI arguments so the merged file is the sole settings source.

The hook handler registry is one consumer of this API: it calls into the Settings Manager to inject hook entries pointing at the container proxy binary. Other modules (e.g., permissions, MCP config) can use the same API to inject their own settings without needing to know about the merge pipeline.

**Why this priority**: Hook handlers can only fire if the merged settings file includes the correct hook configuration pointing to the container proxy binary. Beyond hooks, this module provides the foundation for ccbox to control any Claude Code setting, making it a prerequisite for all settings-dependent features.

**Independent Test**: Can be tested by providing a set of mock settings files with known content, calling the modification API to inject arbitrary settings, and verifying the output contains all user entries plus injected entries in the correct precedence order. Hook-specific ordering (`before`/`after`) can be tested by injecting hook entries and checking their position relative to user-defined hooks.

**Acceptance Scenarios**:

1. **Given** user settings exist at `~/.claude/settings.json` and `.claude/settings.json`, **When** ccbox builds the session, **Then** a single merged settings file is produced respecting Claude Code's precedence rules (project overrides user).
2. **Given** a ccbox module calls the Settings Manager API to set a key (e.g., a permission rule), **When** the final settings file is produced, **Then** the module's value is present in the output alongside user settings.
3. **Given** the hook registry injects hook entries with order `before` for a given event and matcher, **When** the final settings are produced, **Then** ccbox hook entries appear before the user's entries within the same matcher group's hooks array.
4. **Given** the hook registry injects hook entries with order `after`, **When** the final settings are produced, **Then** ccbox hook entries appear after the user's entries within the same matcher group's hooks array.
5. **Given** no user settings files exist, **When** ccbox builds the session, **Then** the merged settings file contains only ccbox-managed entries and is still valid.
6. **Given** settings are finalized, **When** Claude Code is launched, **Then** `--settings <path>` and `--setting-sources ""` are included in the CLI arguments, and the settings file is registered as a session file.

---

### User Story 3 - Container Proxy Binary for Hook Dispatch (Priority: P3)

A lightweight binary inside the container is configured as the hook command for all registered events. When Claude Code fires a hook, this binary receives the hook input on stdin, forwards it over TCP to the ccbox host process, waits for a response, and exits with the appropriate exit code and stdout/stderr output.

**Why this priority**: This is the container-side transport layer. Without it, hook events cannot reach the host, but the host-side handler API and settings merging are independently valuable for design and testing.

**Independent Test**: Can be tested by running the proxy binary with mock stdin input, connecting it to a mock TCP server, and verifying it correctly forwards the request, receives the response, and exits with the expected code and output.

**Acceptance Scenarios**:

1. **Given** Claude Code fires a `PreToolUse` hook, **When** the proxy binary is invoked as the hook command, **Then** it reads the hook JSON from stdin, sends it to the host over TCP, and writes the host's response to stdout.
2. **Given** the host responds with exit code 2 (blocking), **When** the proxy binary receives this response, **Then** it exits with code 2 and writes the reason to stderr.
3. **Given** the host is unreachable (TCP connection fails), **When** the proxy binary attempts to connect, **Then** it exits with a non-blocking error code (not 0, not 2) so Claude Code continues without the hook.
4. **Given** the host responds with a JSON body, **When** the proxy binary receives it, **Then** the JSON is written to stdout verbatim and the binary exits with code 0.

---

### Edge Cases

- What happens when multiple handlers are registered for the same event? Multiple handlers per event are supported. Claude Code runs all matching hooks in parallel and aggregates results. For decision events (e.g., `PreToolUse`), precedence is `deny > defer > ask > allow`. Any handler returning `continue: false` halts Claude processing entirely, taking precedence over event-specific decisions.
- What happens when a user's settings file is malformed JSON? The file is skipped with a warning logged, and remaining files are still merged.
- What happens when the hook response exceeds Claude Code's 10,000 character output cap? The proxy binary does not truncate; Claude Code handles this limit internally.
- What happens when the TCP connection times out during a hook call? The proxy binary exits with a non-blocking error code, allowing Claude Code to continue.
- What happens when `--settings` is already provided by the user? ccbox merges the user-provided settings file contents into the combined settings, respecting precedence.

## Requirements *(mandatory)*

### Functional Requirements

- **FR-001**: System MUST provide an internal API for registering hook handlers by event name, with each handler accepting strongly-typed input and returning strongly-typed output matching Claude Code's hook schemas. Registration MUST support an optional matcher pattern (regex string) to narrow which invocations trigger the handler, matching Claude Code's three-level hook config structure (event → matcher group → hooks array).
- **FR-002**: System MUST accept an `order` parameter (`before` or `after`) when registering a hook handler, controlling placement of the ccbox hook entry relative to user-defined hooks in the merged settings. For a given event and matcher combination, `before` places ccbox entries at the start of the matcher group's hooks array, `after` places them at the end.
- **FR-003**: System MUST provide a general-purpose Settings Manager that discovers and reads Claude Code settings files from all standard locations (`~/.claude/settings.json`, `~/.claude/settings.local.json`, `.claude/settings.json`, `.claude/settings.local.json`), merging them by Claude Code's precedence rules.
- **FR-004**: The Settings Manager MUST expose an API for other ccbox modules to modify the merged settings programmatically (e.g., setting arbitrary keys, injecting hook entries). After all modules have applied their modifications, the Settings Manager produces a final merged settings file written to the session's temporary directory and registered as a session file.
- **FR-005**: The Settings Manager MUST add `--settings <merged-file>` and `--setting-sources ""` to the Claude CLI arguments so the merged file is the sole settings source. The `--settings` flag loads additional settings from a file; `--setting-sources` controls which auto-discovery scopes (`user`, `project`, `local`) are active, and passing an empty value disables all automatic discovery.
- **FR-006**: System MUST include a container-side proxy binary that receives hook input on stdin, forwards it to the host over TCP, and returns the response as stdout/exit code.
- **FR-007**: System MUST extend the existing TCP bridge protocol with a new `hook` message type for hook requests and responses, sharing the same port and connection infrastructure used by `exec` and `log` messages.
- **FR-008**: System MUST define strongly-typed request and response structures for all 26 Claude Code hook events from the initial delivery, based on Claude Code's documented schemas.
- **FR-009**: System MUST support multiple handlers per event name. Handlers with the same event and matcher are grouped into the same matcher group's hooks array. Handlers with different matchers produce separate matcher group entries. Claude Code executes all matching hooks in parallel and aggregates their results (decision precedence: `deny > defer > ask > allow`; any `continue: false` halts processing).
- **FR-010**: System MUST gracefully handle TCP connection failures in the proxy binary by exiting with a non-blocking error code. The proxy binary MUST enforce a 10 second timeout when waiting for the host's TCP response.
- **FR-011**: The Settings Manager MUST preserve all user settings in the merged output. Module-injected settings are additive; they MUST NOT silently overwrite user-defined settings unless the module explicitly targets the same key.

### Key Entities

- **HookHandler**: A registered callback for a specific hook event, with typed input/output, an optional matcher pattern (regex), and an ordering directive (`before`/`after`). Multiple handlers may be registered for the same event and matcher combination.
- **HookRequest**: A message from the container proxy to the host, containing the event name and the hook input JSON.
- **HookResponse**: A message from the host to the container proxy, containing exit code, stdout content, and stderr content.
- **SettingsManager**: The general-purpose module that discovers, merges, and exposes an API for modifying Claude Code settings. Produces the final merged settings file and registers it with the session and CLI args. The hook registry and other modules are consumers of this API.
- **MergedSettings**: The combined settings file produced by the Settings Manager from user settings and all ccbox module contributions (hooks, permissions, etc.).

## Success Criteria *(mandatory)*

### Measurable Outcomes

- **SC-001**: Internal hook handlers can be registered and invoked for any of the 26 Claude Code hook events.
- **SC-002**: Hook handler responses reach the container proxy and produce the correct exit code and output within 1 second for typical payloads.
- **SC-003**: User-defined settings and hooks are fully preserved in the merged settings output; no user configuration is lost during merging.
- **SC-004**: When the host is unreachable, the container proxy fails gracefully and Claude Code continues operating without interruption.
- **SC-005**: All hook request and response types are covered by compile-time type safety (no untyped maps in the handler API).
- **SC-006**: The Settings Manager API can be used by any ccbox module to inject arbitrary settings; it is not coupled to hook-specific logic.

## Clarifications

### Session 2026-04-05

- Q: Should all 28 hook events have typed structs from day one, or start with a subset? → A: All 28 events get fully typed structs from day one.
- Q: Should hook requests share the existing TCP bridge or use a separate dedicated connection? → A: Shared — extend the existing TCP bridge with a new `hook` message type on the same port.
- Q: What should the proxy binary's TCP response timeout be? → A: 10 seconds.
- Correction: Multiple handlers per event are supported (not single-handler-replaces). Claude Code runs all matching hooks in parallel, aggregates results via decision precedence (`deny > defer > ask > allow`), and `continue: false` from any handler halts processing entirely.
- Correction: Hook config uses three-level nesting (event → matcher groups with regex → hooks array), not a flat array. Registration API must support optional matcher patterns. Ordering (`before`/`after`) applies within a matcher group's hooks array. Updated FR-001, FR-002, FR-009, and acceptance scenarios accordingly.
- Correction: `--setting-sources` takes scope names (`user`, `project`, `local`) as a comma-separated list, not arbitrary paths. Empty value disables all auto-discovery. `--settings` loads additional settings from a file/JSON string. Both flags confirmed to exist.
- Refinement: Settings merger broadened to a general-purpose Settings Manager — ccbox's internal API for controlling any Claude Code setting. Discovers, merges, exposes a modification API for other modules (hooks, permissions, etc.), produces the final settings file, registers it as a session file, and adds CLI args. The hook registry is one consumer, not the sole owner.

## Assumptions

- Claude Code's hook system and settings file format remain stable as documented at the time of writing.
- The existing TCP bridge between the container and host (used for exec and log requests) is available and can be extended with new message types.
- The container proxy binary will be built and included in the ccbox container image alongside the existing `ccptproxy` binary.
- Claude Code's `--settings` flag loads additional settings from a file or JSON string. The `--setting-sources` flag accepts a comma-separated list of scopes (`user`, `project`, `local`) to control auto-discovery; passing an empty value disables all automatic settings file discovery.
- Settings precedence follows Claude Code's documented hierarchy: managed policy > project local > project > user local > user.
- The proxy binary can read hook input from stdin and environment variables as Claude Code provides them.
