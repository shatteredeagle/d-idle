#!/bin/bash
# restart.sh NAME : save+quit running d2d, start it on charselect with NAME and load it (Return), wait for ingame
~/d2d-lab/d2dstop.sh >/dev/null
~/d2d-lab/d2dstart.sh --seed 3 --start-screen charselect --start-name "$1" || exit 1
python3 ~/d2d-lab/ctl.py "key Return" >/dev/null
for i in $(seq 40); do python3 ~/d2d-lab/ctl.py state 2>/dev/null | grep -q "screen=ingame" && { echo ingame; break; }; sleep 1; done
sleep 2; python3 ~/d2d-lab/ctl.py "click 400 500" >/dev/null
