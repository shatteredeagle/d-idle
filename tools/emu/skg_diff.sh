#!/bin/bash
# skground (the worker host): the whole Meteor-blast differential. Files go to $D2_SCRATCH.
#   tools/emu/run-on-worker.sh --ttl 2400 skg_diff.sh SEED N
set -e
here=$(cd "$(dirname "$0")" && pwd)
S=${D2_SCRATCH:-/tmp/d2agent-skground}
mkdir -p $S
SEED=${1:-1}
N=${2:-40000}
export D2_MPQ_DIR=${D2_MPQ_DIR:-/tmp/ug-emu}
export D2_PATCH_INSTALLER=${D2_PATCH_INSTALLER:-$HOME/Downloads/LODPatch_114d.exe}
bash $here/skg_build.sh
bash $here/run-py.sh skg_blast_diff.py gen $SEED $N $S/blast.cases
bash $here/run-py.sh skg_blast_diff.py game $S/blast.cases $S/blast.game
nice $S/skg_test --blast $S/blast.cases > $S/blast.d2d
bash $here/run-py.sh skg_blast_diff.py cmp $S/blast.cases $S/blast.game $S/blast.d2d
