#!/usr/bin/env bash
set -Eeuo pipefail

if [[ "$#" -ne 1 ]]; then
  echo "usage: semantic-audit.sh REPOSITORY_ROOT" >&2
  exit 64
fi

root=$1
source="$root/.gooo/resolution-descent-debugger.gooo"
contract="$root/contracts/denominator-v1.json"
test -f "$source"
test -f "$contract"
grep -Eq '^resolution_lattice levels=STAGE,STEP,REASON,BLOCKED_BY$' "$source"
test "$(grep -c '^probe_capability ' "$source")" = 2
test "$(grep -c '^probe_effect ' "$source")" = 2
test "$(grep -c '^descent_rule ' "$source")" = 4
test "$(grep -c '^claim_transition ' "$source")" = 3
test "$(grep -c '^scenario ' "$source")" = 7
jq -e '.scenario_count == 7 and .root_readme_inventory_excluded == true and .precedence == ["REFUTED","UNKNOWN","CLOSED"] and (.unknown_fields|length) == 6' "$contract" >/dev/null
