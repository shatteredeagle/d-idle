#!/bin/bash
# prprep (worker host): rebase the two local candidates onto current upstream main in their own worktrees (fork checkout).
set -u
F=$HOME/d2d-lab/fork/src
cd $F
git fetch upstream 2>&1 | tail -1
mk() { # name branch commits...
  n=$1; b=$2; shift 2
  W=$HOME/d2d-lab/wt-pp-$n
  [ -d $W ] || git worktree add -b $b $W upstream/main 2>&1 | tail -1
  cd $W
  for c in "$@"; do
    git -c user.name=Claude -c user.email=noreply@anthropic.com cherry-pick -x $c >/dev/null 2>&1 || {
      if [ "$(git diff --name-only --diff-filter=U)" = "tools/emu/README.md" ]; then
        sed -i -e '/^<<<<<<< /d' -e '/^=======$/d' -e '/^>>>>>>> /d' tools/emu/README.md
        git add tools/emu/README.md
        GIT_EDITOR=true git -c user.name=Claude -c user.email=noreply@anthropic.com cherry-pick --continue >/dev/null 2>&1 || echo "continue failed $c"
      else echo "CONFLICT $c: $(git diff --name-only --diff-filter=U)"; return 1; fi; }
  done
  git log --oneline -4
  cd $F
}
mk light fid/missile-last-unit-r1 2a2165b1 f45e45c8 2>&1
mk cold fid/cold-enchanted-length-r1 8a7f8520 b3c6341a 2>&1
