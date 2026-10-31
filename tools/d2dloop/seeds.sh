#!/bin/bash
# usage: seeds.sh PREFIX BOTBIN HOURS SEED...   runs m.sh once per seed, one after the other (on the worker)
# whole script (lock waits included) runs as one registered process group: tools/agentrun.sh, reap with tools/agentreap.sh
if [ -z "${AGENTRUN_ON:-}" ] && [ -x "$HOME/d2d-lab/bin/agentrun.sh" ]; then
  exec "$HOME/d2d-lab/bin/agentrun.sh" "${D2D_AGENT:-loop}" -- env AGENTRUN_ON=1 "$0" "$@"
fi
cd ~/d2d-lab/loop; P=$1; B=$2; H=$3; shift 3
for s in "$@"; do ./m.sh $P$s $B $s $H; done
