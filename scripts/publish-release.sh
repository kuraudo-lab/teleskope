#!/usr/bin/env bash
set -euo pipefail

: "${TAG:?TAG is required}"
: "${GH_REPO:?GH_REPO is required}"
[[ "$TAG" =~ ^v(0|[1-9][0-9]*)\.(0|[1-9][0-9]*)\.(0|[1-9][0-9]*)$ ]] || {
  echo 'Only stable vMAJOR.MINOR.PATCH tags are supported.' >&2
  exit 1
}
python3 scripts/release.py verify packages "${TAG#v}"

# A failed API request must not be interpreted as an absent release.
releases=$(gh api --paginate "repos/$GH_REPO/releases?per_page=100" --jq '.[] | {tag_name, draft}')
existing=$(printf '%s\n' "$releases" | jq -s --arg tag "$TAG" '[.[] | select(.tag_name == $tag)]')
if [[ $(jq 'length' <<< "$existing") != 0 ]]; then
  if [[ $(jq -r '.[0].draft' <<< "$existing") != true ]]; then
    echo "Release $TAG is already published; refusing to overwrite it." >&2
    exit 1
  fi
else
  gh release create "$TAG" --verify-tag --draft --title "$TAG" --generate-notes
fi

# Only drafts may be repaired on retry. Publish the already-tested artifacts.
gh release upload "$TAG" packages/*.tar.gz packages/*.zip packages/checksums.txt --clobber
download=$(mktemp -d)
trap 'rm -rf "$download"' EXIT
gh release download "$TAG" --dir "$download"
python3 scripts/release.py verify "$download" "${TAG#v}"
diff packages/checksums.txt "$download/checksums.txt"
[[ $(find "$download" -type f | wc -l) -eq 7 ]] || {
  echo 'Release must contain exactly six archives and checksums.txt.' >&2
  exit 1
}
gh release edit "$TAG" --draft=false --latest
