# Research: Passthrough Command Permissions

**Feature Branch**: `002-passthrough-permissions`
**Date**: 2026-04-04

## Research Tasks

### R1: Where should permissions be enforced — host or container?

**Decision**: Host-side enforcement only (in the bridge server's ExecHandler).

**Rationale**: FR-011 explicitly states "The container is the LLM's domain and cannot be trusted as an enforcement point." The bridge server on the host already receives all exec requests before executing them. The current flow is:

```
Container (ccptproxy --exec) → TCP → Host (bridge.Server) → cmdpassthrough.HandleExec → sh -c
```

The enforcement point is a wrapper around `HandleExec` that checks permissions before delegating to the real executor. The container-side `CommandMatcher` stays as-is for command-name matching (which commands have shims), but argument-level permission checks happen only on the host.

**Alternatives considered**:
- Container-side enforcement (rejected: LLM can tamper with container binaries)
- Dual enforcement (rejected: adds complexity with no security benefit since host enforcement is sufficient)

### R2: How should permissions config be loaded — separate file or extend settings?

**Decision**: Separate file at `.ccbox/permissions.{json,yml,yaml}`, loaded via `common/settings.Load()`.

**Rationale**: The spec explicitly names `.ccbox/permissions.{json,yml,yaml}` as the file path. Using a separate file keeps concerns separated — settings control ccbox behavior, permissions control security policy. The `common/settings` package already handles hierarchical discovery and merge for any relative path.

**Alternatives considered**:
- Embedding permissions in `.ccbox/settings.yaml` (rejected: spec mandates separate file, and mixing operational settings with security policy is bad practice)
- Custom file discovery (rejected: `common/settings` already does exactly what's needed)

### R3: How does the common/settings merge strategy interact with permissions?

**Decision**: Arrays append (low→high precedence). For permissions, this means a parent-level permissions file and a child-level one are merged: command lists accumulate. However, when the same command appears at multiple levels, the child's `rules` array replaces the parent's (map merge behavior — child key overwrites parent key).

**Rationale**: `common/settings.mergeMaps` does recursive map merge where:
- Maps: recursively merged (child keys override parent keys)
- Arrays: appended
- Primitives: replaced

Since the permissions schema uses `passthrough` as a top-level map with command names as keys, a child-level `git` entry will replace a parent-level `git` entry. This is the correct behavior per US-4 AS-2: "the child's rules take precedence for overlapping commands."

For non-overlapping commands, both levels contribute their commands (map merge adds new keys). This satisfies US-4 AS-1.

**Alternatives considered**: None — the existing merge behavior matches the spec requirements exactly.

### R4: How should the pattern syntax be implemented?

**Decision**: A two-phase approach: (1) parse pattern strings into a structured AST at config load time (FR-010: validate at startup), (2) evaluate the AST against actual arguments at exec time.

**Rationale**: The pattern syntax from `pattern-syntax-notes.md` includes:
- Positional matching: each space-separated token matches one argument
- `*` (single-arg wildcard), `**` (cross-arg wildcard), `.` (single char)
- `/regex/` (single arg), `/regex/**` (multi-arg)
- `~` (non-positional: matches anywhere in remaining args)
- `?` (optional: preceding element may be absent)
- `"quoted"` (exact literal), `$` (disable prefix matching)
- `()` (grouping for `?`)
- `\` (escape)

The parser tokenizes the pattern string into elements, each with a type (literal, wildcard, regex, etc.) and modifiers (optional, non-positional). The matcher walks arguments left-to-right, consuming positional elements, then checking non-positional elements against remaining args.

Default prefix matching means `status` matches `status --short` — the pattern doesn't need to account for trailing args. `$` at end disables this.

**Alternatives considered**:
- Pure regex patterns (rejected: the spec defines a custom syntax that's more readable for command patterns)
- Glob-only patterns (rejected: doesn't cover regex, non-positional, or grouping features)

### R5: How should CLI flags merge with file-defined rules?

**Decision**: CLI `-pt:cmd` contributes `{pattern: "**", effect: "allow"}` as the **first** rule in the cascade for that command. File-defined rules are appended after.

**Rationale**: Per FR-002 and the Schema Rules in the spec: "Commands added via CLI flags contribute an implicit `{pattern: "**", effect: "allow"}` as the first rule in the command's cascade." Since evaluation is last-match-wins, this means:
- CLI-only command: single `allow **` rule → everything allowed
- CLI + file deny rules: `allow **` is first, file deny rules follow → "allow all except what the file denies"
- File-only command with rules: file rules only → evaluated as written
- File-only command with no rules: implicit allow all (FR-009)

**Alternatives considered**: None — the spec is explicit about this behavior.

### R6: What package structure should the permissions system use?

**Decision**: New package `internal/permissions/` containing:
- `config.go` — `PermissionsConfig`, `CommandPermission`, `Rule` types + `Load()` function
- `pattern.go` — `ArgPattern` type, pattern parser, and tokenizer
- `matcher.go` — `Matcher` type that evaluates rules against arguments, `MatchResult`
- `merge.go` — CLI + file merge logic

**Rationale**: Follows constitution Principle IV (Single Responsibility). The permissions system is a distinct domain from command passthrough execution. The `cmdpassthrough` package handles execution; `permissions` handles policy.

The enforcement hook is in `cmdpassthrough` or the bridge wiring — a thin wrapper that calls `permissions.Matcher.Check()` before delegating to `HandleExec`.

**Alternatives considered**:
- Adding to `cmdpassthrough` package (rejected: violates single responsibility, permissions is a separate concern)
- Adding to `settings` package (rejected: settings loads config, permissions enforces policy)

### R7: How should the ExecRequest be parsed for permission checking?

**Decision**: The `ExecRequest.Command` field contains a full shell command string (e.g., `"git push --force origin main"`). For permission checking, the first word is the command name (used to look up `CommandPermission`), and remaining words are the arguments to match against patterns.

**Rationale**: The current `HandleExec` passes `req.Command` to `sh -c`, so it's a shell string. The container-side proxy already receives pre-split arguments via `"$@"` and sends them as a single command string. For permission matching, we split on the first space to get the command name, then use `strings.Fields()` for the remaining arguments.

Note: The spec says "Arguments arrive at the proxy already split by the shell (via `"$@"` in hijacker scripts), so pattern matching operates on individual arguments." The proxy reassembles them into a single string for the TCP protocol. On the host side, we re-split — this is safe because the arguments were already shell-split on the container side.

**Alternatives considered**:
- Changing the bridge protocol to send split args (rejected: unnecessary protocol change, re-splitting is safe and simple)
