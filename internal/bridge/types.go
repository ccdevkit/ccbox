package bridge

import "encoding/json"

// ExecRequest is sent from the container ccptproxy to the host TCP server
// to execute a command on the host.
type ExecRequest struct {
	Type    string `json:"type"`    // Always "exec"
	Command string `json:"command"` // Shell command to execute
	Cwd     string `json:"cwd"`    // Working directory for execution
}

// LogRequest is a fire-and-forget log message from container to host.
type LogRequest struct {
	Type    string `json:"type"`              // Always "log"
	Message string `json:"message"`           // Log message text
	Prefix  string `json:"prefix,omitempty"`  // Logger prefix (default: "container")
}

// HookRequest is sent from the container cchookproxy to the host TCP server
// to dispatch a hook event to registered handlers.
type HookRequest struct {
	Type  string          `json:"type"`  // Always "hook"
	Event string          `json:"event"` // Hook event name (e.g., "PreToolUse")
	Input json.RawMessage `json:"input"` // Raw hook input JSON
}

// HookResponse is sent from the host TCP server back to the container cchookproxy
// with the result of hook handler execution.
type HookResponse struct {
	ExitCode int    `json:"exit_code"` // 0 = success, 2 = block, other = error
	Stdout   string `json:"stdout"`    // JSON output (parsed by Claude Code when exit 0)
	Stderr   string `json:"stderr"`    // Error text (shown by Claude Code when exit 2)
}
