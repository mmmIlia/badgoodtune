#!/bin/sh
set -eu

metrics_url=${METRICS_URL:-http://127.0.0.1:18428}
namespace=${K8S_NAMESPACE:-badgoodtune}
kubectl_command=${KUBECTL:-kubectl}
wait_timeout=${METRICS_WAIT_TIMEOUT:-90}
baseline=$(mktemp)
current=$(mktemp)
diagnostic=$(mktemp)
trap 'rm -f "$baseline" "$current" "$diagnostic"' EXIT HUP INT TERM

snapshot() {
    curl --fail --silent --show-error --max-time 10 --get \
        --data-urlencode "query=sum by (__name__) ({job=\"badgoodtune-backend\",pod=~\"$backend_pods\",__name__=~\"badgoodtune_users_registered_total|badgoodtune_logins_total|badgoodtune_albums_created_total|badgoodtune_reviews_created_total|badgoodtune_review_score_count|badgoodtune_review_score_sum\"})" \
        --data-urlencode nocache=1 \
        "$metrics_url/api/v1/query" |
        jq --exit-status 'if .status == "success" and (.data.result | length == 6) then .data.result | map({key: .metric.__name__, value: (.value[1] | tonumber)}) | from_entries else error("expected 6 business metric series, got \(.data.result | length); status=\(.status)") end'
}

./scripts/check-metrics.sh
backend_pods=$("$kubectl_command" get pods -n "$namespace" -l app.kubernetes.io/name=badgoodtune,app.kubernetes.io/component=backend -o json |
    jq --raw-output '[.items[] | select(.metadata.deletionTimestamp == null and .status.phase == "Running") | .metadata.name] | join("|")')
test -n "$backend_pods"
printf '%s\n' 'Waiting for the initial business metric snapshot...'
deadline=$(( $(date +%s) + wait_timeout ))
until snapshot >"$baseline" 2>"$diagnostic"; do
    if [ "$(date +%s)" -ge "$deadline" ]; then
        cat "$diagnostic" >&2
        exit 1
    fi
    sleep 2
done
./scripts/port-forward.sh "$namespace" service/badgoodtune-frontend 18080:80 env BASE_URL=http://127.0.0.1:18080 ./scripts/smoke.sh

printf '%s\n' 'Waiting for the next scrape to verify business metric changes...'
deadline=$(( $(date +%s) + wait_timeout ))
until snapshot >"$current" 2>"$diagnostic" && jq --exit-status --slurpfile baseline "$baseline" '
    to_entries | all(.value == ($baseline[0][.key] + (if .key == "badgoodtune_review_score_sum" then 9 else 1 end)))
  ' "$current" >/dev/null; do
    if [ "$(date +%s)" -ge "$deadline" ]; then
        printf '%s\n' 'Business counters did not change as expected. Check for concurrent traffic or backend restarts.' >&2
        cat "$diagnostic" "$baseline" "$current" >&2
        exit 1
    fi
    sleep 2
done

jq --slurpfile baseline "$baseline" 'to_entries | map({metric: .key, before: $baseline[0][.key], after: .value, delta: (.value - $baseline[0][.key])})' "$current"
