#!/usr/bin/env python3
"""sb_block (card smiteblock): does a block stop Paladin Smite in game.exe 1.14d?

Runs game.exe's own code under unicorn through upstream's harness (tools/emu/emu.py), as
tools/emu/sb_run.sh does:

    tools/emu/run-on-worker.sh sb_run.sh sb_block.py [cases_per_cell] [seed] > /tmp/d2agent-smiteblock/out.txt

The chain Smite runs after it has rolled its damage (FUN_005ce9f0 0x5ceba8 .. 0x5cec2e):
  FUN_0057ec10  to-hit / block / dodge check (bonus 0, range 0)             native (V run)
  `or ax, 1`    0x5cebb0, done here in Python (one instruction)
  FUN_0057dbf0  gate: record flags & 0x8380, bit 0; FUN_0057ca80 queues     native (V run)
  FUN_0057d4f0  finds the queued record and decides what happens to it:     native (V run)
                bit 0 set -> FUN_0057c6c0 (the damage), then FUN_0057cee0 (the reaction)
  the three consumers FUN_0057c6c0 / FUN_0057cee0 / FUN_0057c1e0 are hooked to COUNT their calls (FUN_0057c6c0 = the
  damage application, FUN_0057c1e0 = the resistance step the gate runs): what they do inside is V read only.
Plain attack column: the same chain without the `or ax, 1`.

Emulator R runs the to-hit roll FUN_0057d9b0 natively (calling it natively inside FUN_0057ec10 clobbers EBX in this
harness: the hooked helpers do not preserve it), emulator E replays the roll as a hook.
Hooked, not the claim: stat getters, FUN_00622560 (attack rating), FUN_00622720 (the defender's block chance: the cell value;
its own cap and stat reads are NOT exercised), FUN_00622d00 (not moving), FUN_0057dd60 (the dodge / avoid / evade result:
the cell value 0, 2, 4, 16; the dodge chance itself is NOT exercised), FUN_00554200 / FUN_005a4ee0 / FUN_00622c40 (foe,
monster target, reach: all yes), FUN_00639df0 (no state), the event / animation helpers of FUN_0057d4f0.
Cells: block chance b in (0, 25, 50, 100) x dodge result d in (0, 2, 4, 16), n cases each, own seeds, own attack rating /
defense (hit roll chance 5..95 %). Record flags at +0 = 2 (Smite's "built" record), word +4 = the result.
"""
import random
import sys

import emu

M32 = 0xFFFFFFFF


def lcg(seed):
    p = seed[0] * 0x6AC690C5 + seed[1]
    seed[0], seed[1] = p & M32, p >> 32
    return seed[0]


def main():
    args = sys.argv[1:]
    n = int(args[0]) if args else 2000
    seed0 = int(args[1]) if len(args) > 1 else 7
    knock = int(args[2]) if len(args) > 2 else 0       # extra record bits (8 = knockback) OR-ed in after the check
    rng = random.Random(seed0)
    r = emu.Emu()
    rstats, st = {}, {"d": 0, "b": 0, "hit": 0}
    r.hook(0x625480, lambda e: rstats.get((e.arg(0), e.arg(1)), 0), 3)
    r.hook(0x625500, lambda e: rstats.get((e.arg(0), e.arg(1)), 0), 3)
    r.hook(0x6223f0, lambda e: rstats.get((e.arg(0), "def"), 0), 1)
    r.hook(0x622560, lambda e: st["ar"], 1)
    r.hook(0x57d8a0, lambda e: 0, 2)
    r.hook(0x645830, lambda e: 0, 4)
    r.hook(0x451f80, lambda e: 0, 0)
    r.hook(0x535bc0, lambda e: 0, 0)
    r.hook(0x63d340, lambda e: 1, 2)

    e = emu.Emu()
    stats = {}
    cnt = {"dmg": 0, "react": 0, "react_flags": 0, "res": 0}
    e.hook(0x625480, lambda e: stats.get((e.arg(0), e.arg(1)), 0), 3)
    e.hook(0x57d9b0, lambda e: st["hit"], 2)
    e.hook(0x622720, lambda e: st["b"], 2)
    e.hook(0x622d00, lambda e: 0, 1)
    e.hook(0x57dd60, lambda e: st["d"], 1)
    e.hook(0x554010, lambda e: 0, 4)
    e.hook(0x5347c0, lambda e: 0, 0)
    e.hook(0x554200, lambda e: 1, 1)
    e.hook(0x5a4ee0, lambda e: 1, 0)
    e.hook(0x622c40, lambda e: 1, 3)
    e.hook(0x639df0, lambda e: 0, 2)
    e.hook(0x40b430, lambda e: e.alloc(0x100), 3)       # the pool allocator FUN_0057ca80 calls for the queued record
    e.hook(0x57c1e0, lambda e: cnt.__setitem__("res", cnt["res"] + 1) or 0, 2)
    e.hook(0x57c6c0, lambda e: cnt.__setitem__("dmg", cnt["dmg"] + 1) or 0, 3)
    def react(e):
        cnt["react"] += 1
        cnt["react_flags"] = e.r32(e.arg(1) + 4) & 0xFFFF
        return 0
    e.hook(0x57cee0, react, 2)
    for a, k in ((0x57d3d0, 2), (0x5c0c30, 3), (0x5d10c0, 2), (0x57c9f0, 1), (0x621e40, 3), (0x623c20, 0)):
        e.hook(a, lambda e: 0, k)
    game = e.alloc(0x200)

    def chain(att, dfd, flags):
        rec = e.alloc(0x100)
        for o in range(0, 0x100, 4): e.w32(rec + o, 0)
        e.w32(rec, 2)
        e.w32(rec + 4, flags & 0xFFFF)
        for k in cnt: cnt[k] = 0
        e.call(0x57dbf0, dfd, rec, 0x80, ecx=game, edx=att)
        e.call(0x57d4f0, dfd, 0, ecx=game, edx=att)
        return dict(cnt)

    print(f"sb_block: seed {seed0}, {n} cases a cell, extra record bits {knock:#x}")
    print("b%  dodge | cases hit-roll-ok | blocked(bit 0x10) | SMITE dmg applied | .. of those blocked | resist step ran | "
          "dodge bits set & applied | PLAIN dmg applied | .. blocked | differ from reading")
    keys = ("cases", "ok", "blk", "sm", "smblk", "smres", "smdg", "pl", "plblk", "bad")
    tot = dict.fromkeys(keys, 0)
    for b in (0, 25, 50, 100):
        for d in (0, 2, 4, 16):
            c = dict.fromkeys(keys, 0)
            for _ in range(n):
                att, dfd = e.alloc(0x100), e.alloc(0x100)
                e.w32(att, 0); e.w32(dfd, 1)
                e.w32(att + 0xc, 1000); e.w32(dfd + 0xc, 2000)
                e.w32(att + 0x10, 6); e.w32(dfd + 0x10, 1)         # modes: attacker alive (FUN_0057d4f0 stops at mode 0 = death)
                ra, rd = r.alloc(0x100), r.alloc(0x100)
                r.w32(ra, 0); r.w32(rd, 1)
                sa = [rng.getrandbits(32), rng.getrandbits(32)]
                sd = [rng.getrandbits(32), rng.getrandbits(32)]
                st.update(b=b, d=d, ar=rng.randrange(50, 2500))
                rstats.clear()
                rstats.update({(ra, 0xc): rng.randrange(1, 99), (rd, 0xc): rng.randrange(1, 99),
                               (rd, "def"): rng.randrange(0, 1500)})
                stats.clear(); stats.update({(dfd, 6): 100 << 8})
                r.w32(ra + 0x20, sa[0]); r.w32(ra + 0x24, sa[1])
                hit = r.call(0x57d9b0, 0, 0, ecx=ra, edx=rd) != 0
                st["hit"] = int(hit)
                for a, s in ((att, sa), (dfd, sd)):
                    e.w32(a + 0x20, s[0]); e.w32(a + 0x24, s[1])
                ax = e.call(0x57ec10, dfd, 0, 0, ecx=game, edx=att) & 0xFFFF
                sm = chain(att, dfd, (ax | 1) | knock)           # 0x5cebb0: or ax, 1 ; then the skill's own bits
                att2, dfd2 = e.alloc(0x100), e.alloc(0x100)       # a fresh pair: the first queue entry is consumed
                e.w32(att2, 0); e.w32(dfd2, 1); e.w32(att2 + 0xc, 1001); e.w32(dfd2 + 0xc, 2001)
                e.w32(att2 + 0x10, 6); e.w32(dfd2 + 0x10, 1)
                stats[(dfd2, 6)] = 100 << 8
                pl = chain(att2, dfd2, ax)
                blocked = bool(ax & 0x10)
                dodged = bool(ax & 0x8380)
                roll = lcg(list(sd)) % 100
                if not hit: want_ax = 0
                elif b > 0 and roll < b: want_ax = 0x10
                elif d == 0: want_ax = 5
                else: want_ax = {2: 0x100, 4: 0x80, 16: 0x8000}[d]
                want = (want_ax, int(True), int(bool(want_ax & 1)))     # (ax, smite applies, plain applies): bit 0 alone
                got = (ax, sm["dmg"], pl["dmg"])
                c["cases"] += 1; c["ok"] += hit; c["blk"] += blocked
                c["sm"] += sm["dmg"]; c["smblk"] += sm["dmg"] and blocked
                c["smres"] += sm["res"]; c["smdg"] += sm["dmg"] and dodged
                c["pl"] += pl["dmg"]; c["plblk"] += pl["dmg"] and blocked
                if got != want:
                    c["bad"] += 1
                    if c["bad"] <= 5: print(f"  differ: b {b} d {d} hit {hit} got {got} want {want}")
            print(f"{b:3d}  {d:5d} | {c['cases']} {c['ok']} | {c['blk']} | {c['sm']} | {c['smblk']} | {c['smres']} | "
                  f"{c['smdg']} | {c['pl']} | {c['plblk']} | {c['bad']}")
            for k in tot: tot[k] += c[k]
    print("total", tot)


if __name__ == "__main__":
    main()
