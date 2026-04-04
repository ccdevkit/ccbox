package claude

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/ccdevkit/ccbox/internal/constants"
	"github.com/ccdevkit/ccbox/internal/session"
)

// settingsJSON is the Claude Code settings written into the container to bypass
// permissions and enable MCP servers (FR-029).
const settingsJSON = `{"allowedTools":[],"enableAllProjectMcpServers":true,"bypassPermissions":true}`

// Container paths for session files.
const (
	settingsContainerPath   = constants.ContainerHomeDir + ".claude/settings.json"
	claudeJSONContainerPath = constants.ContainerHomeDir + ".claude.json"
)

// writeSettings writes the Claude Code settings.json into the container via the
// session file writer.
func writeSettings(fw session.SessionFileWriter) error {
	return fw.WriteFile(settingsContainerPath, []byte(settingsJSON), true)
}

// ensureClaudeJSON ensures ~/.ccbox/.claude.json exists on the host with the
// required onboarding/permissions flags and oauthAccount from the host's
// ~/.claude.json. Only creates the file if it doesn't already exist.
// Returns the host path for bind-mounting into the container.
func ensureClaudeJSON() (string, error) {
	home, err := os.UserHomeDir()
	if err != nil {
		return "", fmt.Errorf("get home dir: %w", err)
	}

	ccboxDir := filepath.Join(home, constants.SettingsDirName)
	if err := os.MkdirAll(ccboxDir, 0755); err != nil {
		return "", fmt.Errorf("create %s dir: %w", constants.SettingsDirName, err)
	}

	hostPath := filepath.Join(ccboxDir, ".claude.json")

	// If it already exists, don't overwrite.
	if _, err := os.Stat(hostPath); err == nil {
		return hostPath, nil
	}

	config := map[string]any{
		"has_completed_onboarding":      true,
		"hasCompletedOnboarding":        true,
		"permissions_accepted":          true,
		"bypassPermissionsModeAccepted": true,
	}

	// Copy oauthAccount from host .claude.json to preserve subscription/auth info.
	srcPath := filepath.Join(home, ".claude.json")
	if data, err := os.ReadFile(srcPath); err == nil {
		var hostConfig map[string]any
		if err := json.Unmarshal(data, &hostConfig); err == nil {
			if oauthAccount, ok := hostConfig["oauthAccount"]; ok {
				config["oauthAccount"] = oauthAccount
			}
		}
	}

	content, err := json.Marshal(config)
	if err != nil {
		return "", fmt.Errorf("marshal .claude.json: %w", err)
	}

	if err := os.WriteFile(hostPath, content, 0600); err != nil {
		return "", fmt.Errorf("write .claude.json: %w", err)
	}

	return hostPath, nil
}

// writeSystemPrompt writes a markdown system prompt containing the allowed
// command list into the container at SystemPromptContainerPath.
func writeSystemPrompt(fw session.SessionFileWriter, commands []string) error {
	var b strings.Builder
	b.WriteString("# Allowed Host Commands\n\n")
	b.WriteString("The following commands are available for execution on the host:\n\n")
	for _, cmd := range commands {
		fmt.Fprintf(&b, "- `%s`\n", cmd)
	}
	return fw.WriteFile(constants.SystemPromptContainerPath, []byte(b.String()), true)
}
