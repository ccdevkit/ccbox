package main

import (
	"fmt"
	"os"
	"strings"

	"github.com/ccdevkit/ccbox/cmd/ccptproxy/matcher"
	"github.com/ccdevkit/ccbox/internal/constants"
)

// execSender is the function signature for sending exec requests to the host.
type execSender func(address, command, cwd string) (exitCode int, output []byte, err error)

func main() {
	if len(os.Args) < 2 {
		fmt.Fprintln(os.Stderr, "usage: ccptproxy --setup | --exec <command> [args...]")
		os.Exit(1)
	}

	configPath := os.Getenv("CCBOX_PROXY_CONFIG")
	if configPath == "" {
		configPath = constants.ProxyConfigContainerPath
	}

	switch os.Args[1] {
	case "--setup":
		if _, err := os.Stat(configPath); os.IsNotExist(err) {
			// No proxy config — nothing to set up.
			os.Exit(0)
		}
		if err := runSetup(configPath); err != nil {
			fmt.Fprintf(os.Stderr, "ccptproxy: setup failed: %v\n", err)
			os.Exit(1)
		}

	case "--exec":
		if len(os.Args) < 3 {
			fmt.Fprintln(os.Stderr, "usage: ccptproxy --exec <command> [args...]")
			os.Exit(1)
		}
		// Reconstruct the full command from args after --exec
		command := strings.Join(os.Args[2:], " ")
		stdout, exitCode, err := runExec(configPath, command, SendExec)
		if err != nil {
			fmt.Fprintf(os.Stderr, "ccptproxy: exec failed: %v\n", err)
			os.Exit(1)
		}
		os.Stdout.Write(stdout)
		os.Exit(exitCode)

	default:
		fmt.Fprintf(os.Stderr, "ccptproxy: unknown flag %q\n", os.Args[1])
		os.Exit(1)
	}
}

// runSetup reads the proxy config and generates hijacker shim scripts for each
// passthrough command in the shims directory.
func runSetup(configPath string) error {
	return runSetupWithDir(configPath, constants.ContainerShimsDir)
}

// runSetupWithDir is the testable core of runSetup that accepts a custom shims directory.
func runSetupWithDir(configPath, shimsDir string) error {
	cfg, err := ReadConfig(configPath)
	if err != nil {
		return fmt.Errorf("reading config: %w", err)
	}

	if err := os.MkdirAll(shimsDir, 0755); err != nil {
		return fmt.Errorf("creating shims dir: %w", err)
	}

	for _, cmd := range cfg.Passthrough {
		if err := GenerateHijacker(shimsDir, cmd); err != nil {
			return fmt.Errorf("generating hijacker for %q: %w", cmd, err)
		}
	}

	return nil
}

// runExec reads the proxy config, matches the command, sends it to the host,
// and returns the annotated output and exit code.
func runExec(configPath, command string, sender execSender) ([]byte, int, error) {
	cfg, err := ReadConfig(configPath)
	if err != nil {
		return nil, 0, fmt.Errorf("reading config: %w", err)
	}

	m := matcher.NewCommandMatcher(cfg.Passthrough)
	if !m.Matches(command) {
		return nil, 0, fmt.Errorf("command %q does not match any passthrough command", command)
	}

	cwd, err := os.Getwd()
	if err != nil {
		cwd = "/"
	}

	exitCode, output, err := sender(cfg.HostAddress, command, cwd)
	if err != nil {
		return nil, 0, fmt.Errorf("sending exec request: %w", err)
	}

	// Prepend the note annotation
	annotated := append([]byte(constants.PassthroughNote+"\n"), output...)

	// Log if verbose (best effort, don't fail on log errors)
	if cfg.Verbose {
		_ = SendLog(cfg.HostAddress, fmt.Sprintf("[ccptproxy] exec %q -> exit %d", command, exitCode))
	}

	return annotated, exitCode, nil
}