#!/bin/sh
set -eu

metrics_url=${METRICS_URL:-http://127.0.0.1:18428}
namespace=${K8S_NAMESPACE:-badgoodtune}
kubectl_command=${KUBECTL:-kubectl}
wait_timeout=${METRICS_WAIT_TIMEOUT:-90}
replicas=$("$kubectl_command" get deployment/badgoodtune-backend -n "$namespace" -o jsonpath='{.spec.replicas}')
response=$(mktemp)
trap 'rm -f "$response"' EXIT HUP INT TERM

wait_for() {
    description=$1
    shift
    deadline=$(( $(date +%s) + wait_timeout ))
    until "$@"; do
        if [ "$(date +%s)" -ge "$deadline" ]; then
            printf 'Timed out: %s\n' "$description" >&2
            cat "$response" >&2
            exit 1
        fi
        sleep 2
    done
    printf 'OK: %s\n' "$description"
}

targets_ready() {
    curl --fail --silent --show-error --max-time 10 "$metrics_url/api/v1/targets" >"$response" &&
    jq --exit-status --argjson replicas "$replicas" '
      .status == "success" and
      ([.data.activeTargets[] | select(.labels.job == "badgoodtune-backend")] | length == $replicas and all(.health == "up")) and
      ([.data.activeTargets[] | select(.labels.job == "postgres")] | length == 1 and all(.health == "up"))
    ' "$response" >/dev/null || return 1
    backend_pods=$(jq --compact-output '[.data.activeTargets[] | select(.labels.job == "badgoodtune-backend") | .labels.pod] | sort' "$response")
}

query() {
    curl --fail --silent --show-error --max-time 10 --get --data-urlencode "query=$1" --data-urlencode nocache=1 "$metrics_url/api/v1/query" >"$response"
}

backend_metrics_ready() {
    query 'count by (__name__, pod) ({job="badgoodtune-backend",__name__=~"go_goroutines|process_cpu_seconds_total|badgoodtune_users_registered_total|badgoodtune_logins_total|badgoodtune_albums_created_total|badgoodtune_reviews_created_total|badgoodtune_review_score_count|badgoodtune_review_score_sum"})' &&
    jq --exit-status --argjson replicas "$replicas" --argjson pods "$backend_pods" '
      .status == "success" and (.data.result | length == (8 * $replicas) and all(.value[1] == "1")) and
      ([.data.result[].metric.pod] | unique) == $pods
    ' "$response" >/dev/null
}

postgres_metrics_ready() {
    query 'pg_up{job="postgres"}' &&
    jq --exit-status '.status == "success" and (.data.result | length == 1 and all(.value[1] == "1"))' "$response" >/dev/null &&
    query 'pg_stat_database_numbackends{job="postgres",datname="badgoodtune"}' &&
    jq --exit-status '.status == "success" and (.data.result | length == 1)' "$response" >/dev/null
}

wait_for "one healthy target per backend replica ($replicas) and one PostgreSQL exporter" targets_ready
wait_for 'runtime and business metrics stored for every backend replica' backend_metrics_ready
wait_for 'PostgreSQL is reachable and its database statistics are stored' postgres_metrics_ready
