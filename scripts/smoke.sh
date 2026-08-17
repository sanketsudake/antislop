#!/usr/bin/env bash
# Runs the antislop binary on example/ and compares with the golden expected.txt.
# Usage: scripts/smoke.sh [--update]
set -euo pipefail
root=$(cd "$(dirname "$0")/.." && pwd)
bin="$root/bin/antislop"
cd "$root/example"
actual=$( "$bin" ./... 2>&1 | sed "s#^$PWD/##" | sort || true )
if [[ "${1:-}" == "--update" ]]; then
  printf '%s\n' "$actual" > expected.txt
  echo "updated example/expected.txt"
  exit 0
fi
diff -u expected.txt <(printf '%s\n' "$actual") && echo "smoke: ok"
