#!/usr/bin/env bash
set -Eeuo pipefail

repository=${IMMUTABLE_RELEASE_REPO:-kimjooyoon/gooo-resolution-descent-debugger}
api_version=2026-03-10

fail_closed() {
  echo "immutable release guard: $1" >&2
  exit 1
}

case "${1:-}" in
  --contract)
    grep -q 'enabled' "$0" || fail_closed 'enabled field is not checked'
    grep -q 'immutable' "$0" || fail_closed 'release immutable field is not checked'
    grep -q 'releases/tags' "$0" || fail_closed 'public release route is not declared'
    echo 'immutable_release_guard_contract=CLOSED'
    ;;
  --release)
    tag=${2:-}
    expected_manifest=${3:-}
    [[ -n "$tag" && -n "$expected_manifest" ]] || fail_closed 'release tag and manifest are required'
    payload=$(gh api --method GET -H 'Accept: application/vnd.github+json' -H "X-GitHub-Api-Version: $api_version" "repos/$repository/releases/tags/$tag") || fail_closed "release $tag cannot be read"
    jq -e --arg tag "$tag" '.tag_name == $tag and .draft == false and .prerelease == false and .immutable == true' <<<"$payload" >/dev/null || fail_closed "release $tag is not published immutable"
    expected=$(jq -c '[.assets[] | {name,size,digest}] | sort_by(.name)' "$expected_manifest") || fail_closed 'manifest is invalid'
    manifest_name=$(basename "$expected_manifest")
    observed=$(jq -c --arg manifest "$manifest_name" '[.assets[] | select(.name != $manifest and .name != "SHA256SUMS") | {name,size,digest}] | sort_by(.name)' <<<"$payload") || fail_closed 'release asset response is invalid'
    [[ "$expected" == "$observed" ]] || fail_closed 'release asset names, sizes, or digests differ from manifest'
    manifest_observed=$(jq -c --arg manifest "$manifest_name" '[.assets[] | select(.name == $manifest) | {name,size,digest}]' <<<"$payload") || fail_closed 'manifest asset is missing'
    jq -n --arg tag "$tag" --argjson release_id "$(jq '.id' <<<"$payload")" --argjson assets "$observed" --argjson manifest "$manifest_observed" '{schema:"gooo/resolution-descent-debugger/release-audit/v1",tag:$tag,release_id:$release_id,immutable:true,assets:$assets,manifest:$manifest}'
    ;;
  *)
    fail_closed 'usage: --contract | --release TAG MANIFEST'
    ;;
esac
