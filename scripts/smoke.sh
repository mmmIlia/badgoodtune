#!/bin/sh
set -eu

base_url="${BASE_URL:-http://127.0.0.1:8080}"
cookie_jar="$(mktemp)"
trap 'rm -f "$cookie_jar"' EXIT HUP INT TERM

username="smoke_$(date +%s)"
password="smoke-test-password"

curl --fail --silent --show-error "$base_url/health/live" >/dev/null
curl --fail --silent --show-error "$base_url/health/ready" >/dev/null
curl --fail --silent --show-error "$base_url/healthz" | grep -q '^ok$'

curl --fail --silent --show-error \
  --cookie-jar "$cookie_jar" \
  --header 'Content-Type: application/json' \
  --data "{\"username\":\"$username\",\"password\":\"$password\"}" \
  "$base_url/api/v1/users" >/dev/null

curl --fail --silent --show-error --cookie "$cookie_jar" "$base_url/api/v1/users/me" | grep -q "\"username\":\"$username\""

album_json="$(curl --fail --silent --show-error \
  --cookie "$cookie_jar" \
  --header 'Content-Type: application/json' \
  --data "{\"artist\":\"Portishead\",\"title\":\"Smoke $username\",\"releaseYear\":1994,\"genre\":\"Trip-hop\"}" \
  "$base_url/api/v1/albums")"
album_id="$(printf '%s' "$album_json" | awk -F'"id":"' '{print $2}' | cut -d'"' -f1)"
test -n "$album_id"

album_json="$(curl --fail --silent --show-error \
  --request PATCH \
  --cookie "$cookie_jar" \
  --header 'Content-Type: application/json' \
  --data '{"genre":"Bristol trip-hop","version":1}' \
  "$base_url/api/v1/albums/$album_id")"
printf '%s' "$album_json" | grep -q '"genre":"Bristol trip-hop"'
printf '%s' "$album_json" | grep -q '"version":2'

review_json="$(curl --fail --silent --show-error \
  --cookie "$cookie_jar" \
  --header 'Content-Type: application/json' \
  --data '{"headline":"Still sounds like the future","body":"Sparse, tense, and completely assured.","score":9}' \
  "$base_url/api/v1/albums/$album_id/reviews")"
review_id="$(printf '%s' "$review_json" | awk -F'"id":"' '{print $NF}' | cut -d'"' -f1)"
test -n "$review_id"

curl --fail --silent --show-error "$base_url/api/v1/albums/$album_id" | grep -q '"reviewCount":1'

review_json="$(curl --fail --silent --show-error \
  --request PATCH \
  --cookie "$cookie_jar" \
  --header 'Content-Type: application/json' \
  --data '{"score":8,"version":1}' \
  "$base_url/api/v1/albums/$album_id/reviews/$review_id")"
printf '%s' "$review_json" | grep -q '"score":8'
printf '%s' "$review_json" | grep -q '"version":2'
curl --fail --silent --show-error "$base_url/api/v1/albums/$album_id" | grep -q '"averageScore":8'

curl --fail --silent --show-error \
  --request DELETE \
  --cookie "$cookie_jar" \
  "$base_url/api/v1/albums/$album_id/reviews/$review_id?version=2" >/dev/null
curl --fail --silent --show-error "$base_url/api/v1/albums/$album_id" | grep -q '"reviewCount":0'

curl --fail --silent --show-error \
  --request DELETE \
  --cookie "$cookie_jar" \
  "$base_url/api/v1/albums/$album_id?version=2" >/dev/null
test "$(curl --silent --show-error --output /dev/null --write-out '%{http_code}' "$base_url/api/v1/albums/$album_id")" = "404"

curl --fail --silent --show-error \
  --request DELETE \
  --cookie "$cookie_jar" \
  --cookie-jar "$cookie_jar" \
  "$base_url/api/v1/sessions/current" >/dev/null
test "$(curl --silent --show-error --output /dev/null --write-out '%{http_code}' --cookie "$cookie_jar" "$base_url/api/v1/users/me")" = "401"

curl --fail --silent --show-error \
  --cookie-jar "$cookie_jar" \
  --header 'Content-Type: application/json' \
  --data "{\"username\":\"$username\",\"password\":\"$password\"}" \
  "$base_url/api/v1/sessions" >/dev/null
curl --fail --silent --show-error --cookie "$cookie_jar" "$base_url/api/v1/users/me" | grep -q "\"username\":\"$username\""

curl --fail --silent --show-error "$base_url/albums/$album_id" | grep -q '<div id="app"></div>'

printf '%s\n' 'Smoke test passed.'
