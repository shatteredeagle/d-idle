# crfixbundle: game.exe as the oracle for the repair price of an item the gate (FUN_0062e660) lets through on
# its missing charges alone. Not the price's charges term: the base part, FUN_0062efb0 mode 3, 0x62f9b6..0x62fa90
# (the durability scaling, run as is on a fake item unit) then the npc mult block 0x62faac..0x62fc59.
#   cr_repair_game.py gen SEED N OUT     case lines: R id code dur maxdur indes base npc diff
#   cr_repair_game.py game CASES OUT     game side: `id base_part_price`
# Run with the emulator venv and PYTHONPATH=<repo>/tools/emu (see cr_run.sh).
import random
import sys

from unicorn.x86_const import UC_X86_REG_EBP, UC_X86_REG_EBX, UC_X86_REG_EDI, UC_X86_REG_ESP

import emu
import skill_tips
import vpr_oracle as vo


def gen(seed, n, out):
    rnd = random.Random(seed)
    e = skill_tips.boot()
    names = sorted(vo.npc_ids())
    smiths = [x for x in ("Charsi", "Fara", "Hratli", "Halbu", "Larzuk") if x in names] or names
    codes = vo.item_codes(e)
    lines, k = [], 0
    for code, (cid, rec) in sorted(codes.items()):
        if e.read(rec + 0x132, 1)[0]: continue          # stackable: the restock path, not under test
        base_dur = e.read(rec + 0x112, 1)[0]
        for state in range(8):
            maxd = base_dur if state % 4 != 3 else 0
            dur = {0: maxd // 2, 1: maxd, 2: 0, 3: 0}[state % 4]
            indes = 1 if state >= 6 else 0
            base = rnd.choice([1, 77, 1000, 5000, 20000, 65535, 70000, 250000])
            lines.append(f"R r{k} {code} {dur} {maxd} {indes} {base} {rnd.choice(smiths)} {rnd.randrange(3)}")
            k += 1
    rnd.shuffle(lines)
    open(out, "w").write("\n".join(lines[:n]) + "\n")


def game(cases, out):
    e = skill_tips.boot()
    ids = vo.npc_ids()
    codes = vo.item_codes(e)
    stats = {}
    e.hook(0x625480, lambda e_: stats.get(e_.arg(1), 0), 3)
    e.hook(0x625e00, lambda e_: stats.get(73, 0), 1)
    e.hook(0x65c310, lambda e_: 0, 3)
    mu, res = e.mu, []
    for line in open(cases):
        f = line.split()
        if not f: continue
        _, cid, code, dur, maxd, indes, base, npc, _diff = f
        classid, _ = codes[code]
        unit, data = e.alloc(0x100), e.alloc(0x100)
        e.w32(unit, 4); e.w32(unit + 4, classid); e.w32(unit + 0x14, data)
        e.w32(data + 0x18, 0x10)
        stats.clear(); stats.update({72: int(dur), 73: int(maxd), 152: int(indes)})
        frame = e.alloc(0x200) + 0x100
        e.w32(frame - 8, int(base)); e.w32(frame - 4, 0); e.w32(frame + 0x14, 3)
        mu.reg_write(UC_X86_REG_EBP, frame); mu.reg_write(UC_X86_REG_ESP, frame - 0x80)
        mu.reg_write(UC_X86_REG_EBX, unit); mu.reg_write(UC_X86_REG_EDI, 0)
        mu.emu_start(0x62f9b6, 0x62fa90)
        scaled = vo.struct.unpack("<i", vo.struct.pack("<I", e.r32(frame - 8)))[0]
        row = e.call(0x656900, ids[npc])
        try:
            _b, _s, rep = vo.run_block(e, row, (scaled,) * 3, set())
        except Exception as err:      # an npc whose row the mult block cannot read: no price row either side
            print("skip", cid, npc, err, file=sys.stderr)
            continue
        res.append(f"{cid} {scaled} {rep}")
    open(out, "w").write("\n".join(res) + "\n")


if __name__ == "__main__":
    if sys.argv[1] == "gen": gen(int(sys.argv[2]), int(sys.argv[3]), sys.argv[4])
    else: game(sys.argv[2], sys.argv[3])
