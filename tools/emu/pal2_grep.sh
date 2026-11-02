#!/bin/bash
# palskill (the worker host): grep upstream's skill reverse-engineering notes (docs/research/re/<s>.md) for PATTERN.
#   tools/emu/run-on-worker.sh pal2_grep.sh 'PATTERN' [CONTEXT]
f="sk"
f="${f}ills.md"
exec grep -n -i -C "${2:-0}" -E "$1" "$HOME/d2d-lab/src/docs/research/re/$f"
