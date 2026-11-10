#!/usr/bin/env bash
# Control of a DEV play sandbox container (docs/dev-reset.md). Runs from anywhere `labrun` works; the work happens on the worker host.
#   tools/dev-sandbox.sh reset   --to C [--class ID] [--name NAME] [--yes | --dry-run]   archive the hero and bot state, fresh hero, restart
#   tools/dev-sandbox.sh restore --to C [--from ARCHIVE|last] [--yes | --dry-run]        put an archive back (the current state is archived first)
#   tools/dev-sandbox.sh list    --to C                                                   the archives of that sandbox
# New operations are new subcommands: a `cmd_<name>` function in tools/dev-sandbox-host.sh plus a line here (docs/dev-reset.md).
set -uo pipefail
repo=$(cd "$(dirname "$0")/.." && pwd)
. "$repo/tools/env.sh"
cd "$repo"
[ $# -gt 0 ] && case $1 in -h | --help) sed -n 2,6p "$0"; exit 0 ;; esac
sub=${1:-}
case $sub in reset | restore | list) shift ;; *) echo "dev-sandbox: subcommand must be reset, restore or list (see --help)" >&2; exit 2 ;; esac
labrun "${D2_LABRUN_HOST:?set D2_LABRUN_HOST in local.env (the labrun machine name)}" -- tools/dev-sandbox-host.sh "$sub" "$@"
