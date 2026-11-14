#!/bin/bash
# decomp8: size of drlg-dump's tiles output per level (is the compare non-empty?). usage: d8_dump.sh SEED LEVEL
export D2_MPQ_DIR=${D2_EMU_DIR:-$HOME/d2d-lab/emu}/mpq D2_PATCH_INSTALLER=${D2_EMU_DIR:-$HOME/d2d-lab/emu}/patch/LODPatch_114d.exe
O=$HOME/d2d-lab/d8/dump; rm -rf $O; mkdir -p $O
$HOME/d2d-lab/build-decomp8/tools/drlg-dump/drlg-dump $D2_MPQ_DIR $1-$1 $2 $O tiles >/dev/null 2>&1
ls -l $O | tail -3; for f in $O/*; do echo "$f: $(wc -l < $f) lines"; head -c 300 $f; echo; done | head -20
