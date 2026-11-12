#!/bin/bash
# prpost2 (worker host): include order fix in test_umod_stats (sorted check), then guard, presubmit.
export PATH=$HOME/.local/bin:$PATH D2_MPQ_DIR=/tmp/ug-emu D2_PATCH_INSTALLER=$HOME/Downloads/LODPatch_114d.exe
G="git -c user.name=shatteredeagle -c user.email=26069301+shatteredeagle@users.noreply.github.com"
T=$HOME/d2d-lab/wt-pp-cold5; B=$HOME/d2d-lab/build-pp-cold5; O=$HOME/d2d-lab/pp-cold5
cd $T
python3 - <<'PY'
p="tests/monsters/test_umod_stats.cpp"; s=open(p).read()
s=s.replace("#include <cstdint>\n#include <cstdio>\n#include <exception>\n#include <cstring>\n","#include <cstdint>\n#include <cstdio>\n#include <cstring>\n#include <exception>\n")
open(p,"w").write(s)
PY
sed -n 18,30p tests/monsters/test_umod_stats.cpp
nice -n 19 cmake --build $B -j8 --target test_umod_stats > /dev/null 2>&1; echo "build rc=$?"
cd tools/emu; $B/tests/test_umod_stats --umod-stats $O/game-1.tsv > $O/cr3-1.d2d && uv run python umod_stats.py --compare $O/game-1.tsv $O/cr3-1.d2d 2>&1 | grep -v WARNING | grep -v '^cases' | diff - <(grep -v '^cases' $O/claim-1.full) | wc -l
cd $T
$G add tests/monsters/test_umod_stats.cpp
$G commit -q -m "tests: test_umod_stats include order

Agent: Claude Code"
python3 tools/pr_guard.py upstream/main HEAD 2>&1 | tail -1
timeout 1500 python3 tools/presubmit.py --quick --base upstream/main > $O/presubmit3.txt 2>&1; echo "presubmit rc=$?"
grep -vE '^ok' $O/presubmit3.txt | tail -6 | cut -c1-250
git log --format='%h %an %s' -3 | cut -c1-90
git rev-parse HEAD
