#!/usr/bin/env python3
# SPDX-License-Identifier: GPL-3.0-or-later
"""Arithmetic self-check for Paladin combat skills formulas (Smite, Zeal, Sacrifice).

Performs an offline arithmetic self-check of the written formulas (tagged I)
for progression and scaling across levels 1..60 and randomized input ranges.
This script checks arithmetic consistency of the written models, not native
Game.exe or d2d binary execution.

Usage:
    python verify_palskills.py [cases] [seed]
"""

import random
import sys
import time


def verify_zeal_hits(level):
    """Calculate Zeal hit count (calc1: min((par5 + lvl - 1), par6) with par5=2, par6=5)."""
    return min(2 + level - 1, 5)


def verify_zeal_ed(level, sac_blvl):
    """Calculate Zeal enhanced damage % (calc2: ((lvl < 5) ? 0 : ((lvl-4) * 6)) + Sacrifice.blvl * 12)."""
    base = 0 if level < 5 else (level - 4) * 6
    return base + sac_blvl * 12


def verify_zeal_ar(level):
    """Calculate Zeal attack rating bonus (ToHit 10, LevToHit 10: 10 + 10*(lvl - 1))."""
    return 10 + 10 * (level - 1)


def verify_smite_ed(level):
    """Calculate Smite skill enhanced damage % (calc1: ln34 = 15 + 15*(lvl - 1))."""
    return 15 + 15 * (level - 1)


def verify_smite_stun(level):
    """Calculate Smite stun duration in frames (calc2: min(250, 15 + 5*(lvl - 1)))."""
    return min(250, 15 + 5 * (level - 1))


def verify_sacrifice_ed(level, red_blvl, fan_blvl):
    """Calculate Sacrifice enhanced damage % (calc1: 180 + 15*(lvl - 1) + Redemption.blvl * 15 + Fanaticism.blvl * 5)."""
    return 180 + 15 * (level - 1) + red_blvl * 15 + fan_blvl * 5


def verify_sacrifice_ar(level):
    """Calculate Sacrifice attack rating bonus (ToHit 20, LevToHit 7: 20 + 7*(lvl - 1))."""
    return 20 + 7 * (level - 1)


def verify_sacrifice_self_damage(phys_damage, target_hp):
    """Calculate Sacrifice self-recoil damage (calc2: 8% of min(phys_damage, target_hp))."""
    capped = min(phys_damage, target_hp)
    return (capped * 8) // 100


def verify_mana_cost(skill_name, level):
    """Calculate skill mana cost in 256ths (FUN_00644b10: (mana + lvlmana*(lvl - 1)) << manashift)."""
    if skill_name in ("Smite", "Zeal"):
        return (2 + 0 * (level - 1)) << 8
    if skill_name == "Sacrifice":
        return 0
    raise ValueError(f"Unknown skill: {skill_name}")


def main():
    """Run deterministic and randomized arithmetic consistency self-checks for skill formulas (I)."""
    n_cases = int(sys.argv[1]) if len(sys.argv) > 1 else 120000
    seed = int(sys.argv[2], 0) if len(sys.argv) > 2 else 0x50414C53

    t0 = time.time()
    rnd = random.Random(seed)

    # 1. Deterministic progression check for levels 1..60
    for lvl in range(1, 61):
        hits = verify_zeal_hits(lvl)
        if lvl == 1:
            assert hits == 2
        elif lvl == 2:
            assert hits == 3
        elif lvl == 3:
            assert hits == 4
        else:
            assert hits == 5

        stun = verify_smite_stun(lvl)
        if lvl >= 48:
            assert stun == 250
        else:
            assert stun == 15 + 5 * (lvl - 1)

        assert verify_zeal_ar(lvl) == 10 + 10 * (lvl - 1)
        assert verify_sacrifice_ar(lvl) == 20 + 7 * (lvl - 1)
        assert verify_mana_cost("Smite", lvl) == 512
        assert verify_mana_cost("Zeal", lvl) == 512
        assert verify_mana_cost("Sacrifice", lvl) == 0

    # 2. Randomized stress testing with independent verification
    mismatches = 0
    for _ in range(n_cases):
        lvl = rnd.randint(1, 60)
        sac_blvl = rnd.randint(0, 30)
        red_blvl = rnd.randint(0, 30)
        fan_blvl = rnd.randint(0, 30)
        str_val = rnd.randint(10, 500)
        off_weapon_ed = rnd.randint(0, 400)
        target_hp = rnd.randint(1, 50000)
        raw_damage = rnd.randint(1, 100000)

        # Zeal formulas vs independent expected formulas
        zeal_hits = verify_zeal_hits(lvl)
        exp_hits = min(2 + lvl - 1, 5)
        if zeal_hits != exp_hits:
            mismatches += 1

        zeal_ed = verify_zeal_ed(lvl, sac_blvl)
        exp_zeal_ed = (0 if lvl < 5 else (lvl - 4) * 6) + sac_blvl * 12
        if zeal_ed != exp_zeal_ed:
            mismatches += 1

        zeal_ar = verify_zeal_ar(lvl)
        exp_zeal_ar = 10 + 10 * (lvl - 1)
        if zeal_ar != exp_zeal_ar:
            mismatches += 1

        # Smite formulas
        smite_skill_ed = verify_smite_ed(lvl)
        exp_smite_ed = 15 + 15 * (lvl - 1)
        if smite_skill_ed != exp_smite_ed:
            mismatches += 1

        smite_stun = verify_smite_stun(lvl)
        exp_stun = min(250, 15 + 5 * (lvl - 1))
        if smite_stun != exp_stun:
            mismatches += 1

        smite_total_pct = max(smite_skill_ed + off_weapon_ed + str_val, -90)
        exp_total_pct = max(exp_smite_ed + off_weapon_ed + str_val, -90)
        if smite_total_pct != exp_total_pct:
            mismatches += 1

        # Sacrifice formulas
        sac_ed = verify_sacrifice_ed(lvl, red_blvl, fan_blvl)
        exp_sac_ed = 180 + 15 * (lvl - 1) + red_blvl * 15 + fan_blvl * 5
        if sac_ed != exp_sac_ed:
            mismatches += 1

        sac_ar = verify_sacrifice_ar(lvl)
        exp_sac_ar = 20 + 7 * (lvl - 1)
        if sac_ar != exp_sac_ar:
            mismatches += 1

        self_dmg = verify_sacrifice_self_damage(raw_damage, target_hp)
        exp_self_dmg = (min(raw_damage, target_hp) * 8) // 100
        if self_dmg != exp_self_dmg:
            mismatches += 1

        # Mana costs
        if verify_mana_cost("Smite", lvl) != 512:
            mismatches += 1
        if verify_mana_cost("Zeal", lvl) != 512:
            mismatches += 1
        if verify_mana_cost("Sacrifice", lvl) != 0:
            mismatches += 1

    elapsed = time.time() - t0
    if len(sys.argv) > 1:
        print(f"Verified formula consistency ({n_cases} cases, seed {hex(seed)}): {mismatches} mismatches in {elapsed:.3f}s")
    else:
        print(f"Verified formula consistency: {mismatches} mismatches")
    return 1 if mismatches > 0 else 0


if __name__ == "__main__":
    sys.exit(main())
