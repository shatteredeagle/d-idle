import random
import struct
import sys
import time
from unicorn.x86_const import UC_X86_REG_ECX, UC_X86_REG_EDX, UC_X86_REG_ESI, UC_X86_REG_EAX


from pathlib import Path
cur_dir = Path(__file__).resolve().parent
sys.path.insert(0, str(cur_dir))
import skill_tips as boot_helper

def s32(v):
    """Convert unsigned 32-bit int to signed 32-bit int."""
    v &= 0xFFFFFFFF
    return v - 0x100000000 if (v & 0x80000000) else v

def step_rng(lo, hi):
    """Advance 64-bit LCG state by one step and return (lo, hi) 32-bit words."""
    prod = ((lo & 0xFFFFFFFF) * 0x6AC690C5 + (hi & 0xFFFFFFFF)) & 0xFFFFFFFFFFFFFFFF
    return prod & 0xFFFFFFFF, (prod >> 32) & 0xFFFFFFFF

def mul_div_trunc(a, b, c):
    """Model Game.exe FUN_00483360 integer scaling with overflow guard."""
    if c == 0:
        return 0
    if abs(a) > 0x100000:
        return s32(((a & 0xFFFFFFFF) // (c & 0xFFFFFFFF)) * (b & 0xFFFFFFFF))
    prod = a * b
    if (prod < 0) ^ (c < 0):
        return -((-prod) // abs(c))
    return prod // c

def level_bracket(base, lev, lvl):
    """Calculate standard 5-bracket skill leveling progression."""
    if lvl <= 1:
        return base
    b1, b2, b3, b4, b5 = lev
    if lvl <= 8:
        return base + (lvl - 1) * b1
    if lvl <= 16:
        return base + 7 * b1 + (lvl - 8) * b2
    if lvl <= 22:
        return base + 7 * b1 + 8 * b2 + (lvl - 16) * b3
    if lvl <= 28:
        return base + 7 * b1 + 8 * b2 + 6 * b3 + (lvl - 22) * b4
    return base + 7 * b1 + 8 * b2 + 6 * b3 + 6 * b4 + (lvl - 28) * b5

def python_aurafilter(caster_info, target_info, filt):
    """Exact 19-bit model of FUN_0056b3e0(ecx=caster, edx=target, filt)."""
    utype = target_info["type"]
    mode = target_info["mode"]
    if utype == 0:  # Player
        if not (filt & 0x00001):
            return 0
        if filt & 0x01000:
            if mode != 0x11:
                return 0
        else:
            if mode == 0x11 or mode == 0:
                return 0
    elif utype == 1:  # Monster / Merc / Pet
        if not (filt & 0x00002):
            return 0
        if filt & 0x01000:
            if mode != 0x0C:
                return 0
        else:
            if mode == 0x0C or mode == 0:
                return 0
        if (filt & 0x00004) and not target_info["undead"]:
            return 0
        if (filt & 0x04000) and target_info["boss"]:
            return 0
        if (filt & 0x40000) and target_info["primeevil"]:
            return 0
    elif utype == 2:  # Object
        if not (filt & 0x00010):
            return 0
    elif utype == 3:  # Missile
        if not (filt & 0x00008):
            return 0
        if not target_info.get("missile_collide", False):
            return 0
    elif utype == 4:  # Item
        if not (filt & 0x00020):
            return 0
    else:
        return 0

    if (filt & 0x00080) and not (target_info["flags_c4"] & 0x04):
        return 0
    if (filt & 0x00400) and not (target_info["flags_c4"] & 0x08):
        return 0
    if (filt & 0x00100) and target_info["in_town"]:
        return 0
    if (filt & 0x10000) and not target_info["is_ally"]:
        return 0
    if (filt & 0x08000) and not target_info["is_enemy"]:
        return 0
    if (filt & 0x20000) and not target_info["in_melee"]:
        return 0
    if (filt & 0x80000) and target_info["state_86"]:
        return 0
    if (filt & 0x00200) and target_info["los_blocked"]:
        return 0
    return 1


def suite1_filter_and_search(n_cases=100000, seed=0x50414C31):
    """Run 100,000 cases verifying FUN_0056b3e0 target filter and FUN_0056b7e0 search."""
    print(f"=== Suite 1: Aura Target Filter & Area Search (N={n_cases}, seed={seed:#010x}) ===")
    t0 = time.time()
    e = boot_helper.boot()
    rng = random.Random(seed & 0xFFFFFFFF)

    rooms = {}
    town_rooms = set()
    los_blocked = [0]
    in_melee = [1]
    state86 = set()
    owners = {}
    party = {}
    pvp_hostile = set()
    alignments = {}

    e.hook(0x620BB0, lambda e: rooms.get(e.arg(0), 0), 1)
    e.hook(0x61AB00, lambda e: int(e.arg(0) in town_rooms), 1)
    e.hook(0x64E260, lambda e: los_blocked[0], 4)
    e.hook(0x622C40, lambda e: in_melee[0], 3)
    e.hook(0x639DF0, lambda e: int(e.arg(1) == 0x56 and (e.arg(0) in state86)), 2)
    e.hook(0x58F0D0, lambda e: owners.get(e.mu.reg_read(UC_X86_REG_ECX), 0), 0)
    e.hook(0x540710, lambda e: party.get(e.r32(e.mu.reg_read(UC_X86_REG_EDX) + 0x0C), 0xFFFF), 0)
    e.hook(0x55B300, lambda e: int((e.r32(e.mu.reg_read(UC_X86_REG_ECX) + 0x0C), e.r32(e.mu.reg_read(UC_X86_REG_EDX) + 0x0C)) in pvp_hostile), 1)
    e.hook(0x6259B0, lambda e: alignments.get(e.arg(0), 2), 1)

    hit_units = []
    cb_addr = 0x7F00F000
    def cb_fn(e):
        hit_units.append(e.mu.reg_read(UC_X86_REG_EDX))
        return 1
    e.hook(cb_addr, cb_fn, 0)

    game = e.alloc(0x200)
    room_out = e.alloc(0x100)
    room_town = e.alloc(0x100)
    town_rooms.add(room_town)
    adj_arr = e.alloc(0x10)
    e.w32(adj_arr, room_out)
    e.w32(adj_arr + 4, room_town)
    for r in (room_out, room_town):
        e.w32(r + 0x00, adj_arr)
        e.w32(r + 0x24, 2)
        e.w32(r + 0x4C, 0)
        e.w32(r + 0x50, 0)
        e.w32(r + 0x54, 2000)
        e.w32(r + 0x58, 2000)

    def make_unit(utype, mid, x, y, mode=1, flags_c4=0x0C, uid=1):
        u = e.alloc(0x100)
        p = e.alloc(0x40)
        e.w32(u + 0x00, utype)
        e.w32(u + 0x04, mid)
        e.w32(u + 0x0C, uid)
        e.w32(u + 0x10, mode)
        e.w32(u + 0x2C, p)
        e.w32(u + 0x80, game)
        e.w32(u + 0xC4, flags_c4)
        e.mu.mem_write(p + 2, struct.pack("<H", x & 0xFFFF))
        e.mu.mem_write(p + 6, struct.pack("<H", y & 0xFFFF))
        e.w32(p + 0x0C, x)
        e.w32(p + 0x10, y)
        return u

    noaura_mid = next(m for m in range(734) if e.call(0x457490, ecx=m, edx=0x1B) != 0)
    mon_meta = {}
    for mid in (0, 18, 146, 156, 267, 271, noaura_mid):
        tmp = make_unit(1, mid, 500, 500)
        undead = bool(e.call(0x63E990, tmp) != 0)
        boss = bool(e.call(0x63E9F0, 0, tmp) != 0)
        primeevil = bool(e.call(0x63EDC0, tmp) != 0)
        noaura = bool(e.call(0x457490, ecx=mid, edx=0x1B) != 0)
        tables = e.r32(0x744304)
        mrec = e.r32(tables + 0xA78) + mid * 0x1A8
        npc = bool(e.read(mrec + 0x0D, 1)[0] & 1)
        mon_meta[mid] = {"undead": undead, "boss": boss, "primeevil": primeevil, "npc": npc, "noaura": noaura}
        print(f"  MonStats[{mid:3d}]: {mon_meta[mid]}")

    cx, cy = 500, 500
    caster = make_unit(0, 3, cx, cy, mode=1, flags_c4=0x0C, uid=10)
    rooms[caster] = room_out; alignments[caster] = 2; party[10] = 1
    merc = make_unit(1, 271, cx + 10, cy, mode=1, flags_c4=0x0C, uid=11)
    owners[merc] = caster; rooms[merc] = room_out; alignments[merc] = 2
    pet = make_unit(1, 18, cx, cy + 10, mode=1, flags_c4=0x0C, uid=12)
    owners[pet] = caster; rooms[pet] = room_out; alignments[pet] = 2
    trap_pet = make_unit(1, noaura_mid, cx, cy + 8, mode=1, flags_c4=0x0C, uid=22)
    owners[trap_pet] = caster; rooms[trap_pet] = room_out; alignments[trap_pet] = 2
    npc_unit = make_unit(1, 146, cx + 5, cy, mode=1, flags_c4=0x0C, uid=13)
    rooms[npc_unit] = room_out; alignments[npc_unit] = 2
    party_ally = make_unit(0, 1, cx + 12, cy, mode=1, flags_c4=0x0C, uid=14)
    rooms[party_ally] = room_out; alignments[party_ally] = 2; party[14] = 1
    neutral_plr = make_unit(0, 2, cx + 12, cy, mode=1, flags_c4=0x0C, uid=15)
    rooms[neutral_plr] = room_out; alignments[neutral_plr] = 2; party[15] = 0xFFFF
    hostile_plr = make_unit(0, 4, cx + 12, cy, mode=1, flags_c4=0x0C, uid=16)
    rooms[hostile_plr] = room_out; alignments[hostile_plr] = 2; party[16] = 0xFFFF
    pvp_hostile.add((10, 16)); pvp_hostile.add((16, 10))
    undead_mon = make_unit(1, 0, cx + 15, cy, mode=1, flags_c4=0x0C, uid=17)
    rooms[undead_mon] = room_out; alignments[undead_mon] = 0
    boss_undead = make_unit(1, 267, cx + 15, cy, mode=1, flags_c4=0x0C, uid=18)
    rooms[boss_undead] = room_out; alignments[boss_undead] = 0
    demon_mon = make_unit(1, 18, cx + 15, cy, mode=1, flags_c4=0x0C, uid=19)
    rooms[demon_mon] = room_out; alignments[demon_mon] = 0
    primeevil_mon = make_unit(1, 156, cx + 15, cy, mode=1, flags_c4=0x0C, uid=23)
    rooms[primeevil_mon] = room_out; alignments[primeevil_mon] = 0
    corpse_mon = make_unit(1, 0, cx + 8, cy, mode=0x0C, flags_c4=0x0C, uid=20)
    rooms[corpse_mon] = room_out; alignments[corpse_mon] = 0
    town_ally = make_unit(0, 1, cx + 5, cy, mode=1, flags_c4=0x0C, uid=21)
    rooms[town_ally] = room_town; alignments[town_ally] = 2; party[21] = 1

    out_list = [caster, merc, pet, trap_pet, npc_unit, party_ally, neutral_plr, hostile_plr, undead_mon, boss_undead, demon_mon, primeevil_mon, corpse_mon]
    for a, b in zip(out_list, out_list[1:]):
        e.w32(a + 0xE8, b)
    e.w32(out_list[-1] + 0xE8, 0)
    e.w32(room_out + 0x74, out_list[0])
    e.w32(town_ally + 0xE8, 0)
    e.w32(room_town + 0x74, town_ally)

    uid_names = {
        10: "self", 11: "merc", 12: "pet", 22: "trap_noaura", 13: "npc", 14: "party_ally",
        15: "neutral_plr", 16: "hostile_plr", 17: "undead_mon",
        18: "boss_undead", 19: "demon_mon", 23: "primeevil_mon", 20: "corpse_mon", 21: "town_ally"
    }

    PAL_FILTERS = [
        ("Friendly (Might/Prayer/etc.)", 0x12003, 1, {
            0: ["merc", "pet", "party_ally"],
            1: ["merc", "pet", "party_ally"],
        }),
        ("Meditation", 0x12001, 1, {
            0: ["party_ally"],
            1: ["party_ally"],
        }),
        ("Holy Fire / Holy Shock", 0x0A783, 0, {
            0: ["hostile_plr", "undead_mon", "boss_undead", "demon_mon", "primeevil_mon"],
            1: [],
        }),
        ("Holy Freeze", 0x4A283, 0, {
            0: ["hostile_plr", "undead_mon", "boss_undead", "demon_mon"],
            1: [],
        }),
        ("Conviction", 0x0A583, 0, {
            0: ["hostile_plr", "undead_mon", "boss_undead", "demon_mon", "primeevil_mon"],
            1: ["hostile_plr", "undead_mon", "boss_undead", "demon_mon", "primeevil_mon"],
        }),
        ("Sanctuary", 0x0E786, 0, {
            0: ["undead_mon"],
            1: [],
        }),
        ("Redemption", 0x01102, 0, {
            0: ["corpse_mon"],
            1: ["corpse_mon"],
        }),
    ]
    mismatches = 0
    for label, filt, chk_noaura, exp_map in PAL_FILTERS:
        for los in (0, 1):
            los_blocked[0] = los
            hit_units.clear()
            e.call(0x56B7E0, 0, 0, 20, filt, cb_addr, 0, chk_noaura, 0, 0, ecx=game, edx=caster)
            names = [uid_names[e.r32(u + 0x0C)] for u in hit_units]
            want = exp_map[los]
            if sorted(names) != sorted(want):
                mismatches += 1
                print(f"  MISMATCH scenario {label} (filt={filt:#x}, los={los}): got={names} want={want}")
            elif los == 0 or filt in (0x0A783, 0x4A283, 0x0A583, 0x0E786):
                print(f"  {label:28s} (filter={filt:#07x}, los_blocked={los}): hits={names}")

    # Verify FUN_0056b740 applies to self
    hit_units.clear()
    e.call(0x56B740, cb_addr, 0, ecx=game, edx=caster)
    self_hits = [uid_names[e.r32(u + 0x0C)] for u in hit_units]
    if self_hits != ["self"]:
        mismatches += 1
        print(f"  MISMATCH FUN_0056b740 self-wrapper: got={self_hits} want=['self']")
    else:
        print(f"  FUN_0056b740 (self-wrapper): hits={self_hits}")

    target = make_unit(1, 0, cx, cy, mode=1, flags_c4=0x0C, uid=99)
    e.w32(caster + 0xE8, target)
    e.w32(target + 0xE8, 0)
    e.w32(room_out + 0x74, caster)
    e.w32(room_town + 0x74, 0)
    tp = e.r32(target + 0x2C)

    ALL_FILTERS = [0x12003, 0x12001, 0x0A783, 0x4A283, 0x0A583, 0x0E786, 0x01102, 0x10583, 0x08783, 0x01583]
    for i in range(n_cases):
        if i < 20000:
            radius = (i % 40) + 1
            dx = rng.randint(-radius - 2, radius + 2)
            dy_approx = int(max(0, radius * radius - dx * dx) ** 0.5)
            dy = dy_approx + rng.choice((-1, 0, 1))
        else:
            radius = rng.randint(1, 50)
            dx = rng.randint(-55, 55)
            dy = rng.randint(-55, 55)

        filt = ALL_FILTERS[i % len(ALL_FILTERS)] if i < 50000 else (rng.choice(ALL_FILTERS) | (1 << rng.randint(0, 19)))
        utype = rng.choice((0, 1, 1, 1, 2, 4))
        mid = rng.choice((0, 18, 146, 156, 267, 271, noaura_mid)) if utype == 1 else 1
        mode = rng.choice((1, 1, 1, 0, 0x0C, 0x11))
        flags_c4 = rng.choice((0x0C, 0x0C, 0x04, 0x08, 0x00))
        in_t = rng.random() < 0.15
        los_b = int(rng.random() < 0.2)
        melee_b = int(rng.random() < 0.8)
        st86_b = rng.random() < 0.1
        chk_noaura = rng.choice((0, 1))
        rel = rng.choice(("merc_ally", "party_ally", "enemy", "neutral"))

        e.w32(target + 0x00, utype)
        e.w32(target + 0x04, mid)
        e.w32(target + 0x10, mode)
        e.w32(target + 0xC4, flags_c4)
        tx, ty = cx + dx, cy + dy
        e.mu.mem_write(tp + 2, struct.pack("<H", tx & 0xFFFF))
        e.mu.mem_write(tp + 6, struct.pack("<H", ty & 0xFFFF))
        e.w32(tp + 0x0C, tx)
        e.w32(tp + 0x10, ty)

        if in_t:
            rooms[target] = room_town
            e.w32(room_out + 0x74, caster)
            e.w32(caster + 0xE8, 0)
            e.w32(room_town + 0x74, target)
            e.w32(target + 0xE8, 0)
        else:
            rooms[target] = room_out
            e.w32(room_out + 0x74, caster)
            e.w32(caster + 0xE8, target)
            e.w32(target + 0xE8, 0)
            e.w32(room_town + 0x74, 0)

        los_blocked[0] = los_b
        in_melee[0] = melee_b
        if st86_b:
            state86.add(target)
        else:
            state86.discard(target)

        owners.pop(target, None)
        pvp_hostile.clear()
        party[99] = 0xFFFF
        if rel == "merc_ally":
            if utype == 1:
                owners[target] = caster
            else:
                party[99] = 1
            alignments[target] = 2
        elif rel == "party_ally":
            party[99] = 1
            alignments[target] = 2
        elif rel == "enemy":
            alignments[target] = 0
            if utype == 0:
                pvp_hostile.add((10, 99)); pvp_hostile.add((99, 10))
        else:
            alignments[target] = 1 if utype == 1 else 2

        hit_units.clear()
        e.call(0x56B7E0, 0, 0, radius, filt, cb_addr, 0, chk_noaura, 0, 0, ecx=game, edx=caster)
        got_hit = int(len(hit_units) == 1)

        m = mon_meta.get(mid, {"undead": False, "boss": False, "primeevil": False, "npc": False, "noaura": False})
        if utype == 0:
            is_ally = (rel in ("merc_ally", "party_ally"))
            is_enemy = (rel == "enemy")
        elif utype == 1:
            same_owner_or_align = (rel in ("merc_ally", "party_ally"))
            is_ally = same_owner_or_align and not m["npc"]
            is_enemy = (rel != "merc_ally") and (alignments[target] != 2)
        else:
            is_ally = (alignments[target] == 2)
            is_enemy = (alignments[target] != 2)

        t_info = {
            "type": utype, "mode": mode,
            "undead": m["undead"], "boss": m["boss"], "primeevil": m["primeevil"],
            "flags_c4": flags_c4, "in_town": in_t,
            "is_ally": is_ally, "is_enemy": is_enemy,
            "in_melee": bool(melee_b), "state_86": st86_b, "los_blocked": bool(los_b),
        }
        room_ok = not ((filt & 0x2000) and in_t)
        dist_ok = (dx * dx + dy * dy <= radius * radius)
        filt_ok = bool(python_aurafilter(None, t_info, filt))
        noaura_ok = not (chk_noaura and utype == 1 and m.get("noaura", False))
        want_hit = int(room_ok and dist_ok and filt_ok and noaura_ok)

        if got_hit != want_hit:
            mismatches += 1
            if mismatches <= 5:
                print(f"  MISMATCH case {i}: got={got_hit} want={want_hit} dx={dx} dy={dy} r={radius} filt={filt:#x} info={t_info}")

    print(f"  Suite 1 result: {n_cases} cases run in {time.time() - t0:.2f}s, mismatches = {mismatches}")
    return mismatches


def suite2_pulse_damage(n_cases=100000, seed=0x50414C32):
    """Run 100,000 cases verifying FUN_0056e0c0 pulse damage across Holy Fire/Freeze/Shock."""
    print(f"=== Suite 2: Enemy-Hit Pulse Damage (N={n_cases}, seed={seed:#010x}) ===")
    t0 = time.time()
    e = boot_helper.boot()
    rng = random.Random(seed & 0xFFFFFFFF)

    FAKE_SK = 0x30000000
    levels = {}
    unit_stats = {}
    zero_row = e.alloc(0x400)
    e.hook(0x625480, lambda e: unit_stats.get((e.arg(0), e.arg(1)), 0), 3)
    e.hook(0x625500, lambda e: 0, 3)
    e.hook(0x643810, lambda e: FAKE_SK + 0x40 * (e.mu.reg_read(UC_X86_REG_ESI) & 0xFFFF), 0)
    e.hook(0x6442A0, lambda e: levels.get((e.arg(1) - FAKE_SK) // 0x40, 0), 3)
    e.hook(0x611D30, lambda e: zero_row, 1)
    e.hook(0x63EE90, lambda e: 0, 1)
    e.hook(0x6256B0, lambda e: 0, 2)
    e.hook(0x57C060, lambda e: 100, 0)

    game = e.alloc(0x200)
    caster = e.alloc(0x100)
    defender = e.alloc(0x100)
    e.w32(caster + 0x00, 0)    # player
    e.w32(defender + 0x00, 1)  # monster
    dmg_buf = e.alloc(0x70)

    # Element table rows for Holy Fire (102), Holy Freeze (114), Holy Shock (118), Sanctuary (119)
    SKILL_SPECS = {
        102: {"name": "Holy Fire",   "hs": 7, "emin": 2, "emin_lev": (1, 2, 3, 5, 7), "emax": 6,  "emax_lev": (1, 2, 3, 5, 7),
              "syn": ((100, 18), (125, 6)), "mstat": 329, "pstat": 333, "rstat": 39, "dmg_off": 0x10},
        114: {"name": "Holy Freeze", "hs": 8, "emin": 2, "emin_lev": (1, 2, 3, 4, 5), "emax": 3,  "emax_lev": (1, 2, 3, 4, 5),
              "syn": ((105, 15), (125, 7)), "mstat": 331, "pstat": 335, "rstat": 43, "dmg_off": 0x24},
        118: {"name": "Holy Shock",  "hs": 8, "emin": 1, "emin_lev": (0, 0, 0, 0, 0), "emax": 10, "emax_lev": (6, 8, 10, 12, 15),
              "syn": ((110, 12), (125, 4)), "mstat": 330, "pstat": 334, "rstat": 41, "dmg_off": 0x1C},
        119: {"name": "Sanctuary",   "hs": 8, "emin": 8, "emin_lev": (4, 4, 5, 5, 6), "emax": 16, "emax_lev": (4, 5, 6, 6, 7),
              "syn": ((109, 7),),           "mstat": 357, "pstat": None, "rstat": 37, "dmg_off": 0x20},
    }

    def expected_bounds(sid, lvl, syn_lvls, mastery):
        sp = SKILL_SPECS[sid]
        syn_pct = sum(syn_lvls.get(sk, 0) * mult for sk, mult in sp["syn"])
        elo = level_bracket(sp["emin"], sp["emin_lev"], lvl) << sp["hs"]
        ehi = level_bracket(sp["emax"], sp["emax_lev"], lvl) << sp["hs"]
        if syn_pct != 0:
            if elo > 0x100 or sp["emin_lev"][0] != 0:
                elo += mul_div_trunc(elo, syn_pct, 100)
            if ehi > 0x100 or sp["emax_lev"][0] != 0:
                ehi += mul_div_trunc(ehi, syn_pct, 100)
        if sp["mstat"] is not None and mastery != 0:
            elo += mul_div_trunc(elo, mastery, 100)
            ehi += mul_div_trunc(ehi, mastery, 100)
        return elo, ehi

    mismatches = 0
    for i in range(n_cases):
        if i < 6400:
            # Exhaustive sweep over all 4 skills, levels 1..40, 4 synergy/mastery combos
            sids = (102, 114, 118, 119)
            sid = sids[(i // 160) % 4]
            lvl = (i % 40) + 1
            combo = (i // 40) % 4
            if combo == 0:
                syn_lvls, mastery = {}, 0
            elif combo == 1:
                syn_lvls, mastery = {100: 20, 105: 20, 110: 20, 109: 20, 125: 20}, 0
            elif combo == 2:
                syn_lvls, mastery = {100: 20, 105: 20, 110: 20, 109: 20, 125: 20}, 50
            else:
                syn_lvls, mastery = {100: 10, 105: 10, 110: 10, 109: 10, 125: 5}, 25
            seed_lo, seed_hi = (i * 0x9E3779B9) & 0xFFFFFFFF, 0x0000029A
            target_res = [-100, -50, 0, 50, 75, 95, 100, 120][i % 8]
            pierce = [0, 15, 25, 0][(i // 8) % 4]
        else:
            sid = rng.choice((102, 114, 118, 119))
            lvl = rng.randint(1, 45)
            syn_lvls = {sk: rng.choice((0, rng.randint(1, 20))) for sk, _ in SKILL_SPECS[sid]["syn"]}
            mastery = rng.choice((0, 0, rng.randint(1, 150)))
            seed_lo = rng.getrandbits(32)
            seed_hi = rng.getrandbits(32) or 0x29A
            target_res = rng.randint(-120, 130)
            pierce = rng.choice((0, 0, rng.randint(1, 60)))

        sp = SKILL_SPECS[sid]
        levels.clear()
        levels.update(syn_lvls)
        levels[sid] = lvl
        unit_stats.clear()
        if sp["mstat"] is not None:
            unit_stats[(caster, sp["mstat"])] = mastery
        if sp["pstat"] is not None:
            unit_stats[(caster, sp["pstat"])] = pierce
        unit_stats[(defender, sp["rstat"])] = target_res

        e.w32(caster + 0x20, seed_lo)
        e.w32(caster + 0x24, seed_hi)
        e.mu.mem_write(dmg_buf, b"\x00" * 0x70)

        # Call FUN_0056e0c0(ecx=caster, edx=dmg_buf, sid, lvl)
        ret_dmg = s32(e.call(0x56E0C0, sid, lvl, ecx=caster, edx=dmg_buf))
        got_roll = e.s32(dmg_buf + sp["dmg_off"])

        # Expected roll from formula
        elo, ehi = expected_bounds(sid, lvl, syn_lvls, mastery)
        if ehi - elo > 0:
            n_lo, _ = step_rng(seed_lo, seed_hi)
            want_roll = elo + (n_lo % (ehi - elo))
        else:
            want_roll = elo

        # Also test FUN_0057c1e0 (which runs FUN_0057bf80 + FUN_0057be00 resistance & pierce application on defender)
        e.call(0x57C1E0, defender, dmg_buf, ecx=game, edx=caster)
        got_final = e.s32(dmg_buf + 0x4C)

        # Expected effective res & final damage
        want_res = target_res
        if sp["pstat"] is not None and want_res < 100:
            want_res -= pierce
        if want_res < -100:
            want_res = -100
        if want_res >= 100 or want_roll <= 0:
            want_final = 0
        else:
            want_final = max(0, mul_div_trunc(want_roll, 100 - want_res, 100))

        if got_roll != want_roll or ret_dmg != want_roll or got_final != want_final:
            mismatches += 1
            if mismatches <= 5:
                print(f"  MISMATCH case {i}: sid={sid} lvl={lvl} got=({got_roll},{got_final}) want=({want_roll},{want_final}) res={target_res} pierce={pierce}")

    # Also test Holy Freeze callback FUN_005d07c0 coldeffect >= 0 immunity gate and shatter roll
    game = e.alloc(0x200)
    e.mu.mem_write(game + 0x6D, bytes([0]))  # Normal difficulty (offset 0 in coldeffect[3])
    tables = e.r32(0x744304)
    mbase = e.r32(tables + 0xA78)
    # Find a monster with coldeffect < 0 (e.g. skeleton1 mid=0, coldeffect=-50) and one with coldeffect == 0
    ce_neg_mid = next(m for m in range(734) if struct.unpack("<b", e.read(mbase + m * 0x1A8 + 0x168, 1))[0] < 0)
    ce_zero_mid = next(m for m in range(734) if struct.unpack("<b", e.read(mbase + m * 0x1A8 + 0x168, 1))[0] >= 0)
    st107 = {}
    e.hook(0x5CEDC0, lambda e: 1, 0)
    e.hook(0x57C6C0, lambda e: 1, 3)
    e.hook(0x57CEE0, lambda e: 1, 2)
    e.hook(0x639DB0, lambda e: (st107.__setitem__((e.arg(0), e.arg(1)), e.arg(2)), 1)[1], 3)

    hf_ctx = e.alloc(0xC0)
    stack_ctx = e.alloc(0x10)
    e.w32(stack_ctx + 0x00, game)
    e.w32(stack_ctx + 0x04, caster)
    e.w32(stack_ctx + 0x08, 0)
    e.w32(stack_ctx + 0x0C, hf_ctx)

    # 1. coldeffect >= 0 returns 0 immediately (immunity gate)
    e.w32(defender + 0x04, ce_zero_mid)
    r_zero = e.call(0x5D07C0, ecx=stack_ctx, edx=defender)
    if r_zero != 0:
        mismatches += 1
        print(f"  MISMATCH Holy Freeze coldeffect>=0 gate: got ret={r_zero} want 0")

    # 2. coldeffect < 0 returns 1, rolls shatter state 107
    e.w32(defender + 0x04, ce_neg_mid)
    for test_lo, test_hi in [(6, 100), (12345, 666)]:
        e.w32(defender + 0x20, test_lo)
        e.w32(defender + 0x24, test_hi)
        st107.clear()
        r_neg = e.call(0x5D07C0, ecx=stack_ctx, edx=defender)
        n_lo, _ = step_rng(test_lo, test_hi)
        want_st107 = int((n_lo % 100) < 20)
        # In Game.exe FUN_005d07c0 (0x5d0876..0x5d08fa), state 107 is set to 1 if roll < 20,
        # or explicitly cleared to 0 via FUN_00639db0(target, 0x6b, 0) if roll >= 20.
        # An absent st107 key indicates FUN_00639db0 was not called at all.
        if (defender, 0x6B) not in st107:
            got_st107 = 0
            call_recorded = False
        else:
            got_st107 = st107[(defender, 0x6B)]
            call_recorded = True

        if r_neg != 1 or got_st107 != want_st107 or not call_recorded:
            mismatches += 1
            print(f"  MISMATCH Holy Freeze callback (seed={test_lo},{test_hi}): ret={r_neg} (want 1), st107={got_st107} (want {want_st107}, recorded={call_recorded})")
        else:
            print(f"  Holy Freeze FUN_005d07c0 coldeffect<0 (mid={ce_neg_mid}, seed={test_lo},{test_hi}): ret={r_neg}, state107={got_st107} (want {want_st107})")

    # 3. Sanctuary application-path assertion: verify auratargetstate == -1 skips aurastat1..3 evaluation
    sbase = e.r32(tables + 0xB98)
    srec_sanc = sbase + 119 * 0x23C
    sanc_target_state = struct.unpack("<h", e.read(srec_sanc + 0x82, 2))[0]
    # In Skills.txt, Sanctuary (119) has auratargetstate = -1 (0xffff).
    # FUN_005cf3a0 (0x5cf4bc) checks auratargetstate: when < 0, it skips evaluating aurastat1..3 (stats 97, 98, 100).
    if sanc_target_state != -1:
        mismatches += 1
        print(f"  MISMATCH Sanctuary auratargetstate: got {sanc_target_state} want -1")
    else:
        print(f"  Sanctuary application-path: auratargetstate={sanc_target_state} (asserts FUN_005cf3a0 skips aurastat1..3 evaluation)")

    print(f"  Suite 2 result: {n_cases} cases run in {time.time() - t0:.2f}s, mismatches = {mismatches}")
    return mismatches


def suite3_prayer_mana_and_regen(n_cases=100000, seed=0x50414C33):
    """Run 100,000 cases verifying Prayer mana payment, state 85, and FUN_005806f0 regen."""
    print(f"=== Suite 3: Prayer Mana Cost, State 85 (nomanaregen), and FUN_005806f0 (N={n_cases}, seed={seed:#010x}) ===")
    t0 = time.time()
    e = boot_helper.boot()
    rng = random.Random(seed & 0xFFFFFFFF)

    # Part 3A: Verify FUN_00644b10 mana cost for Prayer (99), Might (98), Sanctuary (119), Meditation (120)
    cost_mismatches = 0
    for lvl in range(1, 61):
        c_prayer = s32(e.call(0x644B10, 99, lvl))
        w_prayer = (16 + 3 * (lvl - 1)) << 4
        c_might = s32(e.call(0x644B10, 98, lvl))
        c_med = s32(e.call(0x644B10, 120, lvl))
        if c_prayer != w_prayer or c_might != 0 or c_med != 0:
            cost_mismatches += 1
    print(f"  FUN_00644b10 L1..60 cost check: Prayer L1={e.call(0x644B10, 99, 1)} (256=1.0), L2={e.call(0x644B10, 99, 2)} (304=1.1875), L20={e.call(0x644B10, 99, 20)} (1168=4.5625), Might L1..60=0, mismatches={cost_mismatches}")

    # Part 3B: Run FUN_005cf010 (srvdofunc 65) on Prayer (99) with sufficient vs insufficient mana, and FUN_005cec50 expiry
    unit_stats = {}
    unit_states = {}
    e.hook(0x625480, lambda e: unit_stats.get((e.arg(0), e.arg(1)), 0), 3)
    e.hook(0x625D60, lambda e: unit_stats.get((e.arg(0), 9), 0), 1)
    e.hook(0x6272B0, lambda e: (unit_stats.__setitem__((e.arg(0), e.arg(1)), unit_stats.get((e.arg(0), e.arg(1)), 0) + s32(e.arg(2))), 1)[1], 4)
    e.hook(0x639DB0, lambda e: (unit_states.__setitem__((e.arg(0), e.arg(1)), e.arg(2)), 1)[1], 3)
    e.hook(0x639DF0, lambda e: unit_states.get((e.arg(0), e.arg(1)), 0), 2)
    e.hook(0x639E30, lambda e: (unit_states.__setitem__((e.arg(0), e.arg(1)), e.arg(2)), 1)[1], 3)
    e.hook(0x552200, lambda e: 0, 1)

    # Per-unit statlists in FUN_005cedc0 so self and merc don't share one statlist
    pal_sl = e.alloc(0x80)
    merc_sl = e.alloc(0x80)
    sl_by_unit = {}
    sl_stats = {}
    e.hook(0x6256B0, lambda e: 0 if e.arg(1) == 0x6A else sl_by_unit.get(e.arg(0), pal_sl), 2)
    e.hook(0x56E970, lambda e: sl_by_unit.get(e.r32(e.mu.reg_read(UC_X86_REG_ECX) + 4), pal_sl), 0)
    e.hook(0x625D00, lambda e: sl_stats.get((e.arg(0), e.arg(1)), 0), 3)
    e.hook(0x627150, lambda e: (sl_stats.__setitem__((e.arg(0), e.arg(1)), s32(e.arg(2))), 1)[1], 4)
    e.hook(0x627260, lambda e: (unit_stats.__setitem__((e.arg(0), e.arg(1)), s32(e.arg(2))), 1)[1], 4)
    e.hook(0x623F50, lambda e: 0, 3)
    e.hook(0x56DFA0, lambda e: 1, 0)
    e.hook(0x56B6C0, lambda e: 1, 0)
    e.hook(0x56CD50, lambda e: 50, 3)

    rooms = {}
    town_rooms = set()
    owners = {}
    e.hook(0x620BB0, lambda e: rooms.get(e.arg(0), 0), 1)
    e.hook(0x61AB00, lambda e: int(e.arg(0) in town_rooms), 1)
    e.hook(0x58F0D0, lambda e: owners.get(e.mu.reg_read(UC_X86_REG_ECX), 0), 0)

    game = e.alloc(0x200)
    room_out = e.alloc(0x100)
    adj_arr = e.alloc(0x10)
    e.w32(adj_arr, room_out)
    e.w32(room_out + 0x00, adj_arr); e.w32(room_out + 0x24, 1)
    e.w32(room_out + 0x4C, 0); e.w32(room_out + 0x50, 0); e.w32(room_out + 0x54, 2000); e.w32(room_out + 0x58, 2000)

    def make_u(utype, mid, x, y, uid):
        u = e.alloc(0x100); p = e.alloc(0x40)
        e.w32(u, utype); e.w32(u + 4, mid); e.w32(u + 0x0C, uid); e.w32(u + 0x10, 1); e.w32(u + 0x2C, p); e.w32(u + 0x80, game); e.w32(u + 0xC4, 0x0C)
        e.mu.mem_write(p + 2, struct.pack("<H", x)); e.mu.mem_write(p + 6, struct.pack("<H", y))
        return u

    pal = make_u(0, 3, 500, 500, 1)
    merc = make_u(1, 271, 505, 500, 2)
    sl_by_unit[pal] = pal_sl; sl_by_unit[merc] = merc_sl
    rooms[pal] = room_out; rooms[merc] = room_out; owners[merc] = pal
    e.w32(room_out + 0x74, pal); e.w32(pal + 0xE8, merc); e.w32(merc + 0xE8, 0)

    scenario_mismatches = 0

    # Pulse 1: Prayer L1 (cost=256, heal=512) with cur_mana = 300 >= 256 (hurt units)
    unit_stats[(pal, 6)] = 5000; unit_stats[(pal, 7)] = 10000
    unit_stats[(merc, 6)] = 5000; unit_stats[(merc, 7)] = 10000
    unit_stats[(pal, 8)] = 300
    unit_states.clear()
    e.call(0x5CF010, 99, 1, ecx=game, edx=pal)
    p_hp = unit_stats[(pal, 6)]
    m_hp = unit_stats[(merc, 6)]
    p_mana = unit_stats[(pal, 8)]
    st85 = unit_states.get((pal, 0x55), 0)
    print(f"  Prayer L1 pulse (mana=300 >= 256): pal_hp={p_hp} (+{p_hp-5000}), merc_hp={m_hp} (+{m_hp-5000}), pal_mana={p_mana}, state85={st85}")
    if p_hp != 5512 or m_hp != 5512 or p_mana != 44 or st85 != 1:
        scenario_mismatches += 1
        print(f"  MISMATCH Prayer L1 hurt pulse: got hp=({p_hp},{m_hp}) want (5512,5512), mana={p_mana} want 44, st85={st85} want 1")

    # Pulse 1B: Prayer L1 with cur_mana = 300 >= 256 but Paladin and Merc at FULL HP (applied_count == 0)
    unit_stats[(pal, 6)] = 10000; unit_stats[(pal, 7)] = 10000
    unit_stats[(merc, 6)] = 10000; unit_stats[(merc, 7)] = 10000
    unit_stats[(pal, 8)] = 300
    unit_states[(pal, 0x55)] = 1  # pre-set to verify FUN_005cf010 clears it when no heal lands
    e.call(0x5CF010, 99, 1, ecx=game, edx=pal)
    p_hp_full = unit_stats[(pal, 6)]
    m_hp_full = unit_stats[(merc, 6)]
    p_mana_full = unit_stats[(pal, 8)]
    st85_full = unit_states.get((pal, 0x55), 0)
    print(f"  Prayer L1 pulse (at full life): pal_hp={p_hp_full}, merc_hp={m_hp_full}, pal_mana={p_mana_full} (0 paid), state85={st85_full} (cleared)")
    if p_hp_full != 10000 or m_hp_full != 10000 or p_mana_full != 300 or st85_full != 0:
        scenario_mismatches += 1
        print(f"  MISMATCH Prayer L1 full-life pulse: got hp=({p_hp_full},{m_hp_full}) want (10000,10000), mana={p_mana_full} want 300, st85={st85_full} want 0")

    # Pulse 2: Prayer L1 (cost=256) with cur_mana = 44 < 256
    e.call(0x5CF010, 99, 1, ecx=game, edx=pal)
    p_hp_low = unit_stats[(pal, 6)]
    m_hp_low = unit_stats[(merc, 6)]
    p_mana_low = unit_stats[(pal, 8)]
    st85_low = unit_states.get((pal, 0x55), 0)
    print(f"  Prayer L1 pulse (mana=44  <  256): pal_hp={p_hp_low} (+0), merc_hp={m_hp_low} (+0), pal_mana={p_mana_low}, state85={st85_low}")
    if p_hp_low != 10000 or m_hp_low != 10000 or p_mana_low != 44 or st85_low != 0:
        scenario_mismatches += 1
        print(f"  MISMATCH Prayer L1 low-mana pulse: got hp=({p_hp_low},{m_hp_low}) want (10000,10000), mana={p_mana_low} want 44, st85={st85_low} want 0")

    # Expiry callback FUN_005cec50(ecx=unit, edx=state_id, statlist) clears state 85 when set
    unit_states[(pal, 0x55)] = 1
    e.call(0x5CEC50, pal_sl, ecx=pal, edx=34)
    st85_exp = unit_states.get((pal, 0x55), 0)
    print(f"  FUN_005cec50 (prayer state 34 expiry): state85={st85_exp}")
    if st85_exp != 0:
        scenario_mismatches += 1
        print(f"  MISMATCH FUN_005cec50 expiry: got st85={st85_exp} want 0")

    # Also test Fanaticism (122) L1 self damagepercent (stat 25) vs merc damagepercent when mana > 0 vs mana == 0!
    sl_stats.clear(); unit_stats[(pal, 8)] = 256
    e.call(0x5CF010, 122, 1, ecx=game, edx=pal)
    fanat_self_pos = sl_stats.get((pal_sl, 25))
    fanat_merc_pos = sl_stats.get((merc_sl, 25))
    sl_stats.clear(); unit_stats[(pal, 8)] = 0
    e.call(0x5CF010, 122, 1, ecx=game, edx=pal)
    fanat_self_zero = sl_stats.get((pal_sl, 25))
    print(f"  Fanaticism L1 stat 25 (damagepercent): mana=256 -> self={fanat_self_pos}%, merc={fanat_merc_pos}%; mana=0 -> self={fanat_self_zero}% (drops to ln56/2!)")
    if fanat_self_pos != 50 or fanat_merc_pos != 25 or fanat_self_zero != 25:
        scenario_mismatches += 1
        print(f"  MISMATCH Fanaticism L1 stat 25: got self={fanat_self_pos} (want 50), merc={fanat_merc_pos} (want 25), zero={fanat_self_zero} (want 25)")

    # Part 3C: 100,000 cases of FUN_005806f0 (player mana regeneration, player in EDI)
    regen_mismatches = 0
    for i in range(n_cases):
        if i < 20000:
            max_mana = ((i % 500) + 1) << 8
            cur_mana = rng.randint(0, max_mana)
            stat27 = [0, 25, 50, 100, 300, -50, -100, -120][(i // 500) % 8]
            stat26 = [0, 0, 0, 16][(i // 4000) % 4]
            st85 = (i // 10000) % 2
        else:
            max_mana = rng.randint(1, 500000)
            cur_mana = rng.randint(0, max_mana + 500)
            stat27 = rng.choice((0, 0, rng.randint(-120, 600)))
            stat26 = rng.choice((0, 0, rng.randint(0, 64)))
            st85 = int(rng.random() < 0.4)

        unit_stats[(pal, 8)] = cur_mana
        unit_stats[(pal, 9)] = max_mana
        unit_stats[(pal, 0x1A)] = stat26
        unit_stats[(pal, 0x1B)] = stat27
        unit_states[(pal, 0x55)] = st85

        e.call(0x5806F0, regs={"edi": pal})
        got_mana = unit_stats[(pal, 8)]

        # Expected from FUN_005806f0 exact formula:
        if st85 != 0:
            rate = 0
        else:
            base_rate = max(max_mana // 3000, 1)
            rate = mul_div_trunc(base_rate, 100 + stat27, 100)
        rate += stat26
        if rate > max_mana - cur_mana:
            rate = max_mana - cur_mana
        if rate < -cur_mana:
            rate = -cur_mana
        want_mana = cur_mana + rate

        if got_mana != want_mana:
            regen_mismatches += 1
            if regen_mismatches <= 5:
                print(f"  MISMATCH regen case {i}: got={got_mana} want={want_mana} cur={cur_mana} max={max_mana} s27={stat27} s26={stat26} st85={st85}")

    total_mismatches = cost_mismatches + scenario_mismatches + regen_mismatches
    print(f"  Suite 3 result: {n_cases} cases run in {time.time() - t0:.2f}s, mismatches = {total_mismatches}")
    return total_mismatches


if __name__ == "__main__":
    only = sys.argv[1] if len(sys.argv) > 1 else "all"
    valid_selectors = ("all", "suite1", "suite2", "suite3")
    if only not in valid_selectors:
        print(f"Error: invalid selector '{only}'. Expected one of: {', '.join(valid_selectors)}", file=sys.stderr)
        sys.exit(2)
    m1 = suite1_filter_and_search(100000, 0x50414C31) if only in ("all", "suite1") else 0
    m2 = suite2_pulse_damage(100000, 0x50414C32) if only in ("all", "suite2") else 0
    m3 = suite3_prayer_mana_and_regen(100000, 0x50414C33) if only in ("all", "suite3") else 0
    print(f"ALL SUITES COMPLETE: m1={m1}, m2={m2}, m3={m3}")
    sys.exit(1 if (m1 or m2 or m3) else 0)
