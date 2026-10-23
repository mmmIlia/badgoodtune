#!/bin/sh
set -eu

base_url=${VMUI_URL:-http://127.0.0.1:8428/vmui/}
link() {
    title=$1
    query=$2
    encoded=$(jq -rn --arg query "$query" '$query | @uri')
    printf '%s\n%s#/?g0.expr=%s&g0.range_input=15m&g0.relative_time=last_15_minutes&g0.tab=0&refresh=5s\n\n' "$title" "$base_url" "$encoded"
}

for indicator in api latency database catalog; do
    link "SLI and objective thresholds: $indicator (5-minute demonstration window, percent)" \
        "100 * (badgoodtune:sli:ratio{indicator=\"$indicator\",window=\"5m\"} or badgoodtune:slo:target{indicator=\"$indicator\"} or badgoodtune:sla:target{indicator=\"$indicator\"})"
done
link 'Known-minute coverage (percent)' '100 * badgoodtune:sli:coverage{window="5m"}'
link 'Remaining error budget (percent; negative means exhausted)' \
    '100 * (1 - (1 - badgoodtune:sli:ratio{window="5m"}) / on(indicator) group_left() (1 - badgoodtune:slo:target))'
link 'Good, bad and unknown catalog minutes' \
    'badgoodtune:sli_minute:state{indicator="catalog"}'
link 'Raw probe success and database connectivity' 'label_set(probe_success{job="badgoodtune-probe"},"source","catalog") or label_set(pg_up{job="postgres"},"source","database")'
