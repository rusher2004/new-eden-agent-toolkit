#!/usr/bin/env bash
# Sticky PR-comment helpers for the Version Gate workflow. Source this file;
# it requires GH_TOKEN, GH_REPO, and PR in the environment.
#
# A single comment is reused (found and edited by a hidden marker) so re-runs
# on synchronize don't accumulate duplicate Version Gate comments -- the
# ::error:: annotation and the required check carry the blocking signal.

MARKER="<!-- version-gate -->"

# find_sticky prints the id of the existing Version Gate comment, or nothing.
# Tolerant of SIGPIPE from `head` closing the pipe under `set -o pipefail`.
find_sticky() {
  gh api "repos/$GH_REPO/issues/$PR/comments" --paginate \
    --jq ".[] | select(.body | startswith(\"$MARKER\")) | .id" 2>/dev/null | head -n1 || true
}

# upsert_comment <body>: edit the sticky comment if present, else create it.
upsert_comment() {
  local id body
  id="$(find_sticky)"
  body="$(printf '%s\n%s' "$MARKER" "$1")"
  if [ -n "$id" ]; then
    gh api -X PATCH "repos/$GH_REPO/issues/comments/$id" -f body="$body" >/dev/null
  else
    gh api -X POST "repos/$GH_REPO/issues/$PR/comments" -f body="$body" >/dev/null
  fi
}

# resolve_comment <body>: edit an existing sticky only (no-op when none
# exists), so PRs that never failed the gate don't get a comment at all.
resolve_comment() {
  local id body
  id="$(find_sticky)"
  [ -z "$id" ] && return 0
  body="$(printf '%s\n%s' "$MARKER" "$1")"
  gh api -X PATCH "repos/$GH_REPO/issues/comments/$id" -f body="$body" >/dev/null
}
