#!/bin/bash
# Worker host (card geararmore): D census of the gear affixes carrying the attack-rating properties
# ignore-ac (115), reduce-ac (116), dmg-ac (120), att-demon (123), att-undead (124), att-mon% (179).
#   tools/emu/run-on-worker.sh gar_census.sh > /tmp/d2agent-NAME/census.txt 2>&1
# Counts rows per table whose prop/aprop/mod cells name one of the property codes (patched 1.14d stack).
set -e
here=$(dirname "$0")
out=${D2_SCRATCH:-/tmp/d2agent-$USER}
for t in uniqueitems setitems magicprefix magicsuffix runes gems automagic cubemain; do
  bash "$here/mpq-table.sh" "$t" > "$out/gar_$t.txt"
done
for t in uniqueitems setitems magicprefix magicsuffix runes gems automagic cubemain; do
  echo "== $t rows: $(wc -l < "$out/gar_$t.txt")"
  for c in ignore-ac reduce-ac dmg-ac att-demon att-undead 'att-mon%'; do
    n=$(awk -F'\t' -v c="$c" 'NR>1{for(i=1;i<=NF;i++) if($i==c){k++;break}} END{print k+0}' "$out/gar_$t.txt")
    echo "  $c: $n"
  done
done
echo "== rows (name first cell) carrying one of the codes"
for t in uniqueitems setitems magicprefix magicsuffix runes gems automagic; do
  awk -F'\t' -v t="$t" 'NR>1{for(i=2;i<=NF;i++) if($i=="ignore-ac"||$i=="reduce-ac"||$i=="dmg-ac"||$i=="att-demon"||$i=="att-undead"||$i=="att-mon%"){print t": "$1" ("$i")";break}}' "$out/gar_$t.txt"
done
