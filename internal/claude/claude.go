package claude

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"

	"github.com/ccdevkit/ccbox/internal/args"
	"github.com/ccdevkit/ccbox/internal/claude/hooks"
	claudesettings "github.com/ccdevkit/ccbox/internal/claude/settings"
	"github.com/ccdevkit/ccbox/internal/constants"
	"github.com/ccdevkit/ccbox/internal/logger"
	"github.com/ccdevkit/ccbox/internal/session"
	"github.com/ccdevkit/ccbox/internal/settings"
)

// Claude holds the state needed to build a claude CLI invocation.
type Claude struct {
	Session             *session.Session
	Token               string
	SettingsManager     *claudesettings.ClaudeSettingsManager
	Registry            *hooks.Registry
	passthroughCommands []string
}

// SetLogger sets the debug logger for hook dispatch and settings merging.
func (c *Claude) SetLogger(log *logger.Logger) {
	if c.SettingsManager != nil {
		c.SettingsManager.SetLogger(log)
	}
	if c.Registry != nil {
		c.Registry.SetLogger(log)
	}
}

// SetPassthroughEnabled stores the allowed passthrough commands for later
// inclusion in the merged system prompt written by BuildRunSpec.
func (c *Claude) SetPassthroughEnabled(commands []string) {
	c.passthroughCommands = commands
}

// New creates a Claude instance for the given session, creating a
// ClaudeSettingsManager with default settings and ensuring ~/.ccbox/.claude.json
// exists on disk for bind-mounting.
func New(sess *session.Session) (*Claude, error) {
	home, err := os.UserHomeDir()
	if err != nil {
		return nil, fmt.Errorf("get home dir: %w", err)
	}
	cwd, err := os.Getwd()
	if err != nil {
		return nil, fmt.Errorf("get cwd: %w", err)
	}

	mgr, err := claudesettings.NewClaudeSettingsManager(claudesettings.OSFS{}, home, cwd)
	if err != nil {
		return nil, fmt.Errorf("create settings manager: %w", err)
	}

	// Set the same defaults that were previously hardcoded.
	mgr.Set("allowedTools", []interface{}{})
	mgr.Set("enableAllProjectMcpServers", true)
	mgr.Set("bypassPermissions", true)

	claudeJSONPath, err := ensureClaudeJSON()
	if err != nil {
		return nil, fmt.Errorf("ensure claude.json: %w", err)
	}
	if err := sess.AddFilePassthrough(claudeJSONPath, claudeJSONContainerPath, false); err != nil {
		return nil, fmt.Errorf("mount claude.json: %w", err)
	}
	return &Claude{Session: sess, SettingsManager: mgr}, nil
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
	hasDebug := false
	hasDebugFile := false
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
		if ca.Value == "--debug" {
			hasDebug = true
		}
		if ca.Value == "--debug-file" {
			hasDebugFile = true
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

	// Hook orchestration: capture user hooks, write proxy hooks as a plugin,
	// and register user hooks on the registry for host-side dispatch.
	// Hooks from --settings are ignored by Claude Code, so we deliver them
	// via --plugin-dir which loads hooks/hooks.json from a plugin directory.
	if c.Registry != nil && c.SettingsManager != nil {
		proxyCmd := filepath.Join(constants.ContainerBinDir, constants.HookProxyBinaryName)
		captured := c.SettingsManager.CaptureAndReplaceHooks(proxyCmd, c.Registry.RegisteredEvents())
		for eventName, capturedHooks := range captured {
			userHooks := make([]hooks.UserHook, len(capturedHooks))
			for i, ch := range capturedHooks {
				userHooks[i] = hooks.UserHook{
					Command: ch.Command,
					Matcher: ch.Matcher,
					If:      ch.If,
				}
			}
			c.Registry.SetUserHooks(hooks.HookEvent(eventName), userHooks)
		}
		if projectDir, err := os.Getwd(); err == nil {
			c.Registry.SetProjectDir(projectDir)
		}

		// Extract proxy hooks from merged settings and write as a plugin file.
		// Then remove hooks from settings (they don't work via --settings).
		if hooksVal, ok := c.SettingsManager.Merged()["hooks"]; ok {
			pluginHooks := map[string]interface{}{"hooks": hooksVal}
			pluginJSON, err := json.MarshalIndent(pluginHooks, "", "  ")
			if err != nil {
				return nil, fmt.Errorf("marshal plugin hooks: %w", err)
			}
			if err := c.Session.FileWriter.WriteFile(constants.HookPluginHooksFile, pluginJSON, true); err != nil {
				return nil, fmt.Errorf("write plugin hooks file: %w", err)
			}
			cliArgs = append(cliArgs, "--plugin-dir", constants.HookPluginContainerDir)
			c.SettingsManager.Delete("hooks")
		}
	}

	// Finalize settings manager and append --settings / --setting-sources args.
	if c.SettingsManager != nil {
		settingsArgs, err := c.SettingsManager.Finalize(c.Session.FileWriter)
		if err != nil {
			return nil, fmt.Errorf("finalize settings: %w", err)
		}
		cliArgs = append(cliArgs, settingsArgs...)
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

	// Pre-accept workspace trust for the project directory so plugin hooks
	// execute without an interactive trust prompt inside the container.
	if err := ensureWorkspaceTrust(cwd); err != nil {
		return nil, fmt.Errorf("ensure workspace trust: %w", err)
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

	// If user passed --debug but not --debug-file, inject one so the
	// entrypoint can tail it through ccdebug back to the host log.
	if hasDebug && !hasDebugFile {
		cliArgs = append(cliArgs, "--debug-file", constants.ClaudeDebugFileContainerPath)
	}

	return &ClaudeRunSpec{
		Args:    cliArgs,
		Env:     env,
		WorkDir: cwd,
	}, nil
}
