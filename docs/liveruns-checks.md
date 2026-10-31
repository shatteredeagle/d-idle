# Live-run runbooks R0..R12 (agent LIVERUNBOOK; see [liveruns.md](liveruns.md) for rules and procedures 3.1 to 3.7)

> status: Game.exe runbooks R0..R12; d2d is the primary backend (ADR 0009, 2026-10-30; docs/plan/STATUS.md), Game.exe frozen (see docs/liveruns.md banner).

Tags V/S/U as in liveruns.md. Every command below was checked against the script or the flag parser named in brackets; none was run live.
`L` = the guard function of liveruns.md rule 1 (`export D2_INSTANCE=iN` first). `<ts>` = the timestamp in the `job` line printed by `--bg`.
Pass/fail texts quote the doc that defines the check; "record" names the doc and section to add one dated line to.

## R0. Reap dry-run and health (B0, no game, about 2 min) - INDEX 33b
- Commands: `tools/devhealth.sh` ; `tools/game.sh reap-all --dry-run` ; `ssh worker.example 'ps -eo pid,args | grep -E "winedevice|wineserver" | grep -v grep'`
  [V game.sh: reap-all needs exactly `--dry-run`; kills nothing].
- Expect: candidates listed only for instances without a Game.exe; i4 (soak running) must NOT appear as a candidate. Compare each candidate pid with `ps`.
- Pass: every listed pid is an orphan `winedevice.exe` of a stopped prefix and none belongs to i4. Fail: a pid of a live instance is listed (the reaper's
  per-prefix lock or layout check is wrong: stop, report with the ps lines).
- Record: docs/leakhunt.md (one line), INDEX 33b status.

## R1. Soak gate verdict, then 24 h (B4, i4) - INDEX P0 1
Now: the 2 h gate is running on i4 (liveruns.md s1). Do NOT start a second soak on i4.
- Read the result after about 12:50: `ssh worker.example 'ls -d ~/d2lod-live/soak/i4-20261029-104844; cat ~/d2lod-live/soak/i4-20261029-104844/report.txt'` or
  `tools/soak/soak.sh report` (exit 0 pass, 1 fail, 3 no data) [V soak24.sh/doc]. `tools/soak/soak24.sh watch --once` for the live status.
- Gate pass: report.txt PASS with deaths/h <= 1, XP/h >= 0 (gate drops the XP floor), longest stall < 10 min, fd slopes within limits (soak24.md s3).
- 24 h run, only after a gate PASS and a quiet night (load <= 12):
  ```
  tools/soak/soak24.sh preflight i4          # needs the game stopped: NOLOCK=1 tools/game.sh stop i4 ; helper deployed from this checkout
  tools/soak/soak24.sh start i4              # profile m24, --supervise, campaign goal, char Booti
  tools/soak/soak24.sh watch --every 10      # second pane; log ~/soak24-watch.log
  ```
  [V soak24.sh: `start` runs preflight, `D2_INSTANCE=i4 tools/live.sh --bg --nocap --max <s+1800> soak24 d2bot --managed --instance i4 --char Booti
  --addr 127.0.0.1:8420 --stop-game --supervise --goals campaign --autostart --run-for <h*60+5>m`, then `soak.sh start i4 --for 24h --profile m24`.]
- Early stop: `tools/soak/soak24.sh stop i4` (never `jobs.sh kill` alone). Cleanup: after the run `tools/soak/soak24.sh stop i4` leaves the game stopped;
  copy report.txt/json, alerts.json and the watch log to `/mnt/cold/d2lod/soak/<run>/`.
- Record: docs/soak24.md hand-off (section 6), memory note `~/.claude/memory/d2-idle-bot.md` (main agent), INDEX P0 1. First crit alert and every manual
  restart (voids the run) with times.
- Known U: the `--supervise` relaunch and re-login path after a crash (soak24.md "Crash survival"); the m24 numbers.

## R2. Countess without `--nokite` (B1, i2 Combi) - INDEX P0 4, boss.md s11 check 1
- Setup: MAIN brings i2 up and logs in Combi (3.1: `oog "start Combi"`). Then `L lr2 d2cheat show` and `L lr2 act2live state`:
  need gold >= 3000, life about 969, belt with 2 heal + 2 mana, TP tome with >= 3 scrolls. If gold is 0 or life is low (BOSSLIVE2 left it so):
  `L lr2 oog leave`, then
  `L lr2 d2cheat --reason "R2: Combi lost gold on death; Countess retest needs potions" --actor lr2 save Combi gold=20000 maxlife=969`,
  `L lr2 oog "start Combi"`; then in town `L lr2 townlive chores potions scrolls` [V townlive: forced chores subset] and check `act2live state`.
  Waypoints: Combi has all 39 (V qdump), Black Marsh included.
- Command (background, about 3 to 14 min):
  `L --bg --max 960 lr2 bossrun --god "R2 Countess pin-fix retest, throwaway Combi" --timeout 14m countess`
  [V cmd/bossrun: flags before BOSS; `--god REASON`; do NOT pass `--nokite`]. Follow: `tools/jobs.sh tail lr2/<ts>-bossrun 40`.
- Expect log lines: `boss presets` for area 25, `boss view: unique` with name "The Countess" (class 45, uniqueNo 6), `note: preset .../836 ...`, and at least one
  `pinned: evasive moves go nowhere, fighting in place` (V string in internal/combat/fight.go:733) while the escort crowd is on her. Summary line
  `boss="The Countess" killed=true loot=N phases=locate,...,loot elapsed=...`. Exit code 0 (the foreground wrapper prints `[exit 0]` in the job log).
- Pass: `killed=true`, exit 0, wall time under 14 min, the pinned line appears OR she died without it within 3 min (then the escort never pinned it:
  say so). Fail: exit 4 (timeout) or 1 with phases `approach,locate,attack,hazard:adds` repeating (the old BOSSLIVE2 run 1 symptom): then rerun once with
  `--nokite` as the control and report both. Exit 5 (died) under god: report max life readings (`d2cheat show`).
- Leave: bossrun portals home before stopping god (V boss.md s10). Check `L lr2 townlive state` area 1 (town), `L lr2 act2live key esc`. Staying in game for R3.
- Record: docs/boss.md new section "12. LIVERUNBOOK R2" with the numbers (kill time, escort size, pinned lines, loot items), flip s11 check 1 to V; INDEX P0 4.

## R3. Smith: preset 754 (B1, i2 Combi) - INDEX P0 3, boss.md s11 check 2
- Precondition: Combi in town with the R2 consumables; Outer Cloister WP (V all waypoints). The quest words are Tools of the Trade started (word 3 = 0x000c,
  V qdump); BOSSLIVE2 cheated Den done + that word and still never found the Smith, so quest dependence is U either way.
- Command: `L --bg --max 960 lr2 bossrun --god "R3 Smith preset 754 check, throwaway Combi" --timeout 14m smith`.
- Expect: NO line `note: preset 1/754 not found` (V string `preset %d/%d not found`, kind 1 = monster preset; if it IS printed the anchor failed and the
  script falls back to object 108 then Explore); a `boss view: unique` line for "The Smith" (class 402) near (15119,4389); summary `killed=true`,
  `loot>=1` and `Horadric Malus` among the loot lines (boss.md B5).
- Pass: Smith seen at/near 754 and killed. Partial: not found in 4 min: then WITHOUT leaving the game run
  `L lr2 act2live goto 15119 4389 -- units 402 -- presets` [V act2live subs `goto X Y`, `units [CLASS]`, `presets`] standing at that tile to see if class 402 is
  streamed there, and `L lr2 d2cheat show`; and read the quest word: `d2cheat qdump` on a copy of Combi.d2s after `oog leave` (copy
  `~/Games/d2lod-inst/i2/.../Saved Games/Diablo II/Combi.d2s`). Report: 754 resolved yes/no, Smith present yes/no, quest word value.
- Leave: home by bossrun; verify town. Record: boss.md section 12 (Smith), INDEX P0 3.

## R4. Andariel (B1, i2 Combi) - INDEX P0 2, boss.md s7.4 A1 to A5 and s11 check 3
- Setup in town: belt 2 heal + 2 mana in 4 slots (the new gate passes with 2 heal in a 4-slot belt, V offline), plus 2 antidotes:
  `L lr2 townlive buy 148 yps yps` [V townlive `buy CLASS CODE...`; Akara is class 148 (V monstats hcIdx; probes.md's "Akara 150" is wrong, 150 is Kashya); antidote code `yps` S, Akara selling it U]; clvl 30 >= 22 (V).
  Catacombs 2 waypoint V (all waypoints).
- Command: `L --bg --max 960 lr2 bossrun --god "R4 Andariel approach/poison/kite retest, throwaway Combi" --timeout 14m andariel`.
- Expect (boss.md A1 to A5): `travel` to Catacombs 2, `exit 36`, `exit 37` land in areas 36, 37 in order (V ids); `note:` about the Hint walk to
  (22549,9520); a pre-fight `antidote` call (belt count -1); `boss view: unique` for Andariel with her real position; phases `kite` every >= 1.5 s when
  she is within 10; summary `killed=true elapsed<12m`. Gate failure text `Andariel needs 3 potions, have 2` means the belt-aware floor is NOT in
  the deployed build: stop and report.
- Pass: A1 hops correct, killed under 12 min. Record the values boss.md s7.4 lists (poison state 2 seen, cloud missile names, kite walkability).
  Under `--god` the escape logic is not exercised (check 7 stays U, say so).
- Leave: town. Record: boss.md section 12, INDEX P0 2.

## R5. Probes riding the boss fights (B1, i2) - INDEX 24b (monflags, missiledump)
- While a `bossrun` job from R2/R3/R4 is fighting (area 25, 28 or 37), in a second shell, `--nolock` because probes read memory only (probes.md: "memory read
  only"; U that the lock-free read is harmless next to a running bossrun, so read the job log for `error` lines):
  `L --nolock lr2p probes monflags -sec 150 -radius 40 -ms 1000 > /tmp/lr2-monflags.jsonl`
  `L --nolock lr2p probes missiledump -sec 120 -ms 250 > /tmp/lr2-missile.jsonl` [V probes flags: -sec 1..170, -ms, -radius; `--nolock` is a live.sh flag
  and its path runs `timeout ... env D2BOT_INSTANCE=$inst` without flock, V]. Start the probe as soon as `tools/jobs.sh tail` shows the arrival in the boss area.
  With `D2_INSTANCE` still exported, a different agent name `lr2p` keeps the binary separate.
- Expect monflags: `monster` lines with `flags`, `flag_bits`, `mon_kind`, `enchants_raw`; the Countess line `unique_name` "The Countess", `mon_kind` boss/unique;
  escort "Dread Jade the Quick"-type champions/uniques show kind champion/unique; end `summary.verdict`. Compare with `tools/shot.sh` once (blue/gold names).
- Pass monflags: normal monsters read bit 2 only, a champion reads 4, a unique 8, minions 16 (D2BS bits, U until now) -> `d2/monflags.go` U notes can go.
  Fail: all flags 0x00 (offset wrong) or mismatch with the screenshot: record raw bytes.
- Pass missiledump: `ok:true` samples, a missile class list with `name`/`kind`, `classified_other` empty or listed for fixing `MissileKindByName`; curse state names resolved.
- Record: docs/difficulty.md (monflags V/U per bit) and probes.md "Live results" (new section), INDEX 24b. Save the jsonl files under /mnt/cold/d2lod/ only if bulky.

## R6. Necromancer W5 checks (B1, i2 Necki, no second instance start) - INDEX P3 18
- Setup: `L lr2 oog leave "start Necki"`. Necki is Nec clvl 15, allow-listed (V). Needs clvl 30 skills Revive (95) and Corpse Explosion (74) learned and
  bound; the normal30 build has Corpse Explosion but NOT Revive (V json), so spend by save edit (skill ids V from `~/d2lod-live/necrolive/data/skills.tsv`:
  Raise Skeleton 70, Corpse Explosion 74, Revive 95; Revive needs clvl 30 and Raise Skeletal Mage + Iron Golem as prerequisites, V; `skills=ID:LVL` sets only the named skills' hard points, others untouched, V internal/cheat/d2s SetSkillLevel; whether the game accepts
  a skill without its prerequisites is U):
  `L lr2 oog leave` ; `L lr2 d2cheat --reason "R6 W5 Revive/Corpse Explosion live check, throwaway Necki" --actor lr2 save Necki level=30 skills=70:8,74:5,95:3 statpoints=0` ;
  `L lr2 oog "start Necki"`. Then bind: `L lr2 necrolive bind` and `L lr2 necrolive state` (agent `lr2`; necrolive subs V: state bind setleft home key auras buff census farm).
- Run: `L --bg --max 600 lr2 necrolive farm 420 2` (kills in Blood Moor/area 2 for 7 min; sub `farm SECS [AREA]` V); in parallel poll `tools/jobs.sh tail`.
  Then `L lr2 necrolive census` for Skeletons/Mages/Golems/revived count.
- Expect (summon.md W5/W6 items 1, 2): `combat summon ... skill Revive grew=true paid=true`; census counts the revived unit as `Other`; the real cap learned
  (`capSeen revive` log); Corpse Explosion casts on a corpse with >= 2 hostiles within 6, kills a weak pack; mana never starves the attack (kills/min
  vs the earlier 17.8). Revive on a champion/boss corpse: only if a Blood Raven or unique dies: else mark U.
- Pass: Revive count and cap recorded; one Corpse Explosion kill seen. Fail: Revive never planned (skill not bound/learned: read `necrolive state` hotkeys).
- Leave: `L lr2 necrolive home`, `L lr2 act2live key esc`. Then end of B1: `tools/game.sh stop i2`.
- Record: docs/summon.md "Live results" (new subsection), INDEX 18.

## R7. ACT2LIVE7 chain: tomb 69, orifice, Duriel, Tyrael (B2, i1 Navi) - INDEX P1 8
- Setup: MAIN brings up i1 and logs Navi (Sorc 24, Act 2, all waypoints; V qdump). `L lr1 act2live state` must list the Horadric Staff
  (code `hst`) in the pack; if not, `L lr1 act2live grab hst` then `key i` and put it in the pack (never end a run with it on the cursor). Max life 290:
  Navi has no gear, dies in seconds without god (V). Optional: `L lr1 d2cheat --reason "R7 Navi under-geared for tomb packs" --actor lr1 save Navi level=30` only
  with the game at the menu (level 30 gives more life).
- Leg 1 in ONE god run (H4), `--bg` because it takes up to 40 min:
  `L --bg --max 3000 lr1 act2live god 2400 wp 46 -- clear -- exitto 69 -- probefar 10 -- tpto 22519 9806 -- tpto 22519 9806 -- place hst 152`
  [V sub names in tools/rigs/act2live/main.go: `god SECS`, `wp LEVEL`, `clear`, `exitto LEVEL`, `probefar [N]`, `tpto X Y`, `place CODE CLASS`; there is no `exit` sub (act2.md's
  "exit 69" means the nav ExitTo, i.e. `exitto 69`)]. If `wp 46` fails with "level 46 not reached", use `wpgo 46 410` (V). If `exitto 69` does not change the area, replace it by the manual gate
  chain `goto 12528 5285 -- click 60 100 -- hold 3 -- click 340 175 -- hold 3 -- click 300 190 -- hold 3` (V act2.md H2; the nav gate-click fallback is U live).
- Expect: Canyon area 46 arrival about (12678,5178); area 69 arrival (22873,9597); `probe step N ... orifice: true`; `PlaceItem: <nil>` then `units` shows object 100 and
  object 152 mode changed; the Staff gone from the pack (`act2live state`).
- Second run (new god chain; the first ends and Navi is unprotected, so enter the portal inside one chain if the 40 min allow; else restart the chain from `wp 46`):
  `L --bg --max 1800 lr1 act2live god 1500 clickobj 100 -- hold 3 -- state -- kill 211 -- hold 2 -- units 251 -- talk 251` [V subs; Duriel class 211 V boss/act2.go; Tyrael 251 S].
  Expect area 73 (Duriel's Lair), no teleport by the bot (V syncTele off in 73), Duriel life readings vs table (4757, cold resist 50), `Tyrael` unit appears after the kill, talk
  opens a menu; quest word 14 done (check with `d2cheat qdump` after `oog leave`).
- Pass: staff placed, area 73 reached. Stretch: Duriel dead and Tyrael talked. Fail: record the first failing step with `tools/shot.sh` and the log line; do not retry more than once per game.
- Leave: after Tyrael the portal leads to the Harem/town; walk or `wp 40` to Lut Gholein; check area 40; `key esc`. Record: docs/act2.md "Live results 7" (T1 to T3, D1, D2).

## R8. Summoner (B2, i1 Navi) - INDEX P1 10
- Precondition: Navi in Lut Gholein; Palace gate speeches done (V X4: Drognan and Jerhyn spoke) else the guards say "You may not pass".
- Chain (god, bg): `L --bg --max 2700 lr1 act2live god 2400 exitto 50 -- exitto 51 -- exitto 52 -- exitto 53 -- exitto 54 -- clickobj 298 -- hold 3 -- state -- summoner`
  [V: `exitto 50..54` took about 3.5 min under god; `clickobj 298` enters the Arcane portal (V Y2); `summoner [W E N S]` teleports blind along the arms (V sub), U result]. Standing
  next to portal 298 first: `obj 298` hops to the preset (V X6: the unit is not streamed until about 80 subtiles away).
- Expect: area 74 arrival (25428,5428); Summoner class 250 listed; kill; the journal object (class 357, preset offset (-3,-3) vs where he stands, S) click opens a red portal (object 60) to the Canyon 46.
- Pass: Summoner found (positions of preset 357 and boss recorded) and dead, or at least found. Known weak spot (V Y3/Y4): void gaps stop blind casts; if `summoner` finds nothing in 10 min stop and record where each arm stopped (`map 6`).
- Leave: `wp` back or portal; town; `key esc`. Record: docs/act2.md live results 7 (S2).

## R9. Remaining Act 2 rows (B2, i1 Navi) - INDEX P1 9
- F2 Claw Viper altar and `vip`, F3 Maggot chest `msf`, C1 cube run, R2 Radament reward, F1 Halls chest `box`. First decide which are already done from Navi's quest words:
  `ssh worker.example` copy + `d2cheat qdump` (3.4); Navi words 9..12 are set (0x801e, 0x0800, 0x8005, 0x01bc, V qdump) so some rows are done: do only the undone ones
  (the act2.md table gives the row: area 61 altar 149 for F2, area 64 chest for F3, area 60 chest for F1).
- Per row: `L --bg --max 1500 lr1 act2live god 1200 go <areas...> -- obj <class> -- pick` with the level ids from the table (`go A1 A2..` = ToArea each; V sub). Cube run: `L lr1 act2live cubemix msf vip`
  [V sub: opens the cube, moves one pack item per code, clicks Transmute, lists the result]; screenshot the panel with `tools/shot.sh` (grid origin, Transmute button).
- Expect: `hst` in the cube result, both inputs gone. Record the 3x4 grid origin and button position (act2.md C1).
- Record: docs/act2.md live results 7 (F1 to F3, C1, R2); INDEX P1 9.

## R10. NPC menus, merc rows, mercdump (B3, i3 Towni) - INDEX P1 11, 12 and 24b mercdump
- Setup: MAIN brings up i3, logs in Towni (Sorc 20, all waypoints, Act 1, V). Gold: `L lr3 d2cheat show`; if < 5000 `L lr3 d2cheat --reason "R10 merc hire needs gold" --actor lr3 gold 20000`
  (only if the stat exists; else save-edit with the game at the menu, 3.3).
- Act 1: `L lr3 townlive units` (classes in town), `L lr3 townlive menu 148` (Akara), `menu 154` (Charsi), `menu 150` (Kashya), `menu 147` (Gheed) [V townlive `menu CLASS [hold=S]`; classes V monstats hcIdx]; screenshot each dialog
  (`tools/shot.sh /tmp/lr3-akara.png i3`). Hire window: Kashya is class 150 (V monstats), `L lr3 townlive menu 150` then `L lr3 townlive hire ROW` for ROW 0..2 [V sub `hire ROW [AURA]`: Town.Hire in the current act, Normal] and read the
  `merc` lines; run `L --bg --max 200 lr3 probes mercdump -sec 120` in parallel with `--nolock` while you click (U lock-free read is harmless).
- Acts 2 to 5 NPC menus: leave the game at the menu and `L lr3 d2cheat --reason "R10 act n town NPC menus" --actor lr3 act Towni 2` then `oog "start Towni"`; repeat for 3, 4, 5 (about 30 s each; the game
  process is not restarted). In each: `townlive units`, `menu CLASS` for Drognan/Jerhyn/Cain (A2), Ormus/Alkor/Hratli/Asheara/Cain (A3), Tyrael/Jamella/Halbu/Cain (A4), Larzuk/Malah/Anya/Qual-Kehk (A5)
  (classes V monstats hcIdx: Atma 176, Drognan 177, Fara 178, Greiz 198, Elzix 199, Jerhyn 201, Lysander 202, Meshif 210 and 264, Cain 244/245/246/265, Tyrael 251/367/521, Asheara 252, Hratli 253, Alkor 254, Ormus 255, Halbu 257, Jamella 405, Larzuk 511, Malah 513, Qual-Kehk 515). The hire window of Greiz (A2), Asheara (A3), Qual-Kehk (A5): `townlive hire ROW AURA`.
- Expect: menu row labels per NPC (Talk / Trade / Cancel plus quest topics), hire rows with aura names; `mercdump` shows `present`, `class` 271 (Rogue) / A2 class / etc., `ui_open` flag 0x08 with the menu.
- Pass: for each NPC the row list is recorded and either matches `quest/act*_menus.go` or the diff is listed. Fail items: far NPCs that need the click-on-body fallback (V X5/Y6).
- Leave: any town, panels closed. Record: actbot.md checklist 1, 2 and progression/verification.md V-04, V-09, merc-farm.md; INDEX 11, 12.

## R11. Act 1 chain by the bot, and held-goal resume (B4, i4) - INDEX P0 5, 13b
- No extra instance: the soak (R1) runs `--goals campaign`, i.e. the Act 1 task chain. Mine its bot log instead of a new run:
  `tools/jobs.sh list soak24` then `tools/jobs.sh tail soak24/<ts>-d2bot 200 | grep -E 'task|quest|Den|Blood Raven|Cain|Smith|Andariel|Tristram|Warriv'` (the gate run's agent is `livebot`, `tools/jobs.sh tail livebot/20261029-104840-d2bot 200`).
- Pass: lines for Den, Blood Raven, Tristram stones/portal, Cain, Smith imbue, Andariel, Warriv "Go East" in order, no fatal. Gaps become the next work list (task.md L1 to L7).
- 13b (held goal resumes on a quest flag flip, persists over a restart): after the 24 h soak, on i4 Booti only: `d2bot --attach` with a goal held by a clvl lock (docs/progression/results.md once run), then
  restart d2bot (not the game) and confirm the hold level persists. Command shape: `L --bg --max 900 lr4 d2bot --attach --addr 127.0.0.1:8431 --goals 'reach:3' --autostart --run-for 10m` (port other than the
  soak's 8420, V default in d2bot flags) - U exact goal spec for a lock; read internal/goals before running.
- Record: task.md, progression/results.md, INDEX P0 5, 13b.

## R12. Druid and Assassin (B5, blocked) - INDEX P3 19, 20
- Blocker: Druki (Dru 4) and Assi (Ass 4) are not on the cheat allow-list (liveruns.md s1); levelling from 4 to 16 and 12 honestly takes hours (druidlive reached clvl 4 in 48 min at load 20, V summon.md).
  Ask the user to approve adding them to `~/.config/d2lod/cheat-allow.txt` on the worker host; do not add them yourself.
- Once allowed (skill ids V from skills.tsv: Raven 221, Oak Sage 226, Eruption 234, Cyclone Armor 235, Wake of Fire Sentry 262, Lightning Sentry 271, Death Sentry 276, Shadow Warrior 268; prerequisites and
  clvl: Eruption 12, Cyclone Armor 12 (Arctic Blast), Lightning Sentry 24, Death Sentry 30, Shadow Warrior 18): same pattern as R6 with `save Druki level=18 skills=...` and `druidlive pets` / `druidlive buff` (V subs; agent
  `lr2` on i2), and for Assi on i3 (`D2_INSTANCE=i3`) `d2bot --attach` fights plus the `trap` log lines (summon.md W5/W6 items 3 to 5).
- Record: summon.md live results, INDEX 19, 20.
