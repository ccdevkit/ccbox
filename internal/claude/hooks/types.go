package hooks

// HookInputBase contains common fields present in every hook event's stdin JSON.
type HookInputBase struct {
	SessionID      string `json:"session_id"`
	TranscriptPath string `json:"transcript_path"`
	CWD            string `json:"cwd"`
	PermissionMode string `json:"permission_mode"`
	HookEventName  string `json:"hook_event_name"`
	AgentID        string `json:"agent_id,omitempty"`
	AgentType      string `json:"agent_type,omitempty"`
}

// HookOutputBase contains common fields for hook stdout JSON responses.
type HookOutputBase struct {
	Continue       *bool  `json:"continue,omitempty"`
	StopReason     string `json:"stopReason,omitempty"`
	SuppressOutput bool   `json:"suppressOutput,omitempty"`
	SystemMessage  string `json:"systemMessage,omitempty"`
}

// HandlerResult is the wire-level response from a hook handler.
// Callers never construct this directly — it's produced by typed output's toResult() method.
type HandlerResult struct {
	ExitCode int
	Stdout   []byte
	Stderr   []byte
}

// BlockError is returned by a handler function to signal exit code 2 (block).
// When Dispatch encounters a BlockError from any handler, that handler's
// decision takes highest precedence over allow results.
type BlockError struct {
	Message string
}

func (e *BlockError) Error() string { return e.Message }
