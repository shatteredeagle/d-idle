#!/usr/bin/env python3
"""esverify game side: monster blows on an Energy Shield wearer, all of it game.exe 1.14d's own code.

    tools/emu/run-on-worker.sh --ttl 2000 esv_run.sh CASES CALC OUT      (esv_run.sh puts gaps' life_apply on the path)

Per blow (landing order) the REAL FUN_0057c6c0 (flag 1: it runs the resistance pass FUN_0057c1e0 itself) applies one
damage record (+8 physical, +0x10 fire, +0x1c lightning, +0x24 cold, +0x20 magic, 256ths, whole rolls << 8) to the
wearer. The event dispatcher FUN_005c0c30 is a hook; at event 11 (absorbdamage, owner = the wearer) it sends execution
through a trampoline into the REAL FUN_005ca6c0 (wearer, attacker, record, skill, level) and back to the dispatcher's
caller, so the shield works on the record exactly where game.exe's state node would, and the resistance pass then runs
on what it left. Nothing of the shield arithmetic is written here.

Hooked (not the claim): life_apply.Game's list (stat get / set as a dict, unit-kind predicates, side effects); for the
shield: FUN_006256b0 (the wearer's state 30 present while the shield is up), FUN_00646ca0 (calc1 / calc2: the case's
level looked up in CALC, the numbers d2d's own calc gives, so both sides use the same percent and ratio), the state
removal FUN_006277e0 / FUN_00626cd0 / FUN_00639db0 (logged), the overlay calls (skill row +0x10c = -1: never reached).
The skills table at [0x744304] is a fake one with the shield's row (state 30, calc indexes), put in only for the
duration of the FUN_005ca6c0 call and restored by the trampoline.

CASES lines: id cls level mana256 dr_pct dr_flat mdr res_fire res_light res_cold nblows {phys fire light cold magic} x n
CALC lines (from esv_probe): `#calc cls C level L pct P per16 R`.
Out: `id life256 mana256 removed` and a summary to stderr (events seen, calls of the hooks with unexpected arguments).
"""
import struct
import sys

import unicorn.x86_const as ux

import life_apply

M32 = 0xFFFFFFFF
SKILL = 58
STATE = 30
C1, C2 = 0x1111, 0x2222
FIELDS = (0x08, 0x10, 0x1C, 0x24, 0x20)          # physical, fire, lightning, cold, magic
RESIST = {0x08: 0x24, 0x10: 0x27, 0x1C: 0x29, 0x24: 0x2B, 0x20: 0x25}
BIG_LIFE = 0x7FFF00 >> 8


class Shield:
    def __init__(self):
        g = self.g = life_apply.Game()
        e = self.e = g.e
        self.calc = {}
        self.level = 0
        self.present = False
        self.removed = 0
        self.ev = {}
        self.odd = {}
        self.es_calls = 0
        self.snap_bad = 0
        self.expect = None
        self.tables = e.alloc(0x1000)
        self.skills = e.alloc(0x23C * (SKILL + 1))
        e.w32(self.tables + 0xB98, self.skills)
        e.w32(self.tables + 0xBA0, SKILL + 1)
        e.w32(self.tables + 0xC4, 0x200)
        e.w32(self.tables + 0xBC0, 0)
        row = self.skills + SKILL * 0x23C
        e.mu.mem_write(row + 0x80, struct.pack("<h", STATE))
        e.mu.mem_write(row + 0x10C, struct.pack("<h", -1))
        e.w32(row + 0x138, C1)
        e.w32(row + 0x13C, C2)
        self.state_node = e.alloc(0x40)
        self.tramp = e.alloc(0x80)
        e.hook(0x5C0C30, self.dispatch, 3)
        e.hook(0x6256B0, self.get_state, 2)
        e.hook(0x646CA0, self.calc_hook, 4)
        e.hook(0x6277E0, self.remove, 2)
        e.hook(0x626CD0, lambda h: 0, 1)
        e.hook(0x639DB0, self.clear_bit, 3)
        for a, n in ((0x621DC0, 3), (0x63E7A0, 1), (0x621E40, 3)):
            e.hook(a, self.odd_hook(a), n)

    def odd_hook(self, a):
        def f(h):
            self.odd[a] = self.odd.get(a, 0) + 1
            return 0
        return f

    def get_state(self, h):
        unit, state = h.arg(0), h.arg(1)
        if unit == self.g.target and state == STATE:
            return self.state_node if self.present else 0
        self.odd[("state", state)] = self.odd.get(("state", state), 0) + 1
        return 0

    def calc_hook(self, h):
        idx = h.arg(1)
        if h.arg(0) != self.g.target or h.arg(2) != SKILL or h.arg(3) != self.level or idx not in (C1, C2):
            self.odd["calc"] = self.odd.get("calc", 0) + 1
        return self.cur[0] if idx == C1 else self.cur[1]

    def remove(self, h):
        if h.arg(0) == self.g.target and h.arg(1) == self.state_node:
            self.present = False
            self.removed += 1
        else:
            self.odd["remove"] = self.odd.get("remove", 0) + 1
        return 0

    def clear_bit(self, h):
        self.odd["639db0"] = self.odd.get("639db0", 0) + 1
        return 0

    def dispatch(self, h):
        e = self.e
        ev = e.mu.reg_read(ux.UC_X86_REG_EDX)
        unit, other, rec = h.arg(0), h.arg(1), h.arg(2)
        self.ev[ev] = self.ev.get(ev, 0) + 1
        if ev != 11 or unit != self.g.target or self.level <= 0:
            return 0
        # nothing resisted before event 11: the record still holds the raw fields
        if tuple(e.s32(rec + f) for f in FIELDS) != self.expect:
            self.snap_bad += 1
        self.es_calls += 1
        sp = e.mu.reg_read(ux.UC_X86_REG_ESP)
        ret = e.r32(sp)
        old_tables = e.r32(0x744304)
        e.w32(0x744304, self.tables)
        code = b""
        for v in (self.level, SKILL, rec, other, unit):
            code += b"\x68" + struct.pack("<I", v & M32)
        call_at = self.tramp + len(code)
        code += b"\xe8" + struct.pack("<i", 0x5CA6C0 - (call_at + 5))
        code += b"\xc7\x05" + struct.pack("<II", 0x744304, old_tables)   # mov dword [0x744304], old
        code += b"\x31\xc0"                                                 # xor eax, eax (the dispatcher's 0)
        code += b"\x68" + struct.pack("<I", ret) + b"\xc3"                  # back to the dispatcher's caller
        e.mu.mem_write(self.tramp, code)
        e.mu.ctl_remove_cache(self.tramp, self.tramp + 0x80)
        e.w32(sp, self.tramp)
        return 0

    def run(self, cls, level, mana, dr_pct, dr_flat, mdr, res, blows):
        g, e = self.g, self.e
        g.reset(BIG_LIFE, 0, 0)
        g.stats.clear()
        g.stats[(g.target, 6)] = BIG_LIFE << 8
        g.stats[(g.target, 7)] = BIG_LIFE << 8
        g.stats[(g.target, 8)] = mana
        g.stats[(g.target, 0x22)] = dr_flat
        g.stats[(g.target, 0x23)] = mdr
        g.stats[(g.target, 0x24)] = dr_pct & M32
        g.stats[(g.target, 0x27)] = res[0] & M32
        g.stats[(g.target, 0x29)] = res[1] & M32
        g.stats[(g.target, 0x2B)] = res[2] & M32
        g.stats[(g.target, 0x25)] = 0
        self.level = level
        self.cur = self.calc.get((cls, level), (0, 1)) if level > 0 else (0, 1)
        self.present = level > 0
        before = g.life()
        removed0 = self.removed
        for phys, fire, light, cold, magic in blows:
            e.mu.mem_write(g.rec, b"\0" * 0x80)
            e.w32(g.rec, 1)
            e.w32(g.rec + 4, 1)
            vals = (phys << 8, fire << 8, light << 8, cold << 8, magic << 8)
            for f, v in zip(FIELDS, vals):
                e.w32(g.rec + f, v)
            self.expect = vals
            e.call(0x57C6C0, g.target, 1, g.rec, ecx=g.game, edx=g.attacker)
            if not self.present:
                self.level = 0          # the state is gone: later blows of the frame see no shield
        lost = before - g.life()
        m = g.stats.get((g.target, 8), 0)
        return lost, m, self.removed - removed0


def main():
    cases_path, calc_path, out_path = sys.argv[1:4]
    s = Shield()
    for line in open(calc_path):
        f = line.split()
        if line.startswith("#calc"):
            s.calc[(int(f[2]), int(f[4]))] = (int(f[6]), int(f[8]))
    n = 0
    with open(out_path, "w") as out:
        for line in open(cases_path):
            if not line.strip() or line.startswith("#"):
                continue
            v = list(map(int, line.split()))
            cid, cls, level, mana, dr_pct, dr_flat, mdr, rf, rl, rc, nb = v[:11]
            blows = [tuple(v[11 + 5 * k: 16 + 5 * k]) for k in range(nb)]
            if level > 0 and (cls, level) not in s.calc:
                raise SystemExit(f"no calc for class {cls} level {level}")
            lost, mana_left, removed = s.run(cls, level, mana, dr_pct, dr_flat, mdr, (rf, rl, rc), blows)
            out.write(f"{cid}\t{lost}\t{mana_left}\t{removed}\n")
            n += 1
    print(f"cases {n} es_calls {s.es_calls} snapshot_not_raw {s.snap_bad} removed {s.removed} events {dict(sorted(s.ev.items()))} "
          f"unexpected {s.odd}", file=sys.stderr)


if __name__ == "__main__":
    main()
