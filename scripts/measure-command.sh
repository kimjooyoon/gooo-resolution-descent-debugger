#!/usr/bin/env bash
set -Eeuo pipefail

if [[ "$#" -lt 4 ]]; then
  echo "usage: measure-command.sh OUTPUT_JSON STAGE LOG COMMAND [ARGS...]" >&2
  exit 64
fi

output=$1
stage=$2
log=$3
shift 3
started=$(date +%s%3N)
set +e
/usr/bin/time -v "$@" >"$log" 2>"$output.time"
status=$?
set -e
finished=$(date +%s%3N)
rss=$(awk -F: '/Maximum resident set size/{gsub(/ /,"",$2); print $2+0}' "$output.time")
wall=$((finished-started))
jq -n --arg stage "$stage" --argjson status "$status" --argjson wall_ms "$wall" --argjson peak_rss_kib "${rss:-0}" \
  '{stage:$stage,status:$status,wall_ms:$wall_ms,peak_rss_kib:$peak_rss_kib}' >"$output"
sed -n '1,260p' "$log"
test "$status" -eq 0
