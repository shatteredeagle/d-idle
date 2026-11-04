# Energy Shield absorb order: game.exe vs the real d2d build (job stateevdiff, 2026-11-04)

Follow-up of `stateevrun-trace.md` (D1: "game.exe absorbs Energy Shield before the resistance pass, d2d after; no real-build differential yet"). This job builds that differential. Nothing here ran a live game; no memory writes; no Go code. Tags: **V run** = observed running (game.exe code under the emulator, d2d's real functions linked from a scratch upstream build), **S** = shown by an earlier trace, **I** inferred, **U** unverified.

## Harness (what ran)

| Piece | File | Role |
|---|---|---|
| d2d probe | `tools/emu/stateev_diff_probe.cpp`, `stateev_diff_build.sh` | links the scratch upstream build's `libgame.a` and calls the REAL `rules::monster_blow` (resistance step) and the REAL `Fight::absorb` (Energy Shield block, `components/game/fight/states.cpp`) on one single-element blow; prints life lost (whole points), mana left (256ths), the ES calc1 / calc2 it used |
| game side + compare | `tools/emu/stateev_diff_run.py`, `stateev_diff.sh` | game.exe's REAL `FUN_0057c6c0` (flag 1: runs the event dispatch and the resistance pass), harness of `ser_events.py` part A (`life_apply.Game`); event 11 is a hook that applies the Energy Shield node's arithmetic (calc1 percent capped by the mana limit, `sk-armor-trace.md` Q5) to the record field; never on poison |

Commands (worker host, from the worktree):

    tools/emu/run-on-worker.sh --ttl 1500 stateev_diff_build.sh > /tmp/d2agent-pa-stateevdiff/build.txt 2>&1
    tools/emu/run-on-worker.sh --ttl 2400 stateev_diff.sh 20000 1 2 > /tmp/d2agent-pa-stateevdiff/out.txt 2>&1

The run took 44 s. The d2d source tree is the read checkout `sorcrest-d2d` (not a git checkout, so its commit is not recorded: **U** on the exact upstream commit). Grid: 20,000 random cases per seed, seeds 1 and 2 = **40,000 cases**: element (phys, magic, fire, lightning, cold, poison), resistance -100..100 (with 0, 100 = immune, -100 weighted in), raw damage 1..1500 whole points, Energy Shield level 0 (20 %, the control) or 1..40, mana 0..600 points (fractions included).

Honest limits: the ES **arithmetic** on the game side is the V-read formula in a hook (tag **I**), not `FUN_005ca6c0` itself (it needs state-list hooks; dead end below). What is V run on the game side is the **order** (event 11 before the resistance pass, inside the real damage function) and the resistance step. calc1 / calc2 come from the d2d probe (`fight.calc` on Skills.txt), reused for the game hook (**I**: same table).

## Results (V run, 40,000 cases, seeds 1 + 2)

| Check | Result |
|---|---|
| Control (ES level 0, resistance step alone): real `monster_blow` vs game.exe's resistance pass, phys / fire / lightning / cold / magic with resistance 0 | 5,420 cases, **0 differ** (life), mana unchanged: the resistance step itself is equal on both sides (S: `sk_resist.py`) |
| Control, magic with resistance != 0 | 391 cases, 382 differ: d2d applies no percent magic resistance to a monster blow (only the flat `mdr`); game.exe applies stat 0x25. Separate rule, listed as D2 |
| d2d self-check: real `Fight::absorb` vs its own source formula | 0 bad in 26,555 ES cases (the probe is the formula) |
| ES cases without the magic-resist rows (25,038) | life differs in **18,532**, by more than 1 point in **8,992**; mana (whole points) differs in the large majority |
| Same cases: game.exe with the absorb moved AFTER the resistance (same game arithmetic, same fine units) differs from game.exe | **16,120** (8,670 by more than 1 point): the order alone changes the result, mostly when the mana limit binds or the target is vulnerable / immune |
| Poison (6,634 cases) | excluded from life: d2d keeps poison as a damage-over-time field (`Taken.poison`), the harness's record path deals it at once; the shield never touches it on either side (mana unchanged in all 6,634 for game and d2d: V run) |

Per element, ES rows (cases / life differs / order matters): cold 5,400 / 4,007 / 4,154; fire 5,418 / 3,931 / 4,108; lightning 5,227 / 3,777 / 3,957; physical 5,179 / 3,807 / 3,901; magic (res 0) 3,814 / 3,010 / 0 (only the whole-point rounding, order cannot matter at res 0).

**First differing case** (seed 1, case 0, magic, resistance 100, raw 1000, ES level 31, mana 55038/256): game 0 life and 0 mana left; d2d 893 life, 254 mana-256ths left. This one mixes D2 (d2d ignores magic resist). **First differing case where only the order matters** with a resistance d2d implements: seed 1, case 9: fire, resistance -100, raw 632, ES level 40, mana 103179/256 = 403 points: game life 220,406/256 = 860 points, mana 1/256 left; d2d 1,063 points, mana 267/256 left. Without the shield the hit is 1,264 points; game.exe takes the shield's share from the raw 632 and doubles the rest, d2d takes it from the doubled 1,264. Smallest cases: seed 1 case 1, physical, resistance 1, raw 5, level 18, mana 59968: game 342/256 = 1.3 points, d2d 2.

Full output with every first-per-key case: `/tmp/d2agent-pa-stateevdiff/out.txt` on the worker host (not kept in the repo); the totals above are copied from it.

## Findings

- **Q1 (V run).** Energy Shield's order differs between game.exe and the real d2d build: game.exe takes its share from the raw record field (so the percent and the mana limit work on the unresisted value) and the resistance works on the rest; d2d takes it from the already-resisted total. For a percent-limited shield the two orders agree up to rounding; they diverge when the mana limit binds (mana pays a fixed amount of raw damage in game.exe, a fixed amount of resisted damage in d2d) and at immunity / vulnerability (game.exe pays mana for damage that is then fully resisted, d2d pays nothing at 100 % resistance).
- **D1 (V run order, I arithmetic).** Order as above. 16,120 of 25,038 cases change by the order alone in game.exe's own arithmetic.
- **D2 (V run).** Magic damage: d2d has no percent magic resistance on a monster blow (`monster_blow`, case 4), game.exe applies stat 0x25 in the same pass. Control rows: 382 of 391 differ. Not an ES matter; noted because it contaminates ES-with-magic rows.
- **D3 (V run + S).** Arithmetic: d2d works in whole points (mana floored to points, the cost truncated), game.exe in 256ths (S: `sk-armor-trace.md` D4). Visible even at resistance 0 (magic ES rows: 3,010 of 3,814 differ by one point at most, none by more).
- **Differential for each D:** D1, D3: exists now (these tools; a real FUN_005ca6c0 run to replace the hook is the remaining step, U). D2: the control rows are the differential already (magic resistance != 0).
- **Proposed d2d change (not applied).** In `Fight::absorb` do not apply the ES to the summed `foe.damage`; apply it per blow to the raw element fields before `rules::monster_blow`'s resistance (per field, in 256ths, MulDiv, as `sk-armor-trace.md` Q5), in the order the event nodes run, and let a state at 0 mana end (D3 of that trace). The cost is that the blow must carry raw fields to `Fight::absorb` (`Foe::take` stores only the resisted `damage`).
- **PR candidate: none.** The harness is the existing kind (real d2d functions linked), mismatches before > 0 (18,532 ES cases), but the game side's ES arithmetic is a hook, so the "after the fix 0" end cannot be shown byte-for-byte on arithmetic; the order claim alone is V run. Needs the real `FUN_005ca6c0` first, then the d2-reviewer.

## Bot (our code)

`grep -rn "absorb\|Energy Shield" internal/combat`: the bot has no damage-intake or absorb model. Energy Shield appears only as a buff name to cast (`internal/combat/castgap.go:23`, `classdata/profiles.json`, `class_test.go:80`) and as state 30 in the armor map (`internal/combat/necro.go:40`, noted `sk-armor-trace.md` B2). **Matches game.exe: not applicable (nothing modelled).** Any future bot ES estimate must take the shield from the raw hit before resistances (D1) and bill mana at 256ths.

## Dead ends

- Running `FUN_005ca6c0` natively as the event 11 handler: not done (state-list and calc hooks needed; the dispatcher is a hook in the proven harness). Next step for a PR-grade claim.
- d2d at the exact upstream main commit: the read checkout used is not a git tree; `components/game/fight.cpp` of the main checkout holds the older whole-sum version of `absorb`, `fight/states.cpp` of the checkout used holds the per-blow Bone / Cyclone version; the ES block is identical in both (V read).
- Poison life comparison: not comparable by harness design.

## Drill recipe (lab, U to confirm; verbs of docs/forkdbg.md)

Same as `stateevrun-trace.md` E1, with two resistances: `debug points 58 1`, `debug cast 58 <x> <y>`, `debug stat 8 <256 x 20>` (20 mana points), then take a 100-point fire hit at fire resistance 0 and at 75: game.exe is predicted to spend the same mana both times when the percent limit binds and to leave a 100-point-share mana difference when the mana limit binds; at fire resistance 100 game.exe still spends mana, d2d does not.

## Matrix-cell update (docs/research/sorc-skill-matrix.md, proposed)

3.11 Energy Shield (b): "order differs from game.exe: d2d absorbs on the resisted total; real-build differential 40,000 cases, order alone changes 16,120 of 25,038 ES cases in game.exe's arithmetic [V run order / I arithmetic, stateev-diff-20261104]".

## Shared-file lines for the merger

See `openspec/changes/stateevdiff/merge-lines.txt` if present; otherwise:

- BACKLOG: SKSTATEEV D1 differential built (stateev-diff-20261104.md); next: run the real FUN_005ca6c0 in place of the hook, then d2-reviewer, then a d2d draft.
- INDEX: research/stateev-diff-20261104.md: Energy Shield absorb order, game.exe vs the real d2d build (40,000 cases)
- CHANGELOG: docs: Energy Shield absorb-order differential against the real d2d build (docs/research/stateev-diff-20261104.md, tools/emu/stateev_diff_*): game.exe absorbs before the resistance pass, d2d after; 18,532 of 25,038 ES cases differ.
