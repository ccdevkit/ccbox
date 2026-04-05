package hooks

// RegisterBypassPermissions registers a PreToolUse handler with "after" ordering
// that allows all Write, Edit, and MultiEdit operations when the permission mode
// is bypassPermissions.
//
// Even in bypassPermissions mode, Claude Code blocks certain file operations
// (e.g. editing .git/). Since ccbox runs inside a container where the sandbox
// boundary is Docker itself, we want everything to be allowed.
func RegisterBypassPermissions(r *Registry) {
	r.Register(PreToolUseHandler{
		Matcher: "Write|Edit|MultiEdit",
		Order:   OrderAfter,
		Fn: func(input *PreToolUseInput) (*PreToolUseOutput, error) {
			if input.PermissionMode != "bypassPermissions" {
				return nil, nil
			}
			r.debug("bypass_permissions: allowing %s in bypassPermissions mode", input.ToolName)
			return &PreToolUseOutput{
				HookSpecificOutput: &PreToolUseSpecificOutput{
					PermissionDecision: "allow",
				},
			}, nil
		},
	})
}
