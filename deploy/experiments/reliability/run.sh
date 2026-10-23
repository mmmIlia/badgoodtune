#!/bin/sh
set -eu

case "$MODE" in
    latency) hold_seconds=0.8; iterations=60; pause_seconds=0.2 ;;
    errors) hold_seconds=45; iterations=1; pause_seconds=0 ;;
    *) printf '%s\n' 'Unknown experiment mode.' >&2; exit 2 ;;
esac

export PGAPPNAME=badgoodtune-reliability-experiment
export PGOPTIONS='-c lock_timeout=3s -c statement_timeout=55s -c idle_in_transaction_session_timeout=5s'
printf '{"event":"experiment_started","mode":"%s"}\n' "$MODE"
iteration=0
while [ "$iteration" -lt "$iterations" ]; do
    psql -X --no-password --set=ON_ERROR_STOP=1 --set="hold_seconds=$hold_seconds" --file=/experiment/lock.sql >/dev/null
    sleep "$pause_seconds"
    iteration=$((iteration + 1))
done
printf '{"event":"experiment_completed","mode":"%s"}\n' "$MODE"
