#!/bin/sh
exec "$(dirname "$0")/../deploy/d2d-container/smoke.sh" "$@"
