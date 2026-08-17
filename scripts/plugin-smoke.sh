#!/usr/bin/env bash
# Runs the custom-gcl binary (built by `golangci-lint custom`) on example/ and
# checks that the module plugin reports the same findings as the standalone
# binary, whose golden output is example/expected.txt.
#
# Only the set of "file:line analyzer" pairs is compared: golangci-lint prints
# a different prefix and orders issues its own way, and the column is not part
# of its text output contract.
set -euo pipefail
root=$(cd "$(dirname "$0")/.." && pwd)
gcl="$root/custom-gcl"
if [[ ! -x "$gcl" ]]; then
  echo "plugin-smoke: $gcl not built; run 'golangci-lint custom' first" >&2
  exit 1
fi
if ! command -v jq >/dev/null 2>&1; then
  echo "plugin-smoke: jq is required to read the JSON output" >&2
  exit 1
fi
tmp=$(mktemp -d)
trap 'rm -rf "$tmp"' EXIT

cd "$root/example"
code=0
"$gcl" run -c "$root/.golangci.dogfood.yml" --output.json.path "$tmp/issues.json" ./... || code=$?
# Exit code 1 means "issues found", which is what this test expects; anything
# else is a runner failure.
if [[ "$code" -gt 1 ]]; then
  echo "plugin-smoke: custom-gcl run failed with exit code $code" >&2
  exit "$code"
fi

# expected.txt lines look like "file.go:6:11: name: message".
sed -E 's/^([^:]+):([0-9]+):[0-9]+: ([a-z][a-z0-9]*):.*/\1:\2 \3/' "$root/example/expected.txt" | sort -u > "$tmp/want"
# Filenames may carry a directory prefix depending on relative-path-mode; every
# example file is at the module root, so the base name is lossless.
jq -r '.Issues[]? | "\(.Pos.Filename | split("/") | last):\(.Pos.Line) \(.Text | split(":")[0])"' \
  "$tmp/issues.json" | sort -u > "$tmp/got"

if ! diff -u "$tmp/want" "$tmp/got"; then
  echo "plugin-smoke: module plugin findings differ from example/expected.txt" >&2
  exit 1
fi
echo "plugin-smoke: ok ($(wc -l < "$tmp/got" | tr -d ' ') findings match example/expected.txt)"
