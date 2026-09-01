# Release policy

Only a human-triggered GitHub Actions workflow may publish a release. The
operator records `enabled=true` from the user-facing GitHub API in
[`immutable-releases-observation-v1.json`](immutable-releases-observation-v1.json).
The workflow binds the exact merged `main` commit and refuses an existing tag,
existing release, or reuse of a failed version. It consumes that receipt and
does not query a repository administration settings endpoint with
`GITHUB_TOKEN`.

The publication order is strict:

1. create the annotated tag at the exact merged commit;
2. create a draft release;
3. upload the source, binary, conformance report, metrics, version, manifest,
   and checksum assets;
4. publish the draft exactly once;
5. read the public release API and verify `draft=false`, `prerelease=false`,
   `immutable=true`, and exact asset names, sizes, and SHA-256 digests.

No public tag or release is overwritten or deleted. Automatic commit, push,
merge, and release authority in the product report remain zero; the workflow's
publication is a separate, explicitly human-triggered authority step.
