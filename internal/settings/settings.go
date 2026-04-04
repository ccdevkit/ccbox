// Package settings handles loading ccbox configuration from the filesystem.
package settings

import (
	"errors"

	"github.com/ccdevkit/common/settings"
)

// Settings represents ccbox configuration loaded from the filesystem.
// Settings are loaded from .ccbox/settings.{json,yaml,yml} files
// discovered by walking up from cwd to root.
type Settings struct {
	Passthrough []string `yaml:"passthrough"`
	ClaudePath  string   `yaml:"claudePath"`
	Verbose     bool     `yaml:"verbose"`
	LogFile     string   `yaml:"logFile"`
}

// settingsPath is the relative path for settings file discovery.
const settingsPath = ".ccbox/settings"

// MergeWithCLI returns a new Settings with CLI flags merged on top of base settings.
// String/bool flags override only when non-zero. Passthrough entries are appended.
// The original Settings is not modified.
func MergeWithCLI(base *Settings, passthrough []string, claudePath string, verbose bool, logFile string) *Settings {
	merged := &Settings{
		ClaudePath: base.ClaudePath,
		Verbose:    base.Verbose,
		LogFile:    base.LogFile,
	}

	// Copy passthrough slice to avoid mutating the original, then append CLI entries
	merged.Passthrough = make([]string, len(base.Passthrough))
	copy(merged.Passthrough, base.Passthrough)
	merged.Passthrough = append(merged.Passthrough, passthrough...)

	if claudePath != "" {
		merged.ClaudePath = claudePath
	}
	if verbose {
		merged.Verbose = true
	}
	if logFile != "" {
		merged.LogFile = logFile
	}

	return merged
}

// Load discovers and loads settings from the filesystem.
// Returns zero-value Settings if no files are found.
// Malformed files are silently ignored and defaults are returned.
func Load() (*Settings, error) {
	cfg := &Settings{}
	if err := settings.Load(settingsPath, cfg, nil); err != nil {
		if errors.Is(err, settings.ErrInvalidConfig) {
			return &Settings{}, nil
		}
		return nil, err
	}
	return cfg, nil
}
