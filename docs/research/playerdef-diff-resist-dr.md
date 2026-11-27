# Player defence differentials: magic resist (D4), resist clamp order (D5), and DR order (D6) (`PLAYERDIFF`)

> **Status:** current (2026-11-26, contributor work item [#24 (`PLAYERDIFF`)](https://github.com/shatteredeagle/d-idle/issues/24), D4/D5/D6 group; follows `docs/research/playerdef-trace.md` and `docs/research/playerdef-diff-20261104.md`). Differential of real `game.exe` 1.14d (`V`) against a Python port of upstream `psi29a/d2decomped` `5a81bcc`; compiled C++ `d2d` probe is still unverified (`U`).
> **Author:** Chuck (@chuckado) with Antigravity
> **Environment:** `game.exe` 1.14d (`SHA256 631066c1649c4ea9ffe48bf97e24c00bca1f7a6759c21150f1a79982589adaaf`, image base `0x00400000`); upstream `psi29a/d2decomped` `tools/emu/emu.py` at commit `5a81bcc82f7a3be309ed036021f8fd8c9cf61af9` (`5a81bcc`).
> **Tags:**
> - **V** (`V run (game)`) — `game.exe` 1.14d's own `FUN_0057c1e0` (invoking `FUN_0057bf80` and `FUN_0057be00` natively on a type-0 player defender) executed under Unicorn using upstream `d2decomped`'s `tools/emu/emu.py` (`5a81bcc`) (`tools/emu/playerdef_resist_dr.py`, 100,000 cases × 2 seeds = 200,000 cases).
> - **V read** — read in upstream `psi29a/d2decomped` commit `5a81bcc` (`file:line` cited) or `game.exe` 1.14d disassembly (`FUN_xxxxxxxx`).
> - **Port (d2d)** — line-for-line Python port of `d2decomped` `5a81bcc` (`panel_stats`, `update_fighters`, `make_fighter`, `reduce_fine_by`, `elemental_fine`, `resisted_fine`, `monster_blow`), **not** a compiled C++ `d2d` probe.
> - **D** — patched 1.14d `DifficultyLevels.txt` `ResistPenalty` (`0`, `-40`, `-100` for Normal/Nightmare/Hell in Expansion).
> - **S** — secondary source (`docs/research/playerdef-trace.md`, `docs/research/playerdef-diff-20261104.md`).
> - **U** — unverified (compiled C++ `d2d` probe, live game, and stubbed absorb/poison paths).

---

## 1. Verdict

| ID | Topic | `game.exe` 1.14d (`V`) | `d2d` `5a81bcc` (`V read` + `Port (d2d)`) | Cases (2 seeds) | Before diff | After diff | Verdict |
|---|---|---|---|---:|---:|---:|---|
| **D4** | Player magic resist (stats `37`/`38`) | `FUN_0057be00` reads `stat 0x25` (37) and max `stat 0x26` (38), applies **no** difficulty penalty, clamps to `[-100, min(75 + stat38, 95)]`, and reduces magic (`rec+0x20`) after MDR (`stat 35`). | `combat/combat.cpp:1003` hardcodes `res = 0, resist = false` for `type == 4` (magic); `Fighter::res` has only 4 slots (`fire/lightning/cold/poison`). | 60,000 | **43,410** (`21,578` / `21,832`) | **0** | **Open deviation** (`V` game, `Port` d2d, compiled d2d `U`) |
| **D5** | Resist clamp order (gear + difficulty + states) | `FUN_0057be00` sums all sources (`gear + states + penalty`) in one stat read and clamps **once** to `[-100, min(75 + max_stat, 95)]`. | `character.cpp:218-219` clamps `gear + penalty` to `[-100, min(75 + gear_max, 95)]` first; `strike.cpp:288-290` then adds state/aura/curse resists with only an upper cap and **no `-100` floor** (`resisted_fine` at `combat.cpp:263` also has no `-100` floor). | 80,000 | **19,540** (`9,764` / `9,776`) | **0** | **Open deviation** (`V` game, `Port` d2d, compiled d2d `U`) |
| **D6** | DR order of stats `34`/`35`/`36` | `FUN_0057bf80` subtracts flat (`stat 34` for physical, `stat 35` for elemental/magic) **before** percent (`stat 36` capped at `50%` and floored at `-100%` for players; elemental resist). | `combat.cpp:139` caps `dr_pct` at `50`; `combat.cpp:269-274` (`reduce_fine_by`) subtracts `flat * 256` first, then applies `clamp(pct, -100, 100)`; `combat.cpp:314-319` (`elemental_fine`) subtracts `mdr * 256` before resist. | 60,000 | **0** (`0` / `0`) | **0** | **Closed in `5a81bcc`** (`V` game, `Port` d2d, compiled d2d `U`) |

---

## 2. Upstream `d2d` Code Paths (`5a81bcc`, `V read`)

| Group | Upstream Location (`5a81bcc`) | Behavior | Tag |
|---|---|---|---|
| **D4** | `components/rules/combat/combat.cpp:1003` (`monster_blow`) | Passes `type != 4 ? defender.res[type] : 0` and `resist = (type != 4)` to `elemental_fine(...)`. `Fighter` (`combat.hpp`) carries no magic-resist field (`stat 37`/`38` ignored). | V read |
| **D5** | `components/game/character/character.cpp:217-219` (`panel_stats`) | Clamps `sum[kRes[i]] + penalty` into `[-100, min(75 + sum[kRes[i] + 1], 95)]` before aura/curse/potion states are added, discarding overcapped gear resist and clamping early at `-100`. | V read |
| **D5** | `components/game/fight/strike.cpp:288-290, 301` (`update_fighters`) | Adds `st_sum[kRes[k]]` onto the already-clamped `actor->player_combat.res[k]` with only an upper `std::min(..., 95)` cap; the `-100` floor at line 301 is applied only to the display `panel.res`, leaving `player_combat.res[k]` able to drop below `-100` (e.g. `-100` Hell + `-50` Lower Resist = `-150`). | V read |
| **D5** | `components/rules/combat/combat.cpp:263-267` (`resisted_fine`) | Computes `fine * (100 - res) / 100` without clamping `res >= -100`, so `player_combat.res < -100` amplifies damage beyond `200%`. | V read |
| **D6** | `components/rules/combat/combat.cpp:139, 269-274, 314-319` | `make_fighter` caps `dr_pct = min(stat(kDamageResist), 50)`. `reduce_fine_by` subtracts `flat_fine` first, then applies `mul_div(fine, 100 - clamp(percent, -100, 100), 100)`. `elemental_fine` subtracts `mdr_fine` first, then applies `resisted_fine`. Matches `game.exe` 1.14d. | V read |

---

## 3. Native Differential Results (`tools/emu/playerdef_resist_dr.py`)

### 3.1 Reproduction Command & Harness Details

Run from this repo root with upstream `psi29a/d2decomped` (`5a81bcc82f7a3be309ed036021f8fd8c9cf61af9`, `tools/emu/emu.py`) on `PYTHONPATH` and your 1.14d MPQ directory in `D2_MPQ_DIR` (tested on `Game.exe` 1.14d `SHA256 631066c1649c4ea9ffe48bf97e24c00bca1f7a6759c21150f1a79982589adaaf`):

```sh
PYTHONPATH=<d2decomped>/tools/emu D2_MPQ_DIR=<mpq_dir> \
    python tools/emu/playerdef_resist_dr.py 100000 20261104 --exe <path/to/Game.exe>

PYTHONPATH=<d2decomped>/tools/emu D2_MPQ_DIR=<mpq_dir> \
    python tools/emu/playerdef_resist_dr.py 100000 20261105 --exe <path/to/Game.exe>
```

- **Native calls & stubbed hooks:** Calls `game.exe` 1.14d's `FUN_0057c1e0` natively on a type-0 player defender (`game+0x70 = 1` Expansion, `game+0x6d = diff` in `0..2`), invoking `FUN_0057bf80` and `FUN_0057be00`. Hooks are stubbed: `FUN_00625480` serves only the defender's test stats (`0x22`..`0x2c`, i.e., stats `34`..`44`), `FUN_00611d30` returns a row with `ResistPenalty` `(0, -40, -100)` at `+0x00`, `FUN_0057c060` returns `100`, and `FUN_005c0c30` plus state/absorb helpers are stubbed to `0`, so gear absorb, Bone/Cyclone Armor, Energy Shield, and poison are untested (`U`).
- **Case split (100,000 cases per seed):** `30,000` D4 (magic `rec+0x20`, `diff 0..2`, `mdr`, `stat37 in [-120..120]`, `stat38 in [0..25]`), `40,000` D5 (fire/lightning/cold `rec+0x10/0x1c/0x24`, `diff 0..2`, `mdr`, `gear_res in [-30..160]`, `state_res in [-120..100]`, `gear_max`, `state_max`), and `30,000` D6 (combined physical `rec+0x08` in 256ths + fire `rec+0x10`, `diff 0..2`, `stat34`, `stat35`, `stat36 in [-120..100]`, `stat39`).
- **Exit code:** Exits `0` when all `after` arms have `0` mismatches against `game.exe`, and `1` if any `after` mismatch occurs.

### 3.2 Seed `20261104` (100,000 cases, `22.97s`, `V` game vs `Port (d2d)`)

| Suite | Cases | Before diff (`5a81bcc` port) | After diff (proposed fix port) | Breakdown of `before` diffs |
|---|---:|---:|---:|---|
| **D4** (`magic_resist`) | 30,000 | 21,578 | 0 | All non-zero `stat 37` cases where `field > mdr` |
| **D5** (`resist_clamp`) | 40,000 | 9,764 | 0 | `overcap = 3,863`, `underflow < -100 = 4,772`, `early_floor = 671`, `state_max`-after-overcap = `458` |
| **D6** (`dr_order`) | 30,000 | 0 | 0 | `0` diffs — flat-before-percent and `50%` / `-100%` bounds already match |
| **TOTAL** | **100,000** | **31,342** | **0** | |

First differing cases (seed `20261104`):
- **D4** (`case 0`): `diff=2` (Hell, `-100`), `field=20` (`5120`), `mdr=0`, `stat37=100`, `stat38=0` -> `game=(1280, 1280)` (`75%` cap, no Hell penalty), `before=(5120, 5120)` (`0%` resist), `after=(1280, 1280)`.
- **D5** (`case 1`): `diff=2` (Hell, `-100`), `cold`, `field=486` (`124416`), `mdr=0`, `gear_res=-20`, `state_res=-100`, `gear_max=20`, `state_max=0` -> `game=(248832, 248832)` (`-100%` floor = `200%` dmg), `before=(373248, 373248)` (`-100 + -100 = -200%` = `300%` dmg, no `-100` floor!), `after=(248832, 248832)`.

### 3.3 Seed `20261105` (100,000 cases, `21.73s`, `V` game vs `Port (d2d)`)

| Suite | Cases | Before diff (`5a81bcc` port) | After diff (proposed fix port) | Breakdown of `before` diffs |
|---|---:|---:|---:|---|
| **D4** (`magic_resist`) | 30,000 | 21,832 | 0 | All non-zero `stat 37` cases where `field > mdr` |
| **D5** (`resist_clamp`) | 40,000 | 9,776 | 0 | `overcap = 3,888`, `underflow < -100 = 4,756`, `early_floor = 635`, `state_max`-after-overcap = `497` |
| **D6** (`dr_order`) | 30,000 | 0 | 0 | `0` diffs |
| **TOTAL** | **100,000** | **31,608** | **0** | |

First differing cases (seed `20261105`):
- **D4** (`case 0`): `diff=0`, `field=5` (`1280`), `mdr=2` (`512`), `stat37=25`, `stat38=10` -> `game=(576, 576)` (`(1280 - 512) * 75 / 100`), `before=(768, 768)`, `after=(576, 576)`.
- **D5** (`case 4`): `diff=1` (Nightmare, `-40`), `lightning`, `field=369` (`94464`), `mdr=12` (`3072`), `gear_res=0`, `state_res=-100`, `gear_max=5`, `state_max=10` -> `game=(182784, 182784)` (`-100%` floor), `before=(219340, 219340)` (`-40 - 100 = -140%`, no `-100` floor!), `after=(182784, 182784)`.

---

## 4. Proposed Upstream `d2d` Fixes (Proposals Only)

1. **D4 (`magic_resist`):** Add `magic_res` (clamped `[-100, min(75 + stat(kMaxMagicResist), 95)]`, no difficulty penalty) to `Fighter` in `make_fighter` (`combat/combat.cpp:140`), and pass `defender.magic_res` with `resist = true` for `type == 4` in `monster_blow` (`combat/combat.cpp:1003`).
   - Proposed commit message: `rules/combat: apply player magic resist (stats 37/38) in monster_blow`
2. **D5 (`resist_clamp`):** Carry unclamped `gear_res` and `gear_max` (or compute `panel_stats` with `st_sum` included) so `update_fighters` (`fight/strike.cpp:288-290`) sums `gear_res + state_res + penalty` and clamps **once** into `[-100, min(75 + gear_max + state_max, 95)]` (and floor `res` at `-100` in `strike.cpp:290`).
   - Proposed commit message: `fight: clamp player elemental resists once after aura/curse states and floor at -100`
3. **D6 (`dr_order`):** No change needed; already matches `game.exe` 1.14d in `5a81bcc`.

---

## 5. Bot Findings

No new bot findings (`B1`–`B4` in `docs/research/playerdef-trace.md` are unchanged).

---

## 6. Live Drill Recipe (`U`)

Unchanged from `docs/research/playerdef-trace.md` section 7 (`R4` for D4/D5, `R5` for D6; `U`, maintainer lab only).

---

## 7. Limits (`U`)

1. **Compiled `d2d` probe not run (`U`):** The `d2d` arm in `tools/emu/playerdef_resist_dr.py` is a line-for-line Python port of `5a81bcc`, not a compiled C++ `d2d` binary.
2. **Stubbed hooks (`U`):** Gear absorb (`stats 142..149`), Bone Armor, Cyclone Armor, Energy Shield, poison (`rec+0x28`), Classic difficulty penalties (`0/-20/-50`), and monster-pierce stats are outside this run (`FUN_005c0c30` stubbed to `0`).
3. **D2 (block recovery) not covered:** Separate PR per Issue #24 instructions.
