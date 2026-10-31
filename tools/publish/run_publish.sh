#!/bin/sh
# Stub: the publisher is in the private ops repo; d2-publish.service keeps calling this path.
exec "$(dirname "$0")/../ops.sh" run publish/run_publish.sh "$@"
