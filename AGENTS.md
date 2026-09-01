# Development boundary

This repository is independent from every sibling under `/Users/alice/meta-go`.
The `.gooo` source is the semantic authority for the resolution lattice,
probe capabilities/effects, descent rules, claim transitions, and fixed
denominator.

The debugger accepts caller-owned fixtures and writes generated probes and
receipts only to caller-owned output. It never mutates its input repository,
opens or merges a pull request, or grants automatic commit, push, merge, or
release authority. `REFUTED > UNKNOWN > CLOSED`; unknown records retain all
six required fields and the exact blocked frontier.

Local build, test, vet, conformance, and integration execution is intentionally
not part of development. GitHub Actions is the verification authority.
