package hooks

// SubagentStartInput is the hook input when a subagent is spawned.
type SubagentStartInput struct {
	HookInputBase
	SubagentID   string `json:"subagent_id,omitempty"`
	SubagentType string `json:"subagent_type,omitempty"`
}

// SubagentStartOutput is the hook output for SubagentStart events.
type SubagentStartOutput struct {
	HookOutputBase
}

// SubagentStopInput is the hook input when a subagent terminates.
type SubagentStopInput struct {
	HookInputBase
	SubagentID   string `json:"subagent_id,omitempty"`
	SubagentType string `json:"subagent_type,omitempty"`
}

// SubagentStopOutput is the hook output for SubagentStop events.
type SubagentStopOutput struct {
	HookOutputBase
}

// TaskCreatedInput is the hook input when a task is created.
type TaskCreatedInput struct {
	HookInputBase
}

// TaskCreatedOutput is the hook output for TaskCreated events.
type TaskCreatedOutput struct {
	HookOutputBase
}

// TaskCompletedInput is the hook input when a task completes.
type TaskCompletedInput struct {
	HookInputBase
}

// TaskCompletedOutput is the hook output for TaskCompleted events.
type TaskCompletedOutput struct {
	HookOutputBase
}

// TeammateIdleInput is the hook input when a teammate becomes idle.
type TeammateIdleInput struct {
	HookInputBase
}

// TeammateIdleOutput is the hook output for TeammateIdle events.
type TeammateIdleOutput struct {
	HookOutputBase
}
