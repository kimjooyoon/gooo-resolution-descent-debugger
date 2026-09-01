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
    grep -q '/immutable-releases' "$0" || fail_closed 'setting route is not declared'
    grep -q 'RELEASE_USER_TOKEN' "$0" || fail_closed 'user token boundary is not declared'
    grep -q 'enabled' "$0" || fail_closed 'enabled field is not checked'
    grep -q 'immutable' "$0" || fail_closed 'release immutable field is not checked'
    echo 'immutable_release_guard_contract=CLOSED'
    ;;
  --setting)
    [[ -n "${RELEASE_USER_TOKEN:-}" ]] || fail_closed 'RELEASE_USER_TOKEN is required; GITHUB_TOKEN is not accepted'
    payload=$(GH_TOKEN="$RELEASE_USER_TOKEN" gh api --method GET -H 'Accept: application/vnd.github+json' -H "X-GitHub-Api-Version: $api_version" "repos/$repository/immutable-releases") || fail_closed 'user API immutable-releases endpoint is unavailable'
    enabled=$(jq -r 'if type == "object" and has("enabled") then (.enabled | tostring) else "missing" end' <<<"$payload") || fail_closed 'setting response is not valid JSON'
    [[ "$enabled" == "true" ]] || fail_closed "repository immutable releases setting is $enabled"
    echo 'repository_immutable_releases=true'
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
    fail_closed 'usage: --contract | --setting | --release TAG MANIFEST'
    ;;
esac
