# SKSTATEEV second pass: state events, run under the emulator (card stateevrun)

Status: second pass of `sk-state-events-trace.md` (the first pass, V read only). Tags: **V run** game.exe's real code run natively under the emulator (`tools/emu/ser_events.py`, entry `run-on-worker.sh ser_run.sh N SEED`), with counted cases; **V read** read in the disassembly or in d2d's source; **D** table data; **S** earlier trace; **I** inference; **U** unverified. Nothing here ran a live game. No code or disassembly is pasted. The d2d side is read from source and, where a model is used, said so: **no number below compares the real d2d build** (see Deviations), so nothing is a PR claim.

Run record: `ser_run.sh 100000 1` on the worker host: seed 1 for the random cases, the boundary grids are fixed. Counts are in the tables.

## Answer in short

| First-pass claim | Verdict | Evidence |
|---|---|---|
| Event 11 (Energy Shield) fires before the resistance pass, owner = the defender | **holds, with a limit**: it fires before resistance, but at QUEUE time (swing gate FUN_0057dbf0 -> FUN_0057c1e0), not inside the damage call, and exactly once per hit | Q1, V run |
| FUN_0057c6c0 runs FUN_0057c1e0 (the resistance pass) | **holds only for callers that pass flag 1**; the melee swing passes flag 0 (the gate already ran it) | Q1, V run |
| Event 3 fires on a miss as well as on a hit, after the damage, only when in reach | **holds** | Q2, V run |
| Events 7 and 3 are "both owned by the defender argument" | **wrong for event 7**: event 7 is owned by the attacker, event 3 by the defender | Q2, V run |
| The reach test mode is 3 for a player attacker, 1 otherwise | **wrong, reversed**: 1 for a player, 3 for a monster | Q2, V run |
| FUN_0057b7d0's event 3 is skipped when record +4 carries 0x20 | **holds**; and bit 0x20 also skips events 5 and 1 in FUN_0057c6c0 (new) but NOT events 7 and 3 of the swing resolver | Q3, V run |
| D1: d2d's Shiver Armor is not modelled at all | **wrong**: `Fight::buff_events` models it on every melee swing of a striker, hit or miss | Q4, V read (d2d) |
| D2: d2d's ES order not read | now read: d2d absorbs AFTER resistances; game.exe before: a real deviation | Q1, D1 below |

## Harness (what ran)

`tools/emu/ser_events.py`, three parts, game.exe 1.14d (the emulator's own MPQ stack), hooks only for what is not the claim (stat getters and setters, unit-kind predicates, logging, stun / freeze side effects: the list of `life_apply.Game` as `sk_resist.py` uses it; for the swing, the list of `sb_block.py`).

- **A** the real FUN_0057c6c0 called with flag 1 on a record with one element field F, the target's resistance res; the event dispatcher FUN_005c0c30 is a hook that logs (event id, owner, other unit) and, at event 11, snapshots the field and subtracts an absorbed amount (a fixed amount or p percent: the part an Energy Shield node plays). Oracle: life taken.
- **B** FUN_0057dbf0 (gate, queues the swing) then FUN_0057d4f0 (resolver), natively, for every combination of attacker kind (player 0 / monster 1), defender kind, hit / miss, reach yes / no, with random extra record bits (0x4 0x8 0x20 0x40 0x1000 0x2c 0x4000). The dispatcher and the helpers FUN_0057c1e0 (resist), FUN_0057c6c0 (damage), FUN_0057d3d0, FUN_005d10c0, FUN_0057cee0, FUN_0057c9f0 are markers, reach FUN_00622c40 is a hook (logs its arguments). Oracle: the ordered list of markers and events.
- **C** the whole chain with the REAL FUN_0057c1e0 and FUN_0057c6c0: gate, resolver, damage on a plain monster target, fire and cold, event hook as in A. Oracle: life taken against two models (absorb before resistance: `resisted(F - a)`; after: `resisted(F) - a`, resisted = the MulDiv of `sk_resist.py`). Variants of the record flags: +0 = 2 (built record, the 57b7d0 add-on skipped), +0 = 0 (gate also runs FUN_0057b7d0), +4 with and without 0x20.

## Q1. Event 11 and the resistance pass

**Counts (V run):**

| Part | Cases | Result |
|---|---|---|
| A, real FUN_0057c6c0 flag 1, 6 elements, res -100..120, F 256..0x200000, fixed and percent absorb (seed 1 for 100,000 random) | 104,158 | event 11 fires exactly once per call, owner = defender, other = attacker (104,158); the field at the event equals F: nothing resisted before it (104,158); life taken equals the BEFORE model (104,158, 0 exceptions); it equals the AFTER model in 22,542 only, none of them where the two models differ. 81,616 cases discriminate: 81,616 equal the before model, 0 the after model, 0 neither |
| C, gate + resolver + real damage, +0 = 2, fire and cold (seed 1) | 101,386 | event 11 once per hit (101,386); life taken equals the BEFORE model in 101,386, the AFTER model in 21,867 (79,519 discriminating: 79,519 before, 0 after, 0 neither) |

So **Energy Shield absorbs the raw (percent-scaled) damage and the resistance works on what it let through** (V run, 0 exceptions). First-pass Q4 holds. Two limits the first pass did not have:

1. **When it fires (V run, B and C).** On a melee hit the sequence is `11 (defender), 5 (attacker), 1 (defender), 7 (attacker), 3 (defender)` in every one of the 101,386 C cases of the plain record: event 11 comes from the gate FUN_0057dbf0 (the call of FUN_0057c1e0 at 0x57dc43 with the record, V read of the gate; V run), at the moment the swing is QUEUED. The later damage call FUN_0057c6c0 gets flag 0 (B shapes: `DMG(flag=0)`), so it does not run the resistance pass again. The first-pass text "FUN_0057c6c0 calls FUN_0057c1e0" is true only when its flag argument is non-zero (Part A uses flag 1; callers with flag 1 are not traced here, U).
2. **A queued swing that never lands has already paid.** B: hit record, reach test fails at resolution: shape `RES, R, DROP`: the resistance pass (and so event 11) ran at queue time, then the resolver dropped the swing with no damage, no event 7 / 3. What the ES node does inside event 11 (mana paid, FUN_005ca6c0) did not run in this harness (hooked), so "mana is lost on a dropped swing" is **I**, from the order. A miss never queues through FUN_0057c1e0: the gate requires bit 0 of record +4 (V read, 0x57dc24), shape `R, E7, E3, REACT, DROP` has no RES. Block / dodge bits 0x8380 skip it too (V read, `test 0x8380`; the same gate ran in `sb_block.py`).

## Q2. Event 7 and event 3 on a miss, owners, reach mode (V run, B)

B ran 16 cells (attacker kind x defender kind x hit x reach) with 1,000 random-flag cases each (seed 7), 16,000 cases in all. Every cell has exactly one shape (the 1,000 cases of a cell are identical); the expectation "events 7 and 3 iff reach, damage iff hit and reach" held in 16,000 of 16,000 cases (the harness's `match` column, rerun after fixing a stale clause of the metric; the shapes did not change).

| Check | Result |
|---|---|
| reach OK, hit or miss: events 7 then 3 fire, then `REACT`, `DROP` | all 8,000 reach-ok cases, 4,000 of them misses: event 3 fired on every miss (4,000 of 4,000) |
| after the damage | on a hit the order is `RES, R, DMG, KILLCRED, E7, E3, ONHIT, REACT, DROP`: damage before event 7 / 3 in 4,000 of 4,000 hit cases with reach |
| reach fails: no event 7 / 3, no damage | 8,000 of 8,000 |
| owner / other | event 7: owner = ATTACKER, other = defender; event 3: owner = DEFENDER, other = attacker, in 8,000 of 8,000 reach-ok cases; event 7 owned by the defender: 0 cases |
| reach test mode (third stack argument of FUN_00622c40; its first two are attacker, defender) | 1 when the attacker unit is a player (unit type 0), 3 for a monster (type 1): 8,000 cases each |
| extra record bits 0x4 0x8 0x20 0x40 0x1000 0x2c 0x4000 | the shape is the same for every extra-bit value: the resolver's events 7 and 3 do not read them |

First-pass Q3 holds on "miss fires event 3, after the damage, only in reach"; it is wrong on the owner of event 7 (it said both defender) and on the 1 / 3 mode (reversed). The mode was hooked, not computed here: the value is what FUN_0057d4f0 pushes (V run of that code). Limit: only unit type 0 / 1 were tried, no hireling or pet.

## Q3. The 0x20 bit and the FUN_0057b7d0 event 3 (V run, C variants)

| Record | Cases | Event sequence (all cases the same) |
|---|---|---|
| +0 = 2, +4 = 1 | 101,386 | 11, 5, 1, 7, 3 |
| +0 = 0, +4 = 1 | 21,386 | **3**, 11, 5, 1, 7, 3 (the gate's FUN_0057b7d0 fires an extra event 3 first) |
| +0 = 0, +4 = 1 with 0x20 | 21,386 | 11, 7, 3: no extra event 3, and no events 5 and 1 |
| +0 = 2, +4 = 1 with 0x20 | 21,386 | 11, 7, 3 |

The 0x20 bit suppresses the extra event 3 of FUN_0057b7d0 (first-pass claim holds), and it also suppresses events 5 and 1 inside FUN_0057c6c0 (new, V run); the damage is still applied (life taken equals the BEFORE model in all 21,386 cases of the +0 = 2 record with 0x20; 16,564 of them discriminating). The swing resolver's events 7 and 3 fire anyway (the flag is not read there), so "a Shiver strike does not re-trigger Shiver" holds for the add-on site only; whether Shiver's own damage ever goes through the swing resolver is U (Shiver's nova is a missile, I). With +0 = 0 the damage is higher than the model (FUN_0057b7d0 adds damage before the resistance pass), so the models are not the oracle there (C: 0 equal before, by design, not a finding).

## Q4. d2d side (V read of source, no run)

| Rule | game.exe | d2d (fight.cpp, ai.cpp, combat.hpp) |
|---|---|---|
| ES order | before resistance, at queue time (Q1) | `Fight::absorb` (fight.cpp 1344..1372) works on `foe.damage`, which is the sum of the already resisted blows (`monster_blow` resists inside: combat.hpp 354); so AFTER resistances, once per frame total, whole points |
| Shiver on a miss | event 3 fires on a miss, in reach (Q2) | `foe.melee_by.push_back` runs for every swing in reach (ai.cpp 2554, hit or not); `buff_events` (fight.cpp 1378..1393) lands the Shiver blow for each `melee_by`: so a miss triggers it, like game.exe |
| Frozen Armor striker | event 2 on damage | `damaged_by` only when `taken.damage > 0` (ai.cpp 2555) |

## Deviations (d2d vs game.exe)

| ID | Rule on both sides | Tag | Differential |
|---|---|---|---|
| D1 | ES order: game.exe absorbs before the 12-row resistance pass (V run, 104,158 + 101,386 cases, 0 exceptions); d2d absorbs after (V read of `Fight::absorb` and `monster_blow`). The two orders differ whenever the absorb is not proportional (a mana-limited ES, rounding, resistance steps): in the run, 81,616 of 104,158 cases (A) and 79,519 of 101,386 (C) | game: V run; d2d: V read | **could be built**: a d2d test mode that applies one blow to a player with ES and prints life / mana, vs this harness with the real ES node (FUN_005ca6c0 un-hooked, mana state in the stat hook); not done here, so no `TESTED-DEVIATION`, no PR candidate |
| D2 | ES paid at queue time: a swing that is dropped at resolution (reach lost) has already run event 11; d2d absorbs only on a landed blow | game: V run (order); consequence I | needs FUN_005ca6c0 un-hooked in B; d2d side U (does d2d ever drop a queued swing?) |
| D3 | Shiver on a miss: both sides fire (no deviation) | V run (game), V read (d2d) | first-pass D1 retracted |

## Bot findings

- **B1.** The bot models neither Shiver Armor nor Energy Shield in combat (first-pass B1 stands: internal/combat/necro.go:40 is the only armor-adjacent data, unchanged and not re-read here, U). With Q1 / Q2: the Shiver benefit also arrives on swings the monster misses, and ES mana is spent at queue time, even for a swing that fails to land.
- No bot code was changed. A bot ES estimate that applies resistances before the absorb would be wrong in the same direction as d2d (D1).

## PR candidates

**No PR candidate.** D1 is proven on the game side only (V run); the d2d side is a source read and a model, not the real build's `Fight::absorb`, so "mismatches before > 0 on the real build" is not shown. The fix shape for whoever builds it: take the absorb out of the per-frame total and apply it to each blow's raw element fields before the resistance (`monster_blow`), with the ES per-field min of Q4 of `sk-armor-trace.md`; then differential against `ser_events.py` with FUN_005ca6c0 un-hooked.

## Dead ends

- Running FUN_0057b7d0 and FUN_005ca6c0 for their own arithmetic: not tried (the add-on damage and the ES node's mana math need more stat hooks; marked U above).
- The first-pass models of the "absorb" are stand-ins (a fixed amount or p percent), not the ES formula: the order claim does not depend on the formula, the ES formula (min of percent and mana limit) is still V read only in `sk-armor-trace.md`.
- B's miss path with block / dodge bits (0x8380) was not driven: the gate's `test 0x8380` is V read (and shown by `sb_block.py`), not re-run here.
- Pets, hirelings and player-vs-player targets (FUN_0057c060's percent scale, FUN_0057bf80 monster-only fields) are hooked off: plain monster target, attacker a player or monster type only.

## Drill recipe (lab, U to confirm; verbs of docs/forkdbg.md, nothing run here)

- **E1 order.** `debug points 58 1`, `debug cast 58 <own x> <own y>`, `debug stat 8 <mana 256ths: 256 x 200>` so mana is never the limit, then take the same 100-point fire hit with `debug hit` or a monster after `debug mon set <id> dmg 100 100`, once with the wearer at fire resistance 0 and once at 50 (`debug stat` on the resistance). Game.exe (before resistance): absorbed = percent x 100 in both, resistance applies to the rest, so mana drain is equal at both resistances; d2d: the drain scales with the resisted damage.
- **E2 queue-time payment.** Same setup, a monster in reach whose target steps away before the swing resolves (or `debug cast` Teleport after it starts its swing): mana drops with no life loss.
- **S1 Shiver on a miss.** `debug points 50 1`, `debug cast 50 ...`, `debug mon set <id> dmg 0 0` is not a miss; use a low-to-hit monster or a monster at a high defence (`debug stat` defence) and read the striker's life after misses (`debug mon` / `monsters`).

## Matrix-cell updates (docs/research/sorc-skill-matrix.md, proposed)

- 3.11 Energy Shield (a): "event 11 fires once per hit at queue time (gate FUN_0057dbf0 -> FUN_0057c1e0), before the resistance pass; a queued swing dropped at resolution has already run it [V run, stateevrun-trace Q1]. d2d absorbs after resistances (D1) [V read]".
- Shiver Armor (rules): "event 3 fires on every in-reach swing, hit or miss, after the damage (V run, stateevrun-trace Q2); d2d models that (Fight::buff_events, V read)". Retract the first-pass note that d2d does not model Shiver.
- sk-melee-trace Q1 reach row: "FUN_00622c40 mode is 1 for a player attacker, 3 for a monster [V run, stateevrun-trace Q2]" (the first-pass text had them reversed).

## Not covered (limits)

Events 12, 13, 0 (levelup, missile hit) and the kill events 9 / 10 are not run. The ES node and Bone / Cyclone Armor order inside event 11 are not run (the dispatcher is a hook). The percent-scale function FUN_0057c060 is the plain-monster value (100).

## Shared-file lines for the merger

See `openspec/changes/stateevrun/merge-lines.txt`.

## Reproduce

`tools/emu/run-on-worker.sh --ttl 1500 ser_run.sh 100000 1 > /tmp/d2agent-stateevrun/out.txt 2>&1`; disassembly: `run-on-worker.sh sk_gearar_dis.py 57dbf0 57c1e0 57c6c0 57d4f0`.
