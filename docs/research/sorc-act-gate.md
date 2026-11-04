# Sorceress per-act skill gate: verdicts, gate rule, what is still needed

> status: draft for the user, 2026-11-01, card actgate (docs only; no game, no d2d slot, no code, no spec change). Answers the user's question "should we make a focused effort to verify all sorceress spells and talents before we start pushing the bot act progression further" with the evidence of the SKILLINV matrix (`docs/research/sorc-skill-matrix.md`, its section 3 to 6), the traces `sk-missile-trace.md`, `sk-mana-trace.md`, `sk-teleport-trace.md`, `sk-armor-trace.md`, `sk-lightning-trace.md`, `sk-nova-trace.md` (read from branch agent/sknova until its merge) and the live d2d drills DRILLS11 to 15 (`docs/probes/drills/README.md`; DRILLS16 was still running and is not used).

Tags: **V read / V run** = a disassembly or an emulator run of game.exe by one of the traces (the trace says which); **live** = a drill on d2d (the lab fork), never on game.exe unless a row says so; **S** = source or document read; **I** = inference or model; **U** = unverified. A drill on d2d that "PASSES against the trace" compares d2d with a game.exe READ, not with a game.exe run: those rows say "compared to trace". Game.exe itself was never run for any skill row in this document, except the older Game.exe Act 2 chain runs of the Teleport row (`docs/act2.md`).

## Answer (plain words)

Yes, a focused check is worth it, and most of it is already done on paper. We have read how the real game works for the missile spells, mana, Teleport, the armors, Lightning and Frost Nova, and we have run the first live checks on our test game (d2d, not the real game). The result: the basics are right (how long a cast takes, how much mana a cast costs and how fast it comes back, how many Charged Bolts fly, how far bolts reach, how long Frozen Armor lasts and that the bot recasts it in time). What is NOT right is a short list of places where our test game is more generous than the real one. Two matter for the plan: Frost Nova and Chain Lightning hit the same monster several times in the test game but once in the real game (so soak numbers that depend on them are too good), and Lightning spells cast about a third faster in the test game than in the real game unless our patch is in the lab binary. Frozen Armor also freezes every monster that hits you in the test game, even bosses. Counting the rows below: Act 1 is close (nothing but Frost Nova is red, the staff swing is unknown), Act 2 has no red row, Act 3 and later are red because of the lightning skills. Nothing here proves the bot is bad at any act; it says which soak numbers we may trust. The cheapest next steps are two short live proofs of fixes already merged, then the d2d fix for the multi-hit problem, which clears the biggest blocker.

Verdict counts (rows of section 1; details there): Act 1 band: GREEN 2, YELLOW 6, RED 1, UNKNOWN 1. Act 2 band: YELLOW 4. Act 3 band: YELLOW 2, RED 1, UNKNOWN 1. Act 4 / Act 5 / NM band: RED 1. Top three blockers: (1) NEXTHIT, d2d has no "just hit" unit state, so Frost Nova (A1) and Chain Lightning (A3) over-hit (6.2 hits per monster per nova cast against 1.0, modelled); (2) the Lightning family cast length and the cast gap (d2d 13 ticks against game.exe 19, held candidate fix/lightning-sequence, card FIXCASTGAP unmerged); (3) the staff swing and resist / immunity rules are not traced at all.

## 0. Verdict rule used here

- **GREEN**: the game.exe rule is V (read or run), d2d matches or its deviation is fixed in the binary the soaks run, AND a live d2d drill or a live use with a measurable oracle passed, AND there is no open bot gap that changes the result.
- **YELLOW**: the skill works and the rule is V, but there is a known d2d deviation or bot gap with a bounded effect (stated per row), or only the live drill is missing (marked LIVE-GAP). A YELLOW row is accepted by the user per effect, not silently.
- **RED**: the d2d result for this skill is not a valid stand-in for game.exe in a way that moves soak numbers or progression decisions, or the bot's use would fail on the real game; progression past the act is blocked until the row is fixed or the user waives it with a named workaround.
- **UNKNOWN**: no trace, so nothing to compare with.
- The binary question: the drills ran on the lab branch `lab-debug-skills`, which contains upstream PR #128 (cast ticks; V by git, `sk-mana-trace.md`). Which binary the soak harness runs is U; the gate therefore requires the soak report to name the d2d branch and commit (section 2).
- N/A rows (skill not bought by the default build) are listed but not counted.

## 1. Verdict tables per act band

Columns: trace (file); d2d deviations (ids from the trace named in the first cell of that row); live drill (which, result, date; "compared to trace" when d2d was checked against a game.exe read); bot fix (card or open); verdict. All drill dates are 2026-11-01 unless said. Ids: SKMISS D1-D5 / B1-B4, SKMANA D1-D7 / B1-B6, SKTELE D1-D7 / B1-B7, SKARMOR D1-D7 / B1-B6, SKLIGHT D1-D5 / B1-B5, SKNOVA D1-D6 / B1-B6.

### Common rows (apply to every band; counted in Act 1)

| Row | Trace | d2d deviation | Live drill | Bot fix | Verdict |
|---|---|---|---|---|---|
| C1 cast ticks (SC 13 at FCR 0) | `cast-trace-gameexe.md` (V oracle, 2,807 + 1,203 lines, 0 differ) | none on binaries with PR #128 (V by git: lab-debug-skills); old branch `lab` has 14 ticks | DRILLS13 M1: action frame at +8 ticks, compared to trace (consistent with 13); DRILLS15 K1 same | none needed | **GREEN** (condition: soak binary named in the report) |
| C2 mana (cost, pay timing, regen) | `sk-mana-trace.md` (V read + V run: cost formula equal, 130,049 of 130,840 cases; regen V read) | SKMANA D1 pays at the command, 7 ticks before the action frame; D3 a cut cast stays paid; D2 floor (only Teleport 25+, Might) | DRILLS13 M1 (D1, D3 confirmed on d2d), DRILLS15 M2 (a real 20-damage blow cuts 7 of 48 casts, mana paid, no missile), M4 regen PASS (compared to trace, exact 3 per frame); M3 floor FAIL as expected | OPEN: B1 `ManaBurnTracker` books at the command, B2 regen model 1.2 to 1.8 times too fast (13.5 points per 30 s against 8.8 at max mana 45), B3 potions instant; no card yet (proposed FIXMANAMODEL, section 3) | **YELLOW** (DEV + BOT-GAP). Effect: on d2d a get-hit cut costs a few mana points that game.exe keeps; the bot rests and sims with optimistic regen, so rest times come out short early on (accepted bound: a few points per fight, early levels only) |
| C3 damage on the monster | `skill-damage-trace.md` (V, 116,000 cases) and `sk-missile-trace.md` D1 (V run on the real build: 31,459 of 104,000 differ, 0 after the local candidate) | SKMISS D1 the roll includes the top 256th (1/512 point mean per hit), K4 whole-point damage (about -17 percent at skill level 1, shrinking per level; PR candidate built, not pushed; whether the lab binary carries it: U) | none dedicated (`M4 roll` of drills15 not run) | none needed | **YELLOW** (DEV). Effect: d2d under-deals, so kills per cast and XP per hour on d2d are a conservative floor, not an overestimate |
| C4 bind and select | `skill-bind-trace.md` (V; 81,756 of 127,032 differ before the rule, 0 after) | d2d stores a select of any id; game.exe only of a learned skill | DRILLS10 P26 observables (record; Fire Bolt D5.a PASS) | the bot selects only skills it has points in (S) | **YELLOW** (DEV). Effect: none for the bot as it runs; matters for a test that selects an unlearned skill |

### Act 1 band (Normal, clvl 1 to 16, through Andariel)

| Skill | Trace | d2d deviation | Live drill | Bot fix | Verdict |
|---|---|---|---|---|---|
| Fire Bolt (36) | missile + mana traces (V read, V run) | SKMISS D1, D2 (mana timing), K4 truncation; flight rules proven upstream (S, `missiles.py`) | D5.a PASS (mana 35 to 32, zombie hp 11 to 8, compared to table); DRILLS15 K1 reach PASS 46.5 against 46.9 subtiles (compared to trace) | OPEN SKMISS B1 ranges 20 against reach 46.9 (conservative), B3 booking, B4 no flight-time model | **YELLOW** (DEV, small). Effect: bounded by C2 and C3; no effect on kills per cast beyond the whole-point truncation |
| Charged Bolt (38), the main attack to clvl 25 | missile trace (spawn, count, wiggle V read; flight S) | SKMISS D1, D2; hit outcome per cast on a pack never compared; whether Charged Bolt rows are NextHit: not stated in the traces (U) | DRILLS15 K2 PASS: 3, 7, 22 bolts at levels 1, 5, 20, all alive at once, 77 frames (compared to trace); `drills10` fight counters equal the independent meter (fixmeter) but no per-cast oracle | none (SKMISS B1: range 10 correct) | **YELLOW** (DEV + LIVE-GAP). Effect: casts per kill on game.exe vs d2d is not measured; the damage roll drill (levels 4, 8, 13) was not run; d2d numbers are a floor via C3 |
| Frozen Armor (40) level 1 | `sk-armor-trace.md` (V run, 184,771 cases for the strike effect) | SKARMOR D1: d2d stuns AND chills every damaging striker for raw calc1 frames, bosses and cold immunes included; game.exe freezes only plain monsters at coldeffect below 0 and chills specials | DRILLS13 D-fro1 PASS: state on 8 ticks after the cast, real duration 120.0 s, recast at 114.0 s, armor absent 0 of 819 samples; recast refreshes the full duration (live); the strike effect not drilled | fixrecast landed (live proof done); fixbuff landed 2026-11-01 (fight-loop recast, states), live proof OPEN | **YELLOW** (DEV). Effect: on d2d every melee striker, Andariel included, is stunned for 30 frames; defence in melee fights is OVERRATED, deaths and clear speed in a d2d soak are optimistic; game.exe fights are not affected. Needs the user's acceptance for the Andariel attempt |
| Frost Nova (44) | `sk-nova-trace.md` (V read + V run: 64 ring missiles, reach 11 to 12 subtiles) | **SKNOVA D1 (V read both sides, magnitude I): d2d hits a monster with many ring missiles per cast (6.2 against 1.0; 3 to 35 times the damage at 1 to 4 subtiles)**, D2 ring angles, D3 irrelevant here | none (not drilled) | OPEN SKNOVA B1 (crowd radius 4 against 11), B3 (anything measured on d2d is inflated), B5 | **RED**. Why: the bot casts it at crowds of 3 within 4 subtiles from clvl 6, exactly where the inflation is largest; a soak through Act 1 would credit damage and chill the real game does not give, and tuning from it is wrong. Unblock: NEXTHIT fixed on d2d (then one ring drill), or the user waives it with the workaround "novas off in the gate soak" (`novaCrowd` raised, report says novas 0) |
| Telekinesis (43), prerequisite point only | matrix 3.6 (S) | none relied on | none | none | **GREEN** (never cast; the point spend is legal by the leveler rule; nothing else relied on) |
| Plain staff swing (0), mana-out fallback at clvl 1 to 3 | none (SKMELEE, decomp row 20, not traced) | not compared | none on d2d; Game.exe early soaks count melee hits (no oracle) | none | **UNKNOWN**. Bound: clvl 1 to 3 and the last rung of the low-mana ladder only |
| Static Field (42) | `static-field-chill-trace.md` (V) | wrong in three ways on d2d (PR #155 candidate) | none | fixstatic: dropped from the FRESH table (not learned), code inert unless learned | N/A on the default build (legacy tables only; YELLOW there) |

Act 1 counts including the common rows: GREEN 2 (C1, Telekinesis), YELLOW 6 (C2, C3, C4, Fire Bolt, Charged Bolt, Frozen Armor), RED 1 (Frost Nova), UNKNOWN 1 (staff swing).

### Act 2 band (Normal, clvl 16 to 26, through Duriel)

New skills or levels; the Act 1 rows carry over.

| Skill | Trace | d2d deviation | Live drill | Bot fix | Verdict |
|---|---|---|---|---|---|
| Charged Bolt levels 13 to 20 (bolts 15 to 22, brackets above 16) | missile trace + damage trace V (bracket tables, masteries) | as Act 1 | DRILLS15 K2: level 20 gives 22 bolts (compared to trace) | none | **YELLOW** (DEV + LIVE-GAP: damage per bolt at 13 to 20 not drilled) |
| Teleport (54), nav hops | `sk-teleport-trace.md` (V read + V run, 100,000 cases against a line copy of d2d's rule) | SKTELE D1 no room limit (25,120 of 100,000 differ), D2 blocked cell refused (1,323), D3 lands on a monster (4,805), D4 refused hop still costs the mana, D7 arrival bookkeeping; D5, D6 not drilled | DRILLS15: T1 town refusal PASS; T2 FAIL, T3 FAIL, T4 FAIL, T7 not reachable, M3 floor FAIL (all d2d against the trace, i.e. deviations CONFIRMED live); hops of 18 subtiles and shorter all land on d2d; game.exe: act2live chain hops, town refusal, contact crossing (V live, `act2.md`); T6 Duriel's lair not run | fixlair merged (Levels.txt value 2 means allowed along a clear line); OPEN B2 a refused hop costs 24 mana on d2d | **YELLOW** (DEV). Effect: d2d is MORE permissive than the real game (long hops land, blocked cells snap there); the bot's hops are at most 18 subtiles and aimed at walkable cells, which both backends accept; a refused hop wastes 23.8 mana on d2d. d2d cannot prove the long-hop speed of a real Act 2 run |
| Frozen Armor level 2 (clvl 22) and recast | as Frozen Armor level 1 | as Act 1 | DRILLS13 D-fro2 PASS: state `40:2`, 132.32 s, casts at 126.0 s, absent 0 of 775 | fixrecast, fixbuff (live proof open) | **YELLOW** (DEV, same effect as Act 1) |
| Merc / pet moved with the hop | `sk-teleport-trace.md` D7 (V read outline) | d2d does not move the hireling with the owner's hop | DRILLS15 T5 FAIL (merc 4.5 to 5 cells behind after a 6 cell hop; compared to the read, UNVERIFIED by play on game.exe) | none; no card (candidate listed in BACKLOG) | **YELLOW** (DEV). Effect: on d2d the Act 2 merc lags behind every hop, so merc contribution and merc deaths in an Act 2 d2d soak are skewed against the merc |

Act 2 counts: GREEN 0, YELLOW 4, RED 0, UNKNOWN 0 (Act 1 rows still apply).

### Act 3 band (Normal, clvl 26 to 32)

| Skill | Trace | d2d deviation | Live drill | Bot fix | Verdict |
|---|---|---|---|---|---|
| Lightning Mastery (63), passive; first useful point clvl 30 | `skill-damage-trace.md` mastery mapping (V, masteries 0 to 600) | none known | none (SKPASS, row 23, not traced beyond the damage mapping) | none | **YELLOW** (LIVE-GAP). Effect: none known; formula is V |
| Chain Lightning (53), first in the order from clvl 26 | `sk-lightning-trace.md` (V read both sides; D1 and D2 modelled) | SKLIGHT D1 NextHit: d2d re-hits the struck monster on the hop's first frame in about 73 percent of hops, game.exe never; D2 one unit per frame; D3 cast length 13 ticks on d2d against 19 steps (-32 percent cast time; held candidate fix/lightning-sequence; lab binary: U); D4 hop order; to-hit equal | none (not drilled; a ticks drill and a hop line drill are not run) | OPEN: FIXCASTGAP (branch agent/fixcastgap, unmerged); SKLIGHT B2 immune target drops Chain Lightning (policy, paused), B3 range 25 against 35.2 | **RED**. Why: d2d overrates the A3 main attack twice (cast speed up to 46 percent more casts per second, and hop re-hits), so Act 3 soak XP per hour on d2d is not a fidelity proof; on game.exe the bot's 650 ms gap is shorter than the 760 ms cast. Turns YELLOW when NEXTHIT and the 19 tick candidate are in the binary and a ticks drill reads 19 |
| Cast gap for 760 ms casts (bot) | `sk-lightning-trace.md` B1, `docs/fixcastgap.md` on its branch | coupled to D3 | not run | FIXCASTGAP unmerged (backend aware: d2d plain SC 650, game.exe 890) | **YELLOW** (BOT-GAP). Effect today: none on d2d (13 tick cast fits 650 ms). Coupling: the day fix/lightning-sequence lands in the binary d2d casts 19 steps and the old gap would drop commands, so these two must ship together |
| Resist and immunity by difficulty, pierce, what an immune hit does (SKRESIST, row 21) | none beyond the K4 missile-path proof (V) | unknown | none | none | **UNKNOWN**. Bound: Normal has few immunes; the unknown grows in NM; the bot skips immune targets with the matching element (B2 above) |

Act 3 counts: GREEN 0, YELLOW 2, RED 1, UNKNOWN 1.

### Act 4, Act 5 and Nightmare band (clvl 32 to 40 and on)

| Skill | Trace | d2d deviation | Live drill | Bot fix | Verdict |
|---|---|---|---|---|---|
| Lightning (49), from clvl 40 | `sk-lightning-trace.md` (D2 piercing one unit per frame and last unit skipped, V read) | SKLIGHT D2 (about 10 percent more hits per cast in packs of 2 to 6, modelled), D3 cast length (same as Chain Lightning), D5 | none | FIXCASTGAP as above | **RED** today for the same reason as Chain Lightning (cast length), YELLOW once the 19 tick candidate is in the binary and drilled; the D2 effect alone would be YELLOW (about 10 percent more hits on d2d) |
| Energy Shield (58) | `sk-armor-trace.md` (V read) | SKARMOR D3 state stays at 0 mana, D4 arithmetic, D6 | DRILLS15 E1 FAIL (compared to trace: life lost 49.0 against 48.5, mana 3.02 to 1.05, shield stays) | none; the table never buys it | N/A (not bought; RED if the plan starts buying it until the absorb fix) |
| Skill delay (Blizzard 45, Meteor 30, Hydra 40, Fire Wall 35 frames) | `sk-mana-trace.md` D6 (V read) | d2d does not enforce it | DRILLS15 M6 FAIL (Blizzard casts 1 to 4 s apart all accepted) | bot models `SkillSpec.Delay` (B6) | N/A (no skill of the default build has a delay; applies to the opt-in Blizzard table) |

Act 4 / 5 / NM counts: GREEN 0, YELLOW 0, RED 1, UNKNOWN 0. SKRESIST (above) is the NM gate row and is counted once, in Act 3. The opt-in Blizzard (END) table adds SKGROUND (row 22, not traced), Ice Bolt, Ice Blast, Glacial Spike, Cold Mastery and Warmth: all UNKNOWN or LIVE-GAP, outside this gate until the user picks that build.

Grand total: GREEN 2, YELLOW 12, RED 3, UNKNOWN 2 (19 rows; the common rows count once).

## 2. Gate rule proposal per act

A "gate soak" is a Gate 2 or Gate 3 soak (docs/review-strategy.md ladder, soak-gates "Gate 2 and Gate 3 on d2d"). The gate says what the bot may be sent into, and what the soak report must show so a reader can audit the skills afterwards. It does not replace the soak-gates criteria, it adds to them.

Per-skill report fields (every act, every skill the bot cast): casts started, casts that made a missile or effect, casts cut before the action frame, mana spent (256ths if the backend gives them), kills credited per skill and casts per kill, hits per cast where the log has them; Frost Nova: novas cast and monsters per nova; armors: casts, recasts, seconds the armor was absent (sampled); staff swings; d2d branch and commit and whether cast length for the Lightning family reads 13 or 19 ticks. Whether `d2dsoak report` already prints each field is U; the card that adds the missing ones is SOAKSKILLS (section 3).

| Act | Must be GREEN, or YELLOW with its stated effect accepted by the user, before the bot is allowed to progress INTO this act in a gate soak | The soak must additionally report |
|---|---|---|
| A1 (a Sorceress fidelity soak, and any run that attempts Andariel) | C1 GREEN. C2, C3, C4, Fire Bolt, Charged Bolt, Frozen Armor YELLOW with effects accepted. Frost Nova RED resolved: NEXTHIT landed and one ring drill passed, OR the user waives it with the workaround "novas off" and the report shows novas 0. Staff swing UNKNOWN accepted as bounded (clvl 1 to 3) or SKMELEE traced. fixbuff and fixmouth live proofs done | per-skill fields above; armor-absent seconds; the Andariel attempt flagged "armor strike effect overrated on d2d" in the report; casts per kill for Charged Bolt against the d2d baseline of `d2d-casts-real.md` |
| A2 (progress past Duriel into Act 3 is a separate go; this gate is for entering Act 2) | A1 gate met. Charged Bolt 13 to 20, Teleport, Frozen Armor 2, merc warp YELLOW with effects accepted. Teleport: hops at most 18 subtiles and aimed at walkable cells (bot rule, tested), refused hops counted | hops tried, hops that moved, mana per hop, mana wasted on refused hops, merc distance after a hop, armor-absent seconds at level 2 |
| A3 | A2 gate met. Lightning Mastery YELLOW accepted. Chain Lightning must be YELLOW or better: NEXTHIT landed AND the 19 tick candidate in the binary with a ticks drill reading 19 AND FIXCASTGAP merged and its live proof done. SKRESIST traced or the user accepts UNKNOWN for Normal | cast length ticks read for Chain Lightning, hops per cast, casts dropped by the gap, immune targets skipped, kills per skill |
| A4 / A5 | A3 gate met. Lightning YELLOW or better (same conditions; its pierce effect, about 10 percent more hits on d2d, accepted). Energy Shield only if the plan buys it, then its absorb fix and drill first | as A3, plus Lightning casts and pierce hits |
| NM and later | advisory until a run reaches it: SKRESIST traced and a drill for immunities and the Static Field floors if that skill is learned; SKGROUND and the skill delay only if the Blizzard (END) table is the build | immune encounters, casts wasted on immune targets, resist-affected kills |

Rule of thumb for the coordinator: the A1 gate blocks counting the d2d container's 24 h soak as a Sorceress fidelity proof; the A2 gate blocks Act 2 progression; later gates stay advisory until a run gets there (as the matrix proposed).

## 3. What is still needed to turn YELLOW, RED and UNKNOWN rows green

Size S = under a day, M = a few days, L = a week or more. "Slot" = needs a hidden d2d slot (no game.exe instance is needed by any card here; a game.exe read of the real game is a separate, later step). Ordered by the act they unblock, cheapest unblocking first.

| # | Card | What it does | Rows it moves | Size | Live slot |
|---|---|---|---|---|---|
| 1 | FIXBUFF live proof (existing) | rerun the DRILLS13 armor recast as a long fight, confirm the `debug states` reply and ids | Frozen Armor (bot gap closed) | S | yes |
| 1 | FIXMOUTH live proof (existing; not a skill, but A1 progression) | DW.m3 three times and DW.m4 | Den exit, A1 entry | S | yes (same slot session) |
| 2 | NEXTHIT (new; SKLIGHT D1 + D2, SKNOVA D1 + D4) | d2d fork patch: unit state `justhit` for NextDelay frames and the hop-search skip, a stateful test mode on a real build, then one ring drill and one hop-line drill | Frost Nova RED, Chain Lightning RED (half), Lightning D2 | L | proof only |
| 3 | LIGHTSEQ and FIXCASTGAP live proof (existing FIXCASTGAP; carry fix/lightning-sequence into the lab binary; ship together) | ticks drill (19 steps), gap drill, command drop count | Chain Lightning RED (other half), Lightning RED, cast-gap row | S to M | yes |
| 4 | SKSMOKE-A1 (new, the matrix step 4 for A1) | live drills "SK": Fire Bolt and Charged Bolt damage means at levels 1, 4, 8, 13, 20 against the oracle, kills per cast on a pack, Frost Nova ring after item 2 | Fire Bolt, Charged Bolt, C3 | M | yes |
| 5 | SKMELEE (existing row 20) | trace the player's blow, then a drill | staff swing UNKNOWN | M | later, S |
| 6 | ARMORHIT (new; SKARMOR D1 harness) | d2d test mode for `buff_events` against `sk_armor.py`, then a Frozen Armor freeze-rule candidate | Frozen Armor YELLOW toward GREEN | M | proof only |
| 7 | FIXMANAMODEL (new bot card; SKMANA B1, B2, B3) | regen in 256ths per frame, book mana at the action frame, potions over 5.12 s | C2 | S | no |
| 8 | Teleport target rules on d2d (existing; SKTELE D1 to D4, drills15 T2 to T4) | room limit, snap, units, refused hop free; test mode on a real Level first | Teleport YELLOW toward GREEN | M | proof only |
| 9 | Merc warp (existing; SKTELE D7, T5) | move the hireling with the hop; game.exe confirmation by play is a later live step | merc row | S | yes |
| 10 | SKRESIST (existing row 21) | resist and immunity by difficulty | SKRESIST UNKNOWN, NM gate | M | later |
| 11 | SKPASS (existing row 23) | masteries, Warmth, Telekinesis, +skills | Lightning Mastery | S | yes |
| 12 | Energy Shield buy decision and absorb (existing; E1, SKARMOR D3, D4) | only if the plan buys it: pure-function differential, d2d fix | Energy Shield N/A | M | proof only |
| 13 | Skill delay on d2d (existing; M6) and SKGROUND (existing row 22) | only for the opt-in Blizzard table | N/A rows | M and L | yes |
| 14 | SOAKSKILLS (new) | `d2dsoak report` per-skill fields of section 2 | the audit trail of every gate | S | no |

Items 1 to 4 and 14 together turn Act 1 into a fully audited gate; item 3 plus 2 does Act 3 and Act 4 / 5.

## 4. Proposed spec delta (text only; for the coordinator)

Target spec: `docs/contracts/soak-gates.md`. Not applied here.

```
## ADDED Requirements

### Requirement: Sorceress skill gate per act
A gate soak (Gate 2 or Gate 3 on d2d) SHALL NOT be counted as a Sorceress fidelity proof for an act, and the bot SHALL NOT be sent into that act in such a soak, unless every skill row of docs/research/sorc-act-gate.md for that act and the acts before it is GREEN, or YELLOW with its effect accepted by the user in docs/decisions.md. A RED row blocks until it is fixed or waived with a named workaround. An UNKNOWN row blocks unless the user accepts its stated bound. The soak report SHALL name the d2d branch and commit and SHALL list per skill the casts, the casts cut before their action frame, the mana spent and the kills credited.

#### Scenario: A red row blocks the act
- **WHEN** the Frost Nova row of the Act 1 band is RED and no waiver is recorded
- **THEN** the Act 1 gate soak is not counted as a fidelity proof

#### Scenario: A waiver names its workaround
- **WHEN** the user waives the Frost Nova row with "novas off"
- **THEN** the report shows novas 0 and the gate accepts the row

#### Scenario: Missing skill fields
- **WHEN** the soak report lacks the per-skill casts, mana or kills
- **THEN** the gate is not auditable and the soak is not counted
```

## 5. Dead ends and limits

- No verdict here comes from a game.exe run: all game.exe statements are reads or emulator runs of the traces. The d2d drills compare d2d with those reads.
- The counts of 6.2 hits per nova cast and 73 percent hop re-hits are models on V-run flights (tag I), not measured on a d2d build; NEXTHIT's first job is to measure them.
- DRILLS16 (running) and the sknova merge (MERGE168) may change a verdict; rerun the tables from the evidence column.
- Whether the lab binary carries the whole-point damage fix and fix/lightning-sequence is U; it is the first thing the soak report names.

## Shared-file lines for the merger

- BACKLOG: actgate (docs only): per-act Sorceress skill gate drafted (docs/research/sorc-act-gate.md): GREEN 2, YELLOW 12, RED 3 (Frost Nova A1, Chain Lightning A3, Lightning A4-A5; cause NEXTHIT plus the 19 tick cast length), UNKNOWN 2 (staff swing, SKRESIST); new cards NEXTHIT, SKSMOKE-A1, ARMORHIT, FIXMANAMODEL, SOAKSKILLS, LIGHTSEQ with FIXCASTGAP; proposed spec delta inside the doc for soak-gates.
- INDEX: `docs/research/sorc-act-gate.md`: per-act Sorceress skill gate, verdicts, gate rule, card list.
- CHANGELOG: "actgate: draft per-act Sorceress skill gate (verdict rows from the traces and DRILLS11 to 15, gate rule, ordered cards)".
- decomp-queue: none (rows 20 to 23 unchanged).
