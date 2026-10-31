#!/bin/sh
# Runs the PINNED OpenSpec CLI without installing anything: telemetry and update ping off.
# Use only for `validate --all --strict` and `archive <id> --yes`; never `init` or `update`
# (they write .claude/skills, slash commands and instruction files that compete with AGENTS.md).
# Run from the repo root; it only touches openspec/. Needs node >= 20.19 on PATH.
set -eu
case "${1:-}" in init|update) echo "openspec.sh: '$1' is blocked (would write skills/commands outside openspec/)" >&2; exit 2;; esac
export OPENSPEC_TELEMETRY=0 DO_NOT_TRACK=1 OPENSPEC_NO_UPDATE_CHECK=1
exec npx -y @fission-ai/openspec@1.15.0 "$@"
