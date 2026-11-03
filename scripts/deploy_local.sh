#!/usr/bin/env bash
set -Eeuo pipefail

# Usage: bash scripts/deploy_local.sh <binary> <deployment-dir> [port] [staging-port]
# The old release is preserved until the new release passes its health check.
BINARY="${1:-./dist/stockpilot}"
BASE="${2:-./production}"
PROD_PORT="${3:-8080}"
STAGING_PORT="${4:-18080}"

if [[ ! -x "$BINARY" ]]; then
    echo "Deployment stopped: executable not found: $BINARY" >&2
    exit 1
fi
BINARY="$(cd "$(dirname "$BINARY")" && pwd)/$(basename "$BINARY")"
mkdir -p "$BASE/releases" "$BASE/shared" "$BASE/logs"
BASE="$(cd "$BASE" && pwd)"
LOG="$BASE/logs/deploy.log"

log() {
    local line="$(date -u '+%Y-%m-%dT%H:%M:%SZ') $*"
    echo "$line" | tee -a "$LOG"
}

health_ok() {
    local port="$1"
    local n
    for n in {1..25}; do
        if curl --silent --fail --max-time 2 "http://127.0.0.1:${port}/healthz" \
            | grep -q '"status":"ok"'; then
            return 0
        fi
        sleep 0.2
    done
    return 1
}

stop_process() {
    local pid="$1"
    if [[ "$pid" =~ ^[0-9]+$ ]] && kill -0 "$pid" 2>/dev/null; then
        kill "$pid" 2>/dev/null || true
        for i in {1..20}; do
            if ! kill -0 "$pid" 2>/dev/null; then break; fi
            sleep 0.1
        done
    fi
}

CURRENT_SLOT=""
if [[ -L "$BASE/current" ]]; then
    CURRENT_SLOT="$(basename "$(readlink "$BASE/current")")"
fi
if [[ "$CURRENT_SLOT" == "blue" ]]; then
    NEXT_SLOT="green"
else
    NEXT_SLOT="blue"
fi

NEXT_BINARY="$BASE/releases/$NEXT_SLOT/stockpilot"
mkdir -p "$(dirname "$NEXT_BINARY")"
cp "$BINARY" "$NEXT_BINARY"
chmod +x "$NEXT_BINARY"
log "Preparing $NEXT_SLOT release (previous: ${CURRENT_SLOT:-none})"

# Validate the candidate on another port, keeping the active service untouched.
STOCKPILOT_ADDR="127.0.0.1:$STAGING_PORT" \
STOCKPILOT_DATA="$BASE/shared/items.json" \
    nohup "$NEXT_BINARY" >> "$BASE/logs/staging.log" 2>&1 < /dev/null &
STAGING_PID=$!
if ! health_ok "$STAGING_PORT"; then
    stop_process "$STAGING_PID"
    log "FAILED: candidate did not pass health check; active release kept (automatic rollback)"
    exit 1
fi
stop_process "$STAGING_PID"
log "Candidate health check OK"

OLD_PID=""
if [[ -f "$BASE/app.pid" ]]; then
    OLD_PID="$(cat "$BASE/app.pid")"
fi
# Atomic symlink replacement prevents a partially-written current link.
ln -s "releases/$NEXT_SLOT" "$BASE/.next-release"
mv -Tf "$BASE/.next-release" "$BASE/current"

if [[ -n "$OLD_PID" ]]; then stop_process "$OLD_PID"; fi
STOCKPILOT_ADDR="127.0.0.1:$PROD_PORT" \
STOCKPILOT_DATA="$BASE/shared/items.json" \
    nohup "$BASE/current/stockpilot" >> "$BASE/logs/server.log" 2>&1 < /dev/null &
NEW_PID=$!
printf '%s\n' "$NEW_PID" > "$BASE/app.pid"

if health_ok "$PROD_PORT"; then
    log "DEPLOYED: $NEXT_SLOT is now active at http://127.0.0.1:$PROD_PORT"
    exit 0
fi

# If the cutover fails, restore both the previous symlink and process.
stop_process "$NEW_PID"
if [[ -n "$CURRENT_SLOT" && -x "$BASE/releases/$CURRENT_SLOT/stockpilot" ]]; then
    ln -s "releases/$CURRENT_SLOT" "$BASE/.next-release"
    mv -Tf "$BASE/.next-release" "$BASE/current"
    STOCKPILOT_ADDR="127.0.0.1:$PROD_PORT" \
    STOCKPILOT_DATA="$BASE/shared/items.json" \
        nohup "$BASE/current/stockpilot" >> "$BASE/logs/server.log" 2>&1 < /dev/null &
    printf '%s\n' "$!" > "$BASE/app.pid"
    if health_ok "$PROD_PORT"; then
        log "ROLLBACK: restored $CURRENT_SLOT after failed cutover"
    else
        log "ERROR: previous release also failed health check"
    fi
else
    rm -f "$BASE/current" "$BASE/app.pid"
    log "FAILED: first deployment unhealthy; no previous release available"
fi
exit 1
