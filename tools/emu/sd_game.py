# SPDX-License-Identifier: GPL-3.0-or-later
"""SEEDDIST game side: game.exe 1.14d's own code under the emulator, outcome samples per Act 1 class.

    tools/emu/run-on-worker.sh --ttl 3000 sd_game.py MODE [ARGS] > $D2_SCRATCH/x.txt

MODE (files go to $D2_SCRATCH/sd/):
  info                      per class: MonStats row, level, TC names, FUN_006538a0's row (life min / max, defence, to-hit, A1 min / max)
  life                      20 games x 1000 units: unit seeds as FUN_00555230 makes them (FUN_00552df0 on a game seed {map seed, 666}, 4
                            creation steps, then one step per unit, gap model per game), the look roll (rand(3)) then the life roll
                            (min + rand(max - min + 1)) through FUN_0045c3e0 on unit +0x20: sd/life_<id>.txt
  swing CFG                 CFG = index into SWINGS: units x 100 swings at a defender: to-hit FUN_0057d9b0 (real), on a hit the
                            physical roll FUN_0057b420 (real, 256ths): sd/swing_<id>_<cfg>.txt "hit dmg256"
  drop CFG                  CFG = index into DROPS (mf, extra draws): first drop FUN_0055a6d0 (real) off the unit seed after the look,
                            the life and the extra draws: sd/drop_<id>_<cfg>.txt, one line per monster
  ctl                       seed-for-seed control files for d2d: sd/ctl_<id>.seeds (low high level), game answers sd/ctl_<id>_<mf>.game
                            and sd/ctlife_<id>.game
Hooks are those of monster_hit.py (stat getters, defence) and drops.py (class roll). Nothing writes game data.
"""
import os
import random
import sys

import drlg
import emu
from monstats import table

SCRATCH = os.path.join(os.environ.get("D2_SCRATCH", "/tmp/d2agent-seeddist"), "sd")
os.makedirs(SCRATCH, exist_ok=True)
MASK = 0xFFFFFFFF
STEP, ROLL = 0x45c370, 0x45c3e0
UNIT_MAKE = 0x552df0

# (MonStats Id, label)
CLASSES = [("fallen1", "Fallen"), ("zombie1", "Zombie"), ("skeleton1", "Skeleton"), ("quillrat1", "Quill Rat"),
           ("corruptrogue1", "Rogue (Dark Hunter)"), ("brute1", "Gargantuan Beast"), ("bloodraven", "Blood Raven"), ("andariel", "Andariel"),
           ("andariel+q", "Andariel quest class (TC4)")]    # +q: drops and ctl only, TreasureClass4 (fresh character, quest not done)
GAMES, PER = 20, 1000
# (defence, defender level, units x 100 swings)
SWINGS = [(0, 1, 220), (20, 2, 200), (120, 12, 200), (400, 30, 200)]
# (magic find, extra draws on the unit seed before the drop: 0 or 1000 + n for 0..n)
DROPS = [(0, 0), (300, 0), (0, 1040)]
CTL_N = 20000


def num(t):
    try: return int(t)
    except ValueError: return 0


class Mirror:
    """Python mirror of the seed (checks the emulated functions once; d2d's Rng is the same code)."""
    def __init__(self, low, high=666): self.low, self.high = low, high
    def step(self):
        p = self.low * 0x6AC690C5 + self.high
        self.low, self.high = p & MASK, p >> 32
        return self.low
    def roll(self, n):
        if n < 1: return 0
        v = self.step()
        return v & (n - 1) if n & (n - 1) == 0 else v % n


class Game:
    def __init__(self, e=None):
        self.e = e or drlg.boot()
        e = self.e
        self.monstats = [r for r in table("monstats") if r.get("id") and r["id"] != "Expansion"]
        self.monlvl = table("monlvl")
        self.out = e.alloc(0x40)
        self.gs, self.scratch, self.unit = e.alloc(8), e.alloc(0x100), e.alloc(0x100)
        e.w32(self.scratch, 1); e.w32(self.unit, 1)
        self.rows = {}
        for cid, label in CLASSES:
            row = next(i for i, r in enumerate(self.monstats) if r["id"].lower() == cid.split("+")[0])
            r = self.monstats[row]
            level = min(num(r["level"]), len(self.monlvl) - 1)
            e.mu.mem_write(self.out, bytes(0x40))
            assert e.call(0x6538a0, row, 1, 0, level, 0x0f, self.out)
            got = [e.s32(self.out + 4 * k) for k in range(11)]
            self.rows[cid] = dict(row=row, level=level, tc=r["treasureclass4"] if "+q" in cid else r["treasureclass1"], tc4=r["treasureclass4"], tcq=r["tcquestid"],
                                  lo=got[0], hi=got[1], ac=got[2], th=got[3], a1lo=got[5], a1hi=got[6], label=label)

    def seed_at(self, unit): return self.e.r32(unit + 0x20), self.e.r32(unit + 0x24)

    def roll(self, unit, n): return self.e.call(ROLL, ecx=unit + 0x20, edx=n)

    def new_game(self, g, salt=0):
        """A game seed {map seed, 666} after its four creation steps (the salt, the class row, keeps classes on their own seeds)."""
        e = self.e
        ms = (0x7a3c9e11 * (g + 1) + 777 + salt * 0x1000193) & MASK
        e.w32(self.gs, ms); e.w32(self.gs + 4, 666)
        mirror = Mirror(ms)
        for _ in range(4):
            e.call(STEP, ecx=self.gs)
            mirror.step()
        assert (e.r32(self.gs), e.r32(self.gs + 4)) == (mirror.low, mirror.high), "STEP differs from the mirror"
        return random.Random(g * 7919 + 13)

    def make_unit(self, g, gaps):
        """FUN_00555230's seed work for the next monster: gap-model extra steps, then FUN_00552df0."""
        e = self.e
        if g % 2 == 1:
            for _ in range(gaps.randint(1, 5)): e.call(UNIT_MAKE, ecx=self.scratch, edx=self.gs)
        before = (e.r32(self.gs), e.r32(self.gs + 4))
        e.call(UNIT_MAKE, ecx=self.unit, edx=self.gs)
        m = Mirror(*before); low = m.step()
        assert self.seed_at(self.unit) == (low, 666), ("unit seed", self.seed_at(self.unit), low)
        assert (e.r32(self.gs), e.r32(self.gs + 4)) == (m.low, m.high), "game seed after FUN_00552df0"
        return low

    def look_and_life(self, cid):
        """The monster maker's first two draws on the unit seed: the look (rand(3), one of up to 3 region sets), then the life."""
        c = self.rows[cid]
        m = Mirror(*self.seed_at(self.unit))
        look = self.roll(self.unit, 3); want = m.roll(3)
        r = self.roll(self.unit, c["hi"] - c["lo"] + 1); want_r = m.roll(c["hi"] - c["lo"] + 1)
        assert (look, r) == (want, want_r), ("roll differs from the mirror", look, want, r, want_r)
        return c["lo"] + r


def info():
    g = Game()
    for cid, label in CLASSES:
        c = g.rows[cid]
        print(cid, label, " ".join(f"{k}={v}" for k, v in c.items() if k != "label"))


def life(games=GAMES):
    g = Game()
    for cid, label in CLASSES:
        if "+q" in cid: continue
        lines = []
        for game in range(games):
            gaps = g.new_game(game, g.rows[cid]["row"])
            for _ in range(PER):
                g.make_unit(game, gaps)
                lines.append(str(g.look_and_life(cid)))
        open(f"{SCRATCH}/life_{cid}.txt", "w").write("\n".join(lines) + "\n")
        print(cid, len(lines), "lives written")


def swing(cfg):
    defence, dlvl, units = SWINGS[cfg]
    g = Game()
    e = g.e
    stats = {}
    e.hook(0x625480, lambda e: stats.get((e.arg(0), e.arg(1)), 0), 3)
    e.hook(0x625500, lambda e: stats.get((e.arg(0), e.arg(1)), 0), 3)
    e.hook(0x6223f0, lambda e: stats.get((e.arg(0), "def"), 0), 1)
    e.hook(0x639df0, lambda e: 0, 2)
    e.hook(0x554010, lambda e: 0, 3)
    e.hook(0x5347b0, lambda e: 0, 0)
    e.hook(0x57d810, lambda e: 0, 0)
    e.hook(0x535bc0, lambda e: 0, 0)
    e.hook(0x46c140, lambda e: 0, 0)
    dfd = e.alloc(0x100)
    e.w32(dfd, 0); e.w32(dfd + 0x10, 1)
    for cid, label in CLASSES:
        if "+q" in cid: continue
        c = g.rows[cid]
        lines = []
        done, game = 0, 0
        gaps = g.new_game(game, c["row"])
        per_game = 20
        while done < units:
            if done and done % per_game == 0:
                game += 1
                gaps = g.new_game(game, c["row"])
            g.make_unit(game, gaps)
            g.look_and_life(cid)
            att = g.unit
            stats.clear()
            stats.update({(att, 0x13): c["th"], (att, 0xc): c["level"], (att, 0x77): 0, (att, 0x15): c["a1lo"], (att, 0x16): c["a1hi"],
                          (dfd, 0xc): dlvl, (dfd, "def"): defence, (dfd, 0x21): 0, (dfd, 0x20): 0})
            for _ in range(100):
                hit = e.call(0x57d9b0, 0, 0, ecx=att, edx=dfd)
                dmg = 0
                if hit:
                    dmg = e.call(0x57b420, 0, 0, 0, 0, 0, 0x80, ecx=att, edx=1)
                    dmg = dmg - (1 << 32) if dmg >= (1 << 31) else dmg
                lines.append(f"{1 if hit else 0} {dmg}")
            done += 1
        open(f"{SCRATCH}/swing_{cid}_{cfg}.txt", "w").write("\n".join(lines) + "\n")
        print(cid, "cfg", cfg, len(lines), "swings", sum(1 for l in lines if l[0] == "1"), "hits")


def drop_oracle():
    import drops
    o = drops.Oracle()
    g = Game(o.e)
    return o, g


def roll_drop(o, cid, c, low, high, mf, ilvl=None):
    e = o.e
    rec = e.call(0x654e00, o.index[c["tc"]], 0)
    assert o.names[(rec - e.r32(0x96c5ec)) // 0x2c] == c["tc"], "class moved on at level 0"
    e.w32(o.unit + 0x20, low); e.w32(o.unit + 0x24, high)
    o.made, o.players, o.mf = [], 1, mf
    e.call(0x55a6d0, 0, rec, 0, c["level"] if ilvl is None else ilvl, 0, 0, 0, 0, ecx=o.game, edx=o.unit)
    out = ",".join(f"{code}:{q}" + (f"*{m}" if m else "") for code, q, m in o.made) or "-"
    return out, e.r32(o.unit + 0x20)


def drop(cfg):
    mf, kmode = DROPS[cfg]
    o, g = drop_oracle()
    for cid, label in CLASSES:
        c = g.rows[cid]
        lines = []
        for game in range(GAMES):
            gaps = g.new_game(game, c["row"])
            extra = random.Random(game * 104729 + cfg)
            for _ in range(PER):
                g.make_unit(game, gaps)
                g.look_and_life(cid)
                n = extra.randint(0, kmode - 1000) if kmode >= 1000 else kmode
                for _ in range(n): g.e.call(STEP, ecx=g.unit + 0x20)
                low, high = g.seed_at(g.unit)
                lines.append(roll_drop(o, cid, c, low, high, mf)[0])
        open(f"{SCRATCH}/drop_{cid}_{cfg}.txt", "w").write("\n".join(lines) + "\n")
        print(cid, "cfg", cfg, len(lines), "drops", sum(1 for l in lines if l != "-"), "non-empty")


def ctl():
    o, g = drop_oracle()
    rng = random.Random(20261102)
    for cid, label in CLASSES:
        c = g.rows[cid]
        states = [(rng.getrandbits(32), rng.choice((666, rng.getrandbits(32)))) for _ in range(CTL_N)]
        open(f"{SCRATCH}/ctl_{cid}.seeds", "w").write("".join(f"{lo} {hi} {c['level']}\n" for lo, hi in states))
        for mf in (0, 300):
            res = [roll_drop(o, cid, c, lo, hi, mf) for lo, hi in states]
            open(f"{SCRATCH}/ctl_{cid}_{mf}.game", "w").write("".join(f"{a} {b:08x}\n" for a, b in res))
        # life on the same states
        lines = []
        for lo, hi in states:
            e = g.e
            e.w32(g.unit + 0x20, lo); e.w32(g.unit + 0x24, hi)
            r = g.roll(g.unit, c["hi"] - c["lo"] + 1)
            lines.append(f"{c['lo'] + r} {e.r32(g.unit + 0x20):08x}")
        open(f"{SCRATCH}/ctlife_{cid}.game", "w").write("\n".join(lines) + "\n")
        print(cid, "ctl", len(states), "states")


if __name__ == "__main__":
    mode = sys.argv[1]
    if mode == "info": info()
    elif mode == "life": life(int(sys.argv[2]) if len(sys.argv) > 2 else GAMES)
    elif mode == "swing": swing(int(sys.argv[2]))
    elif mode == "drop": drop(int(sys.argv[2]))
    elif mode == "ctl": ctl()
    else: raise SystemExit("modes: info life swing CFG drop CFG ctl")
