# Monster hits monster: 256ths life differential, game.exe 1.14d vs d2d (`TICKMVM`)

> **Status:** current (2026-11-20, contributor work item [#26 (`TICKMVM`)](https://github.com/shatteredeagle/d-idle/issues/26), follows `docs/research/tickcpp-diff-monster-vs-monster.md`).
> **Author:** Chuck (@chuckado) with Antigravity
> **Tags:**
> - **V run (game)** — `game.exe` 1.14d's own `FUN_0057b420` and `FUN_0057c6c0` (invoking `FUN_0057c1e0` and stat 6 write via `FUN_00627260`) executed natively under Unicorn using upstream `d2decomped`'s `tools/emu/emu.py` (`tools/emu/tickcpp_mvm.py`, 100,000 cases × 2 seeds = 200,000 cases).
> - **V read** — read in upstream `psi29a/d2decomped` commit `2b7ec10` (`file:line` cited) or `game.exe` 1.14d disassembly (`FUN_xxxxxxxx`).
> - **Port (d2d)** — line-for-line Python port of `d2decomped` `2b7ec10` (`monster_blow` -> `Foe::take` -> `Fight::take_blows` -> `hurt`), **not** a compiled C++ `d2d` binary; see Limits.
> - **S** — secondary source (`docs/research/tickcpp-diff-monster-vs-monster.md`).
> - **I** — inference.
> - **U** — unverified (live game or compiled C++ `d2d` binary).

---

## 1. Verdict

1. **`game.exe` 1.14d keeps the 256ths fraction of life when a monster hits a monster (`V run (game)`).**
   Running `FUN_0057b420` (monster attacker, `EDX = 1`) into `FUN_0057c6c0` (`flag = 1`, type-1 monster defender, calling `FUN_0057c1e0` natively) across 200,000 cases (`100,000` × 2 seeds) shows that `game.exe` subtracts the full 256ths reduced total (`rec + 0x4c`) directly from the monster defender's stat 6 and writes the fractional result back via `FUN_00627260` (zeroing stat 6 when the post-hit life falls below `0x100`). Across integer-start cases alone, `18,811` (seed `20261104`) and `18,891` (seed `20261105`) end with non-zero 256ths in stat 6.
2. **Upstream `d2d` (`2b7ec10`) STILL drops the 256ths fraction for monster-vs-monster hits (`V read`, `D1`).**
   Since `tickcpp-diff-monster-vs-monster.md` (which cited `fight/tick.cpp:344-345`), the monster-vs-monster apply loop was refactored into `Fight::tick` (`components/game/fight/tick.cpp:485-499`) calling `Fight::take_blows` (`components/game/fight/strike.cpp:614-621`). While `d2d` now carries `frac` for pets (`tick.cpp:551`), mercs (`tick.cpp:745-752`), and player-vs-monster (`ai.cpp:3060`), **`Fight::take_blows` (`strike.cpp:616`) still passes `blow.damage << 8` to `soak(...)`, dropping `blow.frac`**.
3. **Differential counts (`V run (game)` vs `Port (d2d)`):**
   - **Before (`2b7ec10` `strike.cpp:616`):** `48,924 / 100,000` cases differ in defender life on seed `20261104` (`1,445` differ in dead-vs-alive; mean `|err| = 0.2490 HP`), and `49,174 / 100,000` differ on seed `20261105` (`1,465` dead-vs-alive; mean `|err| = 0.2506 HP`). Attacker RNG seed differs in `0 / 200,000` cases.
   - **After (carrying `+ blow.frac` in `strike.cpp:616`):** **`0 / 100,000`** life mismatches, **`0`** dead-vs-alive mismatches, and **`0`** seed mismatches on both seeds (`0 / 200,000` total).

---

## 2. Upstream `d2d` Code Path (`2b7ec10`, `V read`)

| Step | Upstream Location (`2b7ec10`) | Behavior | Tag |
|---|---|---|---|
| Roll & reduce | `components/rules/combat.cpp:197-202, 718-799` (`reduce_fine_by`, `monster_blow`) | Rolls `low * 256 + rng((high - low) * 256)` (or `low * 256` when `exact`), reduces in 256ths (`flat * 256` first, then `mul_div(fine, 100 - clamp(pct, -100, 100), 100)`), and stores `taken.damage = total >> 8`, `taken.frac = total & 255`. | V read |
| Queue blow | `components/game/ai.hpp:660-676` (`Foe::take`) | Constructs `Blow` preserving both `taken.damage` and `taken.frac` (`ai.hpp:665`), pushing onto `foe.blows`. | V read |
| Monster-vs-monster dispatch | `components/game/fight/tick.cpp:485-499` (`Fight::tick`) | Iterates `i` from `allies` to `foes.size()` (monster defenders struck by monsters/pets) and calls `take_blows(struck, foes[i], now_ms)` (`tick.cpp:491`). | V read |
| **Bug site** | `components/game/fight/strike.cpp:614-621` (`Fight::take_blows`) | Calls `soak(struck, blow.physical << 8, blow.damage << 8)` (`strike.cpp:616`), then `hurt(..., taken >> 8, now_ms, taken & 255, ...)`. Because `blow.frac` is omitted, `taken & 255` is always `0` when `armor_pool == 0`. | V read |
| Life update | `components/game/ai.cpp:3032-3044` (`hurt`) and `ai.hpp:523-527` | Stores life as `hit_points * 256 - life_frac` (`life_frac` = sub-point damage deficit in `0..255`). Adds `fraction` to `life_frac`, carries `life_frac >> 8` into `damage`, and kills when `hit_points < 1 \|\| (hit_points == 1 && life_frac > 0)` (equivalent to `life_fine < 0x100`). | V read |
| Sibling paths (fixed) | `tick.cpp:551` (`pet_hurt`), `tick.cpp:745-752` (merc), `ai.cpp:3060` (`blow_hurt`) | All three sibling paths already carry `blow.frac` / `side[1].frac` into `life_frac`. Only `Fight::take_blows` still omits it. | V read |

---

## 3. Native Differential Results (`tools/emu/tickcpp_mvm.py`)

### 3.1 Reproduction Command

Run from this repo root with upstream `psi29a/d2decomped` (`2b7ec10`) on `PYTHONPATH` and your 1.14d MPQ directory in `D2_MPQ_DIR`:

```sh
PYTHONPATH=<d2decomped>/tools/emu D2_MPQ_DIR=<mpq_dir> \
    python tools/emu/tickcpp_mvm.py 100000 20261104 --exe <path/to/Game.exe>

PYTHONPATH=<d2decomped>/tools/emu D2_MPQ_DIR=<mpq_dir> \
    python tools/emu/tickcpp_mvm.py 100000 20261105 --exe <path/to/Game.exe>
```

The harness executes `game.exe` 1.14d's `FUN_0057b420` (attacker type 1, `EDX = 1`, stats 21/22 from 8 Act 1 monster classes) and `FUN_0057c6c0` (defender type 1, `flag = 1`, invoking `FUN_0057c1e0` and writing stat 6 via `FUN_00627260`). Starting defender life mixes near-death (`0x100..(mx+2)*256`, 25%), fractional (`0x100..200*256`, 40%), and integer (`1..200 * 256`, 35%) values.

### 3.2 Seed `20261104` (100,000 cases, `58.41s`, `V run (game)` vs `Port (d2d)`)

| Class | Cases | Before: life diff | Before: dead diff | Before: seed diff | Before: mean \|err\| (HP) | After: life diff | After: dead diff | After: seed diff |
|---|---:|---:|---:|---:|---:|---:|---:|---:|
| `plain` | 20,000 | 17,512 | 600 | 0 | 0.4582 | 0 | 0 | 0 |
| `pct` | 20,000 | 15,471 | 496 | 0 | 0.3973 | 0 | 0 | 0 |
| `flat` | 20,000 | 6,861 | 188 | 0 | 0.1792 | 0 | 0 | 0 |
| `both` | 20,000 | 5,969 | 132 | 0 | 0.1319 | 0 | 0 | 0 |
| `exact` | 10,000 | 0 | 0 | 0 | 0.0000 | 0 | 0 | 0 |
| `exactred` | 10,000 | 3,111 | 29 | 0 | 0.1572 | 0 | 0 | 0 |
| **TOTAL** | **100,000** | **48,924** | **1,445** | **0** | **0.2490** | **0** | **0** | **0** |

Integer-start cases ending with non-zero 256ths in `game.exe` stat 6: **`18,811`**.

First differing case per class (seed `20261104`):
- `plain`: case `0` (`fallen1`, `1..2`, `pct=0`, `flat=0`, `init_life=13824` = 54.000 HP) -> `game_life=13450` (`rec_tot=374`), `d2d_before=13568` (dropped `118/256`), `d2d_after=13450`.
- `pct`: case `20001` (`zombie1`, `1..3`, `pct=10`, `flat=0`, `init_life=32488`) -> `game_life=32121` (`rec_tot=367`), `d2d_before=32232`, `d2d_after=32121`.
- `flat`: case `40002` (`skeleton1`, `1..3`, `pct=0`, `flat=2`, `init_life=15142`) -> `game_life=15013` (`rec_tot=129`), `d2d_before=15142` (0 damage applied!), `d2d_after=15013`.
- `both`: case `60001` (`zombie1`, `1..3`, `pct=-50`, `flat=2`, `init_life=18418`) -> `game_life=18180` (`rec_tot=238`), `d2d_before=18418`, `d2d_after=18180`.
- `exactred`: case `90006` (`x0_40`, `3..3`, `pct=-50`, `flat=0`, `exact=1`, `init_life=19864`) -> `game_life=18712` (`rec_tot=1152`), `d2d_before=18840`, `d2d_after=18712`.

### 3.3 Seed `20261105` (100,000 cases, `58.32s`, `V run (game)` vs `Port (d2d)`)

| Class | Cases | Before: life diff | Before: dead diff | Before: seed diff | Before: mean \|err\| (HP) | After: life diff | After: dead diff | After: seed diff |
|---|---:|---:|---:|---:|---:|---:|---:|---:|
| `plain` | 20,000 | 17,562 | 604 | 0 | 0.4609 | 0 | 0 | 0 |
| `pct` | 20,000 | 15,433 | 478 | 0 | 0.3986 | 0 | 0 | 0 |
| `flat` | 20,000 | 6,989 | 208 | 0 | 0.1811 | 0 | 0 | 0 |
| `both` | 20,000 | 6,080 | 128 | 0 | 0.1337 | 0 | 0 | 0 |
| `exact` | 10,000 | 0 | 0 | 0 | 0.0000 | 0 | 0 | 0 |
| `exactred` | 10,000 | 3,110 | 47 | 0 | 0.1578 | 0 | 0 | 0 |
| **TOTAL** | **100,000** | **49,174** | **1,465** | **0** | **0.2506** | **0** | **0** | **0** |

Integer-start cases ending with non-zero 256ths in `game.exe` stat 6: **`18,891`**.

First differing case per class (seed `20261105`):
- `plain`: case `0` (`fallen1`, `1..2`, `init_life=953`) -> `game_life=483` (`rec_tot=470`), `d2d_before=697`, `d2d_after=483`.
- `pct`: case `20000` (`fallen1`, `1..2`, `pct=-50`, `init_life=42509`) -> `game_life=41834` (`rec_tot=675`), `d2d_before=41997`, `d2d_after=41834`.
- `flat`: case `40007` (`andariel`, `6..19`, `flat=12`, `init_life=20736`) -> `game_life=19446` (`rec_tot=1290`), `d2d_before=19456`, `d2d_after=19446`.
- `both`: case `60000` (`fallen1`, `1..2`, `pct=-50`, `flat=1`, `init_life=340`) -> `game_life=0` (`rec_tot=148`; `340 - 148 = 192 < 256` -> dead!), `d2d_before=340` (alive!), `d2d_after=0`.
- `exactred`: case `90001` (`x0_40`, `20..20`, `pct=76`, `flat=8`, `exact=1`, `init_life=7170`) -> `game_life=6433` (`rec_tot=737`), `d2d_before=6658`, `d2d_after=6433`.

---

## 4. Deviations (`d2d` `2b7ec10` vs `game.exe` 1.14d)

| ID | Rule | `game.exe` 1.14d | `d2d` (`2b7ec10`) | Tag | Impact |
|---|---|---|---|---|---|
| **D1** | Monster-vs-monster 256ths life carry | `FUN_0057c6c0` subtracts the full 256ths total (`rec + 0x4c`) from defender stat 6 and kills when `< 0x100`. | `Fight::take_blows` (`components/game/fight/strike.cpp:616`) passes `blow.damage << 8` to `soak(...)`, dropping `blow.frac`. | V run (game), V read + Port (d2d) | ~49% of monster-vs-monster blows leave the defender with up to `255/256` extra HP (mean `~0.25 HP` overall, `~0.46 HP` on plain blows), and ~1.45% of near-death/mixed blows leave a monster alive when `game.exe` kills it. |
| **D2** | Physical fraction into `soak` (`armor_pool`) | `FUN_0057c1e0` runs Bone Armor absorb (`FUN_005c6370`) on the 256ths physical field (`rec + 8`). | `Fight::take_blows` (`strike.cpp:616`) passes `blow.physical << 8` (truncated whole points) instead of `d2d::rules::physical_now(blow)` (`combat.cpp:210`). | V read (d2d), S (`w3_run.py`) | Only affects monsters with `armor_pool > 0` (Bone Armor); not exercised in this no-absorb run (`U`). |

---

## 5. Proposed Upstream `d2d` Fix (Proposal Only)

In `components/game/fight/strike.cpp:616` (`Fight::take_blows`):

```cpp
// Before (2b7ec10):
const int taken = soak(struck, blow.physical << 8, blow.damage << 8);

// Proposed:
const int taken = soak(struck, int(d2d::rules::physical_now(blow)), (blow.damage << 8) + blow.frac);
```

- **Proposed commit message:** `fight: carry 256ths fraction in Fight::take_blows for monster-vs-monster hits`
- **Verification before upstream PR:** Maintainer should build a compiled `d2d` probe calling `rules::monster_blow` + `Fight::take_blows` / `hurt`, confirm `before > 0` and `after = 0` against `tools/emu/tickcpp_mvm.py`, and re-run deterministic scenario hashes (scenarios with converted/charmed/confuse monsters will change slightly as monsters lose the carried 256ths).

---

## 6. Bot Findings

None (`B` list empty). `d-idle` does not model monster-vs-monster fractional HP.

---

## 7. Live Drill Recipe (`U`)

Unchanged from `docs/research/tickcpp-diff-monster-vs-monster.md` (`U`, maintainer lab only):
- Spawn a Necromancer skeleton (or converted monster) attacking a Fallen with known max life; read the defender's stat 6 before and after one landed melee blow with the lab debug verbs. Expect non-zero `stat6 & 255` in `game.exe` 1.14d and matching `life_frac` in `d2d` after the fix.

---

## 8. Limits (`U`)

1. **Compiled `d2d` binary not run (`U`):** The `d2d` comparison in `tools/emu/tickcpp_mvm.py` is a line-for-line Python port of `2b7ec10` (`combat.cpp`, `ai.hpp`, `strike.cpp`, `ai.cpp`), not a compiled C++ binary (no C++ compiler installed on this contributor host).
2. **Absorb / elemental / poison / prime-evil pair scaling not exercised (`U`):** `FUN_005c0c30` (absorb events) is stubbed to 0 and only physical damage (`rec + 8`) is populated, so `D2` (`armor_pool > 0` with `physical_now`) and elemental/poison monster-vs-monster hits are outside this run.
3. **Monster-vs-monster caller above `FUN_0057b420` (`U`):** `FUN_0057b420` is called directly with `EDX = 1` (monster attacker) as in `w3_run.py`, rather than from the full monster AI attack dispatch.
