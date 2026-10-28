#!/bin/bash
# Open (or update) the winget-pkgs pull request for one release.
#
# GoReleaser committed the generated manifests to a versioned branch of the
# darkpawns/winget-pkgs fork; this script opens the pull request against
# microsoft/winget-pkgs. The first submission for a new package identifier
# usually needs human review by the winget team — that is expected; the PR
# body says so and links the submission docs.
set -euo pipefail

tag="${RELEASE_TAG:?RELEASE_TAG must be set}"
head="darkpawns:${tag}"
base="master"
url_file="https://github.com/zax0rz/darkpawns/releases/download/${tag}"
first_note=""
if [ "${WINGET_FIRST_SUBMISSION:-true}" = true ]; then
  first_note="

This is the first submission of this package identifier; happy to answer
anything during review."
fi
body="#### Dark Pawns ${tag}

Automated manifest submission for DarkPawns.DarkPawns.

- Installer: \`${url_file}/\` (portable zip, SHA-256 pinned in the manifest)
- Publisher: Dark Pawns - https://darkpawns.org
- License: non-standard (MIT for original code plus CircleMUD/DikuMUD
  heritage) - https://github.com/zax0rz/darkpawns/blob/main/LICENSE${first_note}"

if existing=$(gh pr list --repo microsoft/winget-pkgs --head "$head" --state open --json url --jq '.[0].url') && [ -n "$existing" ]; then
  echo "winget pull request already open: $existing"
  exit 0
fi
gh pr create \
  --repo microsoft/winget-pkgs \
  --base "$base" \
  --head "$head" \
  --title "New Package: DarkPawns.DarkPawns version ${tag#v}" \
  --body "$body"
