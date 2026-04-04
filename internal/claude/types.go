package claude

// EnvVar represents an environment variable needed by the claude process.
// Secret values are redacted in debug logs.
type EnvVar struct {
	Key    string
	Value  string
	Secret bool
}

// ClaudeRunSpec is the docker-agnostic description of a claude CLI invocation.
// Produced by BuildRunSpec. Contains only what's needed to invoke the claude
// binary — no Docker concepts (mounts, images, ports).
type ClaudeRunSpec struct {
	Args    []string
	Env     []EnvVar
	WorkDir string
}
