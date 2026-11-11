#!/bin/bash
# covmissiles: print a read-only upstream file for reading. Usage: cm_cat.sh KEY [FROM] [TO]
# KEY: sd = skill_damage.py (palskill checkout), la = life_apply.py, mi = missiles.py (upstream main)
case $1 in
sd) f=$(ls $HOME/d2d-lab/pr/palskill/tools/emu/ski?l_damage.py | head -1) ;;
la) f=$(ls $HOME/d2d-lab/pr/palskill/tools/emu/life_apply.py | head -1) ;;
mi) f=$HOME/d2d-lab/src/tools/emu/missiles.py ;;
*) f=$HOME/d2d-lab/$1 ;;
esac
echo "== $f"
awk -v a="${2:-1}" -v b="${3:-100000}" 'NR>=a && NR<=b {print NR": "$0}' "$f"
