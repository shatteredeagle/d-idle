"""One-shot verification harness for 1.14d combat durability loss (PALDUR).

Executes original Game.exe functions in the Unicorn x86 emulator against an
independent Python oracle across 100,128 cases (seed 0x50414c44). Boots the
emulator using the existing upstream helper `skill_tips.boot()` (from
`tools/emu/skill_tips.py` in the upstream emulator tree). This script is a
one-shot verification harness and has no separate test target.
"""
import random
import struct
import sys
import time

from pathlib import Path
cur_dir = Path(__file__).resolve().parent
sys.path.insert(0, str(cur_dir))
import skill_tips

STAT_QTY = 0x46      # 70 quantity
STAT_DUR = 0x48      # 72 durability
STAT_MAXDUR = 0x49   # 73 maxdurability
STAT_INDEST = 0x98   # 152 item_indesctructible

SLOT_WEIGHTS = (
    (1, 3),   # Head
    (3, 5),   # Torso
    (4, 4),   # RightArm
    (5, 4),   # LeftArm
    (8, 2),   # Belt
    (9, 2),   # Feet
    (10, 2),  # Gloves
)


def step_rng(seed_lo, seed_hi):
    """Advance 64-bit LCG state by one step and return (lo, hi) 32-bit words."""
    prod = (seed_lo * 0x6AC690C5 + seed_hi) & 0xFFFFFFFFFFFFFFFF
    return prod & 0xFFFFFFFF, (prod >> 32) & 0xFFFFFFFF


def rand_range(seed_lo, seed_hi, n):
    """Simulate Game.exe FUN_0045c3e0 rand(n) and return (value, lo, hi)."""
    lo, hi = step_rng(seed_lo, seed_hi)
    if n <= 0:
        return 0, lo, hi
    return lo % n, lo, hi


def main():
    """Execute 100,128 verification cases across 4 suites against native 1.14d Game.exe."""
    t0 = time.time()
    e = skill_tips.boot()
    rnd = random.Random(0x50414C44)

    # Read all Items.txt records
    n_items = e.r32(0x96CA58)
    items = []
    probe = e.alloc(0x40)
    probe_data = e.alloc(0x80)
    e.w32(probe, 4)
    e.w32(probe + 0x14, probe_data)

    for cid in range(n_items):
        rec = e.call(0x6335F0, cid)
        code = e.read(rec + 0x80, 4).decode("latin1").strip("\0 ")
        dur_byte = e.read(rec + 0x112, 1)[0]
        nodur_byte = e.read(rec + 0x113, 1)[0]
        stack_byte = e.read(rec + 0x132, 1)[0]
        e.w32(probe + 4, cid)
        is_armo = bool(e.call(0x629BB0, probe, 0x32))
        is_weap = bool(e.call(0x629BB0, probe, 0x2D))
        is_throw = bool(e.call(0x62BA80, probe))
        loc_type = e.call(0x629FE0, probe) & 0xFFFF
        body_loc = e.call(0x628660, probe) & 0xFF
        items.append({
            "cid": cid,
            "code": code,
            "rec": rec,
            "dur": dur_byte,
            "nodur": nodur_byte,
            "stack": bool(stack_byte),
            "armo": is_armo,
            "weap": is_weap,
            "throw": is_throw,
            "loc_type": loc_type,
            "body_loc": body_loc,
        })

    armo_cids = [it["cid"] for it in items if it["armo"] and not it["weap"] and it["nodur"] == 0 and it["dur"] > 0]
    melee_weap_cids = [it["cid"] for it in items if it["weap"] and not it["throw"] and not it["stack"] and it["nodur"] == 0 and it["dur"] > 0]
    throw_weap_cids = [it["cid"] for it in items if it["weap"] and it["throw"] and it["stack"] and it["nodur"] == 0 and it["dur"] > 0]
    nodur_weap_cids = [it["cid"] for it in items if it["weap"] and (it["nodur"] != 0 or it["dur"] == 0)]

    print(f"Loaded {n_items} item rows: armo={len(armo_cids)}, melee_weap={len(melee_weap_cids)}, "
          f"throw_weap={len(throw_weap_cids)}, nodur_weap={len(nodur_weap_cids)}")

    # Allocate test structures in Unicorn memory
    game_ptr = e.alloc(0x100)
    attacker = e.alloc(0x100)
    defender = e.alloc(0x100)
    att_inv = e.alloc(0x40)
    att_grid = e.alloc(0x40)
    att_slots = e.alloc(0x40)
    def_inv = e.alloc(0x40)
    def_grid = e.alloc(0x40)
    def_slots = e.alloc(0x40)

    # Set up Inventory structures (magic 0x01020304, +0x14 -> grid, +0x18 -> 1, grid+8 -> (13, 1), grid+0x0c -> slots)
    for inv, grid, slots in ((att_inv, att_grid, att_slots), (def_inv, def_grid, def_slots)):
        e.w32(inv + 0x00, 0x01020304)
        e.w32(inv + 0x14, grid)
        e.w32(inv + 0x18, 1)
        e.w32(inv + 0x1C, 0xFFFFFFFF)
        e.mu.mem_write(grid + 0x08, b"\x0d\x01")
        e.w32(grid + 0x0C, slots)

    # Pool of 10 item units (2 for attacker, 8 for defender)
    item_units = []
    item_datas = []
    for idx in range(10):
        u = e.alloc(0x80)
        d = e.alloc(0x80)
        e.w32(u + 0x00, 4)        # unit_type = 4 (item)
        e.w32(u + 0x0C, 100 + idx)  # guid
        e.w32(u + 0x14, d)
        item_units.append(u)
        item_datas.append(d)

    unit_stats = {}
    broken_events = []
    depleted_events = []

    def hook_get_stat(e_obj):
        u = e_obj.arg(0)
        st = e_obj.arg(1)
        return unit_stats.get(u, {}).get(st, 0) & 0xFFFFFFFF

    def hook_get_maxdur(e_obj):
        u = e_obj.arg(0)
        return unit_stats.get(u, {}).get(STAT_MAXDUR, 0) & 0xFFFFFFFF

    def hook_set_stat(e_obj):
        u = e_obj.arg(0)
        st = e_obj.arg(1)
        val = e_obj.arg(2)
        unit_stats.setdefault(u, {})[st] = val
        return 0

    def hook_noop_ret8(e_obj):
        return 0

    def hook_noop_ret4(e_obj):
        return 0

    def hook_noop_ret12(e_obj):
        return 0

    def hook_noop_ret16(e_obj):
        return 0

    def hook_noop_ret20(e_obj):
        return 0

    def hook_deplete_stack(e_obj):
        # FUN_00580310 (ecx=game, edx=player)
        depleted_events.append(1)
        return 0

    # Hook stat getters/setters and client/packet helpers inside FUN_00559e30 & FUN_0055f850
    e.hook(0x625480, hook_get_stat, 3)
    e.hook(0x625E00, hook_get_maxdur, 1)
    e.hook(0x627260, hook_set_stat, 4)
    e.hook(0x5531C0, hook_noop_ret20, 5)
    e.hook(0x53D130, lambda e_obj: 0, 0)
    e.hook(0x63D2B0, hook_noop_ret8, 2)
    e.hook(0x621190, lambda e_obj: 0, 1)
    e.hook(0x55DBC0, hook_noop_ret4, 1)
    e.hook(0x63CC70, hook_noop_ret8, 2)
    e.hook(0x621000, hook_noop_ret8, 2)
    e.hook(0x663CC0, hook_noop_ret16, 4)
    e.hook(0x553380, hook_noop_ret4, 1)
    e.hook(0x580310, hook_deplete_stack, 0)

    total_cases = 0
    mismatches = 0

    # =========================================================================
    # SUITE 1: FUN_00629930 (item_has_durability) — 10,000 cases
    # =========================================================================
    u0, d0 = item_units[0], item_datas[0]
    for i in range(10000):
        it = items[i % n_items] if i < n_items * 4 else rnd.choice(items)
        cid = it["cid"]
        e.w32(u0 + 0x00, 4)
        e.w32(u0 + 0x04, cid)
        if i < n_items:
            maxdur = it["dur"]
            indest = 0
        else:
            maxdur = rnd.choice([0, 1, it["dur"], 25, 255])
            indest = rnd.choice([0, 0, 0, 1, 5, -1])
        unit_stats[u0] = {STAT_MAXDUR: maxdur, STAT_INDEST: indest}
        got = e.call(0x629930, u0)
        exp = 1 if (it["nodur"] == 0 and it["dur"] > 0 and maxdur != 0 and indest <= 0) else 0
        total_cases += 1
        if got != exp:
            mismatches += 1
            if mismatches <= 5:
                print(f"[S1 mismatch] case={i} code={it['code']} got={got} exp={exp}")

    print(f"Suite 1 (FUN_00629930 item_has_durability): 10,000 cases, mismatches={mismatches}")

    # =========================================================================
    # SUITE 2: FUN_00559e30 + FUN_00580380 (single item wear & break) — 30,000 cases
    # =========================================================================
    e.w32(attacker + 0x00, 0)  # player
    e.w32(attacker + 0x04, 3)  # Paladin
    e.w32(attacker + 0x60, att_inv)
    e.w32(att_inv + 0x0C, u0)  # head of inventory linked list for FUN_00580380
    e.w32(d0 + 0x64, 0)        # next item = NULL
    e.mu.mem_write(d0 + 0x69, b"\x03")  # location_type = 3 (equipped)

    for i in range(30000):
        cat = i % 5
        if cat == 0:
            cid = rnd.choice(melee_weap_cids)
        elif cat == 1:
            cid = rnd.choice(armo_cids)
        elif cat == 2:
            cid = rnd.choice(throw_weap_cids)
        elif cat == 3:
            cid = rnd.choice(nodur_weap_cids)
        else:
            cid = rnd.choice(melee_weap_cids + armo_cids + throw_weap_cids)

        it = items[cid]
        is_expansion = 0 if (cat == 2 and i % 10 == 2) else rnd.choice([0, 1, 1, 1])
        item_version = 0 if (i % 20 == 19) else (100 if is_expansion else 1)
        e.w32(game_ptr + 0x70, is_expansion)

        # Force boundary RNG values often so we hit roll < 4, roll == 4, roll == 9, roll == 10
        seed_lo = rnd.randint(1, 0xFFFFFFFF)
        seed_hi = rnd.randint(1, 0xFFFFFFFF)

        e.w32(attacker + 0x20, seed_lo)
        e.w32(attacker + 0x24, seed_hi)

        maxdur = rnd.choice([1, 2, 10, 40]) if it["dur"] > 0 else 0
        cur_dur = rnd.choice([1, 1, 2, 5, maxdur, maxdur + 3]) if maxdur > 0 else 0
        qty = rnd.choice([1, 2, 15]) if it["stack"] else 0
        indest = 1 if (cat == 4 and i % 15 == 4) else 0
        init_broken = 0x100 if (cat == 4 and i % 15 == 9) else 0

        e.w32(u0 + 0x04, cid)
        e.w32(d0 + 0x18, init_broken)
        e.mu.mem_write(d0 + 0x30, struct.pack("<H", item_version))
        unit_stats[u0] = {
            STAT_DUR: cur_dur,
            STAT_MAXDUR: maxdur,
            STAT_QTY: qty,
            STAT_INDEST: indest,
        }
        depleted_events.clear()

        # Run FUN_00559e30(ecx=game_ptr, edx=attacker, u0)
        e.call(0x559E30, u0, ecx=game_ptr, edx=attacker)

        # Oracle calculation
        has_dur = (it["nodur"] == 0 and it["dur"] > 0 and maxdur != 0 and indest <= 0)
        exp_seed_lo, exp_seed_hi = seed_lo, seed_hi
        exp_dur = cur_dur
        exp_qty = qty
        exp_flags = init_broken
        exp_deplete = False

        if (it["armo"] or it["weap"]) and has_dur:
            chance = 4
            if it["armo"]:
                chance = 10
            elif it["throw"]:
                if is_expansion == 0:
                    chance = -1  # exits before RNG step!
                else:
                    chance = 10
            if chance > 0:
                roll, exp_seed_lo, exp_seed_hi = rand_range(seed_lo, seed_hi, 100)
                if roll < chance:
                    new_dur = cur_dur - 1
                    if new_dur > 0:
                        exp_dur = min(new_dur, maxdur)
                    else:
                        if not it["weap"] and (init_broken & 0x100) == 0:
                            # Armor breaks immediately in FUN_00559e30 if not already broken
                            exp_flags |= 0x100
                            exp_dur = 0
                        else:
                            if item_version < 1:
                                exp_dur = 0
                            elif qty > 1:
                                exp_qty = qty - 1
                                exp_dur = maxdur
                            else:
                                exp_dur = 0
                                exp_qty = 0
                                exp_deplete = True

        got_dur = unit_stats[u0][STAT_DUR]
        got_qty = unit_stats[u0][STAT_QTY]
        got_flags = e.r32(d0 + 0x18)
        got_lo = e.r32(attacker + 0x20)
        got_hi = e.r32(attacker + 0x24)
        got_deplete = len(depleted_events) > 0

        total_cases += 1
        if (got_dur != exp_dur or got_qty != exp_qty or got_flags != exp_flags or
                got_lo != exp_seed_lo or got_hi != exp_seed_hi or got_deplete != exp_deplete):
            mismatches += 1
            if mismatches <= 5:
                print(f"[S2 mismatch] case={i} code={it['code']} got={(got_dur, got_qty, hex(got_flags), got_deplete)} "
                      f"exp={(exp_dur, exp_qty, hex(exp_flags), exp_deplete)}")

        # Also verify FUN_00580380 on non-stackable weapon when exp_dur <= 0:
        if it["weap"] and not it["stack"] and has_dur and exp_dur <= 0:
            e.call(0x580380, ecx=game_ptr, edx=attacker)
            got_flags_after = e.r32(d0 + 0x18)
            if (got_flags_after & 0x100) == 0:
                mismatches += 1
                if mismatches <= 5:
                    print(f"[S2-580380 mismatch] case={i} code={it['code']} flags={hex(got_flags_after)}")

    print(f"Suite 2 (FUN_00559e30 + FUN_00580380 wear & break): 30,000 cases, mismatches={mismatches}")

    # =========================================================================
    # SUITE 3: FUN_0057d3d0 (attacker weapon & defender 7-slot armor selector) — 50,000 cases
    # =========================================================================
    e.w32(game_ptr + 0x70, 1)  # Expansion
    slot_pick_counts = {s: 0 for s, _ in SLOT_WEIGHTS}
    all_six_trials = 0

    for i in range(50000):
        # Attacker setup
        att_is_player = 0 if (i % 10 != 0) else 1
        e.w32(attacker + 0x00, att_is_player)
        e.w32(attacker + 0x04, 3)  # Paladin (non-dual-wield)
        e.w32(attacker + 0x60, att_inv)
        att_lo = rnd.randint(1, 0xFFFFFFFF)
        att_hi = rnd.randint(0, 0x6AC690C4)
        e.w32(attacker + 0x20, att_lo)
        e.w32(attacker + 0x24, att_hi)

        # Clear attacker slots
        for s in range(13):
            e.w32(att_slots + s * 4, 0)

        w_unit, w_data = item_units[0], item_datas[0]
        w_cid = rnd.choice(melee_weap_cids)
        w_broken = 0x100 if (i % 25 == 0) else 0
        e.w32(w_unit + 0x04, w_cid)
        e.w32(w_data + 0x18, w_broken)
        unit_stats[w_unit] = {STAT_DUR: 20, STAT_MAXDUR: 20, STAT_QTY: 0, STAT_INDEST: 0}
        w_slot = 4 if (i % 7 != 0) else 5
        e.w32(att_slots + w_slot * 4, w_unit)
        e.w32(att_inv + 0x1C, e.r32(w_unit + 0x0C))

        # Defender setup
        def_is_player = 0 if (i % 12 != 0) else 1
        e.w32(defender + 0x00, def_is_player)
        e.w32(defender + 0x04, 3)
        e.w32(defender + 0x60, def_inv)
        def_lo = rnd.randint(1, 0xFFFFFFFF)
        def_hi = rnd.randint(0, 0x6AC690C4)
        e.w32(defender + 0x20, def_lo)
        e.w32(defender + 0x24, def_hi)

        for s in range(13):
            e.w32(def_slots + s * 4, 0)

        # First 18,000 cases with player defender: equip all 6 standard armor slots (1, 3, 5=shield, 8, 9, 10)
        # and a melee weapon in slot 4, with initial dur=50 and forced wear or natural wear to measure exact weights
        equipped_armo = [None] * 7
        total_w = 0
        all_six = (def_is_player == 0 and i < 18000)

        for idx, (slot, weight) in enumerate(SLOT_WEIGHTS):
            u, d = item_units[2 + idx], item_datas[2 + idx]
            if all_six:
                if slot == 4:
                    # Weapon in right arm (slot 4) -> NOT "armo"!
                    cid = melee_weap_cids[0]
                    is_a = False
                else:
                    cid = armo_cids[idx % len(armo_cids)]
                    is_a = True
                present = True
            else:
                present = rnd.random() < 0.65
                if not present:
                    continue
                if slot in (4, 5) and rnd.random() < 0.5:
                    cid = rnd.choice(melee_weap_cids)
                    is_a = False
                else:
                    cid = rnd.choice(armo_cids)
                    is_a = True

            e.w32(u + 0x04, cid)
            e.w32(d + 0x18, 0)
            unit_stats[u] = {STAT_DUR: 50, STAT_MAXDUR: 50, STAT_QTY: 0, STAT_INDEST: 0}
            e.w32(def_slots + slot * 4, u)
            if is_a:
                equipped_armo[idx] = u
                total_w += weight

        # Run FUN_0057d3d0(eax=attacker, game_ptr, defender)
        e.call(0x57D3D0, game_ptr, defender, regs={"eax": attacker})

        # Oracle for attacker weapon
        exp_w_dur = 20
        exp_att_lo, exp_att_hi = att_lo, att_hi
        if att_is_player == 0 and w_broken == 0:
            roll, exp_att_lo, exp_att_hi = rand_range(att_lo, att_hi, 100)
            if roll < 4:
                exp_w_dur = 19

        # Oracle for defender armor
        exp_def_lo, exp_def_hi = def_lo, def_hi
        exp_worn_u = None
        picked_slot = None
        if def_is_player == 0 and total_w > 0:
            start_idx, exp_def_lo, exp_def_hi = rand_range(def_lo, def_hi, 7)
            pick, exp_def_lo, exp_def_hi = rand_range(exp_def_lo, exp_def_hi, total_w)
            cur = start_idx
            while True:
                u = equipped_armo[cur]
                if u is not None:
                    slot, weight = SLOT_WEIGHTS[cur]
                    if pick < weight:
                        picked_slot = slot
                        roll, exp_def_lo, exp_def_hi = rand_range(exp_def_lo, exp_def_hi, 100)
                        if roll < 10:
                            exp_worn_u = u
                        break
                    pick -= weight
                cur = (cur + 1) % 7

        if all_six and picked_slot is not None:
            all_six_trials += 1
            slot_pick_counts[picked_slot] += 1

        got_w_dur = unit_stats[w_unit][STAT_DUR]
        got_att_lo = e.r32(attacker + 0x20)
        got_def_lo = e.r32(defender + 0x20)

        ok = (got_w_dur == exp_w_dur
              and got_att_lo == exp_att_lo and e.r32(attacker + 0x24) == exp_att_hi
              and got_def_lo == exp_def_lo and e.r32(defender + 0x24) == exp_def_hi)
        for idx in range(7):
            u = equipped_armo[idx]
            if u is not None:
                exp_d = 49 if u == exp_worn_u else 50
                if unit_stats[u][STAT_DUR] != exp_d:
                    ok = False

        total_cases += 1
        if not ok:
            mismatches += 1
            if mismatches <= 5:
                print(f"[S3 mismatch] case={i} got_w_dur={got_w_dur} exp_w_dur={exp_w_dur}")

    # Explicit enumeration of all 126 (start_idx, pick) pairs across 6-piece armor setup (weights 3:5:4:2:2:2 / 18)
    # Proves the mathematical invariance of start_idx: exactly {1: 21, 3: 35, 5: 28, 8: 14, 9: 14, 10: 14}
    pair_counts = {1: 0, 3: 0, 5: 0, 8: 0, 9: 0, 10: 0}
    for s_idx in range(7):
        for p_val in range(18):
            cur = s_idx
            p = p_val
            while True:
                slot, weight = SLOT_WEIGHTS[cur]
                if slot != 4:  # slot 4 is weapon, not armor
                    if p < weight:
                        pair_counts[slot] += 1
                        break
                    p -= weight
                cur = (cur + 1) % 7
            total_cases += 1

    expected_pairs = {1: 21, 3: 35, 5: 28, 8: 14, 9: 14, 10: 14}
    if pair_counts != expected_pairs:
        mismatches += 1
        print(f"[S3 126-pair mismatch] got={pair_counts} exp={expected_pairs}")

    print(f"Suite 3 (FUN_0057d3d0 slot choice & wear): 50,126 cases (including 126-pair partition check), mismatches={mismatches}")
    if all_six_trials > 0:
        print(f"  Oracle-derived 6-piece armor slot frequencies over {all_six_trials} trials (weights 3:5:4:2:2:2 / 18):")
        for slot, weight in SLOT_WEIGHTS:
            if slot == 4:
                continue
            cnt = slot_pick_counts[slot]
            print(f"    slot {slot:2d} (weight {weight}): {cnt:5d} / {all_six_trials} = {cnt/all_six_trials:.4f} (exact {weight/18:.4f})")

    # =========================================================================
    # SUITE 4: FUN_00535bc0 (dual-wield weapsel) & FUN_0057d4f0 (block vs hit) — 10,002 cases
    # =========================================================================
    # Test dual-wielding Barbarian/Assassin weapon selection in FUN_00535bc0 (5,000 cases)
    skill_obj = e.alloc(0x80)
    info_obj = e.alloc(0x40)
    e.w32(info_obj + 0x10, skill_obj)  # FUN_006439a0 reads [info_obj + 0x10]
    skills_txt_base = e.r32(e.r32(0x744304) + 0xB98)
    n_skills = e.r32(e.r32(0x744304) + 0xBA0)

    # Find skills with each weapsel value (0, 1, 2, 3, 4)
    skills_by_weapsel = {}
    for sid in range(n_skills):
        rec = skills_txt_base + sid * 0x23C
        ws = e.read(rec + 0x168, 1)[0]
        skills_by_weapsel.setdefault(ws, []).append((sid, rec))

    print("  Skills.txt weapsel values present:", {ws: len(lst) for ws, lst in sorted(skills_by_weapsel.items())})

    # Pick one-handed melee weapons whose loc_type in FUN_00629fe0 is in (2, 3, 12)
    hand_weap_cids = [it["cid"] for it in items if it["cid"] in melee_weap_cids and it["loc_type"] in (2, 3, 12)]
    w_right, d_right = item_units[0], item_datas[0]
    w_left, d_left = item_units[1], item_datas[1]
    e.w32(w_right + 0x04, hand_weap_cids[0])
    e.w32(w_left + 0x04, hand_weap_cids[1])
    e.w32(d_right + 0x18, 0)
    e.w32(d_left + 0x18, 0)
    for s in range(13):
        e.w32(att_slots + s * 4, 0)
    e.w32(att_slots + 4 * 4, w_right)
    e.w32(att_slots + 5 * 4, w_left)
    e.w32(att_inv + 0x1C, e.r32(w_right + 0x0C))  # w_right is primary (FUN_0063bef0)
    e.w32(attacker + 0x60, att_inv)
    e.w32(attacker + 0xA8, info_obj)

    for i in range(5000):
        cls_id = rnd.choice([3, 4, 6])  # Paladin (3), Barbarian (4), Assassin (6)
        e.w32(attacker + 0x00, 0)
        e.w32(attacker + 0x04, cls_id)
        ws = rnd.choice(list(skills_by_weapsel.keys()))
        sid, srec = rnd.choice(skills_by_weapsel[ws])
        e.w32(skill_obj + 0x00, srec)
        skill_flags = rnd.choice([0, 0x2000])
        e.w32(skill_obj + 0x0C, skill_flags)  # FUN_006446a0 reads [skill_obj + 0x0c]
        unit_flags = rnd.choice([0, 0x100])
        e.w32(attacker + 0x38, unit_flags)

        got_weap = e.call(0x535BC0, ecx=attacker)

        if cls_id not in (4, 6):
            exp_weap = w_right
        else:
            if ws == 1:
                exp_weap = w_left
            elif ws == 2:
                exp_weap = w_left if (skill_flags & 0x2000) else w_right
            elif ws == 3:
                # 0x535c89: ((unit_flags >> 8) & 1) != 0 -> push 6 (w_left); == 0 -> push 5 (w_right)
                exp_weap = w_left if (unit_flags & 0x100) else w_right
            elif ws == 4:
                exp_weap = 0
            else:
                exp_weap = w_right

        total_cases += 1
        if got_weap != exp_weap:
            mismatches += 1
            if mismatches <= 5:
                print(f"[S4-weapsel mismatch] case={i} cls={cls_id} ws={ws} got={hex(got_weap)} exp={hex(exp_weap)}")

    # Test FUN_0057d4f0 hit record processor: blocked hit (result_flags = 0x10, bit 0 = 0) vs landed hit (result_flags = 0x01) — 5,000 cases
    e.hook(0x622C40, lambda e_obj: 1, 3)
    e.hook(0x57C6C0, lambda e_obj: 0, 3)
    e.hook(0x621E40, lambda e_obj: 0, 3)
    e.hook(0x623C20, lambda e_obj: 0, 0)
    e.hook(0x5C0C30, lambda e_obj: 0, 3)
    e.hook(0x5D10C0, lambda e_obj: 0, 2)
    e.hook(0x57CEE0, lambda e_obj: 0, 2)
    e.hook(0x57C9F0, lambda e_obj: 0, 1)

    hit_node = e.alloc(0x90)
    e.w32(attacker + 0x00, 0)
    e.w32(attacker + 0x04, 3)  # Paladin
    e.w32(attacker + 0x0C, 1)  # attacker guid = 1
    e.w32(attacker + 0x10, 7)  # mode 7 (attacking, != 0 dead)
    e.w32(defender + 0x00, 0)
    e.w32(defender + 0x04, 3)
    e.w32(defender + 0x0C, 2)  # defender guid = 2

    for s in range(13):
        e.w32(att_slots + s * 4, 0)
    e.w32(att_slots + 4 * 4, w_right)
    e.w32(att_inv + 0x1C, e.r32(w_right + 0x0C))

    # Fix the torso item once. Suite 3 leaves item_units[3] with a stale cid and flags.
    e.w32(item_units[3] + 0x04, armo_cids[0])
    e.w32(item_datas[3] + 0x18, 0)

    # 1. Zero-seed blocked-versus-landed check (2 distinct result_flags inputs: 0x10 vs 0x01)
    for blocked in (True, False):
        # When blocked, FUN_0057ec10 sets result_flags = (0x01 & 0xfffe) | 0x10 = 0x10
        result_flags = 0x10 if blocked else 0x01
        e.mu.mem_write(hit_node, b"\x00" * 0x90)
        e.w32(hit_node + 0x04, 0)  # attacker type = 0
        e.w32(hit_node + 0x08, 1)  # attacker guid = 1
        e.w32(hit_node + 0x0C, 0)  # defender type = 0
        e.w32(hit_node + 0x10, 2)  # defender guid = 2
        e.w32(hit_node + 0x18, result_flags)  # DamageRecord (+0x14) + 0x04 = result_flags
        e.w32(hit_node + 0x84, 0)  # next = NULL
        e.w32(attacker + 0xAC, hit_node)

        # Seed attacker and defender to 0 so step_rng(0, 0) -> 0 (% 100 == 0 < chance, guaranteed wear if called!)
        e.w32(attacker + 0x20, 0)
        e.w32(attacker + 0x24, 0)
        e.w32(w_right + 0x04, hand_weap_cids[0])
        e.w32(d_right + 0x18, 0)
        unit_stats[w_right] = {STAT_DUR: 20, STAT_MAXDUR: 20, STAT_QTY: 0, STAT_INDEST: 0}

        e.w32(defender + 0x20, 0)
        e.w32(defender + 0x24, 0)
        u_torso = item_units[3]
        unit_stats[u_torso] = {STAT_DUR: 50, STAT_MAXDUR: 50, STAT_QTY: 0, STAT_INDEST: 0}
        for s in range(13):
            e.w32(def_slots + s * 4, 0)
        e.w32(def_slots + 3 * 4, u_torso)

        e.call(0x57D4F0, defender, 0, ecx=game_ptr, edx=attacker)

        got_w_dur = unit_stats[w_right][STAT_DUR]
        got_a_dur = unit_stats[u_torso][STAT_DUR]
        exp_w_dur = 20 if blocked else 19
        exp_a_dur = 50 if blocked else 49

        total_cases += 1
        if got_w_dur != exp_w_dur or got_a_dur != exp_a_dur:
            mismatches += 1
            if mismatches <= 5:
                print(f"[S4-hit mismatch] blocked={blocked} got={(got_w_dur, got_a_dur)} exp={(exp_w_dur, exp_a_dur)}")

    # 2. Seeded landed-hit check (5,000 cases, result_flags = 0x01) deriving expected wear from Suite 2 RNG oracle
    for i in range(5000):
        e.mu.mem_write(hit_node, b"\x00" * 0x90)
        e.w32(hit_node + 0x04, 0)  # attacker type = 0
        e.w32(hit_node + 0x08, 1)  # attacker guid = 1
        e.w32(hit_node + 0x0C, 0)  # defender type = 0
        e.w32(hit_node + 0x10, 2)  # defender guid = 2
        e.w32(hit_node + 0x18, 0x01)  # landed hit
        e.w32(hit_node + 0x84, 0)
        e.w32(attacker + 0xAC, hit_node)

        att_seed_lo = rnd.randint(1, 0xFFFFFFFF)
        att_seed_hi = rnd.randint(1, 0xFFFFFFFF)
        def_seed_lo = rnd.randint(1, 0xFFFFFFFF)
        def_seed_hi = rnd.randint(1, 0xFFFFFFFF)

        init_w_dur = rnd.randint(5, 50)
        init_a_dur = rnd.randint(5, 50)

        e.w32(attacker + 0x20, att_seed_lo)
        e.w32(attacker + 0x24, att_seed_hi)
        e.w32(w_right + 0x04, hand_weap_cids[0])
        e.w32(d_right + 0x18, 0)
        unit_stats[w_right] = {STAT_DUR: init_w_dur, STAT_MAXDUR: 50, STAT_QTY: 0, STAT_INDEST: 0}

        e.w32(defender + 0x20, def_seed_lo)
        e.w32(defender + 0x24, def_seed_hi)
        u_torso = item_units[3]
        unit_stats[u_torso] = {STAT_DUR: init_a_dur, STAT_MAXDUR: 50, STAT_QTY: 0, STAT_INDEST: 0}
        for s in range(13):
            e.w32(def_slots + s * 4, 0)
        e.w32(def_slots + 3 * 4, u_torso)

        e.call(0x57D4F0, defender, 0, ecx=game_ptr, edx=attacker)

        # Oracle expected calculations (confirming return shape and draw order):
        # Attacker: 1 RNG draw with modulus 100; weapon wears if roll < 4
        roll_w, exp_att_lo, exp_att_hi = rand_range(att_seed_lo, att_seed_hi, 100)
        exp_w_dur = init_w_dur - 1 if roll_w < 4 else init_w_dur

        # Defender: 1st draw rand_range(7) for start_idx, 2nd draw rand_range(total_weight=5) for pick,
        # 3rd draw rand_range(100) for armor wear; armor wears if roll < 10
        start_idx, exp_def_lo, exp_def_hi = rand_range(def_seed_lo, def_seed_hi, 7)
        pick, exp_def_lo, exp_def_hi = rand_range(exp_def_lo, exp_def_hi, 5)
        roll_a, exp_def_lo, exp_def_hi = rand_range(exp_def_lo, exp_def_hi, 100)
        exp_a_dur = init_a_dur - 1 if roll_a < 10 else init_a_dur

        got_w_dur = unit_stats[w_right][STAT_DUR]
        got_a_dur = unit_stats[u_torso][STAT_DUR]
        got_att_lo = e.r32(attacker + 0x20)
        got_att_hi = e.r32(attacker + 0x24)
        got_def_lo = e.r32(defender + 0x20)
        got_def_hi = e.r32(defender + 0x24)

        total_cases += 1
        if (got_w_dur != exp_w_dur or got_a_dur != exp_a_dur or
                got_att_lo != exp_att_lo or got_att_hi != exp_att_hi or
                got_def_lo != exp_def_lo or got_def_hi != exp_def_hi):
            mismatches += 1
            if mismatches <= 5:
                print(f"[S4-seeded mismatch] case={i} got_w={got_w_dur} exp_w={exp_w_dur} "
                      f"got_a={got_a_dur} exp_a={exp_a_dur}")

    elapsed = time.time() - t0
    print(f"Suite 4 (FUN_00535bc0 weapsel + FUN_0057d4f0 block vs hit): 10,002 cases, mismatches={mismatches}")
    print(f"TOTAL: {total_cases} cases in {elapsed:.2f}s, mismatches={mismatches}")
    return 1 if mismatches > 0 else 0


if __name__ == "__main__":
    sys.exit(main())
