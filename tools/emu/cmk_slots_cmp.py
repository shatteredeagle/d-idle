"""covmonskills2: compare game.exe's monster skill slots (cmk_slots.py, three difficulty files) with d2d's parse (cmk_slots_test --slots).
    python3 -I tools/emu/cmk_slots_cmp.py <slots0.txt> <slots1.txt> <slots2.txt> <d2d_slots.txt> <d2d_bin.txt>
Rows are matched by game.exe's class id (hcIdx) through d2d's own mon_bin map (a MonStats Id can repeat: cr_lancer8 is two rows).
Checks per (row, difficulty, slot): the skill id, the Sk*lvl, whether game.exe gave the unit the skill (found) against d2d's list,
and the level game.exe reads (FUN_006442a0) against d2d's rules::monster_skill_level. Prints counts and every difference."""
import sys

binmap = {}
for line in open(sys.argv[5]):
    t = line.split()
    if t and t[0] == "bin":
        binmap[int(t[1])] = int(t[2])
g = {}
failed = set()
for diff, path in enumerate(sys.argv[1:4]):
    for line in open(path):
        t = line.split()
        if t[:2] == ["spawn", "failed"]:
            failed.add(t[2])
        elif t and t[0] == "G":
            # G Id hc diff d slot k name id sid lvl L found f got v raw r
            g[(int(t[2]), int(t[4]), int(t[6]))] = dict(id=t[1], name=t[7], sid=int(t[9]), lvl=int(t[11]), found=int(t[13]), got=t[15], raw=t[17])
d = {}
for line in open(sys.argv[4]):
    t = line.split()
    if t and t[0] == "slot":
        # slot row Id slot name mode lvl sid L0 L1 L2
        d[(int(t[1]), int(t[3]))] = dict(id=t[2], name=t[4], mode=t[5], lvl=int(t[6]), sid=int(t[7]), L=[int(t[8]), int(t[9]), int(t[10])])
cases = diff_count = 0
kinds = {}
absent = []
seen = set()
for (hc, df, slot), r in sorted(g.items()):
    x = d.get((binmap.get(hc, -1), slot))
    cases += 1
    seen.add((binmap.get(hc, -1), slot))
    if x is None:
        kinds["no d2d slot"] = kinds.get("no d2d slot", 0) + 1
        print("DIFF no d2d slot", hc, r)
        continue
    bad = []
    if x["id"] != r["id"]: bad.append("row id")
    if x["name"] != r["name"]: bad.append("name")
    if x["sid"] != r["sid"]: bad.append("skill id")
    if x["lvl"] != r["lvl"]: bad.append("Sk*lvl")
    if r["found"] != int(r["lvl"] > 0): bad.append("game found != (lvl > 0)")
    if r["found"] and int(r["got"]) != x["L"][df]: bad.append("level")
    if not r["found"] and r["lvl"] <= 0 and df == 0: absent.append((r["id"], slot, r["name"], x["L"]))
    if bad:
        diff_count += 1
        k = ",".join(bad)
        kinds[k] = kinds.get(k, 0) + 1
        print("DIFF", hc, "diff", df, "slot", slot, r["name"], "game", r, "d2d", x)
# the MonStats record as game.exe holds it (R lines of the difficulty 0 file): id word, mode byte, level byte of each of the 8 slots
MODE = {"DT": 0, "NU": 1, "WL": 2, "GH": 3, "A1": 4, "A2": 5, "BL": 6, "SC": 7, "S1": 8, "S2": 9, "S3": 10, "S4": 11, "DD": 12, "SQ": 14, "RN": 15}
rec_cases = rec_bad = 0
rec_kinds = {}
for line in open(sys.argv[1]):
    t = line.split()
    if not (t and t[0] == "R"):
        continue
    hc, slot, idw, modeb, lvlb = int(t[2]), int(t[4]), int(t[6]), int(t[8]), int(t[10])
    x = d.get((binmap.get(hc, -1), slot))
    rec_cases += 1
    bad = []
    if x is None:
        if idw >= 0: bad.append("game slot set, d2d none")
    else:
        mode = 14 if x["mode"].startswith("seq_") else MODE.get(x["mode"], -99)
        if idw != x["sid"]: bad.append("skill id")
        if lvlb != x["lvl"]: bad.append("level byte")
        if modeb != mode: bad.append("mode byte")
    if bad:
        rec_bad += 1
        for b in bad: rec_kinds[b] = rec_kinds.get(b, 0) + 1
        print("RECDIFF", t[1], hc, "slot", slot, "game", idw, modeb, lvlb, "d2d", x)
print("record slots (8 per row)", rec_cases, "differ", rec_bad, rec_kinds)
extra = [k for k in d if k not in seen and d[k]["id"] not in failed]
print("cases", cases, "rows", len({k[0] for k in g}), "differ", diff_count, "kinds", kinds)
print("spawn failed rows", sorted(failed), "d2d slots of those rows", sum(1 for k in d if d[k]["id"] in failed))
print("d2d slots not covered by a game case (rows spawned)", len(extra), extra[:5])
print("slots game.exe skips (Sk*lvl <= 0), d2d lists them with levels L0 L1 L2:", absent)
