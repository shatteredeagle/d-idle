#!/bin/bash
# verifyteleroom: reproduce the SKTELE author's numbers with sk_tele_spot.py (same seed), for the comparison section.
#   tools/emu/run-on-worker.sh --ttl 1500 vtr_cmp.sh 20000 7
n=${1:-20000}
seed=${2:-7}
here=$(cd "$(dirname "$0")" && pwd)
out=${D2_SCRATCH:-/tmp}
bash "$here/run-py.sh" sk_tele_spot.py gen "$n" "$seed" > "$out/cases.tsv"
bash "$here/run-py.sh" sk_tele_spot.py game "$out/cases.tsv" > "$out/game.tsv"
bash "$here/run-py.sh" sk_tele_spot.py d2d "$out/cases.tsv" > "$out/d2d.tsv"
bash "$here/run-py.sh" sk_tele_spot.py diff "$out/game.tsv" "$out/d2d.tsv" "$out/cases.tsv"
