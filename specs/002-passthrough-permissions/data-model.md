# Data Model: Passthrough Command Permissions

**Feature Branch**: `002-passthrough-permissions`
**Date**: 2026-04-04

## Entities

### PermissionsConfig

Top-level configuration parsed from `.ccbox/permissions.{json,yml,yaml}`.

| Field | Type | Description |
|-------|------|-------------|
| Passthrough | map[string]*CommandPermission | Map of command name → permission config. A nil value means "allow all". |

**Validation Rules**:
- `Passthrough` map keys must be non-empty strings (command names)
- All patterns in all rules must be valid (parseable at load time)
- All regex patterns must compile without error
- `effect` must be "allow" or "deny"

**Discovery**: Loaded via `common/settings.Load(".ccbox/permissions", &config, nil)`. Hierarchical merge applies (parent + child directories). Child command entries override parent entries for the same command name.

### CommandPermission

Defines the permission rules for a single passthrough command.

| Field | Type | Description |
|-------|------|-------------|
| Rules | []Rule | Ordered array of pattern/effect rules. nil/empty = allow all. |

**State Transitions**: None — immutable after load.

### Rule

A single entry in the cascading permission array.

| Field | Type | Description |
|-------|------|-------------|
| Pattern | PatternOrArray | String or array of strings (YAML/JSON input) |
| Effect | string | "allow" or "deny" |
| Reason | string | Optional. Included in denial message when this deny rule triggers. Silently ignored on allow rules. |

**Note**: Array patterns are expanded during parsing. `{pattern: ["a", "b"], effect: "deny"}` becomes two internal rules: `{pattern: "a", effect: "deny"}` and `{pattern: "b", effect: "deny"}`.

### PatternOrArray

Custom unmarshaling type that accepts either a string or []string from YAML/JSON.

| Variant | Type | Description |
|---------|------|-------------|
| Single | string | A single pattern string |
| Multiple | []string | Array of pattern strings (shorthand for multiple rules with same effect) |

### ArgPattern (compiled)

A parsed, validated pattern ready for matching. Created at startup from pattern strings.

| Field | Type | Description |
|-------|------|-------------|
| Raw | string | Original pattern string (for error messages) |
| Elements | []PatternElement | Ordered list of parsed pattern elements |
| ExactMatch | bool | True if pattern ends with `$` (prefix matching disabled). `$` is detected after stripping leading/trailing whitespace and collapsing multi-whitespace to single separators — so `push --force$`, `push --force $`, and `push --force           $` are all equivalent. |

### PatternElement

A single token in a parsed pattern.

| Field | Type | Description |
|-------|------|-------------|
| Type | ElementType | literal, wildcard, doubleWildcard, singleChar, regex, regexMulti, quoted |
| Value | string | The literal/regex/quoted value |
| Optional | bool | `?` modifier — element may be absent |
| NonPositional | bool | `~` modifier — matches anywhere in remaining args |
| Group | []PatternElement | For `()` grouping — contains sub-elements |

**ElementType enum**: `literal`, `wildcard` (`*`), `doubleWildcard` (`**`), `singleChar` (`.`), `regex` (`/re/`), `regexMulti` (`/re/**`), `quoted` (`"..."` or `'...'`), `group` (`(...)`)

### CompiledRule

Internal representation after pattern parsing and array expansion.

| Field | Type | Description |
|-------|------|-------------|
| Pattern | *ArgPattern | Compiled pattern matcher |
| Effect | Effect | allow or deny |
| Reason | string | Optional denial reason |

### MatchResult

The outcome of evaluating a command against its permission rules.

| Field | Type | Description |
|-------|------|-------------|
| Allowed | bool | Whether the command is permitted |
| Reason | string | Why (rule description or "no matching rule") |
| MatchedRule | *CompiledRule | The rule that determined the outcome (nil if default deny) |
| Command | string | The full command that was evaluated |

## Relationships

```
PermissionsConfig
  └── passthrough: map[string]*CommandPermission
        └── Rules: []Rule
              ├── Pattern: PatternOrArray → expanded to []CompiledRule
              │     └── ArgPattern
              │           └── []PatternElement
              └── Effect: allow | deny
```

## Merge Behavior

1. **File hierarchy**: `common/settings.Load` merges parent→child. Same command name in child replaces parent's entry (map key override). Different command names accumulate.

2. **CLI + file merge**: For commands added via `-pt:cmd`:
   - If command exists in file: prepend implicit `{pattern: "**", effect: "allow"}` before file rules
   - If command not in file: create entry with single `{pattern: "**", effect: "allow"}` rule
   - If command in file with no rules (nil value): remains unrestricted (allow all)

3. **Default behavior**: 
   - Command listed with no rules → allow all (implicit allow)
   - Command with rules but no match → deny (fail-closed)
   - Command not listed at all → not a passthrough command (handled by existing CommandMatcher)
