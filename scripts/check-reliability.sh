#!/bin/sh
set -eu

metrics_url=${METRICS_URL:-http://127.0.0.1:${FORWARDED_PORT:-18428}}
timeout=${RELIABILITY_WAIT_TIMEOUT:-420}
response=$(mktemp)
trap 'rm -f "$response"' EXIT
trap 'exit 130' INT
trap 'exit 143' TERM
trap 'exit 129' HUP
deadline=$(( $(date +%s) + timeout ))
indicator_query='badgoodtune:sli:ratio{window="5m"} and (time() - timestamp(badgoodtune:sli:ratio{window="5m"}) < 45)'
case "${RELIABILITY_EXPECT:-available}" in
    available) ;;
    healthy)
        indicator_query="WITH (current = ($indicator_query), target = (badgoodtune:slo:target and (time() - timestamp(badgoodtune:slo:target) < 45))) current >= on(indicator) group_left target"
        ;;
    *) printf '%s\n' 'RELIABILITY_EXPECT must be available or healthy.' >&2; exit 2 ;;
esac

query() {
    curl --fail --silent --show-error --max-time 10 --get \
        --data-urlencode "query=$1" --data-urlencode nocache=1 "$metrics_url/api/v1/query" >"$response"
}

ready() {
    query 'min(probe_success{job="badgoodtune-probe"}) * min(up{job="badgoodtune-probe"})' &&
        jq -e '.status == "success" and (.data.result | length == 1 and .[0].value[1] == "1")' "$response" >/dev/null &&
    query '(time() - timestamp(badgoodtune:slo:target)) < 45' &&
        jq -e '.status == "success" and (.data.result | length == 4 and all((.value[1] | tonumber) < 45))' "$response" >/dev/null &&
    query "$indicator_query" &&
        jq -e '.status == "success" and ([.data.result[].metric.indicator] | sort) == ["api","catalog","database","latency"] and all(.data.result[]; (.value[1] | tonumber) >= 0 and (.value[1] | tonumber) <= 1)' "$response" >/dev/null
}

printf '%s\n' 'Waiting for a healthy probe, fresh evaluator data and a complete five-minute observation window...'
until ready; do
    if [ "$(date +%s)" -ge "$deadline" ]; then
        printf '%s\n' 'Timed out waiting for reliability data. Run make reliability-status and inspect evaluator logs.' >&2
        cat "$response" >&2
        exit 1
    fi
    sleep 5
done
jq '.data.result | map({indicator: .metric.indicator, window: .metric.window, ratio: (.value[1] | tonumber)})' "$response"
printf '%s\n' 'OK: four SLI series are available. Availability of data does not itself imply that every SLO is met.'
