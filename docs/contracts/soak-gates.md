---
id: soak-gates
goal: G-C
status: in-force
enforced_by:
  - cmd/d2soak
  - internal/d2dsoakstat/invariants.go
  - cmd/d2dsoak
judged_by_ops: tools/gate-judge.py
adr: [0009, 0013, 0015]
---
# Soak gates contract

## Purpose
Numeric pass criteria for the d2d gate ladder (Gate 1 2 h, Gate 2 8 h, Gate 3 24 h; the old G1/G2 names are retired, see docs/decisions.md Current truth) and the health guard that decides when a soak may start, so a run is judged by numbers and not by opinion. Evidence: docs/review-strategy.md (G1 criteria), docs/soak24.md section 3 (profile m24), docs/g1-log.md (attempt log), docs/plan/plan-v2.md section 5.

> status: reconciled 2026-11-04 against docs/process/spec-ledger-20261104.md; pending deltas: d2d-container-soak (superseded by ADR 0015, container deferred; stays on its card, not folded)

## Requirements

### Requirement: Gate 2 and Gate 3 meaning and floors
Gates 2 and 3 SHALL judge stability and economy (goal G-C); progression to Andariel dead and Act 2 is goal G-B (milestone time per seed), not a pass condition. Gate 2 SHALL reach clvl 16, meet the Andariel criterion at clvl 14+, hire the merc and keep gold moving; while boss.andarielHold is on, reaching the Andariel gate meets it, after the lift an attempt is required. Source: plan-v2 s5, s10; ADR 0013. Reconciled 2026-11-04 (source: 2026-11-04-andarielspec, decision 6).

#### Scenario: Cheated run
- **WHEN** a run used cheated gold or cheated levels
- **THEN** it does not count toward any gate

#### Scenario: Interim evidence
- **WHEN** boss.andarielHold is set
- **THEN** the criterion needs a `boss: andariel gate pass` or `boss: held for andariel (...)` line naming the hold, plus `route guard: andariel floor=` lines showing floor and merc state

#### Scenario: Held with merc alive, floor reached
- **WHEN** boss.andarielHold is set, the merc is alive, and the log has `route guard: andariel floor=` lines and a `boss: held for andariel (...)` line naming the hold at clvl >= floor-1 and >= 14
- **THEN** the Andariel criterion is met

#### Scenario: Held with merc dead
- **WHEN** boss.andarielHold is set and the merc is dead so the floor is 17, and the log shows the `route guard: andariel floor=17` lines with the merc state
- **THEN** the evidence shows why the gate was not passed, and the criterion is not met unless the character reached clvl 17 and the gate line

#### Scenario: Planner text only
- **WHEN** the only Andariel mention is a planner "why" string, with no `boss: andariel gate pass` or `boss: held for andariel (...)` line
- **THEN** the criterion is not met

#### Scenario: Hold lifted
- **WHEN** boss.andarielHold is lifted after m3-andariel has won once
- **THEN** only a boss-run start line and an outcome line at clvl 14 or higher meet the criterion

### Requirement: Soak progress floors
The m24 profile MUST fail a run whose XP does not change for more than 10 minutes (idle and paused exempt), whose no-gain window (no clvl, quest step or gold gain) exceeds 90 minutes, or whose bot health is down for more than 3 minutes at a time. Source: docs/soak24.md section 3 (U until the first 24 h run calibrates it). XP/h is advisory; the judge reports it without failing a run (ops gate-judge change, 2026-11-04).

#### Scenario: Idling bot
- **WHEN** no XP change is seen for 11 minutes outside pause
- **THEN** d2soak reports a stall FAIL

> OPEN QUESTION: code (cmd/d2soak/progress.go) also counts an XP gain of 500 or more as progress for the 90 minute no-gain window; the floors text lists only clvl, quest step and gold. No decision covers it; text left as is.

### Requirement: Resource slopes
A soak MUST keep fd growth (helper, game, bot) at or below 5 per hour after a 10 minute warm-up, RSS at or below 40 MB per hour, threads and goroutines at or below 5 per hour, rungs and sentinel actions at or below 2 per hour. On d2d the runner SHALL sample fd, thread and RSS of d2d and bot each wall minute into `mem.csv`; a series over its slope limit FAILS, under 30 samples is NO-DATA. Source: docs/soak24.md s3; gate3gaps.md s1. Reconciled 2026-11-04 (source: 2026-11-03-gate3gaps).

#### Scenario: Leaking helper
- **WHEN** the fd slope exceeds 5 per hour after warm-up
- **THEN** the run fails the fd criterion

#### Scenario: d2d fd leak on the gate runner
- **WHEN** `mem.csv` shows the d2d fd count rising from 400 to 470 over 8 h after warm-up
- **THEN** the slope is about 9 per hour and the run FAILS the d2d fd criterion

#### Scenario: Bot RSS slope within the limit
- **WHEN** the bot RSS rises 120 MB over 8 h after warm-up
- **THEN** the bot RSS row passes at 15 MB per hour

#### Scenario: No samples for a series
- **WHEN** `mem.csv` lacks the bot columns
- **THEN** the bot rows read NO-DATA and a Gate 3 run is not a pass

#### Scenario: Restart inside the series
- **WHEN** the d2d pid changes
- **THEN** each segment is fitted after its own 10 minute warm-up and the worst slope counts

### Requirement: Heap runaway alert
The soak harness SHALL raise a critical alert when the bot RSS exceeds 8192 MB or rises by more than 1024 MB within 5 minutes. Source: docs/soak24.md section 4 (V offline, live U); attempt 16 grew to 48.8 GB before the fix.

#### Scenario: Runaway
- **WHEN** the bot RSS jumps 1.2 GB in 4 minutes
- **THEN** a `heap_runaway` critical alert is written

### Requirement: Start guard
A soak start MUST be refused when free disk is under 10 GB, when the 1-minute load exceeds the backstop 28, or when `tools/capacity.sh --add <soak cost>` prints `NO`; there is no fixed slot cap. A multi-hour soak is also a big run (lab-rules spec). Source: docs/soak24.md section 2; docs/process/worker-capacity.md (code change to soak24.sh is a follow-up, S).

#### Scenario: Busy host
- **WHEN** the predicted idle+nice after adding the soak is under 25 %
- **THEN** `soak24.sh start` refuses with the capacity.sh output

#### Scenario: Many instances, quiet host
- **WHEN** three slots already run but the predicted idle+nice stays at or above 25 % with iowait under 40 %
- **THEN** the start is allowed

### Requirement: Verdict
A verdict MUST be PASS only when `report.txt` says PASS with exit 0; NO-DATA (exit 3) is never a pass, and the death criteria follow the death-rule spec. Source: docs/soak24.md section 3.

#### Scenario: No data
- **WHEN** a run produced no samples
- **THEN** the verdict is NO-DATA, exit 3, not a pass

### Requirement: Per-seed invariants
`d2dsoak` SHALL report for every seed counters and a verdict for HANG (a pace interval without virtual advance, or `combat no progress, holding`), STALL (any `campaign stalled` rung), WEDGE (one `no route from level` or `no eligible reachable leveling area` storm longer than 15 virtual min with no non-town Entered), PROGRESS (no kill, xp, area, quest or spend for more than 30 min), DEATHS, RESTARTS and EXIT. Source: card gen-invariants; docs/geninv.md.

#### Scenario: Town wedge
- **WHEN** a log has 60 virtual minutes of `no eligible reachable leveling area` lines and no non-town Entered
- **THEN** the seed is VIOLATED with WEDGE and PROGRESS even though the process exited 0 at the time limit

#### Scenario: Frozen clock
- **WHEN** two pace lines are wall time apart and show the same virtual time
- **THEN** the seed is VIOLATED with HANG

### Requirement: Death and exit invariants
A seed SHALL violate DEATHS on more than 15 deaths per 1000 kills from 150 kills, on 3 deaths in 30 min without a clvl gain or 5 in 60 min, or on a death with no progress within 10 virtual min; RESTARTS on any bot restart; EXIT on an end under 95 percent of the requested hours, a non-zero exit or a recovered panic. Source: death-rule FAIL-1 and FAIL-3 (D); soakfix early end 95 percent (D); recovery window (J).

#### Scenario: Early end
- **WHEN** 3 virtual hours were requested and the command exited after 1
- **THEN** the seed is VIOLATED with EXIT; a log with no end marker and no requested length is `open`, not violated

### Requirement: Invariants verdict
The report SHALL end its invariants block with `INVARIANTS: PASS`, `VIOLATED(list)` or `NO-DATA`, separate from the fidelity verdict; the set is VIOLATED when any seed is, PASS only when every seed is PASS. The invariants verdict MUST NOT change the exit code of `run` or `compare`. Source: card gen-invariants; fidelity-gate Verdict exit codes.

#### Scenario: Violated but fidelity PASS
- **WHEN** the fidelity verdict is PASS and one seed has a WEDGE
- **THEN** the exit code is 0, the text and `-json` show `INVARIANTS: VIOLATED(seed N: WEDGE)` and OVERALL stays the last line

### Requirement: Holdout seeds
Seeds 16 to 23 SHALL be holdout seeds, never used for tuning, fitting or diagnosis; the report groups seeds as `tuning` and `holdout` (`-holdout` overrides, empty disables), gives each group its counters and invariants verdict, and prints a COVERAGE line naming seeds, class, act, difficulty, engine and clvl range. Source: docs/research/bot-generality.md sections 3 and 5 (experiment 1); docs/geninv.md.

#### Scenario: Gate run with holdout
- **WHEN** seeds 3 and 16 to 17 are compared
- **THEN** group tuning lists seed 3, group holdout lists seeds 16 to 17, each with its own verdict, and the COVERAGE line lists both seed sets

#### Scenario: No holdout ran
- **WHEN** only seeds 1 to 8 ran
- **THEN** the report says `group holdout: none ran` and the COVERAGE line shows holdout none

### Requirement: Gate ladder on d2d
Release gates SHALL be a ladder on a hidden d2d slot, each rung started only after the one below passed on the same build: Gate 0 the live drill suite (every S and A primitive of docs/primitives.md passes in `cmd/d2drill`), Gate 1 a 2 h unattended soak with one injected fault, Gate 2 an 8 h soak (profile m24), Gate 3 the 24 h soak (profile m24). Cheated gold or levels never count. Source: ADR 0013; docs/review-strategy.md section 3 (d2d gate ladder); docs/probes/drills/README.md.

#### Scenario: Skipping a rung
- **WHEN** Gate 1 has not passed on a build
- **THEN** a Gate 2 run on that build does not count as a gate

#### Scenario: Drill failing
- **WHEN** one S or A primitive drill is FAIL or BLOCKED
- **THEN** Gate 0 is not passed and no Gate 1 soak starts

### Requirement: Gate 1 stability and save semantics
A Gate 1 run SHALL show zero unplanned process exits other than the injected fault, zero panics and zero manual interventions, the bot and its d2d alive at 2 h, and after a graceful stop a saved XP equal to the character XP. Source: translated from docs/review-strategy.md G1 criterion 1 (S).

#### Scenario: Clean two hour run
- **WHEN** a Gate 1 run ends at 2 h
- **THEN** the report shows 0 panics, no exit other than the injected one, and `saveXp == charXp` after the graceful stop

### Requirement: Gate 1 deaths and progression
A Gate 1 run MUST follow the death rule (death-rule spec, the only authority on deaths; `d2dsoak report`) and no state may last more than 10 virtual minutes without a kill or XP gain. XP per hour is NEVER a per-run pass or fail (ADR 0015 item 8; XP/h verdicts come from a seed set); the band requirements below are ADVISORY only. Source: G1 criteria 2 and 3; ADR 0009 item 2. Reconciled 2026-11-04 (source: 2026-11-03-specdelta, 2026-11-01-baselinefree, user decision 2).

#### Scenario: XP below the d2d baseline
- **WHEN** true XP/h of one run is under the d2d baseline
- **THEN** the run is not failed on that account; the band rows show it as ADVISORY and the seed-set verdict decides any claim

#### Scenario: Long silent stretch
- **WHEN** a Gate 1 log has 11 or more consecutive minute lines with no new kill and no new XP
- **THEN** the run is a FAIL on the progression criterion

### Requirement: Gate 1 XP per clvl band (advisory)
XP/h SHALL be computed per clvl band (1-4, 5-7, 8-10, 11-14, 15-20, inclusive, tiling the run from the first minute line in the band to the first above it) and shown against the mean of the baseline runs with at least 0.5 h in that band (`d2dsoak bands`); the rows are ADVISORY and never decide Gate 1. Source: docs/research/baseline-audit.md s4b, s6. Reconciled 2026-11-04 (source: 2026-11-01-baselinefree, user decision 2).

#### Scenario: Fresh run crossing the bands
- **WHEN** a fresh run reaches clvl 14
- **THEN** the report lists one advisory XP/h row per band it entered, each against its own baseline

#### Scenario: Band entered briefly
- **WHEN** the run spent less than 0.5 h in a band
- **THEN** that band's row is marked as having too little exposure whatever its value

#### Scenario: Whole-run mean
- **WHEN** a run crosses several bands
- **THEN** its whole-run XP/h is shown as information only and is never compared with one baseline mean

### Requirement: Band baseline sample (advisory)
A band row SHALL be marked "thin baseline" when fewer than 3 baseline runs have at least 0.5 h in the band. Source: docs/research/baseline-audit.md section 3 (the interval of a 2-run mean is wider than the mean); `d2dsoak bands`. Reconciled 2026-11-04 (source: 2026-11-01-baselinefree, user decision 2).

#### Scenario: Band with one baseline run
- **WHEN** the baseline holds one run in clvl 5-7
- **THEN** the clvl 5-7 row is marked thin baseline

### Requirement: Band tolerance (advisory)
With 3 or more baseline runs in the band, a row SHALL be flagged "below band" only when XP/h is below the band mean by more than 1.96 x sqrt(CV^2 / k + CV^2 / n) of the mean (k d2d runs pooled, 1 for Gate 1; n baseline runs; CV the larger of the band's CV and 0.14); a flag fails nothing. Source: docs/research/baseline-audit.md s3. Reconciled 2026-11-04 (source: 2026-11-01-baselinefree, user decision 2).

#### Scenario: Band with four baseline runs
- **WHEN** four baseline runs have at least 0.5 h in clvl 11-14 and a Gate 1 run's XP/h there is below the mean by more than the tolerance
- **THEN** the clvl 11-14 row is flagged below band and Gate 1 is not failed by it

#### Scenario: Faster than the baseline
- **WHEN** the run's XP/h in a band is above the band mean
- **THEN** the row is not flagged whatever the size of the gap

### Requirement: Gate 1 belt and portal
A Gate 1 run SHALL have no funded window over 5 virtual minutes with zero healing potions outside town (packHP, packMP, beltBoxes) and at least one town-portal trip, not the waypoint fallback. A window is funded when `gold` + `stashGold` reaches one Minor Healing Potion (30 gold) at some minute; others are poverty, not failing; no `stashGold` means 0. Source: G1 criterion 4; docs/beltfill.md; user decisions 2026-11-01, 2026-11-03. Reconciled 2026-11-04 (source: 2026-11-03-specdelta).

#### Scenario: Empty field window
- **WHEN** no healing potion is held outside town for over 5 minutes and pack plus stash gold reaches 30 at some minute of the window (for example pack 6, stash 1418)
- **THEN** the run is a FAIL

#### Scenario: Poverty window
- **WHEN** no healing potion is held outside town for over 5 minutes and pack plus stash gold stays under 30 for the whole window
- **THEN** the window is reported as poverty and does not fail the run

#### Scenario: Gold crosses the price mid-window
- **WHEN** pack plus stash gold is under 30 at first and reaches 30 later in the same window
- **THEN** the window is funded and the run is a FAIL

#### Scenario: Old archive without stashGold
- **WHEN** the minute lines carry no `stashGold`
- **THEN** the stash counts as 0: a window with pack gold of 30 or more fails, one with less is poverty, and the report tag stays "stash unknown (old archive)"

#### Scenario: Waypoint only
- **WHEN** every town return in the run used the waypoint fallback
- **THEN** the run is a FAIL on the portal criterion

### Requirement: Gate 1 route guard respected
A Gate 1 run MUST start no boss run while the route guard or the boss gate refuses (log `blocked: not ready`, `route guard:` floors of the merc-policy spec) and SHALL show fewer than 1 boss-run start per 30 s. Source: translated from G1 criterion 5.

#### Scenario: Refused boss
- **WHEN** the log shows `blocked: not ready` for a boss and a run for that boss starts afterwards without the gate opening
- **THEN** the run is a FAIL

### Requirement: Injected fault recovery on d2d
At about 60 minutes of Gate 1 exactly one fault SHALL be injected: the d2d process of that slot only is killed through the process registry (`tools/agentreap.sh`, never `pkill`, never a pid not started by the run). The supervisor MUST restart it within 3 minutes with exactly one restart counted, the character saved, and no process of the slot orphaned afterwards (`tools/agentreap.sh --list` and `--orphans`). Source: translated from G1 criteria 6 and 7; ADR 0013.

#### Scenario: Injected kill
- **WHEN** the slot's d2d is killed at about 60 minutes
- **THEN** the bot is back in game in at most 3 minutes, `restarts` rises by exactly 1, the saved XP does not drop below its value before the kill and no process of the slot is orphaned

#### Scenario: Double restart
- **WHEN** the kill causes 2 restarts
- **THEN** the run is a FAIL on this criterion

### Requirement: Gate 2 and Gate 3 on d2d
Gates 2 (8 h) and 3 (24 h) SHALL be m24 soaks on a hidden d2d slot, judged by the Gate 1 criteria over their length, the m24 limits, resource slopes and `d2dsoak report` ending `INVARIANTS: PASS`. Gate 3 also records memory curve, restart interval and log size. A d2d exit is planned only as an injected fault or after a `planned recycle` supervisor line; others fail stability. Source: docs/review-strategy.md; gate3gaps.md s2. Reconciled 2026-11-04 (source: 2026-11-03-gate3gaps).

#### Scenario: Gate 2 with a violated invariant
- **WHEN** the Gate 2 report ends with `INVARIANTS: VIOLATED(...)`
- **THEN** the run is not a pass even if the m24 limits hold

#### Scenario: Planned recycle
- **WHEN** supervisor.log shows `planned recycle` before a d2d exit and the bot is in game within 3 wall minutes
- **THEN** the exit is not unplanned and the interval is recorded

#### Scenario: Unplanned d2d exit
- **WHEN** d2d exits with no injected fault and no planned recycle
- **THEN** the stability row FAILS

### Requirement: Full bag and disk guard
A d2d soak SHALL FAIL when `invPct` stays above 95 outside town for more than 60 consecutive minute lines (-1 is unknown and never counts, U threshold). The runner MUST stop with an alert and the verdict NO-DATA when free disk falls under 5 GB, and total logs of a 24 h run SHOULD stay under 20 GB (U). The thresholds marked U stay U until a Gate 3 run calibrates them. Source: docs/research/gate3gaps.md sections 2.3 and 2.4. Reconciled 2026-11-04 (source: 2026-11-03-gate3gaps).

#### Scenario: Full bag
- **WHEN** 61 consecutive minute lines outside town show `invPct` above 95
- **THEN** the inventory criterion FAILS

#### Scenario: Disk falling
- **WHEN** free disk drops under 5 GB at hour 15
- **THEN** the runner stops with an alert and the verdict is NO-DATA

### Requirement: Gate 0.5 smoke
Before Gate 2 or Gate 3 a smoke run SHALL pass on the same build: profile m24, 45 wall minutes on a hidden d2d slot (about 9 virtual hours), one injected fault at 15 wall minutes (U thresholds). It is judged by BELT, NOKILL, PROGRESS, WEDGE, HANG, the death rule, fault recovery and the presence of resource samples, not by XP floors or slopes, and never counts as a ladder gate. Source: docs/research/gate3gaps.md section 3 (U thresholds). Reconciled 2026-11-04 (source: 2026-11-03-gate3gaps).

#### Scenario: Belt failure caught early
- **WHEN** the smoke run shows 6 minute lines outside town without a healing potion
- **THEN** Gate 0.5 FAILS and no Gate 2 starts on that build

#### Scenario: Slow clock
- **WHEN** the run reached only 4 virtual hours in 45 wall minutes
- **THEN** the verdict is NO-DATA, not a pass

#### Scenario: Smoke passes
- **WHEN** all criteria of the subset hold
- **THEN** the build may start Gate 2, which must still pass on its own

### Requirement: A game-lost recovery counts as a restart
When the bot restarts the game after the game process was lost (a crash, a kill, a hang), the minute line's `restarts` SHALL rise by exactly one at the successful restart, on every backend. A restart already counted through the menu observer or as a bot-started new game SHALL NOT be counted a second time.

#### Scenario: d2d process killed
- **WHEN** the slot's d2d is killed under the managed bot and the bot restarts it and re-enters a game
- **THEN** `restarts` on the next minute lines is one higher than before the kill

#### Scenario: Game.exe recovery
- **WHEN** the same recovery runs on Game.exe and the menu observer sees the new game
- **THEN** `restarts` rises by one, not two

### Requirement: xpRisk is XP since the last save on d2d
The minute line's `xpRisk` and the xp-at-risk warnings SHALL be the experience in memory minus the experience stored in the character's save file, on d2d as on Game.exe; when the save file cannot be read the fallback is the experience at the last game entry or restart. The save file SHALL be re-read when it changes, so a Save and Exit brings `xpRisk` to 0.

#### Scenario: Unsaved gain
- **WHEN** the save file holds 1000 XP and the character has 1500
- **THEN** `xpRisk` is 500, and 0 after a save rewrote the file to 1500

#### Scenario: Kill and re-entry
- **WHEN** the d2d process is killed with 300 unsaved XP and the character reloads from the save
- **THEN** `xpRisk` is 0 again, and the lost 300 is not counted as at risk

### Requirement: Gate 1 minutes are virtual minutes
Every minute of the Gate 1 criteria (and of Gates 2 and 3 over their length) SHALL be a virtual (game) minute, not a wall minute, because the lockstep d2d clock runs about 10 times faster. One `minute` log line is one virtual minute; `d2dsoak` MUST print the longest belt window and no kill and no XP stretch with their limits (5 and 10). Source: user decision 2026-11-01; docs/probes/postfix-soak2.md.

#### Scenario: Belt window short in wall time, long in virtual time
- **WHEN** no healing potion is in the pack or any belt box outside town for 27 consecutive minute lines that took 87 s of wall time
- **THEN** the belt criterion FAILS (27 virtual minutes is over 5) and `d2dsoak` reports the violation code BELT

#### Scenario: Belt window at the limit
- **WHEN** the empty window outside town is exactly 5 consecutive minute lines
- **THEN** the belt criterion passes

#### Scenario: No kill and no XP in virtual time
- **WHEN** 26 consecutive minute lines show neither a new kill nor new XP, although they took 2.5 wall minutes
- **THEN** the no-progress criterion FAILS (26 virtual minutes is over 10) and `d2dsoak` reports NOKILL

#### Scenario: Silent stretch at the limit
- **WHEN** exactly 10 consecutive minute lines show neither a new kill nor new XP
- **THEN** the no-progress criterion passes

#### Scenario: Wall limits unchanged
- **WHEN** the injected d2d kill is restarted
- **THEN** the 3 minute restart limit is read in wall minutes, as are the start guards and the m24 monitors

#### Scenario: Wider invariant unchanged
- **WHEN** the same log has area changes, quest steps or gold spent inside that 26 minute stretch so the longest stretch without any progress event is 16 virtual minutes
- **THEN** the PROGRESS invariant (limit 30) still passes and the Gate 1 NOKILL criterion still FAILS

## Numbers

Owner of these values is this contract (the user approves them); the tests named in `enforced in` (`contract_test.go` in `cmd/d2soak` and `internal/d2dsoakstat`) read this table and compare it with the code. Change a row only with the user's go, cited in the `Source:` of the requirement that states it, and change the code in the same branch.

| key | value | unit | enforced in |
|---|---|---|---|
| gate1.beltMin | 5 | virtual min | internal/d2dsoakstat DefaultInvLimits.BeltMin |
| gate1.noKillMin | 10 | virtual min | internal/d2dsoakstat DefaultInvLimits.NoKillMin |
| inv.wedgeMin | 15 | virtual min | internal/d2dsoakstat DefaultInvLimits.WedgeMin |
| inv.progressMin | 30 | min | internal/d2dsoakstat DefaultInvLimits.ProgressMin |
| inv.stallWarnMin | 30 | min | internal/d2dsoakstat DefaultInvLimits.StallWarnMin |
| inv.recoverMin | 10 | virtual min | internal/d2dsoakstat DefaultInvLimits.RecoverMin |
| inv.earlyFrac | 0.95 | share of requested hours | internal/d2dsoakstat DefaultInvLimits.EarlyFrac |
| m24.stallMin | 10 | min | cmd/d2soak profile m24 Limits.MaxNoProgress; DefaultAlertRules.StallXP |
| m24.noGainMin | 90 | min | cmd/d2soak DefaultProgressLimits.NoGain (profile m24 keeps it) |
| m24.unhealthyMin | 3 | min | cmd/d2soak DefaultLimits.MaxUnhealthy |
| m24.warmUpMin | 10 | min | cmd/d2soak DefaultLimits.WarmUp |
| m24.rssMBPerHour | 40 | MB per hour | cmd/d2soak profile m24 Limits.MaxRSSMBPerHour |
| m24.fdsPerHour | 5 | fds per hour | cmd/d2soak profile m24 Limits.MaxFDsPerHour |
| m24.threadsPerHour | 5 | threads per hour | cmd/d2soak profile m24 Limits.MaxThreadsPerHour |
| m24.goroutinesPerHour | 5 | goroutines per hour | cmd/d2soak profile m24 Limits.MaxGoroutinesPerH |
| m24.ladderPerHour | 2 | rungs per hour | cmd/d2soak profile m24 Limits.MaxLadderPerHour |
| m24.sentinelPerHour | 2 | actions per hour | cmd/d2soak profile m24 Limits.MaxSentinelPerHr |
| heap.runawayMB | 8192 | MB | cmd/d2soak DefaultAlertRules.HeapRunawayMB |
| heap.growthMB | 1024 | MB | cmd/d2soak DefaultAlertRules.HeapGrowthMB |
| heap.growthWindowMin | 5 | min | cmd/d2soak DefaultAlertRules.HeapGrowthWindow |

## Pending deltas (not in force)

- d2d-container-soak (open, proposed): ADDED "d2d container one-slot 24 h soak". Superseded by ADR 0015 item 5 (container deferred until Gate A passes); stays on its card.
