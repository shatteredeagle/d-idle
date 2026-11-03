#!/bin/bash
# prbuildb2 (worker host): player mana check and payment, game.exe (sk_mana.py game, FUN_0056c160 /
# FUN_0056bfe0 natively) against a BASE and a HEAD build of the real d2d Fight casts (tests/game/test_sk_mana.cpp).
#   tools/emu/run-on-worker.sh --ttl 1500 pb2_mana.sh N SEED BASE_BIN HEAD_BIN
# N random cases on top of the 90,840 grid cases of `sk_mana.py gen`; the bins are test_sk_mana executables
# under ~/d2d-lab. Prints, per build, the cases compared (the cast path exists), the equal ones and the
# differences by skill id.
set -e
here=$(cd "$(dirname "$0")" && pwd)
UP=${UP:-$HOME/d2d-lab/src}
export D2_MPQ_DIR=${D2_MPQ_DIR:-/tmp/ug-emu}
export D2_PATCH_INSTALLER=${D2_PATCH_INSTALLER:-$HOME/Downloads/LODPatch_114d.exe}
export PYTHONPATH=$here:$UP/tools/emu:${PYTHONPATH:-}
PY=${PY:-$HOME/d2d-lab/pr/walk2-venv/bin/python}
S=${D2_SCRATCH:-/tmp}
N=${1:-40000}; SEED=${2:-1}; BASE=$3; HEAD=$4
cd "$UP/tools/emu"
nice "$PY" "$here/sk_mana.py" gen "$N" "$SEED" > "$S/pb2m_cases.tsv"
nice "$PY" "$here/sk_mana.py" game "$S/pb2m_cases.tsv" > "$S/pb2m_game.tsv"
nice "$BASE" --sk-mana "$S/pb2m_cases.tsv" > "$S/pb2m_base.tsv" || true
nice "$HEAD" --sk-mana "$S/pb2m_cases.tsv" > "$S/pb2m_head.tsv" || true
exec nice "$PY" - "$S" "$here" <<'PYEOF'
import sys, collections
S = sys.argv[1]
cases = [l.split() for l in open(S + "/pb2m_cases.tsv")]
game = {l.split()[0]: l.split()[1:] for l in open(S + "/pb2m_game.tsv")}
print("cases", len(cases), "game answers", len(game))
for name in ("base", "head"):
    d2d = {l.split()[0]: l.split()[1:] for l in open(f"{S}/pb2m_{name}.tsv")}
    cmp = same = 0
    diff = collections.Counter()
    kinds = collections.Counter()
    for cid, skill, lvl, mana in cases:
        a, b = game[cid], d2d.get(cid)
        if b is None or b[0] == "-1": continue
        cmp += 1
        if a == b: same += 1
        else:
            diff[skill] += 1
            kinds["game casts, d2d refuses" if a[0] == "1" and b[0] == "0" else "other" if a[0] != b[0] else "paid differs"] += 1
    skipped = sum(1 for v in d2d.values() if v[0] == "-1")
    print(f"{name}: compared {cmp} of {len(cases)} (no cast path: {skipped}), equal {same}, different {cmp - same}")
    print("  by kind", dict(kinds))
    print("  by skill", dict(sorted(diff.items(), key=lambda kv: -kv[1])[:12]))
PYEOF
