#!/usr/bin/env bash
# Verifies go.mod pins golang.org/x/tools to the version pinned by the golangci-lint
# release named in .custom-gcl.yml, so `golangci-lint custom` does not upgrade x/tools
# underneath golangci-lint.
set -euo pipefail
root=$(cd "$(dirname "$0")/.." && pwd)
gcl=$(sed -n 's/^version: *//p' "$root/.custom-gcl.yml" | tr -d '"' | head -1)
ours=$(go list -m -f '{{.Version}}' golang.org/x/tools)
theirs=$(curl -fsSL "https://raw.githubusercontent.com/golangci/golangci-lint/$gcl/go.mod" | awk '$1=="golang.org/x/tools"{print $2}')
if [[ -z "$theirs" ]]; then echo "deps-check: could not read golangci-lint $gcl go.mod" >&2; exit 1; fi
if [[ "$ours" != "$theirs" ]]; then
  echo "deps-check: golang.org/x/tools is $ours but golangci-lint $gcl pins $theirs; bump both together" >&2
  exit 1
fi
echo "deps-check: ok (x/tools $ours matches golangci-lint $gcl)"
