#!/bin/bash
set -e

# TDD exemption: Shell script tested via integration only (Principle VII amendment).

# Debug helper: pipes a message through ccdebug to the host bridge.
debug_log() {
    if [ -n "$CCBOX_TCP_PORT" ]; then
        echo "$1" | ccdebug 2>/dev/null || true
    fi
}

debug_log "entrypoint: starting (args: $*)"

# Run ccptproxy --setup to install command hijacker scripts if TCP bridge is active
if [ -n "$CCBOX_TCP_PORT" ]; then
    debug_log "entrypoint: running ccptproxy --setup"
    ccptproxy --setup
    debug_log "entrypoint: ccptproxy --setup done"
fi

# Start Xvfb on :99 in background for clipboard access if DISPLAY is set
if [ -n "$DISPLAY" ]; then
    debug_log "entrypoint: starting Xvfb on $DISPLAY"
    Xvfb "$DISPLAY" -screen 0 1024x768x24 -nolisten tcp &
fi

# Start clipboard daemon in background if clip port is configured
if [ -n "$CCBOX_CLIP_PORT" ]; then
    debug_log "entrypoint: starting ccclipd"
    ccclipd &
fi

debug_log "entrypoint: which claude=$(which claude 2>&1)"

# If --debug-file is in the args, tail it through ccdebug so debug logs
# reach the host. Touch the file first so tail -f doesn't fail.
CLAUDE_DEBUG_FILE="/opt/ccbox/claude-debug.log"
for arg in "$@"; do
    if [ "$arg" = "--debug-file" ]; then
        touch "$CLAUDE_DEBUG_FILE"
        chown claude:claude "$CLAUDE_DEBUG_FILE"
        tail -f "$CLAUDE_DEBUG_FILE" 2>/dev/null | ccdebug --prefix claude &
        debug_log "entrypoint: tailing $CLAUDE_DEBUG_FILE through ccdebug"
        break
    fi
done

debug_log "entrypoint: exec gosu claude claude $*"

# Drop from root to unprivileged claude user (UID 1001) per FR-020 and exec Claude Code
exec gosu claude claude "$@"
