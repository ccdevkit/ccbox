# Contract: Hook Bridge Protocol

**Extension to**: `internal/bridge`

## Wire Types

### HookRequest (container → host)

```json
{
  "type": "hook",
  "event": "PreToolUse",
  "input": { ... raw hook input JSON ... }
}
```

### HookResponse (host → container)

Response is a JSON object sent as a single line on the TCP connection:

```json
{
  "exit_code": 0,
  "stdout": "{ ... JSON output ... }",
  "stderr": ""
}
```

- `exit_code`: integer (0, 2, or other)
- `stdout`: JSON string (may be empty; parsed by Claude Code when exit 0)
- `stderr`: error text (may be empty; shown by Claude Code when exit 2)

Unlike exec responses (which use newline-delimited fields), hook responses use JSON for structured parsing and self-documentation.

## Handler Type

```go
// HookHandler processes a hook request and returns exit code, stdout, and stderr.
type HookHandler func(req HookRequest) (exitCode int, stdout []byte, stderr []byte)
```

## Server Changes

The `Server` struct gains:
- A `hookHandler HookHandler` field
- A new `case constants.HookRequestType:` in the `handleConn` switch
- Constructor updated: `NewServer(exec, log, hook)`

## Container Proxy Binary: `cchookproxy`

**Location**: `cmd/cchookproxy/`

**Invocation**: Claude Code runs the configured hook command. The proxy reads stdin, sends a `HookRequest` over TCP, receives a `HookResponse`, and exits.

**Behavior**:
1. Read all of stdin (hook input JSON)
2. Parse `hook_event_name` from the JSON to get the event name
3. Dial `{DOCKER_HOSTNAME}:{CCBOX_TCP_PORT}` with 2-second timeout
4. Send `HookRequest` as newline-delimited JSON
5. Close write side of connection
6. Read response with 10-second timeout
7. Parse exit code from first line
8. Write stdout portion to os.Stdout
9. Write stderr portion to os.Stderr
10. Exit with the parsed exit code

**Failure modes**:
- TCP dial fails → exit 1 (non-blocking), stderr: "cchookproxy: connection failed: {error}"
- Response timeout → exit 1 (non-blocking), stderr: "cchookproxy: response timeout"
- Malformed response → exit 1 (non-blocking), stderr: "cchookproxy: invalid response"
