# Feature Specification: Passthrough Command Permissions

**Feature Branch**: `002-passthrough-permissions`  
**Created**: 2026-04-04  
**Status**: Draft  
**Input**: User description: "permissions system for passthrough commands. These should be configurable in .ccbox/permissions.{json,yml,yaml} (use common/settings package for this). We specify which commands are passed through and for each of those commands, which argument patterns are allowed and denied."

## User Scenarios & Testing *(mandatory)*

### User Story 1 - Define Allowed Passthrough Commands in Permissions File (Priority: P1)

A user creates a `.ccbox/permissions.{json,yml,yaml}` file to declare which commands are allowed to pass through to the host, replacing or supplementing the `-pt`/`--passthrough` CLI flags. When ccbox starts, it reads this file and merges the allowed commands with any CLI-specified passthrough commands.

**Why this priority**: This is the foundational capability — without declaring which commands are permitted, no other permission rules can apply. It also provides a persistent, version-controllable alternative to CLI flags.

**Independent Test**: Can be fully tested by creating a permissions file with a list of allowed commands and verifying that ccbox enables passthrough for exactly those commands (plus any from CLI flags).

**Acceptance Scenarios**:

1. **Given** a `.ccbox/permissions.yaml` file listing `git` and `npm` as allowed commands, **When** ccbox starts without any `-pt` flags, **Then** `git` and `npm` are available as passthrough commands in the container.
2. **Given** a permissions file listing `git` with deny rules and CLI flag `-pt:docker`, **When** ccbox starts, **Then** both `git` and `docker` are available as passthrough commands. `git` has its file-defined rules; `docker` has an implicit `allow **` rule from the CLI flag.
3. **Given** no permissions file exists, **When** ccbox starts with `-pt:git`, **Then** passthrough works as it does today — `git` gets an implicit `allow **` rule.
4. **Given** CLI flag `-pt:git` and a permissions file defining `git` with deny rules, **When** ccbox starts, **Then** the CLI contributes an implicit `{pattern: "**", effect: "allow"}` as the first rule in the cascade, followed by the file's rules. The result is "allow all except what the file denies."
5. **Given** a malformed permissions file, **When** ccbox starts, **Then** the system reports a clear error and refuses to start (fail-closed).

---

### User Story 2 - Cascading Permission Rules with Pattern Matching (Priority: P1)

The user (human operator) configures an ordered array of pattern/effect rules for each command to control what Claude (the LLM in the container) is allowed to execute on the host. Rules are evaluated top-to-bottom; the last rule whose pattern matches determines the outcome (allow or deny). This cascading model lets users express complex logic like "deny all git commands except git pull" naturally.

**Why this priority**: Without argument-level control, passthrough is all-or-nothing per command. Cascading rules are the core value proposition of the permissions system — they let users grant Claude least-privilege access with expressive, composable logic.

**Independent Test**: Can be tested by configuring cascading rules for `git` (e.g., deny `**`, then allow `pull`), then verifying that `git pull` succeeds, `git push` is denied, and the deny message is clear.

**Acceptance Scenarios**:

1. **Given** rules `[{pattern: "**", effect: "deny"}, {pattern: "pull", effect: "allow"}]` for `git`, **When** Claude runs `git pull`, **Then** the command executes because the last matching rule is "allow pull".
2. **Given** the same rules, **When** Claude runs `git push origin main`, **Then** the command is denied because only the "deny **" rule matches.
3. **Given** rules `[{pattern: "**", effect: "allow"}, {pattern: "push ~--force", effect: "deny"}]` for `git`, **When** Claude runs `git push --force origin main`, **Then** the command is denied because the last matching rule is the deny for `push ~--force`.
4. **Given** the same rules, **When** Claude runs `git push origin main`, **Then** the command executes because the deny rule doesn't match (no `--force`), so the "allow **" is the last match.
5. **Given** a permissions file allowing `git` with no rules defined, **When** Claude runs any `git` subcommand, **Then** all subcommands are allowed (no restrictions on arguments — backward-compatible).
6. **Given** rules for `git` with pattern `status$`, **When** Claude runs `git status --short`, **Then** the pattern does not match because `$` disables prefix matching.
7. **Given** rules with pattern `clone /https?:\/\//`, **When** Claude runs `git clone https://github.com/repo`, **Then** the pattern matches because the second arg matches the regex.

---

### User Story 3 - Default Effect When No Rule Matches (Priority: P2)

When a command has rules defined but no rule matches the invocation, the system needs a deterministic default. If any rules are defined for a command, the default when no rule matches is "deny" (fail-closed). This ensures that adding rules doesn't accidentally leave gaps that Claude could exploit.

**Why this priority**: The default behavior determines the security posture. Fail-closed is essential to prevent Claude from executing unintended commands when users define partial rule sets.

**Independent Test**: Can be tested by defining a single allow rule and verifying that commands not matching it are denied.

**Acceptance Scenarios**:

1. **Given** rules `[{pattern: "status", effect: "allow"}]` for `git`, **When** Claude runs `git push`, **Then** the command is denied because no rule matched and the default is deny.
2. **Given** a command with no rules at all (just listed as allowed), **When** Claude runs any subcommand, **Then** all subcommands are allowed (backward-compatible: no rules = no restrictions).

---

### User Story 4 - Hierarchical Permissions File Discovery (Priority: P2)

Permissions files are discovered using the same hierarchical walk as settings files (via the common/settings package). A project-level `.ccbox/permissions.yaml` can override or extend a parent directory or home-level permissions file.

**Why this priority**: Consistency with the existing settings discovery mechanism and support for multi-project configurations.

**Independent Test**: Can be tested by placing permissions files at two directory levels and verifying correct merge behavior.

**Acceptance Scenarios**:

1. **Given** a permissions file at `~/` allowing `git` and a project-level permissions file allowing `npm`, **When** ccbox starts in the project, **Then** both `git` and `npm` are allowed (arrays merge).
2. **Given** a parent permissions file and a child permissions file with overlapping commands, **When** ccbox starts, **Then** the child's rules take precedence for overlapping commands.

---

### User Story 5 - Clear Denial Messages (Priority: P3)

When a passthrough command is denied by the permissions system, Claude (the LLM running in the container) receives a clear message explaining why the command was blocked and which rule caused the denial. This helps Claude understand the boundaries and adjust its approach rather than retrying or working around the restriction.

**Why this priority**: Without clear messages, Claude may retry denied commands or take unproductive paths. This is important but not blocking for core functionality.

**Independent Test**: Can be tested by running a denied command and verifying the error message includes the command, the matching deny rule, and a suggestion.

**Acceptance Scenarios**:

1. **Given** a command denied by a deny rule (last matching rule had effect "deny"), **When** the denial message is shown, **Then** it includes the command that was attempted, the rule that blocked it, the `reason` if one was configured, and how to adjust permissions if needed.
2. **Given** a command denied because no rule matched (fail-closed default), **When** the denial message is shown, **Then** it explains that no rule matched and lists the available patterns for that command.

---

### Edge Cases

- What happens when a permissions file has an allow rule with an invalid regex pattern? The system fails at startup with a clear error identifying the invalid pattern.
- What happens when a command has only deny rules and no allow rules? All invocations are denied — deny is the default when no allow rule matches, and adding only deny rules means no allow rule can ever be the last match.
- What happens when the permissions file is empty? No additional commands are added from the file; CLI flags still contribute their commands with implicit `allow **` rules.
- What happens when a pattern uses `**` across args but the command has fewer args than expected? The pattern does not match (no error, just no match).
- What happens when arg patterns contain shell metacharacters? Patterns are matched against pre-parsed arguments, not raw shell strings. The proxy already receives split arguments via `"$@"`.

## Permissions File Schema

The permissions file is namespaced under a `passthrough` key to allow future permission types. Command names are keys under `passthrough`. A command with a null/empty value allows all arguments (no restrictions). A command with a `rules` array applies cascading evaluation.

### YAML Example

```yaml
passthrough:
  # No restrictions — all arguments allowed
  git:

  # Cascading rules — last match wins
  npm:
    rules:
      - pattern: "**"
        effect: deny
      - pattern: "install"
        effect: allow
      - pattern: "ci"
        effect: allow
      - pattern: "run build"
        effect: allow

  # Deny all except safe read operations
  docker:
    rules:
      - pattern: "**"
        effect: deny
      - pattern: "ps"
        effect: allow
      - pattern: "images"
        effect: allow
      - pattern: "logs *"
        effect: allow

  # Allow everything, block dangerous flags
  kubectl:
    rules:
      - pattern: "**"
        effect: allow
      - pattern:
          - "delete ~--all"
          - "delete ~-A"
        effect: deny
        reason: "Bulk delete is too destructive — delete individual resources instead"

  # pattern can be a string or array of strings
  # array form applies the same effect to multiple patterns
  make:
    rules:
      - pattern: ["build", "test", "lint"]
        effect: allow
```

### JSON Equivalent

```json
{
  "passthrough": {
    "git": null,
    "npm": {
      "rules": [
        { "pattern": "**", "effect": "deny" },
        { "pattern": "install", "effect": "allow" }
      ]
    }
  }
}
```

### Schema Rules

- Top-level key MUST be `passthrough`.
- Each key under `passthrough` is a command name (string).
- A command value of `null`, empty, or omitted means "allow all arguments" (equivalent to CLI `-pt:cmd`).
- A command value with a `rules` array contains ordered `{pattern, effect}` objects.
- `pattern` is a string or array of strings. An array is shorthand for multiple rules with the same effect — each pattern in the array is evaluated independently. See pattern-syntax-notes.md for syntax.
- `effect` MUST be either `"allow"` or `"deny"`.
- `reason` is an optional string. When present on a deny rule that triggers, the reason is included in the denial message shown to the LLM. Silently ignored on allow rules (no warning or error produced).
- Duplicate command keys in a single file follow YAML/JSON specification behavior (last key wins). This is parser behavior, not enforced by ccbox.
- Rules are evaluated top-to-bottom; the last matching rule's effect wins.
- If rules exist but none match, the command is denied (fail-closed).
- Commands added via CLI flags (`-pt`/`--passthrough`) contribute an implicit `{pattern: "**", effect: "allow"}` as the first rule in the command's cascade. If the permissions file also defines rules for the same command, those rules are appended after the implicit allow. This means `-pt:git` combined with deny rules in the file produces "allow all except what the file denies."

## Requirements *(mandatory)*

### Functional Requirements

- **FR-001**: System MUST support permissions configuration in `.ccbox/permissions.{json,yml,yaml}` files using the common/settings package for discovery and merging.
- **FR-002**: System MUST merge passthrough commands from permissions files with those from CLI flags (`-pt`/`--passthrough`), with deduplication. CLI flags contribute an implicit `{pattern: "**", effect: "allow"}` as the first rule in the command's cascade, before any file-defined rules.
- **FR-003**: System MUST support cascading rules — an ordered array of pattern/effect pairs per command, where each rule specifies a pattern and an effect (allow or deny).
- **FR-004**: System MUST evaluate rules top-to-bottom; the last rule whose pattern matches the invocation determines whether the command is allowed or denied (last-match-wins).
- **FR-005**: System MUST support the pattern syntax defined in the project's pattern-syntax-notes.md, including: wildcards (`*`, `**`, `.`), regex (`/pattern/`, `/pattern/**`), optional (`?`), non-positional (`~`), exact string (`" "`, `' '`), grouping (`( )`), escape (`\`), and exact-match terminator (`$`). The `*` and `.` wildcards operate as globs anywhere within a token (e.g., `pre*fix` matches `prefix` or `pre-blah-fix`; `a.b` matches `axb`). Quoting (`""`, `''`) or escaping (`\*`, `\.`) disables wildcard interpretation. Grouping `()` is valid with or without `?` — standalone grouping allows modifiers like `~` to apply to multi-element sequences (e.g., `~(-n 0)` means `-n 0` can appear anywhere in remaining args).
- **FR-006**: System MUST use prefix matching by default (a pattern `git status` matches `git status --short`) and allow disabling it with `$` at end. The `$` is a pattern-level modifier: leading/trailing whitespace is stripped, multiple whitespace between tokens collapses to a single separator, and `$` at the end (after stripping) disables prefix matching regardless of preceding whitespace. `push --force$`, `push --force $`, and `push --force           $` are all equivalent. A `$` appearing anywhere other than the final position (after whitespace stripping) is a validation error and MUST cause a startup failure.
- **FR-007**: System MUST fail closed — if the permissions file is present but malformed, or contains invalid regex, ccbox must refuse to start with a clear error.
- **FR-008**: System MUST provide clear denial messages that identify which rule blocked a command.
- **FR-009**: When a command is listed in the permissions file with no rules (null/empty value), or added only via CLI flag, the system MUST allow all arguments for that command (implicit `allow **`).
- **FR-010**: System MUST validate all patterns at startup (not at match time) to catch configuration errors early.
- **FR-011**: Permissions evaluation MUST happen on the host side (in the bridge server) when exec requests are received from the container. The container is the LLM's domain and cannot be trusted as an enforcement point.
- **FR-012**: When rules are defined for a command but no rule matches an invocation, the system MUST deny the command (fail-closed default).

### Key Entities

- **PermissionsConfig**: Top-level configuration containing a list of command permission entries. Discovered and merged hierarchically.
- **CommandPermission**: Defines a single passthrough command and its ordered array of pattern/effect rules.
- **ArgPattern**: A parsed pattern expression using the defined syntax (wildcards, regex, groups, etc.) that can be matched against actual command arguments.
- **Rule**: A single entry in the cascading array — a pattern paired with an effect (allow or deny).
- **MatchResult**: The outcome of evaluating a command against the cascading rules — allowed, denied (by rule or by default), or unrestricted — including which rule produced the result.

## Success Criteria *(mandatory)*

### Measurable Outcomes

- **SC-001**: Users can define passthrough permissions in a configuration file and have them enforced without using CLI flags.
- **SC-002**: Permissions are enforced on the host (trusted side), so Claude cannot bypass restrictions even if it tampers with container-side binaries or environment.
- **SC-003**: 100% of invalid permission configurations are caught at startup with actionable error messages.
- **SC-004**: Users can express common permission patterns (allow specific subcommands, deny dangerous flags) in 1-3 lines of configuration per command.
- **SC-005**: Existing passthrough behavior (CLI flags with no argument restrictions) continues to work unchanged when no permissions file is present.

## Assumptions

- The common/settings package's hierarchical discovery and merge behavior is suitable for permissions files (same walk-up-directory-tree strategy).
- The host-side bridge server is the correct enforcement point since the container is the LLM's domain and cannot be trusted. The bridge already receives all exec requests before executing them.
- Arguments arrive at the proxy already split by the shell (via `"$@"` in hijacker scripts), so pattern matching operates on individual arguments, not raw shell strings.
- The pattern syntax described in the user's notes is the authoritative design — this spec captures it faithfully and implementation should follow it.
- Performance of pattern matching is not a concern at the expected scale (human-speed command invocations, not automated batch operations).
