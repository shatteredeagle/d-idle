#!/usr/bin/env python3
"""Which stats does a freshly made item carry: does a shield get a base damage (stats 21 / 22) the way a weapon does? (card skshield)

Runs game.exe's own FUN_00557ab0 (the item's base rolls: durability, defence, and for a weapon FUN_005563d0's base
damage) on a fake item unit set up as upstream's drops.py does (FUN_00552df0 / FUN_00552e90 seeds, FUN_0062a6c0 version,
FUN_00628220 ilvl), but with the REAL stat functions (FUN_00627260 sets, FUN_00625480 reads) and the item's own stat
list; tables from game.exe's loader (drlg.boot). Every Weapons.txt and Armor.txt row with a base damage, plus the
shields, is made once at quality 2 (normal). Hooked: the pool allocator, FUN_00625e00 / FUN_00625d00 (max durability
and the modifier list, answered 0: nothing modifies a normal item).

    python skshield_base.py
"""
import sys

from unicorn.x86_const import UC_X86_REG_EDX

import drlg


def rows(e, name):
    lines = e.mpq_file(f"data\\global\\excel\\{name}.txt").decode("latin-1").split("\r\n")
    head = lines[0].split("\t")
    out = []
    for line in lines[1:]:
        if not line: continue
        row = {}
        for key, value in zip(head, line.split("\t")):
            row.setdefault(key, value)            # Armor.txt names mindam / maxdam twice: the first pair is the item's
        out.append(row)
    return out


def main():
    e = drlg.boot()
    codes = []
    while rec := e.call(0x6335f0, len(codes)):
        codes.append(e.read(rec + 0x80, 4).decode("latin-1").rstrip(" \0"))
    e.hook(0x40b430, lambda e: e.alloc(e.mu.reg_read(UC_X86_REG_EDX) + 16), 3)

    def realloc(e):
        new = e.alloc(e.arg(0) + 16)
        old = e.mu.reg_read(UC_X86_REG_EDX)
        if old: e.mu.mem_write(new, e.read(old, min(e.arg(0), 0x400)))
        return new
    e.hook(0x40b4b0, realloc, 4)
    e.hook(0x625e00, lambda e: 0, 1)
    game, gs = e.alloc(0x2000), e.alloc(8)
    e.w32(game + 0x70, 1); e.w32(game + 0x78, 101)

    def make(code, quality=2):
        item, data, args = e.alloc(0x200), e.alloc(0x200), e.alloc(0x100)
        e.w32(item, 4); e.w32(item + 4, codes.index(code)); e.w32(item + 0x14, data)
        e.w32(item + 0x5c, e.call(0x6251f0, 0, 0, 0, 0, 0))
        e.w32(gs, 0x1234567); e.w32(gs + 4, 666)
        e.call(0x552df0, ecx=item, edx=gs)
        e.call(0x552e90, ecx=item, edx=gs)
        e.call(0x62a6c0, item, 101)
        e.call(0x628220, item, 30)
        e.w32(args + 0xc, 30); e.w32(args + 0xfc, item); e.w32(args + 0x30, quality)
        e.call(0x557ab0, args, 1, ecx=game, edx=args + 0xfc)
        return {s: e.call(0x625480, item, s, 0) for s in (21, 22, 23, 24, 159, 160, 31, 72, 73)}

    out, n_w, n_a, w_dmg, a_dmg, w_txt, a_txt = [], 0, 0, 0, 0, 0, 0
    for name, kind in (("Weapons", "weapon"), ("Armor", "armor")):
        for r in rows(e, name):
            code = r.get("code", "")
            if code not in codes: continue
            mn, mx = int(r.get("mindam") or 0), int(r.get("maxdam") or 0)
            shield = r.get("type") in ("shie", "ashd", "head")      # Any Shield: shie, Auric Shields ashd, Voodoo Heads head
            if kind == "armor" and not (mx or shield): continue
            st = make(code)
            any_dmg = any(st[s] > 0 for s in (21, 22, 23, 24, 159, 160))
            if kind == "weapon": n_w += 1; w_dmg += any_dmg; w_txt += bool(mn or mx or int(r.get("2handmaxdam") or 0) or int(r.get("maxmisdam") or 0))
            else: n_a += 1; a_dmg += any_dmg; a_txt += bool(mx)
            out.append(f"{kind:6} {code:4} type {r.get('type', ''):5} txt dmg {mn}-{mx} (2h {r.get('2handmindam', '')}-{r.get('2handmaxdam', '')}): made stats 21/22/23/24 = {st[21]}/{st[22]}/{st[23]}/{st[24]} 159/160 = {st[159]}/{st[160]}  defence {st[31]} dur {st[72]}/{st[73]}")
    print("\n".join(out))
    print(f"weapons made {n_w} ({w_txt} with a txt damage), with a base damage stat (21 22 23 24 159 160) {w_dmg}; "
          f"armor rows with a txt damage or a shield type {n_a} ({a_txt} with a txt damage), with a base damage stat {a_dmg}")


if __name__ == "__main__":
    sys.exit(main())
