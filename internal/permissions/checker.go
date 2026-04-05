package permissions

import (
	"fmt"
	"log"
	"sort"
	"strings"
	"sync"
	"sync/atomic"
	"time"
)

// checkerSnapshot is an immutable compiled rule state that can be atomically swapped.
type checkerSnapshot struct {
	commands map[string][]*CompiledRule // command name → compiled rules (nil = unrestricted)
}

// Checker evaluates commands against permission rules.
// When created via NewLiveChecker, it automatically reloads the config
// from disk when the cache TTL expires (default 1s).
type Checker struct {
	// Immutable after construction.
	cliPassthrough []string
	loadFunc       func() (*PermissionsConfig, error) // nil = static (no reload)
	cacheTTL       time.Duration

	// Atomic snapshot for lock-free reads.
	snap atomic.Pointer[checkerSnapshot]

	// Serializes reload attempts so only one goroutine reloads at a time.
	reloadMu sync.Mutex
	lastLoad atomic.Int64 // UnixNano timestamp of last load
}

// NewChecker creates a static permission checker from a loaded config and optional
// CLI passthrough commands. CLI commands contribute an implicit allow-all
// rule as the first rule in each command's cascade.
// Compiles and validates all patterns at construction time.
// The checker does not auto-reload; use NewLiveChecker for that.
func NewChecker(config *PermissionsConfig, cliPassthrough []string) (*Checker, error) {
	snap, err := buildSnapshot(config, cliPassthrough)
	if err != nil {
		return nil, err
	}
	c := &Checker{
		cliPassthrough: cliPassthrough,
		// loadFunc nil = no auto-reload
	}
	c.snap.Store(snap)
	return c, nil
}

// NewLiveChecker creates a permission checker that automatically reloads
// permissions from disk when the cache expires (default 1s). cliPassthrough
// rules are static and merged on every reload.
func NewLiveChecker(cliPassthrough []string) (*Checker, error) {
	c := &Checker{
		cliPassthrough: cliPassthrough,
		loadFunc:       Load,
		cacheTTL:       time.Second,
	}
	if err := c.reload(); err != nil {
		return nil, err
	}
	return c, nil
}

// buildSnapshot compiles a PermissionsConfig and CLI passthrough args into
// an immutable checkerSnapshot.
func buildSnapshot(config *PermissionsConfig, cliPassthrough []string) (*checkerSnapshot, error) {
	commands := make(map[string][]*CompiledRule)

	// Process CLI passthrough commands first — each gets an implicit "allow **".
	for _, cmd := range cliPassthrough {
		pattern, err := ParsePattern("**")
		if err != nil {
			return nil, err
		}
		commands[cmd] = []*CompiledRule{{
			Pattern: pattern,
			Effect:  EffectAllow,
		}}
	}

	// Process file config.
	if config != nil {
		for name, cmdPerm := range config.Passthrough {
			if cmdPerm == nil {
				// null value = unrestricted (allow all).
				commands[name] = nil
				continue
			}
			if len(cmdPerm.Rules) == 0 {
				// Explicit empty rules array = unrestricted.
				commands[name] = nil
				continue
			}
			for _, rule := range cmdPerm.Rules {
				for _, patStr := range rule.Pattern.Values {
					pattern, err := ParsePattern(patStr)
					if err != nil {
						return nil, fmt.Errorf("command %q: %w", name, err)
					}
					compiled := &CompiledRule{
						Pattern: pattern,
						Effect:  Effect(rule.Effect),
						Reason:  rule.Reason,
					}
					commands[name] = append(commands[name], compiled)
				}
			}
		}
	}

	return &checkerSnapshot{commands: commands}, nil
}

// snapshot returns the current rule state, reloading from disk if the
// cache has expired. Non-blocking for concurrent readers.
func (c *Checker) snapshot() *checkerSnapshot {
	if c.loadFunc == nil {
		return c.snap.Load()
	}

	// Fast path: cache is fresh.
	if time.Since(time.Unix(0, c.lastLoad.Load())) < c.cacheTTL {
		return c.snap.Load()
	}

	// Slow path: try to reload. Only one goroutine reloads at a time.
	if c.reloadMu.TryLock() {
		defer c.reloadMu.Unlock()
		// Double-check after acquiring lock.
		if time.Since(time.Unix(0, c.lastLoad.Load())) >= c.cacheTTL {
			_ = c.reload()
		}
	}

	return c.snap.Load()
}

// reload loads config from disk and swaps the snapshot.
// Caller must hold c.reloadMu (or be in the constructor).
// On load error, logs and keeps the previous snapshot.
func (c *Checker) reload() error {
	config, err := c.loadFunc()
	if err != nil {
		log.Printf("permissions: reload error (keeping previous config): %v", err)
		c.lastLoad.Store(time.Now().UnixNano())
		return err
	}
	snap, err := buildSnapshot(config, c.cliPassthrough)
	if err != nil {
		log.Printf("permissions: compile error (keeping previous config): %v", err)
		c.lastLoad.Store(time.Now().UnixNano())
		return err
	}
	c.snap.Store(snap)
	c.lastLoad.Store(time.Now().UnixNano())
	return nil
}

// HasCommand reports whether the checker has any rules (or unrestricted access) for a command.
func (c *Checker) HasCommand(name string) bool {
	_, ok := c.snapshot().commands[name]
	return ok
}

// Check evaluates a full command string against the checker's rules.
// It splits the command into name + args, looks up the command, and
// evaluates rules using last-match-wins semantics.
func (c *Checker) Check(command string) MatchResult {
	snap := c.snapshot()

	fields := strings.Fields(command)
	if len(fields) == 0 {
		return MatchResult{Allowed: false, Reason: "empty command", Command: command}
	}
	name := fields[0]
	args := fields[1:]

	rules, exists := snap.commands[name]
	if !exists {
		return MatchResult{Allowed: false, Reason: fmt.Sprintf("command not configured: %s", name), Command: command}
	}

	// nil or empty rules = unrestricted.
	if rules == nil || len(rules) == 0 {
		return MatchResult{Allowed: true, Reason: "unrestricted", Command: command}
	}

	// Evaluate all rules, track the last match.
	var lastMatch *CompiledRule
	for _, rule := range rules {
		if MatchPattern(rule.Pattern, args) {
			lastMatch = rule
		}
	}

	if lastMatch == nil {
		var patterns []string
		for _, r := range rules {
			patterns = append(patterns, r.Pattern.Raw)
		}
		return MatchResult{
			Allowed: false,
			Reason:  fmt.Sprintf("no matching rule (fail-closed); available patterns: %s", strings.Join(patterns, ", ")),
			Command: command,
		}
	}

	if lastMatch.Effect == EffectAllow {
		return MatchResult{Allowed: true, Reason: lastMatch.Pattern.Raw, MatchedRule: lastMatch, Command: command}
	}

	reason := lastMatch.Reason
	if reason == "" {
		reason = fmt.Sprintf("blocked by pattern: %s", lastMatch.Pattern.Raw)
	}
	return MatchResult{Allowed: false, Reason: reason, MatchedRule: lastMatch, Command: command}
}

// Commands returns a sorted, deduplicated list of command names known to the checker.
func (c *Checker) Commands() []string {
	snap := c.snapshot()
	names := make([]string, 0, len(snap.commands))
	for name := range snap.commands {
		names = append(names, name)
	}
	sort.Strings(names)
	return names
}
