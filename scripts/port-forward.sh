#!/bin/sh
set -eu

if [ "$#" -lt 4 ]; then
    printf '%s\n' 'Usage: port-forward.sh <namespace> <resource> <local:remote> <command> [args...]' >&2
    exit 2
fi

namespace=$1
resource=$2
mapping=$3
shift 3

log=$(mktemp)
"${KUBECTL:-kubectl}" port-forward --address=127.0.0.1 --namespace "$namespace" "$resource" "$mapping" >"$log" 2>&1 &
forward_pid=$!
trap 'kill "$forward_pid" 2>/dev/null || true; wait "$forward_pid" 2>/dev/null || true; rm -f "$log"' EXIT
trap 'exit 130' INT
trap 'exit 143' TERM
trap 'exit 129' HUP

attempt=0
until grep -q '^Forwarding from ' "$log"; do
    attempt=$((attempt + 1))
    if ! kill -0 "$forward_pid" 2>/dev/null || [ "$attempt" -ge 30 ]; then
        cat "$log" >&2
        exit 1
    fi
    sleep 1
done

FORWARDED_PORT=$(sed -n 's/^Forwarding from 127\.0\.0\.1:\([0-9]*\) -> .*/\1/p' "$log" | head -n 1)
export FORWARDED_PORT

"$@"
