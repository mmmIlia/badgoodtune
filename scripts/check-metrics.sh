#!/bin/sh
set -eu

metrics_url=${METRICS_URL:-http://127.0.0.1:18428}
namespace=${K8S_NAMESPACE:-badgoodtune}
kubectl_command=${KUBECTL:-kubectl}
wait_timeout=${METRICS_WAIT_TIMEOUT:-90}
replicas=$("$kubectl_command" get deployment/badgoodtune-backend -n "$namespace" -o jsonpath='{.spec.replicas}')
database_name=$("$kubectl_command" get configmap/badgoodtune-config -n "$namespace" -o jsonpath='{.data.POSTGRES_DB}')
test -n "$database_name"
database_label=$(printf '%s' "$database_name" | jq --raw-input --slurp .)
required_metrics='[
  "go_goroutines",
  "process_cpu_seconds_total",
  "badgoodtune_users_registered_total",
  "badgoodtune_logins_total",
  "badgoodtune_albums_created_total",
  "badgoodtune_reviews_created_total",
  "badgoodtune_review_score_count",
  "badgoodtune_review_score_sum"
]'
metric_pattern=$(printf '%s' "$required_metrics" | jq --raw-output 'join("|")')
response=$(mktemp)
trap 'rm -f "$response"' EXIT
trap 'exit 130' INT
trap 'exit 143' TERM
trap 'exit 129' HUP

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
    current_pods=$("$kubectl_command" get pods -n "$namespace" -l app.kubernetes.io/name=badgoodtune,app.kubernetes.io/component=backend -o json |
        jq --compact-output '[.items[] | select(.metadata.deletionTimestamp == null and .status.phase == "Running") | .metadata.name] | sort') || return 1
    curl --fail --silent --show-error --max-time 10 "$metrics_url/api/v1/targets" >"$response" &&
    jq --exit-status --argjson replicas "$replicas" --argjson pods "$current_pods" '
      .status == "success" and $replicas > 0 and
      ([.data.activeTargets[] | select(.labels.job == "badgoodtune-backend")] | length == $replicas and all(.health == "up")) and
      ([.data.activeTargets[] | select(.labels.job == "badgoodtune-backend") | .labels.pod] | sort) == $pods and
      ([.data.activeTargets[] | select(.labels.job == "postgres")] | length == 1 and all(.health == "up"))
    ' "$response" >/dev/null || return 1
    backend_pods=$(jq --compact-output '[.data.activeTargets[] | select(.labels.job == "badgoodtune-backend") | .labels.pod] | sort' "$response")
    postgres_pod=$(jq --raw-output '.data.activeTargets[] | select(.labels.job == "postgres") | .labels.pod' "$response")
}

query() {
    curl --fail --silent --show-error --max-time 10 --get --data-urlencode "query=$1" --data-urlencode nocache=1 "$metrics_url/api/v1/query" >"$response"
}

backend_metrics_ready() {
    targets_ready &&
    query "count by (__name__, pod) ({job=\"badgoodtune-backend\",__name__=~\"$metric_pattern\"})" &&
    jq --exit-status --argjson names "$required_metrics" --argjson pods "$backend_pods" '
      .status == "success" and
      ([.data.result[] | [.metric.pod, .metric.__name__, .value[1]]] | sort) ==
      ([$pods[] as $pod | $names[] | [$pod, ., "1"]] | sort)
    ' "$response" >/dev/null
}

postgres_metrics_ready() {
    targets_ready &&
    query "pg_up{job=\"postgres\",pod=\"$postgres_pod\"}" &&
    jq --exit-status '.status == "success" and (.data.result | length == 1 and all(.value[1] == "1"))' "$response" >/dev/null &&
    query "pg_stat_database_numbackends{job=\"postgres\",pod=\"$postgres_pod\",datname=$database_label}" &&
    jq --exit-status '.status == "success" and (.data.result | length == 1)' "$response" >/dev/null
}

wait_for "one healthy target per backend replica ($replicas) and one PostgreSQL exporter" targets_ready
wait_for 'runtime and business metrics stored for every backend replica' backend_metrics_ready
wait_for 'PostgreSQL is reachable and its database statistics are stored' postgres_metrics_ready
