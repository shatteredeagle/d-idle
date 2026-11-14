#!/bin/bash
# prprep2 (worker host): the whole ctest suite on a built candidate tree.   pp_ctest.sh light3|cold4
export PATH=$HOME/.local/bin:$PATH D2_MPQ_DIR=${D2_EMU_DIR:-$HOME/d2d-lab/emu}/mpq D2_PATCH_INSTALLER=${D2_EMU_DIR:-$HOME/d2d-lab/emu}/patch/LODPatch_114d.exe
n=$1; B=$HOME/d2d-lab/build-pp-$n
cd $HOME/d2d-lab/wt-pp-$n; git log --format='%h %s' -1 | cut -c1-70; git status --short | head -2
nice -n 19 cmake --build $B -j8 > $B.all.log 2>&1; echo "full build rc=$?"; tail -1 $B.all.log | cut -c1-150
cd $B && timeout 2400 ctest -j8 --output-on-failure > $B.ctest.log 2>&1; echo "ctest rc=$?"
grep -E "tests passed|tests failed|Failed|\*\*\*" $B.ctest.log | head -12
