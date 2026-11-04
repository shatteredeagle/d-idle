---
id: TASK-9
title: "m1-liveproof: live drills for the basics; M1 exit"
status: In Progress
assignee: []
created_date: '2026-11-04T00:00:00Z'
updated_date: '2026-11-04T00:00:00Z'
labels:
  - "G-C"
  - "soak"
dependencies: []
priority: low
ordinal: 9000
card: "m1-liveproof"
owns:
  - "docs/basics-audit.md"
  - "docs/basics-live.md"
  - "docs/primitives.md"
  - "docs/probes/basics/**"
  - "cmd/d2drill/**"
  - "internal/drill/**"
  - "docs/probes/drills/**"
exclusive:
  - "soak-slot:0"
branch: "agent/drills"
agent: "drills"
result: "D0+D1 live 2026-10-31 (docs/probes/drills): P7a PASS (engine auto-belts a ground potion; same kind goes above in its column, new kind lowest free column), P7 PASS on end state (belt gets both, pickit KEEPs, but the looter reports picked 0: bookkeeping on d2d), P1 FAIL (F8 reproduced: lowest-free-box fill puts hp in mp columns), P2 PASS (5 belt types, reader boxes = engine boxes: 8/8/12/12/16), P3 PASS; BLOCKED P1s (click modifier verb) and P3m (merc cheats); each drill twice, identical, also across two processes; P35 gate live-tested (D0). Not verified live: nothing of D3 to D9 (since run by DRILLS3). 2026-10-31 DRILLS2: D1 rerun after the primitive fixes all PASS (P1 hp/mp columns right, P7 picked 2, P1s PASS with the click modifier verb, P2, P3; P3m BLOCKED); D2 (docs/probes/drills/2026-10-31-d2.txt): portal verb PASS (D2.a one click one portal one source, D2.a2 object 8 ticks after the click vs trace about 7, D2.b empty tome no click, D2.c scroll beside empty tome, D2.e busy waits, D2.h one pair), FAIL D2.t and D2.g (no portal walk-in works in the lockstep lab, W1 in docs/probes/drills/README.md), BLOCKED D2.d (needs debug hit) and D2.f (needs a working walk-in); P12 P14 P23 not yet drilled. 2026-10-31 DRILLS3: D3 to D9, D2.d/D2.d3 and the hover audit H run live on the lab binary with the test cheats, each drill twice, P35 gate around each (docs/probes/drills/2026-10-31-{d3,d4,d5,d6,d7,d8,d9,d2d,h}.txt, README section DRILLS3, hover-audit.md): PASS P4 P5-items P6 P8 P9 P12 P14 P15 P16 P17 P18-revive P22 P24 P26 P28 P30 P31 P32 P33 P34, FAIL P5 gold deposit, hover audit FAIL on field objects and on the merc claim/revive path under lockstep; BLOCKED P13 P18-claim P19 P20 P21 P25 P27 P29 P36 and the merc drink with the missing pieces named in the README. The card stays open: those pieces and the basics-audit update (3.1) remain.\""
live: ""
merge_hash: ""
overlap: ""
source: "openspec-final:openspec/changes/m1-liveproof"
contracts: ["proposed/m1-liveproof"]
---

## Description

<!-- SECTION:DESCRIPTION:BEGIN -->
Policy work (trips, kit, gold floor, early game) sits on basic actions that were only tested against fakes: the healing belt is empty in about a third of the minutes of the long soaks, ground potions are skipped as "already in the pack" while the belt is empty, town portals fail thousands of times, and several specs are tagged UNVERIFIED (skill binding, meter totals, the portal route of town interact, waypoint row choice, belt rows by belt type). docs/primitives.md lists 36 primitives with a definition of correct each. docs/basics-audit.md lists 10 live checks and 2 unknowns (starting gold, unidentified sell prices) that no offline test can close. This card becomes the live drill suite that proves them on a hidden d2d with throwaway characters, before policy work resumes.

What changes:
- A drill harness (`cmd/d2drill`, scripted scenario runner) and one drill per primitive group, run on one hidden d2d slot; output is a table P-id, drill, setup, pass condition, measured, verdict (PASS, FAIL, BLOCKED), cheat ledger ids.
- Setup only through the logged internal/cheat tool or cmd/d2cheat, each with a reason, on throwaway characters, hidden instances, offline only. Gate characters are never used.
- End states are read through the StateReader (state, items, belt, UI flags, level info, unit table), never from the bot's own log lines.
- docs/primitives.md gets a verdict column filled per drill; docs/basics-audit.md checks 1 to 10 and the two unknowns are run as drills D2 to D4 and recorded in docs/basics-live.md.
- Acceptance: every primitive with criticality S or A in docs/primitives.md has a live drill PASS, or a FAIL with a fix card, before policy cards (goldfloor follow-ups, m2-*, m3-*) resume.
- No production code in this card except the harness; the harness is a separate implementation task (see tasks).

Next step: the two open rows of the M1 exit list

Gates (from the card):
- docs/primitives.md: every primitive with criticality S or A has a drill verdict PASS (or FAIL with a named fix card) in its evidence column, checked by the coordinator by eye
- docs/basics-audit.md coverage table has 0 G rows and every S1 or S2 row is C with a test name in the offline-test column and a live proof line (`grep -c '| C ' docs/basics-audit.md` is at least 15)
- starting gold and unidentified sell prices recorded V in docs/basics-live.md with cheat ledger ids
- tools/agentreap.sh m1-liveproof leaves nothing; changecheck -cli 0 warnings; leakcheck 0

Source: git tag openspec-final, openspec/changes/m1-liveproof/ (this task is the copy).
<!-- SECTION:DESCRIPTION:END -->

## Acceptance Criteria
<!-- AC:BEGIN -->
- [x] #1 1.1 `cmd/d2drill` and `internal/drill`: scenario type, cheat-step wrapper, StateReader before/after, result table, ui-clean pre/post check (P35); offline tests with mem.Fake for the runner itself
- [x] #2 1.2 Name the missing cheats and verbs in the report (durability, merc death, `debug wp`, click modifiers); drills that need them report BLOCKED
- [x] #3 2.1 D1 belt: P1 P2 P3 P7 (fill from pack, belt rows per belt type, drink per column and merc, ground potion with empty belt, shift-click and ctrl-click behaviour recorded)
- [x] #4 2.2 D2 portals and town entry: P10 P11 P12 P14 P23 (scroll and tome, 10 casts, return, tome fill, every Act 1 NPC window, waypoint row choice on 3 activated waypoints). result: P10 D2.a/a2/b/c/e/h/t/g and D2.d, D2.d3 (cut cast, DRILLS3) PASS, P11 D2.f PASS (enterportal), P12 D3.tp PASS, P14 D3.n PASS, P23 DW.* PASS for act 1 and 2 rows (fixwp); 10 casts: D2.a2 ran 3, not 10; cross-act rows pend...
- [x] #5 2.3 D3 vendor and gold: P15 P16 P17 P13 P5 P8, plus the two unknowns (starting gold on 2 classes, unidentified sell prices). result (2026-10-31-d3.txt, d3g0-class0/1): P15 D3.b, D3.b2 PASS, P16 D3.r, D3.r0 PASS (finding: needRepair ignores gold), P17 D3.h PASS, P8 D4.g, D4.cap PASS (purse cap 10000 at clvl 1), P5 D3.s, D3.s2 FAIL (gold deposit impossible: hoverFirst hides TypeChar and the promp...
- [x] #6 2.4 D4 pack: P4 P9 P6 (placement with fragmentation, pack-full pickups once per item, cube recipe). result (2026-10-31-d4.txt): P9 D4.a, P4 D4.b/D4.c, P6 D4.x PASS (verified live); fragmentation with 2x4 holes not built (plain 1x1 fill)
- [x] #7 2.5 D5 combat: P26 P27 P28 P29 (BindSkills count and cast effect, walled target, threshold drinks with meter totals, retreat then TP). result (2026-10-31-d5.txt): P26 D5.a PASS (bind 2, cast spends mana; hit effect not checked), P28 D5.b PASS (life thresholds; mana drink and Meter totals need a fight), P27 D5.c and P29 D5.d BLOCKED: no monster (`debug spawn` not built)
- [x] #8 2.6 D6 movement: P22 P24 P25 (10 walks, 5 exits, pocket escape). result (2026-10-31-d6.txt): P22 D6.a, D6.b PASS (10 of 10 within 1.5 cells, camp and Blood Moor), P24 D6.c PASS (5 of 5), P25 D6.d BLOCKED: no way to pin the character
- [x] #9 2.7 D7 build: P30 P31 P32 (level cheat, points, equip swap with empty cursor). result (2026-10-31-d7.txt): P30, P31 PASS (level 12, 55 stat and 11 skill points all spent), P32 PASS (cap worn, cursor empty; a swap of two worn items not built)
- [x] #10 2.8 D8 NPC and travel: P18 P20 P21 P19 (merc claim, hire and revive, Akara reward, Warriv, no gamble when poor). result (2026-10-31-d8.txt): P18 revive D8.a PASS (free at merc level 1, U); BLOCKED with the exact missing piece: claim D8.c, P20 D8.q, P21 D8.w (`debug quest` sets bit 0 only), P19 D8.g (no gamble code), merc drink D8.m (`debug merc hurt`)
- [x] #11 2.9 D9 lifecycle and policy re-gate: P33 P34 P36 (10 leave and enter cycles, death and corpse, paired 8 seed trip and empty-belt check as in goldfloor). result (2026-10-31-d9.txt): P33 D9.b PASS (10 cycles), P34 D9.a PASS (staff lost at death, back from the corpse); P36 D9.c BLOCKED: a soak, not a drill (tools/d2dsoak)
- [x] #12 2.10 DRILLS4 (2026-10-31, docs/probes/drills/README.md DRILLS4; results 2026-10-31-d10, -dw3, -repair, -repair-poor). result: D10 fault recovery live on d2d, two runs identical: kill -9 of the slot's d2d, bot restarted it in 1.5 s and was back in game within 7 s, loop resumed, saved XP not lower (3429 before, 18194 to 19989 after the graceful stop), no leftovers in the registry or process table...
- [x] #13 2.11 DRILLS5 (2026-10-31, docs/probes/drills/README.md DRILLS5; results 2026-10-31-drills5-*.txt): live proofs of the merged fixes and the drills the fork cheats unblocked. PASS: D3.s, D3.s2, D3.r0, D3.r1, D3.tp (fix-bank), H.obj, H.obj.chest, H.obj.shrine, D8.c, D8.a, D3.n, D3.b (fix-obj), D3.r estimate 19 = paid 19, D3.b2, D3.u (fix-price), D1.P1b, D1.P7b (fix-belt2, fix-pickup), DW.m1, DW.m2...
- [ ] #14 3.1 Fill the verdict per P-id in docs/primitives.md and update docs/basics-audit.md (rows to C with test and live proof line); open a fix card per FAIL
- [ ] #15 3.2 changecheck, leakcheck, reap
<!-- AC:END -->
