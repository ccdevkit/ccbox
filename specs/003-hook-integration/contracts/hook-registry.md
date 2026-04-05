# Contract: Hook Registry API

**Package**: `internal/claude/hooks`

## Core Types

```go
// HookHandler is implemented by per-event handler structs (e.g., PreToolUseHandler).
// The event name is implicit from the concrete type — no chance of mismatch.
type HookHandler interface {
    EventName() HookEvent
    MatcherPattern() string
    HandlerOrder() Order
    // invoke unmarshals the raw JSON input into the concrete typed input,
    // calls the user's Fn, and marshals the typed output back to a HandlerResult.
    invoke(input json.RawMessage) (*HandlerResult, error)
}

// HandlerResult is the wire-level response from a hook handler.
type HandlerResult struct {
    ExitCode int
    Stdout   []byte
    Stderr   []byte
}

// Order controls placement of ccbox hook entries relative to user hooks.
type Order string
const (
    OrderBefore Order = "before"
    OrderAfter  Order = "after"
)
```

## Per-Event Handler Structs

Each of the 26 hook events gets a handler struct. Example for PreToolUse:

```go
// PreToolUseHandler handles PreToolUse hook events with fully typed I/O.
type PreToolUseHandler struct {
    Matcher string // regex; empty = match all
    Order   Order
    Fn      func(input *PreToolUseInput) (*PreToolUseOutput, error)
}

func (h PreToolUseHandler) EventName() HookEvent       { return PreToolUse }
func (h PreToolUseHandler) MatcherPattern() string      { return h.Matcher }
func (h PreToolUseHandler) HandlerOrder() Order          { return h.Order }
func (h PreToolUseHandler) invoke(raw json.RawMessage) (*HandlerResult, error) {
    var input PreToolUseInput
    if err := json.Unmarshal(raw, &input); err != nil {
        return nil, fmt.Errorf("unmarshal PreToolUse input: %w", err)
    }
    output, err := h.Fn(&input)
    if err != nil {
        return nil, err
    }
    return output.toResult()
}
```

The same pattern repeats for all 26 events: `PostToolUseHandler`, `SessionStartHandler`,
`UserPromptSubmitHandler`, etc. Each has its own typed `Fn` field.

## Registry

```go
// Registry stores hook handler registrations.
type Registry struct { ... }

// NewRegistry creates an empty hook registry.
func NewRegistry() *Registry

// Register adds a typed handler. The event name, matcher, and order
// are carried by the handler struct itself.
func (r *Registry) Register(handler HookHandler)

// Dispatch finds all matching handlers for the given event and input,
// invokes them, and returns the aggregated result.
func (r *Registry) Dispatch(event HookEvent, input json.RawMessage) *HandlerResult

// HookEntries returns the settings.json hook configuration entries
// for all registered handlers, suitable for injection into the SettingsManager.
func (r *Registry) HookEntries(proxyCommand string) map[string][]MatcherGroup
```

## Usage

```go
registry := hooks.NewRegistry()

registry.Register(hooks.PreToolUseHandler{
    Matcher: "Bash",
    Order:   hooks.OrderBefore,
    Fn: func(input *hooks.PreToolUseInput) (*hooks.PreToolUseOutput, error) {
        // Fully typed: input.ToolName, input.ToolInput, etc.
        return &hooks.PreToolUseOutput{
            HookSpecificOutput: &hooks.PreToolUseSpecificOutput{
                PermissionDecision: "allow",
            },
        }, nil
    },
})

registry.Register(hooks.SessionStartHandler{
    Order: hooks.OrderAfter,
    Fn: func(input *hooks.SessionStartInput) (*hooks.SessionStartOutput, error) {
        // input.Source, input.Model — fully typed
        return &hooks.SessionStartOutput{}, nil
    },
})
```

## Dispatch Rules

1. Find all registrations matching the event name
2. For registrations with a matcher, check if the matcher regex matches the relevant field from the input (tool_name for tool events, source for SessionStart, etc.)
3. Invoke all matching handlers (sequentially on host side — Claude Code handles parallelism at the settings level)
4. Aggregate results:
   - If any handler returns ExitCode 2 → return ExitCode 2 (block)
   - If all return ExitCode 0 → merge stdout JSON, return ExitCode 0
   - If any returns other code → return that code (non-blocking error)
5. If no handlers match → return ExitCode 0, empty stdout
