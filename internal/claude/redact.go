package claude

import "strings"

// RedactToken replaces all occurrences of token in input with a masked version.
// Tokens longer than 4 characters show "***...XXXX" (last 4 chars); shorter
// tokens are fully replaced with "***REDACTED***".
// An empty token returns input unchanged.
func RedactToken(token, input string) string {
	if token == "" {
		return input
	}
	var mask string
	if len(token) > 4 {
		mask = "***..." + token[len(token)-4:]
	} else {
		mask = "***REDACTED***"
	}
	return strings.ReplaceAll(input, token, mask)
}
