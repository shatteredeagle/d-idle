#!/bin/bash
# prprep (worker host): test_boss_cold_length with the claim (passes) and with the claim's components reverted (must fail).
export PATH=$HOME/.local/bin:$PATH D2_MPQ_DIR=${D2_EMU_DIR:-$HOME/d2d-lab/emu}/mpq D2_PATCH_INSTALLER=${D2_EMU_DIR:-$HOME/d2d-lab/emu}/patch/LODPatch_114d.exe
HERE=$(cd "$(dirname "$0")" && pwd)
T=$HOME/d2d-lab/wt-pp-cold; B=$HOME/d2d-lab/build-pp-cold
cd $T
bash $HERE/pp_build.sh cold test_boss_cold_length | tail -3
$B/tests/test_boss_cold_length; echo "with the claim rc=$?"
git diff HEAD~1 HEAD -- components > /tmp/pp_cold_claim.patch
git apply -R /tmp/pp_cold_claim.patch && bash $HERE/pp_build.sh cold test_boss_cold_length | tail -1
$B/tests/test_boss_cold_length 2>&1 | tail -2; echo "claim reverted rc=${PIPESTATUS[0]}"
git apply /tmp/pp_cold_claim.patch && git status --short
