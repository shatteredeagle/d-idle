#!/bin/bash
# runs on the worker host from the synced worktree. usage: run.sh LABEL GROUP ONLY REPEAT
export PATH=$PATH:$HOME/.local/bin
B=$HOME/d2d-lab/fork/bin/d2d.lab-cheats3
go build -o /tmp/livedrills2-d2drill ./cmd/d2drill || exit 1
mkdir -p /tmp/livedrills2-out
D2D_AGENT=livedrills2 D2D_BIN=$B tools/agentrun.sh livedrills2 --ttl 200 -- tools/d2dslot.sh run -n 3 -t 180 -- /tmp/livedrills2-d2drill -group "$2" -only "$3" -repeat "$4" -label "$1" -out /tmp/livedrills2-out -bin $B
