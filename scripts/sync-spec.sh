#!/usr/bin/env bash
# Copies the A2UI specification files this module embeds from a local checkout of
# https://github.com/a2ui-project/a2ui (formerly google/A2UI).
#
# Usage: scripts/sync-spec.sh /path/to/A2UI
#
# v0.9.1 has no directory here: upstream's v0_9_1/json differs from v0_9/json only by the
# "version" const becoming an enum, and the v09 package applies that patch in memory. The
# guard below fails loudly if upstream ever diverges further, because the patch would then be
# wrong. Both the source check and that guard run before anything is copied or deleted, so a
# wrong path or a diverged upstream can never leave spec/ half-emptied.
set -euo pipefail

src="${1:?usage: $0 <path-to-A2UI-checkout>}"
here="$(cd "$(dirname "$0")/.." && pwd)"
specsrc="$src/specification"

for dir in v0_9/json v1_0/json v0_9_1/json; do
  if [ ! -d "$specsrc/$dir" ]; then
    echo "ERROR: $specsrc/$dir not found; expected $src to be a checkout of github.com/a2ui-project/a2ui" >&2
    exit 1
  fi
done

# Guard: v0_9_1 must equal v0_9 once the enum is folded back to the const.
tmp="$(mktemp -d)"
trap 'rm -rf "$tmp"' EXIT
cp -R "$specsrc/v0_9_1/json" "$tmp/json"
perl -pi -e 's/"enum": \["v0\.9", "v0\.9\.1"\]/"const": "v0.9"/' "$tmp"/json/*.json
if ! diff -r "$specsrc/v0_9/json" "$tmp/json" >/dev/null; then
  echo "ERROR: v0_9_1/json differs from v0_9/json beyond the version enum; update the v09 patch" >&2
  diff -r "$specsrc/v0_9/json" "$tmp/json" >&2 || true
  exit 1
fi

for major in v0_9 v1_0; do
  dst="$here/spec/$major"
  rm -rf "$dst/json" "$dst/catalogs" "$dst/testdata"
  mkdir -p "$dst/json" "$dst/catalogs/basic" "$dst/testdata/examples"
  cp "$specsrc/$major/json/"*.json "$dst/json/"
  cp "$specsrc/$major/catalogs/basic/catalog.json" "$dst/catalogs/basic/"
  if [ -f "$specsrc/$major/catalogs/basic/rules.txt" ]; then
    cp "$specsrc/$major/catalogs/basic/rules.txt" "$dst/catalogs/basic/"
  fi
  cp "$specsrc/$major/catalogs/basic/examples/"*.json "$dst/testdata/examples/"
done

printf '%s %s\n' "$(git -C "$src" rev-parse --short HEAD)" "$(git -C "$src" log -1 --format=%cs)" > "$here/spec/SOURCE"
echo "synced spec from $(cat "$here/spec/SOURCE"); now run: go test ./..."
