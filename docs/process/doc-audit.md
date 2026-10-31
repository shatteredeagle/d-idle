# Doc audit (2026-10-30)

Method: `git log` dates per path on master 6a6ff58 (all times 2026, US Eastern; the repo is 2.9 days old: first commit 10-28 12:41, 1,516 commits). Because everything is hours old, 'days since touched' is useless, so the column `code commits since` counts non-merge commits to `internal/ cmd/ tools/` after the doc's last change: the larger it is, the more the code has moved under the doc. `n` = commits touching the path; lines = markdown lines (directories: all .md). Status is a judgement: **live** = current and edited recently or a stable reference; **stale** = known overtaken by later work (reason given); **superseded** = replaced by a named doc; **archive candidate** = history only. No file was deleted or moved. Where a status is `stale?` it was derived only from age (no code-vs-doc check) and needs an owner to confirm.

Totals: see end. Evidence for stale/contradiction claims is in spec-approach-evaluation.md section 1.

| path | purpose (first heading) | last change | n | lines | code commits since | status | note |
|---|---|---|---|---|---|---|---|
| `docs/BACKLOG.md` | Agent backlog (the dispatcher cron reads this) | 10-30 04:01 | 10 | 88 | 74 | stale | item 8 says D2DFORK no commit yet (docs/d2d-fork.md exists); older READY queue admits it was not checked |
| `docs/INDEX.md` | docs/INDEX.md: map of the docs | 10-30 04:00 | 8 | 246 | 74 | stale | header says 833 commits/16 G1 attempts; merc row says not wired; map of per-act state predates d2d. Still the entry point: refresh or generate |
| `docs/act2.md` | Normal Act 2 (agent ACT2, offline) | 10-29 12:02 | 18 | 253 | 255 | stale? | untouched since before the 10-29 12:05 doc sweep; unchecked |
| `docs/act3.md` | Normal Act 3 plan (agent ACT3PLAN, docs only) - also holds t | 10-29 12:02 | 3 | 138 | 255 | stale | offline plan, no live run; shared Act 3-5 work list |
| `docs/act4.md` | Normal Act 4 plan (agent ACT3PLAN, docs only) | 10-29 09:40 | 2 | 110 | 403 | stale | offline plan |
| `docs/act5.md` | Normal Act 5 plan (agent ACT3PLAN, docs only) | 10-29 09:11 | 1 | 125 | 465 | stale | offline plan |
| `docs/actbot.md` | ACTBOT and ACT45BOT: Acts 2-5 wiring in cmd/d2bot (offline,  | 10-29 09:46 | 3 | 94 | 400 | stale? | untouched since before the 10-29 12:05 doc sweep; unchecked |
| `docs/api.md` | Goal manager, Runner contract and HTTP API | 10-28 16:56 | 4 | 156 | 739 | live | stable reference |
| `docs/beltfill.md` | Belt fill (BELTFILL) | 10-29 21:04 | 1 | 43 | 172 | live | touched in the last day |
| `docs/boardhost.md` | Board host and publish timer | 10-28 20:34 | 1 | 22 | 673 | live | stable reference |
| `docs/boss.md` | Boss scripts: internal/boss (agent BOSSRUNS) | 10-30 02:23 | 11 | 208 | 118 | live | touched in the last day |
| `docs/bossgate.md` | Boss readiness gate (BOSSGATE, WP1/WP3/WP5 of docs/soak-anal | 10-30 02:23 | 2 | 17 | 118 | live | touched in the last day |
| `docs/bot.md` | bot: the core that ties the layers together (agent BOT) | 10-29 12:04 | 15 | 215 | 252 | stale? | untouched since before the 10-29 12:05 doc sweep; unchecked |
| `docs/builds-endgame.md` | Endgame builds (clvl 2-90, Normal -> Nightmare -> Hell), one | 10-29 10:41 | 2 | 168 | 339 | stale? | untouched since before the 10-29 12:05 doc sweep; unchecked |
| `docs/cheats.md` | Cheats: logged test cheats (agent CHEAT) | 10-30 01:07 | 10 | 203 | 145 | live | touched in the last day |
| `docs/ci.md` | CI (forge Actions) | 10-29 11:18 | 1 | 21 | 296 | live | stable reference |
| `docs/combat.md` | Combat: internal/combat (agent COMBAT) | 10-30 03:23 | 11 | 201 | 83 | live | touched in the last day |
| `docs/container.md` | Real Diablo II 1.14d headless in the container (agent CONTAI | 10-29 15:14 | 14 | 198 | 207 | stale | Wine container PAUSED; d2d-container/ is the successor |
| `docs/coverage.md` | Test coverage and unattended-run gaps | 10-29 10:09 | 1 | 51 | 382 | stale | point-in-time coverage numbers |
| `docs/d2-1.14d-memory.md` | Diablo II LoD 1.14d: memory, map and input reference for an  | 10-29 06:31 | 4 | 450 | 519 | live | stable reference |
| `docs/d2-knowledge.md` | D2 knowledge index (for the d2-sme agent and anyone asking " | 10-29 16:44 | 2 | 15 | 204 | live | touched in the last day |
| `docs/d2d-backend.md` | d2d as the primary backend: adapter design (agent D2DADAPTER | 10-30 09:13 | 12 | 699 | 22 | live | touched in the last day |
| `docs/d2d-bot.md` | The bot on the d2d backend (agent D2DBACKEND, 2026-10-30) | 10-30 09:11 | 2 | 90 | 22 | live | touched in the last day |
| `docs/d2d-combat.md` | Combat on d2d (agent D2DCOMBAT, 2026-10-30) | 10-30 09:52 | 2 | 72 | 0 | live | touched in the last day |
| `docs/d2d-exit.md` | d2d bot: "no exit to Blood Moor" (agent D2DEXIT, 2026-10-30) | 10-30 09:54 | 2 | 46 | 0 | live | touched in the last day |
| `docs/d2d-fidelity.md` | d2d fidelity checklist (agent D2DFIDELITY, 2026-10-30) | 10-30 08:54 | 4 | 289 | 37 | live | touched in the last day |
| `docs/d2d-fork.md` | d2d private fork: lockstep clock and `dump` verb (agent D2DF | 10-30 08:54 | 4 | 192 | 37 | live | touched in the last day |
| `docs/d2d-gaps.md` | d2d fidelity gaps: cause analysis (agent D2DTOGGLE, 2026-10- | 10-30 08:54 | 1 | 31 | 37 | live | touched in the last day |
| `docs/d2d-probes.md` | d2d probe batch (agent D2DPROBES, 2026-10-30) | 10-30 03:31 | 4 | 130 | 82 | live | probe scripts |
| `docs/d2d-soak.md` | d2d replicated soak: harness, metrics, pass rule (agent D2DS | 10-30 09:12 | 2 | 92 | 22 | live | touched in the last day |
| `docs/d2d-spike.md` | d2d feasibility spike (agent D2DSPIKE, 2026-10-30) | 10-30 02:20 | 2 | 260 | 121 | live | touched in the last day |
| `docs/d2d-town.md` | Town layer on the d2d backend (agent D2DTOWN, 2026-10-30) | 10-30 09:22 | 2 | 53 | 17 | live | touched in the last day |
| `docs/d2d-walk.md` | d2d end-to-end walk: cmd/d2dwalk (agent D2DWALK, 2026-10-30) | 10-30 07:47 | 2 | 124 | 37 | live | touched in the last day |
| `docs/d2dctl.md` | d2dctl: Go client for the d2d control channel | 10-30 07:33 | 2 | 84 | 39 | live | touched in the last day |
| `docs/data.md` | internal/data: static game tables | 10-28 12:58 | 1 | 181 | 823 | live | stable reference |
| `docs/deathautopsy.md` | Death autopsy: G1 attempts 11 to 13 (2026-10-29) | 10-30 02:56 | 4 | 83 | 97 | archive candidate | G1 attempts 11-13 post-mortem, fixes merged |
| `docs/deathspiral.md` | Death spiral guards (G1 attempt 1, 2026-10-29) | 10-30 06:03 | 12 | 103 | 52 | live | touched in the last day |
| `docs/decisions.md` | Decisions (user, with date) | 10-30 08:58 | 4 | 22 | 33 | live | user decisions; 22 lines, the best-kept doc |
| `docs/difficulty.md` | Difficulty: Normal -> Nightmare -> Hell (levels, resists, im | 10-29 10:26 | 6 | 240 | 362 | stale? | untouched since before the 10-29 12:05 doc sweep; unchecked |
| `docs/display.md` | Live display | 10-28 13:11 | 1 | 69 | 813 | live | stable reference |
| `docs/earlygame.md` | Early game: a fresh character, clvl 1 to 12 (agent EARLYGAME | 10-30 03:23 | 3 | 112 | 83 | live | touched in the last day |
| `docs/farm.md` | Farm runs (FARMRUNS, 2026-10-29) | 10-29 10:26 | 2 | 34 | 362 | stale? | untouched since before the 10-29 12:05 doc sweep; unchecked |
| `docs/fleet.md` | Fleet supervisor and party orchestrator (agent FLEET) | 10-29 12:04 | 4 | 126 | 252 | stale? | untouched since before the 10-29 12:05 doc sweep; unchecked |
| `docs/fresh-character.md` | Fresh-character soak path (plan-v2 M1) | 10-30 01:07 | 1 | 49 | 145 | live | touched in the last day |
| `docs/friendly.md` | Friendly units (FRIENDLY) | 10-29 12:02 | 4 | 50 | 255 | stale? | untouched since before the 10-29 12:05 doc sweep; unchecked |
| `docs/g1-log.md` | G1 log (2 h real gate, STRATEGY review) | 10-30 09:29 | 36 | 145 | 7 | live | append-only attempt log |
| `docs/gear-gate.md` | Gear gate: graded risk instead of a hard gate (SOFTGATE) | 10-29 22:26 | 2 | 48 | 167 | live | touched in the last day |
| `docs/gear.md` | Gear and ammo model (agent GEAR, package `internal/gear`) | 10-29 10:44 | 11 | 105 | 337 | stale? | untouched since before the 10-29 12:05 doc sweep; unchecked |
| `docs/input.md` | Input: helper, client, transform (agent INPUT) | 10-28 16:50 | 2 | 72 | 742 | live | stable reference |
| `docs/inventory.md` | Inventory and stash policy (INVSTASH audit, 2026-10-30) | 10-30 09:44 | 3 | 55 | 2 | live | touched in the last day |
| `docs/items.md` | Items, economy and runewords for the offline bot (agent ITEM | 10-29 09:24 | 2 | 231 | 434 | stale? | untouched since before the 10-29 12:05 doc sweep; unchecked |
| `docs/knobs.md` | Knobs and settings reference | 10-30 09:44 | 36 | 435 | 2 | live | touched in the last day |
| `docs/launch.md` | LAUNCH: prefix, install, launch, helper, OOG (agent `launch` | 10-28 20:15 | 4 | 77 | 678 | live | stable reference |
| `docs/leakhunt.md` | fd growth hunt (agent LEAKHUNT, VERIFIED LIVE on the worker host i3, 2 | 10-29 10:15 | 5 | 68 | 381 | stale? | untouched since before the 10-29 12:05 doc sweep; unchecked |
| `docs/leveler.md` | leveler: points, gear and the leveling loop (agent LEVELER) | 10-29 12:02 | 12 | 170 | 255 | stale? | untouched since before the 10-29 12:05 doc sweep; unchecked |
| `docs/lifecycle.md` | Game lifecycle in Go (agent LIFECYCLE) | 10-28 16:50 | 2 | 38 | 742 | live | stable reference |
| `docs/liveruns-briefs.md` | Agent briefs for the live batches (paste as the `prompt` of  | 10-29 11:18 | 1 | 63 | 296 | archive candidate | hand-pasted briefs; replaced by .claude/agents + a brief template |
| `docs/liveruns-checks.md` | Live-run runbooks R0..R12 (agent LIVERUNBOOK; see [liveruns. | 10-29 12:02 | 4 | 174 | 255 | stale? | untouched since before the 10-29 12:05 doc sweep; unchecked |
| `docs/liveruns.md` | Live-run runbook: order, rules, shared procedures (agent LIV | 10-29 11:16 | 1 | 144 | 299 | stale? | untouched since before the 10-29 12:05 doc sweep; unchecked |
| `docs/map.md` | d2map: world map reader (agent MAP) | 10-28 13:01 | 1 | 94 | 820 | live | stable reference |
| `docs/mem.md` | Memory layer: internal/mem, internal/d2, cmd/d2snap | 10-28 16:36 | 2 | 73 | 749 | live | stable reference |
| `docs/memfd.md` | memfd: passing the game-memory descriptor between processes  | 10-28 16:26 | 1 | 19 | 758 | live | stable reference |
| `docs/merc-farm.md` | Mercenaries and post-campaign farming (research, MERCRESEARC | 10-29 09:08 | 3 | 148 | 468 | superseded | by merc.md, merc-wire.md, sme/mercenary.md |
| `docs/merc-wire.md` | Merc wiring (agent MERCWIRE) | 10-30 09:29 | 1 | 55 | 7 | live | touched in the last day |
| `docs/merc.md` | Mercenary policy (internal/merc) | 10-30 09:29 | 2 | 49 | 7 | live | touched in the last day |
| `docs/multi.md` | Multi-instance: one game per agent (agent MULTI) | 10-29 13:45 | 3 | 84 | 230 | live | touched in the last day |
| `docs/nav.md` | nav: walking, exits, waypoints, exploration (agent NAV) | 10-30 05:10 | 12 | 327 | 61 | live | touched in the last day |
| `docs/nightmare-campaign.md` | Nightmare and Hell campaigns (nightmare_all, hell_all) | 10-29 09:44 | 2 | 28 | 402 | stale | offline only |
| `docs/npc.md` | NPC interaction (town/npc.go, cmd/d2bot/task.go) | 10-30 00:34 | 1 | 50 | 149 | live | touched in the last day |
| `docs/oog.md` | OOG: menus, start, leave, crash restart (agent OOG) | 10-28 19:05 | 2 | 55 | 700 | live | stable reference |
| `docs/open-findings.md` | Open findings (merged from the four merge-risk reviews) | 10-30 04:01 | 3 | 89 | 74 | live | review-merged findings; claims partly U |
| `docs/probes.md` | Live probes (cmd/probes) | 10-29 10:05 | 1 | 113 | 385 | stale? | untouched since before the 10-29 12:05 doc sweep; unchecked |
| `docs/publish.md` | Public status page (tools/publish) | 10-28 19:27 | 3 | 107 | 687 | live | images for the public page |
| `docs/quest.md` | Quest layer: internal/quest (agent QUESTS, offline) | 10-29 07:21 | 4 | 83 | 512 | live | stable reference |
| `docs/questmem.md` | In-memory quest and waypoint data (agent MEMQUEST, V-05) | 10-28 23:10 | 2 | 60 | 641 | live | stable reference |
| `docs/quickstart.md` | Quickstart: run D.idle on your own machine (agent USERDOC, 2 | 10-29 12:04 | 2 | 191 | 252 | stale | user docs for the Wine path; container/d2d decision pending |
| `docs/recovery.md` | Recovery phase (broke character) | 10-30 05:40 | 7 | 123 | 57 | live | touched in the last day |
| `docs/release.md` | Release packaging | 10-29 11:14 | 1 | 36 | 301 | live | stable reference |
| `docs/replay.md` | Replay, fixtures and canaries: internal/replay, internal/can | 10-28 16:36 | 1 | 20 | 749 | live | stable reference |
| `docs/resource.md` | Resource policy: internal/resource (agent RESOURCE) | 10-29 11:07 | 5 | 103 | 308 | live | stable reference |
| `docs/restloop.md` | Sorceress rest loop on i4 (diagnosis from the log, no game) | 10-29 00:22 | 1 | 21 | 633 | archive candidate | one-off diagnosis from the log, fixed |
| `docs/review-strategy.md` | Strategic review: plan, risk, verification debt, process (ag | 10-30 03:59 | 4 | 177 | 74 | superseded | by plan-v2 s5/s10/s11; freeze text marked stale in-place |
| `docs/sandbox.md` | Wine mount sandbox (agent SANDBOX) | 10-29 13:45 | 1 | 95 | 230 | stale | Wine mount sandbox; Wine path de-prioritised |
| `docs/saves.md` | Saves: integrity and progression from the .d2s (agent SAVES) | 10-28 17:42 | 2 | 53 | 717 | live | stable reference |
| `docs/sentinel.md` | Sentinel: layer 2 failsafes (agent SENTINEL) | 10-28 17:15 | 4 | 82 | 726 | live | stable reference |
| `docs/setup-ui.md` | Setup wizard UI (WIZARDUI) | 10-28 22:49 | 1 | 11 | 644 | live | stable reference |
| `docs/setup.md` | Setup: doctor checks and wizard backend (WP-SETUP) | 10-29 10:56 | 6 | 77 | 325 | live | stable reference |
| `docs/soak-analysis.md` | Soaky soak analysis (SOAKANALYSIS, 2026-10-29) | 10-30 05:18 | 12 | 286 | 59 | superseded | pre-fix baseline; death rule moved to plan/death-data.md |
| `docs/soak-hosts.md` | Where can soaks and workers run besides the worker host? (agent SOAKHO | 10-29 12:03 | 1 | 223 | 254 | stale | Wine-era host options; paused |
| `docs/soak.md` | Soak judging and game recycle (agent SOAKFIX) | 10-29 11:44 | 10 | 126 | 258 | stale? | untouched since before the 10-29 12:05 doc sweep; unchecked |
| `docs/soak24.md` | Milestone alpha: the 24 h unattended soak (agent SOAK24) | 10-30 09:32 | 15 | 152 | 4 | live | touched in the last day |
| `docs/stallfatal.md` | Stall, ladder rung 4 and "the game is gone" (soaks 2026-10-2 | 10-30 05:10 | 13 | 202 | 61 | live | touched in the last day |
| `docs/streamctl.md` | streamctl: stream control service | 10-28 21:06 | 4 | 63 | 667 | live | stable reference |
| `docs/summon.md` | Necromancer upkeep (internal/combat/necro.go) | 10-29 12:02 | 12 | 118 | 255 | stale? | untouched since before the 10-29 12:05 doc sweep; unchecked |
| `docs/task.md` | Task layer (internal/task) and how the bot wires it (agent T | 10-30 01:28 | 4 | 142 | 135 | live | touched in the last day |
| `docs/nas-container.md` | d2idle container on Tower (the NAS 7.2), agent TOWERDEPLOY, 2 | 10-29 15:14 | 4 | 94 | 207 | stale | Wine container work PAUSED by decisions.md 2026-10-30 |
| `docs/town.md` | Town and loot: internal/town, internal/loot (agent TOWN) | 10-29 23:01 | 8 | 102 | 159 | live | touched in the last day |
| `docs/troubleshooting.md` | Troubleshooting D.idle (agent USERDOC, 2026-10-29) | 10-29 12:04 | 2 | 94 | 252 | stale | same as quickstart |
| `docs/viewport.md` | Viewport: capture and stream library (agent VIEWPORT) | 10-28 16:45 | 1 | 46 | 746 | live | stable reference |
| `docs/watch.md` | d2watch: view-only web viewer of the game instances (agent W | 10-29 09:37 | 3 | 47 | 408 | live | stable reference |
| `docs/waypoints.md` | Waypoints (agent WPUSE, 2026-10-30) | 10-30 03:21 | 1 | 59 | 86 | live | touched in the last day |
| `docs/wineclean.md` | WINECLEAN: prefix hygiene (agent wineclean) | 10-29 13:19 | 1 | 20 | 235 | live | ops |
| `docs/archive` | README.md goals-catalog.md milestones... | 10-29 12:04 | 2 | 2441 | 252 | archive | already archived |
| `docs/classes` | README.md amazon.md assassin.md... | 10-29 12:02 | 13 | 762 | 255 | live | per-class plans |
| `docs/d2d-container` | Dockerfile.draft README.md compose.yml.draft... | 10-30 09:01 | 2 | 271 | 28 | live | draft design; compose and Dockerfile drafts |
| `docs/d2d-probes` | c d2dstart.sh d2dstop.sh... | 10-30 05:35 | 13 | 0 | 59 | live | probe scripts |
| `docs/goals` | README.md execution.md planner.md... | 10-29 12:03 | 3 | 471 | 253 | live | goal manager planner/tracking docs |
| `docs/milestones` | milestones.json... | 10-29 12:04 | 5 | 0 | 252 | stale | json read by board; rescored 10-28 per INDEX |
| `docs/net` | design-now.md multiplayer.md... | 10-28 19:05 | 3 | 298 | 700 | stale | multiplayer design, 10-28, not touched since |
| `docs/plan` | STATUS.md bot-requirements.md code-reality.md... | 10-30 08:22 | 43 | 730 | 37 | live | plan-v2 + STATUS authority; STATUS hand-written |
| `docs/prior-art` | 01-leveling.md 02-runs.md 03-pathing.md... | 10-29 12:03 | 3 | 918 | 253 | archive candidate | research survey, no later change; archive/prior-art/meta already exists |
| `docs/progression` | README.md acts.md bosses.md... | 10-29 12:04 | 6 | 926 | 252 | live | results of cheats/progression probes |
| `docs/publish` | scene-1280.png scene-tile.png... | 10-28 19:27 | 1 | 0 | 687 | live | images for the public page |
| `docs/sme` | README.md audit-bot-vs-human.md blood-raven-caster.md... | 10-30 04:00 | 52 | 4971 | 74 | live | knowledge base, 31 files |
| `docs/stream` | kick-api.md kick-title.md... | 10-28 19:08 | 2 | 162 | 696 | live | Kick API notes |

Totals (115 entries): stale 15, stale? 17, live 75, archive candidate 4, superseded 3, archive 1
