package args

import (
	"fmt"
	"os"
	"strings"
)

// FileSystem abstracts filesystem operations for testability.
type FileSystem interface {
	Stat(path string) (os.FileInfo, error)
	ReadFile(path string) ([]byte, error)
}

// ClaudeArg represents a single argument destined for the Claude CLI.
type ClaudeArg struct {
	Value  string
	IsFile bool
}

// ParsedArgs holds the fully parsed CLI input.
type ParsedArgs struct {
	Passthrough []string
	ClaudePath  string
	Use         string
	Verbose     bool
	LogFile     string
	Version     bool
	Help        bool
	Subcommand  string // "update", "clean", or ""
	CleanAll    bool   // clean --all: remove all images including latest
	CleanForce  bool   // clean --force: stop running containers before removing images
	ClaudeArgs  []ClaudeArg
}

// Claude CLI flags known to take a file path as their next argument.
var fileFlags = map[string]bool{
	"--system-prompt-file":        true,
	"--append-system-prompt-file": true,
	"--resume":                    true,
	"--mcp-config":                true,
}

// Parse parses the ccbox CLI arguments. Everything before "--" is ccbox flags;
// everything after is Claude CLI arguments.
func Parse(args []string, fs FileSystem) (*ParsedArgs, error) {
	result := &ParsedArgs{}

	// Split on "--"
	var ccboxArgs, claudeRaw []string
	for i, a := range args {
		if a == "--" {
			ccboxArgs = args[:i]
			claudeRaw = args[i+1:]
			goto parseCcbox
		}
	}
	ccboxArgs = args

parseCcbox:
	if err := parseCcboxFlags(ccboxArgs, result); err != nil {
		return nil, err
	}

	result.ClaudeArgs = classifyClaudeArgs(claudeRaw, fs)
	return result, nil
}

func parseCcboxFlags(args []string, result *ParsedArgs) error {
	for i := 0; i < len(args); i++ {
		a := args[i]

		switch {
		case a == "--verbose" || a == "-v":
			result.Verbose = true
		case a == "--version":
			result.Version = true
		case a == "--help" || a == "-h":
			result.Help = true
		case strings.HasPrefix(a, "-pt:"):
			cmd := strings.TrimPrefix(a, "-pt:")
			if cmd == "" {
				return fmt.Errorf("empty -pt: prefix value")
			}
			result.Passthrough = append(result.Passthrough, cmd)
		case a == "--passthrough":
			if i+1 >= len(args) {
				return fmt.Errorf("--passthrough requires a value")
			}
			i++
			result.Passthrough = append(result.Passthrough, args[i])
		case a == "--claudePath" || a == "-c":
			if i+1 >= len(args) {
				return fmt.Errorf("%s requires a value", a)
			}
			i++
			result.ClaudePath = args[i]
		case a == "--use":
			if i+1 >= len(args) {
				return fmt.Errorf("--use requires a value")
			}
			i++
			result.Use = args[i]
		case a == "--log":
			if i+1 >= len(args) {
				return fmt.Errorf("--log requires a value")
			}
			i++
			result.LogFile = args[i]
		case a == "--all":
			if result.Subcommand != "clean" {
				return fmt.Errorf("--all is only valid with the clean subcommand")
			}
			result.CleanAll = true
		case a == "--force":
			if result.Subcommand != "clean" {
				return fmt.Errorf("--force is only valid with the clean subcommand")
			}
			result.CleanForce = true
		case a == "update" || a == "clean":
			if result.Subcommand != "" {
				return fmt.Errorf("multiple subcommands: %q and %q", result.Subcommand, a)
			}
			result.Subcommand = a
		case strings.HasPrefix(a, "-"):
			return fmt.Errorf("unknown flag: %s", a)
		default:
			return fmt.Errorf("unknown argument: %s", a)
		}
	}
	return nil
}

func classifyClaudeArgs(raw []string, fs FileSystem) []ClaudeArg {
	result := make([]ClaudeArg, 0, len(raw))
	for i := 0; i < len(raw); i++ {
		a := raw[i]
		// Check if the next value is semantically known to be a file
		nextIsFile := fileFlags[a] && i+1 < len(raw)

		result = append(result, ClaudeArg{
			Value:  a,
			IsFile: isFileCandidate(a, false, fs),
		})

		if nextIsFile {
			i++
			result = append(result, ClaudeArg{
				Value:  raw[i],
				IsFile: isFileCandidate(raw[i], true, fs),
			})
		}
	}
	return result
}

// isFileCandidate determines if an argument refers to an existing file.
// semantic=true means the flag context already indicates this should be a file.
func isFileCandidate(arg string, semantic bool, fs FileSystem) bool {
	if strings.HasPrefix(arg, "-") {
		return false
	}

	candidate := semantic || looksLikePath(arg)
	if !candidate {
		return false
	}

	_, err := fs.Stat(arg)
	return err == nil
}

func looksLikePath(s string) bool {
	return strings.HasPrefix(s, "/") ||
		strings.HasPrefix(s, "./") ||
		strings.HasPrefix(s, "../") ||
		strings.HasPrefix(s, "~/")
}
