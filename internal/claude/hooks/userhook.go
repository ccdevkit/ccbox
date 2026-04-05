package hooks

import (
	"encoding/json"
	"fmt"
	"regexp"
	"strings"
)

// UserHook represents a user-defined hook entry captured from Claude Code settings.
type UserHook struct {
	Command string // Shell command to execute
	Matcher string // Regex matcher from the parent matcher group
	If      string // e.g. "Bash(rm *)" — glob-style filter
}

// ParseIfField parses a Claude Code hook "if" field into its tool name and
// argument pattern components.
//
// Examples:
//
//	"Bash(rm *)"    → ("Bash", "rm *", nil)
//	"Edit(*.ts)"    → ("Edit", "*.ts", nil)
//	"Bash"          → ("Bash", "", nil)  — matches all uses
//	""              → ("", "", nil)
//	"Bash(unclosed" → error
//	"(no tool)"     → error
func ParseIfField(ifField string) (toolName, argPattern string, err error) {
	if ifField == "" {
		return "", "", nil
	}

	openIdx := strings.IndexByte(ifField, '(')
	if openIdx == -1 {
		// No parens — tool name only, matches all uses
		return ifField, "", nil
	}

	toolName = ifField[:openIdx]
	if toolName == "" {
		return "", "", fmt.Errorf("missing tool name in if field: %q", ifField)
	}

	// Find the last ')' to handle nested parens like "Bash(echo (hello))"
	closeIdx := strings.LastIndexByte(ifField, ')')
	if closeIdx == -1 || closeIdx <= openIdx {
		return "", "", fmt.Errorf("unclosed parenthesis in if field: %q", ifField)
	}

	argPattern = ifField[openIdx+1 : closeIdx]
	return toolName, argPattern, nil
}

// GlobMatch performs glob-style pattern matching compatible with Claude Code's
// permission rule syntax. The '*' wildcard matches any sequence of characters
// (including none). '**' in path context matches across directory separators.
//
// Key behaviors matching Claude Code:
//   - "ls *" (space before *) matches "ls -la" but NOT "lsof"
//   - "ls*" (no space) matches both "ls -la" and "lsof"
//   - Special regex characters in the pattern are treated as literals
func GlobMatch(pattern, subject string) bool {
	if pattern == "" {
		return subject == ""
	}

	regexStr := globToRegex(pattern)
	matched, err := regexp.MatchString("^"+regexStr+"$", subject)
	if err != nil {
		return false
	}
	return matched
}

// globToRegex converts a glob pattern to a regex string.
//
// '**' matches any sequence of characters including path separators, and when
// used as a path segment (e.g. "/**/") also matches zero segments.
// '*' matches any sequence of characters.
// '?' matches any single character.
// All other regex metacharacters are escaped.
func globToRegex(glob string) string {
	var buf strings.Builder
	i := 0
	for i < len(glob) {
		ch := glob[i]
		switch ch {
		case '*':
			if i+1 < len(glob) && glob[i+1] == '*' {
				// '**' — match anything including path separators
				i += 2
				// If preceded by '/' and followed by '/', match zero or more
				// path segments: convert "/**/" to "(/.*)?/"
				if buf.Len() > 0 && buf.String()[buf.Len()-1] == '/' && i < len(glob) && glob[i] == '/' {
					// Rewrite trailing '/' from buf + '**/' to optional segment
					s := buf.String()
					buf.Reset()
					buf.WriteString(s[:len(s)-1]) // remove trailing /
					buf.WriteString("(/.*)?/")
					i++ // consume the trailing /
				} else {
					buf.WriteString(".*")
				}
			} else {
				buf.WriteString(".*")
				i++
			}
		case '?':
			buf.WriteRune('.')
			i++
		case '.', '+', '(', ')', '[', ']', '{', '}', '^', '$', '|', '\\':
			buf.WriteByte('\\')
			buf.WriteByte(ch)
			i++
		default:
			buf.WriteByte(ch)
			i++
		}
	}
	return buf.String()
}

// MatchIf evaluates a Claude Code hook "if" field against an event and input.
//
// Rules:
//   - Empty if field → always matches
//   - Non-tool event with non-empty if field → never matches
//   - Tool event: parse if field, check tool name, then glob-match the argument
func MatchIf(ifField string, event HookEvent, input json.RawMessage) bool {
	if ifField == "" {
		return true
	}

	// Non-tool events with an if field never fire
	if !toolEvents[event] {
		return false
	}

	toolName, argPattern, err := ParseIfField(ifField)
	if err != nil {
		return false
	}

	// Extract tool_name from input
	var envelope struct {
		ToolName string `json:"tool_name"`
	}
	if json.Unmarshal(input, &envelope) != nil {
		return false
	}

	// Tool name must match
	if envelope.ToolName != toolName {
		return false
	}

	// No arg pattern means match all uses of this tool
	if argPattern == "" {
		return true
	}

	// Extract the subject to match against
	subject := extractIfSubject(toolName, input)
	return GlobMatch(argPattern, subject)
}

// extractIfSubject extracts the string to match the if-field pattern against,
// based on the tool type.
//
//   - Bash: tool_input.command
//   - Edit/Write/MultiEdit/Read: tool_input.file_path
//   - Other: empty string (won't match any pattern)
func extractIfSubject(toolName string, input json.RawMessage) string {
	var envelope struct {
		ToolInput json.RawMessage `json:"tool_input"`
	}
	if json.Unmarshal(input, &envelope) != nil || envelope.ToolInput == nil {
		return ""
	}

	switch toolName {
	case "Bash":
		var ti struct {
			Command string `json:"command"`
		}
		if json.Unmarshal(envelope.ToolInput, &ti) == nil {
			return ti.Command
		}
	case "Edit", "Write", "MultiEdit", "Read":
		var ti struct {
			FilePath string `json:"file_path"`
		}
		if json.Unmarshal(envelope.ToolInput, &ti) == nil {
			return ti.FilePath
		}
	}
	return ""
}

// matchesUserHook returns true if the user hook's matcher pattern matches the
// input for the given event. Uses the same logic as matchesHandler but operates
// on a UserHook instead of a HookHandler.
func matchesUserHook(hook UserHook, event HookEvent, input json.RawMessage) bool {
	pattern := hook.Matcher
	if pattern == "" {
		return true
	}

	var target matcherTarget
	_ = json.Unmarshal(input, &target)

	var subject string
	if toolEvents[event] {
		subject = target.ToolName
	} else {
		subject = target.HookEventName
	}

	matched, err := regexp.MatchString("^(?:"+pattern+")$", subject)
	if err != nil {
		return false
	}
	return matched
}
