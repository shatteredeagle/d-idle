# Gate 2 rerun runbook (current main)

> status: draft 2026-11-04, docs only, nothing run. Tags: V = read in a repo file (named), S = secondary (another doc's
> statement), U = unverified. The ops repo (private) is not part of this tree: its commands are named, not quoted.

## 1. Preconditions: merged fixes the rerun depends on

All V by `git log origin/main` (merge commits). Check each is an ancestor of the build under test
(`git merge-base --is-ancestor <hash> HEAD`).

| Fix | Merge | What the rerun relies on |
|---|---|---|
| gate2loop | fdbf1b94 | save point after town chores, and only when XP at risk > 0 (docs/gate2loop.md) |
| gate2cursor | dae56176 | `Hand.DropCursor`: a held item no pack cell takes is dropped (docs/gate2cursor.md); cause B (no-area block) is policy, unchanged |
| mercrevive2 | c7061bdb | gold farm cap clvl 17, farm budget 40 min with 60 min cooldown, full-kit reserve 50 percent |
| farmfallback | 962de9e9 | `farm: true` in plan data (+Dark Wood, Black Marsh); empty restricted pick falls back (docs/farmfallback.md) |
| heldlog | 32350aad | Andariel hold line logs per clvl with floor and merc |
| areametrics | 9709162b | `goldPickup`, `area summary`, kill `class`, `merc: gold farm off reason=`; report table per area (docs/areametrics.md) |
| killcredit | 6e395bb6 | kill credit per cast window and nova targets; per-skill kills in the report |
| logfields | f1d14cb0 | cast result, last skill, armor sample, build line in the logs |
| specdelta (judge) | 4c99d0b9 | belt window counts only when pack plus stash gold reaches 30; judge code and tests |
| hirefix | 94b7a627 | merc seen on d2d (dead merc, ids, revive quote) so hire / alive / revive can be confirmed |

Interim Andariel rule (V, openspec/specs/soak-gates/spec.md, "Gate 2 and Gate 3 meaning"): while `boss.andarielHold` is on,
reaching the gate meets the criterion. The ops judge must be the build that implements the "Held with merc alive, floor
reached" scenario (heldlog made the line meetable). U: whether the ops-side judge change has landed; ask before the run.
Not merged (U): the Act 1 `milestone name= word= clvl= virtual=` line and the act1check section (proposal only, spec
act1-campaign, last paragraph). Until it lands, milestone time is read by hand (section 4).

Other preconditions: Gate 1 passed on the same build (spec "Gate ladder on d2d"; skipping a rung does not count); Gate 0.5
smoke passed on the same build (45 wall min, profile m24, fault at 15 min); no cheated gold or levels; uncheated fresh
character per the brief.

## 2. Commands

In this repo (V, docs/d2d-soak.md section 1 and 3):
- `d2dsoak run -seeds S -hours 8 -out DIR -baselines tools/d2dsoak/baselines.json -clock bot -cmd '<bot --backend d2d --sock $D2DSOAK_SOCK ...>'`
  (needs the D2DBACKEND flags, U). Runner refuses above load 45 or under 15 GB free.
- `tools/d2dsoak/run.sh` builds and ships it to the worker host (V).
- Judge: `d2dsoak compare [-baselines F] [-json] [-holdout 16-23] RUN...`; `d2dsoak report LOG...` (invariants, per-skill,
  per-area table); `d2dsoak bands` (advisory XP/h per clvl band). Exit 0 PASS, 2 INCONCLUSIVE, 3 FAIL/NO-DATA; only 0 is a pass
  (V, docs/d2d-soak.md 3c).
- Process hygiene: start through `tools/agentrun.sh <name> --ttl SECS -- cmd`; end with `tools/agentreap.sh <name>` and
  `tools/agentreap.sh --list` (V, AGENTS.md; spec "Injected fault recovery").
- In the ops repo (not readable here, S): `tools/gate-run.sh` (start and `--collect` build the record branch, V by
  docs/process/baseline-runs.md which says collect works "exactly like `gate-run.sh --collect`"), `tools/mergeq.sh` (queue,
  S by BACKLOG MERGEQ line), `tools/capacity.sh` (hangs on this box, do not use), and `docs/gate-run.md`. Use their documented
  flags there, not guesses from here.

## 3. Seeds and holdout rules

- V (ADR 0015 item 8, soak-gates spec): XP/h is never a per-run pass or fail; per run the gate judges stability,
  invariants, funded belt windows. XP/h is judged on a seed set of at least 5 seeds, 2 of them holdout.
- V: holdout seeds are 16 to 23; never used for tuning, fitting or diagnosis. Do not read a holdout log to fix anything; if a
  holdout fails, report it and diagnose on a tuning seed.
- V: goal G-C needs Gate 2 PASS on 3 seeds incl. 1 holdout, then Gate 3 on 1 seed.
- Suggested set (U, a choice): seed 13 again (same settings, no NOKILL; the docs/gate2loop.md "what a live rerun must show"
  list), seed 12 (the other past run), and one holdout from 16 to 23 that has not been diagnosed. Seed 11 single reruns are
  dropped (ADR 0015 item 8).

## 4. Evidence per criterion

| Criterion (soak-gates spec) | Proof |
|---|---|
| Reach clvl 16 | last `minute` line `clvl` >= 16 (V) |
| Merc hired, alive, gold moving | hire line (hirefix judge matches it), `merc:` lines; `gold`/`goldPickup` in minute lines; `merc: gold farm off reason=...` |
| Andariel at clvl 14+ (hold on) | `route guard: andariel floor=F ...` lines with merc state, and `boss: held for andariel (<why>) clvl=N floor=F merc=<state>` at N >= F-1 and >= 14 (V, internal/task/gate.go; 17 when merc dead: criterion not met unless clvl 17 and the gate line). A planner "why" string alone is not enough. After the hold lifts: boss-run start and outcome lines at clvl >= 14 |
| Milestone times (G-B, not a gate) | until act1check lands: first `Entered` lines per area, quest lines and `boss:` lines with virtual time; U after landing: `milestone name= word= clvl= virtual=` |
| XP not stalled | no 10+ virtual minute XP flat (idle/pause exempt); m24 avg XP >= 1000/h is a spec floor (V) |
| No-gain window <= 90 min | no clvl, quest step, gold gain (or +500 XP) gap longer |
| Bot health | down <= 3 min at a time |
| Belt | `d2dsoak report` belt window <= 5 virtual min when funded; poverty windows listed, not failing |
| Invariants | report ends `INVARIANTS: PASS` (HANG, STALL, WEDGE, PROGRESS, DEATHS, RESTARTS, EXIT) |
| Deaths | death-rule spec, `d2dsoak report` |
| Resource slopes | `mem.csv`: fd <= 5/h, RSS <= 40 MB/h, threads <= 5/h after 10 min; < 30 samples NO-DATA |
| Stability | no unplanned d2d exit; `planned recycle` line before any exit; restarts = injected only |
| Save semantics | save point lines >= 20 virtual min apart, each with `xp at risk` atRisk > 0; `chores done` or `shop: visit` before a save point that follows `no town portal: walking back to town` (docs/gate2loop.md) |
| Full bag | `invPct` > 95 outside town for <= 60 consecutive lines |

New diagnostics to read (V, docs): per-area XP/h and gold/h table (areametrics); per-skill kills (killcredit);
`farm: restricted pick empty, falling back` at most once per game (docs/farmfallback.md); `merc: gold farm budget minutes=40
cooldown=60 ...` lines (internal/merc/logs.go).

## 5. Past failures: signatures and early abort

Gate 2 seed 13 (V, docs/gate2loop.md, docs/gate2cursor.md; run NOT PASSED at 258 virtual h, clvl 16):
- Save-point loop: `save point: saving and re-entering the game` 20 to 40 wall s apart (9 s later), `atRisk: 0`, sequence
  `Recovery: farming gold` -> `no town portal: walking back to town` -> `Entered Rogue Encampment` -> save point 6 ms later,
  with no chores line. 789 such lines in 176k log lines. Should be impossible now (gate2loop).
- Cursor fault: `Strangling Gas Potion was worn on pickup and could not be taken off: hand: no free space in target`, then
  `item stuck on the cursor: hand: no free space in target` on every loot call, gold stuck (26), failsafe ladder steps 1 to 5,
  `Failsafe: paused`. Now expect a drop instead; any `item stuck on the cursor` line = fix did not hold.
- No-area block: `no eligible reachable leveling area for level 15` (7683 lines), first at clvl 15, bans such as
  `banned after repeated deaths`, route guards (Tower Cellar needs merc and clvl 16). Expect the farmfallback line, not a flood;
  a WEDGE (> 15 virtual min of these with no non-town Entered) is a violation.
- Merc revive stall: count of `merc: revive skipped (gold) price=P gold=G need=N` lines (V, internal/merc/logs.go). Seed 13
  figures are in the s13 record (docs/research, branch gate2run-s13-*; number U here). Many consecutive skips with gold flat =
  farm not producing. Expect `merc: gold farm` on/off/budget lines instead.
- Seed 12 and Gate 1 BELT at clvl 11 (docs/research/gate2-analysis-20261102.md): healing 0 with gold 33; kit trip fired only
  at 0 potions.

Early-abort rule (S/U, proposed here; the spec has no abort rule, so abort only to save time, record why, and never count the
run). Virtual time from the minute lines, not wall:
- 30 min: character alive, `Entered` lines past town, XP rising, no `item stuck on the cursor`, no `campaign stalled`. Else abort.
- 60 min: injected fault (if planned) recovered with exactly one restart; first save point has `xp at risk` > 0; belt window
  <= 5; no HANG.
- 120 min: Gate 1 criteria hold so far (deaths, belt, no-kill <= 10); clvl is following the earlier Gate 1 pace (clvl 8+ is
  U); no save point with atRisk 0; `mem.csv` samples present. Abort on any WEDGE storm or 3+ `revive skipped (gold)` with no
  farm lines.
- Afterwards: check every 2 virtual hours for the same list; the s13 stall began about 6 h in.

## 6. Capacity and start checks

- Load: `labrun <worker> -- top -bn2 -d1` on the worker host: idle plus nice >= 25 percent (V, AGENTS rules; `tools/capacity.sh`
  hangs here). Start guard of the spec: free disk >= 10 GB, 1 min load <= 28 (V; d2dsoak runner itself refuses above 45).
- `tools/agentreap.sh --list` shows nothing left of earlier runs; a hidden d2d slot is free; no other d2d lock
  (`/tmp/d2d-lab.lock` is how the runner refuses, V docs/d2d-soak.md).
- A Gate 2 is a big run; one at a time per the lab rules.

## 7. Post-run steps

1. Collect: `gate-run.sh --collect` in the ops repo (builds the record branch with record, minute logs, `merge-lines.txt`;
   S via docs/process/baseline-runs.md). Record the branch name in the report.
2. Run `d2dsoak report` on every log and keep the output in the record; for holdout seeds add no diagnosis.
3. Thin any log over 256 KB (the big-file guard; docs/fixbigfiles.md, U for the exact tool) and write the note in the record
   that the tail was thinned, as done in the s13 record (commit 3a74621c).
4. `go run ./cmd/leakcheck -mode fail` 0 hits (no host names, IPs, home paths, names); `go run ./cmd/changecheck status -write`
   then `go run ./cmd/changecheck -cli` 0 warnings.
5. Queue with `tools/mergeq.sh` (ops side); do not edit BACKLOG, INDEX or CHANGELOG, put lines in `merge-lines.txt`.
6. `tools/agentreap.sh <name>` and `tools/agentreap.sh --list` empty; `tools/ci/check.sh`.

## 8. Go / no-go checklist

Go only if all yes:
- [ ] Build contains the ten merges of section 1 (ancestor check) and `go vet`/short tests green via labrun on the worker host.
- [ ] Gate 1 and Gate 0.5 passed on this build; judge has the interim Andariel rule (U, confirm with the ops side).
- [ ] Seed set chosen: >= 1 holdout from 16 to 23, no diagnosis planned on it; seed 13 included for comparison.
- [ ] Character fresh, uncheated; cheat ledger empty for it.
- [ ] Capacity: idle+nice >= 25 percent, disk >= 15 GB, load < 28.
- [ ] Early-abort checklist (section 5) printed next to the run; someone assigned to the 30/60/120 min looks.
- [ ] Post-run owner and record branch name known.
No-go if any: an item above is unchecked, a holdout log was already read, or the build changed after Gate 1.
