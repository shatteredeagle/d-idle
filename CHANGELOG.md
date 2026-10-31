# Changelog

Every release needs notes here. Newest first; one section per day, grouped by area. There are no tagged releases yet: the
units are the merge commits on `master` (`git log --first-parent`), named by their agent branch (`agent/<name>`).

Marks: **live** = a live result is named in the merge subject or in `docs/` (see `docs/progression/results.md`, `docs/INDEX.md`
section 1); **offline** = unit tests, simulation or fake memory only; **research** = documents only. When a line says nothing
else, it is offline-tested. This file was written from merge subjects and `docs/INDEX.md`; it does not claim more than they do.

## Unreleased / known issues

- Container dashboard: fleet slots are server instances (fixes 'No Instances' on the NAS), per-slot reverse proxy to the current d2bot address, real fleet status from the bots, per-slot view-only game video (`/api/v1/instances/{id}/viewport/stream`, `xwd` capture, no new package) and a Game view card with reconnect. Offline-tested (agent/fleetview).

- Container soak: `d2soak sidecar` (D2FLEET_SOAK=1) judges each fleet slot inside the d2idle container; `tools/soak/contsoak.sh`, docs/nas-container.md. Offline-tested.

- Repo diet (ARCHIVE): `cmd/spike` deleted; eight one-off live rigs moved to `tools/rigs/` (`//go:build rigs`, `tools/live.sh` still finds them); stale planning docs moved to `docs/archive/` (see its README); `tools/ci/doclinks.py` checks doc links.

Source: `docs/INDEX.md` sections 1, 2 and 4 (as of 2026-10-29), `docs/BACKLOG.md`.

- No 2 h or 24 h soak has ever completed; the longest unattended run is about 1 h 15 (a 90+ min managed run is recorded in `docs/launch.md`). The 24 h runbook, `soak24.sh`, `m24` profile, `d2bot --supervise`, panic recovery and `d2doctor` have never run live.
- Acts 3, 4 and 5, Nightmare and Hell: code and simulations only; no step has been played by the bot. Immunity, affix and missile reads for Nightmare/Hell are unverified.
- Act 1 is not played through end to end by the bot: Blood Raven, Tristram chain, Cain rescue, Smith and Andariel are unproven live. Smith was never found live (the preset 754 anchor is unverified); Andariel is gated off for lack of potions.
- Act 2: staff placement, tomb 73, Duriel, Tyrael and the Summoner kill are unproven (the Arcane maze is not explorable by nav alone); the real Act 1 to 2 hop was never walked.
- Mercenary: revive, equip and farming runs are unverified; hire rows were probed live only for Kashya and Greiz.
- Classes: Barbarian has no live run; Druid and Assassin reached only clvl 4; Amazon ammo-low and empty-quiver paths not run; Necromancer revive, Corpse Explosion and Weaken/Lower Resist state ids are open live checks.
- Fleet and container: the stop chain (`docker stop` to Save and Exit within the 70 s grace), the 4-slot hour, N games per container and the setup wizard with a stranger's files are untested live.
- Stream: Kick OAuth title flow and Home Assistant deployment not run live.
- Idle-game layer (upgrades, prestige) is parked by the user until the bot plays well.
- Review fixes: all High findings of `docs/archive/review-overnight.md` (H1 to H3), `docs/archive/review-day.md` (H1 to H5) and `docs/archive/review-evening.md` (H1, H2) are marked fixed offline in INDEX section 4 (source S: read from commit messages, none re-run live). Still open per INDEX: evening M10 (resist penalty possibly applied twice in `rules.go`, `hard.go`, `d2/player.go`) and some Lows (for example `soak24.sh --for 1h30m` arithmetic) have no fixing commit; the review-day H3 chicken-loop fix is not re-checked against a Diablo or Baal attempt. The fixes for review-day H4 and evening H1 and H2 are live-untested.
- The four merge-risk reviews (overnight, day, evening, night) are archived in `docs/archive/`; their open findings, fix commits and live checks are one table: `docs/open-findings.md`.
- The full prioritised list of open live checks is `docs/INDEX.md` section 2 (P0 gates the alpha: 2 h soak, then 24 h; Andariel; Summoner; merc; quest events; Larzuk/Charsi; Nightmare/Hell probes; cube chore; farming runs; fleet; wizard).

## 2026-10-29

### Bot core and failsafe
- `agent/panicsafe`: d2bot recovers from panics instead of dying.
- `agent/logclean`: log deduplication and a one-line minute summary; supervise and quest polling lines kept.
- `agent/nopause`: the bot no longer pauses itself on low life; the failsafe auto-resumes; the planner no longer picks far-ahead goals.
- `agent/restartfix`: a planned restart counts only once against the failsafe budget.
- `agent/chickfatal`: an unplanned game loss restarts in managed mode; `Flow.BeforeSave` hook.
- `agent/stallfatal`: a rung-4 restart is no longer fatal; clearing idle limit; first stall rung skips the target.
- `agent/loopfix`: blocked inserted steps are dropped; a fruitless explore gives up.
- `agent/botgaps`: stall recoverer, GuardSave, NearTown, Frenzy pair, far NPC.
- `agent/harden`, `agent/harden2`: fixes for the overnight review (H1-H3, M1, M2, M4) and the day review (H1-H3, M3, M6 to M8).
- `agent/lows`: d2bot shuts down saving first, log rotation, flee chicken, guard applies to allow-listed characters only.
- `agent/race`: Recycler lock, no nested ladder escalation, clean `-race` sweep.
- `agent/chickloop`: heal-first gate in town, waypoint to town, pause on re-entry loops.
- `agent/stopchain`: `docker stop` / SIGTERM goes through Save and Exit within the fleet grace; Save happens before the runner is cancelled (live untested).
- `agent/gatewire`, `agent/fuzz`, `agent/coverage`: entry gates wired; fuzz and new tests for unattended-run paths, which found and fixed a grid area cap, a `SetKnob` NaN and a nil snapshot in `ToTown` (direct commit a305319).
- `agent/integrate2` (checkpoint): quest memory and boss adapter wired, game recycler between goals, leveler pause with a visible reason, class-aware start.
- `agent/doctor`: `d2doctor` health checks (config, paths, tables, helper, token permissions, load, clock, difficulty line).
- `agent/chainfix`: the resist gate is rechecked at campaign start; MaxDeaths auto-resume.

### Classes and combat
- `agent/friendly`, `agent/friendly2`, `agent/flive`: `IsFriendly` for every class (pets, mercs); callers use it; rest loop and hostile-near fixes. Live friendly results, `cmd/friendlive`.
- `agent/summon`, `agent/necrolive`: Necromancer summon upkeep, curses, Bone Armor by state. Live results on Necki (Bone Armor state 14, skeleton cap, golem), mage hotkey, army skips rest.
- `agent/amalive`: Amazon live results (19 kills), shop stock pages fix, ammo bought before shopping. Live.
- `agent/assalive`, `agent/druidlive`, `agent/druidassa`: Assassin and Druid builds, profiles, pet and trap upkeep; live runs of 20 to 26 min to clvl 4 only.
- `agent/palalive`, `agent/barblive`: Paladin live results (Palli clvl 13, 63 kills); Barbarian build and profile manaReserve with live results recorded in docs (INDEX still lists no Barbarian live run: see Unreleased).
- `agent/classcombat`: Necromancer corpse skills (Revive, Corpse Explosion), assassin trap planner.
- `agent/casterfix`: caster staff swing on weak packs, nearTown, Den kit floor.
- `agent/fightwire`, `agent/missiles`: curse danger and retreat, missile and curse readers, sidestep hook, fight telemetry (reads unverified).
- `agent/affix`: monster flag reader, affix resists, danger rules, entry gate (flag layout unverified).
- `agent/gear`, `agent/gearwire`, `agent/armordata`, `agent/scorewire`: weapon and ammo model, build tables, armor data, leveler gear scorer uses resist/FCR/ethereal helpers.
- `agent/endbuild`, `agent/builds80`: endgame build tables for all seven classes registered opt-in, Smite weapon fix, ResistGate.
- `agent/hellfix`: mana-burn chicken and hard-tick order fixes.
- `agent/diffp2`, `agent/wirefarm`: Nightmare/Hell rules in the fight loop (Normal unchanged), tuning, death telemetry, `runs:NAME:N` goals.

### Town, loot and items
- `agent/townfix`: greeting stall, affordability-aware advice, futile-trip memory, jobs kill, ring retry (from live soak findings).
- `agent/scrollfirst`: buy the town portal scroll before potions.
- `agent/itempick`: runeword bases, gem keep, charm filter, gear score helpers.
- `agent/cubejobs`: cube chore (gem and rune upgrades), knob `Bot.CubeJobs`, default off (simulator only).
- `agent/stashfull`: full stash and pack handling; `SetBuild` forwards to the judge.
- `agent/lootpotion`: potion pickup uses the build-aware belt plan.
- `agent/yieldwire`: nav Yield wired to a bounded pack clear.
- `agent/mercread`, `agent/wireup`: merc reader and merc chore, EntryGate in the planner, NM Act 1 floor 36 (hire rows were probed live earlier).

### Navigation
- `agent/navstall`: LevelTo guard fix, waypoint activation in passing.
- `agent/tombfind`, `agent/tombhint`, `agent/tombnav`: waypoint icon column, tomb level hint, far probe, gate click fallback, waypoint load wait (partly live in Act 2 runs).
- `agent/teleexplore`, `agent/arcane`: teleport-aware explorer, island-hopping teleport planner for the Arcane Sanctuary.
- `agent/portalnav`: `openPortal` on a class-aware grid, exit-tile cost (live Act 2 context).
- Direct commit f5b7197: hop from Burial Grounds and Cave Level 1 via Cold Plains, found in a live soak.
- `agent/wpcheat`: the waypoints cheat opens act tabs via quest words. Live verified.

### Quests and acts
- `agent/earlyfix`: Den of Evil quest word gating, walk home without scroll, Akara greeting, area hysteresis (the Den claim itself was live-verified on 2026-10-28).
- `agent/act2live`, `act2live2` to `act2live6`: Act 2 live results 1 to 6 (Radament, Book of Skill, cube Transmute of staff pieces, Arcane portal, waypoint rows, Canyon, real tomb found). Live, Navi.
- `agent/act2code`: real Transmute, quest items stay in pack, TakeWaypoint walk, Act 2 stall recovery.
- `agent/act3plan`, `agent/act3data`, `agent/act45data`: Act 3 to 5 plan docs, quest data, task nodes, boss scripts, nav hints.
- `agent/actwire`, `agent/actbot`, `agent/act45bot`: campaign templates, Acts 2 to 5 wired into d2bot, `normal_all` covers Acts 1 to 5.
- `agent/act1gaps`: Act 1 quest objects (tree, cairn stones, cages), Akara and Charsi rows.
- `agent/nmcampaign`, `agent/nextdiff`, `agent/nmsim`, `agent/questevent`: Nightmare and Hell campaigns chain after completion; manager-level chain simulation; held goals persist; quest events resume held goals.

### Bosses
- `agent/bosslive`, `agent/bosslive2`: Countess approach, locate, fight and loot under god mode, preset anchor fix, critical-life escape, `bossrun --god/--nokite`. Live, Combi, god mode.
- `agent/bosswire`, `agent/bossrun`: boss script robustness, potion gates, `cmd/bossrun` single-script runner.
- `agent/bossfix`: pinned-kite fix, cursed retreat, weak-pack strictness, Smith preset anchor, belt-aware potion floor (pinned-kite fix not re-run live).
- `agent/sealsim`, `agent/sealfix`: fight-loop simulation of seals and Baal waves; seal boss timeouts continue and resume from the nearest node, death checks.

### Resource and poverty
- `agent/manapoverty`, `agent/poverty`: energy rows, cheap rest, poverty advice, flee without potions, poverty mode, resource knobs.
- `agent/danger`: poverty pack-danger limit, act-aware starveArea.
- `agent/earlysorc`, `agent/levelrate`: free heal trips from the Den, caster starvation, planner floor so low areas are not fallback farming at high level.
- `agent/holdrun`: locked runs are held; town belt by build.

### Fleet, container and setup
- `agent/slotgame`, `agent/botlaunch`: bot-owned game per fleet slot (fixes the Yama sibling problem); fleet launches and supervises `d2bot` per slot.
- `agent/fleetfix`: 70 s stop grace, goals marker after enqueue, StopGame, slot instance env (live untested).
- `agent/image`: container image carries `d2fleet` and `d2doctor`, wizard default CMD.
- `agent/wizard`, `agent/wizardwire`, `agent/wizui`, `agent/wizgoals`: setup wizard API mounted in the server, wired into `d2fleet`/`d2dash`, goals picker, container user-test checklist.
- `agent/reapfix`, `agent/reapgo`, `agent/winefix`, `agent/wineleak`: Wine reaping with layout-aware prefixes, safe winedevice reaping in Go at managed stop, wineserver fd investigation (steps, not a leak). Shell dry run only.
- `agent/helperleak`, `agent/helperleak2`: all Win32 calls in the d2input helper on one locked OS thread, thread heartbeat.
- `agent/soakpid`: per-instance soak process matching.

### Soak and tooling
- `agent/soaktools`, `agent/soak24`, `agent/logparse`, `agent/dashwire`, `agent/dashalerts`: soak report level, status and alerts files, 24 h runbook and `m24` profile, JSON-log fight/death parsers, `--soak-dir` and dashboard alert strip (never run live).
- `agent/probes`: prebuilt live probe tools (runewords, monflags, mercdump, missiledump, wpdump); never run.
- `agent/viewhang`: d2watch per-client stream writer, reconnect on load, hidden-tab pause.
- `agent/configdoc`: `docs/knobs.md` reference of tuning knobs, with a generator.
- Cheat allow-list gains Necki; Palli, Palli2, Amazi documented (user approved).

### Docs and research
- Reviews: `agent/review` (overnight), `agent/review2` (day), `agent/review3` (evening) merge-risk reviews.
- Research and doc upkeep: `agent/mercresearch`, `agent/diffresearch`, `agent/itemresearch`, `agent/docindex`, `agent/docsync` (INDEX and PLAN status refresh), backlog refreshes for the dispatcher cron.

## 2026-10-28

### Bot core and failsafe
- `agent/bot`, `agent/core`, `agent/wirebot`: bot runner, death policy, profiles, `cmd/d2bot`; verified death handling and deflaked failsafe tests; d2bot on the unified server. First live runs.
- `agent/sentinel`: out-of-process failsafe sentinel with rules and drills; ignores quest-log button flag 0x11 (direct commit). Live drills.
- `agent/livebot`: managed mode live without setcap, death and IsDead fixes, soak findings. Live.
- `agent/integrate`: resource policy, task layer and quests wired into the bot; a live 31 minute campaign run to clvl 6. Live.
- `agent/infra`: the d2input helper died after about 2000 commands (callback table overflow); fixed, with an exit and crash log and a self-healing client; `tools/helper.sh` is per-instance.
- `agent/ci`: CI workflow, hygiene checks, release tarball script.
- `agent/leakhunt`, `agent/soakfix`: fd study (no real leak), game recycler, soak warm-up and fd plateau.
- `agent/state`, `agent/event`, `agent/obs`: state and event packages, metrics registry, soak harness.

### Classes and combat
- `agent/combat`, `agent/combat2`: fighter, targeting, skills and hotkeys, death detection; verified death data, unreachable targets, chicken. Live.
- `agent/combatclass`, `agent/levelerclass`: data-driven class combat, skill binder, aura upkeep, Paladin melee loop, class builds, skill-id read-back, gear score, weapon swap.
- `agent/classes`, `agent/classpal`, `agent/classnec`, `agent/classama`: class plans and ranking; Paladin, Necromancer and Amazon smoke test findings, `cmd/classsmoke`. Live smoke runs.
- `agent/leveler`: stat and skill spending, equipment, level-run planner.

### Town, loot and items
- `agent/town`, `agent/town2`: town chores, rebuy, loot pickit, valuables, live buying and portals, seed-independent town. Live.
- `agent/townfix`: Akara Den claim fixed, full-pack purchase fix. Live-verified.

### Navigation
- `agent/map`, `agent/nav`, `agent/nav2`, `agent/navfix`, `agent/navfix2`: world-map reader (town verified live), A* navigation and exploration, seed independence, waypoints, dungeon transitions verified 20 consecutive times, re-plan on level change. Live.

### Quests and acts
- `agent/quests`, `agent/memquest`, `agent/liveq`: quest state reader, NPC dialog primitive, act hops; in-memory quest and waypoint source; live verification of quest, waypoint, act and difficulty save state. Live.
- `agent/cheatq`: quest, waypoint, act and difficulty save edits via the logged cheat tool. Live.
- `agent/act2`: Normal Act 2 campaign, cube contract, Radament, Summoner and Duriel scripts.
- `agent/tasklayer`: task layer (step graph, planner, executor, simulation).

### Bosses
- `agent/bossruns`: Act 1 boss script framework and scripts for Blood Raven, Countess, Smith, Andariel (live results arrive 2026-10-29).

### Resource and poverty
- `agent/resource`: pure mana and health resource policy package.

### Fleet, container and setup
- `agent/fleet`, `agent/serverfix`, `agent/server`, `agent/wire`, `agent/dash`: fleet supervisor and party orchestrator, unified instance-scoped HTTP server, dashboard UI, `d2dash` and `d2sentinel` on it.
- `agent/container`, `agent/containergame`: headless container spike, then the real game and bot run headless in the container under plain Wine. Live in the container.
- `agent/setup`, `agent/wizardui`: setup wizard backend, `d2doctor`, wizard front-end (mock-tested).
- `agent/multi`, `agent/launch`, `agent/lifecycle`, `agent/memfd`, `agent/oog`, `agent/partylive`: per-instance prefixes and Xvfb displays, subreaper launcher, game lifecycle in Go, fd passing under Yama, verified out-of-game flows, LAN host and join over loopback. Live.
- `agent/input`, `agent/mem`, `agent/data`, `agent/saves`, `agent/replay`, `agent/data2`: helper protocol v2, typed memory readers, MPQ reader and tables, save snapshots, fixture and canary tooling (live or real tables).
- `agent/paths`: data-path migration with legacy fallbacks.

### Soak and tooling
- `agent/cheat`: logged test cheats, cheat ledger, `d2cheat` (throwaway characters only).
- `agent/watch`, `agent/viewport`, `agent/display`: `d2watch` live viewer, capture and stream library, status hub. Live display.
- `agent/moveboard`, agent status board and hub, `live.sh --bg`, `jobs.sh`, `devhealth.sh`, mute watcher (tooling).
- Stream: `agent/streamctl`, `streamha`, `streamfx`, `selsync`, `scenepage`, `scenestream`, `streamdesk`, `streamgames`, `kicktitle`, `kickapi`: stream control service and scheduler, Home Assistant package, overlay, scene page, tray app, Kick title library. The Kick scene ran live; the OAuth title flow did not. Direct commits add the Kick streamer (Radeon h264_vaapi) and an ingest URL fix noted as verified live.
- Public page: `agent/publish`, `publish2`, `publish3`: public status page generator with fail-closed sanitizer, D.idle name and footer, gated GitHub Pages push, live-now link.
- `agent/milestones`, `agent/milestones2`: alpha, beta and good-bot definitions and tracker (docs).

### Docs and research
- `agent/plan`, `agent/goalcat`, `agent/goals`: leveling strategy, goal catalog, goal manager, API and embedded dashboard (simulation runner).
- `agent/progdesign`, `agent/prior`, `agent/priorart2`, `agent/net`, `agent/review`, `agent/hacheck`: progression design, prior-art specs, multiplayer feasibility, plan review with gotchas and roadmap, rebuilt HA stream dashboard.
- Project skeleton (88d22fb): memory-read probe, input helper, memory reference, plan and agent rules. Agent rules cover cheats, muting, disk policy, commit-often and the four project agent types.
