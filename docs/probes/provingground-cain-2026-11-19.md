# Proving Ground rerun: Cain scroll scenarios on main with cainfix2 (2026-11-19)

> status: current (card pgcain). Tags: V verified (run named), S secondary, U unverified. All runs on the builder (main lane, one at a time), lab slot, CHEATED setup (level 17, waypoints, quest board record): mechanics evidence only, never behaviour or balance evidence.

> verdicts: current verdicts per scenario and class are in [proving-ground-status.md](../process/proving-ground-status.md) (generated from the results ledger) and on the board page /pg. The tables below record these runs only; new runs are cited by run id, not added here (design 2026-11-21 Q10).

## Question and answer in one paragraph

The first Proving Ground run (RED `cain.scroll.field`) was on a bot built from the provingground branch; whether it contained cainfix2 is U. Rerun on origin/main d024dcb23 (contains cainfix2, merge c72b2eb05): the tree step now WORKS. The scenario was RED for a scenario reason: the assertion wanted `bks` but the bot had by then had Akara read it, so the pack held `bkd` (the log line `inventory contents ... pack: bkd x1` at the town visit after the tree). With the assertion widened to `bks` or `bkd` (this branch) `cain.scroll.field` Paladin seed 11 is GREEN. The from-camp scenarios are still RED with a different cause (below). The live sandbox tree stall was not reproduced by any run, because the shipped-verbs variant that exists starts from the camp, not at the tree.

## Results (V, builder, bot of this branch)

| scenario | class/seed | build and verbs | verdict | note |
|---|---|---|---|---|
| cain.scroll.field | Paladin/11 | lab verbs, main + cainfix2, old assertion | RED | pack held bkd, not bks (scenario artifact) |
| cain.scroll.field | Paladin/11 | lab verbs, main + cainfix2, assertion bks or bkd | GREEN | tree reached, `Found Scroll of Inifuss`, picked, read at the next town visit |
| cain.scroll | Paladin/11 | lab verbs | RED | chunk starts, a town trip cancels the travel step, chunk stays `queued: waiting for the fight to end`, no tree in 30 virtual min |
| cain.scroll.sandbox | Paladin/11 | shipped devp13 d2d (fork build f32837d8 under the builder, `D2D_BIN_SANDBOX`), no debug verbs for the bot | RED | the same signature as the lab run: identical log lines, so the shipped build does not change this behaviour |
| cain.scroll.field | Sorceress/3 | lab verbs | RED | `not ready: no potions and no gold, no weapon equipped (yours to equip)`: the Sorceress setup has no weapon, the readiness gate refuses the chunk. Scenario setup gap (U: a fresh hero normally carries a weapon), not a bot defect |
| cain.scroll | Sorceress/3 | not run | | time |

## Findings

1. cainfix2 works at the tree on a lab slot (V, run above): `quest board: step Take the Inifuss bark`, loot picked the scroll, no "exploring for object class 30" loop, zero zone cycling. The earlier RED was either a pre-cainfix2 binary (U) plus the assertion issue.
2. Small oddity (S): right after the pickup the step check reads the pack 8 ms later, still sees no bark ("ran but is still needed (evidence 2)"), and the board abandons that slice; the next pass finds it. Harmless here, but it is a race between pickup and the world read.
3. From the camp the chunk never reaches the tree in 30 virtual minutes (lab and shipped builds alike). The log shows: chunk starts, first town trip (`kit: healing potions 2/4`) cancels the step, then `queued: waiting for the fight to end`; afterwards the hero levels in Stony Field. The queue gate `safeToPause` (no hostile in the clear radius) is the suspect. Branch `agent/cainfix3` adds a 45 s grace after which the chunk starts anyway (unit test, two classes). Live (scenario rerun on the lab slot): the chunk then does start (`starts although hostiles are near`) and the step walks and explores, but the weak CHEATED hero (211 life) is pulled to town again and again (healing potions, emergency portal) and still has no scroll after 30 virtual minutes. So the grace is NOT shown to be the fix: cainfix3 is NOT queued. Open: whether this is the hero's weakness (setup) or the board's re-start after each cancel; needs a trace of `boardPreempts` decisions (a debug log per decision) before a fix.
4. No `room rows` question could be tested: the sandbox variant is camp-start. A field start on the shipped build needs lab verbs (`goto`) which the shipped d2d does not have. To test the live stall (hero at the tree room, class from presets) on devp13 the scenario needs either a fork verb for devp14 that places the hero (lab only) or a hero played there; the offline `questspec` checker already says room rows are not confirmed on devp13 (R5).

## Changes in this branch (agent/pgcain)

`item_in_pack` gains `or` (further codes that count); `cain.scroll`, `cain.scroll.sandbox`, `cain.scroll.field` accept `bks` or `bkd`. Build-specific data: only the scenario JSON.

## How to rerun

`labrun builder --pull provingground-out --detach -- tools/provingground/run.sh -id cain.scroll.field -class Paladin -seed 11 -parallel 1` (main lane when free, long lane otherwise); the sandbox one with `env D2D_BIN_SANDBOX=/data/build/devp13-f32837d8-d2d/apps/d2d/d2d` in front.
