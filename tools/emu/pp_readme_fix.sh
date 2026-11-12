#!/bin/bash
# prprep (worker host): the README conflict resolution had duplicated upstream rows; restore upstream's README and add only this branch's row (in the oracle commit).
#   pp_readme_fix.sh light|cold KEYWORD
export PATH=$HOME/.local/bin:$PATH
G="git -c user.name=shatteredeagle -c user.email=26069301+shatteredeagle@users.noreply.github.com"
cd $HOME/d2d-lab/wt-pp-$1 || exit 1
R=tools/emu/README.md
grep -F "$2" $R | head -1 > /tmp/pp_row_$1.txt; wc -l < /tmp/pp_row_$1.txt
git checkout upstream/main -- $R
python3 - "$R" /tmp/pp_row_$1.txt <<'PY'
import sys
p, rowf = sys.argv[1], sys.argv[2]
row = open(rowf).read().rstrip("\n")
L = open(p).read().split("\n")
at = next(i for i, l in enumerate(L) if l.startswith("| Which self states a new one switches off"))
L.insert(at + 1, row)
open(p, "w").write("\n".join(L))
PY
$G add $R; $G commit -q --fixup=HEAD~1
GIT_SEQUENCE_EDITOR=true $G rebase -q -i --autosquash upstream/main 2>&1 | tail -2
FILTER_BRANCH_SQUELCH_WARNING=1 git filter-branch -f --env-filter 'export GIT_COMMITTER_NAME=shatteredeagle GIT_COMMITTER_EMAIL=26069301+shatteredeagle@users.noreply.github.com' upstream/main..HEAD 2>&1 | tail -1
git log --format='%h %an %cn %s' -2 | cut -c1-100
git diff --stat upstream/main HEAD -- $R | cat
python3 tools/pr_guard.py upstream/main HEAD 2>&1 | tail -2
git status --short | head -3
