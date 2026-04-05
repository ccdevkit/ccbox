package hooks

// SessionStartInput is the stdin JSON for SessionStart hook events.
type SessionStartInput struct {
	HookInputBase
	Source string `json:"source,omitempty"`
	Model  string `json:"model,omitempty"`
}

// SessionStartOutput is the stdout JSON response for SessionStart hooks.
type SessionStartOutput struct {
	HookOutputBase
}

// SessionEndInput is the stdin JSON for SessionEnd hook events.
type SessionEndInput struct {
	HookInputBase
}

// SessionEndOutput is the stdout JSON response for SessionEnd hooks.
type SessionEndOutput struct {
	HookOutputBase
}
