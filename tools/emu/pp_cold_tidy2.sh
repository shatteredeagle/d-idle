#!/bin/bash
# prprep (worker host): move the test_umod_stats tidy edit from the claim commit into the oracle commit.
export PATH=$HOME/.local/bin:$PATH
G="git -c user.name=shatteredeagle -c user.email=26069301+shatteredeagle@users.noreply.github.com"
cd $HOME/d2d-lab/wt-pp-cold || exit 1
F=tests/monsters/test_umod_stats.cpp
git diff HEAD~1 HEAD -- $F > /tmp/pp_umod_tidy.patch
git checkout HEAD~1 -- $F
$G commit -q --amend --no-edit
git apply /tmp/pp_umod_tidy.patch
$G add $F; $G commit -q --fixup=HEAD~1
GIT_SEQUENCE_EDITOR=true $G rebase -q -i --autosquash upstream/main 2>&1 | tail -2
FILTER_BRANCH_SQUELCH_WARNING=1 git filter-branch -f --env-filter 'export GIT_COMMITTER_NAME=shatteredeagle GIT_COMMITTER_EMAIL=26069301+shatteredeagle@users.noreply.github.com' upstream/main..HEAD 2>&1 | tail -1
git log --format='%h %an %cn %s' -3 | cut -c1-100
git show --stat --format= HEAD | cat; git show --stat --format= HEAD~1 | cat
git status --short | head -3
