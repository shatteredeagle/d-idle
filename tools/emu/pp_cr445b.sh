#!/bin/bash
# prpost2 (worker host): CodeRabbit on the Cold Enchanted PR, part 2: commit the test fix, then README / compare fixes, re-run, presubmit, tests.
export PATH=$HOME/.local/bin:$PATH D2_MPQ_DIR=/tmp/ug-emu D2_PATCH_INSTALLER=$HOME/Downloads/LODPatch_114d.exe
G="git -c user.name=shatteredeagle -c user.email=26069301+shatteredeagle@users.noreply.github.com"
HERE=$(cd "$(dirname "$0")" && pwd)
T=$HOME/d2d-lab/wt-pp-cold5; B=$HOME/d2d-lab/build-pp-cold5; O=$HOME/d2d-lab/pp-cold5
cd $T
$G add tests/monsters/test_umod_stats.cpp
$G commit -q -m "tests: test_umod_stats catches exceptions and keeps life in 256ths

main uses the function-try form (tests/.clang-tidy), and the life steps use mul_div on 256ths as make_boss does. The compared output is unchanged.

Agent: Claude Code"
python3 - <<'PY'
p="tools/emu/README.md"; s=open(p).read()
a="`uv run python umod_stats.py 100000 1 > game.tsv; ../../build/tests/test_umod_stats --umod-stats game.tsv > d2d.tsv; uv run python umod_stats.py --compare game.tsv d2d.tsv`"
assert a in s
s=s.replace(a,"`set -o pipefail; uv run python umod_stats.py 100000 1 > game.tsv && ../../build/tests/test_umod_stats --umod-stats game.tsv > d2d.tsv && uv run python umod_stats.py --compare game.tsv d2d.tsv`")
open(p,"w").write(s)
p="tools/emu/umod_stats.py"; s=open(p).read()
a="""    bad, total, examples = {}, 0, {}
    for line in open(d2d_path):
        if line.startswith("#"): continue
        d = line.rstrip("\\n").split("\\t")
        gl = game.get(d[0])
        if not gl: continue
        total += 1
"""
assert a in s
s=s.replace(a,"""    rows = [line.rstrip("\\n").split("\\t") for line in open(d2d_path) if not line.startswith("#")]
    if not rows or {d[0] for d in rows} != set(game):
        raise SystemExit(f"{d2d_path} has {len({d[0] for d in rows})} case ids, {game_path} has {len(game)}: the two files must hold the same nonempty set")
    short = [d[0] for d in rows if len(d) != len(FIELDS) + 1]
    if short:
        raise SystemExit(f"{d2d_path}: {len(short)} rows do not have {len(FIELDS)} fields (first: case {short[0]})")
    bad, total, examples = {}, 0, {}
    for d in rows:
        gl = game[d[0]]
        total += 1
""")
open(p,"w").write(s)
PY
git diff --stat | tail -1
cd tools/emu
for seed in 1 9; do
  uv run python umod_stats.py --compare $O/game-$seed.tsv $O/cr-$seed.d2d 2>&1 | grep -v WARNING > $O/cr2-$seed.full; echo "seed $seed rc=$? $(head -1 $O/cr2-$seed.full)"
  diff <(grep -v '^cases' $O/claim-$seed.full) <(grep -v '^cases' $O/cr2-$seed.full) | wc -l
done
head -n 3000 $O/cr-1.d2d > $O/trunc.d2d; uv run python umod_stats.py --compare $O/game-1.tsv $O/trunc.d2d 2>&1 | grep -v WARNING | tail -1 | cut -c1-200; echo "truncated rc=${PIPESTATUS[0]}"
cd $T
$G add tools/emu/README.md tools/emu/umod_stats.py
$G commit -q -m "tools: umod_stats --compare rejects incomplete files; the README command stops on a failed stage

Agent: Claude Code"
python3 tools/pr_guard.py upstream/main HEAD 2>&1 | tail -1
nice -n 19 cmake --build $B -j8 > $B.all.log 2>&1; echo "full build rc=$?"
for t in test_boss_cold_length test_monsters test_game test_umod_stats; do timeout 900 $B/tests/$t >/tmp/pp5b_$t.out 2>&1; echo "$t rc=$?"; done
timeout 1500 python3 tools/presubmit.py --quick --base upstream/main > $O/presubmit2.txt 2>&1; echo "presubmit rc=$?"
grep -vE '^ok' $O/presubmit2.txt | tail -8 | cut -c1-250
git log --format='%h %an %s' -2 | cut -c1-100
