# Gate 2 seed 11: why Andariel was never attempted

> status: analysis, no code change. Tags: V = counted from the run log on the worker host (read-only), S = from code or a doc in this tree, I = inference, U = unverified. Run: gate2-s11-20261103-1954-g2a (bot-0.log, 132,574 lines, virtual 19:54 to 03:54 wall).

## 1. Verdict

Not a bot defect. A deliberate policy hold kept her away: `boss.andarielHold` defaults to 1 (S: `internal/env/tuning_boss.go`, ADR 0011 gate 3, spec merc-policy "No unattended attempt before the throwaway fight of card m3-andariel has won once"). The m3-andariel card is `deferred` and has no fight win (S: openspec/changes/m3-andariel/tasks.md). So under the shipped defaults the campaign can never attempt Andariel, at any clvl, and the Gate 2 criterion "Attempt Andariel at clvl 14 or higher" cannot pass on a default run. This is a policy/gate-wording conflict, not something to fix in code (paused policy cards: no policy change made).

## 2. Evidence (V)

- `boss: held for andariel (unattended hold: the throwaway fight has not won yet (boss.andarielHold))` at 20:11:47, clvl 8. It is the only `boss:` line in the log (1 line; the line is logged once, not on every pass). 0 `boss: andariel gate pass`, 0 boss-run start lines.
- `route guard: andariel floor=` 7 lines: 17 (merc=none) at 19:57, 15 (merc=alive) at 20:14:59, 17 (merc=dead) from 20:16.
- clvl 14 reached 21:37:42; from there `level plan` picks Cave 2, Stony Field, Cave Level 1, Crypt (clear:18): the leveling goal, not the campaign boss goal.
- The merc did NOT stay alive (corrects the brief): free rogue claimed 20:14:59, `merc: died` 20:15:50 (clvl 9). It was revived at price 120 only at 21:57 (clvl 14), died again twice within 25 s, then was dead to the end: 366 `merc: revive skipped (gold)` lines, price 1920 vs gold 1762 to 2265 at the tail (V: the last lines, 03:44 to 03:53 wall; the price rose with merc level 16). Only 2 `merc: seen class>0` lines. So the route-guard merc term was `dead` for almost the whole clvl 14 to 16 window.

## 3. Rule chain (S)

`internal/task/gate.go`: `BossGate.baseFloor` returns `max(andarielFloor(w), w.Level+1)` while `andarielHeld(w)` (hold knob on, Normal, Andariel not killed, quest not done). The floor therefore sits one level above the character at every clvl, whatever the merc state. Independent second blocker even with the hold lifted: `andarielFloor` is 17 with no or dead merc (15 with a living merc, `routeMercOK`), and the run ended at clvl 16, with the merc dead (V above). So lifting only the hold would still not have produced an attempt at clvl 14 to 16 in this run; a living merc (floor 15) would have.

## 4. No fix, question for the user

Nothing was changed: the hold and floors are ADR 0011 policy. A fake-based regression test of the hold already exists with that policy (S: gate tests) and would only restate it.

Question: which of these should Gate 2 do? (a) Reword the criterion in soak-gates to "reach the Andariel gate at clvl 14 or higher (a `boss: andariel gate pass` or `boss: held for andariel (clvl N < floor)` line counts as not yet)" or drop it while the hold is on; (b) set `boss.andarielHold 0` for the Gate 2 profile only, accepting an unattended first Andariel fight (ADR 0011 gate 3 says no); (c) run m3-andariel first (needs the live game; currently deferred). Recommended: (c), with (a) as the interim wording. Also a possible defect to look at separately (I, not changed): merc revive is gold-starved at clvl 16 (price 1920) and the campaign has no rule to farm gold for it, which keeps the floor at 17 even after the hold is lifted.

Also a judge note carried over from docs/research/gate2-analysis-20261102.md: the judge counted `planner` "why" strings mentioning Andariel as attempts (166 lines mention her here, V `grep -ci`); this run it correctly failed.

## 5. What a Gate 2 rerun must show

- One of the options above decided first.
- If (b) or after (c): a `boss: andariel gate pass clvl=N kit=ok` line at N >= 14, then a boss-run start line and a boss outcome line (attempt event), with merc alive (floor 15) or clvl >= 17.
- If (a): the judge matches `boss:` / `route guard: andariel` lines, not `planner` text.
- Merc kept alive or revived: `merc: revived` without `revive skipped (gold)` runs at clvl 14+.
- Not verified live; no game was used.
