# gooo-resolution-descent-debugger

`gooo-resolution-descent-debugger` consumes an `UNKNOWN` record containing
`stage`, `step`, `reason`, and `blocked_by`, then narrows the causal frontier
with a lower-resolution typed probe. The executable contract is
[`.gooo/resolution-descent-debugger.gooo`](.gooo/resolution-descent-debugger.gooo):
it declares the resolution lattice, probe capabilities and effects, descent
rules, claim transitions, precedence, origin, and the fixed seven-case
denominator. Go only parses that source, lowers it to semantic IR, generates
typed probe artifacts, executes the declared read-only fixture operation, and
verifies the resulting evidence.

The input repository and fixture are read-only. Every generated
`probe.gooo`, `probe-ir.json`, typed `probe.go`, report, and receipt is written
under the caller-owned output directory. A probe is evidence, not advice: an
original `OPEN` claim is retained and an append-only transition records
`OPEN -> DISCHARGED`, `OPEN -> REFUTED`, or `OPEN -> OPEN`. A smaller frontier
never closes a claim by cardinality alone.

## Fixed conformance corpus

| Case | Expected | Causal assertion |
| --- | --- | --- |
| `direct-missing-closed-by-probe` | `CLOSED` | a direct missing field is matched by a lower-resolution probe |
| `dependency-blocked-frontier-narrowed-closed` | `CLOSED` | dependency IDs are narrowed and all required evidence is discharged |
| `stale-source-refuted` | `REFUTED` | a stale source digest contradicts the original claim |
| `ambiguous-evidence-remains-unknown` | `UNKNOWN` | two matching candidates remain ambiguous |
| `unbounded-descent-unknown` | `UNKNOWN` | no lower lattice node exists within the declared bound |
| `forbidden-observation-effect-refuted` | `REFUTED` | a probe requesting a forbidden write effect is rejected |
| `replay-closed` | `CLOSED` | the same typed probe replays byte-for-byte |

All seven cases report exact before/after unknown-frontier cardinalities and
exact `blocked_by` IDs. The decision order is fixed as
`REFUTED > UNKNOWN > CLOSED`; unknown cases carry all six required fields.
The report contains no aggregate score, weighted average, percentage, or
estimated improvement. Improvement is `UNKNOWN` unless the same
scenario/source/contract/toolchain/runner digest has an exact before/after
pair.

## Run in CI or a caller-owned directory

```text
go run ./cmd/gooo-resolution-descent-debugger manifest \
  --source .gooo/resolution-descent-debugger.gooo \
  --contract contracts/denominator-v1.json

go run ./cmd/gooo-resolution-descent-debugger conformance \
  --root . \
  --source .gooo/resolution-descent-debugger.gooo \
  --contract contracts/denominator-v1.json \
  --out /tmp/gooo-resolution-descent-debugger
```

The product has zero input `repository_writes`, zero local verification
authority, and zero automatic commit/push/merge/release authority. GitHub
Actions is the verification authority. It runs Go 1.27 formatting, build,
test, vet, conformance, and generated-artifact integration checks.

## Release boundary

The release workflow requires an exact merged `main` SHA, refuses tag or
release reuse, confirms immutable releases through the user-facing GitHub API
surface, creates a draft before asset upload, publishes once, and verifies the
public release API reports `immutable=true` and the exact asset digests. It
does not query an admin settings endpoint with `GITHUB_TOKEN`, overwrite a
public tag, or delete a failed release.
