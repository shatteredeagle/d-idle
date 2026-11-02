#!/usr/bin/env python3
"""verifyaidel: own harness for FUN_005a73e0 (the think schedule) driven by the REAL MonStats rows.
Runs under tools/emu/run-on-worker.sh (emulator venv, upstream emu.py).
  tools/emu/run-on-worker.sh va_aidel_run.py [SEED] [N_RANDOM_FRAMES] > out.txt
Oracle written here (not copied from the author): scheduled frame = frame + A where
 A = row byte at +0x4f (+difficulty in expansion), 0 -> 15, unless a pending type-2 event > frame
 (nothing filed) or state 0x15 is set (frame + 0x2d).
MonStats rows come from the patched table read through mpq-table.sh into $D2_SCRATCH/ms.txt."""
import os, random, subprocess, sys
import emu
seed = int(sys.argv[1]) if len(sys.argv) > 1 else 11
nrand = int(sys.argv[2]) if len(sys.argv) > 2 else 20
scr = os.environ.get("D2_SCRATCH", "/tmp")
here = os.path.dirname(os.path.abspath(__file__))
txt = subprocess.run(["bash", here + "/mpq-table.sh", "monstats"], capture_output=True, text=True).stdout
txt = txt or sys.exit("no table: " + subprocess.run(["bash", "tools/emu/mpq-table.sh", "monstats"], capture_output=True, text=True, cwd=os.path.dirname(os.path.abspath(__file__)) + "/../..").stderr[:300])
L = [l.split("\t") for l in txt.splitlines() if "\t" in l]
h = L[0]; ia = [h.index("aidel"), h.index("aidel(N)"), h.index("aidel(H)")]
rows = [r for r in L[1:] if len(r) == len(h)]
tri = lambda r: tuple(int(r[k]) if r[k].strip() else 0 for k in ia)
e = emu.Emu("/tmp/casttrace/game.exe")
cur = {}; out = {}
e.hook(0x5415a0, lambda e_: cur["pending"], 1)
e.hook(0x639df0, lambda e_: cur["stun"] if e_.arg(1) == 0x15 else 0, 2)
e.hook(0x5417d0, lambda e_: out.__setitem__("event", (e_.arg(0), e_.arg(1))), 4)
game = e.alloc(0x1000); unit = e.alloc(0x200); data = e.alloc(0x100); row = e.alloc(0x200); ctx = e.alloc(0x20)
e.w32(unit, 1); e.w32(unit + 4, 19); e.w32(unit + 0x10, 1); e.w32(unit + 0x14, data); e.w32(unit + 0x2c, 0); e.w32(data, row)
def case(t, expansion, diff, pending, stun, frame):
    cur.update(pending=pending, stun=stun); out.clear()
    e.mu.mem_write(game + 0x6a, bytes([1 if expansion else 0])); e.w32(game + 0x74, 0)
    e.mu.mem_write(game + 0x6d, bytes([diff]))
    e.mu.mem_write(row + 0x4f, bytes(t))
    e.w32(game + 0xa8, frame); e.w32(ctx, 1); e.w32(ctx + 4, unit)
    e.call(0x5a73e0, ecx=game, edx=ctx)
    return out["event"] if "event" in out else None
rng = random.Random(seed)
n = bad = 0
for r in rows:
    t = tri(r)
    for expansion in (0, 1):
        for diff in ((0, 1, 2) if expansion else (0,)):
            a = t[diff] if expansion else t[0]
            a = a if a else 15
            for k in range(nrand):
                frame = rng.randrange(1, 1 << 30)
                for pending, stun in ((0, 0), (frame - 1, 0), (frame, 0), (frame + 1, 0), (frame + 50, 0), (0, 1)):
                    got = case(t, expansion, diff, pending, stun, frame)
                    if pending > frame: want = None
                    elif stun: want = (2, frame + 0x2d)
                    else: want = (2, frame + a)
                    n += 1
                    if got != want:
                        bad += 1
                        if bad <= 10: print("differ", r[0], t, expansion, diff, pending, stun, frame, got, want)
print(f"rows {len(rows)} cases {n} differ {bad} seed {seed}")
