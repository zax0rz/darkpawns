#!/bin/bash
# Publish the package-manager channels AFTER the GitHub release is live, so
# no formula, manifest or bucket entry ever points at a download URL that
# 404s. GoReleaser generates every file into dist/ (skip_upload: true); this
# script is the only thing that pushes them:
#
#   Homebrew  darkpawns/homebrew-tap      Formula/darkpawns.rb     (default branch)
#   Scoop     darkpawns/scoop-bucket      darkpawns.json           (default branch)
#   winget    darkpawns/winget-pkgs       manifests/**             (branch = the tag)
#             and then the pull request to microsoft/winget-pkgs.
#
# Idempotent: file updates carry the current blob sha; the winget branch is
# created from the fork's default branch only when missing; an open PR is
# reused. Requires GH_TOKEN with contents/pull-requests rights on the three
# repositories, and RELEASE_TAG set.
set -euo pipefail

tag="${RELEASE_TAG:?RELEASE_TAG must be set}"
version="${tag#v}"

# put_file <owner> <repo> <path-in-repo> <branch-or-empty> <local-file> <message>
put_file() {
  local owner="$1" repo="$2" path="$3" branch="$4" file="$5" message="$6" sha=""
  if [ -n "$branch" ]; then
    sha=$(gh api "repos/$owner/$repo/contents/$path?ref=$branch" --jq .sha 2>/dev/null || true)
  else
    sha=$(gh api "repos/$owner/$repo/contents/$path" --jq .sha 2>/dev/null || true)
  fi
  local content
  content=$(base64 -w0 "$file")
  local payload
  payload=$(jq -n --arg message "$message" --arg content "$content" \
    --arg branch "$branch" --arg sha "$sha" \
    '{message: $message, content: $content}
     | if $branch != "" then . + {branch: $branch} else . end
     | if $sha != "" then . + {sha: $sha} else . end')
  gh api -X PUT "repos/$owner/$repo/contents/$path" --input - <<<"$payload" >/dev/null
  echo "channels: pushed $owner/$repo/$path${branch:+ @ $branch}"
}

# --- Homebrew -------------------------------------------------------------
put_file darkpawns homebrew-tap "Formula/darkpawns.rb" "" \
  "dist/homebrew/Formula/darkpawns.rb" \
  "darkpawns $version"

# --- Scoop ----------------------------------------------------------------
put_file darkpawns scoop-bucket "darkpawns.json" "" \
  "dist/scoop/darkpawns.json" \
  "darkpawns $version"

# --- winget: manifests on a versioned branch, then the pull request -------
fork=darkpawns/winget-pkgs
head_branch=$(gh api "repos/$fork" --jq .default_branch)
if ! gh api "repos/$fork/git/ref/heads/$tag" >/dev/null 2>&1; then
  head_sha=$(gh api "repos/$fork/git/ref/heads/$head_branch" --jq .object.sha)
  gh api "repos/$fork/git/refs" -f "ref=refs/heads/$tag" -f "sha=$head_sha" >/dev/null
  echo "channels: created winget branch $tag from $head_branch"
fi
while IFS= read -r manifest; do
  rel="${manifest#dist/winget/}"
  put_file darkpawns winget-pkgs "$rel" "$tag" "$manifest" "darkpawns $version"
done < <(find dist/winget -type f | sort)

head="darkpawns:${tag}"
url_base="https://github.com/zax0rz/darkpawns/releases/download/${tag}"
if existing=$(gh pr list --repo microsoft/winget-pkgs --head "$head" --state open --json url --jq '.[0].url') && [ -n "$existing" ]; then
  echo "channels: winget pull request already open: $existing"
  exit 0
fi
first_note=""
if [ "${WINGET_FIRST_SUBMISSION:-true}" = true ]; then
  first_note="

This is the first submission of this package identifier; happy to answer
anything during review."
fi
gh pr create \
  --repo microsoft/winget-pkgs \
  --base master \
  --head "$head" \
  --title "New Package: DarkPawns.DarkPawns version $version" \
  --body "#### Dark Pawns $tag

Automated manifest submission for DarkPawns.DarkPawns.

- Installer: \`${url_base}/\` (portable zip, SHA-256 pinned in the manifest)
- Publisher: Dark Pawns - https://darkpawns.org
- License: non-standard (MIT for original code plus CircleMUD/DikuMUD
  heritage) - https://github.com/zax0rz/darkpawns/blob/main/LICENSE${first_note}"
