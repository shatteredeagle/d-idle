"""covmonai2: assemble docs/research/covmonai2-runs.txt from the four run outputs saved under /tmp/d2agent-covmonai2 (banner lines dropped).
usage: python3 -I tools/emu/mai3_pack.py"""
parts = [
    ("f1d.txt", "## F1 Duriel,Summoner,Mephisto,Diablo,Izual,UberMephisto,UberDiablo,UberIzual,Hydra,TrappedSoul, 3000 cases each, seed 20261105"),
    ("f2e.txt", "## F2 Act 2, 3 and 4 monsters plus Arach and Fetish, 1500 cases each, seed 20261105"),
    ("f3d.txt", "## F3 Act 5, Baal, towers, traps, Act 4 casters, 1500 cases each, seed 20261105"),
    ("f4b.txt", "## F4 Bighead, Vampire (added by address), 1500 cases each, seed 20261105"),
]
out = ["# covmonai2 run outputs: tools/emu/mai3_run.sh on the worker host. `mai2 AI cases differ` = d2d rules::mon_think replay of game.exe's runs; `acts AI act N` = what game.exe's own think did."]
for name, title in parts:
    out.append(title)
    out += [l.rstrip("\n") for l in open("/tmp/d2agent-covmonai2/" + name) if not l.startswith("labrun")]
open("docs/research/covmonai2-runs.txt", "w").write("\n".join(out) + "\n")
print(len(out))
