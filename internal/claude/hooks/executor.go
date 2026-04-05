package hooks

import (
	"bytes"
	"context"
	"encoding/json"
	"os/exec"
	"sync"

	"github.com/ccdevkit/ccbox/internal/logger"
)

// CommandRunner abstracts subprocess execution for testability.
type CommandRunner interface {
	Run(ctx context.Context, command string, stdin []byte, env []string, dir string) (exitCode int, stdout, stderr []byte, err error)
}

// ShellRunner executes commands via "sh -c".
type ShellRunner struct{}

// Run executes a shell command with the given stdin, environment, and working directory.
func (ShellRunner) Run(ctx context.Context, command string, stdin []byte, env []string, dir string) (int, []byte, []byte, error) {
	cmd := exec.CommandContext(ctx, "sh", "-c", command)
	cmd.Stdin = bytes.NewReader(stdin)
	cmd.Env = env
	cmd.Dir = dir

	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr

	err := cmd.Run()
	exitCode := 0
	if err != nil {
		if exitErr, ok := err.(*exec.ExitError); ok {
			exitCode = exitErr.ExitCode()
		} else {
			return 1, nil, []byte(err.Error()), nil
		}
	}
	return exitCode, stdout.Bytes(), stderr.Bytes(), nil
}

// ExecuteUserHooks filters, runs, and aggregates user-defined hook commands.
//
// Filtering: each hook is checked against the event's matcher and if-field.
// Execution: all matching hooks run in parallel.
// Aggregation: uses Claude Code's precedence rules:
//
//	continue:false > exit 2 (block) > exit 1 (error) > exit 0
//	For exit 0, decision priority: deny > defer > ask > allow
func ExecuteUserHooks(
	ctx context.Context,
	runner CommandRunner,
	log *logger.Logger,
	hooks []UserHook,
	event HookEvent,
	input json.RawMessage,
	env []string,
	dir string,
) *HandlerResult {
	if len(hooks) == 0 {
		return &HandlerResult{ExitCode: 0}
	}

	// Filter by matcher and if-field
	var matching []UserHook
	for _, h := range hooks {
		if !matchesUserHook(h, event, input) {
			debugLog(log, "user hook filtered by matcher: cmd=%q matcher=%q", h.Command, h.Matcher)
			continue
		}
		if !MatchIf(h.If, event, input) {
			debugLog(log, "user hook filtered by if-field: cmd=%q if=%q", h.Command, h.If)
			continue
		}
		matching = append(matching, h)
	}

	if len(matching) == 0 {
		debugLog(log, "all %d user hooks filtered out for %s", len(hooks), event)
		return &HandlerResult{ExitCode: 0}
	}

	debugLog(log, "executing %d/%d matching user hooks in parallel for %s", len(matching), len(hooks), event)

	// Run all matching hooks in parallel
	results := make([]*HandlerResult, len(matching))
	var wg sync.WaitGroup
	for i, h := range matching {
		wg.Add(1)
		go func(idx int, hook UserHook) {
			defer wg.Done()
			debugLog(log, "user hook start: cmd=%q", hook.Command)
			exitCode, stdout, stderr, err := runner.Run(ctx, hook.Command, []byte(input), env, dir)
			if err != nil {
				debugLog(log, "user hook error: cmd=%q err=%v", hook.Command, err)
				results[idx] = &HandlerResult{ExitCode: 1, Stderr: []byte(err.Error())}
				return
			}
			debugLog(log, "user hook done: cmd=%q exit=%d stdout=%d bytes", hook.Command, exitCode, len(stdout))
			results[idx] = &HandlerResult{ExitCode: exitCode, Stdout: stdout, Stderr: stderr}
		}(i, h)
	}
	wg.Wait()

	result := aggregateUserHookResults(results)
	debugLog(log, "user hooks aggregated: exit=%d decision=%q", result.ExitCode, extractDecision(result.Stdout))
	return result
}

// debugLog writes a debug log message if the logger is set.
func debugLog(log *logger.Logger, format string, args ...interface{}) {
	if log != nil {
		log.Debug("hooks", format, args...)
	}
}

// aggregateUserHookResults combines results from parallel user hook execution
// using Claude Code's precedence rules.
func aggregateUserHookResults(results []*HandlerResult) *HandlerResult {
	var (
		stopResult  *HandlerResult
		blockResult *HandlerResult
		errorResult *HandlerResult
		bestResult  *HandlerResult
		bestPri     int
	)

	for _, r := range results {
		if r == nil {
			continue
		}

		// continue:false takes highest precedence
		if stopResult == nil {
			if stop, reason := extractContinueFalse(r.Stdout); stop {
				stopResult = buildStopResult(reason)
			}
		}

		switch r.ExitCode {
		case 2:
			if blockResult == nil {
				blockResult = r
			}
		case 0:
			pri := decisionPriority[extractDecision(r.Stdout)]
			if bestResult == nil || pri > bestPri {
				bestResult = r
				bestPri = pri
			}
		default:
			if errorResult == nil {
				errorResult = r
			}
		}
	}

	if stopResult != nil {
		return stopResult
	}
	if blockResult != nil {
		return blockResult
	}
	if errorResult != nil {
		return errorResult
	}
	if bestResult != nil {
		return bestResult
	}
	return &HandlerResult{ExitCode: 0}
}
