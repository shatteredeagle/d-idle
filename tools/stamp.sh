#!/bin/bash
# Print the build stamp for the checkout: $D2_VERSION if set, else `git describe --always --dirty`, else (no .git,
# e.g. a tree copied by labrun) "tree-" + a short content hash of the Go sources. Used for -ldflags -X in live.sh.
set -eu
repo="$(cd "$(dirname "$0")/.." && pwd)"
if [ -n "${D2_VERSION:-}" ]; then echo "$D2_VERSION"; exit 0; fi
if v="$(git -C "$repo" describe --always --dirty 2>/dev/null)" && [ -n "$v" ]; then echo "$v"; exit 0; fi
h="$(cd "$repo" && find cmd internal go.mod -type f \( -name '*.go' -o -name go.mod \) 2>/dev/null | LC_ALL=C sort | xargs sha256sum | sha256sum | cut -c1-8)"
echo "tree-$h"
