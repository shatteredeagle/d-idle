#!/bin/bash
# prwarp (worker host): every differential of the claim commit and of the parent, one after the other. Needs build-pw at the claim commit.
here=$(cd "$(dirname "$0")" && pwd)
O=$HOME/d2d-lab/pw
export D2_MPQ_DIR=/tmp/ug-emu D2_PATCH_INSTALLER=$HOME/Downloads/LODPatch_114d.exe
bash $here/pw_diff.sh save claim
echo "=== test_warp_arrival (claim), no argument"; $O/bin/claim; echo "rc=$?"
bash $here/pw_diff.sh synth claim 1 20000
bash $here/pw_diff.sh synth parent 31 20000
bash $here/pw_diff.sh synth claim 31 20000
bash $here/pw_full.sh probe claim
for s in 5 82 7 11 23 4242; do
  bash $here/pw_full.sh full $HOME/d2d-lab/pv3/bin/pv3_usewarp-r-parent $s
  bash $here/pw_full.sh full $O/bin/usewarp-claim $s
done
bash $here/pw_full.sh full $HOME/d2d-lab/pv3/bin/pv3_usewarp-r-parent 5 40 136
bash $here/pw_full.sh full $O/bin/usewarp-claim 5 40 136
echo ALLDONE
