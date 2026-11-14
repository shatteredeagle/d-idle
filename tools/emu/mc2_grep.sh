#!/bin/bash
# MERC2: callers of game.exe addresses in the linear disassembly: mc2_grep.sh ADDR [ADDR..]  (hex, no 0x); "imm" ADDR greps any operand
for a in "$@"; do echo "== $a"; grep -E -i "call +(0x)?0*$a\$" "$D2_SCRATCH/all.txt" | head -40; done
