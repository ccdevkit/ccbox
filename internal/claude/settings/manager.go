package settings

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/ccdevkit/ccbox/internal/constants"
	"github.com/ccdevkit/ccbox/internal/logger"
	"github.com/ccdevkit/ccbox/internal/session"
)

// FS abstracts filesystem operations for testability.
type FS interface {
	ReadFile(path string) ([]byte, error)
	Stat(path string) (os.FileInfo, error)
}

// OSFS implements FS using the real filesystem.
type OSFS struct{}

func (OSFS) ReadFile(path string) ([]byte, error) { return os.ReadFile(path) }
func (OSFS) Stat(path string) (os.FileInfo, error) { return os.Stat(path) }

// ClaudeSettingsManager discovers, merges, and manages Claude Code settings.
type ClaudeSettingsManager struct {
	fs         FS
	homeDir    string
	projectDir string
	merged    map[string]interface{}
	finalized bool
	log       *logger.Logger
}

// SetLogger sets the debug logger.
func (m *ClaudeSettingsManager) SetLogger(log *logger.Logger) {
	m.log = log
}

func (m *ClaudeSettingsManager) debug(format string, args ...interface{}) {
	if m.log != nil {
		m.log.Debug("settings", format, args...)
	}
}

// NewClaudeSettingsManager creates a manager that discovers and merges
// settings from standard locations.
//
// Discovery order (lowest to highest precedence):
//  1. {homeDir}/.claude/settings.json
//  2. {homeDir}/.claude/settings.local.json
//  3. {projectDir}/.claude/settings.json
//  4. {projectDir}/.claude/settings.local.json
func NewClaudeSettingsManager(fs FS, homeDir string, projectDir string) (*ClaudeSettingsManager, error) {
	paths := []string{
		filepath.Join(homeDir, ".claude", "settings.json"),
		filepath.Join(homeDir, ".claude", "settings.local.json"),
		filepath.Join(projectDir, ".claude", "settings.json"),
		filepath.Join(projectDir, ".claude", "settings.local.json"),
	}

	merged := make(map[string]interface{})

	for _, p := range paths {
		data, err := fs.ReadFile(p)
		if err != nil {
			if errors.Is(err, os.ErrNotExist) {
				continue
			}
			return nil, err
		}

		var m map[string]interface{}
		if err := json.Unmarshal(data, &m); err != nil {
			// Malformed JSON — skip this file and continue merging the rest.
			continue
		}

		for k, v := range m {
			merged[k] = v
		}
	}

	return &ClaudeSettingsManager{
		fs:         fs,
		homeDir:    homeDir,
		projectDir: projectDir,
		merged:     merged,
	}, nil
}

// Merged returns the merged settings map.
func (m *ClaudeSettingsManager) Merged() map[string]interface{} {
	return m.merged
}

// Set sets a top-level key in the merged settings.
func (m *ClaudeSettingsManager) Set(key string, value interface{}) {
	if m.finalized {
		panic("Set called after Finalize")
	}
	m.merged[key] = value
}

// Delete removes a top-level key from the merged settings.
func (m *ClaudeSettingsManager) Delete(key string) {
	if m.finalized {
		panic("Delete called after Finalize")
	}
	delete(m.merged, key)
}

// SetDeep sets a nested key using dot notation (e.g., "hooks.PreToolUse").
// Creates intermediate maps as needed.
func (m *ClaudeSettingsManager) SetDeep(path string, value interface{}) {
	if m.finalized {
		panic("SetDeep called after Finalize")
	}
	segments := strings.Split(path, ".")
	if len(segments) == 1 {
		m.merged[path] = value
		return
	}

	current := m.merged
	for _, seg := range segments[:len(segments)-1] {
		next, ok := current[seg]
		if !ok {
			newMap := make(map[string]interface{})
			current[seg] = newMap
			current = newMap
			continue
		}
		nextMap, ok := next.(map[string]interface{})
		if !ok {
			newMap := make(map[string]interface{})
			current[seg] = newMap
			current = newMap
			continue
		}
		current = nextMap
	}
	current[segments[len(segments)-1]] = value
}

// CapturedHook represents a user-defined hook entry captured from settings.
type CapturedHook struct {
	Command string // Shell command to execute
	Matcher string // Regex matcher from the parent matcher group
	If      string // Optional "if" field, e.g. "Bash(rm *)"
}

// CaptureAndReplaceHooks extracts all user-defined command hook entries from
// the merged settings, then replaces the entire hooks config with catch-all
// entries pointing to the proxy command for every event that needs handling.
//
// registeredEvents is the set of event names that have Go-side handlers.
// Returns a map of event name -> captured user hooks for runtime dispatch.
func (m *ClaudeSettingsManager) CaptureAndReplaceHooks(proxyCommand string, registeredEvents map[string]bool) map[string][]CapturedHook {
	if m.finalized {
		panic("CaptureAndReplaceHooks called after Finalize")
	}

	m.debug("capturing user hooks and replacing with proxy command: %s", proxyCommand)

	captured := make(map[string][]CapturedHook)

	// Extract existing user hooks
	if hooksRaw, ok := m.merged["hooks"]; ok {
		if hooksMap, ok := hooksRaw.(map[string]interface{}); ok {
			for eventName, eventRaw := range hooksMap {
				groups, ok := eventRaw.([]interface{})
				if !ok {
					continue
				}
				for _, groupRaw := range groups {
					group, ok := groupRaw.(map[string]interface{})
					if !ok {
						continue
					}
					matcher, _ := group["matcher"].(string)
					hookEntries, ok := group["hooks"].([]interface{})
					if !ok {
						continue
					}
					for _, entryRaw := range hookEntries {
						entry, ok := entryRaw.(map[string]interface{})
						if !ok {
							continue
						}
						// Only capture command hooks — HTTP/prompt/agent hooks
						// can't be proxied as subprocesses
						entryType, _ := entry["type"].(string)
						if entryType != "command" {
							continue
						}
						command, _ := entry["command"].(string)
						if command == "" {
							continue
						}
						ifField, _ := entry["if"].(string)
						captured[eventName] = append(captured[eventName], CapturedHook{
							Command: command,
							Matcher: matcher,
							If:      ifField,
						})
					}
				}
			}
		}
	}

	// Log captured hooks summary
	totalCaptured := 0
	for event, hooks := range captured {
		totalCaptured += len(hooks)
		m.debug("captured %d user hooks for %s", len(hooks), event)
	}

	// Build the set of all events that need a proxy entry
	allEvents := make(map[string]bool)
	for event := range registeredEvents {
		allEvents[event] = true
	}
	for event := range captured {
		allEvents[event] = true
	}

	// Replace hooks config with catch-all proxy entries
	newHooks := make(map[string]interface{})
	for event := range allEvents {
		newHooks[event] = []interface{}{
			map[string]interface{}{
				"matcher": "*",
				"hooks": []interface{}{
					map[string]interface{}{
						"type":    "command",
						"command": proxyCommand,
					},
				},
			},
		}
	}
	m.merged["hooks"] = newHooks

	m.debug("replaced hooks config: %d events with proxy entries, %d user hooks captured", len(allEvents), totalCaptured)

	return captured
}

// Finalize writes the merged settings to a session file and returns CLI args.
// After this call, no further modifications are allowed.
func (m *ClaudeSettingsManager) Finalize(fw session.SessionFileWriter) ([]string, error) {
	if m.finalized {
		return nil, fmt.Errorf("already finalized")
	}

	data, err := json.MarshalIndent(m.merged, "", "  ")
	if err != nil {
		return nil, fmt.Errorf("marshal settings: %w", err)
	}

	containerPath := constants.SettingsContainerPath
	if err := fw.WriteFile(containerPath, data, true); err != nil {
		return nil, fmt.Errorf("write settings file: %w", err)
	}

	m.finalized = true
	return []string{"--settings", containerPath, "--setting-sources", ""}, nil
}
