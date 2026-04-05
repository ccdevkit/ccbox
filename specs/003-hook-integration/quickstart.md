# Quickstart: Hook Integration

**Branch**: `003-hook-integration`

## What This Feature Does

Enables ccbox to intercept and respond to Claude Code hook events during containerized sessions. Developers register internal hook handlers in Go, and the framework automatically:
1. Injects the correct settings into Claude Code's configuration
2. Runs a container-side proxy that forwards hook events to the host over TCP
3. Dispatches events to registered handlers and returns responses

## Usage Example (Internal API)

```go
// In session setup code:

// 1. Create a hook registry
registry := hooks.NewRegistry()

// 2. Register a typed handler (all PreToolUse events)
registry.Register(hooks.PreToolUseHandler{
    Order: hooks.OrderBefore,
    Fn: func(input *hooks.PreToolUseInput) (*hooks.PreToolUseOutput, error) {
        // Fully typed: input.ToolName, input.ToolInput, etc.
        return &hooks.PreToolUseOutput{}, nil
    },
})

// 3. Register with matcher (only Bash tool calls)
registry.Register(hooks.PreToolUseHandler{
    Matcher: "Bash",
    Order:   hooks.OrderBefore,
    Fn: func(input *hooks.PreToolUseInput) (*hooks.PreToolUseOutput, error) {
        // Only invoked for Bash tool calls
        return &hooks.PreToolUseOutput{}, nil
    },
})

// 4. Settings Manager merges hooks into Claude Code settings
settingsMgr, _ := settings.NewClaudeSettingsManager(osFS, homeDir, projectDir)
settingsMgr.MergeHooks(registry.HookEntries(proxyCommand), registry.OrderMap())
cliArgs, _ := settingsMgr.Finalize(session.FileWriter)

// 5. Bridge server dispatches incoming hook requests
bridgeServer := bridge.NewServer(execHandler, logHandler, registry.BridgeHandler())
```

## Architecture Overview

```
Host                              Container
┌──────────────┐                 ┌──────────────┐
│ Bridge Server │◄── TCP ──────►│ cchookproxy  │
│  (hook handler)│                │  (reads stdin,│
│              │                 │   sends TCP)  │
├──────────────┤                 ├──────────────┤
│ Hook Registry │                │ Claude Code   │
│  (dispatch)  │                 │  (fires hooks)│
├──────────────┤                 ├──────────────┤
│ Settings Mgr  │── writes ──►  │ settings.json │
│  (merge+inject)│  session file │  (hook config) │
└──────────────┘                 └──────────────┘
```

## Key Files

| File | Purpose |
|------|---------|
| `internal/claude/hooks/registry.go` | Handler registration and dispatch |
| `internal/claude/hooks/handlers.go` | Per-event handler structs (PreToolUseHandler, etc.) |
| `internal/claude/hooks/events.go` | HookEvent enum and Order type |
| `internal/claude/hooks/types.go` | HookInputBase, HookOutputBase, HandlerResult, BlockError |
| `internal/claude/hooks/types_*.go` | Typed input/output structs per event category |
| `internal/claude/settings/manager.go` | Settings Manager (discovery, merge, modification API) |
| `internal/bridge/types.go` | HookRequest/HookResponse wire types |
| `internal/bridge/server.go` | TCP server with hook dispatch |
| `cmd/cchookproxy/main.go` | Container-side hook proxy binary |
| `internal/constants/constants.go` | New constants (HookRequestType, etc.) |
