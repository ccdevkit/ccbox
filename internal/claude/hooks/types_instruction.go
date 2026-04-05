package hooks

import "encoding/json"

// InstructionsLoadedInput is the stdin payload for InstructionsLoaded events.
type InstructionsLoadedInput struct {
	HookInputBase
	Instructions json.RawMessage `json:"instructions,omitempty"`
}

// InstructionsLoadedOutput is the stdout response for InstructionsLoaded events.
type InstructionsLoadedOutput struct {
	HookOutputBase
}

// UserPromptSubmitInput is the stdin payload for UserPromptSubmit events.
type UserPromptSubmitInput struct {
	HookInputBase
	Prompt string `json:"prompt,omitempty"`
}

// UserPromptSubmitOutput is the stdout response for UserPromptSubmit events.
type UserPromptSubmitOutput struct {
	HookOutputBase
}
