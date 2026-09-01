#!/usr/bin/env bash
set -Eeuo pipefail

if [[ "$#" -ne 3 ]]; then
  echo "usage: conformance.sh REPOSITORY BINARY OUTPUT_ROOT" >&2
  exit 64
fi

repo_root=$1
binary=$2
output_root=$3
mkdir -p "$output_root"
test -z "$(find "$output_root" -mindepth 1 -print -quit)"
mkdir -p "$output_root/cases"

"$binary" conformance \
  --root "$repo_root" \
  --source "$repo_root/.gooo/resolution-descent-debugger.gooo" \
  --contract "$repo_root/contracts/denominator-v1.json" \
  --out "$output_root/cases" >"$output_root/runner.json"

jq -e '
  .schema == "gooo/resolution-descent-debugger/conformance/v1" and
  .fixed_denominator == 7 and .decision == "REFUTED" and
  .repository_writes == 0 and
  .summary == {
    cases_total:7, closed:3, unknown:2, refuted:2,
    unknown_frontier_before:9, unknown_frontier_after:3,
    probe_executions:7, replay_comparisons:1, replay_mismatches:0,
    tests_total:7, tests_selected:7, tests_executed:7, tests_reused:0,
    tests_failed:2, tests_unknown:2
  } and
  (.cases | length) == 7 and
  ([.cases[] | select(.decision == .expected)] | length) == 7 and
  ([.cases[] | select(.before_unknown_frontier_cardinality == (.before_blocked_by|length) and .after_unknown_frontier_cardinality == (.after_blocked_by|length))] | length) == 7 and
  ([.cases[] | select((.claim_transitions|length) == 1 and (.claims|length) == 1 and .claims[0].state == "OPEN" and .claim_transitions[0].append_only == true)] | length) == 7 and
  ([.cases[] | select(.case_id == "direct-missing-closed-by-probe" and .before_blocked_by == ["missing:trace.event"] and .after_blocked_by == [] and .claim_transitions[0].from == "OPEN" and .claim_transitions[0].to == "DISCHARGED")] | length) == 1 and
  ([.cases[] | select(.case_id == "dependency-blocked-frontier-narrowed-closed" and .before_blocked_by == ["dep:contract","dep:input"] and .narrowed_frontier == ["dep:input"] and .after_blocked_by == [] and .claim_transitions[0].to == "DISCHARGED")] | length) == 1 and
  ([.cases[] | select(.case_id == "stale-source-refuted" and .before_blocked_by == ["source:current-digest"] and .after_blocked_by == [] and .claim_transitions[0].to == "REFUTED")] | length) == 1 and
  ([.cases[] | select(.case_id == "ambiguous-evidence-remains-unknown" and .decision == "UNKNOWN" and .before_blocked_by == ["evidence:reason-a","evidence:reason-b"] and .after_blocked_by == .before_blocked_by and .claim_transitions[0].to == "OPEN")] | length) == 1 and
  ([.cases[] | select(.case_id == "unbounded-descent-unknown" and .decision == "UNKNOWN" and .before_blocked_by == ["descent:resolution-bound"] and .after_blocked_by == .before_blocked_by and .claim_transitions[0].to == "OPEN")] | length) == 1 and
  ([.cases[] | select(.case_id == "forbidden-observation-effect-refuted" and .before_blocked_by == ["effect:filesystem-write"] and .after_blocked_by == [] and .claim_transitions[0].to == "REFUTED")] | length) == 1 and
  ([.cases[] | select(.case_id == "replay-closed" and .decision == "CLOSED" and .replay_equal == true and .replay_digest != null and .claim_transitions[0].to == "DISCHARGED")] | length) == 1 and
  (all(.cases[]; .decision != "CLOSED" or (.closure_basis == "ACCEPTED_TYPED_PROBE_EVIDENCE" and .probe_evidence.accepted == true))) and
  (all(.cases[]; .decision != "UNKNOWN" or (.original_claim.stage|type) == "string" and (.original_claim.step|type) == "string" and (.original_claim.reason|type) == "string" and (.original_claim.unknown_class|type) == "string" and (.original_claim.next_operation|type) == "string" and (.after_blocked_by|length) > 0))
' "$output_root/cases/conformance-report.json" >/dev/null

test "$(find "$output_root/cases/generated" -type f -name 'probe.gooo' | wc -l | tr -d ' ')" = 7
test "$(find "$output_root/cases/generated" -type f -name 'probe-ir.json' | wc -l | tr -d ' ')" = 7
test "$(find "$output_root/cases/generated" -type f -name 'probe.go' | wc -l | tr -d ' ')" = 7
