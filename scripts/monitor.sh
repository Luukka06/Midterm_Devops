#!/usr/bin/env bash
set -euo pipefail

URL="${1:-http://127.0.0.1:8080}"
LOG_FILE="${2:-./health.log}"
URL="${URL%/}"
mkdir -p "$(dirname "$LOG_FILE")"
TIME="$(date -u '+%Y-%m-%dT%H:%M:%SZ')"

# The health endpoint returns a small JSON object with status="ok".
if RESPONSE="$(curl --silent --show-error --fail --max-time 5 "${URL}/healthz" 2>&1)" \
    && [[ "$RESPONSE" == *'"status":"ok"'* ]]; then
    echo "${TIME} status=UP url=${URL}/healthz" | tee -a "$LOG_FILE"
else
    echo "${TIME} status=DOWN url=${URL}/healthz response=${RESPONSE:-none}" | tee -a "$LOG_FILE" >&2
    exit 1
fi
