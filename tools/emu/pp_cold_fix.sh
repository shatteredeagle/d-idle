#!/bin/bash
# prprep (worker host): pr_guard WARN fix in the oracle commit of the Cold Enchanted branch (FUN_005a1910 is not a function entry).
export PATH=$HOME/.local/bin:$PATH
cd $HOME/d2d-lab/wt-pp-cold || exit 1
sed -i 's/strong, FUN_005a1910$/strong, LAB_005a1910/' tools/emu/umod_stats.py
git add tools/emu/umod_stats.py
git -c user.name=shatteredeagle -c user.email=26069301+shatteredeagle@users.noreply.github.com commit -q --fixup=HEAD~1
GIT_SEQUENCE_EDITOR=true git -c user.name=shatteredeagle -c user.email=26069301+shatteredeagle@users.noreply.github.com rebase -q -i --autosquash upstream/main 2>&1 | tail -2
FILTER_BRANCH_SQUELCH_WARNING=1 git filter-branch -f --env-filter 'export GIT_COMMITTER_NAME=shatteredeagle GIT_COMMITTER_EMAIL=26069301+shatteredeagle@users.noreply.github.com' upstream/main..HEAD 2>&1 | tail -1
git log --format='%h %an %cn %s' -3 | cut -c1-110
python3 tools/pr_guard.py upstream/main HEAD 2>&1 | tail -2
