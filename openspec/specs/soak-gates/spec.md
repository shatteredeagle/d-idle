# Soak gates specification

## Purpose
Numeric pass criteria for the G1 (2 h), G2 (8 h) and 24 h soaks and the health guard that decides when a soak may start, so a run is judged by numbers and not by opinion. Evidence: docs/review-strategy.md (G1 criteria), docs/soak24.md section 3 (profile m24), docs/g1-log.md (attempt log), docs/plan/plan-v2.md section 5.

## Requirements

### Requirement: G2 meaning and floors
The 8 h profile m24 run (Gate 2 on d2d, 24 h Gate 3) SHALL be called a stability and economy soak, not a progression soak, until Andariel and Act 2 work; it SHALL reach clvl 16, attempt Andariel at clvl 14 or higher, hire the merc and keep gold from staying flat. Source: docs/plan/plan-v2.md section 5 and section 10 (approved 2026-10-30); ADR 0013.

#### Scenario: Cheated run
- **WHEN** a run used cheated gold or cheated levels
- **THEN** it does not count toward any gate

### Requirement: Soak progress floors
The m24 profile MUST fail a run whose average XP is below 1000 per hour, whose XP does not change for more than 10 minutes (idle and paused exempt), whose no-gain window (no clvl, quest step or gold gain) exceeds 90 minutes, or whose bot health is down for more than 3 minutes at a time. Source: docs/soak24.md section 3 (U until the first 24 h run calibrates it).

#### Scenario: Idling bot
- **WHEN** no XP change is seen for 11 minutes outside pause
- **THEN** d2soak reports a stall FAIL

### Requirement: Resource slopes
A soak MUST keep helper, game and bot fd growth at or below 5 per hour after a 10 minute warm-up, RSS growth at or below 40 MB per hour, threads and goroutines at or below 5 per hour, and ladder rungs and sentinel actions at or below 2 per hour each. Source: docs/soak24.md section 3.

#### Scenario: Leaking helper
- **WHEN** the fd slope exceeds 5 per hour after warm-up
- **THEN** the run fails the fd criterion

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
A Gate 1 run MUST follow the death rule (death-rule spec, `d2dsoak report`) and SHALL earn true XP per hour at or above the d2d baseline at the same clvl (Early baseline fields of the fidelity-gate spec, numbers not copied here). No single state may last more than 10 virtual minutes without a kill or XP gain. Source: G1 criteria 2 and 3; ADR 0009 item 2; user decision 2026-11-01.

#### Scenario: XP below the d2d baseline
- **WHEN** true XP/h is under the d2d baseline at the same clvl
- **THEN** the run is a FAIL even with no crash

#### Scenario: Long silent stretch
- **WHEN** a Gate 1 log has 11 or more consecutive minute lines with no new kill and no new XP
- **THEN** the run is a FAIL on the progression criterion

### Requirement: Gate 1 belt and portal
A Gate 1 run SHALL have no window longer than 5 virtual minutes with zero healing potions outside town, measured from the minute-line fields packHP and beltBoxes (pack plus belt; one minute line is one virtual minute), and at least one portal trip completed by town portal, not by the waypoint fallback. Source: translated from G1 criterion 4 (the old "bought and used" is now measurable); docs/beltfill.md; user decision 2026-11-01.

#### Scenario: Empty field window
- **WHEN** packHP is 0 and no belt box holds a healing potion outside town for 6 consecutive minute lines
- **THEN** the run is a FAIL

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
Gate 2 (8 h) and Gate 3 (24 h) SHALL be soaks of profile m24 on a hidden d2d slot, judged by the Gate 1 criteria that apply over their length, the m24 limits and floors of this spec, and `d2dsoak report` ending with `INVARIANTS: PASS` including the primitives invariant. Gate 3 additionally records the d2d memory curve and the periodic d2d restart interval. Source: docs/review-strategy.md "Next week" table (24 h soak); docs/d2d-container/risks.md step 10.

#### Scenario: Gate 2 with a violated invariant
- **WHEN** the Gate 2 report ends with `INVARIANTS: VIOLATED(...)`
- **THEN** the run is not a pass even if the m24 limits hold

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
