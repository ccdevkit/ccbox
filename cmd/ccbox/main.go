package main

import (
	"fmt"
	"os"

	"github.com/ccdevkit/ccbox/internal/args"
	"github.com/ccdevkit/ccbox/internal/bridge"
	"github.com/ccdevkit/ccbox/internal/constants"
	"github.com/ccdevkit/ccbox/internal/docker"
	"github.com/ccdevkit/ccbox/internal/logger"
)

// version is set via ldflags at build time: -ldflags "-X main.version=0.1.0"
var version = "dev"

// osFS implements args.FileSystem using the real filesystem.
type osFS struct{}

func (osFS) Stat(path string) (os.FileInfo, error) {
	return os.Stat(path)
}

func (osFS) ReadFile(path string) ([]byte, error) {
	return os.ReadFile(path)
}

func main() {
	parsed, err := args.Parse(os.Args[1:], osFS{})
	if err != nil {
		fmt.Fprintf(os.Stderr, "ccbox: %v\n", err)
		os.Exit(1)
	}

	if parsed.Version {
		fmt.Println(version)
		os.Exit(0)
	}
	if parsed.Help {
		printUsage()
		os.Exit(0)
	}

	switch parsed.Subcommand {
	case "update":
		runUpdate(parsed)
	case "clean":
		runClean(parsed)
	default:
		runDefault(parsed)
	}
}

func printUsage() {
	fmt.Print(`Usage: ccbox [flags] [subcommand] [-- claude-args...]

Flags:
  --version          Print version and exit
  --help, -h         Show this help message
  --verbose, -v      Enable verbose output
  --log <file>       Write debug logs to file
  --claudePath, -c   Path to Claude CLI binary
  --use <image>      Use a specific Docker image
  --passthrough <cmd>  Pass through a host command
  -pt:<cmd>          Shorthand for --passthrough <cmd>

Subcommands:
  update             Update the ccbox Docker image
  clean [--all]      Remove ccbox Docker images (--all removes all, default keeps latest)

Everything after "--" is forwarded to the Claude CLI.
`)
}

func runUpdate(_ *args.ParsedArgs) {
	// TODO: inject a real ImageManager once docker exec wrapper is wired up.
	err := doUpdate(constants.DefaultClaudePath, version, execUpdateRunner{}, execVersionRunner{}, nil)
	if err != nil {
		fmt.Fprintf(os.Stderr, "ccbox update: %v\n", err)
		os.Exit(1)
	}
}

func runClean(parsed *args.ParsedArgs) {
	mgr := docker.NewCLIImageManager()
	var err error
	if parsed.CleanAll {
		err = docker.CleanAllImages(mgr, os.Stderr, parsed.CleanForce)
	} else {
		err = docker.CleanImages(mgr, os.Stderr, parsed.CleanForce)
	}
	if err != nil {
		fmt.Fprintf(os.Stderr, "ccbox clean: %v\n", err)
		os.Exit(1)
	}
}

func runDefault(parsed *args.ParsedArgs) {
	log, err := logger.New(parsed.Verbose, parsed.LogFile)
	if err != nil {
		fmt.Fprintf(os.Stderr, "ccbox: %v\n", err)
		os.Exit(1)
	}
	defer log.Close()

	logHandler := bridge.NewLogHandler(log)
	deps := &orchestrationDeps{
		dockerChecker: realDockerChecker{},
		tokenCapture:  &realTokenCapturer{runner: realCLIRunner{}, log: log},
		versionDetect: &realVersionDetector{runner: execVersionRunner{}},
		imageEnsurer:  &realImageEnsurer{mgr: docker.NewCLIImageManager()},
		containerRunner: realContainerRunner{},
		bridgeServerFactory: func(execHandler bridge.ExecHandler) BridgeServer {
			return bridge.NewServer(execHandler, logHandler)
		},
		ccboxVersion: version,
		log:          log,
		fs:           osFS{},
	}

	os.Exit(runOrchestration(parsed, deps))
}
