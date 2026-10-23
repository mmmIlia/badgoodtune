#!/bin/sh
set -eu

base_url="${BASE_URL:-http://127.0.0.1:8080}"
review_score=${SMOKE_REVIEW_SCORE:-9}
case "$review_score" in
  1|2|3|4|5|6|7|8|9|10) ;;
  *) printf '%s\n' 'SMOKE_REVIEW_SCORE must be an integer between 1 and 10.' >&2; exit 2 ;;
esac
cookie_jar="$(mktemp)"
trap 'rm -f "$cookie_jar"' EXIT
trap 'exit 130' INT
trap 'exit 143' TERM
trap 'exit 129' HUP

username="smoke_$(date +%s)_$$"
password="smoke-test-password"

request() {
  command curl --connect-timeout 5 --max-time 15 "$@"
}

request --fail --silent --show-error "$base_url/health/live" >/dev/null
request --fail --silent --show-error "$base_url/health/ready" >/dev/null
request --fail --silent --show-error "$base_url/healthz" | grep -q '^ok$'

request --fail --silent --show-error \
  --cookie-jar "$cookie_jar" \
  --header 'Content-Type: application/json' \
  --data "{\"username\":\"$username\",\"password\":\"$password\"}" \
  "$base_url/api/v1/users" >/dev/null

request --fail --silent --show-error --cookie "$cookie_jar" "$base_url/api/v1/users/me" | grep -q "\"username\":\"$username\""

album_json="$(request --fail --silent --show-error \
  --cookie "$cookie_jar" \
  --header 'Content-Type: application/json' \
  --data "{\"artist\":\"Portishead\",\"title\":\"Smoke $username\",\"releaseYear\":1994,\"genre\":\"Trip-hop\"}" \
  "$base_url/api/v1/albums")"
album_id="$(printf '%s' "$album_json" | jq --exit-status --raw-output '.id')"
test -n "$album_id"

album_json="$(request --fail --silent --show-error \
  --request PATCH \
  --cookie "$cookie_jar" \
  --header 'Content-Type: application/json' \
  --data '{"genre":"Bristol trip-hop","version":1}' \
  "$base_url/api/v1/albums/$album_id")"
printf '%s' "$album_json" | grep -q '"genre":"Bristol trip-hop"'
printf '%s' "$album_json" | grep -q '"version":2'

review_json="$(request --fail --silent --show-error \
  --cookie "$cookie_jar" \
  --header 'Content-Type: application/json' \
  --data "{\"headline\":\"Still sounds like the future\",\"body\":\"Sparse, tense, and completely assured.\",\"score\":$review_score}" \
  "$base_url/api/v1/albums/$album_id/reviews")"
review_id="$(printf '%s' "$review_json" | jq --exit-status --raw-output '.id')"
test -n "$review_id"

request --fail --silent --show-error "$base_url/api/v1/albums/$album_id" | grep -q '"reviewCount":1'

review_json="$(request --fail --silent --show-error \
  --request PATCH \
  --cookie "$cookie_jar" \
  --header 'Content-Type: application/json' \
  --data '{"score":8,"version":1}' \
  "$base_url/api/v1/albums/$album_id/reviews/$review_id")"
printf '%s' "$review_json" | grep -q '"score":8'
printf '%s' "$review_json" | grep -q '"version":2'
request --fail --silent --show-error "$base_url/api/v1/albums/$album_id" | grep -q '"averageScore":8'

request --fail --silent --show-error \
  --request DELETE \
  --cookie "$cookie_jar" \
  "$base_url/api/v1/albums/$album_id/reviews/$review_id?version=2" >/dev/null
request --fail --silent --show-error "$base_url/api/v1/albums/$album_id" | grep -q '"reviewCount":0'

request --fail --silent --show-error \
  --request DELETE \
  --cookie "$cookie_jar" \
  "$base_url/api/v1/albums/$album_id?version=2" >/dev/null
test "$(request --silent --show-error --output /dev/null --write-out '%{http_code}' "$base_url/api/v1/albums/$album_id")" = "404"

request --fail --silent --show-error \
  --request DELETE \
  --cookie "$cookie_jar" \
  --cookie-jar "$cookie_jar" \
  "$base_url/api/v1/sessions/current" >/dev/null
test "$(request --silent --show-error --output /dev/null --write-out '%{http_code}' --cookie "$cookie_jar" "$base_url/api/v1/users/me")" = "401"

request --fail --silent --show-error \
  --cookie-jar "$cookie_jar" \
  --header 'Content-Type: application/json' \
  --data "{\"username\":\"$username\",\"password\":\"$password\"}" \
  "$base_url/api/v1/sessions" >/dev/null
request --fail --silent --show-error --cookie "$cookie_jar" "$base_url/api/v1/users/me" | grep -q "\"username\":\"$username\""

request --fail --silent --show-error "$base_url/albums/$album_id" | grep -q '<div id="app"></div>'

printf '%s\n' 'Smoke test passed.'
