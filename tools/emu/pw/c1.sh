set -e
S=~/labrun/herdr/code/d2lod-wt/prwarp/tools/emu/pw; W=~/d2d-lab/wt-pw
cd $W
cp $S/test_warp_arrival.v1.cpp tests/game/test_warp_arrival.cpp
python3 - <<'EOF'
p = "tools/emu/README.md"
s = open(p, encoding="utf-8").read()
anchor = "| Skills.cpp skill-list and skill-record accessors"
row = ("| Level warp arrival spot (FUN_005550b0: FUN_0064e7b0 with size 2, mask 0x1c09, flag 1, then FUN_00554ea0; no spot in rings 1 to 49 refuses the warp) | "
       "`set -o pipefail; uv run python sk_tele_spot.py gen --warp 20000 1 > cases.tsv && uv run python sk_tele_spot.py warp cases.tsv > g.txt && ../../build/tests/test_warp_arrival --warp-arrival cases.tsv \\| grep -E '^[0-9]+ ' > d.txt && uv run python sk_tele_spot.py diff g.txt d.txt cases.tsv` | `20000 cases`, then one line: `20000  same` |\n")
assert s.count(anchor) == 1
s = s.replace(anchor, row + anchor)
open(p, "w", encoding="utf-8").write(s)
EOF
git status --short
