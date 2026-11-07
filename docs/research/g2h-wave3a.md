# Gate 2 wave 3a (tag g2h): three-seed result against g2g

> status: current, 2026-11-07. Task TASK-119. Docs only. Record: `docs/soaks/gate2-multiseed-20261107-1811-g2h.md` (generated), per-seed records `docs/soaks/gate2-s{11,13,17}-*-g2h.md`, logs `docs/probes/soak-logs/*g2h*`. Compared with g2g on the same seeds: `docs/research/g2g-five-seed.md`, `docs/research/g2g-diagnosis.md`.

Tags: V = counted from the committed minute logs, the per-seed records, the full bot logs on the worker host (read-only) or `d2dsoak compare`; S = secondary or approximate; U = unverified.

## 0. This is a 3-seed result, not the five-seed claim
Seeds 19 and 21 (wave 3b) were NOT run: the user's direction decision D1 (2026-11-07) pauses gate waves. The Gate 2 contract (`docs/contracts/soak-gates.md`, "Gate 2 and Gate 3 on d2d") needs every seed to pass, and `docs/decisions.md` (2026-11-03) asks for at least 5 seeds with 2 holdout. This run has 3 seeds and 1 holdout seed (17). Every number below is therefore **indicative**; it cannot clear Gate 2.

## 1. Setup (V)
- Bot origin/main 75087859 (contains #296, #298, #299, #286, #300), binary `d2d.lab3-labverbs-f75816c6-linuxbuild`, ladder off, 8 wall h, fault at 60 wall min, tag g2h. Runs gate2-s13-20261107-1007, gate2-s11-20261107-1008, gate2-s17-20261107-1008 (ended 18:08 EST).
- Judge: `d2dsoak` rebuilt from origin/main 3eea4ba7 (`labrun orion -- nice -n 19 go build -o /tmp/d2dsoak.g2h ./cmd/d2dsoak`), copied into each run's `bin/` with the shipped binary kept as `bin/d2dsoak.orig-20261107`, then `gate-run.sh --collect-multi --gate 2 --tag g2h`. The belt rule is the funded rule (#300).
- Restart-containing NOKILL and BELT windows stay literal. Andariel interim floor stays clvl 17 (hold on). Gate 3 not started.
- Virtual minutes reached: 1780 / 2379 / 1899 (seeds 11 / 13 / 17); speed 3.7x / 5.0x / 4.0x virtual per wall. The host was shared with probe and review agents. g2g reached 1963 / 2414 / 1699. The two g2g NOKILL failures sat at virtual minutes 1190 (seed 13) and 1036 (seed 17), inside the 1780 minutes every g2h seed reached, so the shorter exposure on seeds 11 and 13 does not hide that pattern. XP/h is compared over the first 1579 virtual minutes for all runs. Slower speed does not change the verdict (virtual time is what the criteria count).

## 2. Per-seed table (V) against g2g
| item | 11 g2h (g2g) | 13 g2h (g2g) | 17 g2h (g2g) |
|---|---|---|---|
| verdict, all criteria | PASS (PASS) | PASS (FAIL: NOKILL 11) | PASS (FAIL: NOKILL 14) |
| virtual min / clvl | 1780 / 16 (1963 / 16) | 2379 / 17 (2414 / 17) | 1899 / 16 (1699 / 16) |
| XP / kills | 238242 / 11713 (249706 / 12522) | 284517 / 15571 (289572 / 15123) | 242710 / 12436 (243718 / 10696) |
| deaths / writes after start | 0 / 0 (0 / 0) | 0 / 0 (0 / 0) | 0 / 0 (0 / 0) |
| longest NOKILL (limit 10) | 8 (6) | 8 (11) | 10 (14), at the limit |
| longest funded BELT (limit 5) | 2 (3) | 1 (1) | 2 (3) |
| fault recovery, path A (limit 180 s) | 2.3 s (3.4 s) | 2.2 s (2.2 s) | 2.3 s (2.2 s) |
| Andariel interim rule (hold on) | PASS, 14 gate/held lines (PASS) | PASS, 21 (PASS) | PASS, 9 (PASS) |
| fresh games (d2dsoak) | 44 (41) | 32 (88) | 20 (25) |
| bot restarts at end (minute field) | 44 (41) | 50 (88) | 22 (25) |
| restarts at the 461 wall-min mark | 42 | 46 | 22 |
| `mouthRef` summed on minute lines | 344 (422) | 362 (468) | 133 (206) |
| XP/h first 1579 min (baseline 9172) | 8437, -8.0% (8533) | 8885, -3.1% (8451) | 8679, -5.4% (8742) |
| XP/h full run (advisory) | 8024 (7627) | 7171 (7193) | 7662 (8600) |
| dominant `fsWhy` | np_clearing 28, recovery 5, task_step 5 | np_clearing 26, task_step 16, recovery 9 | task_step 9, np_clearing 6, recovery 1 |
| town-portal trips by log text | 311 | 408 | 337 |
| `tp` field on minute lines | max 6, valid | max 7, valid | max 6, valid |

d2dsoak over the three seeds (cut to 1780 min, 88.9 virtual h; V): `INVARIANTS: PASS`, `OVERALL: INCONCLUSIVE (exit 2)` because of "3 seeds, minimum 5". xp_per_hour 8103 (sd 210, -11.6% vs 9172, tolerance 1834, inconclusive), deaths_per_1000_kills 0 vs 1.02 (inconclusive; the zero-deaths-flagged question of g2g does not arise at 3 seeds). Mean XP/h over the first 1579 min: 8667 (-5.5%) against 8575 (-6.5%) in g2g.

## 3. Pass and fail count and which criteria (V)
All three seeds pass every criterion of the per-seed table (stability, deaths, m24 floor, NOKILL, BELT, town portal, route guard, fault, uncheated, invariants, no-gain window, clvl 16, gold, Andariel, merc hire). Pass count 3 of 3 (11, 13, 17). The two g2g failures (seed 13 NOKILL 11, seed 17 NOKILL 14) did not recur. Margin is thin on seed 17: NOKILL 10 is exactly the limit, in a window at area 17 ("Quest: fighting", no restart, no failsafe mechanism noted in it; its cause is U). One run per seed per build, so whether the improvement is the fixes or seed variance is U.

## 4. Mercenary (V, full bot logs; same script on g2g and g2h)
| item | 11 g2h (g2g) | 13 g2h (g2g) | 17 g2h (g2g) |
|---|---|---|---|
| merc died / revived | 39 / 38 (17 / 16) | 42 / 40 (17 / 16) | 25 / 23 (16 / 15) |
| revives per virtual hour | 1.28 (0.49) | 1.01 (0.40) | 0.73 (0.53) |
| alive share, time-weighted | 42.3% (11.4%) | 32.7% (4.0%) | 30.7% (15.7%) |
| `revive skipped (revive cap)` | 127 (52) | 98 (52) | 43 (58) |
| `revive skipped (gold)` | 48 (1) | 147 (0) | 159 (3) |
| `revive skipped (failed: purse below quote)` | 4 (0) | 16 (0) | 6 (0) |
| gold paid for revives | 13260 (3030) | 21570 (3030) | 12810 (4500) |
| minutes with "gold" in the step text (S) | 769 of 1780, 43% (804 of 1963, 41%) | 796 of 2379, 33% (483 of 2414, 20%) | 946 of 1899, 50% (603 of 1699, 36%) |
The g2g alive shares from the same script (11.4, 4.0, 15.7) match the diagnosis (11.5, 4.2, 16.8) to about one point. The merc is dead at the last event on all three g2h seeds (last `seen`/`died`/`revived` event; the per-seed record's "last read alive" is a different read, U which is right). The merc still dies within a few minutes of each revive (39 deaths in 29.7 virtual hours on seed 11), so the larger alive share is bought with 1.6 to 2.5 times the deaths and 3 to 7 times the gold.

## 5. Proof checklist of the three fixes (from `g2g-five-seed.md` section 9)
| fix and item | result | evidence |
|---|---|---|
| #296 revives follow the virtual clock (more than 2 per wall hour) | PASS | 38 / 40 / 23 revives in 8 wall h = 4.8 / 5.0 / 2.9 per wall hour |
| #296 fewer `revive cap` skips | FAIL on 11 and 13, PASS on 17 | 127 / 98 / 43 against 52 / 52 / 58; the cap now binds more because there are 2 to 3 times more attempts; gold skips (48 / 147 / 159) are a new binding limit |
| #296 merc alive share clearly above 4 to 30% | PASS | 42.3 / 32.7 / 30.7% against 11.4 / 4.0 / 15.7% |
| #296 less time in the gold-farm zone list | FAIL (S) | step text with "gold": 43 / 33 / 50% against 41 / 20 / 36% |
| #296 no BELT regression from the larger revive spend | PASS | funded BELT 2 / 1 / 2 against 3 / 1 / 3 |
| #298 `mouthRef` sum well below 160 to 650 per seed | PARTIAL | 344 / 362 / 133 against 422 / 468 / 206 (-18 / -23 / -35%); only seed 17 falls below the old range |
| #298 no ladder-driven NOKILL window | PASS | longest windows 8 / 8 / 10 (limit 10); the 8-minute window on seed 13 (min 853) still contains 2 restarts and 5 np_clearing failsafe lines (within the limit) |
| #298 fewer fresh games than 25 to 88 | PARTIAL | 44 / 32 / 20 against 41 / 88 / 25: down on 13 and 17, +3 on 11 |
| #299 no `town trip abandoned [no town portal scroll]` | PARTIAL | 0 / 0 / 1 against 0 / 0 / 2; the text "no town portal scroll" still appears in 38 / 26 / 21 log lines (also in combat-survival refusals) |
| #299 no nav-pocket NOKILL window | PASS | `nav: stalled in a pocket` 2 / 7 / 6 (g2g 7 / 6 / 11); no top window is a pocket window; seed 11 has an 8-minute window at min 1403 whose step text is "Going to town ([no town portal scroll])", so the scroll gap can still stall for up to 8 minutes |
| #299 / #286 `tp` field valid | PASS | nonzero on all three seeds (max 6 / 7 / 6; 0 on 160 / 149 / 200 minutes), 0 on every minute in g2g |
| Every seed: NOKILL at most 10, funded BELT at most 5, 0 deaths, 0 writes; early-death lines still fire | PASS | table in section 2; `safety decision chicken` 5 / 7 / 6 and `potion: not drinking because chose chicken first` still occur |
| five seeds with two holdout seeds | NOT MET | three seeds, one holdout |

## 6. Caveat: d2d heals the hero on every reload (analysis of how much this inflates the result)
Premise (S, `docs/research/fid-loadhp-trace.md`, TASK-101): on d2d a chicken-and-rejoin or any load returns the hero at full life and mana (`World::take_character`), where game.exe keeps the saved value; B1 of that document says a chicken rate on d2d over-states survival. The bot treats the restart as a normal event; the soak counts "Save and Exit" as reported, not gated.

Bounded estimate from the minute lines (V for the counts, U for the exact heal because a minute line can precede the reload by up to a minute):
- Reload events: restart minutes (the `restarts` counter rises) 44 / 45 / 18 per run; "Save and Exit requested (chicken / leave game)" log lines 161 / 173 / 113, of which most are planned fresh games (`leveling: everything worth farming is cleared, starting a fresh game` 36 / 37 / 15), the injected kill, and failsafe ladder restarts. Real survival chickens (`safety decision` action chicken): 5 / 7 / 6, with the hero at 29 to 38% life on average at the decision.
- Life lost to the heal: at the minute line before each restart the hero had on average 90 to 93% of its life. Summed over all restart minutes the heal is 3.3 / 4.2 / 1.8 full-life bars per run; in no restart minute was the previous life reading below 50% (0 of 107). Mana: 12.0 / 14.0 / 5.5 full-mana equivalents. The runs used about 1260 / 1690 / 1230 potions (potionsUsed on the last minute line). At roughly half a bar per healing potion that heal is under 1% of potion use. For the hero near death only the 18 chickens count: they all rejoin in town in the real game as well, where a heal costs a potion or a healer visit, so the bias is some tens of gold of potions and a few minutes of recovery per chicken, not a death avoided.
- Effect on results: deaths (0): at most those 18 chicken events could have been deaths after a rejoin at low life, which needs a hostile town (not seen); treat the 0-death result as slightly optimistic, not as an artifact. NOKILL: a reload does not hide a no-kill window (kills are counted) so no inflation; the 8-minute seed 13 window contains restarts, and the heal could only shorten a restart-bound window by the potion time not spent (U). BELT: the funded-window length could be shortened by a reload that restores life without a potion (U whether the judge counts a life jump as healing; the funded windows are 2 / 1 / 2 against the limit 5, so the effect needs a 2.5 to 5 times error to flip any verdict).
- Verdict impact: none of the three seeds is within one heal-sized margin of failing except seed 17 NOKILL (10 of 10), and a heal does not touch NOKILL. The same bias applies equally to g2g, so the g2g-to-g2h comparison is not distorted by it.

## 7. Verdict (indicative)
- Three seeds, three PASS on every per-seed criterion; `INVARIANTS: PASS`; d2dsoak `OVERALL: INCONCLUSIVE` only because of the seed count. Both g2g failures (NOKILL on 13 and 17) are gone, seed 17 only just (10 of 10).
- Gate 2 is NOT cleared by this run: the contract needs every one of at least five seeds (two holdout) to pass; seeds 19 and 21 on this build are not run (decision D1). 
- Of the three fixes: #296 works on the clock and alive share but costs more deaths, gold and cap skips, and does not cut gold-farm time; #298 cuts `mouthRef` and fresh games by a modest amount (seed 11 fresh games +3); #299 removes most scroll abandonment and the pocket windows but an 8-minute scroll-driven stall remains on seed 11.
- Open items for a later wave (no run started here): merc cost and cap behaviour at 1 to 2.5 revives per virtual hour (rows above); the seed 17 10-minute window at area 17 (cause U); the seed 11 scroll stall at min 1403.

## 8. Not done
No run was started (no wave 3b). The independent Opus verification of the load-life finding (`fid-loadhp-trace.md` header) is still pending, so section 6 rests on that document as S.
