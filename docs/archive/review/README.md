# Design review (2026-10-28)

Critical review of d2lod-bot at master `bd8c13f` plus the running LEVELER and COMBAT2 branches, by agent REVIEW. Docs only:
no game access, no code changes. Tags: (V) verified in repo/docs or by a command run here, (S) secondary, (U) unverified.

| file | what |
|---|---|
| [findings.md](findings.md) | 19 findings ranked blocker/high/medium/low with evidence and a recommendation each |
| [gotchas.md](gotchas.md) | 74 gotchas (version drift, Wine/Proton, 1.14d quirks, timing, display, permissions, saves, leaks, anti-stuck, fixtures, legal, distros) with status |
| [parallel-work.md](parallel-work.md) | 15 work packages that can start now in directories disjoint from the running agents, dependency graph, order |
| [design-gaps.md](design-gaps.md) | 15 underspecified/contradictory areas with concrete resolutions and interface sketches |
| [roadmap.md](roadmap.md) | milestones M0-M6 with measurable exit criteria, v1 definition of done, risk register, cuts |

## Executive summary
The project is in good shape where it has been tested: reading memory, input through the in-prefix helper, navigation in Act 1,
fighting, the town UI, OOG menus, multi-instance isolation and a sentinel with measured drills. The honest tagging of what is verified
is a real asset. The risks are not in any single package; they are in what nobody owns:

1. **The product cannot run the way it is tested.** Every live run uses `--attach` with a capability granted by sudo. The real path
   (the app launches the game and is its subreaper) has never run with the game, and it collides with the out-of-process sentinel, which
   cannot read a game it did not spawn. Restarts actually work through a bash script that ssh's to the worker host. A topology decision is needed
   before more code assumes the dev setup (findings F1, design-gaps D1).
2. **Nothing beyond Act 1 has an owner.** Quests, act transitions, bosses, the mercenary and the difficulty switch are unassigned, and the
   quest state has no source. The save file can provide it today (F2, D13).
3. **Characters are unprotected.** Kill-and-restart loses unsaved progress; there is no save rotation or checksum check (F3).
4. **Quality gate missing.** No CI; master failed a bot failsafe test in 2 of 4 runs here (F4).
5. **Single-instance surfaces** (one manager, three web servers) and **shared mutable access** (several readers per tick, several input
   writers per window) will hurt as soon as two bots or the sentinel act at once (F5, F6, F7).
6. Upside: the map seed is probably stored per character in the save, which would allow cached routes and reproducible test maps (F10).

## Top 10 actions
1. Decide the process topology (D1) and run `d2bot --managed` live once without setcap (10 min, main agent).
2. Dispatch WP-CI and fix or quarantine `TestFailsafeDeathOverlayNeverDismissed` (it failed 2 of 4 runs).
3. Dispatch WP-MEMFD (fd passing proof) and WP-LIFECYCLE (game.sh into Go, sentinel off bash/ssh).
4. Dispatch WP-SAVES: save rotation now; quest/waypoint/map-id state from the save.
5. Dispatch WP-PROGDESIGN: quests, act travel, merc, bosses, difficulty switch, and the task layer (D12).
6. Run two cheap live checks: map id stable across Save and Exit; what a kill -9 loses.
7. At the LEVELER merge: drop the committed `levelerlive` binary and deduplicate plan data into one package (F15, F17).
8. After the wave-3 merges: one UI-flag table (F9), ctx + Clock in `input.Controller` (F6), an Observer tick (D5).
9. Then WP-EVENT and WP-SERVER before the dashboard grows further (instance-scoped API, one auth).
10. Replace the PLAN ladder with roadmap.md milestones and start continuous soak at M1 (WP-OBS).

## Sources used beyond the repo
- d2s format and seed tooling: [nokka/d2s](https://github.com/nokka/d2s/), [feored/d2mapseed](https://gh.r-mj.com/feored/d2mapseed),
  [D2CE d2s file format](https://github.forks.workers.dev/WalterCouto/D2CE/blob/main/d2s_File_Format.md) (map id at offset 171; S).
