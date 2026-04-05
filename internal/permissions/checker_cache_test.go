package permissions

import (
	"fmt"
	"sync"
	"sync/atomic"
	"testing"
	"time"
)

func TestLiveChecker_ReloadsAfterTTL(t *testing.T) {
	var callCount atomic.Int32

	config1 := &PermissionsConfig{
		Passthrough: map[string]*CommandPermission{
			"git": {Rules: []Rule{{
				Pattern: PatternOrArray{Values: []string{"status"}},
				Effect:  "allow",
			}}},
		},
	}
	config2 := &PermissionsConfig{
		Passthrough: map[string]*CommandPermission{
			"git": {Rules: []Rule{{
				Pattern: PatternOrArray{Values: []string{"**"}},
				Effect:  "allow",
			}}},
		},
	}

	c := &Checker{
		loadFunc: func() (*PermissionsConfig, error) {
			n := callCount.Add(1)
			if n == 1 {
				return config1, nil
			}
			return config2, nil
		},
		cacheTTL: 10 * time.Millisecond,
	}
	if err := c.reload(); err != nil {
		t.Fatal(err)
	}

	// First check: only "status" is allowed.
	r := c.Check("git status")
	if !r.Allowed {
		t.Fatalf("expected allowed, got: %s", r.Reason)
	}
	r = c.Check("git push")
	if r.Allowed {
		t.Fatal("expected denied for 'git push' with config1")
	}

	// Wait for TTL to expire.
	time.Sleep(20 * time.Millisecond)

	// After reload: "**" allows everything.
	r = c.Check("git push")
	if !r.Allowed {
		t.Fatalf("expected allowed after reload, got: %s", r.Reason)
	}

	if callCount.Load() < 2 {
		t.Fatalf("expected at least 2 loads, got %d", callCount.Load())
	}
}

func TestLiveChecker_MalformedConfigKeepsLastGood(t *testing.T) {
	var callCount atomic.Int32

	goodConfig := &PermissionsConfig{
		Passthrough: map[string]*CommandPermission{
			"git": {Rules: []Rule{{
				Pattern: PatternOrArray{Values: []string{"status"}},
				Effect:  "allow",
			}}},
		},
	}

	c := &Checker{
		loadFunc: func() (*PermissionsConfig, error) {
			n := callCount.Add(1)
			if n == 1 {
				return goodConfig, nil
			}
			return nil, fmt.Errorf("config parse error")
		},
		cacheTTL: 10 * time.Millisecond,
	}
	if err := c.reload(); err != nil {
		t.Fatal(err)
	}

	r := c.Check("git status")
	if !r.Allowed {
		t.Fatalf("expected allowed, got: %s", r.Reason)
	}

	time.Sleep(20 * time.Millisecond)

	// After failed reload, last good config still works.
	r = c.Check("git status")
	if !r.Allowed {
		t.Fatalf("expected allowed after failed reload, got: %s", r.Reason)
	}
}

func TestLiveChecker_ConfigRemovedRevertsToCliOnly(t *testing.T) {
	var callCount atomic.Int32

	fileConfig := &PermissionsConfig{
		Passthrough: map[string]*CommandPermission{
			"npm": {Rules: []Rule{{
				Pattern: PatternOrArray{Values: []string{"install"}},
				Effect:  "allow",
			}}},
		},
	}

	c := &Checker{
		cliPassthrough: []string{"git"},
		loadFunc: func() (*PermissionsConfig, error) {
			n := callCount.Add(1)
			if n == 1 {
				return fileConfig, nil
			}
			return nil, nil // config file removed
		},
		cacheTTL: 10 * time.Millisecond,
	}
	if err := c.reload(); err != nil {
		t.Fatal(err)
	}

	// Both commands work initially.
	r := c.Check("git status")
	if !r.Allowed {
		t.Fatalf("expected git allowed, got: %s", r.Reason)
	}
	r = c.Check("npm install")
	if !r.Allowed {
		t.Fatalf("expected npm allowed, got: %s", r.Reason)
	}

	time.Sleep(20 * time.Millisecond)

	// After config removed: git still works (CLI), npm is gone.
	r = c.Check("git status")
	if !r.Allowed {
		t.Fatalf("expected git still allowed, got: %s", r.Reason)
	}
	r = c.Check("npm install")
	if r.Allowed {
		t.Fatal("expected npm denied after config removed")
	}
}

func TestLiveChecker_ConcurrentSafety(t *testing.T) {
	var toggle atomic.Bool

	config1 := &PermissionsConfig{
		Passthrough: map[string]*CommandPermission{
			"git": {Rules: []Rule{{
				Pattern: PatternOrArray{Values: []string{"**"}},
				Effect:  "allow",
			}}},
		},
	}
	config2 := &PermissionsConfig{
		Passthrough: map[string]*CommandPermission{
			"git": {Rules: []Rule{{
				Pattern: PatternOrArray{Values: []string{"status"}},
				Effect:  "allow",
			}}},
		},
	}

	c := &Checker{
		loadFunc: func() (*PermissionsConfig, error) {
			if toggle.Load() {
				return config2, nil
			}
			return config1, nil
		},
		cacheTTL: 1 * time.Millisecond,
	}
	if err := c.reload(); err != nil {
		t.Fatal(err)
	}

	var wg sync.WaitGroup
	for i := 0; i < 20; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for j := 0; j < 100; j++ {
				c.Check("git status")
				c.Commands()
				c.HasCommand("git")
			}
		}()
	}

	// Toggle config mid-flight.
	go func() {
		time.Sleep(2 * time.Millisecond)
		toggle.Store(true)
	}()

	wg.Wait()
	// If we get here without panic or race detector complaints, we pass.
}

func TestStaticChecker_DoesNotReload(t *testing.T) {
	config := &PermissionsConfig{
		Passthrough: map[string]*CommandPermission{
			"git": {Rules: []Rule{{
				Pattern: PatternOrArray{Values: []string{"**"}},
				Effect:  "allow",
			}}},
		},
	}

	c, err := NewChecker(config, nil)
	if err != nil {
		t.Fatal(err)
	}

	if c.loadFunc != nil {
		t.Fatal("expected loadFunc to be nil for static checker")
	}

	r := c.Check("git status")
	if !r.Allowed {
		t.Fatalf("expected allowed, got: %s", r.Reason)
	}
}

func TestLiveChecker_CommandsReflectReload(t *testing.T) {
	var callCount atomic.Int32

	config1 := &PermissionsConfig{
		Passthrough: map[string]*CommandPermission{
			"git": {Rules: []Rule{{
				Pattern: PatternOrArray{Values: []string{"**"}},
				Effect:  "allow",
			}}},
		},
	}
	config2 := &PermissionsConfig{
		Passthrough: map[string]*CommandPermission{
			"git": {Rules: []Rule{{
				Pattern: PatternOrArray{Values: []string{"**"}},
				Effect:  "allow",
			}}},
			"npm": {Rules: []Rule{{
				Pattern: PatternOrArray{Values: []string{"**"}},
				Effect:  "allow",
			}}},
		},
	}

	c := &Checker{
		loadFunc: func() (*PermissionsConfig, error) {
			n := callCount.Add(1)
			if n == 1 {
				return config1, nil
			}
			return config2, nil
		},
		cacheTTL: 10 * time.Millisecond,
	}
	if err := c.reload(); err != nil {
		t.Fatal(err)
	}

	cmds := c.Commands()
	if len(cmds) != 1 || cmds[0] != "git" {
		t.Fatalf("expected [git], got %v", cmds)
	}
	if c.HasCommand("npm") {
		t.Fatal("expected npm not present initially")
	}

	time.Sleep(20 * time.Millisecond)

	cmds = c.Commands()
	if len(cmds) != 2 {
		t.Fatalf("expected 2 commands after reload, got %v", cmds)
	}
	if !c.HasCommand("npm") {
		t.Fatal("expected npm present after reload")
	}
}
