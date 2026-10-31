# Review: evening merges 51c18d9..ee33d7d (agent REVIEW3, 2026-10-29)

Scope: farm runs wiring (internal/boss/farm*.go, internal/bot/farm.go, goals Runs), Nightmare/Hell fight rules
(internal/combat/hard.go, rules.go, internal/resource/hell.go) and the "Normal unchanged" claim, d2bot shutdownSeq,
flee chicken, fleet log rotation / stop grace / goals marker, ToTown nil fix, soak tooling (cmd/d2soak m24,
tools/soak/soak24.sh), tools/winereap.sh + game.sh reap, container image / d2start.sh.

Tags: V = read in the repo or run by me (`go test -p 1 -count=1` of internal/combat, internal/resource, internal/boss,
internal/goals, cmd/d2soak all ok; `bash tools/winereap_test.sh` OK). S = my reasoning, not run. U = unknown, needs a
live check. No code was changed. No game, no the worker host.

## Summary
| # | sev | area | one line |
|---|---|---|---|
| 1 | High | container | d2container SIGKILLs d2fleet 8 s after SIGTERM: no Save and Exit on any `docker stop`, the 90 s grace is dead weight |
| 2 | High | shutdown | worst-case d2bot SIGTERM path is ~120 s, fleet SIGKILLs at 70 s: the save can be skipped or cut mid-write |
| 3 | Med | combat NM/Hell | mana-burn chicken fires on a normal low-mana caster (or any class) near a Mana Burn unit: chicken loop |
| 4 | Med | combat NM/Hell | hardTick runs before the survival step: a mana-burn flee can starve life potions/chicken for up to ~3 s per burst |
| 5 | Med | winereap | prefix path hard-coded to the legacy layout; managed-bot restarts (soak, fleet, container) never reap at all |
| 6 | Med | winereap | not under the instance lock, wineserver check not repeated before kill: TOCTOU against a concurrent start of the same instance |
| 7 | Med | fleet | Fleet.Close drains slots one after another: with 2+ slots the later ones are killed by the container/stop grace |
| 8 | Med | farm | `runs:NAME:N` (Fixed) bypasses the table's `needs` and clvl floor; `Has` is always empty so `auto` never picks Pindleskin |
| 9 | Med | shutdown | leave() runs right after cancelling the runner, before it has stopped: input race on the Esc menu (S) |
| 10 | Med | combat NM/Hell | resist penalty may be double counted (U): if the stat already holds -40/-100 every enchanted pack is +1 danger |
| 11 | Low | farm | loop is restarted with the full Count/Duration and FirstHere on every interrupt (death, level-up points) |
| 12 | Low | combat | antidote every 3 s while poisoned, the antidote's own +poison res is not seen: belt drained |
| 13 | Low | container | token in `?token=` URL, bind-mounted /data not writable for uid 1000 fails with a bare sh error |
| 14 | Low | container | d2container reaper may reap d2fleet before cmd.Wait: exit code is lost (always 0) |
| 15 | Low | soak24 | `--for 1h30m` breaks the arithmetic; wine 24 h cap is redundant with 40/h |
| 16 | Low | fleet | goals marker survives a lost goals store: restarted slot gets `--autostart` and no goals |

"Normal unchanged" (V, read): every new path in fight.go/fighter.go/hard.go is gated on `Difficulty != Normal`
(`hard` in engage, `hardTick` returns at once, `packDanger` adds ResistDanger only off Normal, `PotionPolicyHell` only off
Normal, `PlanForDiff` changes only Hell >= 45, `HellTuning` only Hell; `TestPackDangerNormalIgnoresResists`,
`TestHardTickIsInertInNormal`). Deliberate Normal changes in this range, all reasonable: flee bound (fleeMax 60 s ->
chicken), Anya state decided before the click (task.go), `unreachable()` treats "in town" as WP-capable (task.go:840; S:
benign, clicking an inactive town WP activates it), death telemetry line only off Normal, guard skip for non-allow-listed
characters. `st.spent` is accumulated in Normal too but only read by hardTick. I found no unintended Normal change.

## High

### 1. Container stop never saves (deploy/container/entrypoint/main.go:131-141)
V: `run` handles SIGTERM with `cmd.Process.Signal(SIGTERM)`, waits `8 * time.Second`, then `cmd.Process.Kill()` and
`wineKill()` (`wineserver -k`). The default command is now `run -- d2start.sh`, which `exec`s d2fleet. d2fleet on SIGTERM
first shuts the HTTP server down, then `Fleet.Close` -> per slot SIGTERM to d2bot and up to `-stop-grace` 70 s
(cmd/d2fleet/main.go:283, internal/fleet/bot.go:332). d2bot needs settle (up to 70 s) + Save and Exit (up to 30 s).
Scenario: `docker stop d2idle` (or compose down, host reboot) with a running slot: after 8 s d2fleet is SIGKILLed, its
d2bot children never get SIGTERM, wineserver -k kills Game.exe. All progress since the last Save and Exit is lost; the
`stop_grace_period: 90s` in compose.example.yml (and `--stop-timeout 90` in README) does nothing.
The 8 s predates this range, but the image change made d2fleet the default command, so the first container run hits it.
Fix: make the child grace a flag/env (`D2CONTAINER_GRACE`, default 85 s, below the docker stop timeout), and add a test
in deploy/container/image_test.go that the compose grace > entrypoint grace > d2fleet -stop-grace > d2bot worst case.

### 2. d2bot worst-case SIGTERM path exceeds the fleet grace (cmd/d2bot/live.go:217-292, internal/fleet/bot.go:74)
V: shutdownSeq waits `StopTimeoutSec+10` = 70 s (internal/bot/config.go:82) for the runner to walk home and settle, then
leave (30 s ctx), runnerDone (5 s), apiDone (5 s), game Shutdown (10 s): worst ~120 s. Fleet grace is 70 s.
Scenario: Stop arrives in the field with no TP scroll or a failing ToTown (the soak logs show such cases): settle hits
its 70 s deadline just as the fleet SIGKILLs d2bot, so Save and Exit never runs; or it runs at 65-70 s and is cut
mid-way, then the fleet's CleanGame kills Game.exe while it may be writing the .d2s (U: whether D2 writes the .d2s
atomically; if not, a truncated save).
Fix (S): on SIGTERM there is no reason to walk home first; a single-player Save and Exit from the field is fine and the
next game starts in town. Cap the settle wait at ~10 s on SIGTERM (keep the long wait for a user Stop from the UI), or
derive it from a budget passed by the fleet (`--stop-budget 60s`: settle = budget - 40 s). Add a test that
settleWait + leave + waits < DefaultStopGrace.

## Medium

### 3. Mana-burn chicken triggers on ordinary low mana (internal/resource/hell.go:115-151, internal/combat/hard.go:61-74)
V: `hc.ManaBurn` is true whenever a Mana Burn unit is inside FightRadius. `Observe` counts a "low" whenever the pool is
below 15%, regardless of cause; the `spent` correction only affects the flee test, not `lows`. Two dips -> ManaChicken ->
`Act(Chicken)`. It applies to every class with MaxMana > 0 (no `ManaNeed` check here, unlike PotionPolicyHell).
Scenario: a Nightmare sorceress at clvl 40 that routinely casts down to <15% and waits (the "out of mana: waiting" path)
meets a Mana Burn champion pack: it casts to 10%, potion/regen to 35%, casts to 10% again -> leaves the game. Re-entry
lands next to the same pack (same game seed on re-entry? U) -> reentryLimit 4 in 15 min -> goal paused. A barbarian with
little mana can chicken without ever being burned.
Fix: count a low only when the drop that crossed 15% was foreign (the same spent-corrected window test as ManaFlee), and
gate the whole tracker on `ManaNeed(class) > 0`. Test: casting-only drops near a Mana Burn unit never chicken.

### 4. hardTick pre-empts the survival step (internal/combat/fight.go:176-182)
V: hardTick runs before `f.Safety(snap)`; on ManaFlee it evades and `continue`s, so Safety (life potion, rejuv,
chicken) is skipped for that tick. The ManaFlee condition persists while a >20% drop is inside the 3 s window, and the
evade itself has no counter (unlike kites). Scenario: Hell, life 30%, a Mana Burn + Extra Strong pack: several ticks of
backing off without drinking. S: bounded by the 3 s window, but that is exactly the time a Hell pack kills in.
Fix: move hardTick after the survival step, or run it only when `pct(life) > DrinkLifePct`; count its evades into
`st.kites`.

### 5. winereap is a no-op on the new layout and never runs on the managed path (tools/winereap.sh:14, tools/game.sh:22-35)
V: `prefix=${D2_INST_ROOT:-$HOME/Games/d2lod-inst}/$id/proton/pfx` hard-codes the dev-era layout; tools/lib-paths.sh
`d2_game_root` already returns `<data>/prefixes/<id>` once `<data>/prefixes/i0` exists or ~/Games/d2lod is gone. After
that migration the script prints "0 orphaned" forever. V: reaping is called only from `game.sh stop/reap`; the managed
d2bot (soak24, fleet, container) restarts Wine itself (Recycler, ladder rung 4, game.Shutdown) and has no reap step
(no `winedevice` in any .go file). Per docs/leakhunt.md each wineserver restart leaves ~35 MB and 15 threads.
Fix: source lib-paths.sh and use `$(d2_game_root $id)/proton/pfx` (send it in front like live.sh does); port the same
logic to Go in internal/launch (after `wineserver -k` in Managed.Stop/recycle, same three conditions) so every managed
restart reaps; let d2soak sample a `winedevice.exe` count so the soak can see the growth.

### 6. winereap kill safety (tools/winereap.sh:23-43)
Question asked: can it kill another instance's or the user's Wine processes? V: no, by construction: a kill needs comm
`winedevice.exe`, our uid, and an environ WINEPREFIX equal (after stripping trailing /) to this instance's prefix; other
instances and the user's own Wine/Steam/Lutris prefixes differ; i0 is refused; unreadable environ = no match (the safe
direction); the fake-/proc test covers other instance, other uid, other program, missing environ, other prefix's
server, live server. Residual risks, all against the SAME instance:
- TOCTOU (S): `game.sh` runs the reap after d2game has exited and released the instance lock (after_stop). The scan is
  slow (2-5 forks per /proc entry, ~1000 entries on the worker host: seconds). If `game.sh start iN` or the managed bot launches
  iN in that window, phase 1 saw no wineserver and phase 2 kills the fresh instance's winedevice.exe (driver host),
  leaving a half-started prefix. The soak24 runbook (stop, then start) makes this plausible when the operator is fast.
- PID reuse between check and `kill` (negligible).
- If a Proton build names its server differently (`wineserver64`, U) the live-server guard fails open and a running
  instance's drivers are killed by `game.sh reap iN`. Unverified live: the script has only run on fake /proc.
Fix: take the instance lock (`flock -w 10 $D2LOCK_NEW`) for the whole reap, re-check "no wineserver of the prefix"
immediately before each kill, and additionally require that the process's parent is not a live process of the same
prefix (orphans have ppid 1 or the subreaper); first live use with `--dry-run` on a stopped instance and compare with
`ps -o pid,ppid,etime,comm`.

### 7. Fleet.Close is sequential (internal/fleet/fleet.go:529-538)
V: `for ... f.Remove(ctx, id, Force)`; each Remove cancels the slot and waits `<-done`, i.e. up to the 70 s stop grace,
before the next slot is even signalled. Scenario: 2 slots in one container, `docker stop` (once #1 is fixed): slot 2
gets SIGTERM after ~50-70 s and is SIGKILLed by docker at 90 s, no save. Fix: cancel all slots first, then wait for all
(parallel drain), and size stop_grace_period for one slot, not N.

### 8. Farm run selection gaps (internal/boss/farm_loop.go:149-155, internal/bot/farm.go:42)
V: `pickRun` with `Fixed` returns `tab.Find` without `hasAll(c.Has, r.Needs)` or the clvl floor; `farmOps.Char` always
returns `Has: map[string]bool{}`. Consequences: `runs:auto:N` can never pick Pindleskin (`needs: anya_portal`,
plandata/farm-runs.json:169); `runs:Pindleskin:N` on a character without Anya rescued walks to Harrogath, clicks a
missing red portal (`objRedPortal = 60` is S/U), waits 10 s, fails, 4 times (ConsecFails) -> "fails" -> error ->
r.fail -> retried by the goal loop. Fix: fill `Has["anya_portal"]` from the quest reader (Prison of Ice done), and apply
needs/floor to Fixed too, returning a clear "run not unlocked" error that pauses the goal instead of retrying.

### 9. Save and Exit races the cancelled runner (cmd/d2bot/live.go:282-286)
V: order is stopRunner (cancel) -> leave -> runnerDone (wait). S: a runner cancelled mid-action may still send input on
its way out (release a held key, Esc a panel, finish a click) while LeaveGame opens the Esc menu; one stray Esc closes
the menu and the Save and Exit row click lands in the game. The H4 test only checks call order with fakes. Fix: wait
for runnerDone up to ~2 s before leave (keeps the save early), and have LeaveGame verify the menu is open before the
click (it may already; check launch.Flow.LeaveGame).

### 10. Resist penalty may be applied twice (internal/combat/rules.go:298-302, hard.go:24, d2/player.go:43)
U, flagged in code: if the stat read already contains the difficulty penalty, `Effective` subtracts it again; a
Nightmare character with 30 fire res would be treated as -50 and every FE/CE/LE pack becomes one danger step higher
(more retreats/aborts), and `AntidoteNeeded` fires for anyone below 65 poison res in Hell. Fix: one live read on a
Nightmare character (d2doctor DoctorLine) before any NM/Hell soak; also drop the duplicate `ResistPenalty` (combat and
d2 both define it).

## Low
11. internal/bot/farm.go:78: every re-entry starts a fresh FarmLoop with `Runs: p.Count`, `Duration: UntilMins` and
    `FirstHere: true`. runAction cancels it on errDied, errPoints (level-up), errNewGoal etc. (loop.go:339), so after a
    level-up the loop resumes in the same game and tries a boss that is already dead (one counted fail). Goal progress
    (BossKills) still ends the goal correctly. Fix: pass `Count - Progress.BossKills` and remaining minutes; FirstHere only
    when the last run did not finish in this game.
12. internal/combat/hard.go:52-59: antidote every 3 s while `Poisoned` and the read poison res < 25; the antidote's
    temporary +50 res and poison immunity window are not modelled, re-poisoning drains the belt. Use a 30 s cooldown.
13. deploy/container/README.md "setup.html?token=TOKEN": the token lands in browser history and any reverse proxy log;
    prefer a login form/cookie. d2start.sh:6-8 with a root-owned bind mount: `set -eu` exits with "Permission denied"
    and the container restarts in a loop; print a hint (`chown 1000 <dir>`). Token creation itself is fine (V: 64 hex
    chars from urandom, umask 077 -> 0600, only the path is logged).
14. deploy/container/entrypoint/main.go:63-75, 122-130: `launch.Reap` skips only Spawner-registered pids; d2fleet is a
    plain exec.Cmd child, so the 2 s reaper can wait4 it first and `cmd.Wait` returns ECHILD -> exit code 0; docker
    `restart: on-failure` would not restart a crashed fleet. Register the child in `owned` (or use launch's Spawner).
15. tools/soak/soak24.sh:89: `--for 1h30m` -> `${for_%m}` = `1h30` -> arithmetic error under set -e; accept plain
    minutes or use a Go duration parse via d2soak. cmd/d2soak/analyze.go: `WineMaxGrowth24h 2000` can never trigger
    before `WineMaxFDsPerHour 40` (40 x 24 = 960); harmless, but one of them is dead.
16. internal/fleet/bot.go:211-219: the `goals-seeded` marker is keyed on character|goals in the instance dir; if the
    d2bot goals store is lost/reset the slot restarts with `--autostart` and no goals (idles). Also check the store
    exists before omitting `--goals`. rotWriter (bot.go:422-434): a failed reopen leaves `w.f` closed and every later
    Write errors; acceptable, but log it once.

## What the first 24 h soak will hit (S)
- Booti is clvl ~12 in Act 1 (docs: LEVELRATE), so the run stays in Normal: none of the NM/Hell rules (#3, #4, #10)
  run. The Normal fight loop is unchanged apart from the flee bound, which can now chicken after 60 s of potion-less
  fleeing; expect a few chicken re-entries in the log, watch reentryLimit pauses.
- Orphaned winedevice.exe accumulate on every in-bot Wine restart and are not reaped (#5) nor sampled by d2soak; RSS of
  the host grows by ~35 MB per restart without failing any m24 limit. Check by hand at the end:
  `pgrep -c winedevice.exe` and `tools/game.sh reap i4 --dry-run` (but see #5: verify the prefix path first).
- The stop at the end (`soak24.sh stop`, pkill -TERM) has no external SIGKILL, so #2 does not bite there; the bot's own
  `--run-for` exit runs the same shutdownSeq.
- m24 numbers are U; the wineserver fd judgement needs >= 2 h post-warm-up, fine for 24 h.

## What the first container run will hit (S)
- Wizard serves on 127.0.0.1:8431 (V: EXPOSE 8431 only, 0.0.0.0 inside, the server requires the token for a
  non-loopback bind, the setup API is behind the token).
- First `docker stop` / compose down / image update: #1, the game is killed after 8 s without Save and Exit; if the
  wizard is mid-install or mid-character-creation that state is cut as well. Fix #1 before the user test.
- A bind mount instead of the named volume: #13 permission error.
- Orphaned winedevice per restart inside the container (#5) count against `mem_limit: 3g`.

## Recommended order
1. #1 (entrypoint grace) and #2 (short settle on SIGTERM), with one test that checks the whole grace chain.
2. #5/#6 together: Go-side reap in internal/launch under the instance lock, legacy script uses d2_game_root.
3. #3/#4 before any Nightmare soak; #10 live check with a Nightmare character.
4. #7, #8, #9; the Lows as they come.
