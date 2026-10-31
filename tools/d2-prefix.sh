#!/bin/sh
# Stub: the real script is in the private ops repo (docs/process/public-split.md).
exec "$(dirname "$0")/ops.sh" run d2-prefix.sh "$@"
