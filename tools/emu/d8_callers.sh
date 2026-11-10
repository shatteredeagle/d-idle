#!/bin/bash
# decomp8 (worker host): call sites of game.exe functions from the cached whole-binary listing. usage: d8_callers.sh ADDR...
A=$HOME/d2d-lab/d5/all.txt
[ -s $A ] || { echo "no $A: run d5_callers.sh once"; exit 1; }
for a in "$@"; do echo "=== callers of $a"; grep -E "call +0x0*$a\$" $A | head -${N:-20}; done
