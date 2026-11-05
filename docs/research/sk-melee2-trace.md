# The player's plain Attack on the REAL d2d build: game.exe 1.14d vs d2d's `player_blow` (SKMELEE2)

> status: current (2026-11-01, agent SKMELEE2). Follows docs/research/sk-melee-trace.md (SKMELEE), whose deviations were proven only against a port of d2d's rule. Offline: no game, no d2d slot, nothing pushed or posted. Tags: **V read** = read in game.exe's disassembly (`disasm.py`) or in d2d's code at file:line; **V run** = game.exe's own function run under unicorn, or d2d's real code run in a test binary; **D** = 1.14d tables; **S** = an upstream doc or an earlier trace; **I** = inference; **U** = unverified. d2d = upstream main 72132b99 (a scratch `git worktree` on the worker host). No game code is pasted here: function tags and short formulas only.

## Summary

1. **A real-build differential now exists.** `tools/emu/sk_melee2_test.cpp` (as `tests/test_melee.cpp` on a scratch checkout of upstream 72132b99) runs d2d's REAL `rules::make_fighter` and `rules::player_blow`; `tools/emu/sk_melee2.py` writes the cases (sk_melee.py's grids, draw for draw), runs game.exe's FUN_0057d9b0 / FUN_0057b660 / FUN_0057b420 on them and compares. Seeds 7 and 1, 100,000 cases a part (V run).
2. **D1, D2, D4, D5, D6 are REAL on the build; none was a port artifact.** The real build reproduces the port's counts exactly where the parts overlap (to-hit 44 / 36; weapon whole points 0; barehanded and 256ths differ as the port said).
3. **New D8 (formula V run, aggregation I):** FUN_0057b420 adds the player's stats 17 / 18 (enhanced damage) to the one percentage; d2d's plain blow ignores everything in `sum` 17 / 18 that is not on the weapon (its own test asserts "off the weapon: no effect"). Off-weapon enhanced damage (an armor or helm with ED, jewels socketed in armor) is lost on d2d.
4. **D7 narrowed, still U.** V read: the hand items are toggled by FUN_00535d10 / FUN_00535e20: the weapon the skill uses (FUN_00535bc0) has its damage-related stats merged into the player's list; the other hand's item is flagged 0x40000000 and its damage-related stats are taken back out (FUN_00627910 -> FUN_00627860 / FUN_00627a40). Whether the merged weapon list carries ED inside stats 21 / 22 (op 13 materialised) or as stats 17 / 18 (which FUN_0057b420 would then add to p) is not settled by the code read in the time box; drill R1 decides it.
5. **Shield: V read** for the normal case: a shield is never the item FUN_00535bc0 returns (it tests ItemTypes 45 `weap`), so with a weapon in use the shield is the "other hand" and its damage-related stats (base damage 21 / 22, its own ED) do not reach the player. Only Smite reads the shield (d2d `smite_lo`, combat.hpp:135; S sk-melee-trace.md). Barehanded with a shield is an edge case (I, below).
6. **No PR candidate built.** D1 and D2 are proven on the real build with this harness (before > 0); each fix needs more than one minimal commit fits in this card (D1: every `player_blow` caller and the physical resistance on 256ths; D2: monster undead flags and the weapon's Blunt type plumbed into `Target` / `Fighter`). Both go to a follow-up card with the harness ready.

## Q1 Which SKMELEE deviations are real on the build (V run)

Commands (the worker host; `<upstream>` = the scratch checkout, `<build>` its build directory):

```sh
cp tools/emu/sk_melee2_test.cpp <upstream>/tests/test_melee.cpp
cp tools/emu/sk_melee2.py <upstream>/tools/emu/
ninja -C <build> test_melee                       # tests/CMakeLists.txt globs tests/test_*.cpp
cd <upstream>/tools/emu
python sk_melee2.py gen 100000 7 > c7.tsv
<build>/tests/test_melee --melee c7.tsv > d7.tsv
python sk_melee2.py game c7.tsv > g7.tsv          # D2_MPQ_DIR etc. as for sk_melee.py; about 4 min
python sk_melee2.py cmp c7.tsv g7.tsv d7.tsv
```

Same with seed 1. md5 (first 8): c7 292231a5, g7 f17d3260, d7 ad60fc21; c1 3759d495, g1 f72896d5, d1 e18042e9.

Seeds: to-hit uses the case seed on both sides. For damage, d2d's `Rng` starts at the case seed and draws its to-hit roll first (`target.armor_class` 0: 95 %); game.exe's unit + 0x20 gets the state one LCG step later, so both damage rolls are the same draw (d2d's `Rng` is game.exe's LCG, rules.hpp:157, V read). A d2d miss (about 5 %) is excluded from the damage comparison.

| Part (tag) | game.exe code | cases seed 7 / seed 1 | differ seed 7 | differ seed 1 | verdict |
|---|---|---|---|---|---|
| to-hit (H) | FUN_0057d9b0 | 100,240 / 100,240 | 44 (43 with AR' <= 0, 1 with a negative %) | 36 (all AR' <= 0) | **D5 real** (same as the port: 43 + 1 / 36); 0 differ otherwise |
| damage, weapon (M) | FUN_0057b420 | 80,557 / 80,951 compared | whole points 0; 256ths 80,250; mean game - d2d +0.4980 | whole 0; 256ths 80,632; +0.4985 | **D1 real**; the whole-point range matches |
| damage, barehanded (M) | FUN_0057b420 | 14,388 / 14,142 compared | whole 12,325; mean +3.055 | whole 12,150; mean +3.004 | **D4 real** |
| undead grid (U) | FUN_0057b660 then FUN_0057b420 | 1,423 / 1,433 compared (1,500 each) | undead + blunt: whole 93 of 94 (stat 122 > 0), 47 of 47 (122 = 0), 49 of 97 (122 < 0); undead + other weapon with 122 > 0: 95 of 96; every non-undead and every undead + non-blunt with 122 <= 0 weapon cell: whole 0 | same pattern (95 / 95, 50 / 50, 49 / 96, 90 / 93; 0 elsewhere) | **D2 real** (the Blunt +50 and stat 122 vs undead); barehanded cells differ by D4 only |
| -90 floor (F) | FUN_0057b420 | 380 / 376 compared | whole 38, all with stat 25 under -90 and a skill ED over 0; 0 with stat 25 >= -90 | whole 40, same cells | **D6 real** (gear % floored before the skill ED on d2d); reach: no 1.14d gear gives under -90 (I) |
| off-weapon ED (O) | FUN_0057b420 with stats 17 / 18 = the off-weapon ED | 955 / 957 compared | off 0: whole 0 (weapon); off 15 / 40 / 100 / 300 with a weapon: 156 of 158, 156 of 156, 166 of 166, 151 of 151 (mean +13.1 to +195.7 points); barehanded cells add D4 | same pattern | **D8 real for the formula**; the game-side input (that off-weapon gear reaches the player's 17 / 18) is I (Q3) |

The game side is the oracle run of sk_melee.py (SKMELEE: 0 differences from the reading on these grids); the H and M parts consume the same random draws as sk_melee.py, so their stat values are identical case for case (the M seeds are shifted one step, see above).

## Q2 D7: does the weapon's own enhanced damage reach the player's stats 17 / 18 (V read in part; U)

- `FUN_00535d10(unit, flag)` (V read 0x535d10..0x535e1d): first both hand items (FUN_0063bef0 / FUN_0063bf90, the inventory's two hand slots) go through `FUN_00627910(unit, item, 0)`; then, for a unit passing FUN_006235a0, the item FUN_00535bc0 returns (the hand the current skill uses; a weapon only) goes through `FUN_00627910(unit, item, 1)` and the other hand's item through `(unit, item, 0)`; with no weapon in use, the FUN_0063bef0 slot item gets 1. `FUN_00535e20` does the same with the first slot active (V read 0x535e20..0x535e78).
- `FUN_00627910(unit, item, active)` (V read): an item in body slot 11 / 12 (weapon switch) goes to FUN_006269f0; an item already attached: active 1 on a flagged list calls FUN_00627860, active 0 on an unflagged list FUN_00627a40; otherwise it attaches the list (FUN_00626e10 with the flag).
- `FUN_00627860` clears 0x40000000 on the item's list and adds every stat of its vector (FUN_00624f40: +0x48 when extended, else +0x24) whose ItemStatCost row byte +4 has bit 0 (the mask at 0x6ce270; damagerelated, D) to the unit's list (FUN_00626920, + value); `FUN_00627a40` sets the flag and adds the negated values (V read 0x627860..0x62790b, 0x627a40..0x627ae5). `FUN_00624fe0` skips a flagged child's damage-related stats; `FUN_00626920` stops walking up the parent chain at a flagged list for those stats (V read 0x6269c8..0x6269d3).
- So the weapon in use contributes its damage-related stats to the player, and 17 / 18 are damage-related (D). What value the item's vector holds for 21 / 22 / 17 / 18 is the open part: FUN_00626920 routes a stat with an op (row bytes +0x51 / +0x52) through FUN_006266c0, which recomputes the stat and its op targets (FUN_00626200, then FUN_00625150 per target, V read 0x6266c0..0x6268b3) but whether FUN_00625150 writes the op-13 share into the item's 21 / 22 entries was not read. Three readings remain:
  - **A** (d2d, upstream combat.md S): ED is inside the weapon's 21 / 22 at the player, and the weapon's 17 / 18 do not reach the player's 17 / 18 (would need a mechanism not found yet).
  - **B**: ED inside 21 / 22 and also in the player's 17 / 18: counted twice.
  - **C**: the player's 21 / 22 are the raw base + flats and the ED reaches FUN_0057b420 only as stats 17 / 18, joining p additively with the Strength bonus.
  The community damage formula (weapon ED multiplies the base, then Strength and off-weapon ED multiply the result) matches A only (I). **U**; drill R1 reads the four stats directly.

## Q3 The shield, and off-weapon gear (V read; I where marked)

- A shield never passes FUN_00535bc0 (it returns an item only when FUN_00629bb0(item, 45) says weapon; V read 0x535cb8..0x535cd8), so with a weapon in use the shield is the other hand: FUN_00627a40 takes its damage-related stats (base damage, its own ED, its +min / +max) out of the player (V read). The plain Attack therefore never sees a shield's damage; d2d agrees (it reads the shield only for Smite and block).
- Barehanded with a shield in the FUN_0063bef0 slot: that slot item gets active 1, so the shield's damage stats would reach the player (I: which hand FUN_0063bef0 returns and whether a player can carry only a shield there were not traced; no bot build does this).
- Off-weapon gear (armor, helm, belt, boots, gloves, jewelry, charms) is never one of the two hand items, so FUN_00535d10 never flags it (I: the other FUN_00627910 callers, 0x45d47b, 0x4852a1, 0x4e9091 and the rest, were not all read); its damage-related stats, 17 / 18 among them, then reach the player and FUN_0057b420 adds them to p. This is the game-side input of D8 (I, strong: FUN_0057b420 reads the player's 17 / 18, V read in SKMELEE).

## Deviations d2d vs game.exe (real build)

| # | Rule in game.exe | Rule in d2d | Tag | Real build | Differential |
|---|---|---|---|---|---|
| D1 | physical kept in 256ths (FUN_0057b420 -> record +8 -> FUN_0057dbf0) | `phys = max(damage >> 8, 1)`, `Blow::frac` only from missiles (combat.hpp:507) | V run (both) | **real**: 80,250 / 80,632 weapon cases differ in 256ths, 0 in whole points; +0.498 points a landed blow | yes: `sk_melee2.py` M part vs `test_melee --melee` (this card) |
| D2 | vs undead: +50 to the ED with a Blunt weapon plus stat 122, when the sum is over 0 (FUN_0057b660) | none (no undead flag in `Target`, no Blunt in `Fighter`) | V run (both); Blunt by Equiv D, the walk I | **real**: undead + Blunt 93 / 94, 47 / 47 whole points differ (seed 7); stat 122 > 0 with any weapon 95 / 96 | yes: U part |
| D3 | AR terms 115 / 116 / 123 / 124 / 179 and stat 120 (FUN_0057d8a0, FUN_0057b7d0) | none (no stat ids) | V read | not run (FUN_0057d8a0 hooked to 0 here) | buildable: run FUN_0057d8a0 natively with the boss tests hooked; d2d side has no input, so the before count is the share of cases with those stats |
| D4 | barehanded: p += strength; min / max = stat 21 or 1, stat 22 or 2 | 1 + flats, 2 + flats, no Strength term | V run (both) | **real**: 12,325 / 12,150 barehanded cases differ in whole points | yes: M part |
| D5 | AR' = AR + MulDiv(AR, %, 100), no floor | `max(AR x (100 + %) / 100, 1)` | V run (both) | **real**: 44 / 36 of 100,240 (AR' <= 0 against defense 0 or 1, one negative %) | yes: H part |
| D6 | p floored at -90 once | gear % floored, then again with the skill ED | V run (both) | **real**: 38 / 40 of 380 / 376 F cases, all with stat 25 under -90 | yes: F part; no practical reach (I) |
| D7 | the weapon's 17 / 18 at the player: A, B or C (Q2) | A | U | not decidable offline yet | drill R1, then an oracle mode per reading |
| D8 (new) | FUN_0057b420 adds the player's stats 17 / 18 (off-weapon ED, Q3) to p | `make_fighter` uses only `weapon_sum` 17 / 18 (combat.hpp:111-112); tests/test_combat.cpp:195 asserts no effect | V run (formula, both sides); the game-side input I | **real for the formula**: off 15 / 40 / 100 / 300 with a weapon, 156 of 158, 156 of 156, 166 of 166, 151 of 151 differ (seed 7; seed 1: 157 of 161, 159 of 163, 164 of 164, 155 of 155); +13 to +197 points a blow | yes: O part (the oracle's model A' puts only off-weapon ED in 17 / 18) |

Not covered: swing speed, the defender's block, leech, crushing blow, open wounds, elemental gear, FUN_005d3ac0's state events, Smite / Zeal / Sacrifice dispatch (as SKMELEE).

## Bot findings (described, not changed; real-build facts added)

- B1 (unchanged; real-build note) The resource simulator's melee is a flat 3 damage per 1.0 s swing that always lands (`internal/resource/sim.go:94`, applied at `internal/resource/sim.go:320`). Neither game.exe nor d2d works that way: both roll the same to-hit (V run, real build: equal for every AR' over 0) and the same whole-point range for a weapon; game.exe adds about half a point a landed blow on top.
- B2 (unchanged) `WeakPack` (`internal/resource/types.go:64`) has no AR or damage term; the real build confirms the level ratio enters only through 2 alvl / (alvl + dlvl).
- B3 (now proven on the build) The melee rung (`internal/resource/attack.go:74`) ignores the undead bonus. game.exe gives a staff, mace or scepter +50 points of damage % on undead; the REAL d2d build does not (D2, U part), so a d2d soak under-rates exactly the staff-vs-undead case the rung is used for in Act 1.
- B4 (now proven on the build) d2d soak numbers for early melee carry D1 (-0.498 points a landed blow, real build) and D2; and, for a later build with off-weapon ED gear, D8 (all of that ED is lost on d2d, if Q3's reading holds).
- B5 (new, I) Any bot-side damage estimate that follows d2d's model (weapon ED only) will under-rate off-weapon ED gear on game.exe; no such estimate exists in `internal/resource` today (B1 is a flat 3), so nothing is wrong yet.

## PR candidates

**None built.** Proven on the real build with this harness: D1, D2, D4, D5, D6, D8 (formula). Decision per deviation:

- D1: needs more work. A faithful fix keeps the physical in 256ths through the physical resistance (`resisted_fine`, the missile machinery already on main: `Blow::frac`, strike.cpp:464 passes it to `hurt`) and drops the one-point floor for sub-point blows. `player_blow` has ten call sites (strike.cpp:564, 750; missiles.cpp:652; pets.cpp:192, 801, 883, 900, 953; deaths.cpp:283; tick.cpp:411), and the pet, hireling and monster-missile paths ignore `frac`, so the minimal claim needs a caller audit, det hashes and an adversarial review (docs/process/upstream-pr-standard.md sections 4-6). The harness is ready: before 80,250 / 80,632 (256ths, weapon cases), after expected 0 with the test printing `damage x 256 + frac`.
- D2: needs more work. d2d has no undead flag (MonStats lUndead / hUndead not loaded: no `undead` in monsters.hpp / rules.hpp, V read) and `Fighter` has no Blunt bit (`type_is` exists, rules.hpp:671). One claim, three files plus the test; harness ready (U part: before 93 / 94 and 47 / 47 whole points in the undead + Blunt cells, after expected 0 there).
- D4, D5, D6: real but low reach (barehanded only; AR' <= 0 only; gear % under -90 only). Not worth an upstream claim on their own.
- D8: needs D7 first: a fix built on reading A' would be wrong if reading B or C holds.

## Drill recipe (game.exe on one instance; not run)

- R1 (D7, Q2): a throwaway character with a magic weapon with enhanced damage only (no +min / +max) and nothing else damage-related worn: read the player's stats 17, 18, 21, 22 and the weapon's 17, 18, 21, 22 with the bot's stat reader (`ReadStatList`, docs/d2-1.14d-memory.md). Expected: A: player 17 / 18 = 0, player 21 = base x (100 + ED) / 100; B: player 17 / 18 = ED and 21 with ED; C: player 17 / 18 = ED and 21 = the raw base. Then swap a helm with ED (or a jeweled armor) in and re-read 17 / 18 for Q3.
- R2 (shield, Q3): a Paladin with a spiked shield (Armor.txt damage over 0) and a weapon: player stats 21 / 22 with and without the shield: expect no change (V read). Then without the weapon: does 21 / 22 pick up the shield (the I edge case)?
- R3 / R4 (D2 / D1, lab binary): as in sk-melee-trace.md; the real-build numbers above are the d2d side's expectation.

## Matrix cell updates (docs/research/sorc-skill-matrix.md section 3.12 (b); proposed text)

- 3.12 (b) d2d status: "same to-hit and whole-point weapon range as game.exe on the REAL build (100,000 cases x 2 seeds, sk-melee2-trace.md); floors the physical to whole points (D1, -0.50 a hit), no Blunt / stat 122 vs undead (D2), barehanded rule (D4), off-weapon ED ignored (D8); all V run on the build; weapon ED at the player U (D7, drill R1)."

## Shared-file lines for the merger

- docs/decomp-queue.md, row 20, status column, append: `; real-build differential (SKMELEE2 2026-11-01): docs/research/sk-melee2-trace.md: D1, D2, D4, D5, D6 real on the build, new D8 (off-weapon ED ignored by d2d), D7 narrowed (U, drill R1); no PR candidate (D1 / D2 need a follow-up card)`
- docs/BACKLOG.md (decomp section): `- skmelee2 (card [skmelee2](../archive/openspec-index.md#skmelee2), offline): the plain Attack on the REAL d2d build (docs/research/sk-melee2-trace.md, tools/emu/sk_melee2.py + sk_melee2_test.cpp): D1 (256ths), D2 (Blunt / stat 122 vs undead), D4, D5, D6 real; new D8: d2d ignores off-weapon enhanced damage; D7 U (drill R1); shield damage never reaches the plain Attack (V read). Next: SKMELEEFIX card for D2 then D1 (harness ready).`
- docs/INDEX.md, decomp lane line, after the sk-melee entry: `, [research/sk-melee2-trace.md](sk-melee2-trace.md) (the plain Attack on the real d2d build)`
- CHANGELOG.md (Unreleased, docs): `- Decomp row 20 follow-up (SKMELEE2): the player's plain Attack differentialled against the real d2d build (docs/research/sk-melee2-trace.md, tools/emu/sk_melee2.py, tools/emu/sk_melee2_test.cpp).`
- docs/research/README.md: `- sk-melee2-trace.md: the plain Attack on the real d2d build (per-part differences, D7 and the shield, off-weapon ED); oracle tools/emu/sk_melee2.py, d2d test mode tools/emu/sk_melee2_test.cpp.`
- docs/research/sk-melee-trace.md, a banner under the status line: `> status: deviations re-proven on the real d2d build, D7 narrowed, D8 added: docs/research/sk-melee2-trace.md (SKMELEE2).`
- docs/research/sorc-skill-matrix.md: the cell in the section above.
- Proposed glossary term (not written): **off-weapon ED** = enhanced damage (stats 17 / 18) on gear other than the weapon in use; game.exe adds it to the damage percentage (FUN_0057b420), d2d ignores it.
