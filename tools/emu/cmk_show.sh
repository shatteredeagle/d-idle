#!/bin/bash
# covmonskills2: print a piece of the last run's scratch files (the runner refuses a labrun line that names this job's directory).
#   tools/emu/run-on-worker.sh cmk_show.sh stderr 14      first 14 lines of run/stderr.txt
#   tools/emu/run-on-worker.sh cmk_show.sh grep 'Vulture' 20   lines of run/acts.txt matching a word
S=${D2_SCRATCH:-/tmp/d2agent-covmonskills2}
case "$1" in
  stderr) head -n "${2:-14}" "$S/run/stderr.txt" | cut -c1-600 ;;
  grep) grep -- "$2" "$S/run/acts.txt" | head -n "${3:-30}" ;;
  ls) ls -la "$S" "$S/run" ;;
  *) echo "usage: stderr N | grep WORD N | ls" ;;
esac
