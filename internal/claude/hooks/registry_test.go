package hooks

import (
	"context"
	"encoding/json"
	"errors"
	"testing"

	"github.com/ccdevkit/ccbox/internal/bridge"
)

func TestRegistry_Register_SingleHandler(t *testing.T) {
	r := NewRegistry()
	h := PreToolUseHandler{
		Matcher: "Bash",
		Order:   OrderBefore,
		Fn: func(input *PreToolUseInput) (*PreToolUseOutput, error) {
			return nil, nil
		},
	}

	r.Register(h)

	handlers := r.handlers[PreToolUse]
	if len(handlers) != 1 {
		t.Fatalf("expected 1 handler for PreToolUse, got %d", len(handlers))
	}
	if handlers[0].EventName() != PreToolUse {
		t.Errorf("expected event name %q, got %q", PreToolUse, handlers[0].EventName())
	}
}

func TestRegistry_Register_DifferentEvents(t *testing.T) {
	r := NewRegistry()

	r.Register(PreToolUseHandler{
		Matcher: "Bash",
		Order:   OrderBefore,
		Fn:      func(*PreToolUseInput) (*PreToolUseOutput, error) { return nil, nil },
	})
	r.Register(SessionStartHandler{
		Order: OrderBefore,
		Fn:    func(*SessionStartInput) (*SessionStartOutput, error) { return nil, nil },
	})

	if len(r.handlers[PreToolUse]) != 1 {
		t.Errorf("expected 1 handler for PreToolUse, got %d", len(r.handlers[PreToolUse]))
	}
	if len(r.handlers[SessionStart]) != 1 {
		t.Errorf("expected 1 handler for SessionStart, got %d", len(r.handlers[SessionStart]))
	}
}

func TestRegistry_Register_SameEventMultipleHandlers(t *testing.T) {
	r := NewRegistry()

	r.Register(PreToolUseHandler{
		Matcher: "Bash",
		Order:   OrderBefore,
		Fn:      func(*PreToolUseInput) (*PreToolUseOutput, error) { return nil, nil },
	})
	r.Register(PreToolUseHandler{
		Matcher: "Read",
		Order:   OrderBefore,
		Fn:      func(*PreToolUseInput) (*PreToolUseOutput, error) { return nil, nil },
	})

	handlers := r.handlers[PreToolUse]
	if len(handlers) != 2 {
		t.Fatalf("expected 2 handlers for PreToolUse, got %d", len(handlers))
	}
	if handlers[0].MatcherPattern() != "Bash" {
		t.Errorf("expected first handler matcher %q, got %q", "Bash", handlers[0].MatcherPattern())
	}
	if handlers[1].MatcherPattern() != "Read" {
		t.Errorf("expected second handler matcher %q, got %q", "Read", handlers[1].MatcherPattern())
	}
}

func TestRegistry_Dispatch_SingleHandler(t *testing.T) {
	r := NewRegistry()

	r.Register(PreToolUseHandler{
		Matcher: "Bash",
		Order:   OrderBefore,
		Fn: func(input *PreToolUseInput) (*PreToolUseOutput, error) {
			return &PreToolUseOutput{
				HookSpecificOutput: &PreToolUseSpecificOutput{
					PermissionDecision: "allow",
				},
			}, nil
		},
	})

	inputJSON := json.RawMessage(`{"hook_event_name":"PreToolUse","tool_name":"Bash","tool_input":{},"session_id":"s1","transcript_path":"/t","cwd":"/","permission_mode":"default"}`)

	result := r.Dispatch(PreToolUse, inputJSON)

	if result.ExitCode != 0 {
		t.Fatalf("expected exit code 0, got %d", result.ExitCode)
	}

	var output PreToolUseOutput
	if err := json.Unmarshal(result.Stdout, &output); err != nil {
		t.Fatalf("failed to unmarshal stdout: %v", err)
	}
	if output.HookSpecificOutput == nil {
		t.Fatal("expected HookSpecificOutput to be non-nil")
	}
	if output.HookSpecificOutput.PermissionDecision != "allow" {
		t.Errorf("expected permissionDecision %q, got %q", "allow", output.HookSpecificOutput.PermissionDecision)
	}
}

func TestRegistry_Dispatch_NoHandlers(t *testing.T) {
	r := NewRegistry()

	result := r.Dispatch(PreToolUse, json.RawMessage(`{}`))

	if result.ExitCode != 0 {
		t.Errorf("expected exit code 0, got %d", result.ExitCode)
	}
	if len(result.Stdout) != 0 {
		t.Errorf("expected empty stdout, got %q", string(result.Stdout))
	}
	if len(result.Stderr) != 0 {
		t.Errorf("expected empty stderr, got %q", string(result.Stderr))
	}
}

func TestRegistry_Dispatch_NoMatchingEvent(t *testing.T) {
	r := NewRegistry()

	r.Register(PreToolUseHandler{
		Matcher: "Bash",
		Order:   OrderBefore,
		Fn: func(input *PreToolUseInput) (*PreToolUseOutput, error) {
			t.Fatal("handler should not be called for a different event")
			return nil, nil
		},
	})

	result := r.Dispatch(PostToolUse, json.RawMessage(`{}`))

	if result.ExitCode != 0 {
		t.Errorf("expected exit code 0, got %d", result.ExitCode)
	}
	if len(result.Stdout) != 0 {
		t.Errorf("expected empty stdout, got %q", string(result.Stdout))
	}
	if len(result.Stderr) != 0 {
		t.Errorf("expected empty stderr, got %q", string(result.Stderr))
	}
}

func TestRegistry_Dispatch_MatcherFilters_MatchingToolName(t *testing.T) {
	r := NewRegistry()

	r.Register(PreToolUseHandler{
		Matcher: "Bash",
		Order:   OrderBefore,
		Fn: func(input *PreToolUseInput) (*PreToolUseOutput, error) {
			return &PreToolUseOutput{
				HookSpecificOutput: &PreToolUseSpecificOutput{
					PermissionDecision: "allow",
				},
			}, nil
		},
	})

	inputJSON := json.RawMessage(`{"hook_event_name":"PreToolUse","tool_name":"Bash","tool_input":{},"session_id":"s1","transcript_path":"/t","cwd":"/","permission_mode":"default"}`)

	result := r.Dispatch(PreToolUse, inputJSON)

	if result.ExitCode != 0 {
		t.Fatalf("expected exit code 0, got %d", result.ExitCode)
	}

	var output PreToolUseOutput
	if err := json.Unmarshal(result.Stdout, &output); err != nil {
		t.Fatalf("failed to unmarshal stdout: %v", err)
	}
	if output.HookSpecificOutput == nil || output.HookSpecificOutput.PermissionDecision != "allow" {
		t.Errorf("expected handler to be invoked with allow decision")
	}
}

func TestRegistry_Dispatch_MatcherFilters_NonMatchingToolName(t *testing.T) {
	r := NewRegistry()

	called := false
	r.Register(PreToolUseHandler{
		Matcher: "Bash",
		Order:   OrderBefore,
		Fn: func(input *PreToolUseInput) (*PreToolUseOutput, error) {
			called = true
			return nil, nil
		},
	})

	inputJSON := json.RawMessage(`{"hook_event_name":"PreToolUse","tool_name":"Read","tool_input":{},"session_id":"s1","transcript_path":"/t","cwd":"/","permission_mode":"default"}`)

	result := r.Dispatch(PreToolUse, inputJSON)

	if called {
		t.Fatal("handler should not be called for non-matching tool_name")
	}
	if result.ExitCode != 0 {
		t.Errorf("expected exit code 0, got %d", result.ExitCode)
	}
	if len(result.Stdout) != 0 {
		t.Errorf("expected empty stdout, got %q", string(result.Stdout))
	}
}

func TestRegistry_Dispatch_MatcherFilters_EmptyMatcherMatchesAll(t *testing.T) {
	r := NewRegistry()

	called := false
	r.Register(PreToolUseHandler{
		Matcher: "",
		Order:   OrderBefore,
		Fn: func(input *PreToolUseInput) (*PreToolUseOutput, error) {
			called = true
			return &PreToolUseOutput{
				HookSpecificOutput: &PreToolUseSpecificOutput{
					PermissionDecision: "deny",
				},
			}, nil
		},
	})

	inputJSON := json.RawMessage(`{"hook_event_name":"PreToolUse","tool_name":"Read","tool_input":{},"session_id":"s1","transcript_path":"/t","cwd":"/","permission_mode":"default"}`)

	result := r.Dispatch(PreToolUse, inputJSON)

	if !called {
		t.Fatal("handler with empty matcher should be called for any tool_name")
	}
	if result.ExitCode != 0 {
		t.Fatalf("expected exit code 0, got %d", result.ExitCode)
	}
}

func TestRegistry_Dispatch_MatcherFilters_RegexMatchesMultipleTools(t *testing.T) {
	r := NewRegistry()

	callCount := 0
	r.Register(PreToolUseHandler{
		Matcher: "Bash|Write",
		Order:   OrderBefore,
		Fn: func(input *PreToolUseInput) (*PreToolUseOutput, error) {
			callCount++
			return &PreToolUseOutput{
				HookSpecificOutput: &PreToolUseSpecificOutput{
					PermissionDecision: "allow",
				},
			}, nil
		},
	})

	// Should match "Bash"
	result := r.Dispatch(PreToolUse, json.RawMessage(`{"hook_event_name":"PreToolUse","tool_name":"Bash","tool_input":{},"session_id":"s1","transcript_path":"/t","cwd":"/","permission_mode":"default"}`))
	if result.ExitCode != 0 || callCount != 1 {
		t.Fatalf("expected handler called once for Bash, got callCount=%d exitCode=%d", callCount, result.ExitCode)
	}

	// Should match "Write"
	result = r.Dispatch(PreToolUse, json.RawMessage(`{"hook_event_name":"PreToolUse","tool_name":"Write","tool_input":{},"session_id":"s1","transcript_path":"/t","cwd":"/","permission_mode":"default"}`))
	if result.ExitCode != 0 || callCount != 2 {
		t.Fatalf("expected handler called twice for Write, got callCount=%d exitCode=%d", callCount, result.ExitCode)
	}

	// Should NOT match "Read"
	result = r.Dispatch(PreToolUse, json.RawMessage(`{"hook_event_name":"PreToolUse","tool_name":"Read","tool_input":{},"session_id":"s1","transcript_path":"/t","cwd":"/","permission_mode":"default"}`))
	if callCount != 2 {
		t.Fatalf("expected handler NOT called for Read, got callCount=%d", callCount)
	}
	if result.ExitCode != 0 {
		t.Errorf("expected exit code 0 for non-matching, got %d", result.ExitCode)
	}
}

func TestRegistry_Dispatch_MultiHandler_BlockWinsOverAllow(t *testing.T) {
	r := NewRegistry()

	// Handler A: allows (exit 0)
	r.Register(PreToolUseHandler{
		Order: OrderBefore,
		Fn: func(input *PreToolUseInput) (*PreToolUseOutput, error) {
			return &PreToolUseOutput{
				HookSpecificOutput: &PreToolUseSpecificOutput{
					PermissionDecision: "allow",
				},
			}, nil
		},
	})

	// Handler B: blocks (exit 2) with stderr "blocked"
	r.Register(PreToolUseHandler{
		Order: OrderBefore,
		Fn: func(input *PreToolUseInput) (*PreToolUseOutput, error) {
			return nil, &BlockError{Message: "blocked"}
		},
	})

	inputJSON := json.RawMessage(`{"hook_event_name":"PreToolUse","tool_name":"Bash","tool_input":{},"session_id":"s1","transcript_path":"/t","cwd":"/","permission_mode":"default"}`)

	result := r.Dispatch(PreToolUse, inputJSON)

	if result.ExitCode != 2 {
		t.Fatalf("expected exit code 2 (block), got %d", result.ExitCode)
	}
	if string(result.Stderr) != "blocked" {
		t.Errorf("expected stderr %q, got %q", "blocked", string(result.Stderr))
	}
}

func TestRegistry_Dispatch_MultiHandler_AllAllow(t *testing.T) {
	r := NewRegistry()

	r.Register(PreToolUseHandler{
		Order: OrderBefore,
		Fn: func(input *PreToolUseInput) (*PreToolUseOutput, error) {
			return &PreToolUseOutput{
				HookSpecificOutput: &PreToolUseSpecificOutput{
					PermissionDecision: "allow",
				},
			}, nil
		},
	})

	r.Register(PreToolUseHandler{
		Order: OrderBefore,
		Fn: func(input *PreToolUseInput) (*PreToolUseOutput, error) {
			return &PreToolUseOutput{
				HookSpecificOutput: &PreToolUseSpecificOutput{
					PermissionDecision: "allow",
				},
			}, nil
		},
	})

	inputJSON := json.RawMessage(`{"hook_event_name":"PreToolUse","tool_name":"Bash","tool_input":{},"session_id":"s1","transcript_path":"/t","cwd":"/","permission_mode":"default"}`)

	result := r.Dispatch(PreToolUse, inputJSON)

	if result.ExitCode != 0 {
		t.Fatalf("expected exit code 0, got %d", result.ExitCode)
	}
}

func TestRegistry_Dispatch_MultiHandler_ErrorWinsOverAllow(t *testing.T) {
	r := NewRegistry()

	// Handler A: allows (exit 0)
	r.Register(PreToolUseHandler{
		Order: OrderBefore,
		Fn: func(input *PreToolUseInput) (*PreToolUseOutput, error) {
			return &PreToolUseOutput{
				HookSpecificOutput: &PreToolUseSpecificOutput{
					PermissionDecision: "allow",
				},
			}, nil
		},
	})

	// Handler B: returns error (exit 1)
	r.Register(PreToolUseHandler{
		Order: OrderBefore,
		Fn: func(input *PreToolUseInput) (*PreToolUseOutput, error) {
			return nil, errors.New("handler error")
		},
	})

	inputJSON := json.RawMessage(`{"hook_event_name":"PreToolUse","tool_name":"Bash","tool_input":{},"session_id":"s1","transcript_path":"/t","cwd":"/","permission_mode":"default"}`)

	result := r.Dispatch(PreToolUse, inputJSON)

	if result.ExitCode != 1 {
		t.Fatalf("expected exit code 1, got %d", result.ExitCode)
	}
	if string(result.Stderr) != "handler error" {
		t.Errorf("expected stderr %q, got %q", "handler error", string(result.Stderr))
	}
}

func TestRegistry_Dispatch_DecisionPrecedence_DenyWinsOverAllow(t *testing.T) {
	r := NewRegistry()

	// deny registered first — without precedence logic, "allow" (last) would win
	r.Register(PreToolUseHandler{
		Order: OrderBefore,
		Fn: func(input *PreToolUseInput) (*PreToolUseOutput, error) {
			return &PreToolUseOutput{
				HookSpecificOutput: &PreToolUseSpecificOutput{
					PermissionDecision: "deny",
				},
			}, nil
		},
	})

	r.Register(PreToolUseHandler{
		Order: OrderBefore,
		Fn: func(input *PreToolUseInput) (*PreToolUseOutput, error) {
			return &PreToolUseOutput{
				HookSpecificOutput: &PreToolUseSpecificOutput{
					PermissionDecision: "allow",
				},
			}, nil
		},
	})

	inputJSON := json.RawMessage(`{"hook_event_name":"PreToolUse","tool_name":"Bash","tool_input":{},"session_id":"s1","transcript_path":"/t","cwd":"/","permission_mode":"default"}`)

	result := r.Dispatch(PreToolUse, inputJSON)

	if result.ExitCode != 0 {
		t.Fatalf("expected exit code 0, got %d", result.ExitCode)
	}

	var output PreToolUseOutput
	if err := json.Unmarshal(result.Stdout, &output); err != nil {
		t.Fatalf("failed to unmarshal stdout: %v", err)
	}
	if output.HookSpecificOutput == nil {
		t.Fatal("expected HookSpecificOutput to be non-nil")
	}
	if output.HookSpecificOutput.PermissionDecision != "deny" {
		t.Errorf("expected permissionDecision %q, got %q", "deny", output.HookSpecificOutput.PermissionDecision)
	}
}

func TestRegistry_Dispatch_DecisionPrecedence_DeferWinsOverAsk(t *testing.T) {
	r := NewRegistry()

	// defer registered first — without precedence logic, "ask" (last) would win
	r.Register(PreToolUseHandler{
		Order: OrderBefore,
		Fn: func(input *PreToolUseInput) (*PreToolUseOutput, error) {
			return &PreToolUseOutput{
				HookSpecificOutput: &PreToolUseSpecificOutput{
					PermissionDecision: "defer",
				},
			}, nil
		},
	})

	r.Register(PreToolUseHandler{
		Order: OrderBefore,
		Fn: func(input *PreToolUseInput) (*PreToolUseOutput, error) {
			return &PreToolUseOutput{
				HookSpecificOutput: &PreToolUseSpecificOutput{
					PermissionDecision: "ask",
				},
			}, nil
		},
	})

	inputJSON := json.RawMessage(`{"hook_event_name":"PreToolUse","tool_name":"Bash","tool_input":{},"session_id":"s1","transcript_path":"/t","cwd":"/","permission_mode":"default"}`)

	result := r.Dispatch(PreToolUse, inputJSON)

	if result.ExitCode != 0 {
		t.Fatalf("expected exit code 0, got %d", result.ExitCode)
	}

	var output PreToolUseOutput
	if err := json.Unmarshal(result.Stdout, &output); err != nil {
		t.Fatalf("failed to unmarshal stdout: %v", err)
	}
	if output.HookSpecificOutput == nil {
		t.Fatal("expected HookSpecificOutput to be non-nil")
	}
	if output.HookSpecificOutput.PermissionDecision != "defer" {
		t.Errorf("expected permissionDecision %q, got %q", "defer", output.HookSpecificOutput.PermissionDecision)
	}
}

func TestRegistry_Dispatch_DecisionPrecedence_AskWinsOverAllow(t *testing.T) {
	r := NewRegistry()

	// ask registered first — without precedence logic, "allow" (last) would win
	r.Register(PreToolUseHandler{
		Order: OrderBefore,
		Fn: func(input *PreToolUseInput) (*PreToolUseOutput, error) {
			return &PreToolUseOutput{
				HookSpecificOutput: &PreToolUseSpecificOutput{
					PermissionDecision: "ask",
				},
			}, nil
		},
	})

	r.Register(PreToolUseHandler{
		Order: OrderBefore,
		Fn: func(input *PreToolUseInput) (*PreToolUseOutput, error) {
			return &PreToolUseOutput{
				HookSpecificOutput: &PreToolUseSpecificOutput{
					PermissionDecision: "allow",
				},
			}, nil
		},
	})

	inputJSON := json.RawMessage(`{"hook_event_name":"PreToolUse","tool_name":"Bash","tool_input":{},"session_id":"s1","transcript_path":"/t","cwd":"/","permission_mode":"default"}`)

	result := r.Dispatch(PreToolUse, inputJSON)

	if result.ExitCode != 0 {
		t.Fatalf("expected exit code 0, got %d", result.ExitCode)
	}

	var output PreToolUseOutput
	if err := json.Unmarshal(result.Stdout, &output); err != nil {
		t.Fatalf("failed to unmarshal stdout: %v", err)
	}
	if output.HookSpecificOutput == nil {
		t.Fatal("expected HookSpecificOutput to be non-nil")
	}
	if output.HookSpecificOutput.PermissionDecision != "ask" {
		t.Errorf("expected permissionDecision %q, got %q", "ask", output.HookSpecificOutput.PermissionDecision)
	}
}

func TestRegistry_Dispatch_DecisionPrecedence_BothAllow(t *testing.T) {
	r := NewRegistry()

	r.Register(PreToolUseHandler{
		Order: OrderBefore,
		Fn: func(input *PreToolUseInput) (*PreToolUseOutput, error) {
			return &PreToolUseOutput{
				HookSpecificOutput: &PreToolUseSpecificOutput{
					PermissionDecision: "allow",
				},
			}, nil
		},
	})

	r.Register(PreToolUseHandler{
		Order: OrderBefore,
		Fn: func(input *PreToolUseInput) (*PreToolUseOutput, error) {
			return &PreToolUseOutput{
				HookSpecificOutput: &PreToolUseSpecificOutput{
					PermissionDecision: "allow",
				},
			}, nil
		},
	})

	inputJSON := json.RawMessage(`{"hook_event_name":"PreToolUse","tool_name":"Bash","tool_input":{},"session_id":"s1","transcript_path":"/t","cwd":"/","permission_mode":"default"}`)

	result := r.Dispatch(PreToolUse, inputJSON)

	if result.ExitCode != 0 {
		t.Fatalf("expected exit code 0, got %d", result.ExitCode)
	}

	var output PreToolUseOutput
	if err := json.Unmarshal(result.Stdout, &output); err != nil {
		t.Fatalf("failed to unmarshal stdout: %v", err)
	}
	if output.HookSpecificOutput == nil {
		t.Fatal("expected HookSpecificOutput to be non-nil")
	}
	if output.HookSpecificOutput.PermissionDecision != "allow" {
		t.Errorf("expected permissionDecision %q, got %q", "allow", output.HookSpecificOutput.PermissionDecision)
	}
}

func TestRegistry_Dispatch_HandlerError(t *testing.T) {
	r := NewRegistry()

	r.Register(PreToolUseHandler{
		Matcher: "Bash",
		Order:   OrderBefore,
		Fn: func(input *PreToolUseInput) (*PreToolUseOutput, error) {
			return nil, errors.New("handler failed")
		},
	})

	inputJSON := json.RawMessage(`{"hook_event_name":"PreToolUse","tool_name":"Bash","tool_input":{},"session_id":"s1","transcript_path":"/t","cwd":"/","permission_mode":"default"}`)

	result := r.Dispatch(PreToolUse, inputJSON)

	if result.ExitCode != 1 {
		t.Errorf("expected exit code 1, got %d", result.ExitCode)
	}
	if string(result.Stderr) != "handler failed" {
		t.Errorf("expected stderr %q, got %q", "handler failed", string(result.Stderr))
	}
}

func TestRegistry_Dispatch_ContinueFalse_SingleHandler(t *testing.T) {
	r := NewRegistry()

	boolFalse := false
	r.Register(SessionStartHandler{
		Order: OrderBefore,
		Fn: func(input *SessionStartInput) (*SessionStartOutput, error) {
			return &SessionStartOutput{
				HookOutputBase: HookOutputBase{
					Continue:   &boolFalse,
					StopReason: "stopped by handler",
				},
			}, nil
		},
	})

	inputJSON := json.RawMessage(`{"hook_event_name":"SessionStart","session_id":"s1","transcript_path":"/t","cwd":"/","permission_mode":"default"}`)

	result := r.Dispatch(SessionStart, inputJSON)

	if result.ExitCode != 0 {
		t.Fatalf("expected exit code 0, got %d", result.ExitCode)
	}

	var parsed struct {
		Continue   *bool  `json:"continue,omitempty"`
		StopReason string `json:"stopReason,omitempty"`
	}
	if err := json.Unmarshal(result.Stdout, &parsed); err != nil {
		t.Fatalf("failed to unmarshal stdout: %v", err)
	}
	if parsed.Continue == nil || *parsed.Continue != false {
		t.Errorf("expected continue=false in stdout")
	}
	if parsed.StopReason != "stopped by handler" {
		t.Errorf("expected stopReason %q, got %q", "stopped by handler", parsed.StopReason)
	}
}

func TestRegistry_Dispatch_ContinueFalse_TwoHandlers_NormalAndStop(t *testing.T) {
	r := NewRegistry()

	// Handler A: normal output (continue not set)
	r.Register(SessionStartHandler{
		Order: OrderBefore,
		Fn: func(input *SessionStartInput) (*SessionStartOutput, error) {
			return &SessionStartOutput{}, nil
		},
	})

	// Handler B: continue:false with stopReason
	boolFalse := false
	r.Register(SessionStartHandler{
		Order: OrderBefore,
		Fn: func(input *SessionStartInput) (*SessionStartOutput, error) {
			return &SessionStartOutput{
				HookOutputBase: HookOutputBase{
					Continue:   &boolFalse,
					StopReason: "stopped by handler",
				},
			}, nil
		},
	})

	inputJSON := json.RawMessage(`{"hook_event_name":"SessionStart","session_id":"s1","transcript_path":"/t","cwd":"/","permission_mode":"default"}`)

	result := r.Dispatch(SessionStart, inputJSON)

	if result.ExitCode != 0 {
		t.Fatalf("expected exit code 0, got %d", result.ExitCode)
	}

	var parsed struct {
		Continue   *bool  `json:"continue,omitempty"`
		StopReason string `json:"stopReason,omitempty"`
	}
	if err := json.Unmarshal(result.Stdout, &parsed); err != nil {
		t.Fatalf("failed to unmarshal stdout: %v", err)
	}
	if parsed.Continue == nil || *parsed.Continue != false {
		t.Errorf("expected continue=false in stdout")
	}
	if parsed.StopReason != "stopped by handler" {
		t.Errorf("expected stopReason %q, got %q", "stopped by handler", parsed.StopReason)
	}
}

func TestRegistry_HookEntries_CatchAllPerEvent(t *testing.T) {
	r := NewRegistry()

	// Register handlers with different matchers for same event
	r.Register(PreToolUseHandler{
		Matcher: "Bash",
		Order:   OrderBefore,
		Fn:      func(*PreToolUseInput) (*PreToolUseOutput, error) { return nil, nil },
	})
	r.Register(PreToolUseHandler{
		Matcher: "Edit",
		Order:   OrderAfter,
		Fn:      func(*PreToolUseInput) (*PreToolUseOutput, error) { return nil, nil },
	})

	entries := r.HookEntries("/opt/ccbox/bin/cchookproxy")

	groups, ok := entries["PreToolUse"]
	if !ok {
		t.Fatal("expected PreToolUse key in entries")
	}
	// Should have ONE catch-all group, not per-matcher groups
	if len(groups) != 1 {
		t.Fatalf("expected 1 catch-all matcher group, got %d", len(groups))
	}
	if groups[0].Matcher != "*" {
		t.Errorf("matcher = %q, want %q (catch-all)", groups[0].Matcher, "*")
	}
	if len(groups[0].Hooks) != 1 {
		t.Fatalf("expected 1 hook entry, got %d", len(groups[0].Hooks))
	}
	if groups[0].Hooks[0].Command != "/opt/ccbox/bin/cchookproxy" {
		t.Errorf("command = %q, want %q", groups[0].Hooks[0].Command, "/opt/ccbox/bin/cchookproxy")
	}
}

func TestRegistry_HookEntries_MultipleEvents(t *testing.T) {
	r := NewRegistry()

	r.Register(PreToolUseHandler{
		Matcher: "Bash",
		Order:   OrderBefore,
		Fn:      func(*PreToolUseInput) (*PreToolUseOutput, error) { return nil, nil },
	})
	r.Register(SessionStartHandler{
		Order: OrderBefore,
		Fn:    func(*SessionStartInput) (*SessionStartOutput, error) { return nil, nil },
	})

	entries := r.HookEntries("/opt/ccbox/bin/cchookproxy")

	if len(entries) != 2 {
		t.Fatalf("expected 2 events, got %d", len(entries))
	}
	if _, ok := entries["PreToolUse"]; !ok {
		t.Error("missing PreToolUse entries")
	}
	if _, ok := entries["SessionStart"]; !ok {
		t.Error("missing SessionStart entries")
	}
}

func TestRegistry_HookEntries_IncludesUserHookEvents(t *testing.T) {
	r := NewRegistry()

	// Only user hooks, no Go handlers for this event
	r.SetUserHooks(PostToolUse, []UserHook{{Command: "user-hook"}})

	entries := r.HookEntries("/opt/ccbox/bin/cchookproxy")

	if _, ok := entries["PostToolUse"]; !ok {
		t.Error("expected PostToolUse entry from user hooks")
	}
}

func TestRegistry_HookEntries_EmptyRegistry(t *testing.T) {
	r := NewRegistry()

	entries := r.HookEntries("/opt/ccbox/bin/cchookproxy")

	if len(entries) != 0 {
		t.Errorf("expected empty entries for empty registry, got %d entries", len(entries))
	}
}

func TestRegistry_RegisteredEvents(t *testing.T) {
	r := NewRegistry()

	r.Register(PreToolUseHandler{
		Order: OrderBefore,
		Fn:    func(*PreToolUseInput) (*PreToolUseOutput, error) { return nil, nil },
	})
	r.Register(SessionStartHandler{
		Order: OrderBefore,
		Fn:    func(*SessionStartInput) (*SessionStartOutput, error) { return nil, nil },
	})

	events := r.RegisteredEvents()

	if !events["PreToolUse"] {
		t.Error("missing PreToolUse")
	}
	if !events["SessionStart"] {
		t.Error("missing SessionStart")
	}
	if events["PostToolUse"] {
		t.Error("PostToolUse should not be registered")
	}
}

func TestRegistry_Dispatch_ContinueFalse_TakesPrecedenceOverBlock(t *testing.T) {
	r := NewRegistry()

	// Handler A: continue:false
	boolFalse := false
	r.Register(PreToolUseHandler{
		Order: OrderBefore,
		Fn: func(input *PreToolUseInput) (*PreToolUseOutput, error) {
			return &PreToolUseOutput{
				HookOutputBase: HookOutputBase{
					Continue:   &boolFalse,
					StopReason: "halted",
				},
			}, nil
		},
	})

	// Handler B: block (exit 2)
	r.Register(PreToolUseHandler{
		Order: OrderBefore,
		Fn: func(input *PreToolUseInput) (*PreToolUseOutput, error) {
			return nil, &BlockError{Message: "blocked"}
		},
	})

	inputJSON := json.RawMessage(`{"hook_event_name":"PreToolUse","tool_name":"Bash","tool_input":{},"session_id":"s1","transcript_path":"/t","cwd":"/","permission_mode":"default"}`)

	result := r.Dispatch(PreToolUse, inputJSON)

	// continue:false takes precedence over block — exit 0 with continue:false in stdout
	if result.ExitCode != 0 {
		t.Fatalf("expected exit code 0 (continue:false precedence), got %d", result.ExitCode)
	}

	var parsed struct {
		Continue   *bool  `json:"continue,omitempty"`
		StopReason string `json:"stopReason,omitempty"`
	}
	if err := json.Unmarshal(result.Stdout, &parsed); err != nil {
		t.Fatalf("failed to unmarshal stdout: %v", err)
	}
	if parsed.Continue == nil || *parsed.Continue != false {
		t.Errorf("expected continue=false in stdout")
	}
	if parsed.StopReason != "halted" {
		t.Errorf("expected stopReason %q, got %q", "halted", parsed.StopReason)
	}
}

func TestRegistry_BridgeHandler_DispatchesToRegisteredHandler(t *testing.T) {
	r := NewRegistry()

	r.Register(PreToolUseHandler{
		Matcher: "Bash",
		Order:   OrderBefore,
		Fn: func(input *PreToolUseInput) (*PreToolUseOutput, error) {
			return &PreToolUseOutput{
				HookSpecificOutput: &PreToolUseSpecificOutput{
					PermissionDecision: "allow",
				},
			}, nil
		},
	})

	handler := r.BridgeHandler()

	req := bridge.HookRequest{
		Type:  "hook",
		Event: "PreToolUse",
		Input: json.RawMessage(`{"hook_event_name":"PreToolUse","tool_name":"Bash","tool_input":{},"session_id":"s1","transcript_path":"/t","cwd":"/","permission_mode":"default"}`),
	}

	resp := handler(req)

	if resp.ExitCode != 0 {
		t.Fatalf("expected exit code 0, got %d", resp.ExitCode)
	}
	if resp.Stdout == "" {
		t.Fatal("expected non-empty stdout")
	}

	var output PreToolUseOutput
	if err := json.Unmarshal([]byte(resp.Stdout), &output); err != nil {
		t.Fatalf("failed to unmarshal stdout: %v", err)
	}
	if output.HookSpecificOutput == nil {
		t.Fatal("expected HookSpecificOutput to be non-nil")
	}
	if output.HookSpecificOutput.PermissionDecision != "allow" {
		t.Errorf("expected permissionDecision %q, got %q", "allow", output.HookSpecificOutput.PermissionDecision)
	}
	if resp.Stderr != "" {
		t.Errorf("expected empty stderr, got %q", resp.Stderr)
	}
}

// --- Three-phase Dispatch tests ---

// mockCmdRunner is a test CommandRunner for three-phase dispatch tests.
type mockCmdRunner struct {
	exitCode int
	stdout   []byte
	stderr   []byte
}

func (m *mockCmdRunner) Run(_ context.Context, command string, stdin []byte, env []string, dir string) (int, []byte, []byte, error) {
	return m.exitCode, m.stdout, m.stderr, nil
}

func preToolUseInput(toolName, command string) json.RawMessage {
	data, _ := json.Marshal(map[string]interface{}{
		"hook_event_name": "PreToolUse",
		"tool_name":       toolName,
		"tool_input":      map[string]interface{}{"command": command},
		"session_id":      "s1",
		"transcript_path": "/t",
		"cwd":             "/",
		"permission_mode": "default",
	})
	return data
}

func TestRegistry_Dispatch_ThreePhase_BeforeBlocksShortCircuits(t *testing.T) {
	r := NewRegistry()

	// Register a "before" handler that blocks
	r.Register(PreToolUseHandler{
		Matcher: "Bash",
		Order:   OrderBefore,
		Fn: func(input *PreToolUseInput) (*PreToolUseOutput, error) {
			return nil, &BlockError{Message: "blocked by before handler"}
		},
	})

	// Register an "after" handler that should NOT be called
	afterCalled := false
	r.Register(PreToolUseHandler{
		Matcher: "Bash",
		Order:   OrderAfter,
		Fn: func(input *PreToolUseInput) (*PreToolUseOutput, error) {
			afterCalled = true
			return nil, nil
		},
	})

	// Add user hooks that should NOT be executed
	userRunner := &mockCmdRunner{exitCode: 0}
	r.SetCommandRunner(userRunner)
	r.SetUserHooks(PreToolUse, []UserHook{{Command: "should-not-run", Matcher: "Bash"}})

	result := r.Dispatch(PreToolUse, preToolUseInput("Bash", "test"))

	if result.ExitCode != 2 {
		t.Errorf("ExitCode = %d, want 2 (blocked)", result.ExitCode)
	}
	if afterCalled {
		t.Error("after handler should not be called when before handler blocks")
	}
}

func TestRegistry_Dispatch_ThreePhase_AllPhasesRun(t *testing.T) {
	r := NewRegistry()

	beforeCalled := false
	afterCalled := false

	r.Register(PreToolUseHandler{
		Matcher: "Bash",
		Order:   OrderBefore,
		Fn: func(input *PreToolUseInput) (*PreToolUseOutput, error) {
			beforeCalled = true
			return nil, nil
		},
	})

	r.Register(PreToolUseHandler{
		Matcher: "Bash",
		Order:   OrderAfter,
		Fn: func(input *PreToolUseInput) (*PreToolUseOutput, error) {
			afterCalled = true
			return nil, nil
		},
	})

	// Add user hooks
	userRunner := &mockCmdRunner{exitCode: 0, stdout: []byte(`{}`)}
	r.SetCommandRunner(userRunner)
	r.SetUserHooks(PreToolUse, []UserHook{{Command: "user-hook", Matcher: "Bash"}})

	result := r.Dispatch(PreToolUse, preToolUseInput("Bash", "test"))

	if result.ExitCode != 0 {
		t.Errorf("ExitCode = %d, want 0", result.ExitCode)
	}
	if !beforeCalled {
		t.Error("before handler was not called")
	}
	if !afterCalled {
		t.Error("after handler was not called")
	}
}

func TestRegistry_Dispatch_ThreePhase_UserHookBlocksSkipsAfter(t *testing.T) {
	r := NewRegistry()

	beforeCalled := false
	afterCalled := false

	r.Register(PreToolUseHandler{
		Matcher: "Bash",
		Order:   OrderBefore,
		Fn: func(input *PreToolUseInput) (*PreToolUseOutput, error) {
			beforeCalled = true
			return nil, nil
		},
	})

	r.Register(PreToolUseHandler{
		Matcher: "Bash",
		Order:   OrderAfter,
		Fn: func(input *PreToolUseInput) (*PreToolUseOutput, error) {
			afterCalled = true
			return nil, nil
		},
	})

	// User hook returns exit code 2 (block)
	userRunner := &mockCmdRunner{exitCode: 2, stderr: []byte("blocked by user")}
	r.SetCommandRunner(userRunner)
	r.SetUserHooks(PreToolUse, []UserHook{{Command: "blocking-hook", Matcher: "Bash"}})

	result := r.Dispatch(PreToolUse, preToolUseInput("Bash", "test"))

	if result.ExitCode != 2 {
		t.Errorf("ExitCode = %d, want 2", result.ExitCode)
	}
	if !beforeCalled {
		t.Error("before handler should be called")
	}
	if afterCalled {
		t.Error("after handler should NOT be called when user hooks block")
	}
}

func TestRegistry_Dispatch_ThreePhase_ContinueFalseFromBefore(t *testing.T) {
	r := NewRegistry()

	r.Register(PreToolUseHandler{
		Matcher: "Bash",
		Order:   OrderBefore,
		Fn: func(input *PreToolUseInput) (*PreToolUseOutput, error) {
			f := false
			return &PreToolUseOutput{
				HookOutputBase: HookOutputBase{Continue: &f, StopReason: "halted by before"},
			}, nil
		},
	})

	afterCalled := false
	r.Register(PreToolUseHandler{
		Matcher: "Bash",
		Order:   OrderAfter,
		Fn: func(input *PreToolUseInput) (*PreToolUseOutput, error) {
			afterCalled = true
			return nil, nil
		},
	})

	result := r.Dispatch(PreToolUse, preToolUseInput("Bash", "test"))

	stop, reason := extractContinueFalse(result.Stdout)
	if !stop {
		t.Error("expected continue:false result")
	}
	if reason != "halted by before" {
		t.Errorf("reason = %q, want %q", reason, "halted by before")
	}
	if afterCalled {
		t.Error("after handler should not be called when before returns continue:false")
	}
}

func TestRegistry_Dispatch_ThreePhase_ContinueFalseFromUserHook(t *testing.T) {
	r := NewRegistry()

	r.Register(PreToolUseHandler{
		Matcher: "Bash",
		Order:   OrderBefore,
		Fn: func(input *PreToolUseInput) (*PreToolUseOutput, error) {
			return nil, nil // allow
		},
	})

	afterCalled := false
	r.Register(PreToolUseHandler{
		Matcher: "Bash",
		Order:   OrderAfter,
		Fn: func(input *PreToolUseInput) (*PreToolUseOutput, error) {
			afterCalled = true
			return nil, nil
		},
	})

	stopJSON, _ := json.Marshal(map[string]interface{}{
		"continue":   false,
		"stopReason": "halted by user hook",
	})
	userRunner := &mockCmdRunner{exitCode: 0, stdout: stopJSON}
	r.SetCommandRunner(userRunner)
	r.SetUserHooks(PreToolUse, []UserHook{{Command: "stop-hook", Matcher: "Bash"}})

	result := r.Dispatch(PreToolUse, preToolUseInput("Bash", "test"))

	stop, _ := extractContinueFalse(result.Stdout)
	if !stop {
		t.Error("expected continue:false result from user hook")
	}
	if afterCalled {
		t.Error("after handler should not be called when user hook returns continue:false")
	}
}

func TestRegistry_Dispatch_ThreePhase_AfterBlocks(t *testing.T) {
	r := NewRegistry()

	r.Register(PreToolUseHandler{
		Matcher: "Bash",
		Order:   OrderBefore,
		Fn: func(input *PreToolUseInput) (*PreToolUseOutput, error) {
			return nil, nil // allow
		},
	})

	r.Register(PreToolUseHandler{
		Matcher: "Bash",
		Order:   OrderAfter,
		Fn: func(input *PreToolUseInput) (*PreToolUseOutput, error) {
			return nil, &BlockError{Message: "blocked by after"}
		},
	})

	// User hooks allow
	userRunner := &mockCmdRunner{exitCode: 0, stdout: []byte(`{}`)}
	r.SetCommandRunner(userRunner)
	r.SetUserHooks(PreToolUse, []UserHook{{Command: "ok-hook", Matcher: "Bash"}})

	result := r.Dispatch(PreToolUse, preToolUseInput("Bash", "test"))

	if result.ExitCode != 2 {
		t.Errorf("ExitCode = %d, want 2 (blocked by after handler)", result.ExitCode)
	}
}

func TestRegistry_Dispatch_ThreePhase_DenyFromUserHookBeatsAllowFromGoHandlers(t *testing.T) {
	r := NewRegistry()

	// Before handler allows
	r.Register(PreToolUseHandler{
		Matcher: "Bash",
		Order:   OrderBefore,
		Fn: func(input *PreToolUseInput) (*PreToolUseOutput, error) {
			return &PreToolUseOutput{
				HookSpecificOutput: &PreToolUseSpecificOutput{PermissionDecision: "allow"},
			}, nil
		},
	})

	// After handler allows
	r.Register(PreToolUseHandler{
		Matcher: "Bash",
		Order:   OrderAfter,
		Fn: func(input *PreToolUseInput) (*PreToolUseOutput, error) {
			return &PreToolUseOutput{
				HookSpecificOutput: &PreToolUseSpecificOutput{PermissionDecision: "allow"},
			}, nil
		},
	})

	// User hook denies
	denyJSON, _ := json.Marshal(map[string]interface{}{
		"hookSpecificOutput": map[string]interface{}{
			"hookEventName":      "PreToolUse",
			"permissionDecision": "deny",
		},
	})
	userRunner := &mockCmdRunner{exitCode: 0, stdout: denyJSON}
	r.SetCommandRunner(userRunner)
	r.SetUserHooks(PreToolUse, []UserHook{{Command: "deny-hook", Matcher: "Bash"}})

	result := r.Dispatch(PreToolUse, preToolUseInput("Bash", "test"))

	decision := extractDecision(result.Stdout)
	if decision != "deny" {
		t.Errorf("decision = %q, want %q (deny from user hook should beat allow from Go handlers)", decision, "deny")
	}
}

func TestRegistry_Dispatch_ThreePhase_NoUserHooksStillWorks(t *testing.T) {
	r := NewRegistry()

	called := false
	r.Register(PreToolUseHandler{
		Matcher: "Bash",
		Order:   OrderBefore,
		Fn: func(input *PreToolUseInput) (*PreToolUseOutput, error) {
			called = true
			return nil, nil
		},
	})

	// No user hooks, no command runner
	result := r.Dispatch(PreToolUse, preToolUseInput("Bash", "test"))

	if result.ExitCode != 0 {
		t.Errorf("ExitCode = %d, want 0", result.ExitCode)
	}
	if !called {
		t.Error("handler should be called even without user hooks")
	}
}

func TestRegistry_BridgeHandler_NoMatchingHandler(t *testing.T) {
	r := NewRegistry()

	handler := r.BridgeHandler()

	req := bridge.HookRequest{
		Type:  "hook",
		Event: "PreToolUse",
		Input: json.RawMessage(`{"hook_event_name":"PreToolUse","tool_name":"Bash","tool_input":{}}`),
	}

	resp := handler(req)

	if resp.ExitCode != 0 {
		t.Errorf("expected exit code 0, got %d", resp.ExitCode)
	}
	if resp.Stdout != "" {
		t.Errorf("expected empty stdout, got %q", resp.Stdout)
	}
	if resp.Stderr != "" {
		t.Errorf("expected empty stderr, got %q", resp.Stderr)
	}
}
