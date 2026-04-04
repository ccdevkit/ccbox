package matcher

import "strings"

// CommandMatcher determines whether a command string matches any of the
// configured passthrough command names using exact first-word matching.
type CommandMatcher struct {
	commands map[string]struct{}
}

// NewCommandMatcher creates a CommandMatcher for the given command names.
func NewCommandMatcher(commands []string) *CommandMatcher {
	m := &CommandMatcher{
		commands: make(map[string]struct{}, len(commands)),
	}
	for _, cmd := range commands {
		m.commands[cmd] = struct{}{}
	}
	return m
}

// Matches reports whether the first word of input exactly equals one of
// the configured command names. For example, with command "git":
// "git status" matches, "git" matches, "gitk" does not match.
func (m *CommandMatcher) Matches(input string) bool {
	firstWord := strings.Fields(input)
	if len(firstWord) == 0 {
		return false
	}
	_, ok := m.commands[firstWord[0]]
	return ok
}
