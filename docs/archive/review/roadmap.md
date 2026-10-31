# Revised roadmap

Replaces the PLAN.md milestone ladder (findings F11). Each milestone has measurable exit criteria; "unattended" means no human input
and no agent touching the instance; sentinel actions are allowed but counted. Soak runs are continuous from M1 on (WP-OBS).
Baseline today (V docs/bot.md): Blood Moor farming at ~2,700 XP/h at clvl 1, 0 deaths in 6 min; a 9-minute run died once and stalled
(since fixed); level 3 not reached because of 0 gold.

## M0. Foundations (now, 1-2 weeks, mostly parallel-work batch 1)
Exit criteria:
- CI green on master for 20 consecutive runs of `go test -race -short -count=3` (flaky list empty or owned).
- Process topology decided and written (design-gaps D1); WP-MEMFD result recorded.
- `d2bot --managed` launched the game, logged in through OOG and read memory with **no setcap**, on one instance, once (10-minute live check).
- Map id stability across Save and Exit measured (findings F10); kill -9 progress loss measured (F3).
- Save rotation active for every live character (WP-SAVES).

## M1. Normal Act 1 unattended (wave 3 merged + leveler)
Exit criteria (one instance, 2 h run, twice, on two different characters/seeds):
- Fresh level-1 character reaches clvl 12 within 2 h, or >= 15,000 XP/h average after clvl 5.
- Deaths <= 1 per hour; no death with an unused potion in the belt and life above the chicken threshold 1 s before.
- 0 human interventions; sentinel actions <= 2 per hour; bot ladder rung >= 3 (leave game) <= 1 per hour.
- Longest no-progress window < 5 min.
- Town loop works with real money: buys potions and TP scrolls, repairs, sells, stashes (each verified live once by TOWN2).
- Stat and skill points spent per build plan within 1 game of gaining them (LEVELER).
- Process RSS growth < 5 % per hour after the first 15 min; fds and goroutines flat.

## M2. Act 1 boss and the first act transition
Exit criteria:
- Den of Evil, Blood Raven (Burial Grounds), Countess and Andariel completed unattended from a clvl-appropriate character (cheated level allowed for the test, flagged).
- Quest completion detected from the save and in game (WP-SAVES + PROGRESSION).
- Act 1 -> Act 2 travel via Warriv, Act 2 waypoints learned; mercenary hired in Act 2 and revived after a forced death.
- Corpse retrieval succeeds in >= 80 % of deaths where the policy chooses to retrieve (>= 5 deaths sampled).

## M3. Product spine (parallel to M2)
Exit criteria:
- One `d2bot` binary in managed mode supervises 2 instances; one HTTP server; dashboard shows both with goals, health, deaths, viewport.
- Sentinel runs in the product topology and passes the existing drill table (V docs/sentinel.md) plus a real death and a forced Wine crash, all in managed mode.
- Game restart from a hang loses <= 1 save cycle of progress (measured).
- Config is layered (D4); state files versioned with migrations (D11).
- 24 h soak on one instance: no fatal error, no manual restart, report produced by `d2soak`.

## M4. Normal complete
Exit criteria:
- A character goes from clvl 1 to killing Baal in Normal unattended, total wall time <= 40 h, deaths <= 1 per hour averaged, 0 human interventions after start.
- Each act's required quests and transitions are Steps with offline tests and one recorded fixture per boss arena.
- Character enters Nightmare (difficulty screen verified).

## M5. Headless container and setup wizard (v1 gate)
Exit criteria:
- Container image (no Blizzard bytes) runs on the NAS (deployed through the UI after asking) or a Proxmox LXC; volumes for game files, prefixes and state.
- A fresh user path: wizard -> provide installers or an existing verified game dir -> CD keys typed in the installer window via the browser viewport -> patch -> data extract -> character created -> first goal running, with no shell commands, on two different hosts (the worker host desktop mode and a container).
- `d2bot doctor` passes on both and gives an actionable fix for each failure it can detect.

## M6. Nightmare and Hell, farming goals
Exit criteria:
- Nightmare Act 1-5 progression unattended with deaths <= 2 per hour (XP loss counted).
- Farming goals (Countess, Andariel, Mephisto, Pindleskin) with per-run timing and drop logs; runs per hour within 2x of a scripted human baseline.
- Teleport-based navigation (clvl 18+) with walk fallback.

## M7 and later (deferred)
Multi-bot party over loopback TCP/IP (NET phase A), friends over Tailscale (phase C), meta layer (upgrades, prestige, cheats as rewards), hardcore.

## Definition of done for a usable v1
1. One binary, two modes (desktop and container), installed by the wizard from the user's own installers or game dir; nothing Blizzard-owned shipped or committed.
2. A new character goes Normal Act 1 to Nightmare entry unattended (M4) with <= 1 death per hour and no human intervention.
3. Goals: reach level, farm area/run, kill boss, idle windows; queue, priorities and schedules work; progress and ETA visible.
4. Dashboard on phone and desktop: live viewport (view-only), status, goals, deaths, health, knobs/profiles.
5. Failsafes: in-bot ladder, sentinel and an external watchdog; 72 h soak with 0 manual restarts and a soak report.
6. Characters protected: rotated save backups, checksum validation, restore from the UI.
7. Upgrades: binary replaced without losing state (migrations), runner/Proton version pinned and checked at start.
8. Docs: setup guide, troubleshooting (doctor output), CHANGELOG with release notes, THIRD_PARTY notices.

## Risk register

| id | risk | likelihood | impact | mitigation | trigger to act |
|---|---|---|---|---|---|
| R1 | Managed mode/subreaper does not work with the real game or Proton re-parents differently | med | high | M0 live check; SYS_PTRACE fallback | M0 check fails |
| R2 | Quest/waypoint state only readable after Save and Exit | med | med | save cycles; observed deltas; QUESTMEM later | WP-SAVES live checklist |
| R3 | Proton/Wine update changes discovery or rendering | med | high | pin runner, canary at start, container pins image | any runner change |
| R4 | Patch or stub downloads disappear | med | med | accept user-supplied patch / existing game dir | download 403/404 |
| R5 | Host load (the worker host 30-40) makes timing knobs unreliable | high | med | verify-by-effect, load guard, run soak on a quieter host | ladder rate up with load |
| R6 | State-machine explosion when quests land | high | med | task layer (D12) before quest code | Runner > 10 states |
| R7 | Save corruption or progress loss on kill | med | high | WP-SAVES rotation, save cycles | first corrupt save |
| R8 | Fixture rot hides offset regressions | med | med | envelope + canary over fixtures in CI | any offset change |
| R9 | Agent parallelism causes merge churn in shared files (env, contracts, AGENTS.md) | high | low | consumer-side interfaces, per-domain files, main merges | conflicts in >2 merges/day |
| R10 | Ladder-only runewords unavailable offline break the gear plan | med | med | check runes.txt (WP-DATA2), plan without them | before M6 gear work |
| R11 | Xvfb clipped client area (800x587) breaks bottom-row clicks | med | med | 1024x768 Xvfb test, assert screen size | belt/HUD click failures |
| R12 | Legal/ToS: distribution or online use | low | high | no Battle.net path, no Blizzard bytes, private repo, license review before any release | any plan to publish |
| R13 | Helper (d2input) dies silently | med | med | supervised helper, exit logs, sentinel rule | > 1 helper death per day |
| R14 | Memory/context of the coordinating main agent (33 KB memory file) | med | low | compress memory, roadmap as the index | next compaction |

## What to cut or defer
- Defer: multi-bot party, PvPGN/D2GS realm (riskiest legally and technically, V docs/net/multiplayer.md), meta layer, hardcore, gambling,
  cube recipes beyond what M6 needs, Cow level, D2DX, framebuffer-from-memory and in-prefix capture (Xvfb capture works, V docs/watch.md),
  chat reading.
- Cut from v1: realm/Battle.net anything (permanently), human-bot mixed play, second class build until the Sorceress is at M4.
- Keep small: the Lightning fallback build only as data; respec flow after M4.
- Stop doing: new live tools per agent that bypass `D2BOT_INSTANCE` (`input.FindGamePID`, V docs/cheats.md hazards); remove it when the owners next touch combatlive/townlive.
