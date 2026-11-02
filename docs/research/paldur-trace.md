# Durability loss per hit and per block: game.exe 1.14d vs d2d vs the bot (decomp item PALDUR)

> status: current (2026-11-02, agent paldur, Sonnet, first pass; no independent verifier yet). Offline: no game, no d2d slot, nothing pushed or posted. Tags: **V read** = read in game.exe's disassembly (`vq_dis.py` = upstream `disasm.py`) or in d2d's code at file:line; **V run** = game.exe's own function run under unicorn by `tools/emu/pal_wear.py` and compared with d2d's real `Fight` code; **D** = 1.14d patched tables (ItemTypes, Weapons, Armor); **S** = an upstream doc or an earlier trace; **I** = inference; **U** = unverified. d2d = the upstream main read tree and the sorcrest scratch build (it holds `Fight::wear_weapon / wear_armor / wear`, `fight.hpp:784-795`). No game code is pasted here: function tags and short formulas only.

## Summary

1. **The whole rule is three functions** (V read, V run): `FUN_0057d4f0` (the on-hit post step, 35 direct callers: the player's Attack, every melee skill, the monster attack handler) calls `FUN_0057d3d0` only when the blow's result record has bit 0 (the blow hit) set; `FUN_0057d3d0` wears the attacker's weapon when the attacker is a player, and one armor piece of the defender when the defender is a player; `FUN_00559e30` rolls and takes one durability point.
2. **Chance: armor 10 % per landed hit, weapon 4 % per landed hit, a throwing stack 10 %** (V run). The armor piece is picked by weight among the worn armor (V run: table at 0x732b90: head 3, torso 5, right arm 4, left arm 4, belt 2, feet 2, gloves 2; a right-hand weapon is not armor, so a Paladin with a shield draws from head, torso, shield, belt, feet, gloves = 18). One point per loss. The piece picked is not re-drawn when it cannot wear (indestructible, no durability): the point is simply not lost.
3. **A blocked blow wears nothing, neither the blocker's shield nor anything else; neither does a dodge, avoid or evade or a miss** (V run on `FUN_0057ec10`: any non-zero answer of the block function clears the hit bit; `FUN_0057d4f0` reads that bit). Only an unblocked, landed blow costs durability.
4. **d2d has the same rule for melee and for the player's blow**: 0 differences in 34,939 armor cases and 0 in the plain-weapon cases of 15,253 weapon cases (seed 7; seed 1: 13,876 and 6,316), the selection, the roll, the break rules and the seed stream included. The player's own seed is consumed by the same number of draws in all but one state (D3).
5. **Where d2d deviates (see D1..D6):** it also wears armor on monster missiles, corpse explosions and Fire Blast (game.exe's missile code has no path to the wear function: I); game.exe never wears a throwing stack while game dword +0x70 is 0 (d2d has no such gate; meaning of +0x70 U); a broken armor picked costs game.exe a seed draw, not d2d (no outcome effect).
6. **No PR candidate**: the one proven d2d difference (javelin stacks with +0x70 = 0, 38 of 15,253) rests on an input whose live value is unknown (U), and D1 (missile wear) is inference from a call-graph, not a run.

## Q1 Who calls the wear (V read, call-graph scan V run)

| Step | game.exe | What it does |
|---|---|---|
| Call sites | `FUN_0057d4f0` (ECX game, EDX attacker, stack: defender, 0) | 35 direct callers (`pal_callgraph.py`): `FUN_0056f070` (the plain Attack, via its tail at 0x56f1da, S sk-melee-trace.md "Charge-ups"), `FUN_0056c2d0`, 0x56f1f0 (a second attack function, call at 0x56f44a), `FUN_005a7670` (the monster attack handler: after a monster's melee blow `FUN_005a5490`, 0x5a7792; S monster-missile-trace.md Q1), and 31 skill do-functions 0x5c4cd0..0x5dbe50 (the melee skills; which skill is which was not mapped, U) |
| Wear only from here | `FUN_0057d3d0` has exactly one direct caller (0x57d5f6, inside `FUN_0057d4f0`), `FUN_00559e30` exactly one (`FUN_0057d3d0`: 0x57d3f3 weapon, 0x57d4d8 armor) | V read, grep of the full linear disassembly (1,068,553 lines, `vq_dis.py --all`) and `pal_callgraph.py` |
| Record | 0x57d4f0..0x57d563 | walks the attacker's list at unit +0xac for the entry whose (type, id) pair names the attacker and the defender, copies its 0x70-byte hit record; none found: return. Then `FUN_00622c40(attacker, defender, 3 if the attacker is a monster else 1)` must be non-zero (the "in reach" test of `FUN_00622c40`, S sk-melee-trace.md Q1; the branch taken when it is 0, 0x57d670, was not read: U whether the on-hit events still run) |
| The gate | 0x57d591 `test record +4, 1` | clear: only the two on-hit events (7 and 3) run, `FUN_0057d3d0` is NOT called. Set: a player or monster attacker whose unit +0x10 (its mode, I) is 0 (dead) ends the step; else events and `FUN_0057d3d0(game, attacker, defender)` (0x57d5f6) | V read |

## Q2 The record's bit 0: what clears it (V run)

`FUN_0057ec10(ECX game, EDX attacker; defender, bonus, range)` builds the result flags the record keeps at +4: bit 0 when the to-hit step (`FUN_0057d9b0`) hits (a player defender in mode 3 RN is hit without a roll, S sk-melee-trace.md), then `FUN_0057dfb0` (block, dodge; can-block 1) answers: bit 1 sets 0x10 (the blocked result), 2 sets 0x100, 4 sets 0x80, 0x10 sets 0x8000, and **any non-zero answer clears bit 0** (0x57ed12..0x57ed21); bit 2 is added at the end when state 0x36 is absent (0x57ed27). `python3 pal_wear.py flags` runs the real function with the three sub-steps hooked (to-hit roll 0 or 1, block answer 0, 1, 2, 4, 0x10, 3; defender a monster, a player, a running player): 36 runs, result `0x0005` for answer 0 with a hit, `0x0010`, `0x0100`, `0x0080`, `0x8000`, `0x0110` for the others (bit 0 clear in all of them), `0x0000` without a hit (a running player gets 0x0005 from a missed roll, as before). The block function's own answer (`FUN_0057dfb0` returns 1 when its roll beats the block chance, else it hands over to `FUN_0057dd60`) is V read; what `FUN_0057dd60` returns for dodge, avoid and evade (the bit 2, 4 and 0x10 answers) was not read: U (they too clear the hit bit, V run, whatever their cause).

Tag: V run for "a blocked or avoided blow does not reach `FUN_0057d3d0`", V read for "block is answer 1".

## Q3 Which item loses durability (V run, hooks named)

`FUN_0057d3d0(EAX unit A; stack game, unit B)`:

- **A is a player (type 0): its weapon.** `FUN_00535bc0(A)` names the hand (S sk-melee-trace.md: "the hand the current skill uses"); no weapon (barehanded) means no wear. `FUN_00559e30(game, A, item)`.
- **B is a player: one armor piece.** The table at 0x732b90 (count at 0x732bc8 = 7, `pal_wear.py table`, V run: entry body location / weight = 1/3, 3/5, 4/4, 5/4, 8/2, 9/2, 10/2 = head, torso, right arm, left arm, belt, feet, gloves in the order d2d lists them). `FUN_0063bde0(inventory, location)` gives the item; it counts when `FUN_00629bb0(item, 0x32)` holds (is of ItemTypes row 50 = Any Armor, D: the dump shows row 50 `armo`, row 45 `weap`) and the item passes the durability gate; the weight sum of those is W. Then **a start slot = rand(7) and a pick = rand(W)** on B's seed (+0x20 / +0x24, `FUN_0045c3e0`), and the walk from the start slot around the ring: skip a slot with no armor, take it when pick < its weight, else pick -= weight. The weights are therefore the true odds (the ring walk starts at a random place but the pick is not tied to it: V run, d2d's version is the same walk and gave 0 differences with the seed carried).
- An item that is not armor (a right-hand weapon) gets weight 0; a Paladin with a shield and full armor: head 3/18, torso 5/18, shield 4/18, belt, feet, gloves 2/18 each; a two-hander (no shield): total 14.

The slot is drawn before the item is tested; if the item cannot lose durability (stat 152 indestructible, nodurability, no max) the draw is spent and nothing happens (V run: `gate=0` cases, same in d2d).

## Q4 The chance and the loss (V run)

`FUN_00559e30(ECX game, EDX unit, item)`:

1. `FUN_00629bb0(item, 0x32)` armor or `(item, 0x2d)` weapon (ItemTypes row 45 = Weapon, D), else return. `FUN_00629930(item)` (the durability gate: S price-trace.md, pr-price2-proof.md: nodurability 0, durability column > 0, stat 0x48 max > 0, stat 152 <= 0), else return.
2. Chance: **armor 10; a weapon 4; a throwable weapon (`FUN_0062ba80`: the Throwable column of the item's own ItemTypes row, I for the column, D: equal to d2d's rule, see D5) 10, but only when game dword +0x70 is not 0**; with +0x70 = 0 it returns before any roll (D2).
3. Roll: the unit's seed steps once (the usual LCG, `lo x 0x6ac690c5 + hi`), the roll is the new low dword % 100; `roll >= chance` returns (so a roll of 0..9 wears armor, 0..3 a weapon).
4. Durability stat 0x48: new = dur - 1. **new > 0:** capped at the max (`FUN_00625e00`: base max plus the max-durability stats), stored. **new <= 0 and an armor** that is not already flagged (item flag 0x100, `FUN_006280a0`): `FUN_0055f850` breaks it (sets flag 0x100, takes it off and back on, stat 0x48 = 0, a sound, a stat recompute): the armor stays on the body, disabled. **new <= 0 and not a stack** (a weapon, or an armor already flagged): stat 0x48 = 0 only: **a weapon at 0 durability is not flagged broken** and stays in the hand. **A stack** (`FUN_0062a670` >= 1): quantity (stat 0x46) 1 -> 0 takes the item out (`FUN_00580310`, out of ammo); more than 1: quantity - 1 and durability back to the max.
5. Events (a notification each, `FUN_0053d130`) change no number.

d2d: `Fight::wear` (`fight.cpp:2482`): the same ladder (armor or throwable 10, else 4; durability - 1 when that is at least 1, capped at `max_durability`; armor breaks with flag 0x100; a weapon stays at 0; a stack loses one and comes back whole, its last goes out of ammo), `wear_armor` (`fight.cpp:2512`) the same table and the same two draws, `wear_weapon` (`fight.cpp:2505`) the first weapon in either arm slot.

## Q5 The differential (V run)

Commands (the worker host; `tools/emu/run-on-worker.sh`):

- `pal_build.sh` compiles `pal_wear_test.cpp` (the real `Fight` on a Paladin, the World's rng seeded from each case) against the sorcrest scratch build.
- `pal_run.sh N SEED`: `pal_wear.py gen N SEED` (192 grid cases plus N random ones; W: weapon or javelin stack, durability 0..200, max 1..250, quantity 1..6, gate, game +0x70 0 or 1; A: seven slots present or empty, each with durability, max, gate, and a broken flag only at durability 0), `pal_wear.py game` (the real `FUN_0057d3d0` + `FUN_00559e30` + `FUN_0045c3e0`, the items faked and hooked: see the docstring), the d2d binary, `pal_wear.py compare`. Case file md5 for seed 1 / 20,000 (20,192 cases): 94ebf4709f3708a7fa83cf291403eed4; seed 7 / 50,000 (50,192 cases): 1bf108303bb33a6771ebd8a4e5fd5ffe.

| run | cases (W / A) | items' states differ | seed only differs |
|---|---|---|---|
| seed 1, N 20,000 | 20,192 (6,316 / 13,876) | 20, all javelin stacks with +0x70 = 0 (of 191 such cases) | 805: 634 armor cases with a broken armor present, 171 javelin stacks with +0x70 = 0 |
| seed 7, N 50,000 | 50,192 (15,253 / 34,939) | 38, all javelin stacks with +0x70 = 0 (of 423) | 2,091: 1,706 armor cases with a broken armor present, 385 javelin stacks with +0x70 = 0 |

Everything else is equal: armor 0 differ in 13,876 and 34,939 cases (the 5 % to 10 % of cases with an indestructible, gate-less or broken piece included); plain weapons and javelin stacks with +0x70 = 1: 0 differ, the seed stream too. Hooked, so NOT in the claim: the item side (type tests, the gate, the stat store, `FUN_00535bc0`, `FUN_0063bde0`), see the docstring; the real code run is the selection loop, the roll, and the durability ladder.

## Deviations d2d vs game.exe (D1..D6)

| id | rule in game.exe | rule in d2d | tag | differential |
|---|---|---|---|---|
| D1 | armor wears only from `FUN_0057d4f0`; the missile hit chain (entry `FUN_005adf10` and the 2,081 functions it reaches by direct calls), the hit builder `FUN_005a89a0`, the on-hit events `FUN_0057c6c0` and the apply step `FUN_0057dbf0` reach none of `FUN_0057d4f0`, `FUN_0057d3d0`, `FUN_00559e30` (`pal_callgraph.py`); the monster's missile branch (`FUN_005a6d50`) does not either. A hit by an arrow or a spell therefore wears nothing. | `Foe::take` counts every `taken.hit` (`ai.hpp:425`) and `Fight` wears one armor piece per count (`fight.cpp:4774`): monster missiles (`ai.hpp:796-797`), the corpse explosion (`fight/deaths.cpp:172`), Fire Blast (`deaths.cpp:191`) and the foe loops of the melee branches | I (direct edges only: pointer tables, e.g. the skill do-functions and the missile hit functions, are not followed; V run of the scan itself) | Could be built: run the real missile damage chain (`mon_missile.py`'s hooks, S monsterhit-trace.md) on a player target with sentinels at 0x57d4f0 and 0x559e30 and count entries; or the lab drill R3 below (d2d side only). Not built. |
| D2 | a throwing stack (the item's own Throwable column) wears only when game dword +0x70 is not 0 (`FUN_00559e30` 0x559e7b..0x559e8b) | no such gate | V run (38 of 15,253 weapon cases, javelin stacks with +0x70 = 0); the live value of +0x70 is U (the stock emulator game has it non-zero, S monster-stats-trace.md; it also gates the monster area level there) | built (`pal_run.sh`); to settle the input: read game +0x70 live (R6) |
| D3 | a broken armor (flag 0x100, durability 0) that the slot walk picks still costs the roll and one seed draw, no change | `Fight::wear` returns at `broken()` before the roll (`fight.cpp:2489`) | V run (1,706 of 34,939 armor cases, seed only; the items' states equal) | built; no outcome effect, only the player's later seed stream |
| D4 | the gate is `FUN_00629930` over the ItemTypes / items.txt columns and the whole stat list (stat 152) | the item's own `max_durability` field, `rules::indestructible` over `item.props` (`fight.cpp:2489`) | I (real items carry max 0 when the column says none; weapons refused by the column: 49 bases (bows, crossbows, throwing potions and some Amazon bases), armor 0 refused, D `pal_tables.py`; javelins and throwing knives pass); price-trace.md already ran `FUN_00629930` natively with the tables (S) | could be built: run `FUN_00629930` on generated items against d2d's `has_durability`; not needed unless a mod changes the tables |
| D5 | throwable = the Throwable column of the item's own ItemTypes row | `throwable()` = weapons.txt missiletype > 0 and stackable (`fight.hpp:96`) | D (36 weapon bases by each rule, the same 36 bases, `pal_tables.py`): equal in the patched tables; I for the column reading of `FUN_0062ba80` (record +0x10 byte) | table check done; no run needed |
| D6 | the weapon is `FUN_00535bc0`'s: the hand the current skill uses (S sk-melee-trace.md) | the first weapon found in either arm slot, in item order (`fight.cpp:2505`) | U (a dual wielder: Barbarian, Assassin with claws) | could be built: run `FUN_00535bc0` natively with a fake inventory (needs `FUN_0063c9b0`, `FUN_006235a0`, `FUN_00620250` hooked); not built; irrelevant to a Paladin |

## What d2d does the same (not deviations)

- Landed melee hit wears one point at 10 % (armor) or 4 % (weapon); blocked, dodged, missed: nothing (d2d: `combat.hpp:303` and `:409-413` return before `hit = true`; V read) (V run for the game side, Q2).
- The player's blow wears the weapon once per landed hit (`fight.cpp:1005`); each landed monster blow wears one armor piece (`fight.cpp:4774`).
- Break rules, cap at the max, the stack ladder, the weighted slot walk with two draws: 0 differences (Q5).
- `debug wear` halves durability (`devctl_verbs.hpp:642`): a cheat, not a rule.

## Bot findings (B1..B4): described, NOT changed

| id | finding | where |
|---|---|---|
| B1 | The repair trigger reads the lowest durability percent over every equipped item, the weapon included: needRepair at below 30 % (knob `repairBelowPct`), the trip at below 20 % (`tripRepairPct`). With the rule above (per landed hit, a Paladin with a shield: head 1/60, torso 1/36, shield 1/45, belt, feet, gloves 1/90 of a point; the weapon 1/25 per landed swing; arithmetic on the V run rule, I) a quilted armor (20) takes about 540 landed hits from full to below 30 % and a hand axe (28) about 500 landed swings. Blocked hits cost nothing, so a high-block build repairs less often. The bot never needs a model of the loss (it reads the stats), but a cost-per-hour estimate for the repair trip can use these odds. | `internal/town/chores.go:110` (lowestDurability), `:183-184` (needRepair), `:103` (trip); `internal/env/tuning_town.go:22`, `:61` (30 and 20) |
| B2 | A weapon at 0 durability is not flagged broken and stays in the hand (Q4): the bot sees durability 0 percent and repairs it (right); it does not see a "broken" flag on weapons, so it must not wait for one. Armor at 0 is flagged and disabled. | `internal/town/shop.go:147-152` (durabilityPct: 0 when the stat is 0) |
| B3 | No drill lets the game wear an item: every repair drill damages by the `debug durability` cheat; the live repair effect and the loss rule stay unconfirmed on the lab binary (`docs/town.md`). The recipe below closes it. | `internal/drill/drepair.go:39`, `internal/drill/dtown.go:64` |
| B4 | Fire at range does not wear armor in game.exe (D1, I) but does in d2d: on d2d a repair trip will come earlier than in the real game against archers and casters (Fallen Shaman fire bolts, Skeleton and Dark archers). A bot soak on d2d that reads "repair trips per hour" overstates the cost for ranged-heavy areas until D1 is settled. | none (an interpretation note for the soak reading) |

## PR candidates

None. D1 is an inference from a call-graph (I), no differential ran the missile chain; D2 is proven only as a conditional on an input of unknown live value (U); D3 changes the seed stream, not an outcome; D4 to D6 are not deviations on the patched tables or not proven. A PR would need D1 run natively and a verifier (d2-reviewer).

## Drill recipe (lab binary; not run here)

Verbs as in `docs/forkdbg.md`, `docs/forkcheats.md`, `docs/d2d-fork.md`; the lab binary shows d2d's side only: game.exe's side stays V run.

- **R1 weapon rate (confirms the d2d side of Q4):** `debug equip` a Paladin with a hand axe and a shield, `debug durability all 9999`, `debug mon set <id> life 1000000` on a monster that cannot block, 300 plain Attacks (`cmd skill`), read the weapon's durability pair from `items`: expect about 12 losses of 1 (4 % of 300 landed swings; binomial sd 3.4); never more than the landed count.
- **R2 armor rate and slot odds:** `debug durability all 9999`, then `debug hit 1` 400 times (routed through `hurt_side`, `forkcheats.md`: the same roll and ring walk as a monster blow); expect about 40 points lost, split torso 28 %, shield 22 %, head 17 %, belt / feet / gloves 11 % each.
- **R3 missiles (settles D1 on d2d's side only):** stand in a Fallen Shaman or archer line with `debug durability all 9999`, take 100 arrow or bolt hits, expect d2d to lose about 10 points; game.exe is expected to lose none (D1, I): the game.exe half needs the live original game, not available here, or the emulator run described under D1.
- **R4 block:** a high-block shield (gear for block, or `debug stat`), 100 blocked blows: expect 0 points lost on every item.
- **R5 break and flags:** `debug durability <armor slot> 1`, then `debug hit 1` until it loses the point: expect durability 0 and the armor disabled (its defense no longer counts in `dump stats`); the same on the weapon: durability 0, still equipped, damage unchanged unless the stat dump says otherwise (U: what a 0-durability weapon does to damage was not traced).
- **R6 +0x70:** read the game dword at +0x70 of the live game structure (memory read only, always allowed): non-zero would make D2 moot for a normal game.
- **R7 live repair effect:** the existing D3.r drill after R1 or R2 (`docs/town.md` still says UNVERIFIED).

## Limits (not covered)

1. The item side is hooked: type membership, the gate, stat store and get, `FUN_00535bc0` (the weapon pick), `FUN_0063bde0`. Their rules are V read or S, not part of the V run counts.
2. The d2d side is `Fight::wear*` called directly with the case's seed; the callers (`fight.cpp:1005`, `:4774`) that decide when a hit counts are V read only, not run.
3. Skills other than the plain Attack (Zeal's repeated hits, Smite, Sacrifice, Vengeance) go through the 31 do-functions; which ones reach `FUN_0057d4f0` for every hit was not mapped (U).
4. Magic or better weapons running out of ammo (d2d breaks and keeps them; game side `FUN_00580310`) and the "next stack takes its slot" step were not run (quality 2 items only).
5. Mercenary and pet wearers: `FUN_0057d3d0` tests the player type only (V read), so a merc wears nothing and a hired merc's weapon never wears; d2d has no merc wear either (S: `fight.hpp:784`); not run.
6. What a 0-durability weapon does to damage and to-hit was not traced.
7. The statistical rate (10 % / 4 %) is the exact-seed equality with d2d, not a separate frequency measurement; the 10 % and 4 % come from reading the immediate compared against `rand % 100` (V read) and the seed-equal V run.

## Dead ends

- The brief's repair anchors (`FUN_0062e660`, `FUN_00629930`) give the repair gate, not the loss; `FUN_00629930` is reused as the loss gate (S price-trace.md).
- `FUN_00622c40`'s zero branch at 0x57d670 and the second half of `FUN_0057d4f0` (events 7 and 3) were not read; they do not touch durability.
- `pal_callgraph.py` over direct edges cannot prove a missing indirect path: D1 stays I.
- An early draft allowed a broken flag on weapons and on armor with durability left; those states do not occur in the game (only `FUN_0055f850` sets 0x100 and it zeroes the durability) and made d2d's early return on `broken()` look like state differences; the generator now sets the flag only at durability 0.

## Matrix-cell updates (proposed text)

`docs/research/paladin-act1-recon.md` section 6, row 7 (durability loss on hit and repair for melee), the cells to replace:

- "d2d's loss-on-hit rule: unknown" -> "traced (docs/research/paldur-trace.md): game.exe wears one point per landed unblocked melee hit, armor 10 %, weapon 4 % (V run); d2d the same, 0 differences in 34,939 armor and 15,253 weapon cases; a blocked, dodged or missed blow wears nothing (V run); d2d also wears on monster missiles (D1, I)".
- "the loss per hit and per block: not traced" -> "V run (paldur-trace.md Q1..Q5); per block: none".
- Risk cell: "if d2d never wears items the repair trip is never exercised" no longer applies: d2d wears at the game's rate for melee, more than game.exe against ranged attackers (D1, I).
- Section 6 table row 5 (card PALDUR): "decomp done; drill R1..R7 remains".

`docs/coverage/ledger.tsv` and `denominators.tsv` (shared, not edited here): the universe has no key for this rule. Proposed for the merger: add `F19 durability loss per landed hit` to the formulas keys (total 19) and a ledger line `formulas<TAB>F19<TAB>TESTED-DEVIATION<TAB>docs/research/paldur-trace.md<TAB>50192<TAB>seed 7 / 2026-11-02<TAB>D2 javelin gate +0x70 (V run); D1 missile wear I`. Not done here because adding a key edits `denominators.tsv`, which this card does not own.

## Shared-file lines for the merger

See `openspec/changes/paldur/merge-lines.txt` (BACKLOG, INDEX, CHANGELOG).

Glossary term missing (for the report, not edited): "landed hit" (a blow whose result record has bit 0 set: hit and not blocked, dodged or avoided).
