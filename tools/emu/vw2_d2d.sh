#!/bin/bash
# verifywaypoint2 (the worker host): run d2d's real waypoint code (test_game --waypoint-refuse, the PRE-fix binary
# built for the earlier waypoint card, upstream main of 2026-10-31) on the cases vw2_game.py wrote, and compare each
# case's refused flag with game.exe's (vw2_game_out.txt).
#   tools/emu/run-on-worker.sh --ttl 1500 vw2_game.py 23 100000 > $D2_SCRATCH/vw2_game.txt 2>&1
#   tools/emu/run-on-worker.sh --ttl 1500 vw2_d2d.sh > $D2_SCRATCH/vw2_d2d.txt 2>&1
set -u
S=${D2_SCRATCH:-/tmp/d2agent-verifywaypoint2}
BIN=$HOME/d2d-lab/pr/wp-scratch/test_game.before
export D2_MPQ_DIR=$HOME/d2d-lab/pr/wp-scratch/data
[ -x "$BIN" ] || { echo "missing $BIN"; exit 3; }
cd "$HOME/d2d-lab/pr/wp-scratch" || exit 3
# the d2d runner reads at most the first 4 fields + arrival times of each case line: the same file format as wp_refuse.py
"$BIN" --waypoint-refuse "$S/vw2_cases.txt" 2> "$S/vw2_d2d.err" | grep -E '^[0-9]+ [01]$' > "$S/vw2_d2d_out.txt"
echo "d2d lines: $(wc -l < "$S/vw2_d2d_out.txt")  game lines: $(wc -l < "$S/vw2_game_out.txt")"
head -3 "$S/vw2_d2d.err"
# compare: ids are the line index; count differing flags and the two directions
paste -d' ' "$S/vw2_d2d_out.txt" "$S/vw2_game_out.txt" | awk '
  { if ($1 != $3) bad_id++; n++; if ($2 != $4) { diff++; if ($2 == 1 && $4 == 0) d2d_only++; else game_only++ } }
  END { printf "cases %d id-mismatch %d d2d-vs-game differ %d (d2d refuses game travels %d, game refuses d2d travels %d)\n", n, bad_id, diff, d2d_only, game_only }'
# my two rules against the two real programs, per case: rule D (arrival window) vs d2d, rule B (field + 10000 > now) vs game.exe
paste -d' ' "$S/vw2_d2d_out.txt" "$S/vw2_game_out.txt" "$S/vw2_rules_out.txt" | awk '
  { n++; if ($2 != $6) dd++; if ($4 != $7) bg++ }
  END { printf "rule D vs d2d: %d of %d cases differ; rule B vs game.exe: %d of %d cases differ\n", dd + 0, n, bg + 0, n }'
