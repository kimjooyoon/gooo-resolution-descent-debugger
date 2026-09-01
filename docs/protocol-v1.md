# Resolution descent debugger protocol v1

## Authority chain

```text
.gooo source -> semantic IR -> typed probe .gooo/IR/Go -> fixture observation
             -> append-only claim transition -> precedence decision
```

The `.gooo` source is authoritative for the four-level lattice
`STAGE -> STEP -> REASON -> BLOCKED_BY`, the available probe capabilities,
their effects, the legal descent edges, the three claim transitions, the
unknown tuple, and the fixed denominator. The JSON denominator is an
independent fixed-vector check; it cannot add scenarios at runtime.

## Claim semantics

Each input begins with a durable `OPEN` claim. Its original record remains in
the output `claims` array. Evidence appends a transition record with the
previous and next lifecycle values. `OPEN -> DISCHARGED` requires a matching
probe and a read-only declared effect. `OPEN -> REFUTED` requires a digest
contradiction or forbidden effect. Unknown or ambiguous evidence appends
`OPEN -> OPEN`; it never becomes closure merely because the number of blocked
IDs decreased.

The top-level decision is computed with exact precedence
`REFUTED > UNKNOWN > CLOSED`. A `UNKNOWN` record always contains non-empty
`stage`, `step`, `reason`, `unknown_class`, `next_operation`, and `blocked_by`.

## Probe boundary

The debugger accepts a caller-owned JSON fixture. It does not patch the input
repository, run tests in another repository, or invoke a natural-language
recommendation path. Generated probes are typed artifacts with a stable
operation, capability, effect, input digest, and expected observation. The Go
runtime executes the typed IR operation against the fixture and serializes the
observation as evidence.

`forbidden_write` is a declared effect, but it is not an executable authority:
any fixture that selects it is `REFUTED` and reports the effect ID. This keeps
the capability/effect distinction observable without granting mutation power.

## Exact comparison

Every case carries `before_unknown_frontier_cardinality`,
`after_unknown_frontier_cardinality`, `before_blocked_by`, and
`after_blocked_by`. The report also retains the probe's narrowed frontier and
the full transition ledger. A zero after-cardinality is meaningful only when
the transition evidence is accepted. For identical
scenario/source/contract/toolchain/runner digests, exact before/after values
may support an improvement claim; no aggregate score is emitted.
