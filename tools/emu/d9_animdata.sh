#!/bin/bash
# decomp9 (worker host): write the AnimData.d2 of the 1.14d stack to stdout (binary). usage: d9_animdata.sh > animdata.d2
D=${D2_DATA:-$HOME/d2d-lab/data}
S=${D2_SCRATCH:-/tmp/d2agent-$USER}/mpqlink
M=${MPQCAT:-$HOME/d2d-lab/src/build/tools/mpq-cat/mpq-cat}
mkdir -p "$S"
for f in "$D"/*; do ln -sf "$f" "$S/$(basename "$f")"; done
ln -sf "$D/Patch_D2.mpq" "$S/patch_d2.mpq"
"$M" "$S" 'data\global\animdata.d2'
