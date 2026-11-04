#!/bin/sh
# Pull the event lines the floor tables need from finished gate runs on the worker host.
# Read-only (grep only). Usage, from inside the repo:  tools/floors/extract.sh OUTDIR RUNDIR...
# RUNDIR is a run directory under the worker host's gate-run folder (the one with logs/bot-0.log).
# Output per run: OUTDIR/<run>.ev.jsonl (minute, Entered, chicken lines) and OUTDIR/<run>.super.log
out=$1; shift
mkdir -p "$out"
for r in "$@"; do
  n=$(basename "$r")
  labrun orion -- sh -c "cd $r/logs && grep -E '\"msg\":\"(minute|Entered [^\"]*)\"|chose chicken first|Save and Exit requested' bot-0.log" 2>/dev/null | grep '^{' > "$out/$n.ev.jsonl"
  labrun orion -- sh -c "cat $r/logs/supervisor.log" 2>/dev/null | grep '^20' > "$out/$n.super.log"
done
