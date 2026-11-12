#!/bin/bash
# decomp5 (worker host): the function start (previous `push ebp; mov ebp, esp` or a ret/int3 boundary) of each address: d5_funcof.sh ADDR...
A=$HOME/d2d-lab/d5/all.txt
for a in "$@"; do
  awk -v t=$(printf '%08x' 0x$a) 'BEGIN{start=""} { if ($1 > t) {print t " in func " start; exit} if ($2=="push" && $3=="ebp") {getline nxt; if (nxt ~ /mov +ebp, esp/) start=$1} }' $A
done
