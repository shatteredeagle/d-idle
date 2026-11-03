#!/bin/bash
# prbuildb2 (worker host): the player's Teleport destination, game.exe (sk_tele_spot.py game, FUN_005ca360 -> the warp
# FUN_00554ea0 natively on synthetic rooms) against BASE and HEAD builds of the real Fight::spot (tests/game/test_tele_spot.cpp).
#   tools/emu/run-on-worker.sh --ttl 1500 pb2_tele.sh N SEED BASE_BIN HEAD_BIN [HEAD_BIN...]
# The units' bit (0x1000) is taken out of the cases (d2d keeps units in the Crowd, not the grid). Prints per build
# and per class of case (target room not near / target blocked / free target / Teleport 0): cases and equal.
set -e
here=$(cd "$(dirname "$0")" && pwd)
UP=${UP:-$HOME/d2d-lab/src}
export D2_MPQ_DIR=${D2_MPQ_DIR:-/tmp/ug-emu}
export D2_PATCH_INSTALLER=${D2_PATCH_INSTALLER:-$HOME/Downloads/LODPatch_114d.exe}
export PYTHONPATH=$here:$UP/tools/emu:${PYTHONPATH:-}
PY=${PY:-$HOME/d2d-lab/pr/walk2-venv/bin/python}
S=${D2_SCRATCH:-/tmp}
N=${1:-100000}; SEED=${2:-1}; shift 2
cd "$UP/tools/emu"
nice "$PY" "$here/sk_tele_spot.py" gen "$N" "$SEED" > "$S/pb2t_raw.tsv"
nice "$PY" - "$S" <<'PYEOF'
import sys
S = sys.argv[1]
with open(S + "/pb2t_cases.tsv", "w") as out:
    for line in open(S + "/pb2t_raw.tsv"):
        f = line.split()
        head, rest = f[:6], f[7:]
        words = [(rest[i], rest[i + 1], int(rest[i + 2]) & ~0x1000) for i in range(0, len(rest), 3)]
        words = [w for w in words if w[2]]
        out.write(" ".join(head + [str(len(words))] + [f"{x} {y} {v}" for x, y, v in words]) + "\n")
PYEOF
nice "$PY" "$here/sk_tele_spot.py" game "$S/pb2t_cases.tsv" > "$S/pb2t_game.tsv"
k=0
for bin in "$@"; do
	nice "$bin" --tele-spot "$S/pb2t_cases.tsv" > "$S/pb2t_d2d$k.tsv" 2> "$S/pb2t_d2d$k.err" || true
	k=$((k + 1))
done
exec nice "$PY" - "$S" "$k" <<'PYEOF'
import sys, collections
S, builds = sys.argv[1], int(sys.argv[2])
ROOM, SIDE = 40, 200
cases = {}
for line in open(S + "/pb2t_cases.tsv"):
    f = list(map(int, line.split()))
    cid, tele, cx, cy, tx, ty, n = f[:7]
    words = {(f[7 + 3 * k], f[8 + 3 * k]): f[9 + 3 * k] for k in range(n)}
    cases[cid] = (tele, cx, cy, tx, ty, words)
game = {l.split()[0]: l.split()[1:] for l in open(S + "/pb2t_game.tsv")}
def klass(c):
    tele, cx, cy, tx, ty, words = c
    if tele == 0: return "Teleport 0"
    if not (0 <= tx < SIDE and 0 <= ty < SIDE) or max(abs(tx // ROOM - cx // ROOM), abs(ty // ROOM - cy // ROOM)) > 1: return "target room not near"
    blocked = any(not (0 <= tx + dx < SIDE and 0 <= ty + dy < SIDE) or words.get((tx + dx, ty + dy), 0) & 0x0c09 for dx, dy in ((0, 0), (-1, 0), (1, 0), (0, -1), (0, 1)))
    return "target blocked" if blocked else "free target"
tot = collections.Counter(klass(c) for c in cases.values())
print("cases", len(cases), dict(tot))
print("game lands", sum(1 for v in game.values() if v[0] == "1"))
for b in range(builds):
    d2d = {l.split()[0]: l.split()[1:] for l in open(f"{S}/pb2t_d2d{b}.tsv")}
    same = collections.Counter(); kinds = collections.Counter()
    for cid, c in cases.items():
        a, d = game[str(cid)], d2d.get(str(cid))
        k = klass(c)
        if a == d: same[k] += 1
        else: kinds[(k, "game lands, d2d refuses" if a[0] == "1" and d[0] == "0" else "game refuses, d2d lands" if a[0] == "0" else "both land, other spot")] += 1
    print(f"build {b}: equal {sum(same.values())} of {len(cases)}; equal by class {dict(same)}")
    print("  different", {f"{k} / {t}": v for (k, t), v in sorted(kinds.items())})
PYEOF
