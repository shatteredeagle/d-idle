# D.idle

_Say it: die dull. Status page._

A hobby project that turns Diablo II: Lord of Destruction into an idle game. A bot plays the real game by itself, offline, and a layer of goals, routines and progression is being built around it, the way an idle game gives you things to unlock and decisions to check in on.  
It is early days. Right now the work is on the part underneath: a bot that plays reliably for hours with nobody watching.  

_Streams when the bot is running._

## Between runs

No bot is running right now. The last run of each:

- **Loopy**, Sorceress, d2d: level 4, 7,700 XP, 334 kills, 0 deaths, 13 potions drunk, run time 0h 05m. Now: playing through the early game in Rogue Encampment, ended. Last update 2026-10-31 00:10 UTC.
- **Soaky**, Sorceress, d2lod (original game): level 15, 195,140 XP, 7,100 kills, 4 deaths, 203 potions drunk, run time 8h 05m. Now: levelling up toward level 22 in Stony Field, ended. Last update 2026-10-30 20:26 UTC.
- **Fresha**, Sorceress, d2lod (original game): level 11, 75,549 XP, 2,387 kills, 3 deaths, 121 potions drunk, run time 4h 04m. Now: levelling up toward level 20 in Stony Field, ended. Last update 2026-10-30 20:12 UTC.

## Now building on d2d

- d2d is an open-source reimplementation of the Diablo II engine. It lets the bot run headless (no screen) and far faster than real time. Measured with the bot attached: 22 to 33 times faster.
- On 2026-10-30 we decided to make d2d the main way the bot runs. The original-game path is frozen: it keeps working, and it stays as the reference we compare against.
- Progress so far: a fresh Sorceress reaches level 5 in about 3 to 4 hours of game time, which takes minutes of real time. The bot's loop on d2d is still being tuned.
- Goal, not built yet: the bot and d2d packaged in one container, so a friend can start it with a single command.

d2d is the work of the d2decomped project (GPL-3.0). Our fixes to it go upstream as open pull requests. [d2decomped](https://github.com/psi29a/d2decomped), pull requests [#127](https://github.com/psi29a/d2decomped/pull/127) and [#128](https://github.com/psi29a/d2decomped/pull/128).

## Why we are making it

- We like Diablo II and we like idle games. This is what happens when you put them together: the game plays itself, and the fun is in the setup, the progress and watching it go.
- A bot that can be left alone is a satisfying engineering problem. It has to cope with getting stuck, with danger and with dying, without a person stepping in.
- The plan is the bot first and the game second. Once the bot is dependable, the idle-game layer goes on top: things to unlock, runs that reset for a bonus, rewards, and parties of bots.
- If it turns out the bot can be flexible, the game around it can be flexible too. If not, a very good bot that the game layer deliberately limits is still a fine foundation.

## What is different about it

- A game around the bot. Most Diablo II bots are tools for farming. Here the bot is the engine and the game layer (goals, routines, progression, a dashboard) is part of the design.
- Built to be left alone. Several layers of safety catch a stuck bot, a dead character or a crashed game, and a separate watchdog process steps in when the bot cannot help itself.
- Offline and on the original game. It never connects to Blizzard's online service, never changes the game's files, and nothing from Blizzard is included. You use your own copy.
- Quiet on your computer. It reads the game from the outside and sends keys and clicks only to the game window, so you can keep using your machine. Many copies of the game can run at once.
- As far as we have found, nobody has layered a game like this over an unmodified classic PC game. The closest projects, such as bots for mobile games and the World of Warcraft playerbot servers, share parts of the idea.

## How it works

- It looks at what is happening in the game (health, position, items), the way you look at the screen.
- It presses keys and moves the mouse like a person would. The bot never edits the game while it plays.
- Our test tools can edit throwaway characters to set up a situation, and every one of those edits is logged.
- A separate watchdog notices when it is stuck or in danger and steps in.
- The game layer decides what the bot should be working toward, and the bot works out how.

## Where we are

Each stage is a gate made of milestones. Open a stage to see its milestones.

<details open>
<summary><b>Alpha: a reliable Act 1 bot on the product path</b>: ████░░░░░░ 47%</summary>

| Milestone | Progress | Checks verified |
| --- | --- | --- |
| M0 Foundations: the product path runs live | ██████░░░░ 64% | 2 of 7 |
| A Act 1 self-leveler: clvl 1 to 15, 2 h unattended | █████░░░░░ 50% | 2 of 11 |
| B Normal Act 1 complete: quests, Andariel, hop to Act 2 | ████░░░░░░ 40% | 1 of 11 |
| G1 Day-long soak on the product path (24 h) | ███░░░░░░░ 37% | 1 of 8 |

</details>

<details>
<summary><b>Beta: Normal complete, installable by a friend</b>: ███░░░░░░░ 30%</summary>

| Milestone | Progress | Checks verified |
| --- | --- | --- |
| C1 Normal Acts 2 and 3 | ░░░░░░░░░░ 8% | 0 of 6 |
| C2 Normal Acts 4 and 5, Baal, hop to Nightmare | ░░░░░░░░░░ 0% | 0 of 5 |
| F Headless container and setup wizard, usable by a friend | ████░░░░░░ 41% | 1 of 6 |
| G2 Multi-day soak (72 h) | ░░░░░░░░░░ 0% | 0 of 4 |

</details>

<details>
<summary><b>Good-bot gate: the game layer may start</b>: ██░░░░░░░░ 26%</summary>

| Milestone | Progress | Checks verified |
| --- | --- | --- |
| D1 Nightmare playthrough | ░░░░░░░░░░ 0% | 0 of 4 |
| D2 Hell playthrough and farming runs | █░░░░░░░░░ 12% | 0 of 4 |
| H Good-bot gate | ████░░░░░░ 40% | 2 of 10 |

</details>

## Right now

Between runs.

**Being worked on:**

- Long unattended test runs (working)
- Working on the bot (working)
- Playing in a party of bots (working)
- Running the bot on the d2d engine (blocked)

## What shipped

<details open>
<summary><b>2026-10-30</b> (51)</summary>

- Public scrub package C.
- Public scrub package A; cutover executed.
- Lut Gholein trace.
- FreshGameToLevel reset after NewGame, run A/B.
- The bot drinks potions between town-portal tries and no longer sweeps into a camped spot.
- ADR for the d2d primary backend decision.
- Fixed tool hot loop in serveStream.
- Survey of Diablo II reimplementation/decompilation projects.

<details>
<summary>43 more</summary>

- Tool fit report.
- Lab and lab2 sections.
- Sourced cast timing documentation.
- First-contact greeting no longer burns hover tries, fresh position per click, 3 s interact budget.
- D2d lab notes.
- Spec approach evaluation and notes audit.
- OpenSpec SME reference and fit notes.
- Link-wait bound fix, arrival diagnostics, d2d liveness Beat, one-line unknown-field summary.
- Combat on d2d, 51 kills 0 deaths live.
- Per-instance long-run test layout so two soaks run at once.
- Rejuv cap 3 across belt+pack, belt move, boss drink threshold.
- Cap rejuvenation potions, sell surplus, inventory contents log.
- One NPC visit per trip, interact within 4 cells, no needless game screens close.
- D2d town verbs, tool adapter, tool smoke tool, Akara trade verified live.
- Replicated d2d long-run test test rig, stats, pass rule.
- Tool --backend d2d, internal/tool, seams for combat/town.
- D2d container design, draft recipe, GPL notes.
- Autoloot/autoparty off by default in d2d launches, death-gold recheck, d2d-gaps.
- Internal/tool nav world source from d2d verbs, tool.
- Recovery farms the best XP-per-kill safe area.
- D2d statistical fidelity results.
- No_gain counts XP progress.
- Bounded nav field/search, heap guard, heap_runaway alert, memory cgroup recommendation.
- Internal/tool phase 1 read-only d2d snapshot source, tool.
- Den floor per class, Prayer resting aura.
- Skip Imbue, Monastery route guard, escalating repeat-site bans, crowd panic rule, autopsy and potion counter fixes.
- Fresh-character build tables, SelectBuild, Nova floor.
- Blood Raven is not retried from Act 2.
- Internal/tool control-channel client, fake server, tool CLI.
- Blood Raven kit gate, attempts gated on kit and gold, claim-step stub, no Act 1 paid hire.
- Early-game audit fixes.
- D2d backend design.
- Sweep the death-spot gold pile, count it in the corpse decision, bank-trip gap fix.
- Second tool findings pass.
- Tool research findings.
- Death rule checks in tool.
- Repeat-site ban, planner dump, corpse floor, potion bug tag.
- StaffPlaced uses the persisted quest word.
- Death metrics from real long-run test logs.
- No_gain, loop_no_gain, idle_in_town, below_floor_clvl, time-in-state report.
- Short nav skips, farm instead of idling, skip reason logging.
- Planning-session input.
- NPC menu re-observation, optional quest step skip, interaction logging.

</details>

</details>

<details>
<summary><b>2026-10-29</b> (139)</summary>

- Cast-first combat loop, watchdog, Barracks kit gate.
- Quest-walk safety interrupts carried out, no pack kiting, death post-mortems.
- Retreat counts pack spares, belt purchase chore, mana pays casts.
- Early gear policy.
- No-area is a planning block, death streak forces recovery.
- SME pace tables and proposed long-run test milestones.
- Pack potions move to the belt.
- Spare potions sold, recovery loot, potion floor.

<details>
<summary>131 more</summary>

- Recovery phase, Blood Raven attempt budget.
- SME economy data and Blood Raven caster notes.
- Trip budget and gold floor.
- Stat floor for gear requirements, NeedStr from blocked rebuys.
- Rebuy rejection reasons, gate settles when blocked by stats.
- Bot vs veteran audit.
- SME per-class knowledge base.
- SME mechanics knowledge base.
- Keep and wear found gear for empty slots.
- SME play-patterns knowledge base.
- Rebuy bar fix, shop logging, quest skip reasons.
- SME builds and gear knowledge base.
- Gate refusals are planning blocks, route-aware targets, ban lift.
- Slots in instances interface, bot proxy, viewport video.
- Poor-mode pickup of sellable gear and small gold, per-fight loot summary.
- Gated character keeps its safe area, planning blocks never restart the game.
- Bwrap mount-namespace allowlist for every Wine process.
- Gear gate, repeated-death area ban, older-corpse retrieval, rebuy weapon/armor first.
- Periodic character point, planned games not counted as restarts.
- Heartbeat in recovery, instance lock guard, exit 3 final, GameWait 6 min, Den test race.
- Failsafe ownership table, soak24 watchdog/supervise preflight.
- Prune tool, rigs to tools/rigs, archive stale notes, open-findings table.
- Pre-fix gate baseline numbers.
- Mark character-on-area-change claims UNVERIFIED, results plan.
- Every build reports its commit, long-run test preflight refuses dev builds.
- SavedExp connected, long-run test --character, level table to 99.
- Unreliable bot tests made load-proof.
- FieldGate connected into trip.
- Xp lost/at-risk telemetry, long-run test XP from saved exp.
- No character churn on repeat quest events, stale hold level resumes at once.
- Thin-belt retreat and field gate, kit-safe PlanBuys.
- Soak24 duration parsing, winedevice sampler, tick-age/restart alerts.
- Boss readiness gates, Cold Plains kit floor, boss runs own the step.
- Kit reserve, off-class weapons, per-slot equip memory.
- Single equip slot table, bounded equip backoff, off-centre retry, cursor return.
- Live run runbooks and d2-live briefs.
- Pruned goal seeds stay refused across restarts.
- Manager and panic-recovery fixes.
- Seeded idempotent --goals adds.
- Release packaging, per-binary --version, dry run.
- Live check plan for Assassin/Druid.
- Single persisted hold record, cleared on every resume/pause/remove.
- Charsi imbue and Larzuk socket flows.
- End-to-end telemetry chain tests.
- Seal boss timeouts continue, resume from nearest node, death checks.
- Settings reference and generator.
- Long-run test fight/death parsers accept JSON logs.
- Resist gate rechecked at campaign start, MaxDeaths auto-resume.
- Full stash/pack handling, SetBuild forwards to judge.
- Log dedup, minute summary line.
- Manager-level chain simulation, persisted hold level.
- Tool health checks.
- Hold locked runs, town belt by build, long-run test report fight/death summaries.
- Go winedevice reaping at managed game stop.
- --long-run test-dir for tool/tool.
- Necro corpse skills, assassin trap planner.
- Endgame tables registered, Smite weapon fix, ResistGate.
- Full-campaign simulation tests, Act-aware planning fix, hop floors.
- Endgame build research and tables.
- Runs goals connected, Nightmare/Hell rules in fight loop.
- Container image carries tool/tool, wizard default CMD.
- Tool shutdown order, log rotation, flee chicken, guard for allow-listed only.
- Wineserver long-run test check by plateau/headroom, safe winedevice reaping.
- Wineserver fd investigation.
- New tests for unattended-run file locations, coverage notes.
- Pinned kite fix, cursed retreat, weak-pack strictness, Smith preset anchor, belt-aware potion floor.
- 70 s stop grace, goals marker after enqueue, StopGame, slot instance env.
- Countess live results, bossrun --god/--nokite, unpinned input helper.
- Dashboard long-run test alert strip.
- Act 1 quest objects, Akara/Charsi rows.
- Chain Nightmare/Hell campaign goals after completion.
- Acts 4 and 5 connected into tool, normal_all covers Acts 1-5.
- Tool per-client stream writer, reconnect on load, hidden-tab pause.
- Bot-owned game for fleet slots.
- Fleet launches and supervises tool per slot.
- Missile and curse readers, dodge rule, fight telemetry.
- Wizard goals picker.
- Caster staff swing on weak packs, nearTown, Den kit floor.
- Campaign templates, weapon equip, quest item pickup, World flags.
- Merc chore, EntryGate in planner, NM Act 1 floor 36.
- Leveler gear scorer uses resist/FCR/ethereal helpers.
- Free heal trips from the Den, NearTownArea, caster starvation.
- Runeword bases, gem keep, charm filter, gear score helpers.
- Gate click fallback, waypoint load wait.
- Act 3-5 plan notes and work list.
- Monster flag reader, affix resists, danger rules, entry gate.
- Difficulty survival research.
- Recycler lock, no nested ladder escalation, clean -race sweep.
- Merc and farming research.
- Poverty pack-danger limit, act-aware starveArea.
- No self-pause on low life, failsafe auto-resume, planner no far-ahead pick.
- Energy rows, cheap rest, poverty advice connected.
- LevelTo guard fix, waypoint activation in passing.
- Stall recoverer, GuardSave, NearTown, Frenzy pair, far NPC.
- Planner floor so Den/Blood Moor are not fallback farming at high level, fresh game for LevelTo.
- Barbarian build, profile manaReserve, live results.
- All Win32 calls on one locked OS thread in the tool input helper, thread heartbeat.
- Den quest word gating, walk-home without scroll, Akara greeting, area hysteresis.
- Paladin live results.
- StaffTombLevel reader, a2env TombHinter and ProbeFar.
- Act 2 live results 5.
- Heal-first gate in town, waypoint to town, re-entry loop pause.
- Island-hopping teleport planner, boss Teleporter.
- Act 2 live results 4, nav teleport in act2live rig, god restore on die.
- A planned restart counts once against the failsafe budget.
- Buy the town portal scroll before potions.
- Flee without potions, poverty mode, resource settings.
- Teleport-aware explorer, InteractNPC hover fallback.
- Act 2 live results 3, StallRecoverer rig, Arcane portal and waypoint verified.
- Blocked inserted steps dropped, fruitless explore gives up.
- Rung-4 restart is not fatal, clearing idle limit, skipTarget on first rung.
- Real Transmute, quest items stay in pack, TakeWaypoint walk, Act 2 stall recovery.
- Druid live results, tool.
- Act 2 staff pieces and Transmute verified live.
- Per-instance long-run test process matching.
- Greeting stall, affordability-aware advice, futile-trip memory, jobs kill, ring retry.
- Assassin live results.
- Druid and Assassin builds, profiles, pet and trap upkeep.
- Amazon live results, shop stock pages fix, ammo before shop.
- Necromancer live results, mage hotkey, army skips rest, curse dedupe, tool.
- Nav Yield connected to bounded pack clear.
- Act 2 live results, act2 nav hints, tool.
- Countess live results, preset anchor fix, critical-life escape.
- Setup wizard connected into tool/tool, container user-test checklist.
- Live friendly results, owner-type fix, tool.
- Setup interface mount in server, character-step hold.
- Internal/gear weapon/ammo model and Necromancer/Amazon build tables.
- Boss script robustness, potion gates, live-check procedure.
- Necromancer summon upkeep, curses, Bone Armor by state.
- Callers use IsFriendly, rest loop and hostileNear fixes, xp remainder.
- IsFriendly for every class.

</details>

</details>

<details>
<summary><b>2026-10-28</b> (73)</summary>

- Rebuilt HA stream dashboard, fixed sync loops and notifications.
- Amazon smoke test findings.
- Real game and bot run headless in the container.
- Necromancer smoke test findings.
- Setup wizard front-end.
- Normal Act 2 campaign, cube contract, Radament/Summoner/Duriel scripts.
- Re-scored milestone tracker, new S showcase milestone.
- Dungeon transitions verified, ErrLevelChanged re-plan, hint corrections, waypoint walk fix.

<details>
<summary>65 more</summary>

- Class builds, skill-id read-back, gear score, weapon swap, allow-lists, merc hire interface.
- Data-driven class combat, left/right skill binder, aura upkeep, Paladin melee loop.
- Akara Den claim fixed, full-pack purchase fix.
- Include any in-game instance in auto mode, adopt running keepAlive stream after restart.
- Paladin smoke test findings and tool.
- In-memory quest and waypoint source, live-verified.
- Act 1 boss script framework and scripts.
- Scene tile, grid3, keep-alive, StreamDesk and HA toggles.
- Fd study, game recycler, oog fd tools.
- TV-style stream scene page and generator.
- Shared stream selection, auto scheduler rule, StreamDesk and HA sync.
- Managed mode live, death/IsDead fixes, long-run test findings.
- Kick OAuth login and set-title library.
- Home Assistant package and dashboard for the stream.
- Stream control service, scheduler, max-live guard.
- LAN host/join over loopback, party driver, menu TCP/IP screens.
- Live-now link on the public page, sanitizer extension.
- Kick interface research for stream title.
- Desktop tray app for the stream.
- Pure mana/health resource policy package.
- Quest state reader, NPC dialog primitive, act hops, Act 1 handlers.
- Percent layout, grouped shipped feed, plain work titles, dark-mode check.
- This status page, with a safety check that keeps private details out.
- The roadmap: what alpha, beta and good-bot mean, and how far along each is.
- Checked in the real game that quests, waypoints, acts and difficulty are saved correctly.
- Research on how other projects add goals and progression on top of a game.
- Management for several bots at once, and goals now show up live.
- The bot now runs on the shared server and exposes its settings.
- Run several bots at once and play together as a party.
- The dashboard can show several running bots.
- Counters and a rig for long unattended test runs.
- The bot can start inside a container with no screen, a step toward easy installs.
- A live viewer to watch the bot's game from another device.
- The dashboard and watchdog now handle several bots.
- One web server that serves every running game.
- Record a game session and replay it later for testing.
- More game knowledge: mercenaries, item properties, effects, projectiles and runewords.
- A logged tool to set up test characters at any quest, waypoint or act.
- The bot spends stat and skill points, equips gear and plans its level-up runs.
- Learned what happens on death, skips enemies it cannot reach, and retreats when health is low.
- Automatic checks run on every change now.
- Lets the bot see the game's state without special permissions.
- Buys potions and scrolls, opens portals, and finds its way around town in any game.
- A catalogue of goals the bot can be given, and a way to track them.
- Plans for each character class and ideas for playing as a party.
- Navigation that works in any randomly generated map, finding exits and waypoints.
- A separate watchdog that steps in when the bot is stuck or in danger.
- The first real runs: the bot fights, shops and recovers from trouble.
- A live viewer for watching several bots at once.
- A logged tool to set up test characters quickly.
- Several games can now run side by side, each on its own hidden screen.
- Town chores: shopping, restocking and picking up loot.
- Fighting: picking targets, using skills and noticing danger and death.
- The main bot program with a policy for what to do after dying.
- Finding paths, exploring maps and using waypoints.
- A live status board and first screens for following the bot.
- Research on whether several bots can play together.
- Research on how other Diablo II bots work.
- Reliable key and mouse control that reconnects after hiccups.
- The bot can read the game's state: health, position, items and open screens.
- A launcher that starts the game and its helpers reliably.
- The bot reads the map around it and tracks where it has been.
- A goal manager with a built-in dashboard and a simulator.
- Reads the game's own data files to learn items, monsters and maps.
- Strategy for levelling up quickly.

</details>

</details>

---

Last updated: 2026-10-31 02:15 UTC

_Numbers are conservative._  
_A check only counts as done once it has been verified for real: a live run in the game, or automated tests when that is what the check asks for._  
_Half credit for work that is only partly proven._  
_Anything untried counts for nothing._  

_D.idle is an unofficial fan project. It is not affiliated with or endorsed by Blizzard Entertainment. Diablo is a trademark of its owner._
