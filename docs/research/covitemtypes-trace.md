# Item types (itemtypes.txt) in game.exe 1.14d: d2d and our bot compared

> status: research, 2026-11-02, agent covitemtypes (card covitemtypes; coverage gap 10 `itemtypes`, first pass). Offline only: no game, no d2d slot, nothing pushed, posted or filed. Tags: V run = game.exe's own code run natively under the emulator and compared byte for byte with the real d2d code (commands below); V read = read in the disassembly (`tools/emu/cit_dis.sh`); D = read from the patched 1.14d table (`mpq-table.sh itemtypes|weapons|armor|misc`, patch layer `Patch_D2.mpq`); S = upstream's docs; I = inferred; U = unverified.

**Answer.** On the stock 1.14d data d2d's item-type rules equal game.exe's on every case tried: the Equiv chain ("is type A a type B"), the item-to-type membership, the equip slot of an item, the two-hands rule, the socket cap, the Beltable and class gates, and the table columns it loads. No d2d deviation is proven (no PR candidate). Five latent deviations exist in the code shape (own row versus chain, Equiv2 skipped, depth cap, row 0, duplicate codes); the stock table never triggers them. Our bot differs from game.exe in two places (B1, B2: the socket cap; B3: the throwable flag).

## What was run (V run)

Oracle `tools/emu/cit_oracle.py` runs game.exe's functions natively on the loaded 1.14d tables, NO hooks, with fake units `{4, class}` (and `{0, class}` for the player). d2d side `tools/emu/cit_d2d_test.cpp` (built by `tools/emu/cit_build.sh` against the scratch upstream build `sorcrest-d2d`, whose `rules.hpp` already has the `shoots`/`quiver` fields and cites FUN_0063dbc0; the `src` checkout is older, see D7) calls d2d's own `rules::type_is`, `type_class`, `compcode::Types::isa`, `can_wear`, `hands_ok`, `most_sockets` and the `put_in_belt` Beltable walk on the real loaded tables (`load_game_data`). Compare scripts: `cit_diff.py`, `cit_diff2.py`, `cit_diff3.py`, `cit_diff4.py` (plain python, counts only).

```
tools/emu/run-on-worker.sh mpq-table.sh itemtypes > it.txt        (also weapons armor misc)
tools/emu/run-on-worker.sh cit_oracle.py rows|isa|item|item2|hands|slot|sock > game files
tools/emu/run-on-worker.sh cit_build.sh isa|item|belt|flags|item2|hands|slot|sock > d2d files
python3 -I tools/emu/cit_diff.py DIR ; cit_diff2.py DIR ; cit_diff3.py DIR ; cit_diff4.py DIR
```

| Run | Game function | Cases | Result |
|---|---|---|---|
| record dump | the loaded ItemTypes records (`sgptDataTables+0xbf8`, 0xe4 bytes, count `+0xbfc`) | 103 records x 28 columns | every column has one record offset (Q2) |
| is-a pairs | `FUN_00504a20(a, b)` | 10,609 row pairs; 9,604 with both codes | d2d `rules::type_is` 0 mismatches, `compcode::Types::isa` 0 mismatches |
| membership | `FUN_00629bb0(item, type row)` | 659 Items records x 98 coded type rows = 64,582 cells | d2d `type_is(type) or type_is(type2)` 0 mismatches; the 659 type and type2 ids equal d2d's |
| matrix vs walk (game only) | 629bb0 versus 504a20 on type and type2 | 67,877 cells | 0 mismatches |
| own-row getters | `FUN_0062ea80` BodyLoc1/2, `0062e9e0` Magic, `0062e990` Rare, `0062ea30` Normal, `0062c060` class-specific | 659 x 6 = 3,954 cells | 0 mismatches vs d2d body / always_magic / can_rare / always_normal / class chain |
| Beltable | `FUN_0062bad0` | 659 | d2d `put_in_belt` chain 0 mismatches; 27 beltable items |
| slot fit | `FUN_0062ed50(item, slot 0..12)` | 659 x 13 = 8,567 | slots 1..10: 6,590 cells 0 mismatches (class and requirement gates on, identified, wearer of the item's class with 999 str/dex/level); the other 870 cells are D6 |
| two hands | `FUN_0063dbc0(A, B, unit)` | 362 items (type BodyLoc1 = right arm) squared x classes 1, 4, 6 = 393,132 | d2d `hands_ok` 0 mismatches (game true 92,362, d2d true 92,362) |
| socket cap | `FUN_0062bc20(item)` | 659 x ilvl 0..110 = 73,149 | d2d `most_sockets` 0 mismatches |

Total cases for the ledger line: 9,604 + 64,582 + 3,954 + 8,567 + 393,132 + 73,149 + 659 = **553,647** (the game-only matrix cross-check, 67,877, is not counted). Seeds: none (every run is an exhaustive grid, not random).

Not run: the d2d `store_page`, `autostack`, `throwable`, charm and gem flags (read from the txt as loaded; Q6), the game's stack and AutoStack path (U), the vendor's StorePage reader (`FUN_0062e8b0` area, U), the equip requirement function `FUN_0062eaf0` (upstream already traced it, S: `docs/research/re/hireling.md`).

## Q1. How does game.exe answer "is type A a type B"? (V read + V run)

`FUN_00504a20(a, b)` (stack args, `ret 8`), table pointer `[0x87d82c]`, record stride 0xe4, Equiv1 the word at +4, Equiv2 the word at +6 (id 0 = none):
- `b == 0` returns 1 (0x504a2d..0x504a38); `a == 0` returns 0 (0x504a3e..0x504a46).
- Otherwise a stack walk: push `a`; pop `x`; `x == b` returns 1 (0x504a6c, 0x504ab4); push Equiv1 of `x` and, ONLY when Equiv1 is non-zero, Equiv2 (the `test dx,dx; je 0x504aa3` at 0x504a7b skips both words when Equiv1 is 0, D1); pending count capped at 0x40: past it an error path (0x504ac1: message 0xb52, then exit -1, D4). Nodes are not marked, so a diamond is walked twice (harmless).
- Callers seen: the compcode setup at 0x506000 (weapon 0x2d, torso 3, shield 0x33, helm 0x25, circlet 0x4b). The matrix of `FUN_00629bb0` below is what the item rules use; where it is filled from this walk was not read (I: the 67,877-cell run equals the walk).

`FUN_00629bb0(item, type id)` (the one almost every rule calls): returns 1 when the bit `[matrix + (row(type of item) * width + id / 32) * 4]` of the precomputed closure matrix (`sgptDataTables+0xc04`, 103 rows, width 4 dwords) is set; if not, it tries the Items record's type2 (word at +0x120) the same way (0x629c27..0x629c64). (V read.) The matrix is the closure of 504a20 plus row 0: V run, 67,877 cells equal 504a20(type) or 504a20(type2), including column 0, which is set for every item (659 cells). Row 0 is the blank "None" row.

Facts of the stock table (D, `cit_diff2.py`): 104 rows in `itemtypes.txt` including the "Expansion" separator row, which game.exe's loader drops (103 records) and d2d's `txt::Table` drops too, so the row indexes agree; 98 rows have a Code, 5 game rows are blank (ids 0, 1, 14, 17, 23: "None", "None", "Not Used" x3); no duplicate codes; no row has Equiv2 without Equiv1; every Equiv names an existing Code; the longest chain is 5 deep; 221 proper is-a pairs between coded rows; no self-equiv.

V run result: 9,604 coded pairs, d2d `rules::type_is` (string walk, depth cap 8) and `compcode::Types::isa` (index walk) both equal 504a20, 0 mismatches. Pairs touching a blank row (game only): a coded and b = 0: 1 (98 pairs); a blank other than 0 and b = 0: 1 (4 pairs, plus the 0/0 pair); a = 0 and b coded: 0 (98); a blank and b coded: 0 (392); a coded and b blank other than 0: 0 (392); a blank and b blank, a != b, b != 0: 0 (16); a blank equal to b: 1 (4). d2d has no counterpart for blank rows (D3).

## Q2. Where are the columns in the record? (V run)

Matched by value over all 103 records (`cit_diff.py` section 1; a column is placed only when one byte equals the text for every row): Equiv1 +4 (word), Equiv2 +6 (word), Repair +8, Body +9, BodyLoc1 +0xa, BodyLoc2 +0xb (values are BodyLocs.txt rows: head 1, neck 2, tors 3, rarm 4, larm 5, rrin 6, lrin 7, belt 8, feet 9, glov 10; equal to d2d's `body_slot` numbering), Shoots +0xc (word: bowq 5, xboq 6), Quiver +0xe (word: bow 27, xbow 35), Throwable +0x10, Reload +0x11, ReEquip +0x12, AutoStack +0x13, Magic +0x14, Rare +0x15, Normal +0x16, Charm +0x17, Gem +0x18, Beltable +0x19, MaxSock1/25/40 +0x1a/0x1b/0x1c, TreasureClass +0x1d, Rarity +0x1e, StaffMods +0x1f (0..6 = ama..ass, 255 none), CostFormula +0x20, Class +0x21 (0..6 as StaffMods, 255 none), StorePage +0x22 (armo 0, weap 1, misc 3, blank 255), VarInvGfx +0x23. Blank flag cells read as 0; blank class, storepage and staffmods as 255. d2d loads the same columns by name (`gamedata_load.cpp`, text): 98 types x 14 fields, 0 mismatches against the table text (`cit_diff2.py` last section); the d2d numbering of StorePage (0, 1, 3, -1) and StaffMods equals the game byte.

## Q3. What does game.exe use the type flags for, and does d2d do the same? (V read, V run on the stock data)

| Rule | game.exe | d2d | Result |
|---|---|---|---|
| Beltable | `FUN_0062bad0`: the item's OWN type row +0x19, no chain | `put_in_belt` walks the Equiv chain | equal on the data (D: Beltable types are poti elix scro hpot mpot rpot spot apot wpot; no type gets it only by chain); 659 items, 0 diff (D2) |
| Throwable | `FUN_0062ba80`: own row +0x10 | `ItemType.throwable`, own row | 36 items (the jave/tkni/taxe/ajav/tpot/thro group); read, equal by column |
| class-only item | `FUN_0062c060`: own row +0x21 < 7 | `type_class` walks the chain | equal on the data (no type has a class only through its chain); 659 items, 0 diff (D2) |
| Magic / Rare / Normal | own row +0x14 / +0x15 / +0x16 getters (`0062e9e0`, `0062e990`, `0062ea30`) | `always_magic`, `can_rare`, `always_normal` | 659 items, 0 diff; 153 items have Rare 0, 7 Magic 1, 121 Normal 1 |
| equip slot | `FUN_0062ed50(item, slot)`: slot == BodyLoc1 or BodyLoc2, or slot 0xb / 0xc when a BodyLoc is 4 or 5 (weapon switch) | `can_wear`: BodyLoc1/2 == slot (1..10) | slots 1..10 equal (6,590 cells); 11 and 12 and 0 see D6 |
| two hands | `FUN_0063dbc0(A, B, unit)`, below | `hands_ok` | 393,132 cases, 0 diff |
| socket cap | `FUN_0062bc20` (below) | `most_sockets` | 73,149 cases, 0 diff |
| StorePage | value +0x22 | `store_page` | same numbering; the vendor reader not run (U) |
| AutoStack | +0x13 (the pickup's stacking step `FUN_00560020`, belt-pickup-trace.md Q1) | `autostack` and `stackable` and kind == 2 | not compared (U) |

`FUN_0063dbc0` (esi = A, edi = B, ebx = the unit; plain `ret`; called from the body placement `FUN_0063dd90` via `FUN_0063de60(unit, slot, item, flag)`), V read:
1. `FUN_0062e6f0(A)` = A's type Shoots (+0xc, signed word); when it is non-zero and B is of that type (`629bb0(B, shoots)`), return 1; the same with B shooting at A.
2. `FUN_0062e740` = Quiver (+0xe): if A or B has a non-zero Quiver column (a quiver not matched in step 1), return 0.
3. `FUN_006289c0` (an Items-record flag through `FUN_00628980`, not read: U, presumably 2handed) and `FUN_0062a1e0` (a player of class 4, Barbarian, and Items +0x13d, presumably 1or2handed: U): a two-hander blocks the other hand unless it is a Barbarian's 1or2handed item (0x63dc2a..0x63dc56). The 393,132-case run shows d2d's `two_handed` / `one_or_two` reading equal on every pair.
4. Both items weapons (`629bb0(x, 0x2d)`): unit nil returns 0; unit type 1 (a hireling, class dispatch table at 0x63dcfc, not decoded: U); a player (type 0): class 4 returns 1, class 6 (Assassin) returns 1 only when both are h2h (0x43); else 0.
5. Not both weapons: 1 when either is a weapon (weapon plus shield or plus anything non-weapon), else 0.

`FUN_0062bc20(item)` returns min(Items gemsockets +0x138, MaxSock[tier]) with tier by the item level at item data +0x2c (floor 1 via `FUN_006281e0`): level <= 0x19 (25) uses MaxSock1, <= 0x28 (40) MaxSock25, above MaxSock40 (0x62bc81..0x62bc97); a cap of 0 returns 0 (V read; V run on 659 x 111 cases).

## Deviation list (d2d vs game.exe)

All five are latent: the stock table never reaches them, so each is tagged V read (code) + D (data shows it does not trigger) and no differential run can show a difference today.

| # | Game | d2d | Tag | Differential |
|---|---|---|---|---|
| D1 | `504a20` walks Equiv2 only when Equiv1 is non-zero | `type_is` and `Types::isa` walk both | V read; data D: 0 rows have Equiv2 without Equiv1 | could be built: copy the table with one row's Equiv1 blanked and Equiv2 set (a patch-layer file in a scratch `D2_MPQ_DIR`), run `cit_oracle.py isa` and `cit_build.sh isa` on it; expected 1 pair differs |
| D2 | Beltable, class-only, Throwable, Magic, Rare, Normal read the item's OWN type row | Beltable (`put_in_belt`) and class (`type_class`) walk the chain | V read; V run 0 diff on 659 items; data D: no type has either only by chain | same shape: a scratch table with a child of `poti` whose own Beltable is 0; oracle `item` mode column 5 against `cit_build.sh belt` |
| D3 | row 0 ("None") is a type id; `504a20(a, 0)` is 1, the matrix column 0 is set for every item | blank-code rows are not indexed | V run (the game side counts above); no item or caller reaches it | not needed |
| D4 | more than 64 pending nodes is a fatal error (0x504ac1) | `type_is` stops at depth 8 and answers false; `Types::isa` unbounded | V read; longest chain 5 | a synthetic 9-deep chain in a scratch table |
| D5 | rows are ids; the first row with a code is the id | `tables.types[code]` keeps the LAST row of a code, `Types::index_.emplace` the FIRST (and our bot the first) | I (read); no duplicate code in the table | scratch table with a duplicate code |
| D6 | slots 0xb and 0xc are the weapon-switch hands, accepted for any arm item by `62ed50`; slot 0 accepts items with no BodyLoc | `can_wear` takes slots 1..10; the switch is a set showing (`d2s.hpp` +0x10) | V run: 870 cells (slot 0 for 146 items, slots 11 and 12 for 362 items each); a modelling difference, not a rule | not comparable by slot; the switch swap is a different function (`docs/research/re/weapon-switch.md`) |
| D7 | `63dbc0` as above | the older upstream `src` checkout's `hands_ok` (read, not run) let any `misl` quiver join any two-hander and a one-hander join a mismatched quiver; the newer `sorcrest-d2d` build has the Shoots/Quiver rule and runs 0 diff | I (older code read only) | already covered: the 393,132-case run is on the newer build; if the older rule is still in any branch, run `cit_build.sh hands` against it (expected differences: bow with xboq, sword with bowq) |

## Bot findings (described, NOT changed)

| # | Bot | Rule | Tag |
|---|---|---|---|
| B1 | `internal/town/socket.go:19-25` (`ExpectedSockets`): `ilvl >= 40` tier 2, `ilvl >= 25` tier 1 | game and d2d: `ilvl <= 25` tier 0, `<= 40` tier 1, above tier 2 | V run (game side) + model of the Go code (I). Model vs game on ilvl 1..110: 553 of 72,490 cells differ; ilvl 25: 235 items (e.g. `2ax` Double Axe 4 on game, 5 on the bot; `6cb` Hydra Bow 3 vs 4), ilvl 40: 102 items. Also the comment on `internal/data/items.go:19` (`ilvl <25, 25-39, >=40`) is the same off-by-one |
| B2 | same function: `if m := ty.MaxSock[tier]; m > 0 && m < n` ignores a cap of 0 | game returns min(gemsockets, cap) = 0 | V run: ring `rin` and amulet `amu` (gemsockets 1, MaxSock 0/0/0): game 0 at every ilvl, bot model 1 (216 cells). Only matters if a ring or amulet is socketed or planned |
| B3 | `internal/combat/melee.go:57` (`w.Throwable || IsA(rangedType)`): `ItemBase.Throwable` is read from a `throwable` column (`internal/data/items.go:128`, columns list line 108) | game: `FUN_0062ba80`, the TYPE row's Throwable (+0x10): 36 items (javelins, throwing knives and axes, the exceptional and elite sets, the tpot potions) | D: the patched `weapons.txt` has no `throwable` column (armor.txt and misc.txt do), so for a weapon the bot's flag is false: a javelin or throwing knife in hand counts as a melee blow (`b.Weapon = true`). `internal/gear/gear.go:135` uses the type row correctly. Effect U (a Sorceress rarely wields one) |
| B4 | three IsA implementations: `internal/data/db.go:350`, `internal/gear/gear.go:97`, `internal/gear/tables.go:272` | reachability, as 504a20 | I (read): all three are reachability over Equiv1/Equiv2 with a seen set, same answers as game.exe on the stock chain; no run (no Go change allowed here) |
| B5 | `internal/gear/tables.go:268`: `Body: nonEmpty(BodyLoc1, BodyLoc2)` keeps only the first body location | rings have two (rrin, lrin), BodyLoc2 matters for the slot | I (read); where `Body` is used was not traced (U) |
| B6 | `internal/town/ammo.go:139`, `internal/data/price.go:162` (`Quiver != ""` as the quiver test) | game: `FUN_0062e740` (Quiver column non-zero) = bowq, xboq, mboq, mxbq | V read + D: equal on the table (the four quiver types are the four with a Quiver value) |

## PR candidates

No PR candidate: no d2d deviation is proven by the existing differential (every d2d comparison above shows 0 mismatches; D1-D5 do not trigger on stock data).

## Dead ends

- The first run of `isa` printed zeros: the emulator does not run the loader at 0x504920 that fills `[0x87d82c]`; the oracle sets the global to `sgptDataTables+0xbf8`, whose records have the same layout (Equiv1 +4, Equiv2 +6, checked in `rows`). The second copy that the real client builds is therefore the same data by assumption (I), not run.
- `cit_d2d_test --item` first used a 104-entry table index that included the "Expansion" row against the game's 103 ids; d2d's table drops that row, so its indexes equal the game's. Found by 1,577 false mismatches on `gemz`, not a real difference.
- d2d `can_wear` with an unidentified item (the default) refuses everything; the probe sets `identified`.
- The unit-type 1 (hireling) branch of 63dbc0 (dispatch at 0x63dcfc) was read to its start only.

## Drill recipe for the fork verbs (U, not run; `docs/forkdbg.md`, `docs/forkcheats.md`)

1. Socket cap (B1, B2): `debug give 2ax pack q4 l25` (a magic Double Axe, gemsockets 5, at item level 25) and take Larzuk's reward (Act 5, Siege): the cap `FUN_0062bc20` gives is 4 at level 25 (MaxSock1) and 5 at level 26 (MaxSock25); `dump items` shows the sockets. That Larzuk's reward uses this function is S (`docs/research/re/quests-act5.md`), not run. The bot's model says 5 at level 25. Also a ring (`rin`): cap 0. Use a throwaway character `--no-save`; every verb is ledgered.
2. Throwable melee (B3): `debug give jav` then equip it and `dump stats` / a left click on a monster: game.exe throws it (type Throwable), the bot's melee blow would not.
3. Two hands (checks the 393,132-case result live): `debug give` a bow, `aqv` and `cqv` into the pack, equip each pair; the game accepts bow plus aqv only.
4. D1, D2: need a patched table in a scratch install; no live check planned.

## Matrix-cell updates (proposed)

No matrix doc covers item types. Proposed `docs/coverage/rule-coverage.md` effect (generated): `itemtypes` 98 of 98 rows run-tested 0 diff (aspect: Equiv chain, item membership, slot, hands, socket cap, flags; cases 553,647). Proposed new note for `docs/research/price-trace.md` D10: the repair gate's ItemTypes Repair flag (+0x8) has the same column offset as found here (V run), the gate itself is not rerun.

## Limits

- Row coverage here is the stock 1.14d table; the latent deviations only matter for a modded table.
- Run on the emulator with fake units; the live server path (what a vendor lists, what a pickup stacks) is not run.
- d2d side built from the scratch upstream build named above, not from a PR branch; it is newer than the `src` checkout.
- The ledger counts one line for all 98 rows although the aspects differ per type (a type that no item uses, such as the pure parents `weap`, `armo`, `misc`, is exercised by the is-a pairs and the membership matrix only).

## Glossary term missing

"closure matrix" (the precomputed type-by-type bit table at `sgptDataTables+0xc04`).

## Shared-file lines for the merger

See `openspec/changes/covitemtypes/merge-lines.txt`.
