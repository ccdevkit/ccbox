package permissions

import (
	"encoding/json"
	"fmt"

	"gopkg.in/yaml.v3"
)

// Effect represents the outcome of a permission rule.
type Effect string

const (
	EffectAllow Effect = "allow"
	EffectDeny  Effect = "deny"
)

// MatchResult is the outcome of evaluating a command against its permission rules.
type MatchResult struct {
	Allowed     bool
	Reason      string
	MatchedRule *CompiledRule
	Command     string
}

// PermissionsConfig is the top-level configuration parsed from .ccbox/permissions.{json,yml,yaml}.
type PermissionsConfig struct {
	Passthrough map[string]*CommandPermission `yaml:"passthrough" json:"passthrough"`
}

// UnmarshalJSON provides helpful errors for common config mistakes.
func (pc *PermissionsConfig) UnmarshalJSON(data []byte) error {
	// Check if user wrote an array instead of a map at top level
	if len(data) > 0 && data[0] == '[' {
		return fmt.Errorf("\"passthrough\" must be a map of command names, not an array; e.g.:\n  passthrough:\n    git:\n      rules:\n        - effect: allow\n          pattern: \"**\"")
	}

	type permissionsConfigAlias PermissionsConfig
	var alias permissionsConfigAlias
	if err := json.Unmarshal(data, &alias); err != nil {
		return err
	}
	*pc = PermissionsConfig(alias)
	return nil
}

// CommandPermission defines the permission rules for a single passthrough command.
type CommandPermission struct {
	Rules []Rule `yaml:"rules" json:"rules"`
}

// UnmarshalJSON provides a helpful error when rules are placed directly under
// the command name instead of under a "rules" key.
func (cp *CommandPermission) UnmarshalJSON(data []byte) error {
	// Check if the user wrote an array directly (common mistake)
	if len(data) > 0 && data[0] == '[' {
		return fmt.Errorf("rules array found directly under command name; wrap it in a \"rules\" key, e.g.:\n  git:\n    rules:\n      - effect: allow\n        pattern: status")
	}
	type commandPermissionAlias CommandPermission
	var alias commandPermissionAlias
	if err := json.Unmarshal(data, &alias); err != nil {
		return err
	}
	*cp = CommandPermission(alias)
	return nil
}

// Rule is a single entry in the cascading permission array.
type Rule struct {
	Pattern PatternOrArray `yaml:"pattern" json:"pattern"`
	Effect  string         `yaml:"effect" json:"effect"`
	Reason  string         `yaml:"reason,omitempty" json:"reason,omitempty"`
}

// UnmarshalJSON provides helpful errors for common rule mistakes.
func (r *Rule) UnmarshalJSON(data []byte) error {
	// Check if user wrote a bare string instead of an object
	if len(data) > 0 && data[0] == '"' {
		var s string
		if json.Unmarshal(data, &s) == nil {
			return fmt.Errorf("rule must be an object with \"pattern\" and \"effect\" keys, got string %q; e.g.:\n  - pattern: %s\n    effect: allow", s, s)
		}
	}

	type ruleAlias Rule
	var alias ruleAlias
	if err := json.Unmarshal(data, &alias); err != nil {
		return fmt.Errorf("invalid rule: %w; each rule needs \"pattern\" and \"effect\" keys, e.g.:\n  - pattern: \"**\"\n    effect: allow", err)
	}

	if alias.Pattern.Values == nil {
		return fmt.Errorf("rule is missing \"pattern\" key; each rule needs \"pattern\" and \"effect\", e.g.:\n  - pattern: \"**\"\n    effect: allow")
	}
	if alias.Effect == "" {
		return fmt.Errorf("rule is missing \"effect\" key; each rule needs \"pattern\" and \"effect\" (\"allow\" or \"deny\"), e.g.:\n  - pattern: \"**\"\n    effect: allow")
	}

	*r = Rule(alias)
	return nil
}

// PatternOrArray accepts either a string or []string from YAML/JSON.
type PatternOrArray struct {
	Values []string
}

// UnmarshalYAML handles both scalar string and sequence of strings.
func (p *PatternOrArray) UnmarshalYAML(value *yaml.Node) error {
	switch value.Kind {
	case yaml.ScalarNode:
		if value.Tag != "!!str" {
			return fmt.Errorf("\"pattern\" must be a string or array of strings, got %s value %q; e.g.:\n  pattern: \"**\"", value.Tag, value.Value)
		}
		p.Values = []string{value.Value}
		return nil
	case yaml.SequenceNode:
		var items []string
		if err := value.Decode(&items); err != nil {
			return err
		}
		p.Values = items
		return nil
	default:
		return fmt.Errorf("\"pattern\" must be a string or array of strings; e.g.:\n  pattern: \"**\"\n  # or\n  pattern: [\"build\", \"test\", \"lint\"]")
	}
}

// UnmarshalJSON handles both a JSON string and an array of strings.
func (p *PatternOrArray) UnmarshalJSON(data []byte) error {
	var s string
	if err := json.Unmarshal(data, &s); err == nil {
		p.Values = []string{s}
		return nil
	}
	var arr []string
	if err := json.Unmarshal(data, &arr); err == nil {
		p.Values = arr
		return nil
	}
	return fmt.Errorf("\"pattern\" must be a string or array of strings; e.g.:\n  pattern: \"**\"\n  # or\n  pattern: [\"build\", \"test\", \"lint\"]")
}

// ArgPattern is a parsed, validated pattern ready for matching.
type ArgPattern struct {
	Raw        string
	Elements   []PatternElement
	ExactMatch bool
}

// PatternElement is a single token in a parsed pattern.
type PatternElement struct {
	Type          ElementType
	Value         string
	Optional      bool
	NonPositional bool
	Group         []PatternElement
}

// ElementType identifies the kind of a pattern element.
type ElementType string

const (
	ElementLiteral        ElementType = "literal"
	ElementWildcard       ElementType = "wildcard"
	ElementDoubleWildcard ElementType = "doubleWildcard"
	ElementSingleChar     ElementType = "singleChar"
	ElementRegex          ElementType = "regex"
	ElementRegexMulti     ElementType = "regexMulti"
	ElementQuoted         ElementType = "quoted"
	ElementGroup          ElementType = "group"
)

// CompiledRule is the internal representation after pattern parsing and array expansion.
type CompiledRule struct {
	Pattern *ArgPattern
	Effect  Effect
	Reason  string
}
