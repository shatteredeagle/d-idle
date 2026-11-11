#!/bin/bash
# palskill (the worker host): show the first N lines of tag TAG side by side (case, game, d2d) for a seed.
#   tools/emu/run-on-worker.sh pal2_show.sh TAG SEED N
S=${D2_SCRATCH:-/tmp/d2agent-palskill}
tag=$1; seed=$2; n=${3:-10}
grep "^$tag " "$S/pal2-c$seed.tsv" | head -n "$n" > "$S/show-c"
grep "^$tag " "$S/pal2-g$seed.tsv" | head -n "$n" > "$S/show-g"
grep "^$tag " "$S/pal2-b$seed.tsv" | head -n "$n" > "$S/show-b"
paste -d '|' "$S/show-c" "$S/show-g" "$S/show-b"
