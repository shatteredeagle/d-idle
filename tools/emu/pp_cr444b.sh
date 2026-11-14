#!/bin/bash
# prpost (worker host): CodeRabbit's third comment on the Lightning PR: the README / docstring command stops when a stage fails or the test skipped.
G="git -c user.name=shatteredeagle -c user.email=26069301+shatteredeagle@users.noreply.github.com"
cd $HOME/d2d-lab/wt-pp-light4 || exit 1
python3 - <<'PY'
p="tools/emu/README.md"; s=open(p).read()
a="`../../build/tests/test_light_pierce --light-hit 1 400 > d2d.txt; uv run python light_pierce.py d2d.txt > g.txt; grep -E '^[HT]' d2d.txt \\| diff g.txt -`"
assert a in s
s=s.replace(a,"`set -o pipefail; ../../build/tests/test_light_pierce --light-hit 1 400 > d2d.txt && grep -q '^T' d2d.txt && uv run python light_pierce.py d2d.txt > g.txt && grep -E '^[HT]' d2d.txt \\| diff g.txt -`")
open(p,"w").write(s)
p="tools/emu/light_pierce.py"; s=open(p).read()
a="""    ../../build/tests/test_light_pierce --light-hit SEED N > d2d.txt
    uv run python light_pierce.py d2d.txt > game.txt
    grep -E '^[HT]' d2d.txt | diff game.txt -
"""
assert a in s
s=s.replace(a,"""    set -o pipefail
    ../../build/tests/test_light_pierce --light-hit SEED N > d2d.txt \\
        && grep -q '^T' d2d.txt \\
        && uv run python light_pierce.py d2d.txt > game.txt \\
        && grep -E '^[HT]' d2d.txt | diff game.txt -

(the grep -q stops the chain when the test skipped and wrote no cast)
""")
open(p,"w").write(s)
PY
git diff --stat | tail -1
export PATH=$HOME/.local/bin:$PATH D2_MPQ_DIR=${D2_EMU_DIR:-$HOME/d2d-lab/emu}/mpq D2_PATCH_INSTALLER=${D2_EMU_DIR:-$HOME/d2d-lab/emu}/patch/LODPatch_114d.exe
cd tools/emu; bash -c "set -o pipefail; ../../build/tests/test_light_pierce --light-hit 1 400 > /tmp/pp_d2d.txt 2>/dev/null && grep -q '^T' /tmp/pp_d2d.txt && uv run python light_pierce.py /tmp/pp_d2d.txt > /tmp/pp_game.txt && grep -E '^[HT]' /tmp/pp_d2d.txt | diff /tmp/pp_game.txt -; echo chain rc=\$?"
cd ../..
$G add tools/emu/README.md tools/emu/light_pierce.py
$G commit -q -m "tools: the Lightning oracle command stops on a skipped test or a failed stage

Agent: Claude Code"
python3 tools/pr_guard.py upstream/main HEAD 2>&1 | tail -1
git push ghfork fid/missile-last-unit-r4 2>&1 | tail -1; git push "${PRIV:?private remote}" fid/missile-last-unit-r4 2>&1 | tail -1; git rev-parse HEAD
