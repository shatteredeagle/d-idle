#!/bin/bash
# usage: kfbuild.sh BINNAME   (from a worktree on the head box) sync the worktree to the worker ~/d2d-lab/killfix/src and build cmd/d2bot as ~/d2d-lab/loop/BINNAME
set -e
. "$(dirname "$0")/../env.sh"
O=$D2_WORKER_USER@$D2_WORKER_HOST; W=$(git rev-parse --show-toplevel)
ssh $O mkdir -p d2d-lab/killfix/src
rsync -a --delete --exclude .git --exclude '*.test' "$W"/ $O:d2d-lab/killfix/src/
scp -q "$W"/tools/d2dloop/pm.sh $O:d2d-lab/loop/
ssh $O "cd ~/d2d-lab/killfix/src && go build -o ~/d2d-lab/loop/$1 ./cmd/d2bot && echo built $1"
