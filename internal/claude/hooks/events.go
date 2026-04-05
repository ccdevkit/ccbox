package hooks

// HookEvent identifies a Claude Code hook event type.
type HookEvent string

const (
	// Lifecycle
	SessionStart HookEvent = "SessionStart"
	SessionEnd   HookEvent = "SessionEnd"

	// Instruction
	InstructionsLoaded HookEvent = "InstructionsLoaded"

	// Prompt
	UserPromptSubmit HookEvent = "UserPromptSubmit"

	// Tool
	PreToolUse         HookEvent = "PreToolUse"
	PostToolUse        HookEvent = "PostToolUse"
	PostToolUseFailure HookEvent = "PostToolUseFailure"
	PermissionRequest  HookEvent = "PermissionRequest"
	PermissionDenied   HookEvent = "PermissionDenied"

	// Agent/Task
	SubagentStart HookEvent = "SubagentStart"
	SubagentStop  HookEvent = "SubagentStop"
	TaskCreated   HookEvent = "TaskCreated"
	TaskCompleted HookEvent = "TaskCompleted"
	TeammateIdle  HookEvent = "TeammateIdle"

	// Workflow
	Stop        HookEvent = "Stop"
	StopFailure HookEvent = "StopFailure"

	// Compact
	PreCompact  HookEvent = "PreCompact"
	PostCompact HookEvent = "PostCompact"

	// File/Config
	FileChanged  HookEvent = "FileChanged"
	CwdChanged   HookEvent = "CwdChanged"
	ConfigChange HookEvent = "ConfigChange"

	// Worktree
	WorktreeCreate HookEvent = "WorktreeCreate"
	WorktreeRemove HookEvent = "WorktreeRemove"

	// MCP
	Elicitation       HookEvent = "Elicitation"
	ElicitationResult HookEvent = "ElicitationResult"

	// Other
	Notification HookEvent = "Notification"
)

// Order specifies whether a hook runs before or after the event.
type Order string

const (
	OrderBefore Order = "before"
	OrderAfter  Order = "after"
)
