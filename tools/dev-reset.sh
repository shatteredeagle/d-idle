#!/usr/bin/env bash
# Thin entry for the first sandbox operation: tools/dev-reset.sh --to d2play-20261108 [--class ID] [--name NAME] [--yes | --dry-run]
# Same as `tools/dev-sandbox.sh reset ...` (docs/dev-reset.md). Restore and list are `tools/dev-sandbox.sh restore|list`.
exec "$(dirname "$0")/dev-sandbox.sh" reset "$@"
