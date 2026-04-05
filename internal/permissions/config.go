package permissions

import (
	"encoding/json"
	"fmt"

	"github.com/ccdevkit/common/settings"
)

// Load discovers and parses permissions from .ccbox/permissions.{json,yml,yaml}
// using hierarchical file walk. Returns nil config (not error) if no files found.
// Returns error if files exist but are malformed or contain invalid patterns.
func Load() (*PermissionsConfig, error) {
	// Use a raw map for settings.Load to avoid mixed-tag detection
	// (PermissionsConfig has both json and yaml struct tags).
	var raw map[string]interface{}
	err := settings.Load(".ccbox/permissions", &raw, nil)
	if err != nil {
		return nil, fmt.Errorf("loading permissions config: %w", err)
	}
	if raw == nil {
		return nil, nil
	}

	// Re-serialize as JSON and decode into PermissionsConfig
	// to leverage the custom UnmarshalJSON on PatternOrArray.
	data, err := json.Marshal(raw)
	if err != nil {
		return nil, fmt.Errorf("marshaling permissions config: %w", err)
	}

	var config PermissionsConfig
	if err := json.Unmarshal(data, &config); err != nil {
		return nil, fmt.Errorf("parsing permissions config: %w", err)
	}

	if config.Passthrough == nil {
		return nil, nil
	}

	if err := validate(&config); err != nil {
		return nil, err
	}
	return &config, nil
}

// validate checks the parsed config for semantic errors.
func validate(config *PermissionsConfig) error {
	for name, cmd := range config.Passthrough {
		if name == "" {
			return fmt.Errorf("empty command name in permissions config")
		}
		for _, rule := range cmd.Rules {
			if rule.Effect != "allow" && rule.Effect != "deny" {
				return fmt.Errorf("invalid effect %q for command %q, must be \"allow\" or \"deny\"", rule.Effect, name)
			}
		}
	}
	return nil
}
