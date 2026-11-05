"""covmonai: the think-timing columns of MonStats (patched stack) for one row per AI the pass ran: aidel, aidist, aip1..aip8, each as
Normal/Nightmare/Hell. Reads the table mpq-table.sh monstats wrote ($D2_SCRATCH/monstats.txt, first line `Id hcIdx ...`).
usage: cvm_aitab.py <table> [Id ...]   (default: the Act 1 and 2 rows the trace names)"""
import sys

ROWS = ["zombie1", "fallen1", "fallenshaman1", "quillrat1", "skeleton1", "sk_archer1", "skmage_fire1", "wraith1", "corruptrogue1", "cr_archer1",
        "cr_lancer1", "brute1", "foulcrow1", "bighead1", "vulture1", "mummy1", "swarm1", "scarab1", "sandraider1", "sandleaper1", "pantherwoman1",
        "slinger1", "baboon1", "clawviper2", "unraveler1", "sandmaggot1", "blunderbore1", "arach1", "fetish1", "vampire1", "goatman1"]
lines = open(sys.argv[1]).read().split("\n")
head = lines[0].split("\t")
rows = {r.split("\t")[0]: r.split("\t") for r in lines[1:] if r}
col = {name: i for i, name in enumerate(head)}


def triple(row, base):
    return "/".join(row[col[c]] or "-" for c in (base, base + "(N)", base + "(H)"))


print("Id\tAI\tVelocity\tRun\taidel N/NM/H\taidist N/NM/H\t" + "\t".join("aip%d" % k for k in range(1, 9)))
for rid in sys.argv[2:] or ROWS:
    r = rows.get(rid)
    if not r:
        print(rid, "(no row)")
        continue
    print("\t".join([rid, r[col["AI"]], r[col["Velocity"]], r[col["Run"]], triple(r, "aidel"), triple(r, "aidist")]
                    + [triple(r, "aip%d" % k) for k in range(1, 9)]))
