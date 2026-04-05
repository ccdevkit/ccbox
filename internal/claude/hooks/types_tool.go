package hooks

import "encoding/json"

// PreToolUseInput is the stdin payload for PreToolUse hook events.
type PreToolUseInput struct {
	HookInputBase
	ToolName  string          `json:"tool_name"`
	ToolInput json.RawMessage `json:"tool_input"`
}

// PreToolUseSpecificOutput holds PreToolUse-specific output fields.
// HookEventName is always "PreToolUse" and is set automatically during serialization.
type PreToolUseSpecificOutput struct {
	HookEventName      string `json:"hookEventName"`
	PermissionDecision string `json:"permissionDecision,omitempty"`
}

// MarshalJSON ensures hookEventName is always "PreToolUse".
func (o PreToolUseSpecificOutput) MarshalJSON() ([]byte, error) {
	type alias PreToolUseSpecificOutput
	o.HookEventName = "PreToolUse"
	return json.Marshal(alias(o))
}

// PreToolUseOutput is the stdout response for PreToolUse hook events.
type PreToolUseOutput struct {
	HookOutputBase
	HookSpecificOutput *PreToolUseSpecificOutput `json:"hookSpecificOutput,omitempty"`
}

// PostToolUseInput is the stdin payload for PostToolUse hook events.
type PostToolUseInput struct {
	HookInputBase
	ToolName   string          `json:"tool_name"`
	ToolInput  json.RawMessage `json:"tool_input"`
	ToolResult json.RawMessage `json:"tool_result"`
}

// PostToolUseOutput is the stdout response for PostToolUse hook events.
type PostToolUseOutput struct {
	HookOutputBase
}

// PostToolUseFailureInput is the stdin payload for PostToolUseFailure hook events.
type PostToolUseFailureInput struct {
	HookInputBase
	ToolName  string          `json:"tool_name"`
	ToolInput json.RawMessage `json:"tool_input"`
	Error     string          `json:"error,omitempty"`
}

// PostToolUseFailureOutput is the stdout response for PostToolUseFailure hook events.
type PostToolUseFailureOutput struct {
	HookOutputBase
}

// PermissionRequestInput is the stdin payload for PermissionRequest hook events.
type PermissionRequestInput struct {
	HookInputBase
	ToolName  string          `json:"tool_name"`
	ToolInput json.RawMessage `json:"tool_input"`
}

// PermissionRequestOutput is the stdout response for PermissionRequest hook events.
type PermissionRequestOutput struct {
	HookOutputBase
}

// PermissionDeniedInput is the stdin payload for PermissionDenied hook events.
type PermissionDeniedInput struct {
	HookInputBase
	ToolName  string          `json:"tool_name"`
	ToolInput json.RawMessage `json:"tool_input"`
}

// PermissionDeniedOutput is the stdout response for PermissionDenied hook events.
type PermissionDeniedOutput struct {
	HookOutputBase
}
