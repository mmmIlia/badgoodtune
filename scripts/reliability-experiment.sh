#!/bin/sh
set -eu

kubectl_command=${KUBECTL:-kubectl}
namespace=${K8S_NAMESPACE:-badgoodtune}
directory=deploy/experiments/reliability
annotation=experiments.badgoodtune.io/statement-timeout
mode=${1:-}

if [ "$("$kubectl_command" config current-context)" != minikube ] || [ "$namespace" != badgoodtune ]; then
    printf '%s\n' 'Experiments are restricted to context minikube and namespace badgoodtune.' >&2
    exit 1
fi

managed() {
    "$kubectl_command" get deployment/badgoodtune-backend -n "$namespace" -o json |
        jq --exit-status --arg annotation "$annotation" '.metadata.annotations[$annotation] == "managed"' >/dev/null
}

stop_fault() {
    "$kubectl_command" delete job/reliability-faults -n "$namespace" --ignore-not-found --wait=true >/dev/null
    "$kubectl_command" exec -n "$namespace" postgres-0 -- sh -ec \
        'psql -X -qAt -v ON_ERROR_STOP=1 -U "$POSTGRES_USER" -d "$POSTGRES_DB" -c "SELECT pg_terminate_backend(pid) FROM pg_stat_activity WHERE application_name = '\''badgoodtune-reliability-experiment'\'' AND pid <> pg_backend_pid()"' >/dev/null
}

case "$mode" in
    prepare)
        if ! managed; then
            "$kubectl_command" get deployment/badgoodtune-backend -n "$namespace" -o json |
                jq --exit-status '[.spec.template.spec.containers[] | select(.name == "backend") | .env[]? | select(.name == "DATABASE_STATEMENT_TIMEOUT")] | length == 0' >/dev/null || {
                    printf '%s\n' 'Refusing to overwrite an existing explicit statement timeout.' >&2; exit 1;
                }
        fi
        "$kubectl_command" create configmap reliability-experiment -n "$namespace" \
            --from-file="$directory/run.sh" --from-file="$directory/lock.sql" --dry-run=client -o yaml |
            "$kubectl_command" apply -f -
        "$kubectl_command" annotate deployment/badgoodtune-backend -n "$namespace" "$annotation=managed" --overwrite
        "$kubectl_command" set env deployment/badgoodtune-backend -n "$namespace" DATABASE_STATEMENT_TIMEOUT=2s
        "$kubectl_command" rollout status deployment/badgoodtune-backend -n "$namespace" --timeout=180s
        ;;
    latency|errors)
        managed || { printf '%s\n' 'Run make reliability-demo-prepare first.' >&2; exit 1; }
        stop_fault
        trap stop_fault EXIT
        trap 'exit 130' INT
        trap 'exit 143' TERM
        trap 'exit 129' HUP
        "$kubectl_command" create --dry-run=client -f "$directory/fault-job.yaml" -o json |
            jq --arg mode "$mode" '(.spec.template.spec.containers[0].env[] | select(.name == "MODE") | .value) = $mode' |
            "$kubectl_command" apply -f -
        printf 'Running bounded %s experiment; table contents will not change.\n' "$mode"
        if ! "$kubectl_command" wait --for=condition=complete job/reliability-faults -n "$namespace" --timeout=100s; then
            "$kubectl_command" logs job/reliability-faults -n "$namespace"
            exit 1
        fi
        "$kubectl_command" logs job/reliability-faults -n "$namespace"
        ;;
    recover)
        stop_fault
        printf '%s\n' 'Fault job stopped and its database sessions terminated; no backend restart required.'
        ;;
    traffic)
        "$kubectl_command" delete job/reliability-traffic -n "$namespace" --ignore-not-found
        "$kubectl_command" apply -f "$directory/traffic-job.yaml"
        "$kubectl_command" wait --for=jsonpath='{.status.active}'=1 job/reliability-traffic -n "$namespace" --timeout=120s
        "$kubectl_command" wait --for=condition=Ready pod -l job-name=reliability-traffic -n "$namespace" --timeout=120s
        ;;
    stop-traffic)
        "$kubectl_command" delete job/reliability-traffic -n "$namespace" --ignore-not-found
        ;;
    finish)
        stop_fault
        "$kubectl_command" delete job/reliability-traffic configmap/reliability-experiment -n "$namespace" --ignore-not-found
        if managed; then
            "$kubectl_command" set env deployment/badgoodtune-backend -n "$namespace" DATABASE_STATEMENT_TIMEOUT-
            "$kubectl_command" annotate deployment/badgoodtune-backend -n "$namespace" "$annotation-"
            "$kubectl_command" rollout status deployment/badgoodtune-backend -n "$namespace" --timeout=180s
        fi
        ;;
    *) printf '%s\n' 'Usage: reliability-experiment.sh prepare|latency|errors|recover|traffic|stop-traffic|finish' >&2; exit 2 ;;
esac
