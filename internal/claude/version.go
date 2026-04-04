package claude

import (
	"fmt"
	"regexp"
	"strings"
)

// VersionRunner allows injecting test doubles for version command execution.
type VersionRunner interface {
	Output(name string, args ...string) ([]byte, error)
}

var versionRegexp = regexp.MustCompile(`(\d+\.\d+\.\d+(?:-[a-zA-Z0-9.]+)?)`)

// DetectVersion runs the claude CLI with --version and parses the version string.
func DetectVersion(claudePath string, runner VersionRunner) (string, error) {
	out, err := runner.Output(claudePath, "--version")
	if err != nil {
		return "", fmt.Errorf("failed to run %s --version: %w", claudePath, err)
	}

	match := versionRegexp.FindString(strings.TrimSpace(string(out)))
	if match == "" {
		return "", fmt.Errorf("no version found in output: %q", string(out))
	}

	return match, nil
}
