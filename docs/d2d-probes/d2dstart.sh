#!/bin/bash
# our own helper (not d2d code). usage: d2dstart.sh [d2d args...]   one instance only, niced, private socket
cd ~/d2d-lab
pgrep -x d2d >/dev/null && { echo "d2d already running"; exit 1; }
rm -f ~/d2d-lab/d2d.sock
nohup ~/d2d-lab/start.sh "$@" > ~/d2d-lab/probe-run.log 2>&1 &
for i in $(seq 40); do [ -S ~/d2d-lab/d2d.sock ] && python3 ~/d2d-lab/ctl.py ping 2>/dev/null | grep -q '^ok' && { echo up; exit 0; }; sleep 0.5; done
echo "no socket"; tail -5 ~/d2d-lab/probe-run.log; exit 1
