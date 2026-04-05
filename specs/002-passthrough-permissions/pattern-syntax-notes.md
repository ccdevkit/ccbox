# Pattern Syntax Notes (from user's handwritten notes)

## Proposed Permissions Pattern Syntax

Example: `` `git /regex/ * prefix* *suffix ** "exact match" -.` ``

### Legend

| Symbol | Meaning | Scope |
|--------|---------|-------|
| `/pattern/` | Regex | One arg |
| `/pattern/**` | Regex | Multiple args |
| `*` | Multi chars (glob) | Anywhere within arg: `*`, `--*`, `pre*fix`, `*suffix` |
| `**` | Multi chars | Across args |
| `.` | Single char (glob) | Anywhere within arg: `.`, `v.`, `a.b`, `.v` |
| `word` | Exact match | One arg |
| `\` | Escape char | Next char — disables special meaning: `\*`, `\.`, `\~`, `\?`, `\\` |
| `?` | Optional | Preceding element or group |
| `~` | Non-positional | Match anywhere in remaining args (unordered) |
| `" "` | Exact string | Literal match, disables glob/wildcard interpretation |
| `' '` | Exact string | Same as `" "` — single-quote variant |
| `( )` | Group args | Group elements for modifiers: `(origin main)?`, `~(-n 0)` |
| `$` (at end) | Exact match | Disable prefix matching |

### Semantics

- Args separated by spaces
- Each arg matched positionally (except `~` args)
- `/pattern/**` - special flag to apply the pattern to multiple args
- Should automatically use prefix matching otherwise it could be bypassed by adding something on the end
  - Option to disable: `$` at end

### Wildcard & Dot Glob Behavior

- `*` and `.` operate as globs **anywhere within a token**, not just as standalone tokens
- `*` matches zero or more characters: `--*` matches `--verbose`, `pre*fix` matches `prefix` or `pre-blah-fix`, `*suffix` matches `my-suffix`
- `.` matches exactly one character: `v.` matches `v1`, `a.b` matches `axb`, `.v` matches `xv`
- Quoting (`""`, `''`) disables wildcard interpretation: `"v."` matches the literal string `v.`
- Escaping disables wildcard interpretation: `\*` matches a literal `*`, `\.` matches a literal `.`

### Grouping `()`

- `()` groups multiple elements so that modifiers apply to the entire group
- Valid with `?`: `(origin main)?` — the group is optional
- Valid with `~`: `~(-n 0)` — the two-element sequence `-n 0` can appear anywhere in remaining args
- Valid standalone: `(origin main)` — equivalent to `origin main` (no-op, but harmless)

### Whitespace & `$` Rules

- Leading and trailing whitespace is stripped from the pattern string before parsing
- Multiple consecutive whitespace characters between tokens are treated as a single separator
- `$` at the end of the pattern (after whitespace stripping) is a pattern-level modifier, not an argument token
- Whitespace before `$` is irrelevant — all of these are equivalent:
  - `push --force$`
  - `push --force $`
  - `push --force           $`
- All parse to pattern elements `[push, --force]` with exact-match enabled
- `$` is ONLY valid at the final position (after whitespace stripping). A `$` appearing mid-pattern (e.g., `push $ --force`) is a validation error at startup.
