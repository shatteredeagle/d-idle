# Paladin melee skill rules: Smite, Zeal, and Sacrifice; game.exe 1.14d vs d2d (PALSKILL)

> status: research, game.exe 1.14d vs d2d (decomp queue item 3, Issue #3). Follows docs/research/pal-skill-trace.md and docs/research/paladin-act1-recon.md section 6 rows 1 & 5.
> Author: Chuck (@chuckado) with Claude / Antigravity
> Tags: **V read** disassembled in game.exe 1.14d (addresses and functions given, no disassembly or decompiled code reproduced); **V run** run natively in game.exe under the emulator (scripts named with seeds); **D** from the patched 1.14d tables (Skills.txt, Items.txt, MonStats.txt read with mpq reader); **S** d2d source or upstream doc / earlier trace; **I** inference; **U** not verified.

## Summary

- **Smite (V read, I).** `srvdofunc 150` (`FUN_005ce9f0`) requires an equipped shield (`FUN_0063c8f0`). Base damage is shield base min / max damage (`Items.txt +0xfe/+0xff`), plus stat 17/18 min / max damage added directly by Holy Shield (`state 0x65`, `FUN_00647bc0` / `FUN_00647d00`). Damage percent bonus is `calc1` (`ln34 = 15 + 15*(lvl-1)%`), plus `stat 25` (off-weapon ED), plus `stat 0` (Strength) via `ItemGetStrBonus` (`FUN_00629860` returns 100 for shields, yielding `+1%` damage per point of Strength). Stun length is `calc2` (`min(250, ln12) = min(250, 15 + 5*(lvl-1))` frames; hard-capped at 250 frames = 10.0 seconds). Knockback is set via `ResultFlags = 8` (bit 3). Smite **unconditionally hits** because `FUN_005ce9f0` forces bit 0 of the hit flags after `FUN_0057ec10` (`0x5ceba8` call, setting bit 0 via bitwise OR with 1 at `0x5cebb0`): shield block, weapon block, defense and attack rating are completely bypassed (a blocking target may trigger a block animation, but still takes full damage, stun, and knockback). Mana cost is 2 mana (`manashift 8, mana 2, minmana 1`), paid in `srvdofunc 150`. A held button repeats Smite attack animation (mode `S1`) each cycle, paying 2 mana per swing.
- **Zeal (V read, I).** `srvstfunc 37` (`FUN_005daf40`) and `srvdofunc 13` (`FUN_005dbc60`). Hit count is `calc1` (`min(par5 + lvl - 1, par6)` with `par5 = 2, par6 = 5`): 2 hits at level 1, 3 at level 2, 4 at level 3, and 5 hits at level 4+ (cap 5). Damage percent bonus is `calc2` (`((lvl < 5) ? 0 : ((lvl-4) * 6)) + Sacrifice.blvl * 12%`). Attack rating bonus is `ToHit 10, LevToHit 10` (`10 + 10*(lvl-1)%`). Mana cost is 2 mana (`manashift 8, mana 2, minmana 1`), **paid once at srvstfunc 37** when the Zeal sequence starts; individual swings 2..5 cost 0 mana. When a target dies or leaves melee reach during the sequence (`FUN_00622c40` fails), `srvdofunc 13` searches within melee reach for another enemy using filter `0x20003` (`FUN_0056bd10`). In `game.exe`, Zeal uses 100% frame rollback (`Param2 = 100`, `FUN_004efc20`, `FUN_0056e210`, `FUN_00553b10` resetting current animation frame `[esi + 0x44]` directly back to `action_frame` `[edi + 0xa8]`), executing rapid consecutive strikes (~4 frames at base 1H speed, U without live animation timing capture). A held button executes a fresh 2..5 hit Zeal sequence as soon as the previous sequence finishes, paying 2 mana per sequence.
- **Sacrifice (V read, I).** `srvstfunc 29` (`FUN_005ce790`) and `srvdofunc 64` (`FUN_005ce8e0`). Damage percent bonus is `calc1` (`(180 + 15*(lvl-1)) + Redemption.blvl * 15% + Fanaticism.blvl * 5%`). Attack rating bonus is `ToHit 20, LevToHit 7` (`20 + 7*(lvl-1)%`). Mana cost is 0 mana (`mana 0, lvlmana 0, minmana 0`). Self-damage is `calc2` (`par3 = 8%`): `srvdofunc 64` (`0x5ce961..0x5ce984`) takes the physical damage dealt, **caps it at the target's current hitpoints** (`min(phys_damage, target_hp)`), and inflicts 8% of that capped value onto the caster as direct internal damage (flag `0x1000`). If self-damage meets or exceeds current player life (reducing life to 0), the player dies. A held button repeats Sacrifice attack animation (mode `A1`, `seqnum 2`) each cycle, taking 8% life cost on each landed hit.
- **d2d implementation in `skill_built` (S).** Contrary to early concerns that Smite, Zeal, and Sacrifice might fall back to plain attack, d2d's `skill_built` (`components/game/fight.hpp:62-75`) explicitly implements all three:
  - Smite: `skill.srvdofunc == ServerDoFunction::kSmite` -> true.
  - Zeal: `runs(skill, ServerStartFunction::kRepeatedHit, ServerDoFunction::kRepeatedHit)` -> true.
  - Sacrifice: `runs(skill, ServerStartFunction::kSacrifice, ServerDoFunction::kSacrifice)` -> true.
  None of Smite, Zeal, or Sacrifice falls back to plain attack in d2d.
- **Deviations d2d vs game.exe:**
  - `D8`: Smite blockable in d2d: `components/rules/combat.hpp:499` checks `target.block > 0` without verifying `!swing.smite`, allowing blocking monsters to nullify Smite damage; in `game.exe`, `FUN_005ce9f0` forces the hit bit (setting bit 0 via bitwise OR with 1 at `0x5cebb0`), so Smite is completely unblockable.
  - `D9`: Zeal missing frame rollback in d2d: `components/game/fight/strike.cpp:901` waits for `now_ms >= actor->pmode_until` (the full ~14-16 frame `A1` animation duration) between every Zeal strike; `game.exe` uses 100% frame rollback (`Param2 = 100`, `FUN_00553b10`) to strike rapidly (~4 frames at base 1H speed, U).

## 1. Smite, Zeal and Sacrifice Dispatch (game.exe 1.14d)

| Field / Mechanic | Smite (Id 97) | Zeal (Id 106) | Sacrifice (Id 96) | Tag |
|---|---|---|---|---|
| srvstfunc | None | **37** (`FUN_005daf40`) | **29** (`FUN_005ce790`) | D, V read |
| srvdofunc | **150** (`FUN_005ce9f0`) | **13** (`FUN_005dbc60`) | **64** (`FUN_005ce8e0`) | D, V read |
| anim / seqnum | S1 / None | A1 / None | A1 / **2** | D, V read |
| Base weapon / shield | Shield equipped (`FUN_0063c8f0`); reads shield min/max from `Items.txt +0xfe/+0xff` | Mainhand weapon physical damage | Mainhand weapon physical damage (`SrcDam 128` = 100%) | D, V read |
| Holy Shield synergy | Active `state 0x65` adds Holy Shield stat 17/18 min/max to base shield damage | None | None | V read, D |
| Hit count | 1 hit per action cycle | **calc1**: `min((par5 + lvl - 1), par6)` (par5=2, par6=5): L1=2, L2=3, L3=4, L4+=5 | 1 hit per action cycle | D, V read |
| Damage % bonus | **calc1**: `ln34` = `15 + 15*(lvl-1)%` | **calc2**: `((lvl < 5) ? 0 : ((lvl-4) * 6)) + Sacrifice.blvl * 12%` | **calc1**: `(180 + 15*(lvl-1)) + Redemption.blvl * 15% + Fanaticism.blvl * 5%` | D, V read |
| Stat bonuses to damage | stat 25 (off-weapon ED) + stat 0 (Strength) via `ItemGetStrBonus` = 100 (+1% per Str) | stat 25 (ED) + stat 0 (Strength) / stat 2 (Dexterity) per weapon type | stat 25 (ED) + stat 0 (Strength) / stat 2 (Dexterity) per weapon type | V read |
| Attack Rating bonus | None (always hits, bypasses AR check) | `ToHit 10, LevToHit 10` = `10 + 10*(lvl-1)%` | `ToHit 20, LevToHit 7` = `20 + 7*(lvl-1)%` | D, V read |
| Hit check & Block bypass | Calls `FUN_0057ec10` (`0x5ceba8`) then sets bit 0 via bitwise OR with 1 (`0x5cebb0`): ignores defense, shield block, and weapon block | Standard melee hit check `FUN_0057ec10` | Standard melee hit check `FUN_0057ec10` | V read |
| Stun duration | **calc2**: `min(250, ln12)` = `min(250, 15 + 5*(lvl-1))` frames; cap 250 frames (10.0 s) | None | None | D, V read |
| Knockback | ResultFlags = 8 (bit 3 set) | None | None | D, V read |
| Mana cost | 2 mana (`manashift 8, mana 2, minmana 1`), paid in `srvdofunc 150` | 2 mana (`manashift 8, mana 2, minmana 1`), **paid once at srvstfunc 37** | 0 mana (`mana 0, lvlmana 0, minmana 0`) | D, V read |
| Life cost / Self-damage | None | None | **calc2**: `par3` = 8% of `min(phys_damage_dealt, target_current_hp)`. Direct internal damage (`0x1000` flag); lethal | D, V read |
| Target reacquisition | Target locked | If current target dies or leaves reach (`FUN_00622c40`), searches reach for new target (`0x20003` filter, `FUN_0056bd10`) | Target locked | V read |
| Animation & Rollback | S1 full cycle | A1 with `Param2 = 100` (% frame rollback via `FUN_004efc20` / `FUN_0056e210` / `FUN_00553b10`) | A1 full cycle | D, V read |
| Held button behavior | Repeats Smite attack each animation cycle (S1), paying 2 mana per swing | Repeats full Zeal sequence (2..5 hits) upon completion if target in reach, paying 2 mana per sequence | Repeats Sacrifice attack each animation cycle (A1), paying 8% life of damage dealt per hit | V read |

### Smite Dispatch Details (FUN_005ce9f0, srvdofunc 150)
1. **Shield Requirement (V read):** Calls `FUN_0063c8f0` on caster; if no shield is equipped, returns 0 immediately. Reads base shield damage from `Items.txt` row of the shield: min damage at offset `+0xfe`, max damage at `+0xff`.
2. **Holy Shield Bonus (V read, D):** Checks if state 0x65 (`holy_shield`, D) is active on the caster (`FUN_00639df0`). If active, calls `FUN_00647bc0` and `FUN_00647d00` on the Holy Shield statlist to retrieve min and max damage bonus, and adds them directly to the shield's base damage.
3. **Damage Scaling (V read, D):** Calls `FUN_00646ca0` with calc1 (`ln34 = 15 + 15*(lvl-1)%`). Passes shield item, caster, min/max base damage, and calc1 to `FUN_0057b420`. `FUN_0057b420` queries `ItemGetStrBonus` (`FUN_00629860`), which returns 100 for shields, adding `Strength * 100 / 100 = +1%` per point of Strength. Off-weapon ED (`stat 25`) is added. Min damage is clamped to -90% floor if negative.
4. **Stun & Knockback (V read, D):** Evaluates calc2 (`min(250, ln12)` = `min(250, 15 + 5*(lvl-1))` frames), writing the stun length to the blow packet offset `+0x44`. ResultFlags (`Skills.txt +0x130 = 8`) is merged into packet flags (bit 3 = knockback).
5. **Always-Hit Rule (V read):** `FUN_005ce9f0` calls `FUN_0057ec10` at `0x5ceba8` to perform standard defense and block checks, but immediately sets bit 0 via bitwise OR with 1 at `0x5cebb0`. Because bit 0 indicates hit landed, Smite cannot miss. Even if a target rolls a block (bit 4), the hit still lands and delivers full damage, stun, and knockback.

### Zeal Dispatch Details (FUN_005daf40 / FUN_005dbc60, srvstfunc 37 / srvdofunc 13)
1. **Sequence Initialization (V read, D):** In `srvstfunc 37` (`FUN_005daf40`), evaluates calc1 (`min((par5 + lvl - 1), par6)` with par5=2, par6=5). Sets remaining hit count on the skill node via `FUN_00644560`: level 1 has 2 hits, level 2 has 3, level 3 has 4, level 4+ has 5 hits. Pays 2 mana (`FUN_00644b10`).
2. **Strike Execution (V read, D):** In `srvdofunc 13` (`FUN_005dbc60`), verifies the target is alive and in melee reach (`FUN_00622c40`). If the target died or left reach, invokes target finder `FUN_0056bd10` with enemy filter `0x20003` within melee reach (`FUN_0056e510`) to switch to another target.
3. **Damage & AR Bonus (V read, D):** Attack rating bonus is computed by `FUN_006449f0`: `ToHit + (lvl - 1) * LevToHit` = `10 + 10*(lvl-1)%`. Damage bonus is calc2: `((lvl < 5) ? 0 : ((lvl-4) * 6)) + Sacrifice.blvl * 12%`.
4. **Rapid Chaining via Rollback (V read):** After each strike, decrements remaining hit count. If hits remain (> 0), calls `FUN_004efc20` (reading `Param2 = 100`, frame rollback) and `FUN_0056e210`. `FUN_0056e210` invokes `FUN_00553b10` with rollback percent 100: at `0x553b47`, `100 - rollback_pct = 0`, resetting current anim frame `[esi + 0x44]` directly back to `action_frame` (`[edi + 0xa8]`), eliminating the wind-up and recovery delays for subsequent strikes (~4 frames per hit at base 1H speed, U).

### Sacrifice Dispatch Details (FUN_005ce790 / FUN_005ce8e0, srvstfunc 29 / srvdofunc 64)
1. **Damage & AR Bonus (V read, D):** Attack rating bonus is `ToHit + (lvl - 1) * LevToHit` = `20 + 7*(lvl-1)%`. Damage bonus is calc1: `180 + 15*(lvl-1)% + Redemption.blvl * 15% + Fanaticism.blvl * 5%`. Mana cost is 0.
2. **Self-Damage Cap at Target HP (V read, D):** In `srvdofunc 64` (`FUN_005ce8e0` at `0x5ce961..0x5ce984`), physical damage dealt is compared to target current hitpoints (`stat 6`, `FUN_00625480`). If damage dealt exceeds target HP, damage is clamped to target HP: `capped_damage = min(phys_damage, target_hp)`.
3. **Self-Damage Infliction (V read, D):** Evaluates calc2 (`Param3 = 8%`). Calculates `8% * capped_damage` via MulDiv (`FUN_00483360`). Inflicts this damage on the caster with internal direct damage flag `0x1000` (`FUN_0057c6c0` / `FUN_0057cee0`). This damage can reduce player HP to 0 and kill the player.

---

## 2. Deviations d2d vs game.exe (Paladin Melee)

| id | rule | game.exe | d2d | tag | differential today / buildable |
|---|---|---|---|---|---|
| D8 | Smite unblockable | `FUN_005ce9f0` forces hit bit (setting bit 0 via bitwise OR with 1 at `0x5cebb0`), completely ignoring monster shield/weapon block | `components/rules/combat.hpp:499` checks `target.block > 0` without `!swing.smite`, allowing blocking monsters to nullify Smite damage | V read, S | Buildable: test `player_blow` with `swing.smite = true` against target with block > 0 |
| D9 | Zeal rapid frame rollback | `Param2 = 100` (% frame rollback via `FUN_004efc20` / `FUN_00553b10`); anim frame resets directly to action frame, strikes chain rapidly (~4 frames at base 1H speed, U) | `components/game/fight/strike.cpp:901` waits for full `pmode_until` (~14-16 frames) between every strike | V read, S | Buildable: measure Zeal sequence duration in `Fight::strike` |

---

## 3. Verification & Formula Checks

> **Note on Case Separation:**
> - **8,640-Case Elemental Differential (V run):** In [`pal-skill-trace.md`](pal-skill-trace.md#7-differentials-v-run-the-worker-host-nice-one-process-at-a-time), the 8,640 elemental cases represent an emulator vs. d2d differential comparing native `game.exe` 1.14d (`pal_dmg.py gen, game`, calling `FUN_00644d50` / `FUN_00644e40` / `FUN_00644f20`) line-for-line against d2d commit `72132b99` (`pal_test --skill-damage`, testing `elem_damage` / `elem_length`). The input space is an exhaustive Cartesian product grid: 6 skills (Holy Bolt, Blessed Hammer, Holy Fire, Holy Freeze, Holy Shock, Fist of the Heavens) x 40 skill levels (1..40) x 6 synergy point tiers (0, 1, 5, 10, 20, 30) x 3 character levels (clvl 1, 30, 99) x 2 synergy patterns = 8,640 cases. This matrix has **no random seed** (exhaustive evaluation) and captured **0 differ**.
> - **Melee Formula Self-Check (I):** The formulas for Zeal, Smite, and Sacrifice documented above are inferred from reading `Skills.txt` calcs and decompiled 1.14d dispatch functions (`FUN_005ce9f0`, `FUN_005daf40`, `FUN_005dbc60`, `FUN_005ce790`, `FUN_005ce8e0`). The helper `tools/emu/verify_palskills.py` provides an offline arithmetic consistency self-check of these written formulas. Because it checks the arithmetic of the written formulas rather than executing native `game.exe` or `d2d` binaries, these formula rows are tagged **I** (inference), without claiming empirical emulator case counts.

### 3.1 Executable Verification Command & Captured Output

- **Executable Command:** `python tools/emu/verify_palskills.py`
- **Prerequisites:** Python 3 (standard library only)
- **Input Parameters:** Progression across skill levels 1..60 and parameterized input ranges
- **Captured Output:**
  ```text
  Verified formula consistency: 0 mismatches
  ```

### 3.2 Per-Rule Verification Results

| Verification Suite | Tag | Executable Command & Inputs | Case / Level Range | Captured Output | Verified Rule & Notes |
|---|:---:|---|---|---|---|
| Zeal Hit Count (`calc1`) | I | `python tools/emu/verify_palskills.py` | Skill levels 1..60 | `0 mismatches` | `min(2 + lvl - 1, 5)`: 2 at L1, 3 at L2, 4 at L3, 5 at L4..L60 |
| Smite Stun Length (`calc2`) | I | `python tools/emu/verify_palskills.py` | Skill levels 1..60 | `0 mismatches` | `min(250, 15 + 5*(lvl - 1))`: 15 at L1, 110 at L20, capped at 250 at L48..L60 |
| Zeal Attack Rating Bonus | I | `python tools/emu/verify_palskills.py` | Skill levels 1..60 | `0 mismatches` | `10 + 10*(lvl - 1)%` verified across all levels |
| Sacrifice Attack Rating Bonus | I | `python tools/emu/verify_palskills.py` | Skill levels 1..60 | `0 mismatches` | `20 + 7*(lvl - 1)%` verified across all levels |
| Smite Mana Cost (`FUN_00644b10`) | I | `python tools/emu/verify_palskills.py` | Skill levels 1..60 | `0 mismatches` | `(2 + 0*(lvl - 1)) << 8 = 512` (2.0 mana in 256ths) verified across all levels |
| Zeal Mana Cost (`FUN_00644b10`) | I | `python tools/emu/verify_palskills.py` | Skill levels 1..60 | `0 mismatches` | `(2 + 0*(lvl - 1)) << 8 = 512` (2.0 mana in 256ths) verified across all levels |
| Sacrifice Mana Cost (`FUN_00644b10`) | I | `python tools/emu/verify_palskills.py` | Skill levels 1..60 | `0 mismatches` | `0` mana verified across all levels |
| Smite / Zeal / Sacrifice Arithmetic Suite | I | `python tools/emu/verify_palskills.py` | Levels (1..60), synergies (0..30), Str (10..500), off-weapon ED (0..400), target HP (1..50,000) | `0 mismatches` | Arithmetic consistency self-check for Zeal ED/AR, Sacrifice ED/AR/capped self-damage, Smite ED/Strength scaling (I) |

---

## 4. Bot findings

| id | finding | tag | impact |
|---|---|---|---|
| B6 | Smite always hits and stuns up to 10 s; shield base damage + Holy Shield bonus + Strength % ED; costs 2 mana | V read, I | Excellent single-target lock against tough foes or bosses; independent of weapon damage/durability |
| B7 | Zeal hit count scales 2 -> 3 -> 4 -> 5 at skill levels 1, 2, 3, 4+; costs 2 mana once per sequence | V read, I | Putting 4 points into Zeal reaches the max 5 hits; holding the key repeats 5-hit sequences |
| B8 | Sacrifice costs 0 mana but inflicts 8% of damage dealt as self-damage (capped at foe HP); lethal | V read, I | High risk at low character life; safe only when life leech or potion buffer exceeds self-hurt |

---

## 5. Drill recipe (live; lab binary `lab-debug-skills`)

> **Note on Evidence Tags:** Per the contributor rulebook (`docs/contributing/agent-work.md`), agent runs operate offline without access to live game instances or the worker host. The live verification drills below specify maintainer-only commands for `lab-debug-skills` and are tagged **U** (unverified offline).

| Drill ID | Mechanic Tested | Tag | Exact Executable Commands & Inputs | Expected Behavior on `game.exe` 1.14d | Expected Behavior on `d2d` today |
|---|---|:---:|---|---|---|
| **Z1** | Zeal hit count scaling & rapid rollback (D9) | U | `debug points 106 1`<br>`debug points 106 4`<br>`debug cast 106 unit <target>` | 2 strikes land at level 1; 5 strikes land at level 4+. Strikes chain rapidly every ~4 frames (U) via 100% frame rollback (`FUN_004efc20` / `FUN_00553b10`). | 2 strikes at level 1; 5 strikes at level 4+. Each strike waits for full ~14-16 frame animation recovery (`strike.cpp:901`). |
| **Z2** | Zeal retargeting on kill | U | Spawn 2 monsters in reach.<br>`debug mon set <id1> life 1`<br>`debug cast 106 unit <id1>` | When target 1 dies on strike 1, strikes 2..5 dynamically switch to monster 2 via `0x20003` filter search (`FUN_0056bd10`). | Target 1 dies on strike 1; subsequent strikes search for nearby targets in melee reach. |
| **S1** | Smite unconditional hit & block bypass (D8) | U | Spawn high-block monster.<br>`debug mon set <id> block 75`<br>`debug cast 97 unit <id>` | 100% hit rate and damage dealt on every swing. Target may show block animation, but damage, stun, and knockback are never avoided (`0x5cebb0`). | Monster with `target.block > 0` blocks Smite swings and nullifies damage (`combat.hpp:499`). |
| **S2** | Smite stun length | U | `debug points 97 1`<br>`debug points 97 20`<br>`debug cast 97 unit <id>` | Level 1 stuns target for 15 frames (0.60 s); level 20 stuns target for 110 frames (4.40 s) (`min(250, 15 + 5*(lvl-1))`). Target cannot attack or move. | Target stunned. |
| **C1** | Sacrifice self-damage cap at foe HP | U | `debug mon set <id> life 50 maxlife 50`<br>`debug cast 96 unit <id>` (with 500 damage) | Player takes 8% of target current HP (8% of 50 = 4 life cost), NOT 8% of 500 (40 life cost) (`min(phys_dam, target_hp)` at `0x5ce961`). | Player takes 4 self-damage (capped at target HP). |
| **C2** | Sacrifice lethal self-damage | U | `debug stat 6 10`<br>`debug cast 96 unit <id>` (with 200 damage to healthy foe) | Player takes 8% of 200 = 16 damage. Because player HP is 10 <= 16, player dies from direct self-damage (`0x1000` flag). | Player dies from lethal self-damage. |

---

## 6. Matrix cell updates (proposed text for `pal-skill-matrix.md`)

| Row | Cell | New value |
|---|---|---|
| Smite | hit rule | V read: unconditional hit (sets bit 0 via bitwise OR with 1 in `FUN_005ce9f0`), ignores AR, defense, and block (d2d deviates, D8) |
| Smite | damage | V read, I: shield base (`Items.txt +0xfe/+0xff`) + Holy Shield bonus + Str bonus (1% per Str) + calc1 ED (`15 + 15*(lvl-1)%`) |
| Smite | stun & knockback | V read, I: `ResultFlags = 8` (knockback); stun = `min(250, 15 + 5*(lvl-1))` frames (cap 250 frames) |
| Smite | mana | V read, I: 2.0 mana across all levels, paid in do; repeats on hold |
| Zeal | hit count | V read, I: `min(lvl + 1, 5)` hits (2 at L1, 3 at L2, 4 at L3, 5 at L4+) |
| Zeal | damage & AR | V read, I: AR `10 + 10*(lvl-1)%`; ED `((lvl < 5) ? 0 : ((lvl-4)*6)) + Sacrifice*12%` |
| Zeal | mana & retarget | V read: 2.0 mana paid once at srvstfunc; retargets in reach via `0x20003` filter; rollback 100% (d2d lacks rollback, D9) |
| Sacrifice | damage & AR | V read, I: AR `20 + 7*(lvl-1)%`; ED `180 + 15*(lvl-1)% + Redemption*15% + Fanaticism*5%` |
| Sacrifice | cost & self-hurt | V read, I: 0 mana; self-damage 8% of `min(phys_damage, target_hp)`; lethal if self-hurt >= player life |

---

## 7. Not traced (U) and limits

- The exact frame duration of Zeal rollback strikes across various weapon classes, WSM, and IAS (nominally ~4 frames for standard 1H weapons) is tagged U pending live frame-by-frame measurement.

- The live drill expectations in Section 5 (Z1, Z2, S1, S2, C1, C2) were designed for maintainer execution on `lab-debug-skills` and are tagged U because offline agents do not connect to live game instances.
- Exact hit recovery frame duration when a target blocks a Smite swing (in game.exe the block animation plays if block roll succeeds, but full damage and stun apply regardless).
- The weapon speed modifier (WSM) and IAS table for Zeal rollback animation frames when equipped with two-handed weapons.
- Vengeance (Id 111) and Charge (Id 107) melee dispatch details (deferred to future card).
- Monster AI response to Smite knockback into walls or impassable tiles.
