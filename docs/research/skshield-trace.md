# Does a carried shield's damage reach the player's stats? game.exe 1.14d vs d2d vs the bot (SKSHIELD)

> status: current (2026-11-02, agent SKSHIELD, first pass, Sonnet). Offline: no game, no d2d slot, nothing pushed or posted. Tags: **V read** = read in game.exe's disassembly (`disasm.py`, addresses given) or in d2d's code at file:line; **V run** = game.exe's own functions run under unicorn on the worker host (`tools/emu/skshield_*.py`), or d2d's real build run (the earlier SKMELEE2 files); **D** = 1.14d tables through the emulator's MPQ stack (`tools/emu/mpq-table.sh`); **S** = an upstream doc or an earlier trace; **I** = inference; **U** = unverified. d2d = upstream main 72132b99 (the read checkout and the build from SKMELEE2). No game code is pasted: function tags and short formulas only. Not an independent verification: a Sonnet decomposition, nothing here is a PR claim.

## Summary

1. **A shield is never one of the two "hand items" the melee hit toggles.** FUN_0063bef0 and FUN_0063bf90 (the two hand items) return a hand item only when FUN_00629bb0(item, 45) says Weapon, and FUN_00535d10 (the toggle the hit code runs round the damage build, FUN_0057b7d0) touches only those two and the weapon the skill uses (FUN_00535bc0, also weapon-only). ItemTypes: Shield `shie` -> `shld` -> `armo`, `seco`: none of them is `weap` (D). V read + V run (382 scenarios, 0 differ from the reading). **The earlier reading "FUN_00627a40 takes the shield's damage stats out of the player" (sk-melee2-trace.md Q3) is wrong**: that function never sees a shield.
2. **A worn shield is attached active: nothing flagged, nothing withheld.** FUN_004c0d20 (the equip attach; V read 0x4c0e00..0x4c0e82) hands every non-weapon item to FUN_00627910 with active 1; only a weapon that is not the primary one of a dual-wield class gets active 0 (the carrier flag 0x40000000, which withholds damage-related stats). A broken (item flag 0x100) or unusable (0x4000) shield is not attached at all: all of its stats are off (V read, V run). The shield's **flat damage (21 / 22), damage % (25), IAS (93) and every other stat reach the player's list regardless of weapon, hand state or toggle** (V run, 382 scenarios).
3. **A shield has no base damage in its stat list, so no base damage reaches 21 / 22.** The base-damage writer (FUN_005563d0, from the item maker FUN_00557ab0 at 0x557d46) runs only after FUN_00629bb0(item, 45) at 0x557c7d; the item loader (FUN_0062cbe0, 0x62d450) has the same gate. V run of the real item maker: 306 of 306 Weapons.txt rows get a base-damage stat (21 / 22, 23 / 24 or 159 / 160), 0 of 69 armor rows (54 shield rows, of which 39 have a txt min / max, plus 15 boots rows with a txt kick damage). The Armor.txt min / max of a shield (spiked shield 5-9 in D) is therefore used only by code that reads the table (Smite; U, see Q7), not through the player's stats.
4. **A shield's enhanced damage (17 / 18) reaches the player's 17 / 18** and FUN_0057b420 adds the player's 17 / 18 to the damage percentage (S sk-melee-trace.md Q2, V read; formula V run in sk-melee2-trace.md). The weapon's own ED does not: op 13 folds it into the weapon's 21 / 22 and the player's 17 / 18 stay 0 (V run, item built as the property code builds it). **This settles SKMELEE2's D7 as reading A: d2d is right about the weapon, wrong about everything else that carries ED (D1 below = SKMELEE2's D8, now with its game-side input run).** Limit found: an item that already carries a flat 21 / 22 (a +max damage shield with ED) has its ED absorbed (player 17 / 18 stay 0).
5. **Barehanded with a shield:** no weapon in use, nothing toggled; the shield's flats and ED are in the player's list (V run, "hand S" scenarios), the barehanded rule of FUN_0057b420 then reads stat 21 as the minimum when over 0 (S, D4 of SKMELEE).
6. **d2d**: ignores the shield's ED (D1), keeps a broken shield's props in the fighter's sum (D2, V read d2d, V run game). It already reads the shield's flats and damage % (they go in `sum`). **Bot**: scores a shield by defence only and gives no value to damage stats on any item (B1, B2): consistent with the game for base damage, blind for flats and ED.
7. **No PR candidate built** (D1 is proven by the existing harness, but no branch can be made from this box; see PR candidates).

## Q1 Which items does the hit-time hand toggle touch? (V read, V run)

- `FUN_0063bef0(inventory)` (0x63bef0): body-slot array of grid 0 (index 5, then index 4): returns the first item for which FUN_00629bb0(item, 0x2d) is true and whose owner id (item +0xc) equals the inventory's (+0x1c). `FUN_0063bf90` returns the other of the two slots' weapons (0x63bf90..0x63c040). Both need a magic word 0x1020304 in the inventory. So with a weapon in slot 4 and a shield in slot 5, the "primary" is the slot 4 weapon; with a weapon in both, slot 5 is read first (V run: the scenario model that says so agrees in all "W+W2" rows). V read 0x63bf0a..0x63bf75.
- `FUN_00535d10(unit, flag)` (0x535d10..0x535e1d): with flag != 0, both hand items go `FUN_00627910(unit, item, 0)` (deactivate). Then, only for a unit passing `FUN_006235a0`, the skill's weapon (FUN_00535bc0) is activated and the other hand's item deactivated; with no skill weapon the primary one is activated. `FUN_006235a0` (0x6235a0..0x6235f8, V read): for a player the class (unit +4) is 4 (Barbarian) or 6 (Assassin); for a monster a class in 0x1a1..0x1a2 below the table count. So **only Barbarian and Assassin ever have a second weapon whose damage stats are carried off; for every other class both hand weapons stay attached active and the toggle only switches both off when the flag is set** (V run: "cls 3" rows, flag 1: the weapon's stats drop, the shield's stay).
- Callers: FUN_0057b7d0 (the melee hit's damage build, V read 0x57b800 before the build, 0x57bd84 `FUN_00535e20` after it, the flag is its third stack argument: 0 for a normal blow, 1 at 0x5d5854, in the double attack function; S sk-melee-trace.md Q2 names FUN_0057b7d0) and the setup function at 0x59f900 (flag 0).
- Other FUN_00627910 callers I read (0x4e92xx, 0x4ebcxx, 0x5d57xx, 0x6462xx, 0x55c8xx, 0x623cxx): every one is guarded by FUN_00629bb0(item, 0x2d) (weapon) or by the results of FUN_0063bef0 / FUN_00535bc0; none passes a shield. The remaining callers (0x45d47b, 0x485xxx, 0x55f56b, 0x5774d2) re-attach with the state read back by FUN_00625820 (preserve). V read, not exhaustive (D-limit 1).

## Q2 How is a worn shield attached? (V read, V run)

- `FUN_004c0d20` (0x4c0d20; unit in EDI, item in ESI, one stack argument): a gem (type 0x14) and a cloak (0x4a) have their own branches. Everything else (0x4c0e00): `FUN_00663cc0` (inventory bookkeeping, hooked in the run), then FUN_006280a0(item, 0x4000) non-zero returns (unusable: no attach); for a player (unit type 0, 0x4c0e29): EBX = FUN_0063bef0; `FUN_006235a0` (dual wield class); active = 1 (0x4c0e40); if the item is a weapon (45) other than the primary, active = 0 (0x4c0e4d..0x4c0e59). A broken item (flag 0x100) is skipped unless the argument is non-zero (0x4c0e60..0x4c0e7a); then `FUN_00627910(unit, item, active)` (0x4c0e82). A non-player unit: active 1 (0x4c0e8e..0x4c0eae). For non-dual-wield classes the primary-weapon test is skipped (0x4c0e3c: the branch to 0x4c0e8e) so a Paladin's weapon is active too.
- `FUN_00627910` -> `FUN_00626e10(unit, list, active)`: active 1 clears flag 0x40000000 and adds every stat of the item's vector to the unit's list with FUN_00626920 (0x626f88..0x626fb7); active 0 sets the flag and adds only the stats whose ItemStatCost damage-related bit is clear (0x626fc2..0x627020; the mask byte at 0x6ce270 is 4, read in the run, rows have byte +4 of 6 or 7). `FUN_00626920` stops its walk up the parent chain at a flagged list for damage-related stats (0x6269c8..0x6269d3).
- Run: **382 scenarios** (4 classes x hands x shield stat sets x item state x toggle flag x skill hand x weapon ED; enumerated, no seed) through the real FUN_004c0d20 and FUN_00535d10 on a unit and an inventory built by `skshield_run.py`: **0 differ** from the reading coded in `model()`. Selected rows (Paladin, flag 0; stats 21 / 22 / 17 / 18 / 25 / 93 / 31 / 80 as listed):

| hand | shield list | state | player's list |
|---|---|---|---|
| shield only | none | ok | empty |
| shield only | flat +2 / +5 | ok | 21 = 2, 22 = 5 |
| shield only | flat +2 / +5 | broken or unusable | empty |
| shield only | ED 40 | ok | 17 = 18 = 40 |
| shield only | damage % 30 | ok | 25 = 30 |
| shield only | IAS 20 | ok | 93 = 20 |
| shield only | defence 10, MF 15 | ok / broken | 31 = 10, 80 = 15 / empty |
| weapon 3-7 + shield | flat +2 / +5 | ok | 21 = 5, 22 = 12 |
| weapon 3-7 + shield | ED 40 | ok | 17 = 18 = 40, 21 = 3, 22 = 7 |
| weapon 3-7 with ED 50 + shield | ED 40 | ok | 17 = 18 = 40, 21 = 4, 22 = 10 |
| weapon 3-7 + shield | flat +2 / +5, ED 40 | ok | 21 = 5, 22 = 12 (ED absorbed) |
| weapon alone, flag 1 | | ok | empty (Paladin) / 3-7 kept (Barbarian, the skill weapon) |

Barbarian and Assassin with a second weapon in slot 5 (skill weapon = the right hand): the right weapon's 3-7 only, the other off (rows "W+W2"). The same 382 give 0 differences with the unit list type 0 or 1; **16 differ with unit list type 4** (an item-type list as the player's, which FUN_00626200's op 13 then treats as an item's: the ED vanishes) and 48 with the item lists type 0 (op 13 not applied): both variants are wrong inputs, see Limits.

## Q3 Does a shield carry base damage? (V read, V run, D)

- `FUN_00557ab0` (the item maker's base rolls, drops.py's FUN_00557ab0): at 0x557c7d `FUN_00629bb0(item, 0x2d)`; for a weapon it rolls durability (stats 0x48 / 0x49), calls `FUN_005563d0` at 0x557d46 (0x5563d0..0x55649b: stats 0x15 / 0x16 from the Items row bytes +0xfe / +0xff, 0x17 / 0x18 from +0x102 / +0x103, 0x9f / 0xa0 from +0x100 / +0x101 when non-zero) and sets stat 0x44 from the row; for a non-weapon it takes the armor branch (defence, durability). The item loader `FUN_0062cbe0` has the same weapon-only branch (0x62d450..0x62d6f4, gate at 0x62d450).
- Run (`skshield_base.py`): the real FUN_00557ab0 on a fake item per row (seeds and version as drops.py; the stat functions NOT hooked; ilvl 30, quality 2): **weapons made 306, with a base damage stat 306; armor rows with a txt damage or a shield type 69, with a base damage stat 0.** (Shield rows 54: spiked shield 5-9, barbed shield 18-35, targe 2-6 in D; Voodoo Heads rows have 0-0; 15 boots rows with kick damage 3-8 ... 6-12.)
- So the "Spiked Shield 5-9" in the table never shows up as stat 21 / 22 of a shield. d2d agrees by construction: its shield goes through `smite_lo` / `smite_hi` only (combat.hpp:129-134).

## Q4 What do the player's 21 / 22 / 17 / 18 hold? (V run on the merge; V read for FUN_0057b420; I for real-item structure)

- Flat 21 / 22 from a shield add to the weapon's range (21 = 5, 22 = 12 in the table). d2d does the same: `sum` is built from every worn item's props (fight.cpp:505-511) and `make_fighter` adds `stat(kMinDamage)` / `stat(kMaxDamage)` from it (combat.hpp:111-112): **no deviation for flats** (V read d2d, V run game).
- ED (17 / 18, op 13 on 22 / 24 / 160 and 21 / 23 / 159, D): FUN_00626920 sends a stat whose row has the op byte (+0x51) to FUN_006266c0, which recomputes the op targets with FUN_00626200 (op 13: the item's base stat times ED / 100, MulDiv, only for a list owned by an item, SKMELEE) and FUN_00625150. Run with the item as the property code builds it (a main list for base stats, set with FUN_00627260; a child list for modifiers, added with FUN_00627030): a weapon 3-7 with ED 50 reads 21 = 4, 22 = 10 and 17 = 18 = 0 at the weapon and at the player (**reading A: the weapon's ED is in 21 / 22, not in the player's 17 / 18**); a shield with ED 40 and no flat damage keeps 17 = 18 = 40 and passes them to the player; the same shield with a flat +2 / +5 loses the ED (the op consumes it, its share of a zero base is 0). The first runs with an item as one flat list showed raw sums (a "model C") and a differential built on that was wrong: see Dead ends.
- FUN_0057b420 then uses the player's 17 / 18 (S sk-melee-trace.md Q2: `min' = min + MulDiv(min, stat 18 + p, 100)`): a shield's ED is an addition to the percentage, a weapon's is already inside the range. d2d: `weapon_stat(17 / 18)` only (combat.hpp:111-112, :105), the shield's ED is dropped (D1).

## Q5 Barehanded with a shield (V run, S)

The shield is in slot 5, FUN_0063bef0 returns 0 (no weapon): FUN_00535d10 with the flag 0 does nothing to a shield; with the flag 1 nothing to toggle. The shield's flats / ED / damage % are in the player's list either way (rows "hand S"). FUN_0057b420 without a weapon: minimum = stat 21 if over 0 else 1, maximum = stat 22 if over 1 else 2, and p += strength (S SKMELEE Q2). So a barehanded Paladin with a +2 / +5 shield hits 2-5 in game.exe; d2d computes `1 + 2` / `2 + 5` (combat.hpp:111-112): part of SKMELEE's D4, shield flats only make it visible.

## Q6 The carrier flag, in short (V read, V run)

0x40000000 on an item's list means "inactive: damage-related stats withheld", and it is only ever set on **weapons** (FUN_004c0d20 for a non-primary weapon of a dual-wield class, FUN_00535d10 / the weapon switch code for the toggled hand). Barbarian (class 4) and Assassin (class 6) are the dual-wield classes (FUN_006235a0). Shields, armor, jewelry and charms are attached active and stay so. Damage-related is a table bit (ItemStatCost, D: stats 17-19, 21-25, 48-66, 68, 70, 81, 93, 108, 111-113, 115-117, 119-124, 134-136, 140, 141, 150, 154-156, 159, 160, 179, 180, 195, 196, 198, 218, 219, 224-229, ... 326): so a toggled-off weapon also loses its IAS (93), crushing blow (136), open wounds (135) and to-hit (119) with its damage.

## Q7 Smite (U)

FUN_00535bc0 returns a hand item only when it passes FUN_00629bb0(item, 0x2d), so a shield is never the item the skill uses on this route; how the Smite damage reads the shield's Armor.txt min / max was not traced (candidate: the skill damage function FUN_00647bc0 and its SrcDam share, S sk-melee-trace.md Q1). d2d adds the shield's table min / max and the Str bonus for Smite (`smite_lo`, `smite_hi`, `smite_pct`, combat.hpp:129-134), with `stat(kMaxDamagePercent) - weapon_stat(...)`, i.e. the off-weapon ED. U: whether game.exe's Smite includes the player's 17 / 18 the same way.

## Deviations d2d vs game.exe

| # | Rule in game.exe | Rule in d2d | Tag | Differential |
|---|---|---|---|---|
| D1 | the player's 17 / 18 hold the ED of every worn item that has no flat 21 / 22 of its own (shield, armor, helm, jewelry, charms, socketed jewels); FUN_0057b420 adds them to the percentage | `weapon_sum` 17 / 18 only (combat.hpp:105, :111-112; fight.cpp:545-548); the ED of anything else is dropped (tests/test_combat.cpp:195 asserts "no effect") | input V run (382 scenarios, shield; other gear V read: the same non-weapon branch of FUN_004c0d20), formula V run (SKMELEE2 O part, re-compared here), real-item structure I | **yes, exists**: SKMELEE2's O part. Re-run of `sk_melee2.py cmp` on its kept files (cases 1,000, seeds 7 / 1): weapon cases with off-weapon ED 15 / 40 / 100 / 300: 156 of 158, 156 of 156, 166 of 166, 151 of 151 differ in whole points (seed 7; seed 1: 157 / 161, 159 / 163, 164 / 164, 155 / 155; mean +13 to +196 points a blow); with 0 off-weapon ED: 0 of 161 / 168 differ in whole points |
| D2 | a broken worn item (flag 0x100) is not attached: none of its stats count (FUN_004c0d20 0x4c0e60..0x4c0e7a, FUN_006280a0) | the fighter's `sum` adds every worn item's props with no broken test (fight.cpp:505-511); only the weapon's base range is dropped when broken (combat.hpp:108); the char panel does exclude broken (character.hpp:110, :219) | game V read + V run (rows "broken": player's list empty), d2d V read | **buildable**: a `tests/` case that builds a character with a worn shield carrying props (a +% damage and a flat), sets item flag 0x100, calls the Fight builder at fight.cpp:500 and prints `sum`; before: the props are in; oracle = this card's `skshield_run.py` broken rows (all stats 0). Not built here |
| D3 | an item with a flat 21 / 22 and ED: the ED is absorbed | the ED is dropped anyway (D1) | V run on the synthetic structure, I for real items | none needed: no difference unless D1 is fixed naively; a fix must sum the ED per item, not use `sum` 17 / 18 |
| — | D7 (SKMELEE2): the weapon's own ED in 21 / 22, not at the player's 17 / 18 | the same (reading A) | V run | **not a deviation** (settled) |

## Bot findings (described, not changed)

- B1 `internal/leveler/score.go:207-214` (`scoreOf`, armor branch): a shield scores `(MinAC + MaxAC) / 2` (superior, ethereal factors) plus the stat value; its Armor.txt damage is not used. Correct for the plain Attack (the game gives a shield no base damage, Q3); Smite would use it (U, Q7), so a Paladin's shield pick ignores a Smite build.
- B2 `internal/leveler/score.go:288-324` (`statValueCtx`): no case for stats 17, 18, 21, 22, 25 or 93, so a shield, helm or glove with ED or +min / +max damage is worth nothing in the equip choice, while the game gives them to the player (Q2, Q4). Harmless for the Sorceress (nothing in the bot's score reads damage); a melee class would under-value such gear; the weapon branch uses only the table average (`scoreOf`, lines 215-228).
- B3 `internal/leveler/prefs.go:34` (`paladin`, `WantsShield`) and `score.go:438-440`: a two-hander is kept off while a shield is worn: consistent with a shield being active in all states (Q2); nothing there assumes shield damage.
- B4 `docs/research/paladin-act1-recon.md` section 2 row "Hit recovery and block" says "Smite is a shield attack: damage from the shield" (I, there): still unproven here (Q7); do not let a bot damage estimate add the shield's table damage to the plain Attack.

## PR candidates

**None built.** D1 is proven on the real d2d build with the existing differential (O part: before 156 of 158 ... 151 of 151 differ), but (a) this box cannot create a worktree or touch the fork checkout, so no local branch with a "before > 0, after 0" pair exists, and (b) the fix is not one line: `make_fighter` has one `phys_pct` for min and max and takes `sum` of the whole character; the faithful rule needs the ED of each non-weapon item that carries no flat 21 / 22 (D3), so the Fight builder (fight.cpp:505-519) must keep a separate off-weapon ED sum. Plan for a follow-up card (SKSHIELDFIX): build the sum per item in fight.cpp, add `off_ed` to `phys_pct` in `make_fighter`, run `sk_melee2.py gen/game/cmp` with the existing `sk_melee2_test.cpp` (expected after: off 15 / 40 / 100 / 300 weapon cases 0 differ in whole points; bare cases keep D4); needs the D2d review standard (docs/process/upstream-pr-standard.md) and an independent verifier. D2 is a candidate once its test exists (see Deviations).

## Dead ends

- **Model C (retracted).** A first equip run built each item as one flat list (stats set with FUN_00627260 on it): the player's list then held raw sums, 17 / 18 of the weapon's ED included and 21 / 22 without ED. I ran SKMELEE2's M and O parts with that input against the real d2d build's kept output (`skshield_melee.py`): O part 872 of 955 compared differ in whole points (mean -2.2 points), M part more weapon-with-ED cases. That would have been a large d2d overstatement for every weapon with ED and a Strength bonus; it was an artefact of the item structure. With a main list plus a modifier child list (as upstream's drops.py documents the property code), op 13 folds the weapon's ED into 21 / 22 and nothing reaches 17 / 18 (reading A). The earlier S claim in upstream's combat.md stands. Run counts of the flat structure are not evidence.
- The unit list type 4 and the item list type 0 variants (Q2) are wrong inputs: FUN_00626200's op 13 needs an item-type owner list. They are in the run only as sensitivity (`skshield_run.py unit=80000000:4`, `item=0:0`).
- Reading FUN_00627a40 as "takes the shield's stats out" (SKMELEE2 Q3): wrong, see Summary 1.
- The path through FUN_005563d0's callers other than FUN_00557ab0 (none: one xref) and the other 6251f0 creators of a unit's list: the real type and flags of a player's main list were not found (V read attempt over all callers, too many); the run is shown insensitive to unit type 0 or 1.

## Limits

1. The structure is minimal: a unit with an inventory and a list, items with a main and a child list; the list flags and types (0x80000000 type 0 or 1 for the unit; 0x80000000 / 0x40 type 4 for the item) are inferred (I). The ED conclusion (weapon folded into 21 / 22, shield's kept) depends on op 13 and on the item list type 4; with other values the run goes the other way (Q2). A live read (drill S1) settles it.
2. Not run: gems and jewels socketed in the shield (their stats join through FUN_0062xxxx child lists, not traced); sets and runewords; the char panel's own stats; charms; the weapon-switch slots 11 / 12 (FUN_006269f0); throwing weapons; FUN_0057b420 itself (earlier traces); Smite.
3. FUN_00627910 callers not all read (Q1); FUN_004c0d20's gem and cloak branches not read.
4. d2d side: V read of fight.cpp / combat.hpp plus the SKMELEE2 build's kept outputs; I did not rebuild or re-run `test_melee`. The case files and outputs are the ones SKMELEE2 kept; their md5 are in sk-melee2-trace.md Q1.
5. Class numbering (4 Barbarian, 6 Assassin) is read from FUN_006235a0 and the standard class order; no table row checked.

## Drill recipe (fork verbs; not run)

Method: `docs/forkdbg.md` (one hidden slot, a throwaway Paladin, `d2drill -probe '<verbs>'` ledgered), lab binary as in docs/d2d-fork.md and docs/forkcheats.md.

- **S1 (live, game.exe; Q2, Q4, U1):** a character with a magic shield carrying "+x% enhanced damage" (no +min / +max) and a weapon: read the player's stats 17, 18, 21, 22 with the bot's stat reader (`ReadStatList`, docs/d2-1.14d-memory.md): expect 17 = 18 = x, 21 / 22 = the weapon's range (with its own ED folded in); unequip the shield, re-read: 17 = 18 = 0. With the shield's durability at 0: 17 / 18 and the shield's defence gone.
- **S2 (live, Q3):** read the shield item's own stats 21 / 22: expect none (0 / 0), against a weapon's.
- **S3 (live, Q5):** the same shield with the weapon off: 21 / 22 = the shield's flats only (no 1-2 added), a punch on a dummy shows it.
- **S4 (lab d2d, D1):** `debug equip <shield> 5` with a shield that has ED (`debug give <code> q4` until one rolls ED), `debug equip <weapon> 4`, `dump stats` ("gear" sums include the shield's 17 / 18); fight a fixed-defence monster and compare mean damage with and without the shield; expectation on the lab build: no change from the shield's ED (D1), game.exe: + the ED on the percentage.
- **S5 (lab d2d, D2):** `debug durability 5 0` on the shield; `dump stats` gear sums exclude it (character.hpp), the fight does not (fight.cpp:509): the difference shows as the shield's flat damage staying in hits.

## Matrix cell updates (proposed text)

- `docs/research/sorc-skill-matrix.md` 3.12 (b) d2d status, append: "a shield's base damage never reaches the player (no stat 21 / 22 on a shield, 306 / 306 weapon rows made with one, 0 / 69 armor rows: skshield-trace.md Q3); its flats and damage % do (d2d too), its ED does not on d2d (D1, V run); D7 settled as reading A."
- `docs/research/paladin-act1-recon.md` section 6, row 4 (player's blow) Cheapest check, append: "shield question answered offline (docs/research/skshield-trace.md): no base damage, flats and ED yes; Smite's own damage read U".
- `docs/research/sk-melee2-trace.md` Q3 (correction, not edited here): replace "FUN_00627a40 takes its damage-related stats out of the player" by "the shield is never a hand item; it is attached active (skshield-trace.md Q1 / Q2)"; D7: "A (V run, skshield-trace.md Q4)"; D8: add "game-side input: V run for a shield".
- `docs/research/sk-melee-trace.md` summary 5 (open U): "answered: skshield-trace.md".

## Reproduce

On the worker host, from the worktree:

```sh
tools/emu/run-on-worker.sh skshield_run.py > /tmp/d2agent-skshield/run.txt 2>&1          # Q1, Q2, Q4: 382 scenarios, "differ from the model 0"
tools/emu/run-on-worker.sh skshield_run.py unit=80000000:4 > run4.txt                        # sensitivity: 16 differ (wrong input)
tools/emu/run-on-worker.sh skshield_run.py item=0:0 > run0.txt                               # sensitivity: 48 differ (wrong input)
tools/emu/run-on-worker.sh skshield_base.py > base.txt 2>&1                                  # Q3: weapons 306 / 306, armor rows 0 / 69
tools/emu/run-on-worker.sh skshield_cmp.py <skmelee2 run>/c7.tsv <skmelee2 run>/g7.tsv <skmelee2 run>/d7.tsv   # D1's formula: O part, seed 7 (c1 g1 d1: seed 1)
```

Reading aids (output to the scratch dir): `skshield_dis.py ADDR[:N] ...` (disassemble), `skshield_xref.py ADDR ...` (call sites with the pushes), `skshield_scan.py ID ...` (immediate passed to a call), `skshield_funcs.py LO HI` (function starts), `skshield_all.py FILE` (the whole code listing, one line an instruction, for grep).

## Proposed glossary terms (not written)

- **carrier flag**: list flag 0x40000000 on an item's stat list: damage-related stats of that item are withheld from the unit (an inactive weapon). Never set on armor or shields.
- **off-weapon ED**: as in sk-melee2-trace.md (enhanced damage on gear other than the weapon in use).

## Shared-file lines for the merger

See `openspec/changes/skshield/merge-lines.txt` (BACKLOG, INDEX, CHANGELOG); also for docs/decomp-queue.md item 15: `SKSHIELD done (docs/research/skshield-trace.md): shield never a hand item, attached active; no base damage (306/306 weapons vs 0/69 armor rows); flats and ED reach the player; D1 = off-weapon ED (existing O part, 156/158 ... 151/151), D2 broken item stats (V read + V run); D7 settled as A; no PR candidate built`.
