# Release policy

Only a human-triggered GitHub Actions workflow may publish a release. The
workflow binds the exact merged `main` commit and refuses an existing tag,
existing release, or reuse of a failed version. Before creating a tag it calls
the user-facing API route that exposes whether immutable releases are enabled;
it never asks the repository administration settings endpoint with
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
