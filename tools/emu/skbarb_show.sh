#!/bin/bash
# skbarb: print the calc cells of the clvl-30, no-synergy cases at skill levels 1, 5, 10, 20, 30 from a run of skbarb_run.sh.
#   tools/emu/run-on-worker.sh skbarb_show.sh 1 > out.txt        (run number = the SEED argument of skbarb_run.sh)
S=${D2_SCRATCH:-/tmp/d2agent-skbarb}
awk -v skills=126 'BEGIN{n=split("1 5 10 20 30",L," ")}
$1<2400 { id=$1; k=int(id/80); r=id%80; if (r%2==0) { lvl=r/2+1; for(i=1;i<=n;i++) if (lvl==L[i]) { printf "%d L%d:", 126+k, lvl; for(f=2;f<=NF;f++) printf " %s", $f; printf "\n" } } }' "$S/run${1:-1}/game.tsv"
