#!/bin/sh
set -eu

response=$(mktemp)
trap 'rm -f "$response"' EXIT
trap 'exit 130' INT
trap 'exit 143' TERM
trap 'exit 129' HUP

deadline=$(( $(date +%s) + ${ALERTS_WAIT_TIMEOUT:-90} ))
until curl --fail --silent --show-error --max-time 10 http://127.0.0.1:18880/api/v1/alerts >"$response" &&
    jq --exit-status 'any(.data.alerts[]?; .name == "BadGoodTuneWarningLogs" and .state == "firing")' "$response" >/dev/null; do
    if [ "$(date +%s)" -ge "$deadline" ]; then
        printf '%s\n' 'Timed out waiting for BadGoodTuneWarningLogs to fire.' >&2
        cat "$response" >&2
        exit 1
    fi
    sleep 2
done
cat "$response"
