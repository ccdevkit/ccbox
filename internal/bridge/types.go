package bridge

// ExecRequest is sent from the container ccptproxy to the host TCP server
// to execute a command on the host.
type ExecRequest struct {
	Type    string `json:"type"`    // Always "exec"
	Command string `json:"command"` // Shell command to execute
	Cwd     string `json:"cwd"`    // Working directory for execution
}

// LogRequest is a fire-and-forget log message from container to host.
type LogRequest struct {
	Type    string `json:"type"`    // Always "log"
	Message string `json:"message"` // "[source] message text"
}
