<!--
Sync Impact Report
- Version change: 1.0.0 → 1.1.0
- Added principles:
  7. Red-Green-Refactor TDD
- Changed:
  - Project name corrected from "ccignore" to "ccbox"
  - Development Workflow updated with TDD cycle requirements
- Templates requiring updates:
  - .specify/templates/plan-template.md — verify TDD phase guidance present
  - .specify/templates/spec-template.md — verify test-first language present
  - .specify/templates/tasks-template.md — verify red-green-refactor phasing compatible
- Follow-up TODOs: Review templates for TDD alignment
-->

# ccbox Constitution

## Core Principles

### I. Simplicity Over Cleverness

All code MUST be straightforward and readable. Premature abstraction,
generic solutions for specific problems, and clever tricks that obscure
intent are prohibited. When choosing between a simple approach and an
elegant one, simplicity wins.

**Rationale**: ccdevkit tools are maintained by small teams and AI agents.
Code that is easy to read is easy to trust, review, and extend.

### II. Explicit Over Implicit

- Function signatures MUST communicate intent through clear naming and
  typed parameters.
- Error messages MUST include actionable context (file paths, values,
  what the caller should do).
- Dependencies MUST be obvious: no global state, no `init()` side
  effects, no hidden coupling.

**Rationale**: Implicit behavior creates debugging nightmares and makes
onboarding harder for both humans and AI agents.

### III. Fail Fast, Fail Clearly

- Inputs MUST be validated at system boundaries before processing.
- Errors MUST be returned promptly with context using `fmt.Errorf`
  and `%w` for wrapping.
- Errors MUST NOT be silently swallowed. Every ignored error requires
  an explicit comment justifying the decision.

**Rationale**: Late failures are expensive. Early, clear failures reduce
time-to-fix and prevent cascading issues.

### IV. Single Responsibility

Each package, type, and function MUST do one thing well:

- Packages own a single domain.
- Types represent a single concept.
- Functions perform a single operation.
- No "utils" or "helpers" grab-bag packages.

**Rationale**: Single-responsibility code is independently testable,
replaceable, and comprehensible.

### V. No Over-Engineering

- Build only what is needed now. Speculative generalization is
  prohibited.
- Three similar lines of code are preferable to a premature
  abstraction.
- Interfaces MUST NOT be introduced until a second consumer exists.

**Rationale**: Unused abstractions add cognitive load and maintenance
cost with zero current value.

### VI. Test What Matters

- Tests MUST cover the happy path, edge cases, and documented error
  conditions.
- Table-driven tests are REQUIRED for functions with multiple input
  cases.
- Tests MUST focus on behavior and outputs, not implementation details.
- Trivial getters, standard library functions, and code with no
  branching logic SHOULD NOT be tested.

**Rationale**: Tests are a safety net, not a coverage metric. Testing
behavior ensures refactoring freedom; testing implementation creates
brittle suites.

### VII. Red-Green-Refactor TDD

All new code MUST EITHER be developed using the Red-Green-Refactor cycle OR include explicit justification for exemption in code comments and the Complexity Tracking table:

1. **Red**: Write a failing test that defines the expected behavior
   before writing any implementation code. The test MUST compile and
   fail for the right reason (asserting the missing behavior, not a
   syntax error).
2. **Green**: Write the minimum implementation code required to make
   the failing test pass. No more, no less.
3. **Refactor**: Clean up the implementation and test code while
   keeping all tests green. Remove duplication, improve naming, and
   simplify structure.

Additional requirements:

- Each Red-Green-Refactor cycle MUST address a single behavior or
  requirement. Do not batch multiple behaviors into one cycle.
- Tests MUST be committed alongside or before the implementation they
  verify — never after.
- Bug fixes MUST begin with a failing test that reproduces the bug
  before applying the fix.
- Refactoring steps MUST NOT change behavior. If a refactoring
  requires new behavior, start a new Red-Green-Refactor cycle.

**Rationale**: Writing tests first forces clear thinking about
interfaces and expected behavior before implementation details cloud
judgment. The discipline prevents untested code from entering the
codebase and produces a living specification of system behavior.

## Coding Standards

- **Language**: Go (version matching the ccdevkit ecosystem).
- **File organization**: Package declaration, imports (stdlib / external
  / internal), constants, types, package variables, functions
  (constructors, methods, helpers).
- **Naming**: PascalCase for exports, camelCase for unexported symbols,
  lowercase single-word package names, `-er` suffix for single-method
  interfaces, all-caps for acronyms at word boundaries.
- **Error handling**: Always wrap with `%w`, lowercase messages, include
  context (paths, values). Sentinel errors only when callers need to
  match.
- **Documentation**: Every exported symbol MUST have a doc comment
  starting with its name. Comments explain _why_, not _what_.
- **Formatting**: `gofmt` / `goimports` enforced. Soft line limit 100
  chars, hard limit 120 chars.
- **Dependencies**: Prefer stdlib. Minimize external dependencies. No
  circular imports. Lower-level packages MUST NOT import higher-level
  packages.
- **Project structure**: `cmd/<tool>/` for CLI entry points (minimal
  wiring only), `internal/` for all business logic, `testdata/` for
  fixtures.

## Development Workflow

- **TDD cycle**: All new features and bug fixes MUST follow
  Red-Green-Refactor (Principle VII). Implementation PRs without
  corresponding test-first commits MUST be rejected at review.
- **Code review gates**: File organization correct, dependency direction
  downward, errors wrapped with context, doc comments on all exports,
  tests written before or alongside implementation, tests cover happy
  path and error cases, no hardcoded secrets, input validated at
  boundaries.
- **Commits**: Atomic, one logical change per commit. Conventional
  commit messages preferred. Test commits SHOULD precede or accompany
  implementation commits.

## Governance

This constitution is the authoritative source of coding standards for
ccbox. It supersedes ad-hoc conventions, oral agreements, and
conflicting documentation.

- **Amendments**: Any change to this constitution MUST be documented
  with a version bump, rationale, and migration plan for existing code
  that violates the new rule.
- **Versioning**: Semantic versioning applies. MAJOR for principle
  removals or redefinitions, MINOR for new principles or material
  expansions, PATCH for clarifications and typo fixes.
- **Compliance**: All code changes MUST be verified against these
  principles during review. Violations require explicit justification
  tracked in the Complexity Tracking section of the implementation plan.

**Version**: 1.1.0 | **Ratified**: 2026-03-07 | **Last Amended**: 2026-03-25
