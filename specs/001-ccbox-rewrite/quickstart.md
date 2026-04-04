# Quickstart: ccbox Development

## Prerequisites

- Go 1.24+ (`go version`)
- Docker Desktop or Docker Engine running (`docker info`)
- Claude Code CLI installed and authenticated (`claude --version`)
- Node.js 22+ (for npm distribution testing only)

## Clone and Build

```bash
git clone <repo-url> && cd ccbox
git checkout 001-ccbox-rewrite
make build        # Builds bin/ccbox
make install      # Copies to /usr/local/bin/ccbox
```

## Run Tests

```bash
go test ./...     # All unit tests
go test ./internal/args/...          # Single package
go test -v ./cmd/ccptproxy/matcher/... # Verbose matcher tests
go test -run TestCommandMatcher ./cmd/ccptproxy/matcher/...  # Single test
```

## Build Docker Image (for integration testing)

```bash
make docker       # Builds base image as ghcr.io/ccdevkit/ccbox-base:dev
```

## Run ccbox Locally

```bash
# Interactive session
make run

# With arguments
make run ARGS="-- -p 'hello'"

# With passthrough
make run ARGS="-pt:git -- -p 'run git status'"

# Debug mode
make run ARGS="-v -- -p 'hello'"
```

## Project Layout

```
cmd/ccbox/        # Host CLI binary
cmd/ccptproxy/      # Container proxy binary
cmd/ccclipd/      # Container clipboard daemon
cmd/ccdebug/      # Container debug log forwarder — reads Claude's debug log inside the container and sends entries to the host via TCP
internal/         # All business logic (11 internal packages + 4 command binaries)
Dockerfile        # Multi-stage container build
entrypoint.sh     # Container init script
Makefile          # Build, test, and run targets
npm/              # npm packaging for distribution
.github/workflows/ # CI/CD pipelines
```

## Development Workflow (TDD)

Per constitution Principle VII, all new code follows Red-Green-Refactor:

```bash
# 1. RED: Write failing test
go test ./internal/args/... -run TestNewFunction
# FAIL

# 2. GREEN: Implement minimum code
# Edit internal/args/args.go
go test ./internal/args/... -run TestNewFunction
# PASS

# 3. REFACTOR: Clean up
go test ./internal/args/...
# ALL PASS
```

## Key Architecture Concepts

1. **Host binary** (`ccbox`): Orchestrates everything — captures OAuth, builds images, bridges PTY
2. **Container proxy** (`ccptproxy`): Intercepts commands via PATH hijacking, routes to host via TCP
3. **Identity path mapping**: CWD is mounted at the same path inside the container
4. **Three execution paths**: PTY mode (TTY detected — raw terminal mode, stdin interceptor for paste detection, PTY bridge between host and container, resize signal forwarding), interceptor mode (no TTY, piped — stdin interceptor active for paste detection but no PTY or raw terminal), simple mode (direct stdin/stdout passthrough, no clipboard support)
5. **Settings cascade**: CLI flags > project `.ccbox/settings.json` > global `~/.ccbox/settings.json`
6. **System prompt injection**: When passthrough commands are configured, ccbox injects a system prompt (FR-024) instructing Claude how to use the proxy for whitelisted commands.
7. **Clipboard image bridging**: Clipboard images are bridged from host to container via a daemon (ccclipd). On paste, the host transcodes images to PNG and writes them to a shared directory; the stdin interceptor rewrites paths so Claude can access them.

## Common Tasks

| Task | Command |
|------|---------|
| Build binary | `make build` |
| Run tests | `go test ./...` |
| Build Docker image | `make docker` |
| Run with args | `make run ARGS="..."` |
| Clean build artifacts | `make clean` |
