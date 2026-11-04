# FIXCASTGAP: a cast gap per skill

> status: current. Card openspec/changes/fixcastgap. Source: finding B1 of docs/research/sk-lightning-trace.md (branch agent/sklight, read only), docs/research/sk-mana-trace.md, docs/research/cast-trace-gameexe.md, docs/d2d-casts-real.md, docs/primitives.md P26. Offline only; nothing run live.

Tags: V = verified (how is said), D = read from the patched Skills.txt through the traces, I = inference, U = unverified. Every number says which backend it is for: **game.exe** (1.14d), **d2d** (the lockstep fork).

## 1. Where the gap is used and what an early command does

- One knob `combat.castGapMs` (650) is read by `Fighter.castReady` (internal/combat/fight.go) for the fight loop's casts and by waits after buffs, summons and corpse casts in druid.go, necro*.go, assassin.go, fighter.go (those are other classes; unchanged). `castReady` compares the time since the last cast command with the gap; the fight loop polls every 60 ms.
- **d2d, a command during a cast is dropped, not queued, and costs nothing (V-d2d).** docs/d2d-casts-real.md: at gap 450 the fighter issued 1.6 to 1.75 commands per real cast; real casts per kill (mana spent / cost) did not depend on the gap (1.16 vs 1.19, 0.73 vs 0.75, 0.61 vs 0.62), so the extra commands vanished without mana and without a second cast. Mana is paid on the first tick after a command that starts a cast (+1) and the missile is made at tick +8; a hit that cuts the cast at +1 to +7 leaves no missile and the mana stays spent (docs/primitives.md P26, drill DRILLS13 M1, V-d2d; d2d pays at the command, game.exe at the action frame: sk-mana-trace D1, V read).
- **game.exe, a command during a cast: U.** No trace or live count exists for what the cast key does while the mode is SC/SQ. The knob's comment and sk-lightning-trace B1 assume the same drop (S). The recipe in section 5 would count it.
- Cost of an early command is therefore a lost poll (no mana, no second cast) on d2d; a too LONG gap costs real casts. Both errors are small; the point of the fix is the 760 ms skills.

## 2. Real cast lengths (ticks of 40 ms, FCR 0)

Model, one rule for both kinds (D + V): speed = floor(256 * rate / 100), rate = min(100 + 120 FCR / (120 + FCR), 175). A plain SC cast lasts `ceil(14 * 256 / speed) - 1` ticks, the Lightning sequence `ceil(19 * 256 / speed)`. Checked against the oracle lines of docs/research/cast-trace-gameexe.md (V, cast_rates.py on game.exe code): Sorceress SC 13 / 12 / 11 / 10 / 9 / 8 / 7 at FCR 0 / 9 / 20 / 37 / 63 / 105 / 200, Lightning 19 / 18 / 11 at FCR 0 / 7 / 194; TestCastTicksMatchOracle pins all ten.

| Skill | anim / seqtrans (D) | game.exe ticks, ms | d2d today | tag |
|---|---|---|---|---|
| Fire Bolt, Charged Bolt, Ice Bolt, Frost Nova, Frozen Armor, Teleport | SC / SC | 13 = 520 ms | 14 = 560 ms measured (Fire Bolt and Charged Bolt, V-d2d; fidelity row 7a) | game.exe V (oracle) for the SC length; the other four skills share the row's anim (D), not timed one by one (I); d2d one tick longer: V-d2d for the two bolts, I for the rest |
| Lightning, Chain Lightning | SQ / SC | 19 = 760 ms | plain SC, 14 = 560 ms until upstream/lab `fix/lightning-sequence` is in the lab binary (sklight D3: S) | game.exe V (sequence length); d2d S |

The 14 vs 13 on d2d is a command-to-start latency of one tick (I: not separated from the cast length by any probe). The gap adds that tick to every length.

## 3. The fix

- `internal/combat/castgap.go`: table `castTicks(name, fcr)` (the model above for every plan skill plus the other Sorceress SC spells; 0 = unknown) and `castGapFor`. After a cast the next command waits `max(CastGapMs, (ticks + 1) * 40 + CastMarginMs)`. The gap is that of the LAST cast (`Fighter.lastCastName`, set in `cast`), because it is the previous cast that has to finish.
- Knobs (docs/knobs.md regenerated): `combat.castPerSkill` 1 (0 = the old flat gap for every skill, the explicit override), `combat.castMarginMs` 90 (the old 650 minus 560, so the SC skills keep exactly 650), `combat.castFCR` 0 (assumed FCR; 0 is safe for any gear, more FCR only shortens a cast).
- **Backend-aware (coordinator decision: d2d is the primary backend).** `Fighter.D2D` is set by the d2d constructors (internal/d2dcombat/rig.go, cmd/d2bot/behaviors.go from `--backend d2d`); `castTicks(name, fcr, d2d)` then treats Lightning and Chain Lightning as plain SC (14 ticks with the dispatch tick, V-d2d for the SC length, S for "d2d casts them plain": sklight D3). Game.exe keeps the 19-tick sequence (V, oracle). When `fix/lightning-sequence` lands in the lab binary, the d2d row changes in ONE place: drop the `&& !d2d` in `castTicks` (castgap.go).
- Result at FCR 0: Fire Bolt, Charged Bolt, Ice Bolt, Frost Nova, Frozen Armor, Teleport 650 ms on both backends (unchanged). Lightning and Chain Lightning: **d2d 650 ms**, **game.exe 890 ms** (19 + 1 ticks + 90). `CastGapMs` stays the floor and the fallback for unknown skills; raising it raises every gap. To LOWER the gap for a fast caster use `castPerSkill 0` with a lower `castGapMs`, or set `castFCR`.
- Skill choice, ranges and every other wait are unchanged.

## 4. Tests

`go test -p 1 -count=1 ./internal/combat ./internal/env ./tools/knobdump`: TestCastTicksMatchOracle; TestCastGapCoversEveryPlanSkill (eight skills x seven FCR values: gap >= cast + 1 tick; Chain Lightning 890, Fire Bolt 650); TestCastGapFallbackAndOverride (unknown skill, flag off, raised CastGapMs, FCR 200); TestLightningGapPerBackend (890 game.exe, 650 d2d, via the Fighter too); TestFightLoopLightningNeverReissuedEarly (game.exe table, a level 30 Lightning fight on the fake game, the cast taking no clock time: with the old flat gap commands arrive before 760 ms, with the table none do). The sim helpers `oldCastGap` and the castGapMs sweeps now also set `CastPerSkill` 0, as they test the flat knob.

## 5. Live proof recipe (not run; DRILLS15 holds the slot)

Needs the lab binary, the fork verbs of docs/forkdbg.md / forkcheats.md, hidden slot, throwaway Sorceress.
1. `debug stat 12 30`, `debug points 49 5`, `debug points 53 5`, `debug points 36 1`, `debug mana full`; bind Lightning and Fire Bolt; FCR 0 gear.
2. `debug spawn` a Fallen with `debug mon set <id> life 3000 maxlife 3000 res light 0 res fire 0` (a punching bag that survives 30 casts) 12 subtiles away.
3. Run the Fighter through the d2dcasts2-style probe (cmd/d2dcasts2, `-castgap`) for 60 s per arm, once with `combat.castPerSkill 0 castGapMs 650`, once with the default, for Lightning (then Chain Lightning, then Fire Bolt). Per arm count: commands (bot log `combat decision act=cast`), real casts (mana spent / skill cost from `debug stat 8`, or `debug missiles` births for Lightning), dropped commands = commands - real casts, double-paid = mana spent - real casts x cost.
4. Pass: on d2d (plain SC today) both arms 650 ms, commands per real cast 1.00 to 1.04 and equal throughput; on a d2d build with the sequence fix, run the new arm with the `&& !d2d` dropped: 1.00 to 1.04 against the flat arm above 1.05; on game.exe (hidden instance, mode starts) the same with the 890 ms gap; Fire Bolt arms equal; double-paid 0 in every arm.
5. Verdict goes to the card's `live:` field.

## 6. Shared-file lines (for the merger; this card does not own them)

- docs/BACKLOG.md: fixcastgap (card [fixcastgap](archive/openspec-index.md#fixcastgap), offline): per-skill cast gap, `combat.castPerSkill` / `castMarginMs` / `castFCR`, Lightning and Chain Lightning 890 ms on game.exe (760 ms sequence) and 650 ms on d2d (plain SC), the other plan skills 650 on both; tests pin the oracle lengths; live proof recipe in docs/fixcastgap.md, open (needs the live slot and the lab binary with `fix/lightning-sequence`).
- docs/INDEX.md: `fixcastgap.md` - per-skill cast gap, cast lengths per skill and backend, live proof recipe.
- CHANGELOG: "The bot waits one cast length per skill between casts: Lightning and Chain Lightning 890 ms on Game.exe (650 ms on d2d) instead of the flat 650 ms (new knobs combat.castPerSkill, castMarginMs, castFCR)."
- docs/d2d-combat.md and docs/fixmissile.md still describe the flat knob (not owned here).
