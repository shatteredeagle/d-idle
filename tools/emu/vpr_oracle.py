# Verifier VERIFYPRICES (card verifyprices): game.exe's own code as the oracle for the two claims of
# fix/vendor-prices-followon-v2, written without the author's price_* scripts.
#   vpr_oracle.py gen SEED N OUT          case file (M = npc mult lines, G = repair gate lines, N = npc dump)
#   vpr_oracle.py game CASES OUT          game.exe: M -> FUN_0062efb0's mult block 0062faac..0062fc59 run as is
#                                         (prices in, npc row from FUN_00656900, quest check FUN_0065c310 hooked);
#                                         G -> FUN_0062e660 (repair gate) on a fake item unit, stat getters hooked
#   vpr_oracle.py rows OUT                game npc rows by npc.txt name (FUN_00656900 by MonStats hcIdx)
import random
import struct
import sys

from unicorn.x86_const import UC_X86_REG_EBP, UC_X86_REG_EDI, UC_X86_REG_ESI, UC_X86_REG_ESP, UC_X86_REG_EIP

import emu
import skill_tips


def txt(name):
    raw = emu.mpq_cache().read(f"data\\global\\excel\\{name}.txt").decode("latin-1").split("\r\n")
    head = raw[0].split("\t")
    return [dict(zip(head, line.split("\t"))) for line in raw[1:] if line.strip()]


def npc_ids():
    hc = {r["Id"]: int(r["hcIdx"]) for r in txt("monstats") if r.get("hcIdx", "").strip().lstrip("-").isdigit()}
    return {r["npc"]: hc[r["npc"]] for r in txt("npc") if r["npc"] in hc}


def item_codes(e):
    codes, cid = {}, 0
    while (rec := e.call(0x6335f0, cid)):
        code = e.read(rec + 0x80, 4).rstrip(b" \0").decode("latin-1")
        codes[code] = (cid, rec)
        cid += 1
    return codes


EDGES = [0, 1, 1023, 1024, 0xfffe, 0xffff, 0x10000, 0x10001, 0x103ff, 0x10400, 0x1ffff, 0x20000, 99999, 100000, 1 << 20, (1 << 20) + 1023, (1 << 24) - 1]


def gen(seed, n, out):
    rnd = random.Random(seed)
    e = skill_tips.boot()
    names = sorted(npc_ids())
    codes = item_codes(e)
    lines = ["N"]
    for i in range(n):
        r = rnd.random()
        if i < len(EDGES) * 4:
            p = EDGES[i % len(EDGES)]
        elif r < 0.4:
            p = rnd.randrange(0xf000, 0x12000)
        elif r < 0.8:
            p = int(2 ** rnd.uniform(4, 24))
        else:
            p = rnd.randrange(0, 0x10000)
        lines.append(f"M m{i} {p} {rnd.choice(names)} {rnd.randrange(3)} {rnd.randrange(8)}")
    smiths = [x for x in ("Charsi", "Fara", "Hratli", "Halbu", "Larzuk") if x in names] or names
    k = 0
    for code, (cid, rec) in sorted(codes.items()):
        base_dur = e.read(rec + 0x112, 1)[0]
        for state in range(12):
            maxd = base_dur if state % 4 != 3 else 0
            dur = {0: maxd // 2, 1: maxd, 2: 0, 3: 0}[state % 4]
            chg = state // 4               # 0 none, 1 full, 2 missing
            ident, eth, indes = 1, 0, 0
            if rnd.random() < 0.1: ident = 0
            if rnd.random() < 0.1: eth = 1
            if rnd.random() < 0.1: indes = 1
            qty = rnd.randrange(0, 3) if rnd.random() < 0.5 else 1
            lines.append(f"G g{k} {code} {dur} {maxd} {qty} {ident} {eth} {indes} {chg} {rnd.choice(smiths)} {rnd.randrange(3)}")
            k += 1
    open(out, "w").write("\n".join(lines) + "\n")


def run_block(e, row, prices, qset):
    """FUN_0062efb0's mult block as is: [ebp-0x10] buy, edi sale, [ebp-8] repair in; out [ebp-0x10], [ebp-4], edi."""
    mu = e.mu
    frame = e.alloc(0x200) + 0x100
    quests = e.alloc(0x40)
    buy, sale, rep = prices
    e.w32(frame - 0x10, buy); e.w32(frame - 8, rep); e.w32(frame + 0x10, row); e.w32(frame + 0xc, quests)
    qset_now[0] = qset
    mu.reg_write(UC_X86_REG_EBP, frame); mu.reg_write(UC_X86_REG_ESP, frame - 0x80)
    mu.reg_write(UC_X86_REG_ESI, row); mu.reg_write(UC_X86_REG_EDI, sale & 0xffffffff)
    mu.emu_start(0x62faac, 0x62fc59)
    s = lambda v: struct.unpack("<i", struct.pack("<I", v & 0xffffffff))[0]
    return s(e.r32(frame - 0x10)), s(e.r32(frame - 4)), s(mu.reg_read(UC_X86_REG_EDI))


qset_now = [set()]


def game(cases, out):
    e = skill_tips.boot()
    ids = npc_ids()
    codes = item_codes(e)
    e.hook(0x65c310, lambda e_: 1 if e_.arg(1) in qset_now[0] else 0, 3)
    stats = {}
    e.hook(0x625480, lambda e_: stats.get(e_.arg(1), 0), 3)
    e.hook(0x625e00, lambda e_: stats.get(73, 0), 1)
    charges = []
    def stat_array(e_):
        buf, most = e_.arg(2), e_.arg(3)
        for i, v in enumerate(charges[:most]):
            e_.w32(buf + 8 * i, 204); e_.w32(buf + 8 * i + 4, v)
        return len(charges)
    e.hook(0x6261d0, stat_array, 4)
    res = []
    for line in open(cases):
        f = line.split()
        if not f or f[0] == "N": continue
        if f[0] == "M":
            _, cid, p, npc, diff, qbits = f
            row = e.call(0x656900, ids[npc])
            flags = [e.s32(row + 0x10 + 4 * q) for q in range(3)]
            qs = {flags[q] for q in range(3) if int(qbits) >> q & 1 and flags[q]}
            b, s, r = run_block(e, row, (int(p),) * 3, qs)
            cap = e.s32(row + 0x40 + 4 * int(diff))
            res.append(f"{cid} {b} {s} {r} {cap}")
        else:
            _, cid, code, dur, maxd, qty, ident, eth, indes, chg, npc, diff = f
            classid, _ = codes[code]
            unit, data = e.alloc(0x100), e.alloc(0x100)
            e.w32(unit, 4); e.w32(unit + 4, classid); e.w32(unit + 0x14, data)
            e.w32(data + 0x18, (0x10 if int(ident) else 0) | (0x400000 if int(eth) else 0))
            stats.clear(); stats.update({72: int(dur), 73: int(maxd), 152: int(indes), 70: int(qty)})
            charges[:] = [] if chg == "0" else [(10 << 8 | 10) if chg == "1" else (10 << 8 | 3)]
            res.append(f"{cid} {e.call(0x62e660, unit)}")
    open(out, "w").write("\n".join(res) + "\n")


def rows(out):
    e = skill_tips.boot()
    lines = []
    for name, hc in sorted(npc_ids().items()):
        row = e.call(0x656900, hc)
        if row: lines.append(f"N {name} " + " ".join(str(v) for v in struct.unpack("<19i", e.read(row, 76))[1:]))
    open(out, "w").write("\n".join(lines) + "\n")


if __name__ == "__main__":
    cmd = sys.argv[1]
    if cmd == "gen": gen(int(sys.argv[2]), int(sys.argv[3]), sys.argv[4])
    elif cmd == "game": game(sys.argv[2], sys.argv[3])
    elif cmd == "rows": rows(sys.argv[2])
