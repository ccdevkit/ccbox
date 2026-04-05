package permissions

import (
	"fmt"
	"regexp"
	"strings"
)

// tokenizePattern splits a pattern string into tokens, respecting quoted strings.
// Quoted strings (using " or ') are kept as single tokens with quotes preserved.
// Backslash escapes within quotes are processed (e.g., \" becomes ").
// Unquoted regions are split by whitespace.
func tokenizePattern(s string) ([]string, error) {
	var tokens []string
	var current []byte
	i := 0
	for i < len(s) {
		ch := s[i]
		switch {
		case ch == '"' || ch == '\'':
			// Start of quoted string — read until matching close quote.
			quote := ch
			var buf []byte
			buf = append(buf, ch) // preserve opening quote
			i++
			closed := false
			for i < len(s) {
				if s[i] == '\\' && i+1 < len(s) {
					// Escape sequence inside quotes.
					buf = append(buf, s[i], s[i+1])
					i += 2
					continue
				}
				if s[i] == quote {
					buf = append(buf, s[i]) // preserve closing quote
					i++
					closed = true
					break
				}
				buf = append(buf, s[i])
				i++
			}
			if !closed {
				return nil, fmt.Errorf("unclosed %c quote in pattern", quote)
			}
			current = append(current, buf...)
		case ch == '(':
			// Start of group — read until matching ')'.
			var buf []byte
			buf = append(buf, ch) // preserve opening paren
			i++
			depth := 1
			for i < len(s) && depth > 0 {
				if s[i] == '(' {
					depth++
				} else if s[i] == ')' {
					depth--
				}
				buf = append(buf, s[i])
				i++
			}
			if depth != 0 {
				return nil, fmt.Errorf("unclosed '(' in pattern")
			}
			current = append(current, buf...)
		case ch == ' ' || ch == '\t' || ch == '\n' || ch == '\r':
			if len(current) > 0 {
				tokens = append(tokens, string(current))
				current = current[:0]
			}
			i++
		default:
			current = append(current, ch)
			i++
		}
	}
	if len(current) > 0 {
		tokens = append(tokens, string(current))
	}
	return tokens, nil
}

// ParsePattern normalizes whitespace, extracts the exact-match terminator ($),
// and splits the pattern into token elements.
func ParsePattern(raw string) (*ArgPattern, error) {
	// Normalize: trim leading/trailing whitespace.
	trimmed := strings.TrimSpace(raw)

	// Extract exact-match terminator.
	exactMatch := false
	if strings.HasSuffix(trimmed, "$") {
		exactMatch = true
		trimmed = strings.TrimSuffix(trimmed, "$")
		trimmed = strings.TrimRight(trimmed, " \t")
	}

	// Validate: $ must not appear anywhere in the remaining pattern.
	if strings.Contains(trimmed, "$") {
		return nil, fmt.Errorf("'$' must only appear at the end of a pattern")
	}

	// Split into tokens (quote-aware).
	tokens, err := tokenizePattern(trimmed)
	if err != nil {
		return nil, err
	}

	elements := make([]PatternElement, 0, len(tokens))
	for _, tok := range tokens {
		nonPositional := false
		if strings.HasPrefix(tok, "~") {
			nonPositional = true
			tok = tok[1:]
		}

		optional := false
		if strings.HasSuffix(tok, "?") {
			optional = true
			tok = tok[:len(tok)-1]
			if tok == "" {
				return nil, fmt.Errorf("bare '?' is not a valid pattern token")
			}
		}

		var newElements []PatternElement
		switch {
		case len(tok) >= 2 && tok[0] == '(' && tok[len(tok)-1] == ')':
			// Group — recursively parse content between parens.
			inner := tok[1 : len(tok)-1]
			innerTokens, err := tokenizePattern(inner)
			if err != nil {
				return nil, fmt.Errorf("in group: %w", err)
			}
			subElements := make([]PatternElement, 0, len(innerTokens))
			for _, st := range innerTokens {
				subPat, err := ParsePattern(st)
				if err != nil {
					return nil, fmt.Errorf("in group: %w", err)
				}
				subElements = append(subElements, subPat.Elements...)
			}
			newElements = append(newElements, PatternElement{Type: ElementGroup, Group: subElements})
		case (len(tok) >= 2 && tok[0] == '"' && tok[len(tok)-1] == '"') ||
			(len(tok) >= 2 && tok[0] == '\'' && tok[len(tok)-1] == '\''):
			// Quoted string — strip quotes and process escape sequences.
			inner := tok[1 : len(tok)-1]
			inner = processEscapes(inner)
			newElements = append(newElements, PatternElement{Type: ElementQuoted, Value: inner})
		case len(tok) == 2 && tok[0] == '\\':
			// Backslash escape — literal character (use ElementQuoted to bypass glob interpretation).
			newElements = append(newElements, PatternElement{Type: ElementQuoted, Value: string(tok[1])})
		case tok == "**":
			newElements = append(newElements, PatternElement{Type: ElementDoubleWildcard, Value: tok})
		case tok == "*":
			newElements = append(newElements, PatternElement{Type: ElementWildcard, Value: tok})
		case isAllDots(tok):
			for range len(tok) {
				newElements = append(newElements, PatternElement{Type: ElementSingleChar, Value: "."})
			}
		case strings.HasPrefix(tok, "/") && strings.HasSuffix(tok, "/**"):
			re := tok[1 : len(tok)-3]
			re = strings.ReplaceAll(re, `\/`, "/")
			if _, err := regexp.Compile(re); err != nil {
				return nil, fmt.Errorf("invalid regex in pattern %q: %w", tok, err)
			}
			newElements = append(newElements, PatternElement{Type: ElementRegexMulti, Value: re})
		case len(tok) > 1 && tok[0] == '/' && tok[len(tok)-1] == '/':
			re := tok[1 : len(tok)-1]
			re = strings.ReplaceAll(re, `\/`, "/")
			if _, err := regexp.Compile(re); err != nil {
				return nil, fmt.Errorf("invalid regex in pattern %q: %w", tok, err)
			}
			newElements = append(newElements, PatternElement{Type: ElementRegex, Value: re})
		default:
			newElements = append(newElements, PatternElement{Type: ElementLiteral, Value: tok})
		}

		if nonPositional {
			for i := range newElements {
				newElements[i].NonPositional = true
			}
		}
		if optional {
			for i := range newElements {
				newElements[i].Optional = true
			}
		}
		elements = append(elements, newElements...)
	}

	return &ArgPattern{
		Raw:        raw,
		Elements:   elements,
		ExactMatch: exactMatch,
	}, nil
}

// processEscapes replaces backslash-escaped characters with their literal values.
func processEscapes(s string) string {
	var buf []byte
	i := 0
	for i < len(s) {
		if s[i] == '\\' && i+1 < len(s) {
			buf = append(buf, s[i+1])
			i += 2
			continue
		}
		buf = append(buf, s[i])
		i++
	}
	return string(buf)
}

// isAllDots returns true if s is non-empty and consists entirely of '.' characters.
func isAllDots(s string) bool {
	if len(s) == 0 {
		return false
	}
	for _, c := range s {
		if c != '.' {
			return false
		}
	}
	return true
}
