# Sentinel: layer 2 failsafes (agent SENTINEL)

Code: `internal/sentinel` (pure rule engine `rules.go`, `config.go`, `obs.go`; real world `observe_linux.go`, `act.go`; `runner.go`, `server.go`),
`cmd/d2sentinel`, `tools/sentinel.sh`, panel `web/live/panels/sentinel.js`. Out-of-process, does NOT trust the bot (PLAN.md "Failsafes").
Tags: VERIFIED LIVE = observed on i5 (Multi) 2026-10-28; UNVERIFIED = not observed.

## Safety model
- OBSERVE-ONLY by default. `--act` lets every rule act, `--act-rules a,b` only those, `--act-instances i5` restricts acting to those instances
  (others stay observe-only), `-instances i1,i5` is the watch list (config `enabled:false` per instance too), `--keep i5` protects from the idle reaper.
- Every decision is a JSONL line (`<data>/logs/sentinel.jsonl` on the worker host (old `~/.local/state/d2lod/sentinel.jsonl` moved on start)): `kind` = `would` (observe-only), `acted` (with `result`: ok, verified,
  detail, took), `skipped` (instance lock held), plus `obs` records (on change of the summary and every 30 s) and `start`.
- Engine: a rule fires only after its condition held for `For`; then Cooldown; after MaxAttempts within Window it escalates once (e.g. Esc x3 ->
  restart game) and then only alerts; the condition false for 30 s clears the counters. One action per instance per step, 5 s settle after any action, so
  the sentinel never fights the bot. Restarts, OOG start, leave/rejoin and the idle stop take `flock /tmp/d2lod-game-<id>.lock` (wait 20 s, else `skipped`);
  Esc, dismiss and SIGCONT do not. i0 is never restarted, stopped or touched through X.

## Rules (priority order; thresholds in the JSON `thresholds`, per rule `rules.<name>.{enabled,for,cooldown,maxAttempts}`)
| rule | condition (default for) | action -> escalation |
|---|---|---|
| proc-gone | no Game.exe or zombie (8 s) | restart-game -> alert (3 in 15 min) |
| crash-dialog | unexpected window on the instance's display (title crash/error/only one copy, or any window >= 100x60 that is not Diablo II/helper/IME) or BlizzardError.exe of the instance (3 s) | dismiss (X key Return to that window on the instance's own display) -> restart-game |
| helper-gone | Game.exe up, helper port closed (10 s) | `game.sh helper` -> alert |
| window-gone | helper up but sees no Diablo II window (45 s) | restart-game |
| game-stopped | /proc state T (10 s) | SIGCONT -> restart-game |
| game-hung | IsHungAppWindow via helper `hung` (30 s) | restart-game |
| game-frozen | frame hash unchanged AND CPU ticks flat (90 s) | restart-game |
| death-overlay | in game and (life 0 or mode 0/17) (10 s) | respawn: Esc, verify alive from memory, up to 3 -> restart-game |
| esc-menu-stuck | UI[9] open (20 s) | Esc until closed, verified from memory -> restart-game |
| panel-stuck | any other panel flag open (90 s) | Esc -> alert |
| out-of-game | menu state (30 s), character configured | `Flow.Start` (OOG) -> restart-game |
| loading-stuck | loading/ambiguous state (150 s) | restart-game |
| bot-silent / bot-stuck | bot health unreachable / reports a stuck reason (60 s) | `botRestartCmd` (config) -> alert |
| idle-reaper | no lock, no bot heartbeat, memory fingerprint unchanged (30 min), not i0, not in `keep` | Save and Exit if in game, then `game.sh stop` |
| no-progress | a goal is active (bot says so, or `expectProgress`) and no xp/gold/area/position change (8 min) | ladder, one step per 2 min: Esc x3 -> bot-reset (POST `botResetUrl`) -> leave-rejoin -> restart-game -> alert |
Idle reaper note: the frame is NOT required to be unchanged (the town animates, VERIFIED: hash changes every few seconds), only the memory state, the lock and the bot.

## Observation (per instance, every 2 s)
Process (`mem.FindGameFor`, /proc stat state and CPU ticks), game memory (`internal/d2`: state, mode, life, position, area, level, xp, gold, UI flags,
menu screen via `launch.Identify`), frame hash (`import -window <id> rgb:-` of the game window found by `xdotool search --name '^Diablo II$'` on the
instance's own Xvfb display every 5 s, never the root window, i0 not captured), helper `info` and `hung`, window list for dialogs, BlizzardError.exe,
lock probe, optional bot health (`botUrl`, tolerant decode of state / time_in_state / last_progress / stuck_reason / goal_active in many spellings).
Gotcha found live: UI flags 6 and 7 (new stat / skill point buttons) are set on a character with unspent points; `d2.UIState.AnyPanelOpen()` counts them
(a levelled character looks permanently "in a panel"). The sentinel ignores them; the bot should too.

## Visibility
`GET /api/v1/sentinel` (default `127.0.0.1:8895`, `-addr`; `/api/sentinel` is a deprecated alias for one release, answering with `Deprecation: true` and a `Link` to the new path): `{time, mode, summary, instances:[{id, health ok|attention|down, line, acting, obs, rules:[{rule,active,forSec,attempts,step}], lastAction}], recent:[records], thresholds}`.
d2watch can poll it and show `summary` (e.g. `sentinel[observe-only] i4 OK | i5 ATTENTION`) and per instance `line`/`health`. `/live/` is a status hub page with a
`sentinel` panel (internal/status, Order 5; JS module `web/live/panels/sentinel.js`); the d2lod status page can mount the same panel from `Runner.Snapshot()`.

The HTTP side is `internal/server` (`Runner.NewServer`): `/api/v1/sentinel`, `/live/`, open `/healthz` and `/readyz`, with the same guard as the bot
server (Host/Origin/CSP checks, one token: Bearer, `X-Token`, `d2token` cookie, `?token=`). Token: `-token` or `$D2BOT_TOKEN`. A non-loopback `-addr`
without a token refuses to start; on loopback no token is needed (tools/sentinel.sh uses 127.0.0.1 over ssh). `-allow-host NAME` accepts one extra Host on loopback.

## Operating
`tools/sentinel.sh start [flags]` builds, ships to `~/d2lod-live/sentinel/` on the worker host, setcap (worker-sudo), runs user service `d2sentinel` (instances from
tools/instances.conf without i0, observe-only unless flags). `stop`, `status`, `log [N]`, `json`, `once [ids]` (one observation JSON per instance).
Example for the drills: `tools/sentinel.sh start --act --act-instances i5 --keep i5`. Restarts call `internal/launch` (Life; see docs/lifecycle.md; `game.sh` is no longer used by the sentinel; the sentinel holds the lock because
the sentinel holds the lock) and run tools/mute-d2.sh afterwards.

## Fault drills on i5 (all VERIFIED LIVE, thresholds above, observation period 2 s)
| drill | injected | detected | acted | recovered |
|---|---|---|---|---|
| kill -9 Game.exe | 15:09:42.3 | +1.5 s | +11.5 s (8 s threshold + poll) restart-game | in game as Multi, town, +48 s (game.sh stop/start/helper + OOG 36.5 s) |
| SIGSTOP Game.exe | 15:11:33.5 | +1.7 s (state T) | +13.7 s SIGCONT (not 2 min: the sentinel resumes it) | running at once |
| Esc menu left open | 15:13:52.5 | +0 | +22 s Esc | closed +0.9 s (first try failed: bug, see below) |
| inventory left open | 15:14:34 | +0 | +90.7 s Esc | closed +0.9 s |
| kill helper (d2input) | 15:16:20.5 | +0.1 s | +10.2 s `game.sh helper` | listening +12 s |
| Save and Exit to main menu | 15:27:15.5 | +0 | +30.5 s oog-start | in game +34.5 s |
| second Game.exe in the prefix ("Diablo II Critical Error: Only one copy...") | 15:28:09 | +2.5 s | +6.7 s Return to the dialog | dialog gone +8 s, original game untouched |
| death | NOT achieved, see below | | | |
Bugs the drills found and fixed: (1) panel list included UI flags 6/7, so the Esc loop pressed Esc 4 times and toggled the menu open again (see Gotcha);
(2) log noise (the `obs` change key contained rule durations).
Death: `d2cheat setstat stat=6 value=0` (and maxlife 0) gives life 0 but the player stays in mode 5 with no overlay and no death (VERIFIED: the server only dies on damage).
Walking out of town to be killed failed (navlive: "no path on the known map" / no movement on i5; nav agent's area). So the real "You have died. Press ESC" overlay and the
Esc-through-respawn path were NOT exercised live; `respawn` refuses to press Esc while the mode is not 0/17 (Esc would open the menu on a living character) and verifies
alive from memory otherwise. The player death mode ids 0/17 are UNVERIFIED (bot's assumption). The sentinel rule fires on life 0 or dead mode, so the dead-overlay state of i4
(whatever mode it shows) will be reported as `would act death-overlay` when run observe-only: check the first real occurrence and adjust `Observation.Dead`.

## Cannot catch
A game that renders and answers but plays wrongly (bot logic loops that move around), items lost, wrong character logged in, a bot that is alive and reports
healthy while useless (no-progress needs a goal flag from the bot or `expectProgress`), a frozen game whose CPU still ticks and whose frame still changes, drops of
the d2sentinel process itself (systemd `Restart=` is not set), Battle.net-style dialogs, i0 (user's desktop; alert only).
