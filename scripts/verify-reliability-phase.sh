#!/bin/sh
set -eu

mode=${1:-}
case "$mode" in latency|errors|recovery) ;; *) printf '%s\n' 'Expected latency, errors or recovery.' >&2; exit 2 ;; esac
metrics_url=${METRICS_URL:-http://127.0.0.1:${FORWARDED_PORT:-18428}}
response=$(mktemp)
targets=$(mktemp)
trap 'rm -f "$response" "$targets"' EXIT
trap 'exit 130' INT
trap 'exit 143' TERM
trap 'exit 129' HUP
deadline=$(( $(date +%s) + ${RELIABILITY_PHASE_TIMEOUT:-120} ))

observed() {
    if [ "$mode" = recovery ]; then
        curl --fail --silent --show-error --max-time 10 --get --data-urlencode nocache=1 \
            --data-urlencode 'query=probe_success{job="badgoodtune-probe"} and (time() - timestamp(probe_success{job="badgoodtune-probe"}) < 15)' \
            "$metrics_url/api/v1/query" >"$response" &&
            jq -e '.status == "success" and (.data.result | length == 1 and .[0].value[1] == "1")' "$response" >/dev/null
        return
    fi
    curl --fail --silent --show-error --max-time 10 --get --data-urlencode nocache=1 \
        --data-urlencode 'query=badgoodtune:slo:target and (time() - timestamp(badgoodtune:slo:target) < 45)' \
        "$metrics_url/api/v1/query" >"$targets" || return 1
    curl --fail --silent --show-error --max-time 10 --get --data-urlencode nocache=1 \
        --data-urlencode 'query=badgoodtune:sli:ratio{window="5m"} and (time() - timestamp(badgoodtune:sli:ratio{window="5m"}) < 45)' \
        "$metrics_url/api/v1/query" >"$response" &&
        jq -e --arg mode "$mode" --slurpfile objectives "$targets" '
          ($objectives[0].data.result | map({key: .metric.indicator, value: (.value[1] | tonumber)}) | from_entries) as $target |
          $objectives[0].status == "success" and ($target | length == 4) and
          .status == "success" and (.data.result | length == 4) and
          (.data.result | map({key: .metric.indicator, value: (.value[1] | tonumber)}) | from_entries |
            .database >= $target.database and
            (if $mode == "latency" then .latency < $target.latency and .api >= $target.api and .catalog >= $target.catalog
             else .api < $target.api and .catalog < $target.catalog end))
        ' "$response" >/dev/null
}

printf 'Waiting for observed %s results after scraping and rule evaluation...\n' "$mode"
until observed; do
    if [ "$(date +%s)" -ge "$deadline" ]; then
        printf 'Timed out verifying %s. Inspect raw metrics and ensure there is no concurrent experiment.\n' "$mode" >&2
        cat "$response" >&2
        exit 1
    fi
    sleep 5
done
jq '.data.result | map({metric: .metric, value: (.value[1] | tonumber)})' "$response"
printf 'OK: %s is visible in collected metrics.\n' "$mode"
