package hooks

import "encoding/json"

// FileChangedInput is sent when a file is modified on disk.
type FileChangedInput struct {
	HookInputBase
	FilePath string `json:"file_path,omitempty"`
}

// FileChangedOutput is the hook response for FileChanged events.
type FileChangedOutput struct{ HookOutputBase }

// CwdChangedInput is sent when the working directory changes.
type CwdChangedInput struct {
	HookInputBase
	OldCwd string `json:"old_cwd,omitempty"`
	NewCwd string `json:"new_cwd,omitempty"`
}

// CwdChangedOutput is the hook response for CwdChanged events.
type CwdChangedOutput struct{ HookOutputBase }

// ConfigChangeInput is sent when configuration changes.
type ConfigChangeInput struct {
	HookInputBase
	Config json.RawMessage `json:"config,omitempty"`
}

// ConfigChangeOutput is the hook response for ConfigChange events.
type ConfigChangeOutput struct{ HookOutputBase }

// WorktreeCreateInput is sent when a git worktree is created.
type WorktreeCreateInput struct {
	HookInputBase
	WorktreePath string `json:"worktree_path,omitempty"`
}

// WorktreeCreateOutput is the hook response for WorktreeCreate events.
type WorktreeCreateOutput struct{ HookOutputBase }

// WorktreeRemoveInput is sent when a git worktree is removed.
type WorktreeRemoveInput struct {
	HookInputBase
	WorktreePath string `json:"worktree_path,omitempty"`
}

// WorktreeRemoveOutput is the hook response for WorktreeRemove events.
type WorktreeRemoveOutput struct{ HookOutputBase }

// ElicitationInput is sent when an elicitation prompt is presented.
type ElicitationInput struct {
	HookInputBase
	Message json.RawMessage `json:"message,omitempty"`
}

// ElicitationOutput is the hook response for Elicitation events.
type ElicitationOutput struct{ HookOutputBase }

// ElicitationResultInput is sent when an elicitation result is received.
type ElicitationResultInput struct {
	HookInputBase
	Result json.RawMessage `json:"result,omitempty"`
}

// ElicitationResultOutput is the hook response for ElicitationResult events.
type ElicitationResultOutput struct{ HookOutputBase }

// NotificationInput is sent when a notification is emitted.
type NotificationInput struct {
	HookInputBase
	Message string `json:"message,omitempty"`
}

// NotificationOutput is the hook response for Notification events.
type NotificationOutput struct{ HookOutputBase }
