"""covmonskills2: the table scans behind the bot findings B2 and B3 of docs/research/covmonskills2-trace.md (D: patched MonStats, Skills, Levels).
    python3 -I tools/emu/cmk_bot.py <monstats.txt> <skills.txt> <levels.txt>
B2: Act 2 / 3 / boss rows that name a missile (MissA1 / MissA2 or a Skill slot with a srvmissile and Sk*lvl > 0) and have `rangedtype` other than 1.
B3: monster rows that carry a raise, heal, curse, summon or minion skill and whether the bot's killFirstNames fragments (internal/combat/target.go:76:
    shaman, oblivion, unraveler, bloodraven) occur in the MonStats Id (the bot matches Key + display name; the name is not in this scan: I)."""
import collections
import sys


def load(path, key):
    L = open(path).read().split("\n")
    i = [n for n, l in enumerate(L) if l.split("\t")[0] == key][0]
    h = L[i].split("\t")
    return [dict(zip(h, l.split("\t"))) for l in L[i + 1:] if l.strip()]


M = load(sys.argv[1], "Id")
S = {s["skill"]: s for s in load(sys.argv[2], "skill")}
acts = collections.defaultdict(set)
for l in load(sys.argv[3], "Name"):
    a = l.get("Act", "")
    if a in ("0", "1", "2", "3", "4"):
        for p in ("mon", "nmon", "umon"):
            for k in range(1, 11):
                v = l.get("%s%d" % (p, k), "")
                if v:
                    acts[v].add(int(a) + 1)
for b in ("duriel", "radament", "summoner", "mephisto", "izual", "diablo", "baalcrab"):
    acts[b].add(9)
print("B2 rows (Act 2 / 3 / boss) with a missile and rangedtype not 1:")
n = 0
for m in M:
    if not (acts[m["Id"]] & {2, 3, 9}):
        continue
    miss = [m.get("Skill%d" % k) for k in range(1, 9)
            if m.get("Skill%d" % k) in S and int(m.get("Sk%dlvl" % k) or 0) > 0 and any(S[m["Skill%d" % k]].get(c) for c in ("srvmissile", "srvmissilea", "srvmissileb", "srvmissilec"))]
    if (m.get("MissA1") or m.get("MissA2") or miss) and m.get("rangedtype") != "1":
        n += 1
        print("  ", m["Id"], m["AI"], "rangedtype", repr(m.get("rangedtype")), "MissA1", m.get("MissA1"), "MissA2", m.get("MissA2"), "skills", miss[:3])
print("B2 count", n)
KF = ("shaman", "oblivion", "unraveler", "bloodraven")
DO = {97: "raise", 31: "raise", 58: "raise", 96: "heal/bestow", 30: "curse", 112: "curse-cast", 131: "overseer-whip", 135: "minion-spawner", 119: "summon", 133: "impregnate", 128: "cry-help", 91: "nest", 87: "maggot-lay"}
out = {}
for m in M:
    for k in range(1, 9):
        name = m.get("Skill%d" % k, "")
        if name in S and int(m.get("Sk%dlvl" % k) or 0) > 0:
            do = int(S[name].get("srvdofunc") or 0)
            if do in DO:
                out.setdefault((DO[do], name), []).append(m["Id"])
print("B3 skill carriers by role and skill; Ids with no killFirstNames fragment:")
for (role, name), ids in sorted(out.items()):
    miss = [i for i in ids if not any(f in i for f in KF)]
    print("  ", role, name, len(ids), "no fragment in Id:", len(miss), miss[:8])
