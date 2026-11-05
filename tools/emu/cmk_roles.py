"""covmonskills2: which monster skills the Act 2 and Act 3 spawn lists (and the bosses) carry, grouped by role, from the patched tables (D).
    python3 -I tools/emu/cmk_roles.py <monstats.txt> <skills.txt> <levels.txt>      (files written by tools/emu/mpq-table.sh)
Roles by the skill row: raise (srvdofunc 97, 31, 58 or a summon of a monster), summon (summon / pettype / sumskill set, or do 87, 91, 133, 135),
teleport (do 27, 98, 129), aura or curse (aurastate set or do 30, 65, 66, 112, 131), breath and bolts (a srvmissile set), melee special (the rest).
Acts by levels.txt (Act 0..4 = Act 1..5) over mon1..10 / nmon1..10 / umon1..10; bosses by name."""
import collections
import sys


def load(path, key):
    L = open(path).read().split("\n")
    i = [k for k, l in enumerate(L) if l.split("\t")[0] == key][0]
    h = L[i].split("\t")
    return [dict(zip(h, l.split("\t"))) for l in L[i + 1:] if l.strip()]


mons = {m["Id"]: m for m in load(sys.argv[1], "Id")}
skills = {s["skill"]: s for s in load(sys.argv[2], "skill")}
levels = load(sys.argv[3], "Name")
acts = collections.defaultdict(set)
for l in levels:
    if l.get("Act", "") in ("0", "1", "2", "3", "4"):
        for p in ("mon", "nmon", "umon"):
            for k in range(1, 11):
                v = l.get("%s%d" % (p, k), "")
                if v in mons:
                    acts[v].add(int(l["Act"]) + 1)
BOSSES = ["duriel", "radament", "summoner", "mephisto", "izual", "diablo", "baalcrab", "andariel", "bloodraven", "griswold", "nihlathak", "bishibosh"]
for b in BOSSES:
    if b in mons and not acts[b]:
        acts[b].add(0)


def role(s):
    do = int(s.get("srvdofunc") or 0)
    r = []
    if do in (97, 31, 58): r.append("raise")
    if s.get("summon") or s.get("pettype") or s.get("sumskill1") or do in (87, 91, 133, 135, 119, 56, 57): r.append("summon")
    if do in (27, 98, 129): r.append("teleport")
    if s.get("aurastate") or do in (30, 65, 66, 112, 131): r.append("aura/curse")
    if s.get("srvmissile") or s.get("srvmissilea") or s.get("srvmissileb") or s.get("srvmissilec"): r.append("missile")
    return ",".join(r) or "other"


rows = collections.defaultdict(list)   # skill -> monsters
for mid, m in mons.items():
    if not (acts[mid] & {0, 2, 3}):
        continue
    for k in range(1, 9):
        name = m.get("Skill%d" % k, "")
        if name and name in skills:
            rows[name].append((mid, k, m.get("Sk%dlvl" % k, ""), m.get("Sk%dmode" % k, ""), m["AI"], sorted(acts[mid])))
print("monster skills used by Act 2 / Act 3 spawn lists and the listed bosses:", len(rows), "skills,", sum(len(v) for v in rows.values()), "slots,",
      len({r[0] for v in rows.values() for r in v}), "monster rows")
for name in sorted(rows, key=lambda n: int(skills[n]["Id"])):
    s = skills[name]
    ms = rows[name]
    print("%s\tid %s\tdo %s\t%s\t%d slots\t%s" % (name, s["Id"], s.get("srvdofunc", ""), role(s), len(ms),
                                              "; ".join("%s(%s) s%d L%s %s act%s" % (r[0], r[4], r[1], r[2], r[3], "".join(map(str, r[5]))) for r in ms[:6]) + (" ..." if len(ms) > 6 else "")))
