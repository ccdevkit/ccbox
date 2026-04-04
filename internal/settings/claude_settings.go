package settings

import (
	"encoding/json"
	"errors"
	"io/fs"
	"os"
)

// ReadClaudeSettings reads and unmarshals a Claude settings.json file.
// Returns an empty map if the file does not exist.
func ReadClaudeSettings(path string) (map[string]interface{}, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		if errors.Is(err, fs.ErrNotExist) {
			return map[string]interface{}{}, nil
		}
		return nil, err
	}
	var result map[string]interface{}
	if err := json.Unmarshal(data, &result); err != nil {
		return nil, err
	}
	return result, nil
}

// MergeClaudeSettings merges host Claude settings with ccbox overrides.
// Override values take precedence over host values for matching keys.
// Host settings that ccbox does not override are preserved.
func MergeClaudeSettings(hostSettings, overrides map[string]interface{}) map[string]interface{} {
	merged := make(map[string]interface{})
	for k, v := range hostSettings {
		merged[k] = v
	}
	for k, v := range overrides {
		merged[k] = v
	}
	return merged
}
