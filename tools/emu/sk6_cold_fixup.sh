#!/bin/bash
# decomp6 (worker host): presubmit fixes for fid/cold-enchanted-length, squashed into commit 1; then push
HERE=$(cd "$(dirname "$0")" && pwd)
. $HERE/d6_env.sh
cd $T
git switch -q fid/cold-enchanted-length
sed -i 's/\bdm\b/damage_rows/g' tests/monsters/test_umod_stats.cpp
sed -i 's/FUN_005a17e0 strong, FUN_005a1910 fast,/FUN_005a17e0 strong, the fast init,/' tools/emu/umod_stats.py
git status --short | grep -v '^??'
ID=(-c user.name=shatteredeagle -c user.email=26069301+shatteredeagle@users.noreply.github.com)
git "${ID[@]}" commit -q -a --fixup=HEAD~1
GIT_SEQUENCE_EDITOR=true git "${ID[@]}" rebase -q -i --autosquash upstream/main
git log --format='%h %an %s' -3
python3 tools/pr_guard.py upstream/main 2>&1 | tail -2
python3 tools/presubmit.py --quick --base upstream/main 2>&1 | grep -E "FAIL|WARN|presubmit:"
git push -q origin fid/cold-enchanted-length 2>&1 | tail -1; git ls-remote origin fid/cold-enchanted-length | cut -c1-12
