# SPDX-License-Identifier: GPL-3.0-or-later
"""covitemtypes: game.exe's item-type classification as the oracle. Everything runs natively, no hooks.

    tools/emu/run-on-worker.sh cit_oracle.py rows  > rows.tsv     raw ItemTypes records (0xe4 bytes each)
    tools/emu/run-on-worker.sh cit_oracle.py isa   > isa.tsv      `a b r`: FUN_00504a20(a, b) for every row pair
    tools/emu/run-on-worker.sh cit_oracle.py item  > item.tsv     `c code type type2 belt throw t0 t1 ..`: per Items class
    tools/emu/run-on-worker.sh cit_oracle.py d2d   rows.tsv       diff helper not used here

FUN_00504a20(a, b): is ItemTypes row a (or an Equiv ancestor) row b. FUN_00629bb0(unit, t): the precomputed bit matrix
(sgptDataTables+0xc04) of the unit's item type, falling back to the Items record's type2 (+0x120).
FUN_0062bad0 / 0062ba80: the item's OWN ItemTypes row Beltable (+0x19) / Throwable (+0x10).
Units are fakes {4, class}: all these read.
"""
import sys

import skill_tips

N_TYPES = 0x87d830
TYPES = 0x87d82c
REC = 0xE4


def main():
    mode = sys.argv[1] if len(sys.argv) > 1 else ""
    e = skill_tips.boot()
    dt = e.r32(0x744304)   # sgptDataTables
    n = e.r32(dt + 0xbfc)
    base = e.r32(dt + 0xbf8)
    if mode == "isa" and not e.r32(TYPES):
        # the emulator does not run the 0x504920 loader of the second ItemTypes copy that FUN_00504a20 walks;
        # point it at sgptDataTables' copy (same 0xe4 records: Equiv1 +4, Equiv2 +6, checked in `rows`).
        e.w32(TYPES, base)
    if mode == "rows":
        print("n", n, "base", hex(base), "87d82c", hex(e.r32(TYPES)), "87d830", e.r32(N_TYPES), "width", e.r32(dt + 0xc00))
        for i in range(n):
            print(i, e.read(base + i * REC, REC).hex())
    elif mode == "isa":
        for a in range(n):
            for b in range(n):
                print(a, b, e.call(0x504a20, a, b) & 0xFFFFFFFF)
    elif mode == "item":
        nclass = e.r32(0x96ca58)
        for c in range(nclass):
            rec = e.call(0x6335f0, c)
            if not rec:
                continue
            code = e.read(rec + 0x80, 4).decode("latin1").strip()
            u = e.alloc(8)
            e.w32(u, 4)
            e.w32(u + 4, c)
            t1 = e.r32(rec + 0x11e) & 0xFFFF
            t2 = e.r32(rec + 0x120) & 0xFFFF
            bits = "".join("1" if e.call(0x629bb0, u, t) & 0xFFFFFFFF else "0" for t in range(n))
            print(c, repr(code), t1, t2, e.call(0x62bad0, u) & 0xFF, e.call(0x62ba80, u) & 0xFF, bits)
    elif mode == "hands":
        # FUN_0063dbc0 (esi = item A, edi = item B, ebx = the unit; a plain ret): may A and B be held in the two hands together.
        # Items: every Items record whose type's BodyLoc1 is 4 (right arm); classes 1 (sorceress: any non-barbarian, non-assassin),
        # 4 (barbarian), 6 (assassin). Output `class codeA bits(B in code order)`.
        out = e.alloc(8)
        items = {}
        for c in range(e.r32(0x96ca58)):
            rec = e.call(0x6335f0, c)
            if not rec:
                continue
            code = e.read(rec + 0x80, 4).decode("latin1").strip()
            u = e.alloc(8)
            e.w32(u, 4)
            e.w32(u + 4, c)
            e.call(0x62ea80, u, out, out + 4)
            if e.read(out, 1)[0] == 4:
                items[code] = u
        order = sorted(items)
        for cls in (1, 4, 6):
            pl = e.alloc(8)
            e.w32(pl, 0)
            e.w32(pl + 4, cls)
            for a in order:
                bits = "".join("1" if e.call(0x63dbc0, regs={"esi": items[a], "edi": items[b], "ebx": pl}) & 0xFF else "0" for b in order)
                print(cls, a, bits)
    elif mode == "sock":
        # FUN_0062bc20(item): min(Items gemsockets +0x138, the type's MaxSock1/25/40 by item level, +0x2c of the item data, floor 1);
        # `code ilvl sockets` for ilvl 0..110
        for c in range(e.r32(0x96ca58)):
            rec = e.call(0x6335f0, c)
            if not rec:
                continue
            code = e.read(rec + 0x80, 4).decode("latin1").strip()
            data = e.alloc(0x80)
            u = e.alloc(0x20)
            e.w32(u, 4)
            e.w32(u + 4, c)
            e.w32(u + 0x14, data)
            vals = []
            for il in range(0, 111):
                e.w32(data + 0x2c, il)
                vals.append(str(e.call(0x62bc20, u) & 0xFF))
            print(code, ",".join(vals))
    elif mode == "slot":
        # FUN_0062ed50(item, slot): may the item sit in body slot 0..12 (type BodyLoc1/2, or the weapon-switch slots 11 and 12 for
        # an arm item); requirements are NOT in this function (FUN_0062eaf0)
        for c in range(e.r32(0x96ca58)):
            rec = e.call(0x6335f0, c)
            if not rec:
                continue
            code = e.read(rec + 0x80, 4).decode("latin1").strip()
            u = e.alloc(8)
            e.w32(u, 4)
            e.w32(u + 4, c)
            print(code, "".join("1" if e.call(0x62ed50, u, s) & 0xFF else "0" for s in range(13)))
    elif mode == "item2":
        # per item: FUN_0062ea80 (BodyLoc1, BodyLoc2), 0062e9e0 Magic, 0062e990 Rare, 0062ea30 Normal, 0062c060 class-specific
        out = e.alloc(8)
        for c in range(e.r32(0x96ca58)):
            rec = e.call(0x6335f0, c)
            if not rec:
                continue
            code = e.read(rec + 0x80, 4).decode("latin1").strip()
            u = e.alloc(8)
            e.w32(u, 4)
            e.w32(u + 4, c)
            e.w32(out, 0)
            e.call(0x62ea80, u, out, out + 4)
            b1, b2 = e.read(out, 1)[0], e.read(out + 4, 1)[0]
            vals = [e.call(a, u) & 0xFF for a in (0x62e9e0, 0x62e990, 0x62ea30)] + [e.call(0x62c060, u) & 0xFF]
            print(c, repr(code), b1, b2, *vals)
    else:
        raise SystemExit(__doc__)


if __name__ == "__main__":
    main()
