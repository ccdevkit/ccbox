# TCP Protocol Contract: Host Server

## Transport

- **Binding**: `127.0.0.1:0` (dynamic port, localhost only)
- **Connection model**: One-shot. Each message opens a new TCP connection, sends the request, receives a response (if applicable), and closes.
- **No authentication**: Security relies entirely on localhost binding.

## Message Discrimination

The server uses a try-parse strategy on received data:
1. Attempt JSON parse with `type: "exec"` → exec handler
2. Attempt JSON parse with `type: "log"` → log handler

## Error Handling

- Unknown `type` values in valid JSON MUST be silently dropped (no response, connection closed).
- Malformed JSON (parse failure) MUST be silently dropped (no response, connection closed).

## Exec Request

**Direction**: Container → Host

**Request**:
```
{"type":"exec","command":"<shell command>","cwd":"<directory>"}\n
<TCP write-side close (CloseWrite)>
```

**Response**:
```
<exit_code>\n
<combined stdout+stderr bytes>
<TCP close>
```

**Behavior**:
- Commands execute via `sh -c` (Unix) or `cmd /C` (Windows)
- Working directory from request's `cwd` field, falls back to server's CWD
- No stdin piping — commands are non-interactive
- Container-side ccptproxy prepends `[NOTE: This command was run on the host machine]` to output
- Container paths in output are rewritten: `/home/claude/` is replaced with the actual host home directory path

## Log Request

**Direction**: Container → Host

**Request**:
```
{"type":"log","message":"[source] message text"}\n
<TCP close>
```

**Response**: None (fire-and-forget).

**Behavior**:
- 2-second dial timeout
- Failures silently swallowed
- Messages displayed on host stderr with prefix

