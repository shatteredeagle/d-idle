# Primitives P1..P36: what is still open (2026-11-04)

> status: current snapshot, docs only (card `primcheck`). `docs/primitives.md` is NOT edited; this page reads it, the drill README, the archived fix cards and `git log` at origin/main `7b4baf3`. No game, no d2d slot, nothing verified live by this card.
> Tags: **V** = read in the named repo file or commit this session; **S** = secondary (a doc quoting another doc); **U** = unverified. **UNSURE** = the repo does not settle it; no guess is made.

Method and limit. The last live drill file in `docs/probes/drills/` is dated 2026-11-01 (DRILLS29, `2026-11-01-*`; no 2026-11-02..04 drill files exist, V `ls`). Everything dated 2026-11-02 and 2026-11-03 below is therefore OFFLINE code (cards beltfix to beltfix5, kitstall, kitstall2, mercfix, mercwire, hirefix, townshop, codefix, review3fix, navexit21, navexit22, gate1fix, areareach, recoverykit) plus unattended Gate 1 soak records, which are live on d2d but are soaks, not drills. A primitive row "changed since" means the repo has a later fix or soak that touches it, not that it was re-drilled.

## 1. Open S or A primitives, ranked

Ranking = criticality (S before A) x how much is still unproven live x how recently code changed under it. Only primitives with a real open item are listed; the rest are "closed" in the table.

| rank | prim | crit | open item | why it ranks here |
|---|---|---|---|---|
| 1 | P10 / P12 | S / A | tome stat 70 reads as 0 in the field (`inv-tp-dump ... counted=0 raw70=no70`) while the town reads 13 to 19: a false "portal empty" flips the verdict after a 120 s debounce and starts a trip loop. Cause (reader or d2d side) is U. docs/townshop.md "Not fixed / U" (V) | the only open portal-source defect; townshop fixed the cost (chore budget, 2-scroll proof) but "awaits a live check" (BACKLOG TOWNSHOP, V) |
| 2 | P18 | A | paid hire, revive price and merc kept after the free-rogue claim are offline fixes (mercfix, hirefix, mercwire, 2026-11-02/03) with "not verified live"; Gate 2 of 2026-11-02 failed on merc=none. IW and 0A hireling codes U live (docs/research/hirefix.md, V). Live probe still open (BACKLOG MERCFIX, V) | a failed claim blocks the Andariel route (gate2 analysis, S) |
| 3 | P29 | S | the retreat action itself was never run live: D5.p29 only records that Safety would choose chicken/drink-rejuv (DRILLS5), D-portalmelee records `chicken` as a decision, not executed (DRILLS12: "the chicken action itself is not run by the harness"). codefix F1 changed the abort path (RetreatOut vs LeaveGame) offline only (docs/research/codefix.md, V) | S primitive with no live action drill |
| 4 | P23 | S | DW.m3/DW.m4 mouth gate PASS (DRILLS23) but 50 `walk did not finish in 2m0s` in the last Den cycle at 29 percent life; card idea MONSTALL "unproven, investigate only if a real character stalls the same way" (BACKLOG, V). DW.m4 ends with `open: inventory` in DRILLS18..23 (see P35) | open but explicitly parked by the backlog |
| 5 | P26 | S | d2d differs from game.exe in D15 (Teleport room limit, blocked target, merc not moved, cut Fire Bolt keeps mana, Energy Shield, Blizzard double pay; DRILLS15, V); `cmd select` effect on the right-click skill UNVERIFIED (DRILLS10); sksmoke-a1: Fire Bolt d2d PASS vs table, Charged Bolt no verdict, game.exe live BLOCKED (openspec archive sksmoke-a1 result, V) | these are fork-vs-game deviations, not bot FAILs; whether each is an upstream PR item is UNSURE from this repo alone |
| 6 | P35 | S | DW.m4 verdict FAIL only on the dirty-UI check (`open: inventory`), identical in DRILLS18, 19, 21, 23 (README lines 551, 583, 632, 726, V). README calls it "the same harness issue"; no fix card found. Product vs harness: UNSURE | hygiene check that fails on a recurring drill |
| 7 | P5 | A | closed live (DRILLS5), but beltfix5 (2026-11-03) added a stash-gold withdraw chore and counts stash gold in `Input.Gold`; codereview1 F6 (withdraw only when the purse cannot pay) and F7 (in-town futility latch) are "Open (not done)" in codefix.md section 3 (V). Not drilled live since | new code on a closed primitive, no live proof |
| 8 | P1 / P3 / P7 | A / S / S | closed by drills; later belt policy cards (beltfix..5) are Gate 1 fixes, and Gate 1 multi-seed 2140-ms4 PASSED the belt criterion on 5 of 5 seeds (docs/soaks/gate1-multiseed-20261103-2140-ms4.md, V). Earlier ms1 and ms3 records are NOT PASSED (cards gate1multi ms1, ms3 results, V) | nothing open on the primitive; listed only because 4 policy cards touched it |

Not ranked because only B or C: P6, P13, P19 (not built), P21, P24, P25, P36 (all see table).

## 2. BLOCKED drills: is the blocker lifted?

Every BLOCKED row in docs/primitives.md names a missing fork verb or cheat. docs/forkcheats.md (cheats, 2026-10-31, V: file read) and docs/forkdbg.md (2026-11-01, V) show every verb exists in the lab binary. The table below says what the drill later did.

| drill | blocked on (primitives.md) | verb exists? | rerun result | state now |
|---|---|---|---|---|
| D2.d cut cast | no hit cheat | `debug hit` (forkcheats) | PASS D2.d, D2.d3 (DRILLS3) | closed |
| D2.f portal round trip | walk-in | enterportal (not a cheat) | PASS (D2.f, DRILLS3 row 93; D2.* 11/11 x2, DRILLS7) | closed |
| D1.P3m merc drink | no merc cheat | `debug merc hire/kill/revive/hurt` (forkcheats, forkcheats2) | D8.m PASS (DRILLS5, `merc hurt 20`: life 20 to 32, one potion). D1.P3m as a drill id still "BLOCKED" in DRILLS9's D1 line: the drill itself lacks the merc step (V, primitives.md DRILLS3 note) | covered by D8.m; the D1.P3m id is not rerun (UNSURE if the card is still wanted) |
| D3.i identify | no unid magic cheat | `debug give <code> pack q4 unid` (forkcheats2) | D3.i, D3.u PASS (DRILLS5) | closed |
| D3.g0 unid sell price | same | same | D3.u: unid magic Cap sells 32 = cost/2 (DRILLS5) | closed |
| D5.a hit effect (P26) | no monster | `debug spawn` | probe RECORDED Fire Bolt 35 to 32 mana, zombie hp 11 to 8 (DRILLS10) | closed for a hit; `cmd select` U |
| D5.b mana in fight, Meter totals | no monster, `Attach` built no Meter | spawn; fixmeter | D5.p28 PASS (DRILLS5); D-meter PASS (DRILLS10) | closed |
| D5.c, D5.d (P27, P29) | no monster | `debug spawn` | D5.p27, D5.p29 PASS "recorded" (DRILLS5); D5.dodge PASS | P27 closed; P29 only recorded (open, section 1 rank 3) |
| D6.d (P25 pocket) | no pin | `debug pin` | FAIL (DRILLS5), PARTIAL (6), FAIL (7), PASS x3 (DRILLS8, 225 ticks); D22-PC/D22-P pocket hops PASS (DRILLS24..27) | closed; harness caveat: reset navigator stall memory between pocket drills (`ResetStallMemory`, DRILLS27, V) |
| D8.m | no merc hurt | `debug merc hurt` | PASS (DRILLS5) | closed |
| D8.c Rogue claim | quest bit | `debug quest set` | PASS run 1; BLOCKED run 2 by design: reward is once per character (DRILLS9) | closed (once-per-character) |
| D8.q, D8.w (P20, P21) | `debug quest` | exists | PARTIAL PASS (DRILLS5), PASS through the bot's own talk nodes (DRILLS10, fixtalk) | closed for Akara and Warriv act 1 to 2 |
| D8.g (P19 gamble) | no gamble code | n/a | not run | open, not built (V: only a menu-entry flag in `internal/d2dctl/town_verbs.go`, no gamble flow) |
| H.monster | no monster | `debug spawn` | UNSURE: not found in later rows; H.obj, H.ground, H.pause PASS | UNSURE (no evidence either way) |
| D9.c (P36) | "a soak, not a drill" | n/a | Gate 1 soaks, multi-seed 2140-ms4 PASSED 5 of 5 (V) | closed as P36 evidence |
| Waypoint `knownWaypointPos` towns 75, 103, 109 (P23) | missing table rows | n/a (code) | DW.a9 to a11 PASS, no `none on level N` (DRILLS5, fix-nav) | closed |
| Gate 2 (P36 longer run) | n/a | n/a | the 2026-11-02 run FAILED (BELT at virtual minute 432, NOKILL, merc, docs/research/gate3-prep-20261103.md, V). No Gate 2 rerun record exists on main (V `ls docs/soaks`) | UNSURE whether a rerun exists outside main |

## 3. Primitives never drilled live

| prim | state | evidence |
|---|---|---|
| P19 Gamble | never built, never drilled | primitives.md row, DRILLS3 D8.g BLOCKED; no gamble flow in code (V) |
| P29 Retreat and escape use | decision recorded only; the escape action never run in a drill | DRILLS5, DRILLS12 |
| P21 Travel NPC / act change | only Warriv act 1 to 2 (`go east`, area 40); the other act hops (Act 2 to 3, 3 to 4, 4 to 5) not drilled | DRILLS5, DRILLS10; other acts UNSURE |
| P20 Quest dialog and reward | only the Akara skill-point reward; Kashya and other quest NPCs not drilled | DRILLS10 |
| P27 Target selection | recorded kills/ignores of a champion pack and D-sync arrivals; the "walled monster never targeted" setup of the original drill row was not found run | DRILLS5, DRILLS7, DRILLS8 |
| P18 paid hire and paid revive | free Rogue claim and free (cheat) revive drilled; paid hire path offline only | hirefix result (V), D8.a note "U whether free revive is a rule at merc level 1" |
| P35 UI hygiene | never a drill of its own; runs as the P35.pre/post gate around every drill (README line 13, V) | README |

Everything else (P1..P17, P22..P26, P28, P30..P34, P36) has at least one live PASS or a live FAIL-then-PASS in the table below.

## 4. The table

Columns: verdict = latest row in docs/primitives.md (dates are the drill group dates); evidence = the doc, commit-range or file read (V); later = a fix, soak or drill after that row; open = what remains.

| id | latest verdict (date) | evidence (V = read in repo) | changed since the row? | open item / state |
|---|---|---|---|---|
| P1 Belt fill | PASS D1.P1, P1s, P1b (2026-10-31); P1s PASS in group x3 (2026-11-01) | docs/primitives.md rows 55-56, 133, 200; fix-belt, fix-belt2 cards; V | yes, offline: beltfix to beltfix5 (2026-11-02/03) kit and shop timing; Gate 1 ms4 belt criterion PASS on 5 seeds (2026-11-03) | closed |
| P2 Belt layout | PASS D1.P2, 5 belt types (2026-10-31) | primitives.md row 57; V | no | closed |
| P3 Drink by column | PASS D1.P3 (2026-10-31); merc D8.m PASS (2026-10-31) | rows 58, 139; V | codereview1 F1 fight-abort side effect fixed offline by codefix (2026-11-03) | closed; D1.P3m id itself not rerun (UNSURE) |
| P4 Pack fit / make room | PASS D4.b, D4.c (2026-10-31) | row 106; V | no | closed |
| P5 Stash items and gold | PASS (was FAIL) D3.s, D3.s2 (2026-10-31); D3 12/12 (2026-10-31 DRILLS8) | rows 129, 188; fix-bank; V | yes, offline: beltfix5 withdraw chore (2026-11-03) | closed; F6/F7 of codereview1 open as code items (section 1 rank 7); stash cap 2.5M still U |
| P6 Cube | PASS D4.x (2026-10-31) | row 108; V | no | closed |
| P7 Pickup potion, belt priority | PASS D1.P7, P7a, P7b (2026-10-31) | rows 59, 133; fix-pickup; V | no | closed |
| P8 Pickup gold, full pack | PASS D4.g, D4.cap (2026-10-31) | row 107; V | no (primitives-review F2 loot-seam fix merged earlier) | closed |
| P9 Pickup item by rule | PASS D4.a (2026-10-31) | row 105; V | no | closed |
| P10 TP and enter portal | PASS D2.* 11/11 x2 (DRILLS7, 2026-10-31); D-portalmelee PASS (DRILLS12, 2026-11-01); D-castcut RECORDED (2026-11-01) | rows 84-92, 179, 218-226; fixcrowd, fixportalmelee, enterportal; V | yes: codefix F8 cut-cast memory (offline, 2026-11-03); navexit21 empty-tome scroll order (offline) | open: tome stat 70 dropout in the field (section 1 rank 1, U cause) |
| P11 Return through portal | PASS (was BLOCKED) D2.f (2026-10-31) | row 93; V | codefix F4: chain walk home restricted to `ChainHomeArea` (offline) | closed |
| P12 Buy TP / fill tome | PASS D3.tp (tome 5 charges, fill target is policy) (2026-10-31) | row 97; fix-bank note; V | yes: townshop (2026-11-02), navexit21 (2026-11-03), both offline | open: same as P10 stat 70 dropout; "awaits a live check" |
| P13 Identify | PASS (was BLOCKED) D3.i, D3.u (2026-10-31) | row 132; V | no | closed |
| P14 Approach NPC | PASS D3.n (2026-10-31); H.pause 12/12 (2026-11-01 DRILLS9, fixstall4) | rows 98, 198; fixstall..fixstall4; V | no | closed (Cain not in a drill, UNSURE) |
| P15 Vendor buy/sell | PASS D3.b, D3.b2 (2026-10-31) | rows 99, 131; V | yes: kitstall2 sell never touches worn items (offline, 2026-11-02) | closed |
| P16 Repair | PASS D3.r, r0, r1 (2026-10-31); price table D3.rt | rows 100, 130; fix-bank, fix-price; V | no (a repair-gate PR draft exists, prbuildprices2, not a bot change) | closed |
| P17 Heal at NPC | PASS D3.h (2026-10-31) | row 101; V | no | closed |
| P18 Merc hire/revive/feed | PASS D8.a revive, D8.c claim, D3/D8 smoke (2026-10-31/11-01) | rows 117, 140, 188, 202; V | yes, offline: mercfix, mercwire, mercpolicy, hirefix (2026-11-02/03) | open: paid hire, revive quote, merc kept after claim, IW/0A codes; none live (rank 2) |
| P19 Gamble | not built, D8.g BLOCKED (2026-10-31) | row 119; no gamble flow in code, V | no | open: never built, never drilled |
| P20 Quest dialog/reward | PASS D8.q through bot talk nodes (2026-11-01) | row 209; fixtalk; V | no | closed for Akara; other NPCs not drilled |
| P21 Travel NPC / act change | PASS D8.w, Warriv `go east` to area 40 (2026-11-01) | row 210; fixtalk; V | no | closed for act 1 to 2; other act hops UNSURE |
| P22 Walk to a point | PASS D6.a, D6.b, D-run, D-emptyplan x3 (2026-10-31, 11-01) | rows 112, 168, 217; fixrun, fixemptyplan; V | gate1fix R1/R2 walk limit and target cool-down (offline, 2026-11-03) | closed |
| P23 Waypoint / mouth | PASS DW.a* cross-act, DW.m1/m2 (2026-10-31); DW.m3 and DW.m4 PASS (DRILLS23, 2026-11-01) | rows 134, 208, 245-247; fixwp, fix-nav, fixmouth..4, roamleg; V | yes: areareach (2026-11-03, offline, waypoints activated on the walk), review3fix C1 route-ban fix (offline); upstream #153 merged 2026-11-01 (BACKLOG, S) | open: MONSTALL 50 `walk did not finish` (parked); WpCooldownSec removal waits for the lab binary (BACKLOG, S) |
| P24 Level transition | PASS D6.c 5 of 5 (2026-10-31) | row 113; V | yes: navexit21, navexit22 seed-17 border gate (offline, 2026-11-03), review3fix | closed; the navexit fixes are not live-proven |
| P25 Stuck recovery | PASS x3 D6.d (DRILLS8, 2026-10-31); D22-PC, D22-P pocket hops PASS (2026-11-01) | rows 186, 249-252; fixstall..fixstall4; V | yes: gate1fix R2 (offline) | closed |
| P26 Skill bind and cast | D5.a PASS, D22 PASS (2026-10-31/11-01); D15 deviations FAIL (2026-11-01) | rows 109, 75-82, 242-252; V | sksmoke-a1 SK drill design 2026-11-03, live BLOCKED (archive result, V) | open: deviations vs game.exe (S, decisions UNSURE); `cmd select` effect U |
| P27 Target selection | D-sync.tp PASS, D-sync.wp PASS x3 (after harness fix), D5.p27 recorded (2026-10-31) | rows 136, 137, 167, 177, 187; fixmisc, fix-missile; V | no | closed (walled-monster case not found run, UNSURE) |
| P28 Life/mana monitor | PASS D5.b, D5.p28, D-meter (2026-10-31, 11-01) | rows 110, 138, 211; fixmeter; V | beltfix mana reserve, codefix F3 (offline) | closed |
| P29 Retreat and escape | recorded only, not run (2026-10-31) | row 137; DRILLS12 note; V | codefix F1 abort path (offline, 2026-11-03) | open: no live action drill (rank 3) |
| P30 Stat points | PASS D7.a, D7.b (2026-10-31) | row 115; V | no | closed |
| P31 Skill points | PASS D7.a, D7.b (2026-10-31) | row 115; V | no | closed |
| P32 Equip / swap | PASS D7.c (2026-10-31) | row 116; V | yes: kitstall, kitstall2 worn-item rules (offline, 2026-11-02); "staff and gloves loss cause still U" in card kitstall2 result (V) | closed; staff/gloves loss cause U |
| P33 Save/exit/rejoin | PASS D9.b 10 cycles (2026-10-31) | row 121; V | no | closed |
| P34 Death and corpse | PASS D9.a (2026-10-31) | row 120; V | no (post-death policy cards are policy) | closed |
| P35 UI hygiene | gate around every drill; DW.m4 FAIL on dirty UI (2026-11-01) | README line 13, lines 551-726; V | no | open: recurring `open: inventory` at the end of DW.m4, product vs harness UNSURE |
| P36 Trip worth it / gate | D10 PASS x2 (2026-11-01); Gate 1 multi-seed 2140-ms4 PASSED 5 of 5 seeds, mean 10607 vs baseline 9172 XP/h (2026-11-03) | rows 135, 199; docs/soaks/gate1-multiseed-20261103-2140-ms4.md; V | yes: specdelta (belt window counts only funded windows), beltfix..5, gate1fix, recoverykit (2026-11-03) | closed for Gate 1; Gate 2 of 2026-11-02 failed, rerun not on main (UNSURE) |

## 5. Things this page could not settle

- Whether the D15 d2d-vs-game.exe deviations (P26) are tracked as upstream PRs: the bundles are private cards; UNSURE.
- Whether H.monster, D1.P3m and Cain/Warriv-other-acts were ever rerun: no result in docs/probes/drills after 2026-11-01.
- Whether a Gate 2 rerun passed: no record on main.
- Whether the P35 dirty-UI finding of DW.m4 is a bot or a harness defect: the README says harness, no card owns it.

## Shared-file lines for the merger
See `openspec/changes/primcheck/merge-lines.txt`.
