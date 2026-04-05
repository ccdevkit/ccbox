package claude

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/ccdevkit/ccbox/internal/args"
	"github.com/ccdevkit/ccbox/internal/constants"
	"github.com/ccdevkit/ccbox/internal/session"
	"github.com/ccdevkit/ccbox/internal/settings"
)

// Claude holds the state needed to build a claude CLI invocation.
type Claude struct {
	Session              *session.Session
	Token                string
	passthroughCommands  []string
}

// SetPassthroughEnabled stores the allowed passthrough commands for later
// inclusion in the merged system prompt written by BuildRunSpec.
func (c *Claude) SetPassthroughEnabled(commands []string) {
	c.passthroughCommands = commands
}

// New creates a Claude instance for the given session, writing the required
// session files (settings.json) via the session's FileWriter and ensuring
// ~/.ccbox/.claude.json exists on disk for bind-mounting.
func New(sess *session.Session) (*Claude, error) {
	if err := writeSettings(sess.FileWriter); err != nil {
		return nil, fmt.Errorf("write settings: %w", err)
	}
	claudeJSONPath, err := ensureClaudeJSON()
	if err != nil {
		return nil, fmt.Errorf("ensure claude.json: %w", err)
	}
	if err := sess.AddFilePassthrough(claudeJSONPath, claudeJSONContainerPath, false); err != nil {
		return nil, fmt.Errorf("mount claude.json: %w", err)
	}
	return &Claude{Session: sess}, nil
}

// BuildRunSpec produces a docker-agnostic ClaudeRunSpec from parsed CLI args
// and settings. It also registers file passthroughs on the session for CWD,
// ~/.claude/, and any file args from ParsedArgs. The fs parameter is used to
// read user-provided --append-system-prompt-file content from the host.
func (c *Claude) BuildRunSpec(parsedArgs *args.ParsedArgs, _ *settings.Settings, fs args.FileSystem) (*ClaudeRunSpec, error) {
	// Scan for --append-system-prompt / --append-system-prompt-file before the
	// rewrite loop so we read files at their original host paths.
	scanResult, err := scanAppendArgs(parsedArgs.ClaudeArgs, fs)
	if err != nil {
		return nil, fmt.Errorf("scan append system prompt args: %w", err)
	}

	stripSet := make(map[int]bool, len(scanResult.StripIndices))
	for _, idx := range scanResult.StripIndices {
		stripSet[idx] = true
	}

	// Build args, rewriting file paths to container paths.
	cliArgs := make([]string, 0, len(parsedArgs.ClaudeArgs)+6)
	hasPermissionMode := false
	hasDangerouslySkip := false
	for i, ca := range parsedArgs.ClaudeArgs {
		if stripSet[i] {
			continue
		}
		if ca.Value == "--permission-mode" {
			hasPermissionMode = true
		}
		if ca.Value == "--allow-dangerously-skip-permissions" {
			hasDangerouslySkip = true
		}
		if ca.IsFile {
			containerPath := filepath.Join(constants.ContainerHomeDir, filepath.Base(ca.Value))
			if err := c.Session.AddFilePassthrough(ca.Value, containerPath, true); err != nil {
				return nil, err
			}
			cliArgs = append(cliArgs, containerPath)
		} else {
			cliArgs = append(cliArgs, ca.Value)
		}
	}

	// Merge user append content with ccbox system prompt and write the
	// combined file into the container.
	ccboxContent := buildCcboxSystemPrompt(c.passthroughCommands)
	merged := ccboxContent
	if scanResult.UserContent != "" {
		merged = scanResult.UserContent + "\n\n" + merged
	}
	if err := c.Session.FileWriter.WriteFile(constants.SystemPromptContainerPath, []byte(merged), true); err != nil {
		return nil, fmt.Errorf("write merged system prompt: %w", err)
	}
	cliArgs = append(cliArgs, "--append-system-prompt-file", constants.SystemPromptContainerPath)

	// Default to bypassPermissions — the container sandbox replaces the
	// permission prompt system (FR-001).
	if !hasPermissionMode {
		cliArgs = append(cliArgs, "--permission-mode", "bypassPermissions")
	}
	if !hasDangerouslySkip {
		cliArgs = append(cliArgs, "--allow-dangerously-skip-permissions")
	}

	// Build env vars.
	env := []EnvVar{
		{Key: constants.EnvTerm, Value: os.Getenv(constants.EnvTerm)},
		{Key: constants.EnvColorTerm, Value: os.Getenv(constants.EnvColorTerm)},
		{Key: constants.EnvClaudeOAuthToken, Value: c.Token, Secret: true},
	}

	// Register CWD passthrough (rw, identity path).
	cwd, err := os.Getwd()
	if err != nil {
		return nil, err
	}
	if err := c.Session.AddFilePassthrough(cwd, cwd, false); err != nil {
		return nil, err
	}

	// Register ~/.claude/ passthrough (rw).
	home, err := os.UserHomeDir()
	if err != nil {
		return nil, err
	}
	claudeDir := filepath.Join(home, constants.ClaudeConfigDirName)
	containerClaudeDir := filepath.Join(constants.ContainerHomeDir, constants.ClaudeConfigDirName)
	if err := c.Session.AddFilePassthrough(claudeDir, containerClaudeDir, false); err != nil {
		return nil, err
	}

	return &ClaudeRunSpec{
		Args:    cliArgs,
		Env:     env,
		WorkDir: cwd,
	}, nil
}
