package hooks

import (
	"encoding/json"
	"fmt"
)

// HookHandler is the interface that all per-event handler structs implement.
type HookHandler interface {
	EventName() HookEvent
	MatcherPattern() string
	HandlerOrder() Order
	invoke(input json.RawMessage) (*HandlerResult, error)
}

// toResult converts a nil-or-populated HookOutputBase into a HandlerResult.
func (o *HookOutputBase) toResult() (*HandlerResult, error) {
	if o == nil {
		return &HandlerResult{ExitCode: 0}, nil
	}
	stdout, err := json.Marshal(o)
	if err != nil {
		return nil, err
	}
	return &HandlerResult{ExitCode: 0, Stdout: stdout}, nil
}

// --- PreToolUse (special: has HookSpecificOutput) ---

func (o *PreToolUseOutput) toResult() (*HandlerResult, error) {
	if o == nil {
		return &HandlerResult{ExitCode: 0}, nil
	}
	stdout, err := json.Marshal(o)
	if err != nil {
		return nil, err
	}
	return &HandlerResult{ExitCode: 0, Stdout: stdout}, nil
}

// SessionStartHandler handles SessionStart events.
type SessionStartHandler struct {
	Matcher string
	Order   Order
	Fn      func(*SessionStartInput) (*SessionStartOutput, error)
}

func (h SessionStartHandler) EventName() HookEvent  { return SessionStart }
func (h SessionStartHandler) MatcherPattern() string { return h.Matcher }
func (h SessionStartHandler) HandlerOrder() Order    { return h.Order }
func (h SessionStartHandler) invoke(raw json.RawMessage) (*HandlerResult, error) {
	var input SessionStartInput
	if err := json.Unmarshal(raw, &input); err != nil {
		return nil, fmt.Errorf("unmarshal SessionStart input: %w", err)
	}
	output, err := h.Fn(&input)
	if err != nil {
		return nil, err
	}
	if output == nil {
		return &HandlerResult{ExitCode: 0}, nil
	}
	return output.toResult()
}

// SessionEndHandler handles SessionEnd events.
type SessionEndHandler struct {
	Matcher string
	Order   Order
	Fn      func(*SessionEndInput) (*SessionEndOutput, error)
}

func (h SessionEndHandler) EventName() HookEvent  { return SessionEnd }
func (h SessionEndHandler) MatcherPattern() string { return h.Matcher }
func (h SessionEndHandler) HandlerOrder() Order    { return h.Order }
func (h SessionEndHandler) invoke(raw json.RawMessage) (*HandlerResult, error) {
	var input SessionEndInput
	if err := json.Unmarshal(raw, &input); err != nil {
		return nil, fmt.Errorf("unmarshal SessionEnd input: %w", err)
	}
	output, err := h.Fn(&input)
	if err != nil {
		return nil, err
	}
	if output == nil {
		return &HandlerResult{ExitCode: 0}, nil
	}
	return output.toResult()
}

// InstructionsLoadedHandler handles InstructionsLoaded events.
type InstructionsLoadedHandler struct {
	Matcher string
	Order   Order
	Fn      func(*InstructionsLoadedInput) (*InstructionsLoadedOutput, error)
}

func (h InstructionsLoadedHandler) EventName() HookEvent  { return InstructionsLoaded }
func (h InstructionsLoadedHandler) MatcherPattern() string { return h.Matcher }
func (h InstructionsLoadedHandler) HandlerOrder() Order    { return h.Order }
func (h InstructionsLoadedHandler) invoke(raw json.RawMessage) (*HandlerResult, error) {
	var input InstructionsLoadedInput
	if err := json.Unmarshal(raw, &input); err != nil {
		return nil, fmt.Errorf("unmarshal InstructionsLoaded input: %w", err)
	}
	output, err := h.Fn(&input)
	if err != nil {
		return nil, err
	}
	if output == nil {
		return &HandlerResult{ExitCode: 0}, nil
	}
	return output.toResult()
}

// UserPromptSubmitHandler handles UserPromptSubmit events.
type UserPromptSubmitHandler struct {
	Matcher string
	Order   Order
	Fn      func(*UserPromptSubmitInput) (*UserPromptSubmitOutput, error)
}

func (h UserPromptSubmitHandler) EventName() HookEvent  { return UserPromptSubmit }
func (h UserPromptSubmitHandler) MatcherPattern() string { return h.Matcher }
func (h UserPromptSubmitHandler) HandlerOrder() Order    { return h.Order }
func (h UserPromptSubmitHandler) invoke(raw json.RawMessage) (*HandlerResult, error) {
	var input UserPromptSubmitInput
	if err := json.Unmarshal(raw, &input); err != nil {
		return nil, fmt.Errorf("unmarshal UserPromptSubmit input: %w", err)
	}
	output, err := h.Fn(&input)
	if err != nil {
		return nil, err
	}
	if output == nil {
		return &HandlerResult{ExitCode: 0}, nil
	}
	return output.toResult()
}

// PreToolUseHandler handles PreToolUse events.
type PreToolUseHandler struct {
	Matcher string
	Order   Order
	Fn      func(*PreToolUseInput) (*PreToolUseOutput, error)
}

func (h PreToolUseHandler) EventName() HookEvent  { return PreToolUse }
func (h PreToolUseHandler) MatcherPattern() string { return h.Matcher }
func (h PreToolUseHandler) HandlerOrder() Order    { return h.Order }
func (h PreToolUseHandler) invoke(raw json.RawMessage) (*HandlerResult, error) {
	var input PreToolUseInput
	if err := json.Unmarshal(raw, &input); err != nil {
		return nil, fmt.Errorf("unmarshal PreToolUse input: %w", err)
	}
	output, err := h.Fn(&input)
	if err != nil {
		return nil, err
	}
	if output == nil {
		return &HandlerResult{ExitCode: 0}, nil
	}
	return output.toResult()
}

// PostToolUseHandler handles PostToolUse events.
type PostToolUseHandler struct {
	Matcher string
	Order   Order
	Fn      func(*PostToolUseInput) (*PostToolUseOutput, error)
}

func (h PostToolUseHandler) EventName() HookEvent  { return PostToolUse }
func (h PostToolUseHandler) MatcherPattern() string { return h.Matcher }
func (h PostToolUseHandler) HandlerOrder() Order    { return h.Order }
func (h PostToolUseHandler) invoke(raw json.RawMessage) (*HandlerResult, error) {
	var input PostToolUseInput
	if err := json.Unmarshal(raw, &input); err != nil {
		return nil, fmt.Errorf("unmarshal PostToolUse input: %w", err)
	}
	output, err := h.Fn(&input)
	if err != nil {
		return nil, err
	}
	if output == nil {
		return &HandlerResult{ExitCode: 0}, nil
	}
	return output.toResult()
}

// PostToolUseFailureHandler handles PostToolUseFailure events.
type PostToolUseFailureHandler struct {
	Matcher string
	Order   Order
	Fn      func(*PostToolUseFailureInput) (*PostToolUseFailureOutput, error)
}

func (h PostToolUseFailureHandler) EventName() HookEvent  { return PostToolUseFailure }
func (h PostToolUseFailureHandler) MatcherPattern() string { return h.Matcher }
func (h PostToolUseFailureHandler) HandlerOrder() Order    { return h.Order }
func (h PostToolUseFailureHandler) invoke(raw json.RawMessage) (*HandlerResult, error) {
	var input PostToolUseFailureInput
	if err := json.Unmarshal(raw, &input); err != nil {
		return nil, fmt.Errorf("unmarshal PostToolUseFailure input: %w", err)
	}
	output, err := h.Fn(&input)
	if err != nil {
		return nil, err
	}
	if output == nil {
		return &HandlerResult{ExitCode: 0}, nil
	}
	return output.toResult()
}

// PermissionRequestHandler handles PermissionRequest events.
type PermissionRequestHandler struct {
	Matcher string
	Order   Order
	Fn      func(*PermissionRequestInput) (*PermissionRequestOutput, error)
}

func (h PermissionRequestHandler) EventName() HookEvent  { return PermissionRequest }
func (h PermissionRequestHandler) MatcherPattern() string { return h.Matcher }
func (h PermissionRequestHandler) HandlerOrder() Order    { return h.Order }
func (h PermissionRequestHandler) invoke(raw json.RawMessage) (*HandlerResult, error) {
	var input PermissionRequestInput
	if err := json.Unmarshal(raw, &input); err != nil {
		return nil, fmt.Errorf("unmarshal PermissionRequest input: %w", err)
	}
	output, err := h.Fn(&input)
	if err != nil {
		return nil, err
	}
	if output == nil {
		return &HandlerResult{ExitCode: 0}, nil
	}
	return output.toResult()
}

// PermissionDeniedHandler handles PermissionDenied events.
type PermissionDeniedHandler struct {
	Matcher string
	Order   Order
	Fn      func(*PermissionDeniedInput) (*PermissionDeniedOutput, error)
}

func (h PermissionDeniedHandler) EventName() HookEvent  { return PermissionDenied }
func (h PermissionDeniedHandler) MatcherPattern() string { return h.Matcher }
func (h PermissionDeniedHandler) HandlerOrder() Order    { return h.Order }
func (h PermissionDeniedHandler) invoke(raw json.RawMessage) (*HandlerResult, error) {
	var input PermissionDeniedInput
	if err := json.Unmarshal(raw, &input); err != nil {
		return nil, fmt.Errorf("unmarshal PermissionDenied input: %w", err)
	}
	output, err := h.Fn(&input)
	if err != nil {
		return nil, err
	}
	if output == nil {
		return &HandlerResult{ExitCode: 0}, nil
	}
	return output.toResult()
}

// SubagentStartHandler handles SubagentStart events.
type SubagentStartHandler struct {
	Matcher string
	Order   Order
	Fn      func(*SubagentStartInput) (*SubagentStartOutput, error)
}

func (h SubagentStartHandler) EventName() HookEvent  { return SubagentStart }
func (h SubagentStartHandler) MatcherPattern() string { return h.Matcher }
func (h SubagentStartHandler) HandlerOrder() Order    { return h.Order }
func (h SubagentStartHandler) invoke(raw json.RawMessage) (*HandlerResult, error) {
	var input SubagentStartInput
	if err := json.Unmarshal(raw, &input); err != nil {
		return nil, fmt.Errorf("unmarshal SubagentStart input: %w", err)
	}
	output, err := h.Fn(&input)
	if err != nil {
		return nil, err
	}
	if output == nil {
		return &HandlerResult{ExitCode: 0}, nil
	}
	return output.toResult()
}

// SubagentStopHandler handles SubagentStop events.
type SubagentStopHandler struct {
	Matcher string
	Order   Order
	Fn      func(*SubagentStopInput) (*SubagentStopOutput, error)
}

func (h SubagentStopHandler) EventName() HookEvent  { return SubagentStop }
func (h SubagentStopHandler) MatcherPattern() string { return h.Matcher }
func (h SubagentStopHandler) HandlerOrder() Order    { return h.Order }
func (h SubagentStopHandler) invoke(raw json.RawMessage) (*HandlerResult, error) {
	var input SubagentStopInput
	if err := json.Unmarshal(raw, &input); err != nil {
		return nil, fmt.Errorf("unmarshal SubagentStop input: %w", err)
	}
	output, err := h.Fn(&input)
	if err != nil {
		return nil, err
	}
	if output == nil {
		return &HandlerResult{ExitCode: 0}, nil
	}
	return output.toResult()
}

// TaskCreatedHandler handles TaskCreated events.
type TaskCreatedHandler struct {
	Matcher string
	Order   Order
	Fn      func(*TaskCreatedInput) (*TaskCreatedOutput, error)
}

func (h TaskCreatedHandler) EventName() HookEvent  { return TaskCreated }
func (h TaskCreatedHandler) MatcherPattern() string { return h.Matcher }
func (h TaskCreatedHandler) HandlerOrder() Order    { return h.Order }
func (h TaskCreatedHandler) invoke(raw json.RawMessage) (*HandlerResult, error) {
	var input TaskCreatedInput
	if err := json.Unmarshal(raw, &input); err != nil {
		return nil, fmt.Errorf("unmarshal TaskCreated input: %w", err)
	}
	output, err := h.Fn(&input)
	if err != nil {
		return nil, err
	}
	if output == nil {
		return &HandlerResult{ExitCode: 0}, nil
	}
	return output.toResult()
}

// TaskCompletedHandler handles TaskCompleted events.
type TaskCompletedHandler struct {
	Matcher string
	Order   Order
	Fn      func(*TaskCompletedInput) (*TaskCompletedOutput, error)
}

func (h TaskCompletedHandler) EventName() HookEvent  { return TaskCompleted }
func (h TaskCompletedHandler) MatcherPattern() string { return h.Matcher }
func (h TaskCompletedHandler) HandlerOrder() Order    { return h.Order }
func (h TaskCompletedHandler) invoke(raw json.RawMessage) (*HandlerResult, error) {
	var input TaskCompletedInput
	if err := json.Unmarshal(raw, &input); err != nil {
		return nil, fmt.Errorf("unmarshal TaskCompleted input: %w", err)
	}
	output, err := h.Fn(&input)
	if err != nil {
		return nil, err
	}
	if output == nil {
		return &HandlerResult{ExitCode: 0}, nil
	}
	return output.toResult()
}

// TeammateIdleHandler handles TeammateIdle events.
type TeammateIdleHandler struct {
	Matcher string
	Order   Order
	Fn      func(*TeammateIdleInput) (*TeammateIdleOutput, error)
}

func (h TeammateIdleHandler) EventName() HookEvent  { return TeammateIdle }
func (h TeammateIdleHandler) MatcherPattern() string { return h.Matcher }
func (h TeammateIdleHandler) HandlerOrder() Order    { return h.Order }
func (h TeammateIdleHandler) invoke(raw json.RawMessage) (*HandlerResult, error) {
	var input TeammateIdleInput
	if err := json.Unmarshal(raw, &input); err != nil {
		return nil, fmt.Errorf("unmarshal TeammateIdle input: %w", err)
	}
	output, err := h.Fn(&input)
	if err != nil {
		return nil, err
	}
	if output == nil {
		return &HandlerResult{ExitCode: 0}, nil
	}
	return output.toResult()
}

// StopHandler handles Stop events.
type StopHandler struct {
	Matcher string
	Order   Order
	Fn      func(*StopInput) (*StopOutput, error)
}

func (h StopHandler) EventName() HookEvent  { return Stop }
func (h StopHandler) MatcherPattern() string { return h.Matcher }
func (h StopHandler) HandlerOrder() Order    { return h.Order }
func (h StopHandler) invoke(raw json.RawMessage) (*HandlerResult, error) {
	var input StopInput
	if err := json.Unmarshal(raw, &input); err != nil {
		return nil, fmt.Errorf("unmarshal Stop input: %w", err)
	}
	output, err := h.Fn(&input)
	if err != nil {
		return nil, err
	}
	if output == nil {
		return &HandlerResult{ExitCode: 0}, nil
	}
	return output.toResult()
}

// StopFailureHandler handles StopFailure events.
type StopFailureHandler struct {
	Matcher string
	Order   Order
	Fn      func(*StopFailureInput) (*StopFailureOutput, error)
}

func (h StopFailureHandler) EventName() HookEvent  { return StopFailure }
func (h StopFailureHandler) MatcherPattern() string { return h.Matcher }
func (h StopFailureHandler) HandlerOrder() Order    { return h.Order }
func (h StopFailureHandler) invoke(raw json.RawMessage) (*HandlerResult, error) {
	var input StopFailureInput
	if err := json.Unmarshal(raw, &input); err != nil {
		return nil, fmt.Errorf("unmarshal StopFailure input: %w", err)
	}
	output, err := h.Fn(&input)
	if err != nil {
		return nil, err
	}
	if output == nil {
		return &HandlerResult{ExitCode: 0}, nil
	}
	return output.toResult()
}

// PreCompactHandler handles PreCompact events.
type PreCompactHandler struct {
	Matcher string
	Order   Order
	Fn      func(*PreCompactInput) (*PreCompactOutput, error)
}

func (h PreCompactHandler) EventName() HookEvent  { return PreCompact }
func (h PreCompactHandler) MatcherPattern() string { return h.Matcher }
func (h PreCompactHandler) HandlerOrder() Order    { return h.Order }
func (h PreCompactHandler) invoke(raw json.RawMessage) (*HandlerResult, error) {
	var input PreCompactInput
	if err := json.Unmarshal(raw, &input); err != nil {
		return nil, fmt.Errorf("unmarshal PreCompact input: %w", err)
	}
	output, err := h.Fn(&input)
	if err != nil {
		return nil, err
	}
	if output == nil {
		return &HandlerResult{ExitCode: 0}, nil
	}
	return output.toResult()
}

// PostCompactHandler handles PostCompact events.
type PostCompactHandler struct {
	Matcher string
	Order   Order
	Fn      func(*PostCompactInput) (*PostCompactOutput, error)
}

func (h PostCompactHandler) EventName() HookEvent  { return PostCompact }
func (h PostCompactHandler) MatcherPattern() string { return h.Matcher }
func (h PostCompactHandler) HandlerOrder() Order    { return h.Order }
func (h PostCompactHandler) invoke(raw json.RawMessage) (*HandlerResult, error) {
	var input PostCompactInput
	if err := json.Unmarshal(raw, &input); err != nil {
		return nil, fmt.Errorf("unmarshal PostCompact input: %w", err)
	}
	output, err := h.Fn(&input)
	if err != nil {
		return nil, err
	}
	if output == nil {
		return &HandlerResult{ExitCode: 0}, nil
	}
	return output.toResult()
}

// FileChangedHandler handles FileChanged events.
type FileChangedHandler struct {
	Matcher string
	Order   Order
	Fn      func(*FileChangedInput) (*FileChangedOutput, error)
}

func (h FileChangedHandler) EventName() HookEvent  { return FileChanged }
func (h FileChangedHandler) MatcherPattern() string { return h.Matcher }
func (h FileChangedHandler) HandlerOrder() Order    { return h.Order }
func (h FileChangedHandler) invoke(raw json.RawMessage) (*HandlerResult, error) {
	var input FileChangedInput
	if err := json.Unmarshal(raw, &input); err != nil {
		return nil, fmt.Errorf("unmarshal FileChanged input: %w", err)
	}
	output, err := h.Fn(&input)
	if err != nil {
		return nil, err
	}
	if output == nil {
		return &HandlerResult{ExitCode: 0}, nil
	}
	return output.toResult()
}

// CwdChangedHandler handles CwdChanged events.
type CwdChangedHandler struct {
	Matcher string
	Order   Order
	Fn      func(*CwdChangedInput) (*CwdChangedOutput, error)
}

func (h CwdChangedHandler) EventName() HookEvent  { return CwdChanged }
func (h CwdChangedHandler) MatcherPattern() string { return h.Matcher }
func (h CwdChangedHandler) HandlerOrder() Order    { return h.Order }
func (h CwdChangedHandler) invoke(raw json.RawMessage) (*HandlerResult, error) {
	var input CwdChangedInput
	if err := json.Unmarshal(raw, &input); err != nil {
		return nil, fmt.Errorf("unmarshal CwdChanged input: %w", err)
	}
	output, err := h.Fn(&input)
	if err != nil {
		return nil, err
	}
	if output == nil {
		return &HandlerResult{ExitCode: 0}, nil
	}
	return output.toResult()
}

// ConfigChangeHandler handles ConfigChange events.
type ConfigChangeHandler struct {
	Matcher string
	Order   Order
	Fn      func(*ConfigChangeInput) (*ConfigChangeOutput, error)
}

func (h ConfigChangeHandler) EventName() HookEvent  { return ConfigChange }
func (h ConfigChangeHandler) MatcherPattern() string { return h.Matcher }
func (h ConfigChangeHandler) HandlerOrder() Order    { return h.Order }
func (h ConfigChangeHandler) invoke(raw json.RawMessage) (*HandlerResult, error) {
	var input ConfigChangeInput
	if err := json.Unmarshal(raw, &input); err != nil {
		return nil, fmt.Errorf("unmarshal ConfigChange input: %w", err)
	}
	output, err := h.Fn(&input)
	if err != nil {
		return nil, err
	}
	if output == nil {
		return &HandlerResult{ExitCode: 0}, nil
	}
	return output.toResult()
}

// WorktreeCreateHandler handles WorktreeCreate events.
type WorktreeCreateHandler struct {
	Matcher string
	Order   Order
	Fn      func(*WorktreeCreateInput) (*WorktreeCreateOutput, error)
}

func (h WorktreeCreateHandler) EventName() HookEvent  { return WorktreeCreate }
func (h WorktreeCreateHandler) MatcherPattern() string { return h.Matcher }
func (h WorktreeCreateHandler) HandlerOrder() Order    { return h.Order }
func (h WorktreeCreateHandler) invoke(raw json.RawMessage) (*HandlerResult, error) {
	var input WorktreeCreateInput
	if err := json.Unmarshal(raw, &input); err != nil {
		return nil, fmt.Errorf("unmarshal WorktreeCreate input: %w", err)
	}
	output, err := h.Fn(&input)
	if err != nil {
		return nil, err
	}
	if output == nil {
		return &HandlerResult{ExitCode: 0}, nil
	}
	return output.toResult()
}

// WorktreeRemoveHandler handles WorktreeRemove events.
type WorktreeRemoveHandler struct {
	Matcher string
	Order   Order
	Fn      func(*WorktreeRemoveInput) (*WorktreeRemoveOutput, error)
}

func (h WorktreeRemoveHandler) EventName() HookEvent  { return WorktreeRemove }
func (h WorktreeRemoveHandler) MatcherPattern() string { return h.Matcher }
func (h WorktreeRemoveHandler) HandlerOrder() Order    { return h.Order }
func (h WorktreeRemoveHandler) invoke(raw json.RawMessage) (*HandlerResult, error) {
	var input WorktreeRemoveInput
	if err := json.Unmarshal(raw, &input); err != nil {
		return nil, fmt.Errorf("unmarshal WorktreeRemove input: %w", err)
	}
	output, err := h.Fn(&input)
	if err != nil {
		return nil, err
	}
	if output == nil {
		return &HandlerResult{ExitCode: 0}, nil
	}
	return output.toResult()
}

// ElicitationHandler handles Elicitation events.
type ElicitationHandler struct {
	Matcher string
	Order   Order
	Fn      func(*ElicitationInput) (*ElicitationOutput, error)
}

func (h ElicitationHandler) EventName() HookEvent  { return Elicitation }
func (h ElicitationHandler) MatcherPattern() string { return h.Matcher }
func (h ElicitationHandler) HandlerOrder() Order    { return h.Order }
func (h ElicitationHandler) invoke(raw json.RawMessage) (*HandlerResult, error) {
	var input ElicitationInput
	if err := json.Unmarshal(raw, &input); err != nil {
		return nil, fmt.Errorf("unmarshal Elicitation input: %w", err)
	}
	output, err := h.Fn(&input)
	if err != nil {
		return nil, err
	}
	if output == nil {
		return &HandlerResult{ExitCode: 0}, nil
	}
	return output.toResult()
}

// ElicitationResultHandler handles ElicitationResult events.
type ElicitationResultHandler struct {
	Matcher string
	Order   Order
	Fn      func(*ElicitationResultInput) (*ElicitationResultOutput, error)
}

func (h ElicitationResultHandler) EventName() HookEvent  { return ElicitationResult }
func (h ElicitationResultHandler) MatcherPattern() string { return h.Matcher }
func (h ElicitationResultHandler) HandlerOrder() Order    { return h.Order }
func (h ElicitationResultHandler) invoke(raw json.RawMessage) (*HandlerResult, error) {
	var input ElicitationResultInput
	if err := json.Unmarshal(raw, &input); err != nil {
		return nil, fmt.Errorf("unmarshal ElicitationResult input: %w", err)
	}
	output, err := h.Fn(&input)
	if err != nil {
		return nil, err
	}
	if output == nil {
		return &HandlerResult{ExitCode: 0}, nil
	}
	return output.toResult()
}

// NotificationHandler handles Notification events.
type NotificationHandler struct {
	Matcher string
	Order   Order
	Fn      func(*NotificationInput) (*NotificationOutput, error)
}

func (h NotificationHandler) EventName() HookEvent  { return Notification }
func (h NotificationHandler) MatcherPattern() string { return h.Matcher }
func (h NotificationHandler) HandlerOrder() Order    { return h.Order }
func (h NotificationHandler) invoke(raw json.RawMessage) (*HandlerResult, error) {
	var input NotificationInput
	if err := json.Unmarshal(raw, &input); err != nil {
		return nil, fmt.Errorf("unmarshal Notification input: %w", err)
	}
	output, err := h.Fn(&input)
	if err != nil {
		return nil, err
	}
	if output == nil {
		return &HandlerResult{ExitCode: 0}, nil
	}
	return output.toResult()
}
