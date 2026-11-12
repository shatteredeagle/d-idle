#!/bin/bash
# prprep (worker host): amend the claim commit with the narrowed change, clean the messages, run pr_guard.   pp_finish.sh light|cold "new subject or -"
n=$1
cd $HOME/d2d-lab/wt-pp-$n || exit 1
git add -A components docs tests tools
git -c user.name=shatteredeagle -c user.email=26069301+shatteredeagle@users.noreply.github.com commit -q --amend --no-edit
FILTER_BRANCH_SQUELCH_WARNING=1 git filter-branch -f --msg-filter 'grep -v "^(cherry picked from"' --env-filter 'export GIT_COMMITTER_NAME=shatteredeagle GIT_COMMITTER_EMAIL=26069301+shatteredeagle@users.noreply.github.com' upstream/main..HEAD 2>&1 | tail -1
git log -3 --format='%h %an %cn | %s' ; git log -2 --format=%B | head -30
python3 tools/pr_guard.py upstream/main HEAD 2>&1 | tail -6
