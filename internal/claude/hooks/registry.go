package hooks

import (
	"context"
	"encoding/json"
	"regexp"

	"github.com/ccdevkit/ccbox/internal/bridge"
	"github.com/ccdevkit/ccbox/internal/logger"
)

// MatcherGroup represents a matcher group in Claude Code's hook settings.
type MatcherGroup struct {
	Matcher string      `json:"matcher,omitempty"`
	Hooks   []HookEntry `json:"hooks"`
}

// HookEntry represents a single hook entry in the settings.
type HookEntry struct {
	Type    string `json:"type"`
	Command string `json:"command"`
}

// Registry stores hook handler registrations and dispatches events.
type Registry struct {
	handlers   map[HookEvent][]HookHandler
	userHooks  map[HookEvent][]UserHook
	runner     CommandRunner
	projectDir string
	log        *logger.Logger
}

// NewRegistry creates an empty hook registry.
func NewRegistry() *Registry {
	return &Registry{
		handlers:  make(map[HookEvent][]HookHandler),
		userHooks: make(map[HookEvent][]UserHook),
		runner:    ShellRunner{},
	}
}

// SetUserHooks stores captured user hooks for runtime dispatch.
func (r *Registry) SetUserHooks(event HookEvent, hooks []UserHook) {
	r.userHooks[event] = hooks
	for _, h := range hooks {
		r.debug("captured user hook: event=%s matcher=%q if=%q cmd=%q", event, h.Matcher, h.If, h.Command)
	}
}

// SetCommandRunner sets the command runner for user hook subprocess execution.
func (r *Registry) SetCommandRunner(runner CommandRunner) {
	r.runner = runner
}

// SetProjectDir sets the project directory for CLAUDE_PROJECT_DIR env var.
func (r *Registry) SetProjectDir(dir string) {
	r.projectDir = dir
}

// SetLogger sets the debug logger for hook dispatch tracing.
func (r *Registry) SetLogger(log *logger.Logger) {
	r.log = log
}

// debug logs a message if the logger is set.
func (r *Registry) debug(format string, args ...interface{}) {
	if r.log != nil {
		r.log.Debug("hooks", format, args...)
	}
}

// RegisteredEvents returns the set of event names that have Go handlers registered.
func (r *Registry) RegisteredEvents() map[string]bool {
	result := make(map[string]bool)
	for event := range r.handlers {
		result[string(event)] = true
	}
	return result
}

// Register adds a typed handler. The event name, matcher, and order
// are carried by the handler struct itself.
func (r *Registry) Register(handler HookHandler) {
	event := handler.EventName()
	r.handlers[event] = append(r.handlers[event], handler)
	r.debug("registered %s handler: event=%s matcher=%q order=%s", handler.HandlerOrder(), event, handler.MatcherPattern(), handler.HandlerOrder())
}

// HookEntries returns the settings.json hook configuration with one catch-all
// entry per event that has either Go handlers or user hooks.
// proxyCommand is the full path to the cchookproxy binary.
func (r *Registry) HookEntries(proxyCommand string) map[string][]MatcherGroup {
	// Collect all events that need proxy entries
	allEvents := make(map[string]bool)
	for event := range r.handlers {
		allEvents[string(event)] = true
	}
	for event := range r.userHooks {
		allEvents[string(event)] = true
	}

	result := make(map[string][]MatcherGroup)
	for event := range allEvents {
		result[event] = []MatcherGroup{
			{
				Matcher: "*",
				Hooks: []HookEntry{
					{Type: "command", Command: proxyCommand},
				},
			},
		}
	}
	return result
}

// BridgeHandler returns a bridge.HookHandler that dispatches hook requests
// through this registry.
func (r *Registry) BridgeHandler() bridge.HookHandler {
	return func(req bridge.HookRequest) bridge.HookResponse {
		r.debug("bridge hook request: event=%s", req.Event)
		result := r.Dispatch(HookEvent(req.Event), req.Input)
		r.debug("bridge hook response: event=%s exit=%d stdout=%d bytes stderr=%d bytes", req.Event, result.ExitCode, len(result.Stdout), len(result.Stderr))
		return bridge.HookResponse{
			ExitCode: result.ExitCode,
			Stdout:   string(result.Stdout),
			Stderr:   string(result.Stderr),
		}
	}
}

// matcherTarget extracts the field to match against from raw input JSON.
// For tool events, this is tool_name. For others, hook_event_name.
type matcherTarget struct {
	ToolName      string `json:"tool_name"`
	HookEventName string `json:"hook_event_name"`
}

// toolEvents is the set of events where the matcher applies to tool_name.
var toolEvents = map[HookEvent]bool{
	PreToolUse:         true,
	PostToolUse:        true,
	PostToolUseFailure: true,
	PermissionRequest:  true,
	PermissionDenied:   true,
}

// matchesHandler returns true if the handler's matcher pattern matches the input.
func matchesHandler(handler HookHandler, event HookEvent, target matcherTarget) bool {
	pattern := handler.MatcherPattern()
	if pattern == "" {
		return true
	}
	var subject string
	if toolEvents[event] {
		subject = target.ToolName
	} else {
		subject = target.HookEventName
	}
	matched, err := regexp.MatchString("^(?:"+pattern+")$", subject)
	if err != nil {
		return false
	}
	return matched
}

// decisionPriority maps permissionDecision values to their precedence.
// Higher values take priority: deny > defer > ask > allow.
var decisionPriority = map[string]int{
	"":      0,
	"allow": 1,
	"ask":   2,
	"defer": 3,
	"deny":  4,
}

// extractDecision parses permissionDecision from a handler result's stdout JSON.
func extractDecision(stdout []byte) string {
	var envelope struct {
		HookSpecificOutput *struct {
			PermissionDecision string `json:"permissionDecision"`
		} `json:"hookSpecificOutput"`
	}
	if json.Unmarshal(stdout, &envelope) == nil && envelope.HookSpecificOutput != nil {
		return envelope.HookSpecificOutput.PermissionDecision
	}
	return ""
}

// extractContinueFalse checks if a handler's stdout JSON has continue explicitly set to false.
// Returns (true, stopReason) if continue is false, (false, "") otherwise.
func extractContinueFalse(stdout []byte) (bool, string) {
	if len(stdout) == 0 {
		return false, ""
	}
	var check struct {
		Continue   *bool  `json:"continue,omitempty"`
		StopReason string `json:"stopReason,omitempty"`
	}
	if json.Unmarshal(stdout, &check) != nil {
		return false, ""
	}
	if check.Continue != nil && !*check.Continue {
		return true, check.StopReason
	}
	return false, ""
}

// buildStopResult creates a HandlerResult with exit 0 and continue:false in stdout.
func buildStopResult(reason string) *HandlerResult {
	boolFalse := false
	out := HookOutputBase{
		Continue:   &boolFalse,
		StopReason: reason,
	}
	data, _ := json.Marshal(out)
	return &HandlerResult{ExitCode: 0, Stdout: data}
}

// Dispatch finds all matching handlers for the given event and input,
// orchestrates a three-phase pipeline, and returns the aggregated result.
//
// Phase 1: Run "before" Go handlers sequentially
// Phase 2: Run user hooks (parallel subprocesses)
// Phase 3: Run "after" Go handlers sequentially
//
// Short-circuits on block (exit 2) or continue:false.
// Aggregates results across phases using decision precedence:
// deny > defer > ask > allow.
func (r *Registry) Dispatch(event HookEvent, input json.RawMessage) *HandlerResult {
	allHandlers := r.handlers[event]

	// Partition Go handlers by order
	var beforeHandlers, afterHandlers []HookHandler
	for _, h := range allHandlers {
		if h.HandlerOrder() == OrderBefore {
			beforeHandlers = append(beforeHandlers, h)
		} else {
			afterHandlers = append(afterHandlers, h)
		}
	}

	var target matcherTarget
	_ = json.Unmarshal(input, &target)

	userHooks := r.userHooks[event]
	r.debug("dispatch %s: before=%d user=%d after=%d target=%q", event, len(beforeHandlers), len(userHooks), len(afterHandlers), target.ToolName+target.HookEventName)

	// Collect results from all phases for final aggregation
	var allResults []*HandlerResult

	// Phase 1: Run "before" Go handlers
	r.debug("dispatch %s: phase 1 — running %d before handlers", event, len(beforeHandlers))
	phase1Result := runGoHandlers(beforeHandlers, event, target, input)
	if phase1Result != nil {
		allResults = append(allResults, phase1Result)
		if isTerminalResult(phase1Result) {
			r.debug("dispatch %s: phase 1 short-circuit exit=%d", event, phase1Result.ExitCode)
			return phase1Result
		}
	}

	// Phase 2: Run user hooks
	if len(userHooks) > 0 && r.runner != nil {
		r.debug("dispatch %s: phase 2 — executing %d user hooks", event, len(userHooks))
		env := r.buildUserHookEnv()
		phase2Result := ExecuteUserHooks(context.Background(), r.runner, r.log, userHooks, event, input, env, r.projectDir)
		if phase2Result != nil && (phase2Result.ExitCode != 0 || len(phase2Result.Stdout) > 0) {
			allResults = append(allResults, phase2Result)
			if isTerminalResult(phase2Result) {
				r.debug("dispatch %s: phase 2 short-circuit exit=%d", event, phase2Result.ExitCode)
				return phase2Result
			}
		}
	}

	// Phase 3: Run "after" Go handlers
	r.debug("dispatch %s: phase 3 — running %d after handlers", event, len(afterHandlers))
	phase3Result := runGoHandlers(afterHandlers, event, target, input)
	if phase3Result != nil {
		allResults = append(allResults, phase3Result)
		if isTerminalResult(phase3Result) {
			r.debug("dispatch %s: phase 3 short-circuit exit=%d", event, phase3Result.ExitCode)
			return phase3Result
		}
	}

	// Aggregate across all phases
	final := aggregatePhaseResults(allResults)
	r.debug("dispatch %s: final exit=%d decision=%q", event, final.ExitCode, extractDecision(final.Stdout))
	return final
}

// buildUserHookEnv builds the environment variables for user hook subprocesses.
func (r *Registry) buildUserHookEnv() []string {
	var env []string
	if r.projectDir != "" {
		env = append(env, "CLAUDE_PROJECT_DIR="+r.projectDir)
	}
	return env
}

// runGoHandlers runs a slice of Go handlers sequentially and returns the
// aggregated result. Returns nil if no handlers match.
func runGoHandlers(handlers []HookHandler, event HookEvent, target matcherTarget, input json.RawMessage) *HandlerResult {
	var (
		best     *HandlerResult
		bestPri  int
		block    *HandlerResult
		errRes   *HandlerResult
		stopRes  *HandlerResult
	)

	for _, h := range handlers {
		if !matchesHandler(h, event, target) {
			continue
		}
		result, err := h.invoke(input)
		if err != nil {
			if be, ok := err.(*BlockError); ok {
				if block == nil {
					block = &HandlerResult{ExitCode: 2, Stderr: []byte(be.Message)}
				}
			} else if errRes == nil {
				errRes = &HandlerResult{ExitCode: 1, Stderr: []byte(err.Error())}
			}
			continue
		}
		if stopRes == nil {
			if stop, reason := extractContinueFalse(result.Stdout); stop {
				stopRes = buildStopResult(reason)
			}
		}
		pri := decisionPriority[extractDecision(result.Stdout)]
		if best == nil || pri > bestPri {
			best = result
			bestPri = pri
		}
	}

	if stopRes != nil {
		return stopRes
	}
	if block != nil {
		return block
	}
	if errRes != nil {
		return errRes
	}
	return best
}

// isTerminalResult returns true if the result should short-circuit dispatch.
func isTerminalResult(r *HandlerResult) bool {
	if r == nil {
		return false
	}
	if r.ExitCode == 2 {
		return true
	}
	if stop, _ := extractContinueFalse(r.Stdout); stop {
		return true
	}
	return false
}

// aggregatePhaseResults combines results from all three phases using
// Claude Code's precedence rules.
func aggregatePhaseResults(results []*HandlerResult) *HandlerResult {
	var (
		stopRes  *HandlerResult
		blockRes *HandlerResult
		errRes   *HandlerResult
		bestRes  *HandlerResult
		bestPri  int
	)

	for _, r := range results {
		if r == nil {
			continue
		}
		if stopRes == nil {
			if stop, reason := extractContinueFalse(r.Stdout); stop {
				stopRes = buildStopResult(reason)
			}
		}
		switch r.ExitCode {
		case 2:
			if blockRes == nil {
				blockRes = r
			}
		case 0:
			pri := decisionPriority[extractDecision(r.Stdout)]
			if bestRes == nil || pri > bestPri {
				bestRes = r
				bestPri = pri
			}
		default:
			if errRes == nil {
				errRes = r
			}
		}
	}

	if stopRes != nil {
		return stopRes
	}
	if blockRes != nil {
		return blockRes
	}
	if errRes != nil {
		return errRes
	}
	if bestRes != nil {
		return bestRes
	}
	return &HandlerResult{ExitCode: 0}
}
