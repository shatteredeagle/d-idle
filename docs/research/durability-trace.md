# Durability Loss per Hit and per Block in 1.14d (`PALDUR`)

> **Authorship & Epistemic Notice:** This document was written by an AI agent (`Antigravity`) for work item [#2 (`PALDUR`)](https://github.com/shatteredeagle/d-idle/issues/2) following `docs/contributing/agent-work.md`.
> **Author:** Chuck (@chuckado) with Claude / Antigravity
> Every factual claim is tagged:
> - `V read` — verified by reading the 1.14d `Game.exe` binary (`SHA256 631066c1649c4ea9ffe48bf97e24c00bca1f7a6759c21150f1a79982589adaaf`, image base `0x00400000`) or `Patch_D2.mpq` (`SHA256 082f6bbd8edb2bee6efdc9eb7de9ae4bdbe4ddca9dbc8f7ba6fd5fbf3a36d2eb`)
> - `V read + self-check` — verified by static disassembly and executing original 1.14d `Game.exe` functions in the repo's Unicorn x86 harness compared against an independent Python oracle (not against a real `d2d` build; `100,128` test cases, `0` mismatches, `tools/emu/verify_paldur.py` with seed `0x50414c44`)
> - `V run` — verified by executing 1.14d `Game.exe` and comparing line-for-line with a real `d2d` build (reserved for real differentials; not run here)
> - `D` — arithmetic deduction from verified formulas or tables
> - `S` — secondary source: `d2d` (`d2decomped`) source at commit `6935cf6` or `d-idle` repository code/docs
> - `I` — inference
> - `U` — unverified in a live game instance

---

## 1. Executive Summary

Issue #2 (`PALDUR`, from `docs/community/open-work.md` and `docs/research/paladin-act1-recon.md` section 6 row 7) asks:
> *In `game.exe` 1.14d, what is the chance and the slot choice for durability loss when the player hits and when the player is hit or blocks, and does `d2d` implement the same rule (it has `debug wear` verbs but the loss rule is unknown)?*

### Short Answers

1. **When durability loss runs (`V read + self-check`):**
   - Across the entire 1.14d `Game.exe` binary, the single-item durability wear routine `FUN_00559e30` (`0x559e30..0x55a07f`) has only two callers: `0x57d3f3` (attacker's weapon) and `0x57d4d8` (defender's armor), both inside `FUN_0057d3d0` (`0x57d3d0..0x57d4e3`).
   - `FUN_0057d3d0` has only one caller: `0x57d5f6` inside the melee hit-record processor `FUN_0057d4f0` (`0x57d4f0..0x57d683`), gated behind `(dmg.result_flags & 0x01) != 0` at `0x57d591` (plus Amazon Impale's dedicated `FUN_005daa40` at `0x5daa40..0x5dab3d`, called at `0x5dac99` inside `FUN_005dabe0`).
2. **Blocking an attack NEVER causes durability loss (`V read + self-check`):**
   - In the melee to-hit and block/dodge resolver `FUN_0057ec10` (`0x57ec10..0x57ed44`), when `FUN_0057dfb0` (`0x57dfb0..0x57e08b`, shield block + `FUN_0057dd60` weapon block / dodge / avoid / evade) returns non-zero (`0x01` shield block, `0x02` avoid, `0x04` dodge, `0x08` evade, `0x10` weapon block), `FUN_0057ec10` sets `result_flags |= 0x10` (for shield block) or `0x8000` (for weapon block) and clears bit `0` (`0x57ed16`: `result_flags &= ~0x0001`).
   - Because `FUN_0057d4f0` skips `FUN_0057d3d0` whenever `(dmg.result_flags & 0x01) == 0` (`0x57d591..0x57d595`), a blocked attack, missed swing, or dodged/avoided/evaded swing never calls `FUN_0057d3d0` or `FUN_00559e30`. A shield only loses durability when a melee attack **lands** (`result_flags & 0x01 != 0`) and the shield's arm slot (`4` or `5`) is selected in the defender's 7-slot armor lottery (`FUN_0057d3d0`).
3. **When the player hits in melee (`FUN_0057d3d0` -> `FUN_00535bc0` -> `FUN_00559e30`, `V read + self-check`):**
   - If the attacker is a player (`[attacker + 0x00] == 0`), `FUN_0057d3d0` selects the active weapon via `FUN_00535bc0` (`0x535bc0..0x535cff`) and passes it to `FUN_00559e30`.
   - For a normal melee weapon (`ItemTypes` `0x2d` `"weap"`, not throwable), the chance to lose `1` durability (`stat 72`, `0x48`) per landed melee hit is **`4%`** (`rand_range(&attacker.seed, 100) < 4`).
   - For a throwable weapon (`ItemTypes` `+0x10` `Throwable != 0`, `FUN_0062ba80`) used in melee, the chance per landed melee hit is **`10%`** in Expansion (`[game + 0x70] != 0`, `I`) and **`0%`** in Classic (`[game + 0x70] == 0`, `I`, early exit at `0x559e85..0x559e91`).
   - Bows/crossbows and Phase Blades (`Items.txt` `nodurability != 0` or `durability == 0`), items with `stat 152` (`item_indesctructible > 0`), and missile attacks (`FUN_005ad730`) have **`0%`** durability loss in `FUN_00559e30`.
   - **Paladin Smite (`srvdofunc` 150, `FUN_005ce9f0`, `V read`):** Smite reads the shield (`FUN_0063c8f0`) for base damage (`[shield_rec + 0xfe..0xff]`), sets `result_flags |= 0x01` (`0x5cebb0`), and queues a standard melee hit record processed by `FUN_0057d4f0` (`0x5cec35`). Because `FUN_0057d3d0` calls `FUN_00535bc0(attacker)`, **Smite wears the Paladin's equipped weapon (`4%` chance per hit), not the shield**.
4. **When the player is hit in melee (`FUN_0057d3d0` -> `FUN_00559e30`, `V read + self-check`):**
   - If the defender is a player (`[defender + 0x00] == 0`), `FUN_0057d3d0` scans the 7-entry slot-weight table at `0x00732b90` (`[0x00732bc8] = 7`):
     - Slot `1` (**Head / Helm**): weight **`3`**
     - Slot `3` (**Torso / Body Armor**): weight **`5`**
     - Slot `4` (**Right Arm**, if `"armo"`): weight **`4`**
     - Slot `5` (**Left Arm**, if `"armo"`): weight **`4`**
     - Slot `8` (**Belt**): weight **`2`**
     - Slot `9` (**Feet / Boots**): weight **`2`**
     - Slot `10` (**Gloves**): weight **`2`**
   - Only equipped items matching `ItemTypes` `0x32` (`"armo"`, via `FUN_00629bb0(item, 0x32)`) are included in `total_weight`. A weapon in slot `4` or `5` is not `"armo"` and contributes weight `0`; a shield (including Paladin auric shields) is `"armo"` and contributes weight `4`.
   - One occupied `"armo"` slot is chosen with probability `weight[slot] / total_weight` (`3/18`, `5/18`, `4/18`, `2/18`, `2/18`, `2/18` when all 6 armor pieces are worn), and `FUN_00559e30` rolls a **`10%`** chance (`rand_range(&defender.seed, 100) < 10`) to lose `1` durability on that chosen armor piece.
5. **Does `d2d` (`6935cf6`) implement the same rule? (`S`, `V read`):**
   - **`d2d` DOES implement combat durability wear** in `Fight::wear`, `Fight::wear_weapon`, and `Fight::wear_armor` (`components/game/fight/missiles.cpp:130-180`), triggered from `Fight::tick` (`components/game/fight/tick.cpp:588-589`) and `Fight::melee` (`components/game/fight/strike.cpp:607`) when `hits > 0` (`S`).
   - Its base percentages (`4%` melee weapon, `10%` throwable/armor), 7-slot armor weight table (`Head 3, Torso 5, RightArm 4, LeftArm 4, Belt 2, Feet 2, Gloves 2`), and block exemption (`taken.blocked = true, taken.hit = false` in `components/rules/combat.hpp:611`) match `Game.exe` 1.14d (`S`, `V read`).
   - However, we found **six deviations (`D1`–`D6`)** between `Game.exe` 1.14d and `d2d` (`6935cf6`), including two high-impact gameplay bugs in `d2d`:
     - **`D1` (Monster missiles and explosions wear player armor in `d2d`):** In `Game.exe`, only melee hits (`FUN_0057d4f0`) call `FUN_0057d3d0`; missiles (`FUN_005ad730`) and area blasts (`FUN_0056b9c0`) never wear armor (`V read`). In `d2d`, monster missiles (`ai.hpp:797`), corpse explosions (`ai.cpp:2422`), suicide explosions (`deaths.cpp:172`), and Fire Enchanted death blasts (`deaths.cpp:191`) all call `foe.take(taken)` with `taken.hit = true`, incrementing `foe.hits` and triggering `wear_armor` (`S`).
     - **`D2` (Non-stackable weapons worn to `0` durability in `d2d` never get `flag 0x100` `broken`):** In `Game.exe`, `FUN_00559e30` sets a non-stackable weapon's `stat 72 = 0` on the hit frame, and `FUN_00580380` sets `flag 0x100` (`broken`) via `FUN_0055f850` when the attack animation finishes (`V read + self-check`). `d2d` ported `FUN_00559e30` without `FUN_00580380` (`missiles.cpp:141-146, 238`), so a weapon worn to `0` durability by normal combat hits in `d2d` never gets `item.flags |= 0x100u`, `rules::broken(item)` (`rules.hpp:519`) stays `false`, and only `debug wear` sets `0x100u` (`S`).

---

## 2. Combat Durability Call Graph in `Game.exe` 1.14d (`V read`)

| Function VA | Size | Calling Convention | Role | Callers in `Game.exe` 1.14d |
| :--- | :--- | :--- | :--- | :--- |
| `FUN_0057ec10` | `0x135` (`0x57ec10..0x57ed44`) | `eax = attacker`, `ecx = defender`, `(game, &dmg, ar_bonus)` (`ret 0xc`) | Melee to-hit roll (`result_flags \|= 0x01`) followed by block/dodge roll (`FUN_0057dfb0`) | Melee attack & skill strike handlers (`FUN_0057ef50`, Zeal, Fury, Fend, etc.) |
| `FUN_0057dfb0` | `0xdc` (`0x57dfb0..0x57e08b`) | `__fastcall(ecx = game, edx = defender, attacker, is_melee)` (`ret 8`) | Shield block roll (`0x01`) + delegates to `FUN_0057dd60` | `0x57ece8` (`FUN_0057ec10`), `0x5ad816` (missile block) |
| `FUN_0057dd60` | `0x247` (`0x57dd60..0x57dfa6`) | `eax = attacker`, `ecx = defender`, `(game, &out, is_melee)` (`ret 0xc`) | Weapon Block (`0x10`), Avoid (`0x02`), Dodge (`0x04`), Evade (`0x08`) | `0x57dfd6` (`FUN_0057dfb0`) |
| `FUN_0057d4f0` | `0x194` (`0x57d4f0..0x57d683`) | `__fastcall(ecx = game, edx = attacker, defender, arg1)` (`ret 8`) | Dequeues melee hit record from `[attacker + 0xac]`; calls `FUN_0057d3d0` iff `dmg.result_flags & 0x01 != 0` | Melee animation frame & skill `srvdofunc` handlers (including Smite `0x5cec35`) |
| `FUN_0057d3d0` | `0x114` (`0x57d3d0..0x57d4e3`) | `eax = attacker`, `(game, defender)` (`ret 8`) | Wears attacker's active weapon (`FUN_00535bc0`) and selects 1 of 7 defender `"armo"` slots (`0x00732b90`) | **Only** `0x57d5f6` inside `FUN_0057d4f0` |
| `FUN_00559e30` | `0x250` (`0x559e30..0x55a07f`) | `__fastcall(ecx = game, edx = owner_unit, item)` (`ret 4`) | Rolls `4%` / `10%` wear on `item`, decrements `stat 72`, breaks armor or depletes stack | **Only** `0x57d3f3` and `0x57d4d8` inside `FUN_0057d3d0` |
| `FUN_00535bc0` | `0x140` (`0x535bc0..0x535cff`) | `__fastcall(ecx = player)` (`ret`) | Resolves the player's active swinging weapon (`Skills.txt` `weapsel` for Barbarian/Assassin dual wield) | `0x57d3e4` (`FUN_0057d3d0`) and 13 attack/skill helpers |
| `FUN_00629930` | `0x59` (`0x629930..0x629988`) | `__stdcall(item)` (`ret 4`) | Returns `1` iff item uses durability (`nodurability == 0 && durability > 0 && maxdur != 0 && stat_152 <= 0`) | `0x559e5e` (`FUN_00559e30`), `0x5803ed` (`FUN_00580380`), `0x62e88a` (`FUN_0062e660` repair cost), `0x5761fb` (`FUN_005761c0` smith repair) |
| `FUN_0055f850` | `0xac` (`0x55f850..0x55f8fb`) | `__fastcall(ecx = game, edx = player, item)` (`ret 4`) | Sets `item_data.flags \|= 0x100` (`broken`), strips item stats from player, sets `stat 72 = 0`, plays sound `9` | `0x559f3a` (`FUN_00559e30` armor), `0x5800ca` (`FUN_00580030` magic+ throwable), `0x58042a` (`FUN_00580380` weapon), `0x5dab36` (`FUN_005daa40` Impale) |
| `FUN_00580380` | `0xcd` (`0x580380..0x58044c`) | `__fastcall(ecx = game, edx = player)` (`ret`) | Scans equipped weapons (`location_type == 3`); calls `FUN_0055f850` on any weapon with `stat 72 <= 0` and `(flags & 0x100) == 0` | `0x581095` (end of attack sequence), `0x56ca96`, `0x56cace`, `0x5d8e55`, `0x5d8e9d`, `0x5da8ce`, `0x5dad2a` |
| `FUN_005daa40` | `0xfe` (`0x5daa40..0x5dab3d`) | `__fastcall(ecx = game, edx = player, skill_lvl)` (`ret 4`) | Amazon Impale (`srvdofunc` 9) dedicated weapon durability wear (`param3%` chance to lose `param4` durability) | `0x5dac99` (Impale `srvdofunc` 9 inside `FUN_005dabe0`) |

---

## 3. Why Blocking, Dodging, and Missiles Never Cause Durability Loss (`V read + self-check`)

### 3.1 Melee Hit vs Block (`FUN_0057ec10` -> `FUN_0057dfb0` -> `FUN_0057d4f0`)

1. **To-hit check (`0x57ec10..0x57ece1`, `V read`):**
   - `FUN_0057ec10` computes the melee hit chance and rolls `rand_range(&attacker.seed, 100)`.
   - If the attack misses, `dmg.result_flags` (at `[dmg + 0x04]`) has bit `0` (`0x0001`) cleared (`0x57eccd`: `result_flags &= ~0x0001`) and `FUN_0057ec10` returns `0` without calling `FUN_0057dfb0`.
   - If the attack hits, `FUN_0057ec10` sets bit `0` (`0x57ecd8`: `result_flags |= 0x0001`).
2. **Block / Dodge / Avoid / Evade check (`0x57ece2..0x57ed3b`, `V read`):**
   - When `[dmg + 0x00] & 0x20 == 0` (blockable attack), `FUN_0057ec10` calls `block_type = FUN_0057dfb0(ecx = game, edx = defender, attacker, 1)` (`0x57ece8`).
   - Inside `FUN_0057dfb0`:
     - First calls `FUN_0057dd60` (`0x57dfd6`), which checks Assassin Weapon Block (`stat 187`, `item_fractionaltargetac`, returning `0x10`), Amazon Dodge (`stat 69`, returning `0x04`), Avoid (`stat 68`, returning `0x02`), and Evade (`stat 129`, returning `0x08`).
     - If `FUN_0057dd60` returns `0`, `FUN_0057dfb0` checks whether the defender can shield-block (`0x57dfe3..0x57e083`) and rolls `rand_range(&defender.seed, 100) < block_chance`. On a shield block, `FUN_0057dfb0` returns `0x01`.
   - Back in `FUN_0057ec10` (`0x57ecf3..0x57ed21`):
     - If `block_type != 0`:
       - For shield block (`block_type == 0x01`), sets `result_flags |= 0x0010` (`0x57ecfe`) and sets `[dmg + 0x4c] = 1`.
       - For dodge (`0x04`) or avoid (`0x02`), sets `result_flags |= 0x0080`.
       - For evade (`0x08`), sets `result_flags |= 0x0100`.
       - For weapon block (`0x10`), sets `result_flags |= 0x8000`.
       - Then at **`0x57ed16`**, `FUN_0057ec10` **clears bit `0` (`0x0001`) of `dmg.result_flags` (`result_flags &= 0xfffe`)**, writes the updated flags back to `[dmg + 0x04]` (`0x57ed1c`), and returns `0` (`0x57ed1f`).
3. **Hit-record application (`FUN_0057d4f0`, `0x57d563..0x57d66d`, `V read + self-check`):**
   - `FUN_0057d4f0` copies the `0x70`-byte `DamageRecord` from `[hit_node + 0x14]` into its local stack frame (`result_flags` at `[ebp - 0x70]`).
   - After verifying melee reach (`FUN_00622c40` at `0x57d584`), `0x57d591..0x57d595` tests bit `0` (`(dmg.result_flags & 0x01) != 0`).
   - When bit `0` (`0x01`) is `0` (miss, shield block, weapon block, dodge, avoid, or evade), execution branches directly to `0x57d5fd`, bypassing `FUN_0057d3d0` (`0x57d5f6`) and proceeding to `FUN_0057cee0` (`0x57d658`, where `result_flags & 0x10` triggers the defender's block animation mode via `FUN_0057eb00` at `0x57d2c6`).
   - Therefore, **blocking an attack never wears the shield, any armor piece, or the attacker's weapon** (`V read + self-check`).

### 3.2 Missile Hits and Area Damage (`FUN_005ad730` & `FUN_0056b9c0`, `V read`)

- **Missile hit processor (`FUN_005ad730`, `0x5ad730..0x5ad8de`, `V read`):** Resolves missile to-hit (`FUN_0057ebb0`), block (`FUN_0057dfb0` with `is_melee = 0` at `0x5ad816`), hit effects (`FUN_0057c6c0`), and damage application (`FUN_0057cee0` at `0x5ad8bd`). It **never calls `FUN_0057d3d0` or `FUN_00559e30`**. Neither player missiles nor monster missiles cause weapon or armor durability loss in `Game.exe` 1.14d (`V read`).
- **Area/radial damage processor (`FUN_0056b9c0`, `0x56b9c0..0x56ba8a`, `V read`):** Applies radial damage via `FUN_0057c3a0` and `FUN_0057cee0` (`0x56ba71`) without calling `FUN_0057d3d0` or `FUN_00559e30`. Monster corpse explosions, suicide blasts, and Fire Enchanted death blasts never wear player armor in `Game.exe` 1.14d (`V read`).

---

## 4. Attacker Weapon Slot Selection and Loss Chance (`V read + self-check`)

### 4.1 Player Gate in `FUN_0057d3d0` (`0x57d3d7..0x57d3f6`, `V read`)

At the start of `FUN_0057d3d0(eax = attacker, game, defender)`:
1. Checks `attacker != NULL` and `[attacker + 0x00] == 0` (`unit_type == 0`, player). Monsters and mercenaries have `unit_type == 1` and never wear weapons (`0x57d3dd`).
2. Calls `weapon = FUN_00535bc0(ecx = attacker)` (`0x57d3e4`).
3. If `weapon != NULL`, calls `FUN_00559e30(ecx = game, edx = attacker, weapon)` (`0x57d3f3`).

### 4.2 Active Weapon Selection (`FUN_00535bc0`, `0x535bc0..0x535cff`, `V read + self-check`)

1. **Default primary weapon (`FUN_0063c9b0`, `0x63c9b0..0x63caa8`, `V read`):**
   - Inspects body slot `4` (`RightArm`, `[body_slots + 0x10]`) first, then body slot `5` (`LeftArm`, `[body_slots + 0x14]`).
   - Returns the first item that passes both `FUN_0062a4e0(item)` (`0x62a4e0..0x62a524`: `type == 4` and `(item_data.flags & 0x4100) == 0`, i.e., neither `0x0100` broken nor `0x4000` unequipped/requirements-unmet) and `FUN_00629bb0(item, 0x2d)` (`ItemTypes` `0x2d` `"weap"`).
2. **Dual-wielding check (`FUN_006235a0`, `0x6235a0..0x6235f8`, `V read + self-check`):**
   - For players (`unit_type == 0`), `FUN_006235a0` returns `1` only for `class_id == 4` (**Barbarian**) and `class_id == 6` (**Assassin**); for Amazon (`0`), Sorceress (`1`), Necromancer (`2`), Paladin (`3`), and Druid (`5`), it returns `0`, so `FUN_00535bc0` immediately returns the weapon found by `FUN_0063c9b0`.
   - For Barbarian and Assassin (`FUN_006235a0 == 1`), `FUN_00535bc0` reads the active skill (`FUN_00620250` -> `[player_info + 0x10]`), fetches its `Skills.txt` record (`FUN_00644140`), and switches on `weapsel = uint8_t[skill_rec + 0x168]` (`0x535c27..0x535cc7`, jump table at `0x00535d00`), where `FUN_0063c050(inv, 5)` returns the primary weapon (`== FUN_0063bef0(inv)`) and `FUN_0063c050(inv, 6)` returns the secondary/off-hand weapon (`!= FUN_0063bef0(inv)`):

| `Skills.txt` `weapsel` (`+0x168`) | `Game.exe` Branch (`FUN_00535bc0`) | Weapon Selected for Durability Wear (`V read + self-check`) |
| :---: | :--- | :--- |
| `0` (or `> 4`) | `0x535caa` (`default`) | Primary weapon (`FUN_0063c050(inv, 5)`) if `"weap"`, else off-hand (`FUN_0063c050(inv, 6)`) |
| `1` | `0x535cc4` (`jt[0]`) | Off-hand weapon (`FUN_0063c050(inv, 6)`) |
| `2` | `0x535c3f` (`jt[1]`) | Checks skill flag `FUN_006446a0(skill) & 0x2000` (`[skill + 0x0c] & 0x2000`): if both hands hold `"weap"`, picks off-hand (`6`) when bit `0x2000` is set, else primary (`5`) |
| `3` | `0x535c89` (`jt[2]`) | Checks unit flag `([player + 0x38] >> 8) & 1`: picks off-hand (`6`) when bit `0x100` is `1`, else primary (`5`) |
| `4` | `0x535ca5` (`jt[3]`) | Clears fallback `[ebp - 4] = 0` and returns `NULL` (`0`, no weapon wear) |

3. **Paladin Smite (`srvdofunc` 150, `FUN_005ce9f0`, `0x5ce9f0..0x5cec45`, `V read`):**
   - `FUN_005ce9f0` checks that the Paladin has a shield equipped (`FUN_0063c8f0` at `0x5cea76`), reads the shield's `Items.txt` record (`+0xfe`, `+0xff`) for base Smite damage (`0x5cea96..0x5ceaa0`), sets `dmg.result_flags |= 0x01` (`0x5cebb0`), queues the hit record (`0x5cec23`), and immediately calls `FUN_0057d4f0` (`0x5cec35`).
   - Inside `FUN_0057d4f0` -> `FUN_0057d3d0`, `FUN_00535bc0` selects the Paladin's **equipped weapon** (`slot 4` or `5` matching `"weap"`), NOT the shield (`"armo"`). Thus **Smite wears the Paladin's equipped weapon at `4%` per landed hit, and never wears the shield** (`V read`).

### 4.3 Weapon Durability Loss Chance (`FUN_00559e30`, `0x559e30..0x559ec5`, `V read + self-check`)

Given `item` passed to `FUN_00559e30(ecx = game, edx = unit, item)`:
1. Checks `is_armo = FUN_00629bb0(item, 0x32)` and `is_weap = FUN_00629bb0(item, 0x2d)`. If both are `0`, returns.
2. Calls **`FUN_00629930(item)`** (`0x629930..0x629988`, `V read + self-check`), which returns `1` if and only if all five conditions hold:
   - `[item + 0x00] == 4` (`unit_type == 4`)
   - `Items.txt` `nodurability` byte (`[rec + 0x113]`) `== 0`
   - `Items.txt` `durability` byte (`[rec + 0x112]`) `> 0`
   - `FUN_00625e00(item)` (`max_durability`, `stat 73` + modifiers) `!= 0`
   - `FUN_00625480(item, 0x98, 0)` (`stat 152`, `item_indesctructible`) `<= 0`
   - *(Across all `659` `Items.txt` rows in 1.14d, `202` armors, `227` non-throwable melee weapons, and `30` throwable weapons have `nodurability == 0 && durability > 0`; `49` weapons—including all bows, crossbows, and `7cr` Phase Blade—have `nodurability != 0` or `durability == 0` and return `0` from `FUN_00629930`; `D + V read + self-check`.)*
3. Determines the percentage threshold `chance` (`0x559e70..0x559e97`, `V read + self-check`):

| Item Category | Condition in `FUN_00559e30` | Classic (`[game + 0x70] == 0`) | Expansion (`[game + 0x70] != 0`) | RNG Stream Advanced |
| :--- | :--- | :---: | :---: | :--- |
| Standard melee weapon | `is_weap != 0 && is_armo == 0 && !FUN_0062ba80(item)` | **`4%`** (`4 / 100`) | **`4%`** (`4 / 100`) | `attacker` unit seed (`[attacker + 0x20..0x24]`) |
| Throwable weapon used in melee | `is_weap != 0 && is_armo == 0 && FUN_0062ba80(item) != 0` | **`0%`** (returns before RNG step at `0x559e8b`) | **`10%`** (`10 / 100`) | `attacker` unit seed (Expansion only) |
| Armor / Shield | `is_armo != 0` | **`10%`** (`10 / 100`) | **`10%`** (`10 / 100`) | `defender` unit seed (`[defender + 0x20..0x24]`) |
| Bow / Crossbow / Phase Blade / Indestructible | `FUN_00629930(item) == 0` | **`0%`** (returns before RNG step) | **`0%`** (returns before RNG step) | None |

4. Advances `[unit + 0x20..0x24]` via the standard 64-bit MWC step `prod = uint64(seed_lo) * 0x6ac690c5 + seed_hi` (`0x559ea5..0x559ec1`) and checks `(uint32(prod) % 100) < chance`. If `>= chance`, returns without changing durability (`V read + self-check`).

### 4.4 Special Case: Amazon Impale (`FUN_005daa40`, `0x5daa40..0x5dab3d`, `V read`)

Amazon Impale (`skill 19`, `srvdofunc` 9 at `FUN_005dabe0`) runs a separate weapon-wear check `FUN_005daa40` at `0x5dac99` (inside `FUN_005dabe0`) in addition to normal melee hit processing:
- Evaluates `Skills.txt` `calc2` (`par3 = 25`, `25%` chance in 1.14d) against `rand_range(&player.seed, 100)`.
- If the roll is `< calc2` (or if `calc2 <= 0`, which forces wear unconditionally at `0x5daa76`), subtracts `par4` (`[skill_rec + 0x194] = 1` durability in 1.14d) from `stat 72` (`0x48`), and if `new_dur <= 0`, immediately breaks the weapon via `FUN_0055f850` (`0x5dab36`) (`V read`).

---

## 5. Defender Armor Slot Selection and Loss Chance (`V read + self-check`)

### 5.1 The 7-Slot Weight Table at `0x00732b90` (`FUN_0057d3d0`, `0x57d3f8..0x57d4e3`)

When a melee hit lands (`dmg.result_flags & 0x01 != 0`) and `defender` is a player (`[defender + 0x00] == 0`) with a non-null `Inventory` (`[defender + 0x60] != 0`), `FUN_0057d3d0` iterates `i = 0 .. 6` (`[0x00732bc8] = 7`) over the table at `0x00732b90` (`V read + self-check`):

| Table Index `i` | Table VA | Body Slot ID (`+0x00`) | Equipment Location | Weight (`+0x04`) | Standard 6-Armor Share (`total = 18`, `D`) | No-Shield 5-Armor Share (`total = 14`, `D`) | Effective Loss Chance per Landed Hit (`weight / 18 * 10%`, `D`) |
| :---: | :--- | :---: | :--- | :---: | :---: | :---: | :---: |
| `0` | `0x00732b90` | `1` | **Head (Helm)** | **`3`** | `3 / 18` (`16.67%`) | `3 / 14` (`21.43%`) | `1.667%` (`1 / 60`) |
| `1` | `0x00732b98` | `3` | **Torso (Body Armor)** | **`5`** | `5 / 18` (`27.78%`) | `5 / 14` (`35.71%`) | `2.778%` (`1 / 36`) |
| `2` | `0x00732ba0` | `4` | **Right Arm** (if `"armo"`, e.g. shield) | **`4`** | `0 / 18` (weapon) or `4 / 18` | `0 / 14` (weapon) | `2.222%` (`1 / 45`, if shield) |
| `3` | `0x00732ba8` | `5` | **Left Arm** (if `"armo"`, e.g. shield) | **`4`** | `4 / 18` (`22.22%`, shield) | `0 / 14` (empty / 2H / off-hand weapon) | `2.222%` (`1 / 45`, if shield) |
| `4` | `0x00732bb0` | `8` | **Belt** | **`2`** | `2 / 18` (`11.11%`) | `2 / 14` (`14.29%`) | `1.111%` (`1 / 90`) |
| `5` | `0x00732bb8` | `9` | **Feet (Boots)** | **`2`** | `2 / 18` (`11.11%`) | `2 / 14` (`14.29%`) | `1.111%` (`1 / 90`) |
| `6` | `0x00732bc0` | `10` | **Gloves** | **`2`** | `2 / 18` (`11.11%`) | `2 / 14` (`14.29%`) | `1.111%` (`1 / 90`) |
| — | `0x00732bc8` | — | **Table count (`7`), sum of 6 worn armor weights** | **`18`** | **`18 / 18` (`100%`)** | **`14 / 14` (`100%`)** | **`10.000%` total across worn armor** |

### 5.2 Exact Slot-Selection Algorithm (`0x57d440..0x57d4dd`, `V read + self-check`)

1. **Candidate collection (`0x57d440..0x57d478`, `V read`):**
   - Initializes `worn[0..6] = {NULL}` (`[ebp - 0x20 .. ebp - 0x08]`) and `total_weight = 0` (`ebx = 0`).
   - For each `i in 0..6`:
     - Calls `item = FUN_0063bde0(inv, slot_table[i].slot)` (`0x57d451`), which reads `[body_slots + slot * 4]`.
     - If `item != NULL` and **`FUN_00629bb0(item, 0x32) != 0`** (`ItemTypes` `0x32` `"armo"`), sets `worn[i] = item` and adds `slot_table[i].weight` to `total_weight`.
     - **Important subtlety (`V read + self-check`):** `FUN_0057d3d0` checks only `FUN_00629bb0(item, 0x32)` (`"armo"`). It does **not** filter out `"armo"` items that are indestructible (`stat 152 > 0`) or already broken (`flag 0x100`) when building `worn[0..6]` and `total_weight`. Thus an indestructible or already-broken armor piece still absorbs its weight share (`weight[i] / total_weight`) of hits on the player; an already-broken armor piece (`durability == 0, maxdur > 0`) still passes `FUN_00629930` and rolls wear RNG, remaining clamped at 0 durability without re-triggering the break event (`0x559f30`).
2. **Two-step RNG slot pick (`0x57d47e..0x57d4dd`, `V read + self-check`):**
   - If `total_weight <= 0`, returns without advancing the defender's RNG.
   - Otherwise, advances `defender`'s unit RNG (`[defender + 0x20..0x24]`) **twice** (`FUN_0045c3e0`):
     1. `start_idx = rand_range(&defender.seed, 7)` (`0x57d48c`, using `[0x00732bc8] = 7`)
     2. `pick = rand_range(&defender.seed, total_weight)` (`0x57d497`)
   - Loops circularly (`esi = start_idx`; each step `esi = (esi + 1) % 7` at `0x57d4ba..0x57d4c9`):
     - If `worn[esi] != NULL`:
       - If `pick < slot_table[esi].weight` (`0x57d4af..0x57d4b1`), breaks out of the loop and calls `FUN_00559e30(ecx = game, edx = defender, worn[esi])` (`0x57d4d8`).
       - Otherwise subtracts `pick -= slot_table[esi].weight` (`0x57d4b3`) and continues.
   - **Mathematical invariance of `start_idx` (`D`):** Because `pick` is drawn uniformly on `{0, 1, ..., total_weight - 1}`, for every fixed `start_idx in 0..6` the circular traversal partitions `{0, ..., total_weight - 1}` into contiguous intervals of exact length `weight[k]` for each occupied `"armo"` slot `k`. Over all `7 * 18 = 126` `(start_idx, pick)` pairs in the standard 6-piece armor setup, Head is chosen in `21/126 = 3/18`, Torso in `35/126 = 5/18`, Shield in `28/126 = 4/18`, and Belt/Boots/Gloves in `14/126 = 2/18` each (`D + V read + self-check`, explicitly enumerated in `tools/emu/verify_paldur.py` Suite 3).
3. **Armor wear roll (`FUN_00559e30`, `V read + self-check`):**
   - Inside `FUN_00559e30(game, defender, worn[esi])`, if `FUN_00629930(worn[esi]) != 0`, advances `defender`'s unit RNG a **third** time (`rand_range(&defender.seed, 100) < 10`, **`10%` chance**) to decrement `stat 72` by `1`.

---

## 6. Item Breakage at `0` Durability and Smith Repair (`V read + self-check`)

### 6.1 Three Distinct `new_dur <= 0` Paths in `FUN_00559e30` (`0x559edb..0x55a07a`)

When the `4%` or `10%` roll succeeds in `FUN_00559e30`, it reads `cur_dur = FUN_00625480(item, 0x48, 0)` and computes `new_dur = cur_dur - 1`:
- **If `new_dur > 0` (`0x559edb`):** Caps `new_dur = min(new_dur, FUN_00625e00(item))`, updates `stat 72 = new_dur` (`FUN_00627260`), sends an `S->C` stat update (`FUN_005531c0` / `FUN_0053d130`), and returns.
- **If `new_dur <= 0` (`cur_dur` was `1` or `0`), `Game.exe` branches on item type (`0x559f15..0x55a07a`, `V read + self-check`):**

1. **Path A — Unbroken Armor / Shield (`is_weap == 0 && (item_data.flags & 0x100) == 0`, `0x559f15..0x559f45`, `V read + self-check`):**
   - Immediately calls **`FUN_0055f850(ecx = game, edx = defender, item)`** (`0x559f3a`), which:
     - Sets `item_data.flags |= 0x0100` (`broken`, via `FUN_006280d0(item, 0x100, 1)` at `0x55f86c`)
     - Deactivates the item in inventory (`FUN_0063d2b0`) and removes its equipped stat list / auras / skills from the player (`FUN_0055c730` at `0x55f88d`, `FUN_0055dbc0` at `0x55f898`)
     - Sets `stat 72 = 0` (`FUN_00627260` at `0x55f8b6`) and sends the stat update packet (`FUN_0053d130`)
     - Updates composite player appearance (`FUN_00663cc0`) and plays sound `9` (`cursor_durability_break`, `FUN_00553380` at `0x55f8f0`).
2. **Path B — Stackable Weapon with `qty > 1` (`is_weap != 0`, `item_data.version >= 1`, `stat 70 > 1`, `0x559f48..0x559fcc`, `V read + self-check`):**
   - For a stackable weapon used in melee (e.g., Javelin, Pilum, Throwing Knife, Throwing Axe, Amazon Javelin), `stat 72` (`durability`) acts as the sub-counter for a single unit of the stack (`stat 70`, `0x46` `quantity`).
   - When `new_dur <= 0` and `qty = FUN_00625480(item, 0x46, 0) > 1`:
     - Decrements `stat 70 = qty - 1` (`0x559f72`) and sends the `stat 70` packet
     - **Resets `stat 72 = FUN_00625e00(item)` (`max_durability`)** (`0x559fa6`) and sends the `stat 72` packet!
   - When `new_dur <= 0` and `qty <= 1` (`0x559fce..0x55a030`):
     - Sets `stat 72 = 0` (`0x559fd3`) and `stat 70 = 0` (`0x55a001`)
     - Calls `FUN_00580310(ecx = game, edx = attacker)` (`0x55a02c`) -> `FUN_00580030` (`0x580030..0x5801d4`), which breaks magic/rare/set/unique/ethereal/socketed/personalized throwing weapons via `FUN_0055f850` (`0x5800ca`), or deletes depleted normal/low/superior stacks and auto-equips a matching replacement stack from inventory (`0x5800d6..0x5801bb`).
3. **Path C — Non-Stackable Weapon (`is_weap != 0`, `stat 70 == 0`, `0x559f48..0x55a07a` + `FUN_00580380`, `V read + self-check`):**
   - Inside `FUN_00559e30` (`0x559fce..0x55a02c` for `version >= 1`, or `0x55a04e` for `version == 0`), `Game.exe` sets `stat 72 = 0` (and calls `FUN_00580310`, whose `FUN_00580030` immediately returns at `0x580058` because the weapon is neither stackable nor throwable). **`FUN_00559e30` does NOT call `FUN_0055f850` on a non-stackable weapon during the hit frame!**
   - Instead, when the player's attack animation or skill action completes (`0x581095`, `0x56ca96`, `0x56cace`, `0x5d8e55`, `0x5d8e9d`, `0x5da8ce`, `0x5dad2a`), `Game.exe` calls **`FUN_00580380(ecx = game, edx = player)`** (`0x580380..0x58044c`), which scans all equipped items (`FUN_0063e020(item) == 3`), finds any `"weap"` (`FUN_00629bb0(item, 0x2d)`) with `FUN_00629930(item) != 0`, `stat 72 <= 0`, and `(flags & 0x100) == 0`, and calls **`FUN_0055f850(game, player, item)`** (`0x58042a`) to set `flag 0x100` (`broken`) and strip weapon stats after the swing finishes (`V read + self-check`).

### 6.2 Smith Repair and Unbreaking (`FUN_005761c0`, `FUN_0055f900`, `FUN_0062e660`, `V read`)

- **Repair price (`FUN_0062efb0` mode `3` -> `FUN_0062e660`, `V read`, `S`):** As traced in `docs/research/price-trace.md` section 6, `FUN_0062e660` (`0x62e660..0x62eb9f`) gates durability repair cost on `FUN_00629930(item)` (`0x62e88a`) and rejects ethereal items (`flag 0x400000` at `0x62e6b0`, returning `0`).
- **Smith repair execution (`FUN_005761c0`, `0x5761c0..0x57629f`, and `FUN_0055f900`, `0x55f900..0x55f9a0`, `V read`):**
  - `FUN_005761c0` rejects ethereal items (`FUN_006280a0(item, 0x400000) != 0` at `0x5761d7..0x5761e5`, branching to `0x57629d`).
  - If `FUN_00629930(item) != 0` and `cur_dur < max_dur`, restores `stat 72 = max_dur` (`0x576216`) and calls **`FUN_0055f900(ecx = game, edx = player, item)`** (`0x576233`).
  - `FUN_0055f900` checks `FUN_006280a0(item, 0x100)` (`0x55f917`); if the item was broken (`flag 0x100`), it clears `flag 0x100` (`FUN_006280d0(item, 0x100, 0)` at `0x55f92a`), re-applies the item's stats to the player (`FUN_0055c730` at `0x55f94a`), and restores the player's composite appearance (`FUN_00663cc0` at `0x55f992`) (`V read`).

---

## 7. Unicorn Verification (`V read + self-check`, `100,128` Cases, `0` Mismatches)

We executed `FUN_00629930`, `FUN_00559e30`, `FUN_00580380`, `FUN_0057d3d0`, `FUN_00535bc0`, and `FUN_0057d4f0` natively inside the Unicorn x86 harness booted on `Game.exe` 1.14d and `Patch_D2.mpq` via the existing upstream emulator helper `skill_tips.boot()` (`seed = 0x50414c44`, `100,128` total cases across 4 suites, `14.08s` runtime, **`0` mismatches**):

| Suite | `Game.exe` Functions Executed (`V read + self-check`) | Cases | Scenarios Verified | Mismatches |
| :--- | :--- | :---: | :--- | :---: |
| **Suite 1** | `FUN_00629930` (`item_has_durability`) | `10,000` | All `659` `Items.txt` rows (`202` `"armo"`, `227` melee `"weap"`, `30` throwable `"weap"`, `49` no-durability `"weap"`, `151` misc) across `max_durability in {0, 1, base, 25, 255}` and `stat 152 (item_indesctructible) in {-1, 0, 1, 5}` | **`0`** |
| **Suite 2** | `FUN_00559e30` (`wear_single_item`) + `FUN_00580380` (`break_zero_dur_weapons`) | `30,000` | Melee weapons (`4%`), armor/shields (`10%`), throwable weapons in melee in Expansion (`10%`) vs Classic (`0%` early exit before RNG step), bows/Phase Blades (`0%`), indestructible (`0%`), `dur > 1 -> min(dur - 1, maxdur)`, `dur == 1` immediate armor break (`flag 0x100`), `dur == 1` stackable `qty > 1 -> (qty - 1, dur = maxdur)` vs `qty == 1 -> FUN_00580310`, and `dur == 1` non-stackable weapon deferred break (`stat 72 = 0` in `FUN_00559e30`, then `flag 0x100` in `FUN_00580380`) | **`0`** |
| **Suite 3** | `FUN_0057d3d0` (`wear_attacker_and_defender`) | `50,126` | Player vs monster attacker/defender, randomized subsets of the 7 body slots `[1, 3, 4, 5, 8, 9, 10]` with weapons vs shields in slots `4`/`5`, exact 2-step defender RNG slot choice (`start_idx = rand(7)`, `pick = rand(total_weight)`), 3rd RNG step `10%` armor wear, `126`-pair partition check (`D + V read + self-check`, exactly `21/126`, `35/126`, `28/126`, `14/126`, `14/126`, `14/126`), and `16,500`-trial oracle-derived 6-piece frequency check (`D`, `slot 1: 0.1671 vs 0.1667`, `slot 3: 0.2798 vs 0.2778`, `slot 5: 0.2230 vs 0.2222`, `slot 8: 0.1122 vs 0.1111`, `slot 9: 0.1096 vs 0.1111`, `slot 10: 0.1084 vs 0.1111`) | **`0`** |
| **Suite 4** | `FUN_00535bc0` (`attacker_active_weapon`) + `FUN_0057d4f0` (`process_hit_record`) | `10,002` | `5,000` cases of Paladin (`class 3`), Barbarian (`class 4`), and Assassin (`class 6`) across all 5 `Skills.txt` `weapsel` values (`0, 1, 2, 3, 4`), `[skill + 0x0c] & 0x2000`, and `[player + 0x38] & 0x100`; plus `2` zero-seed blocked-versus-landed checks (`result_flags = 0x10` vs `0x01`) and `5,000` seeded landed-hit checks using seeded attacker/defender values against Suite 2's RNG oracle | **`0`** |
| **Total** | All 6 combat durability routines | **`100,128`** | Full oracle parity on durability, quantity, `0x100` broken flag, and unit RNG state | **`0`** |

---

## 8. Comparison with `d2d` (`6935cf6`) and `d-idle` (`S`, `V read`)

### 8.1 `d2d` Implementation Summary (`S`)

Contrary to the open question in `docs/community/open-work.md` (*"it has `debug wear` verbs but the loss rule is unknown"*), `d2d` (`6935cf6`) **does** implement automatic combat durability wear in addition to its `debug wear` socket verb (`components/game/fight.hpp:834-836`, `components/game/fight/missiles.cpp:130-180`):
- **When player hits (`S`):** When a player's melee swing lands (`taken.hit == true`), `actor->hits` is incremented (`components/game/fight/strike.cpp:604`, `tick.cpp:515`). In `Fight::melee` (`strike.cpp:607`) and `Fight::tick` (`tick.cpp:588`), `if (actor->hits > 0) wear_weapon()` is called and `actor->hits` is reset to `0`.
- **When player is hit (`S`):** When an attack against the player lands (`taken.hit == true`), `foe.take(taken)` increments `foe.hits` (`fight.hpp:356`). In `Fight::tick` (`tick.cpp:589`), `if (actor->hits > 0)` (on the player's tick) calls `wear_armor(now_ms)` once and resets `actor->hits = 0`.
- **Core wear and slot weights (`S`):** `Fight::wear` (`missiles.cpp:130-151`) uses `chance = (armo || throwable) ? 10 : 4` and decrements `item.durability` when `rng(100) < chance`. `Fight::wear_armor` (`missiles.cpp:160-180`) uses the exact 7-slot weight table `{Head 3, Torso 5, RightArm 4, LeftArm 4, Belt 2, Feet 2, Gloves 2}` filtered by `type_is(..., "armo")` and draws `start_index = rng(7)` and `pick = rng(total)`.

### 8.2 `d2d` Deviations from `Game.exe` 1.14d (`D1`–`D6`)

| ID | Topic | `Game.exe` 1.14d Behavior (`V read + self-check`) | `d2d` (`6935cf6`) Behavior (`S`) | Impact |
| :--- | :--- | :--- | :--- | :--- |
| **`D1`** | **Monster missiles and explosions wearing player armor** | `FUN_0057d3d0` is called **only** from melee hit processor `FUN_0057d4f0` (`0x57d5f6`). Missile hits (`FUN_005ad730`) and radial/explosion damage (`FUN_0056b9c0`) never call `FUN_0057d3d0` or `FUN_00559e30`. | Monster missiles (`ai.hpp:797`), corpse explosions (`ai.cpp:2422`), suicide explosions (`deaths.cpp:172`), and Fire Enchanted death blasts (`deaths.cpp:191`) all call `foe.take(taken)` with `taken.hit = true`, incrementing `foe.hits` and triggering `wear_armor(now_ms)` (`tick.cpp:589`). | **High:** Player armor and shields wear down under ranged/explosion fire in `d2d` when they never wear in 1.14d. |
| **`D2`** | **Non-stackable weapon reaching `0` durability never gets `flag 0x100` (`broken`)** | `FUN_00559e30` sets `stat 72 = 0` on the hit frame, and `FUN_00580380` (`0x580380..0x58044c`) sets `item_data.flags \|= 0x100` (`broken`) via `FUN_0055f850` when the attack animation finishes. | `Fight::wear` (`missiles.cpp:141-146`) sets `item.flags \|= 0x100u` **only** when `armor` hits `0` durability; for a non-stackable weapon (`lines 143-146`), it sets `item.durability = 0` without setting `0x100u` (`missiles.cpp:238` notes `FUN_00580380` is not run). Because `rules::broken(item)` (`rules.hpp:519`) checks `(item.flags & 0x100u) != 0`, a weapon worn to `0` durability in combat is never marked `broken` (only `debug wear` sets `0x100u`). | **High:** In `d2d`, a melee weapon worn to `0` durability by normal combat hits never breaks (`flags & 0x100 == 0`), retains its stats, and keeps being rolled by `wear()`. |
| **`D3`** | **Multiple hits in a single tick collapsed into one wear roll** | `FUN_0057d4f0` calls `FUN_0057d3d0` once **per landed hit record**. If a player takes `N` melee hits in the same frame/tick, `FUN_0057d3d0` runs `N` times. | `foe.take(taken)` increments `foe.hits`, and `Fight::tick` (`tick.cpp:588-589`) runs `if (actor->hits > 0) wear_armor(now_ms); actor->hits = 0;` **at most once per tick**, regardless of whether `actor->hits` was `1` or `> 1`. | **Low–Medium:** Multi-monster simultaneous melee hits in the same tick only roll armor wear once in `d2d`. |
| **`D4`** | **Dual-wielding `wear_weapon` slot choice** | `FUN_00535bc0` selects the weapon that swung using `Skills.txt` `weapsel` (`+0x168`) and swing phase flags (`[skill + 0x0c] & 0x2000`, `[player + 0x38] & 0x100`). | `Fight::wear_weapon` (`missiles.cpp:153-158`) loops over `actor->character.items` and calls `wear(i); return;` on the **first** weapon in `kRightArm` or `kLeftArm`. | **Medium (Barbarian / Assassin):** Off-hand weapon in dual-wield skills never wears in `d2d` until the first weapon breaks (and by `D2`, a non-stackable first weapon never gets `0x100u`). |
| **`D5`** | **Classic throwable melee wear & `has_durability` check** | `FUN_00559e30` checks `FUN_00629930` (`nodurability == 0 && durability > 0 && maxdur != 0 && stat_152 <= 0`) and skips throwable melee wear in Classic (`[game + 0x70] == 0` at `0x559e85`). | `Fight::wear` (`missiles.cpp:137`) checks `item.max_durability > 0 && !indestructible(item) && !broken(item)` without checking `base.no_durability` or Classic vs Expansion for throwables. | **Low:** `d2d` runs Expansion rules (`is_expansion = true`) where generated items with `no_durability` already have `max_durability == 0`. |
| **`D6`** | **RNG stream separation** | Weapon wear advances `[attacker + 0x20..0x24]`; armor slot pick and armor wear advance `[defender + 0x20..0x24]`. | Both `wear_weapon` and `wear_armor` advance the shared `Fight::rng` (`fight.hpp:838`). | **Low:** Affects deterministic RNG lockstep traces only. |

### 8.3 `d-idle` Bot Findings (`B1`–`B5`)

| ID | `d-idle` File / Doc | Current Bot / Doc State (`S`) | Finding from 1.14d Trace (`V read + self-check`) |
| :--- | :--- | :--- | :--- |
| **`B1`** | `docs/research/paladin-act1-recon.md` section 6 row 7 & `docs/community/open-work.md` (`PALDUR`) | Lists durability loss per hit and per block as unknown in both `Game.exe` 1.14d and `d2d`. | Fully resolved: `4%` weapon wear per landed melee hit (`10%` throwable in LoD), `10%` armor wear per landed melee hit across the 7-slot table (`3:5:4:4:2:2:2`), `0%` on block (`result_flags` bit `0` cleared at `0x57ed16`), and Smite wears the weapon (`4%`) rather than the shield. |
| **`B2`** | `internal/town/chores.go:416-435` (`needsRepair`) & `internal/env/tuning_town.go:35` | `needsRepair` triggers a smith visit when any equipped item has `(flags & 0x100) != 0` (`broken`) or `MaxDurability > 0 && Durability * 100 / MaxDurability <= RepairDurabilityPct` (default `25%`), skipping `Ethereal` items (`chores.go:420`). | Matches 1.14d (`FUN_005761c0` skips ethereal `0x400000`, repairs `stat 72 < max_dur`, and clears `0x100` via `FUN_0055f900`). Note that in `d2d`, because of `D2`, a weapon worn to `0` durability by combat hits has `Durability == 0` (`0% <= 25%`) even though `(flags & 0x100) == 0`, so `needsRepair`'s percentage check (`<= 25%`) catches it even when `broken` (`0x100`) is missing! |
| **`B3`** | `internal/town/chores.go:416-435` (`needsRepair` indestructible check) | `needsRepair` checks `it.Ethereal` and `it.MaxDurability <= 0`, without checking `stat 152` (`item_indesctructible`) if `MaxDurability > 0`. | In 1.14d (`FUN_00629930`), an item with `stat 152 > 0` (`item_indesctructible`, e.g. Zod-socketed or unique with `indestructible` property that still has a nonzero base `MaxDurability`) never loses durability (`stat 72` stays at `MaxDurability`), so `Durability * 100 / MaxDurability == 100% > 25%` and it will not false-trigger `needsRepair` unless spawned damaged. |
| **`B4`** | `docs/forkdbg.md:62` & `docs/forkcheats.md:60` (`debug wear` verb) | Documents `debug wear [slot] [durability]` as setting durability (`0` sets broken `0x100`) for town repair drills. | Because of `D2` in `d2d`, `debug wear <weapon_slot> 0` sets `item.flags \|= 0x100u`, whereas natural combat wear to `0` in `d2d` leaves `item.flags & 0x100u == 0`. |
| **`B5`** | Paladin Zealot / Smiter repair cadence (`docs/research/paladin-act1-recon.md`) | Notes fast-attacking melee builds (Zeal) face weapon durability pressure. | At `4%` (averaging `1/25`, `D`) per landed melee hit, a weapon with `24` max durability (e.g., Scepter `scp` `24`, War Scepter `wsp` `70`, Flail `fla` `30`) loses durability averaging `1` point per `25` landed hits (an expected `~450` landed hits for `18` losses to reach the `25%` repair threshold of `6/24` on a Scepter, `D`). |

---

## 9. Maintainer Live Verification Recipe & Unverified (`U`) Limits

### 9.1 Proposed Live `d2d` Probe Recipe (for Maintainers with Lab Access, `U`)

The following steps represent proposed maintainer live verification recipes (`U`, not executed offline) to observe deviations `D1` and `D2` in a live `d2d` instance via the `debug` socket without modifying game memory:
1. **Proposed Probe for `D2` (`weapon at 0 durability missing 0x100`, `U`):**
   - Set the player's melee weapon to `1` durability via `debug wear 4 1` (or `rarm`).
   - Melee-attack a monster until the weapon's durability drops from `1` to `0` (averaging `~25` landed hits at `4%`, `D`).
   - Expected live observation on `d2d` (`S`, unverified/expected from `missiles.cpp:141-146, 238`): inspect equipped weapon's `durability` and `flags`; `durability` drops to `0`, while `(flags & 0x100)` remains `0` (`broken == false`).
2. **Proposed Probe for `D1` (`monster missiles wearing armor`, `U`):**
   - Stand against a ranged-only monster (e.g., Quill Rat or Skeleton Archer) and let monster missiles hit the player without any melee contact.
   - Expected live observation on `d2d` (`S`, unverified/expected from `tick.cpp:589` and `ai.hpp:797`): equipped armor durability decrements (`10%` wear roll, `D`, per tick in which at least one missile lands).

### 9.2 Unverified (`U`) Limits

- **`U` (Live 1.14d Battle.net / single-player process capture):** All 1.14d results in this document are verified by static disassembly (`V read`) and native execution of the original 1.14d `Game.exe` functions in Unicorn across `100,128` cases (`V read + self-check`), not by attaching a live debugger to a running Windows `Game.exe` window.
