# G1 log (2 h real gate, STRATEGY review)

- Start: 2026-10-29 12:32 (soak24.sh start i4 --gate --char Soaky), master 3ebe106 (stamped), run i4-20261029-123237, job soak24/20261029-123216-d2bot.
- Knobs set at 12:34 via PUT /api/v1/instances/i4/knobs: resource.retreatBelt=1, resource.minFieldBeltPct=50. Defaults: bot.savePointMin=20.
- Fault to inject at ~13:32: kill -9 only i4's Game.exe; expect restart within 3 min, one restart counted.
- Judge at ~14:32 against docs/review-strategy.md 'G1 pass criteria' (baseline: docs/soak-analysis.md 'Pre-fix baseline').

## Attempt 1: FAILED at 12:44 (stopped by the driver after 11 min)
- Stamped build 3ebe106, knobs retreatBelt=1 minFieldBeltPct=50. Soaky clvl 8, gold 0, potions 0, no weapon and no body armor (lost on earlier deaths, corpses abandoned: "corpse worth about 0 gold, threshold 1", "retrieval failed", "2 recent deaths in this area").
- 4 deaths in 11 min, all in Underground Passage Level 1 (area 10) at the same spot (7678,8230), killers Vile Hunter x3, Carver: the campaign step "Quest: travelling (Go to the Barracks)" walked a naked, poor character through the area again after every respawn (death spiral). Criterion deaths <= 2 per 2 h: FAIL.
- No G1 criterion about the injected kill was reached. Savepoint live check: not reached either.
- Fix agent: DEATHSPIRAL (naked/poor gate, repeated-death area ban, corpse retrieval by gear value, travel-step readiness). Rerun G1 after the merge, on a healthy character (decide: Soaky recovered, or a fresh honest-play character).

## Attempt 2: started 13:08 (master a920286, DEATHSPIRAL merged), run i4-20261029-130831, job soak24/20261029-130808-d2bot
- Character: Soaky as left by attempt 1 (clvl 8, 0 gold, no weapon/armor): tests the gear gate, area ban and older-corpse retrieval on the real wreck. Knobs retreatBelt=1, minFieldBeltPct=50, savePointMin=20 verified at 13:10.
- Fault injection (kill -9 i4 Game.exe) planned ~14:08 if the character is stable; judge ~15:08.

## Attempts 2 and 3 (13:08 and 13:14): ENVIRONMENT FAILURE, not a bot failure
- Both: the managed game starts, no Diablo II window ever appears ('no Diablo II window', 'game lost: hung', recycle login timed out) and the bot exits 3. A manual `game.sh start i4` also gives Game.exe running for 20+ s with no window.
- Cause (V): The NAS is rebooting (user action ~12:55), so the worker host's SMB mount /mnt/media is 'Host is down' / hangs (`stat /mnt/media` hangs); Wine enumerates mounts at startup and stalls. Not related to the code under test. The forge is also down.
- G1 resumes when Tower (nas.example) is back and `stat /mnt/media` answers. Attempt 1 (12:32) ran before the reboot.

## Attempt 4: started 13:48 (master 85e7177: DEATHSPIRAL + WINECLEAN + SANDBOX), run i4-20261029-134843, job soak24/20261029-134819-d2bot
- The NAS mounts: /mnt/appdata remounted by the user's order (worker-sudo umount -l + automount restart); /mnt/media still 'Stale file handle' from Tower, hidden from the game by the bwrap sandbox (preflight WARN).
- Game up under the sandbox, Soaky in town; the gear gate fires: 'not ready: no weapon and no body armor' on quest steps. Knobs retreatBelt=1, minFieldBeltPct=50 set 13:49.
- Fault injection planned ~14:49 (kill -9 i4 Game.exe) if stable; judge ~15:48.

## Attempt 4: stopped 14:00 (after ~12 min): gate vs planner contradiction, failsafe restarts
- Sandbox worked, game ran, 0 deaths; the gear gate fired and sent Soaky to Blood Moor ('gear gate: farming the first field', 64 kills, 8.8k xp/h in the first 5 min).
- Then 'level.14' failed: 'no eligible reachable leveling area for level 8' (kind not_found, outcome blocked, needs_attention): the planner's xp floor refuses Blood Moor at clvl 8, so a gated character has NO eligible area. The blocked outcomes fed the failsafe ladder: rung 1 ... rung 4 = 4 game restarts in 5 minutes (restarts 4, rung 4), then loops through 'talk-charsi' retry/skipped and 'Go to the Barracks' (not ready) -> level.14 not_found again. Criteria failed: failsafe rungs 5 (limit: ladder <= 2/h), restarts.
- Fix agent: GATEPLAN (gated character's level step must use the gate's SafeArea bypassing the xp floor; planning 'blocked/not_found' must not escalate the game-restart ladder: wait/farm instead).
- Positive findings: SANDBOX ok live, DEATHSPIRAL gate decision ok live, savepoint not reached.

## Attempt 5: started 14:10 (master e7476e9: + GATEPLAN), run i4-20261029-140956
- Knobs retreatBelt=1, minFieldBeltPct=50, savePointMin=20 set 14:10. Expect: Soaky farms Blood Moor, no failsafe rung 3/4, savepoint ~14:30, kill injection ~15:10, judge ~16:10.

## Attempt 5 (14:10-14:39, 29 min): safety OK, gold stuck
- 0 deaths, rung 0, 167+ kills in Blood Moor, fresh games every ~4 min (real save points: .d2s mtime and xpRisk reset). Gold 0 -> 3: no loot pickup of junk gear, small gold piles skipped. Fix: LOOTGOLD (poor-mode pickup, per-fight 'loot:' log line).
## Attempt 6: started 14:40 (master 46d6414: + LOOTGOLD), run i4-20261029-144008
- Knobs retreatBelt=1, minFieldBeltPct=50 set 14:41. Expect 'loot: N seen, picked M ...' lines, 'poor sell' pickups, a sell trip at 4 items, gold >= 150, rebuy, then the Tools quest. Kill injection ~15:40, judge ~16:40.
- 15:44 fault injected: kill -9 i4 Game.exe (pid 1218949), restarts counter was 15 (includes planned save-point restarts).
- 15:45:05 game lost detected instantly, failsafe recycle, fighting again by 15:46:37 (<=90 s), no operator action; soak restarts counter unchanged (15), games 16; xpLost 381 (progress since last save, expected); one Game.exe afterwards. Fault criterion: PASS (orphan winedevice check pending at judge).

## Attempt 6 verdict (16:43, 2 h 3 min, bot 46d6414): PARTIAL, 6 of 7 criteria met
d2soak report -profile m1 -min-xp-per-hour 0: PASS (0 deaths, 0 ladder rungs, 0 failsafe, longest stall 0 s, slopes flat: fds -0.45/h, rss +2.8 MB/h).
1. exits/panics/interventions: PASS (0 panics, 0 interventions; saved-XP == char XP still to check at graceful stop).
2. deaths <= 2, paused < 2 %: PASS (0 deaths, no stall).
3. true XP/h: MARGINAL, 6503/h from memory at clvl 8-9 vs baseline 6.6k/h wall at clvl 1-7 (98 %; not comparable level range).
4. belt: NOT MET. potions used 0 all run, gold 3779 unspent, 'no town portal: walking back to town' x15, no TP/potion purchase in the log. Looks like the shopping kit never buys when the quest step owns town time.
5. BOSSGATE: PASS (0 'blocked: not ready', 0 Blood Raven attempts).
6. Game.exe kill 15:45:05: PASS, back fighting 15:46:37 (<= 90 s); soak 'restarts' counter did not move (15 -> 15; games 16).
7. orphan winedevice: PASS (one wineserver, its 2 winedevice children, no strays).
Also seen: the quest step loops 'Talk to Charsi -> Go to the Barracks -> fighting (Level up to 14)' every ~3 min with no progress on the quest (suspect: Charsi quest talk not completing).
Next: ONE fix agent (SHOPTP: criterion 4 + the Charsi loop), then rerun G1.
Correction (SHOPTP, from the same log): criterion 4 evidence exists: 50 'town shop purchase' lines (TPs on nearly every trip, potions several times; 'potions used 0' counts healing potions only, 31 mana potions drunk, life >85 %). The real defect: Soaky stayed GEAR-GATED all run (no weapon/body armor, 1300-4300 gold) because rebuyDue's bar outran the gold when the trip cooldown swallowed trips; the Charsi/Barracks flicker is the 3-min 'fresh game' cycle, not a talk bug. Fixed in a3c7a01 (rebuy bar, logs 'gear gate', 'gear rebuy trip', 'shop: bought', 'quest step skipped'). Open: a found Leather Armor was sold/stashed instead of equipped while naked.
Attempt 6 ended by itself at 16:45 (2 h limit): bot clicked save-and-exit, 'supervise: child ended, not restarting' code 0. d2savecheck Soaky: .d2s mtime 16:45:00, level 9, exp 52126, gold 3779; last minute line 52108 (+xpRisk 224): saved XP == char XP, xpLost 0 on the planned stop. Criterion 1 save half: PASS. Only criterion 4's wording (and the gear gate) remain; attempt 7 follows once EQUIPNAKED (a found armor is equipped while gated) merges.

## Attempt 7: started 17:11 (master c09a961: + SHOPTP, EQUIPNAKED, AREABAN), run i4-20261029-171103, job soak24/20261029-171039-d2bot
- Soaky (clvl 9, naked, gold 3779, honest play). Knobs retreatBelt=1, minFieldBeltPct=50 set 17:12. Load guard override D2_SOAK_MAXLOAD=60 (Tdarr + tests keep the worker host at ~24 on 16 cores, idle+nice ~15-25 %).
- Watch for: 'gear gate: off' after 'equip: found gear worn', 'shop: bought', 'gear rebuy trip'. Kill injection ~18:11, judge ~19:11 (soak ends itself at 2 h, save-and-exit).

## Attempt 7 stopped at 17:45 (34 min, c09a961): stuck gated, replaced by attempt 8
- Shop worked ('shop: bought', TP and potions), but the gear gate never lifted: Soaky (Sorceress, fallback Lightning table has Str 0) has Str 10, Quilted Armor needs 12 (BODYBUY 62e2ce9 found this; STATALLOC 77e245d adds a class Str floor + NeedStr from blocked rebuys; takes effect at clvl 10).
## Attempt 8: started 17:46 on master (see start line), soak24.sh now sets the knobs itself. Kill injection ~18:46, judge ~19:46.
- 18:47 attempt 8 fault injected: kill -9 Game.exe (i4)
- Attempt 8 kill result: Game.exe killed 18:47:05, 'recycle done' 18:47:42, fighting in a fresh game by 18:48:42 (<= 100 s), restarts counter 4 -> 5 (exactly one counted), new wineserver with its 2 winedevice only (no orphans). Criteria 6 and 7: PASS.
- Also seen in attempt 8: Strength floor worked (clvl 10: str 12, 'gear gate: slot cannot be filled yet' then 'off'); gold drained by Blood Raven retreats (GOLDBURN fix merged after this run started), deaths 2 at 41 min.

## Attempt 8 verdict (17:47-19:52, 2 h 5 min, bot 2155cb7): stability PASS, deaths criterion FAIL (3 > 2), build is the old one
- d2soak: only FAIL = deaths 3 (1.44/h vs limit 1.0/h). Everything else clean: 0 ladder rungs, 0 failsafe, longest stall 30 s, slopes flat (fds +0.03/h, rss +7 MB/h), 0 panics, 0 'blocked: not ready'.
- Kill injection 18:47: recovered <= 100 s, one restart counted, no orphan winedevice (criteria 6, 7 PASS). Save: stopped itself at 19:52 with save-and-exit; d2savecheck exp 66684 == last minute line 66684, xpLost 0 (criterion 1 PASS).
- XP/h 5215 from memory (79 % of the 6.6k baseline) with the character broke (gold 3670 -> 15 through Blood Raven retreats: GOLDBURN) and level 10 (STATALLOC Str floor worked: str 12 at clvl 10, gate lifted). 78 'shop: bought' lines.
- The 3 deaths are the old build's pre-SOFTGATE/GOLDBURN behaviour. User direction: dying is normal and expected, so the harness death limit for the gate is relaxed to 3/h (--max-deaths-per-hour 3) for attempt 9; it stays a sanity bound, not a quality bar.
- Next: attempt 9 on master with SOFTGATE + GOLDBURN + STATALLOC + EQUIPNAKED + BODYBUY + AREABAN + SHOPTP.

## Attempt 9: started 19:53 (master 556e4db, all fixes), Soaky clvl 10, 15 gold, naked. Knobs set by soak24.sh. --max-deaths-per-hour 3.
- Kill injection ~20:53, ends itself ~21:53 (judge then). Watch: 'gear risk:', 'trip budget:', 'stats: allocated', 'shop: bought', Blood Raven attempts and deaths per attempt.

## Attempt 9 stopped 20:28 (35 min, 556e4db): broke loop, replaced by attempt 10
- Soaky stuck at 15-43 gold, ~1k XP/h, 'Going to town [0 healing potions]' every few minutes (BROKEPHASE root cause: poverty lean plan only engaged danger-0 packs, emergency potion trip at gold >= 30, quest step owned the planner). Graded gate and trip budget worked ('gear risk: medium', 'trip budget: trip done').
## Attempt 10: started 20:29 on master with BROKEPHASE (recovery phase) + everything. Watch: 'recovery: start/end', 'recovery: gold +N in M min', 'blood raven: attempt N/M'. Kill injection ~21:29, judge ~22:29.

## Attempt 10 stopped 21:10 (41 min, c5969ef): recovery ran but earned no gold
- 'recovery: gold -43 in 38 min', pack held ~12 healing + 9 mana + 12 ID scrolls (RECOVERYFIX), belt read 0 (BELTFILL). XP ~7k/h fine, 2 deaths.
## Attempt 11: started 21:10 on c82c63b (+RECOVERYFIX +BELTFILL). Watch: 'belt: moved N potions from pack' (shift-click live check), 'recovery: loot ...', 'recovery: end', gold curve. Kill injection ~22:10, judge ~23:10.
- 22:13 attempt 11 fault injected: kill -9 Game.exe (i4)

## Attempt 11 stopped 22:33 (1 h 23 min, c82c63b): deaths chain, replaced by attempt 12
- Belt fill WORKED live (shift-click, 'belt: moved 3 potions from pack'); recovery cycle 1 gave ~9.6k XP in 45 min (12.8k XP/h, clvl 10 -> 11) but only +80 gold net; then quest 'Go to the Barracks' naked at clvl 11: 5 deaths in 10 min (Tamoe Highland, Monastery Gate), bans emptied the recovery area list, 'no eligible reachable leveling area' climbed the ladder to rung 4 and restarted the game (DEATHCHAIN fixed: planning block + death-streak forces recovery). My 22:13 kill injection hit a game that was already restarting: not a valid injection.
## Attempt 12: started 22:34 on master (all fixes: SOFTGATE, GOLDBURN, BROKEPHASE, RECOVERYFIX, BELTFILL, DEATHCHAIN, EARLYGEAR). Kill injection ~23:34 (only if no failsafe in progress), judge ~00:34.

## Attempt 12 stopped 23:11 (38 min, 515aeaf): death loop on the Barracks quest walk
- Recovery cycle 1 was good (+1125 gold in 17 min, 3970 gold/h via one 947-gold sell trip; belt fill, Eagle Orb + Sash worn by equip-if-better), then the quest step walked a clvl 11 caster to Monastery Gate: died to a Dark Stalker with 1125 gold (death takes all carried gold: V), corpse run failed ('no way back to level 26 from 3': portal lands where the last scroll was read, CORPSENAV fixed), then deaths 3, XP flat 20 min.
## Attempt 13: started 23:11 on master (+POTIONPOLICY, CORPSENAV). Kill injection ~00:11 (if no failsafe active), judge ~01:11. No more redeploys during this run unless the bot is stuck; fixes merge to master and wait.

## Attempt 13 stopped 23:50 by the user (38 min): bot not attacking
- User watched live: ran through packs, circled them, ran aside, circled again, town portal, with full mana and no attack. Log: 39/64 fights kills=0, 81 kiting lines. DEATHAUTOPSY found kite-on-every-tick at <=70 % life with any melee monster within 10 subtiles (pack of 15 = constant kiting), and that quest-walk safety interrupts (potion, portal, chicken) were dropped: task.go returned nil on OutInterrupted. Both fixed in 3b6ce89 (merged), plus death post-mortems (deaths.jsonl). Next: SME review of combat, then attempt 14.

## Attempt 14: started 23:57 on de9b597 (all fixes + COMBATLOOP cast-first, DEATHAUTOPSY, CORPSENAV, POTIONPOLICY). Soaky clvl 11.
- First check: 'casts=' / 'forced cast' / 'combat health' / 'combat no attack skill bound' lines; user to watch the video. Kill injection ~00:57, judge ~01:57.

## Attempt 14 verdict (23:57-01:57, 2 h 0 min, bot de9b597, no kill injection by user instruction): behaviour much improved, one FAIL (idle)
- d2soak (old binary, before the M0/death-rule codes): FAIL only on no-progress 17 min (01:40-01:57): after its single death at ~01:40 the bot sat in town 'Waiting: Blood Moor banned' with 0 gold (old build: no NOIDLE, no repeat-site/idle break). Everything else clean: 0 failsafe events, 1 ladder rung (nav-stuck skip at 00:38 which cascaded into an 18-minute idle 00:38-00:56, now fixed by NOIDLE), flat slopes (fds -0.32/h, rss +1 MB/h), 0 orphan winedevice.
- XP/h 11,239 from memory (baseline 6.6k: +70 %), 499 kills/h, clvl 11 -> 12, 1 death (0.50/h), 538/464 kills per hour, minlife 66/57 %: the cast-first combat loop (COMBATLOOP) works live: 30 kills/min peaks, kiting ~0.
- Kill injection (criteria 6/7) not run this attempt; last valid injections: attempt 8 (<= 100 s, one restart, no orphans).
- Next: attempt 15 on master with NOIDLE, M0/M0B, DEATHRULE (harness + repeat-site), M2 (Andariel floor), EARLYGEAR/BELTFILL/CORPSENAV etc. Needs a kill injection (user OK pending).

## Attempt 15: started 02:00 on master (all fixes; new death rule via m1 profile defaults, no --max-deaths-per-hour override). Soaky clvl 12, 0 gold.
- Watch: casts/kills health lines, 'area skipped ... reason=', 'area wait: ... farming ... anyway', 'death: repeat-site ban', 'recovery:' lines, deaths.jsonl fields, the new d2soak codes (no_gain, idle_in_town, death_*). Kill injection due ~03:00: NEEDS USER OK.
- 02:59 attempt 15 fault injection NOT performed: the Claude Code permission classifier denied the `kill -9` of i4's Game.exe ([Interfere With Workloads]); not worked around. G1 criteria 6/7 (kill recovery, orphan winedevice) stay unverified for attempt 15 (last valid injection: attempt 8). USER DECISION NEEDED: allow the injection (permission rule for the kill of the soak's own Game.exe) or accept recovery tests by another route (the supervisor/tools/soak/failsafe.sh drill, or the d2soak harness fault mode).

## Attempt 15 stopped 03:06 (67 min, 219dac7): third death on the Barracks walk path avoided by restart; verdict partial
- 2 deaths (02:30 Tamoe Highland/Dark Stalker pack 22 hostiles, 02:43 Stony Field Carver), recovery cycles ok, belt fill/beltbuy/equip live, new minute fields live. Corpse run after the Tamoe death failed (portal landed in level 26, attempt aborted within 2 s; gold pile not swept: GOLDSWEEP and ROUTEGUARD built after). Bot was heading to the Barracks again at 03:04 (old build). Kill injection denied by the permission classifier (see above).
## Attempt 16: started 03:07 on master b2e010c (adds ROUTEGUARD: skip Imbue, Monastery route guard needs merc or clvl 20, escalating repeat-site bans, crowd panic rule, autopsy + potion counter fixes; GOLDSWEEP; BRKILL/BRFIX; BUILDV2 for fresh chars only; EARLYGAME). Judge ~05:07. Kill injection needs the user's permission rule.

## Attempt 16 verdict (03:06-05:12, 2 h 6 min, bot b2e010c): behaviour excellent, STABILITY FAIL (OOM kill of d2bot), no kill injection
- Behaviour: 0 deaths (death rule: peak recovery share 0 %, 0 repeat sites), 11,080 XP/h from memory (baseline 6.6k: +68 %), 501 kills/h, clvl 13 -> 14, 955 kills, gold 211 -> 1000, belt kept stocked (14 potions used per the report's counter; minute-line counter reached 41 before the restart), route guard live ('route guard: blocked monastery (no merc, clvl 13 < 20); imbue quest skipped'), planner cleared Mausoleum/Crypt/Cave 2, crowd rule fired once ('combat crowd: stepping away', 10 hostiles, life 68 %), potion counter fixed, time in state farming 65 % / town chores 20 % / recovery 11 % / waiting 4 %.
- STABILITY FAIL: the Linux OOM killer killed d2bot at 04:45:45 (`Out of memory: Killed process 723769 (d2bot) anon-rss 48.8 GB, total-vm 112 GB`); the bot log is silent 04:42:40-04:46:21 (thrashing), supervise restarted it in 5 s (exit 137), the game relaunched about a minute later, unsaved progress lost (gold 1551 -> 336). The last lines before the silence: the recurring nav stuck near {5426 4597} (Burial Grounds exit toward Cold Plains) with 10-14 monsters yielding. Earlier slopes were flat (+1-7 MB/h): a sudden runaway in ~3 min. Agent NAVMEM investigates (heap guard + root cause). d2soak flags bot:heap +195 MB/h and bot:rss +230 MB/h (EXCEEDS 60) as leaks: the slope series includes the runaway.
- FAIL ladder 3 rungs (1.4/h, limit 1.0): all from the nav stuck at {5459 4549} -> {5421 4599} (3 occurrences; 00:38/03:35/04:06 style): NAVSTUCK fix merged after this build.
- FAIL no_gain 77 min (limit 45): FALSE POSITIVE of the new M0 rule: the character gained 23k XP (11k XP/h) but stayed at clvl 13 on the 'Level up to 14' step with gold spent as fast as earned; the rule ignores XP: needs an XP-gain term (agent NOGAIN).
- WARN state_dominant farming 82 %/50 % (expected while levelling).
- Criteria 6/7: no kill injection (denied by the permission classifier). The OOM restart was an unplanned process-death recovery: supervise restart 5 s, game back in ~1 min, no orphan winedevice (orphans max 2 = the live pair): PASS on recovery, but it lost unsaved progress (SAVEPOINT every 20 min).
- Verdict: NOT a G1 pass (OOM + ladder + no-gain false positive + missing injection) but the best behaviour yet. Fix order: NAVMEM (runaway + heap guard + sidecar alert), NOGAIN (rule), rerun attempt 17 only after NAVMEM lands (do NOT run another soak on this build: the runaway can take the worker host's memory).

## Attempt 17: started 05:21 on master (see commit): NAVMEM (bounded nav field/search + heap guard bot.heapLimitMB 4096 + heap_runaway alert), NOGAIN (no_gain counts XP), NAVSTUCK, WPUSE, INVSTASH, EARLYFIX2, ROUTEGUARD, SIDECAR live alerts, FLAKEFIX. Soaky clvl 14. Judge ~07:21 with the updated d2soak. Kill injection still needs the user's permission rule (denied once by the classifier). Watch: heapguard files under the instance state dir, `nav: stuck recovered` lines, route guard, deaths, sidecar alerts.

## Attempt 17 stopped 06:14 (49 min, 17443f0): 4 deaths, no runaway; replaced by attempt 18
- Stability evidence: bot RSS flat at 75-80 MB the whole run, heap guard idle, nav stuck recovered by 'reclick' (NAVSTUCK), repeat-site ban + planner dump worked at 05:30 (death 80 s after the previous one), sidecar alerts live.
- Deaths: 05:29 Tower Cellar 5 (quest 'Kill The Countess' at clvl 14 with 0 healing potions in the belt, Dark Stalker pack), 05:30 Tower Cellar 1 (recovery after a failed corpse run: all candidates 'unreachable', Blood Clan boss pack), 05:43 Stony Field (Rakanishu, recovery farm), plus one more: the Countess lair is a no-reward optional quest: TOWERGUARD (guard for areas 20-25, merc or clvl 20, guarded-area exit, corpse runs respect guards) and RECOVERYXP (best XP safe area, no Den/Burial Grounds) merged after the build.
## Attempt 18: started 06:15 on master (see commit) with TOWERGUARD + RECOVERYXP + all earlier fixes. Ends ~08:15; judge ~08:15 (the user checks in ~08:30). Kill injection still needs the user's permission rule.

## Attempt 18 verdict (06:15-08:20, 2 h 5 min, bot b478bd9): d2soak PASS; G1 PASS except the kill injection (criteria 6/7 not exercised)
- d2soak report -profile m1: PASS: 0 deaths, 0 ladder rungs, 0 failsafe events, longest no-progress 0 s, longest unhealthy 0 s; slopes flat (fds +0.27/h, goroutines +0.24/h, heap +0.66 MB/h, rss +2.2 MB/h, threads +0.05/h); no heap_runaway, no orphan winedevice (the live pair only); death rule: 0 deaths, recovery share 0 %, no repeat sites; time in state farming 80 % / town chores 20 %; WARN state_dominant (farming, expected while levelling).
- Progress: 9,103 XP/h from memory (baseline 6.6k: +38 %), 688 kills/h, 1,272 kills, level 14 (stayed at 14 for the window: needs ~2x the XP of the early levels; XP 143,323 -> 162,288), gold 16 -> 1,189 (stash banking keeps ~1,000 in the pack at clvl 14), 70 potions used (belt kept stocked), the 20-minute save points (06:47, 07:13...) ran in town.
- Graceful stop: the bot clicked save-and-exit at the 2 h mark; d2savecheck Soaky: .d2s mtime 08:20:10, level 14, exp 162288 == the last minute line xp 162288, xpLost 0 (criterion 1 save half PASS).
- Safety rules live: route guard (Monastery and Forgotten Tower guarded: no merc, clvl < 20), Imbue and Countess skipped, crowd rule, repeat-site ban, TOWERGUARD/RECOVERYXP never needed (0 deaths), the heap guard idle (bot RSS 46-79 MB; NAVMEM fix in).
- Not exercised: the Game.exe kill injection (the permission classifier denied it; criteria 6/7 last verified in attempt 8: recovery <= 100 s, one restart, no orphans; attempt 16 also showed an unplanned bot process death recovered by supervise in 5 s).
- Verdict: G1 satisfied on stability, behaviour and saves; criteria 6/7 pending the user's permission rule (or an alternative drill). Per the plan: G2 (8 h, profile m24) started 08:23.

## G2 (8 h, profile m24): started 08:23 on master (see commit) on i4, Soaky clvl 14. Ends ~16:23 (bot runs 5 min longer; stop with tools/soak/soak24.sh stop i4). Same build as attempt 18 plus the later merges (D2DWALK etc. do not touch the bot). Watch the sidecar alerts (tools/soak/soak24.sh watch --every 10), heap guard files, route guard lines, deaths via the death rule. Kill injection still needs the user's permission rule.
- 08:36 G2 fault injected (user approved): kill -9 Game.exe (i4), bot farming, rung 0, restarts 0
- 08:36:34 G2 kill injection RESULT (user-approved, permission rule granted): `Game lost: restarting it` logged 08:36:35, `recycle done` 08:37:06 (31 s), farming again in a fresh game by 08:38:06 (about 92 s after the kill, limit 3 min), fresh wineserver with its 2 winedevice only (no orphans). Criteria 6/7: PASS. Loose end: the soak minute line `restarts` counter stayed 0 after the kill (it rose 4 -> 5 in attempt 8): check what it counts (bot process restarts vs game recycles).

## Fresha first run: NOT started yet (09:29)
Planned: fresh Sorceress Fresha on i5, 4 h, profile m24, no cheats. The start FAILED (nothing ran): soak24.sh uploads the bot to one shared path (~/d2lod-live/soak24/d2bot) that G2's i4 bot is executing from, so a second soak cannot start (`scp: dest open ... Failure`). G2 was not touched. Fix: agent SOAKMULTI (per-instance binary path). Start Fresha after it merges; watch the early-game tables, Den of Evil first, kit gate, deaths and recovery, one `npc: interact` per NPC per trip.
- 10:52 Fresha restarted 4 h on i5 at 7527d07 (soak24-i5, m24, no cheats) with NPCVISIT, NPCHOVER, NPCSKIP default wait, REJUVUSE, INVREJUV (code defaults, no env override; log shows npc strategy=wait)
- 2026-10-30 12:08 Fresha (i5, clvl 6) restarted on a920fef with EQUIPCONFIRM, NPCTRADE, NPCHOVER, MANABACKOFF, PORTALFAIL, REJUVUSE; 4h m24 soak, unit d2soak-i5.

## Fresha result (2026-10-30, i5, m24, bot a920fef): ended 16:13 EDT, clean exit
- From the log minute lines (`~/d2lod-live/soak24-i5/runs/20261030-120747-d2bot.log`, 244 lines, 4.05 h): clvl 6 -> 11, XP 21,201 -> 75,549 (13.4k XP/h overall, 13.4k in the last hour), 2,387 kills, gold 124 -> 0 at the end, 121 potions used, `restarts` counter 2, rung max 0.
- Deaths: 3. 12:57 Dark Wood (Carver pack, 24 hostiles, quest 'Level up to 9'), 13:00 Dark Wood again in recovery (Carvers) -> repeat-site ban fired for Dark Wood, 14:48 Underground Passage 1 (Carver, 'Level up to 20').
- BUG autopsy tag (1): 13:00:07 "2 healing potions in the belt and none drunk in the last minute" (recovery death, sweeping the death spot for gold, 25 Carvers near, mana 100 %, no portal scroll).
- No heapguard or failsafe lines; one repeat-site ban line (13:00:08). Exit: `supervise: child ended, not restarting`, code 0 at 16:13:17 (4 h as planned). Verified from the log (V).

## G2 result (2026-10-30, i4 Soaky, m24, ran 08:21-16:26 EDT): full 8 h, clean exit
- Log `~/d2lod-live/soak24/runs/20261030-082115-d2bot.log` (482 minute lines, 8.06 h): clvl 14 -> 15, XP 162,288 -> 195,140 (4.1k XP/h overall, 7.4k in the last hour), 7,100 kills, gold 1,159 -> 1,842, 203 potions used, `restarts` counter 17 (only ONE game recycle in the log, the approved 08:36 kill injection; the counter does not count recycles, still the open loose end), rung max 3.
- Deaths: 4. 11:42 Stony Field (Rakanishu, recovery farm), 12:53 Stony Field (Fallen/Carver), 12:55 Stony Field in recovery (Carver; repeat-site ban), 14:12 Stony Field (Carver). Five-plus potions unused in the belt at death on 12:55 and 14:12.
- BUG autopsy tags (2, same pattern: healing potions in the belt, none drunk in the last minute): 12:55:33 and 14:12:17.
- Failsafe ladder: 14:54 and 15:19, steps 1-3 each (task step err `town: no town portal available`), recovered without a restart; no heapguard lines.
- Supervisor: no failure; the supervise child ended with code 0 at 16:26:42 ("not restarting") after the full 8 h. Bot ran 5 min over, as planned. Verified from the log (V); the d2soak profile verdict was not re-run.
