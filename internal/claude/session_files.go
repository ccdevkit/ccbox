package claude

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/ccdevkit/ccbox/internal/args"
	"github.com/ccdevkit/ccbox/internal/constants"
)

// Container path for the .claude.json file inside the container.
const claudeJSONContainerPath = constants.ContainerHomeDir + ".claude.json"

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

// ensureWorkspaceTrust updates ~/.ccbox/.claude.json to mark the given
// directory as trusted so that plugin hooks execute without an interactive
// trust prompt. This is the container equivalent of "claude-trust".
func ensureWorkspaceTrust(cwd string) error {
	home, err := os.UserHomeDir()
	if err != nil {
		return fmt.Errorf("get home dir: %w", err)
	}

	hostPath := filepath.Join(home, constants.SettingsDirName, ".claude.json")

	config := map[string]any{}
	if data, err := os.ReadFile(hostPath); err == nil {
		json.Unmarshal(data, &config)
	}

	projects, _ := config["projects"].(map[string]any)
	if projects == nil {
		projects = map[string]any{}
	}
	proj, _ := projects[cwd].(map[string]any)
	if proj == nil {
		proj = map[string]any{}
	}
	proj["hasTrustDialogAccepted"] = true
	projects[cwd] = proj
	config["projects"] = projects

	content, err := json.Marshal(config)
	if err != nil {
		return fmt.Errorf("marshal .claude.json: %w", err)
	}
	return os.WriteFile(hostPath, content, 0600)
}

// buildCcboxSystemPrompt returns the ccbox system prompt content describing
// the container environment and, if non-empty, the allowed passthrough commands.
func buildCcboxSystemPrompt(commands []string) string {
	var b strings.Builder
	b.WriteString("# ccbox Environment\n\n")
	b.WriteString("You are running inside ccbox, a container-based pseudo-sandbox. ")
	b.WriteString("This gives you broad freedom within the container, but it is not a fully isolated security boundary.\n\n")
	b.WriteString("## Key constraints\n\n")
	b.WriteString("- The **only directory shared with the host** is the project directory (your current working directory). Changes there are immediately visible on the host.\n")
	b.WriteString("- `/tmp` is local to the container — use it freely for scratch files, build artifacts, etc.\n")
	b.WriteString("- **MCP tools** (`mcp__*`) execute on the host, outside the sandbox. Treat them with the same caution you would give any host-side operation.\n")
	b.WriteString("- **Passthrough tools** also execute on the host. Be careful with destructive or sensitive operations.\n")

	if len(commands) > 0 {
		b.WriteString("\n## Allowed Host Commands\n\n")
		b.WriteString("The following commands are available for execution on the host:\n\n")
		for _, cmd := range commands {
			fmt.Fprintf(&b, "- `%s`\n", cmd)
		}
	}

	return b.String()
}

// appendScanResult holds the result of scanning ClaudeArgs for append system
// prompt flags.
type appendScanResult struct {
	UserContent  string // user's append text (inline or from file)
	StripIndices []int  // indices into ClaudeArgs to remove
}

// scanAppendArgs scans ClaudeArgs for --append-system-prompt and
// --append-system-prompt-file, extracting user content and recording which
// indices to strip from the final args. Returns an error if both append flags
// are present (they are mutually exclusive in Claude Code).
func scanAppendArgs(claudeArgs []args.ClaudeArg, fs args.FileSystem) (appendScanResult, error) {
	var result appendScanResult
	var foundInline, foundFile bool

	for i := 0; i < len(claudeArgs); i++ {
		switch claudeArgs[i].Value {
		case "--append-system-prompt":
			if foundFile {
				return result, fmt.Errorf("--append-system-prompt and --append-system-prompt-file are mutually exclusive")
			}
			foundInline = true
			result.StripIndices = append(result.StripIndices, i)
			if i+1 < len(claudeArgs) {
				i++
				result.UserContent = claudeArgs[i].Value
				result.StripIndices = append(result.StripIndices, i)
			}

		case "--append-system-prompt-file":
			if foundInline {
				return result, fmt.Errorf("--append-system-prompt and --append-system-prompt-file are mutually exclusive")
			}
			foundFile = true
			result.StripIndices = append(result.StripIndices, i)
			if i+1 < len(claudeArgs) {
				i++
				result.StripIndices = append(result.StripIndices, i)
				data, err := fs.ReadFile(claudeArgs[i].Value)
				if err != nil {
					return result, fmt.Errorf("read --append-system-prompt-file %q: %w", claudeArgs[i].Value, err)
				}
				result.UserContent = string(data)
			}
		}
	}

	return result, nil
}
