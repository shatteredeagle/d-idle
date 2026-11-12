# SPDX-License-Identifier: GPL-3.0-or-later
"""decomp13 PALSKILL matrix: the Paladin kill-speed model (tag I: arithmetic over V/S/D inputs; it does NOT run game.exe).
    python3 pm_dps.py table     per-skill damage per second at clvl 6 and 12 (rows of docs/research/pal-skill-matrix.md section 4)
    python3 pm_dps.py order     the best skill-point allocation per clvl 1..20 (section 5)
Inputs and where they come from (docs/research/pal-skill-matrix.md section 2): Skills.txt rows (D), swing ticks = ceil(256 frames / rate) (S,
skiasgame2 198,988 cases), Zeal strike spacing = the action frame (V run, tools/emu/zd_game.py; d2d = the full swing, V run, zd_zeal_test),
to-hit = AR' / (AR' + DEF) x 2 alvl / (alvl + dlvl), 5..95 with AR = (dex - 7) x 5 + CharStats ToHitFactor 20 (S, sk-melee-trace.md),
damage = weapon avg x (100 + Str x StrBonus/100 + skill ED + Might) / 100 (S sk-melee-trace.md; additivity of Might I)."""
import math
import sys

SSD = dict(name="Short Sword", lo=2, hi=7, wsm=0, frames=15, act=7)      # D weapons.txt, AnimData PAA11HS (decomp9)
MAC = dict(name="Mace", lo=3, hi=10, wsm=0, frames=15, act=7)


def ticks(frames, wsm, ias=0):
    rate = 256 * (100 - wsm) // 100 + (120 * ias // (120 + ias)) * 256 // 100   # S rate rule, IAS term as fid-pal-attack.md
    return math.ceil(256 * frames / rate), rate


def hit(ar, bonus, defense, alvl, dlvl):
    a = ar * (100 + bonus) / 100
    return min(95, max(5, a / (a + defense) * 100 * 2 * alvl / (alvl + dlvl))) / 100


def might(l): return 0 if l <= 0 else 40 + 10 * (l - 1)


def sac_ed(l, red=0, fan=0): return 180 + 15 * (l - 1) + 15 * red + 5 * fan
def sac_ar(l): return 20 + 7 * (l - 1)
def zeal_n(l): return min(l + 1, 5)
def zeal_ed(l, sac): return (0 if l < 5 else (l - 4) * 6) + 12 * sac
def zeal_ar(l): return 10 + 10 * (l - 1)
def smite_ed(l): return 15 + 15 * (l - 1)
def hf_pulse(l): return ((256 + 128 * (l - 1)) + (768 + 128 * (l - 1)) - 1) / 2 / 256     # points per pulse, roll min..max-1 (V run pm_dmg.py)


def per_target(clvl, str_, dex, wpn, mlvl, defense, skills, attack, d2d=False, ias=0):
    """damage per second against one monster, life price per second, for `attack` in Attack / Sacrifice / Zeal / Smite."""
    ar = (dex - 7) * 5 + 20
    avg = (wpn["lo"] + wpn["hi"]) / 2
    t, rate = ticks(wpn["frames"], wpn["wsm"], ias)
    m = might(skills.get("Might", 0))
    base = str_ + m
    if attack == "Attack":
        dmg = avg * (100 + base) / 100 * hit(ar, 0, defense, clvl, mlvl)
        return dmg * 25 / t, 0.0, t
    if attack == "Sacrifice":
        l = skills["Sacrifice"]
        d = avg * (100 + base + sac_ed(l)) / 100
        h = hit(ar, sac_ar(l), defense, clvl, mlvl)
        return d * h * 25 / t, 0.08 * d * h * 25 / t, t
    if attack == "Zeal":
        l = skills["Zeal"]
        n = zeal_n(l)
        d = avg * (100 + base + zeal_ed(l, skills.get("Sacrifice", 0))) / 100
        h = hit(ar, zeal_ar(l), defense, clvl, mlvl)
        act = math.ceil(256 * wpn["act"] / rate)
        cast = n * t if d2d else n * act + (t - act)       # d2d waits the full swing per strike; game.exe waits `action frame` ticks (zd_game.py)
        return n * d * h * 25 / cast, 0.0, cast / n
    if attack == "Smite":
        l = skills["Smite"]
        d = 2.0 * (100 + base + smite_ed(l)) / 100          # Buckler 1-3 (D armor table, palskill-trace.md Q2); always hits (V read)
        return d * 25 / 12, 0.0, 12
    raise ValueError(attack)


def table():
    cases = [
        (6, 34, 21, SSD, 6, {"Might": 4, "Sacrifice": 1, "Prayer": 1}, "v2 table at clvl 6 (Might 4, Sacrifice 1, Prayer 1)"),
        (12, 46, 21, SSD, 12, {"Might": 8, "Sacrifice": 1, "Smite": 1, "Zeal": 1}, "v2 table at clvl 12 (Might 8, Sacrifice 1, Smite 1, Zeal 1)"),
        (12, 46, 21, SSD, 12, {"Might": 8, "Sacrifice": 1, "Smite": 1, "Zeal": 4}, "same, Zeal 4 (v2 reaches it at clvl 15)"),
        (12, 46, 21, MAC, 12, {"Might": 8, "Sacrifice": 1, "Smite": 1, "Zeal": 4}, "same, Mace 3-10 instead of the Short Sword"),
    ]
    for clvl, s, d, w, ml, sk, label in cases:
        for defense in (15, 60):
            print("## %s | %s | defense %d (monster level %d)" % (label, w["name"], defense, ml))
            for atk in ("Attack", "Sacrifice", "Zeal", "Smite"):
                if atk not in sk and atk != "Attack":
                    continue
                if atk == "Sacrifice" and sk.get("Sacrifice", 0) < 1:
                    continue
                dps, life, tk = per_target(clvl, s, d, w, ml, defense, sk, atk)
                extra = ""
                if atk == "Zeal":
                    dps2, _, tk2 = per_target(clvl, s, d, w, ml, defense, sk, atk, d2d=True)
                    extra = "  | d2d %.1f dps (%.0f ticks per strike)" % (dps2, tk2)
                print("%-9s %5.1f dps  life price %4.2f/s  (%4.1f ticks per hit)%s" % (atk, dps, life, tk, extra))
    print("## Holy Bolt (undead only, magic, cast 16 ticks at FCR 0, mana 2.0 + 1/16 per level) average per cast: level 1 12, 2 20, 3 28, 4 36 (V run pm_dmg.py, midpoints)")
    for l in (1, 2, 3, 4):
        print("level %d: %.1f dps at 16 ticks" % (l, (8 * l + 4 if l == 1 else 8 * l + 4) * 25 / 16))
    print("## Holy Fire single target, pulse every 50 frames:")
    for l in (1, 2, 3, 4, 6):
        print("level %d: %.2f points per pulse = %.2f dps per monster in range %d subtiles" % (l, hf_pulse(l), hf_pulse(l) / 2, 6 + (l - 1)))


def order():
    """For each clvl, the allocation of P = clvl points (clvl 2..L one point each, plus the Den of Evil reward; U that the bot takes it)
    over Might (m), Sacrifice (s) and Zeal (z, needs clvl >= 12 and s >= 1, z <= clvl - 11), best single-target dps per attack choice."""
    str_of = lambda c: 25 + int(0.5 * 2 * max(0, c - 1))     # v2 table: about 2 Str per level
    print("clvl | best allocation (attack) -> dps | plain Attack + Might-only dps | ratio | life price/s")
    for c in range(1, 21):
        P = c  # c-1 level points + 1 Den reward (from clvl 5 on in a normal game; I)
        P = c - 1 + (1 if c >= 5 else 0)
        best = None
        for s in range(0, P + 1):
            for z in range(0, min(P - s, max(0, c - 11), 20) + 1):
                if z and s < 1:
                    continue
                m = P - s - z
                sk = {"Might": m, "Sacrifice": s, "Zeal": z}
                for atk in ("Attack", "Sacrifice", "Zeal"):
                    if atk == "Sacrifice" and s < 1: continue
                    if atk == "Zeal" and z < 1: continue
                    dps, life, _ = per_target(c, str_of(c), 21, SSD, c, 15, sk, atk)
                    if best is None or dps > best[0]:
                        best = (dps, life, atk, sk)
        mo = {"Might": P}
        dps0, _, _ = per_target(c, str_of(c), 21, SSD, c, 15, mo, "Attack")
        print("%2d | %-9s %s -> %5.1f | %5.1f | x%.2f | %.2f" % (c, best[2], {k: v for k, v in best[3].items() if v}, best[0], dps0, best[0] / dps0, best[1]))


if __name__ == "__main__":
    {"table": table, "order": order}[sys.argv[1]]()
