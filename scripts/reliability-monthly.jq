[
  .[] |
  range(0; .timestamps | length) as $index |
  {minute: ((.timestamps[$index] / 60000 | floor) * 60), value: .values[$index]} |
  select(.minute > $start and .minute <= $end)
] |
group_by(.minute) | map(min_by(.value)) as $samples |
($samples | map(select(.value == 1)) | length) as $good |
($samples | map(select(.value >= 0)) | length) as $known |
(($end - $start) / 60) as $expected |
{
  period_start_utc: $month,
  evaluated_until_utc: ($end | todateiso8601),
  expected_minutes: $expected,
  good_minutes: $good,
  known_minutes: $known,
  unknown_minutes: ($expected - $known),
  coverage: ($known / $expected),
  observed_availability: (if $known > 0 then $good / $known else null end),
  availability_lower_bound: ($good / $expected),
  sla_target: 0.99,
  status: (if $known / $expected < 0.999 then "insufficient_data"
           elif $good / $expected >= 0.99 then "provisionally_met"
           else "provisionally_breached" end),
  final: false
}
