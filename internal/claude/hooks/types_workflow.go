package hooks

// StopInput is sent when a session stops normally.
type StopInput struct {
	HookInputBase
	Reason string `json:"reason,omitempty"`
}

// StopOutput is the hook response for Stop events.
type StopOutput struct {
	HookOutputBase
}

// StopFailureInput is sent when a session stops due to an error.
type StopFailureInput struct {
	HookInputBase
	Error string `json:"error,omitempty"`
}

// StopFailureOutput is the hook response for StopFailure events.
type StopFailureOutput struct {
	HookOutputBase
}

// PreCompactInput is sent before context compaction occurs.
type PreCompactInput struct {
	HookInputBase
}

// PreCompactOutput is the hook response for PreCompact events.
type PreCompactOutput struct {
	HookOutputBase
}

// PostCompactInput is sent after context compaction completes.
type PostCompactInput struct {
	HookInputBase
}

// PostCompactOutput is the hook response for PostCompact events.
type PostCompactOutput struct {
	HookOutputBase
}
