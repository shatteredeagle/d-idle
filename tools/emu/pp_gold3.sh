#!/bin/bash
# prprep (worker host): committer identity of the rebased gold clamp branch, then push it to the private remote (PRIV names it).
PRIV=${PRIV:?name of the private remote}
cd $HOME/d2d-lab/wt-pp-gold || exit 1
git status --short | head -2
FILTER_BRANCH_SQUELCH_WARNING=1 git filter-branch -f --env-filter 'export GIT_COMMITTER_NAME=shatteredeagle GIT_COMMITTER_EMAIL=26069301+shatteredeagle@users.noreply.github.com' upstream/main..HEAD 2>&1 | tail -1
git log --format='%h %an %cn' -1
git push "$PRIV" HEAD:refs/heads/fid/load-gold-clamp-r1 2>&1 | tail -1
git rev-parse HEAD
