#!/bin/bash
# runs on the worker host from the synced worktree. usage: run.sh LABEL GROUP ONLY REPEAT
export PATH=$PATH:$HOME/.local/bin
B=$HOME/d2d-lab/fork/bin/d2d.lab-cheats3
go build -o /tmp/livedrills3-d2drill ./cmd/d2drill || exit 1
mkdir -p /tmp/livedrills3-out
D2D_AGENT=livedrills3 D2D_BIN=$B tools/agentrun.sh livedrills3 --ttl 175 -- tools/d2dslot.sh run -n 3 -t 170 -- /tmp/livedrills3-d2drill -group "$2" -only "$3" -repeat "$4" -label "$1" -out /tmp/livedrills3-out -bin $B
