#!/bin/sh
set -eu

metrics_url=${METRICS_URL:-http://127.0.0.1:${FORWARDED_PORT:-18428}}
now=${RELIABILITY_NOW:-$(date +%s)}
case "$now" in ''|*[!0-9]*) printf '%s\n' 'RELIABILITY_NOW must be Unix seconds.' >&2; exit 2 ;; esac

if month=$(date -u -d "@$now" +%Y-%m-01T00:00:00Z 2>/dev/null); then
    start=$(date -u -d "$month" +%s)
else
    month=$(date -u -r "$now" +%Y-%m-01T00:00:00Z)
    start=$(date -j -u -f '%Y-%m-%dT%H:%M:%SZ' "$month" +%s)
fi
end=$(( (now - 15) / 60 * 60 ))
if [ "$end" -le "$start" ]; then
    printf '%s\n' 'No complete evaluated minute in this UTC calendar month yet.' >&2
    exit 1
fi

curl --fail --silent --show-error --max-time 30 --get \
    --data-urlencode 'match[]=badgoodtune:sli_minute:state{indicator="catalog"}' \
    --data-urlencode "start=$start" --data-urlencode "end=$((end + 15))" \
    "$metrics_url/api/v1/export" |
    jq --slurp --arg month "$month" --argjson start "$start" --argjson end "$end" \
        -f scripts/reliability-monthly.jq
