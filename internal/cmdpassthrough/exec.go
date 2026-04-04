package cmdpassthrough

import (
	"bytes"
	"os/exec"
	"syscall"

	"github.com/ccdevkit/ccbox/internal/bridge"
)

// HandleExec executes a command via sh -c and returns the exit code and
// combined stdout+stderr output. The working directory is set from the request.
func HandleExec(req bridge.ExecRequest) (int, []byte) {
	cmd := exec.Command("sh", "-c", req.Command)
	cmd.Dir = req.Cwd

	output, err := cmd.CombinedOutput()
	if err != nil {
		if exitErr, ok := err.(*exec.ExitError); ok {
			if status, ok := exitErr.Sys().(syscall.WaitStatus); ok {
				return status.ExitStatus(), output
			}
		}
		// If we can't determine the exit code (e.g., command not found),
		// return 127 which is the conventional shell exit code for this.
		if output == nil {
			output = []byte(err.Error())
		}
		return 127, output
	}

	return 0, output
}

// RewriteContainerPaths replaces all occurrences of containerHome with hostHome
// in the given output. This translates container paths (e.g. /home/claude/) back
// to host paths (e.g. /Users/brad/) so the user sees familiar locations.
func RewriteContainerPaths(output []byte, containerHome, hostHome string) []byte {
	return rewriteContainerPaths(output, containerHome, hostHome)
}

func rewriteContainerPaths(output []byte, containerHome, hostHome string) []byte {
	if output == nil {
		return nil
	}
	return bytes.ReplaceAll(output, []byte(containerHome), []byte(hostHome))
}
