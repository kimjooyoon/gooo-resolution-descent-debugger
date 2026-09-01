#!/usr/bin/env bash
set -Eeuo pipefail

if [[ "$#" -ne 2 ]]; then
  echo "usage: integration.sh GENERATED_ROOT OUTPUT_DIR" >&2
  exit 64
fi

generated_root=$1
output_dir=$2
test -d "$generated_root"
mkdir -p "$output_dir"
test -z "$(find "$output_dir" -mindepth 1 -print -quit)"

cp -R "$generated_root" "$output_dir/generated"
printf 'module github.com/kimjooyoon/gooo-resolution-descent-debugger-generated-probes\n\ngo 1.27.0\n' > "$output_dir/go.mod"
(cd "$output_dir" && go test ./...)

count=$(find "$output_dir/generated" -type f -name 'probe.go' | wc -l | tr -d ' ')
test "$count" = 7
nonstrings=$(find "$output_dir/generated" -type f -name 'probe.go' -exec awk '/^package generatedprobe$/{found=1} END{if(!found) exit 1}' {} \; -print | wc -l | tr -d ' ')
test "$nonstrings" = 7
