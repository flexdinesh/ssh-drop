#!/usr/bin/env bash
# Temporary workaround: GoReleaser renders "# No zap stanza required" in casks
# without a zap stanza, which Homebrew's Cask/ZapNotRequired cop rejects in tap
# CI. This rewrites it to `zap :not_required`.
#
# Revert once GoReleaser ships zap.not_required:
# https://github.com/goreleaser/goreleaser/pull/7253
# Revert steps: docs/release.md, "Cask Zap Workaround".
set -euo pipefail

cask="${1:?usage: scripts/patch-homebrew-cask.sh <cask.rb>}"
comment='  # No zap stanza required'

if ! grep -qxF "$comment" "$cask"; then
  echo "$cask: GoReleaser zap comment not found; revisit this workaround" >&2
  exit 1
fi

sed "s/^  # No zap stanza required\$/  zap :not_required/" "$cask" > "$cask.tmp"
mv "$cask.tmp" "$cask"
