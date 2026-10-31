# Review: merges ffecf16..f3b7a45 (2026-10-29, REVIEW2)

Scope: 124 commits, 171 files. Focus: Acts 2-5 campaign wiring and difficulty chaining, the fleet bot launcher
(bot-owned game), the combat changes (affix, curse retreat, weak-pack swing, poverty), and the cheat package.
Tags: **V** = read in the repo or run; **S** = game knowledge / opinion; **U** = not checked.

Checks run (V): `go test -race -p 2 -count=1` over internal/fleet, cmd/d2fleet, internal/bot, internal/goals,
internal/task, cmd/d2bot, internal/combat, internal/cheat on the worker host (labrun): all ok, no races. `go vet ./...`: clean.
A throwaway test (not committed) of `task.Template("nightmare_all").Report` on a Normal world: see H1.

## High

**H1. A Nightmare/Hell campaign is marked Done in a Normal game (and chains on to Hell).** V (run).
`internal/task/campaign.go:82` returns `Stuck` for the wrong difficulty, but `internal/task/progress.go:13` (`Report`)
still counts `QuestDone(w, q)` from the world, and the world's quests are the *current game's* difficulty
(`internal/bot/task.go:255`, `Read(int(p.Difficulty))`). `internal/bot/task.go:195-196` publishes that report after
every tick, also the Stuck one, and `internal/goals/goal.go:458-464` finishes a campaign at `Current >= Target`.
Scratch test: normal world with every quest done, `nightmare_all` reports **Current=28 Target=28, stuck=true**.
Scenario: `normal_all` (chained) completes, the chained `reach_level` ends with the character in the field, the
`nightmare_all` goal starts, `ensureDifficulty` cannot switch out of town (H2), the tick reports 28/28, the manager
marks Nightmare Done, `chainLocked` queues `hell_all`, which then does the same thing. The user sees "Campaign:
Nightmare done" without one Nightmare quest played.
Fix: in `Report`, when `c.wrongDifficulty(w)` return `Current 0` (or don't publish: `setTaskReport` only when the plan
is not wrong-difficulty). Add a test: wrong-difficulty world never reports Current > 0.

**H2. Difficulty switch is tried once per clvl, for the life of the process; failure leaves a permanent pause.** V.
`internal/bot/goal.go:489-496`: `diffTried[p.Level]` is set *before* the in-town and Cycler checks, and is only reset
in `NewRunner` (`internal/bot/runner.go:216`). If the campaign goal starts while the character is out of town (the
normal case after a `reach_level` goal ends mid-sweep, or after Baal in the Worldstone Chamber), the one try is spent
logging "difficulty wish not met", then every tick goes to `taskStep`, `Plan` is Stuck, and `internal/bot/task.go:202-203`
calls `selfPause` **without** `autoResumeIn`. A failed `NewGame` sets `diffBan`, but campaigns never read it, so even
Resume does not help until the character levels up or d2bot restarts. Also the campaign never walks to town first.
Fix: key `diffTried` by (goal gen, level) or clear it on goal start/resume; do the in-town check before marking; for
a campaign with the wrong difficulty, go to town (`r.rest`) and retry the switch with a backoff instead of Stuck; give
every self pause on the unattended path an auto resume (as `loop.go:519-520` does).

**H3. Chicken exits never block a boss quest: `ForgetFails` on every new game resets the death count.** V.
`internal/bot/task.go:367-377` maps `ErrLeftGame` to `KindDeath` "so a boss that is too strong does not loop forever",
but a chicken is Save and Exit, i.e. a new game, and `syncTaskGame` (`internal/bot/task.go:308-312`) calls
`ex.ForgetFails()` which clears `deaths` (`internal/task/executor.go:130-133`). `deaths[step]` never exceeds 1, so
`MaxDeaths` (3) is unreachable for chickens; the same holds for `boss.ErrTooWeak/ErrAborted` (`KindDanger`, `fails`).
The only remaining brake is `reentryLimit` 4 per 15 min (`internal/bot/goal.go:773-774`): a Diablo or Baal attempt
takes longer than 4 minutes, so a too-weak character loops travel, seals/waves, chicken, new game for 24 h (potions,
gold, repairs burnt each time). When attempts are quicker (Andariel from the WP) it hits the reentry pause, which has no
auto resume, so the soak halts.
Fix: keep death/chicken/danger counts per quest across games (forget them on level up, not on new game); after
MaxDeaths turn the block into a level step (raise that entry's floor by a few levels) instead of a block; auto resume
the reentry pause.

**H4. Fleet SIGKILLs d2bot after 10 s, but d2bot's `--stop-game` shutdown needs up to about 50 s.** V (read).
`internal/fleet/bot.go:105-106` (StopGrace 10 s, `cmd/d2fleet/main.go:208` does not override) vs
`cmd/d2bot/live.go:225-247`: runner wait 5 s, API wait 5 s, then `LeaveGame` up to 25-30 s, then `Shutdown(10 s)`.
Remove, Restart (`kick`) and `d2fleet serve` SIGTERM all hit this. The kill lands in the middle of Save and Exit, so
the .d2s (quests, waypoints, XP since the last exit) is not written; in bot-owned mode the fleet does no OOG leave of
its own (`internal/fleet/fleet.go:392-397`). docs/container.md already lists "save and exit within 10 s" as U. A
container `docker stop` default (10 s) is worse.
Fix: StopGrace >= 60 s for the bot-owned model (configurable), d2bot leaves the game *first* on SIGTERM (before waiting
on runner/API), and document `stop_grace_period: 90s` for the container.

**H5. Goals are marked "seeded" before d2bot has stored them: a failed first start loses the user's goals.** V.
`internal/fleet/bot.go:307-311` writes `goals-seeded` right after `Start`, but d2bot only enqueues `--goals` after
`openGame` succeeded (`cmd/d2bot/live.go:68` then `:146-156`; `openGame` = helper 60 s + `Launch` 90 s + attach). A
first container run is exactly when the game launch fails or times out (fresh prefix, first Wine boot). Every restart
then passes only `--autostart`, the manager has no goals, the slot shows `running` and does nothing.
Fix: mark seeded only after d2bot reports the goals stored (log line "goals enqueued", or check
`instances/<id>/goals.json` contains them), or let d2bot enqueue `--goals` idempotently (skip a goal whose
type+params already exist) and always pass them.

## Medium

**M1. Fleet `watch` treats any game absence of one poll (5 s) as death, also d2bot's own recycles and restarts.** V.
In bot-owned mode `internal/fleet/fleet.go:352-365` polls `Runtime.Alive` (game pids) every 5 s. d2bot itself stops
and relaunches the game (`cmd/d2bot/recycle.go:88-101`, Recycler; `internal/bot/loop.go:374+` gameLost restart). The
fleet then SIGTERMs d2bot mid-launch, runs `Runtime.Stop` and restarts the slot: every planned recycle becomes an
unplanned slot restart with H4's save risk. Similarly after a d2bot crash, `ProcBots` restarts d2bot in 2 s
(`bot.go:277-284`) while the fleet kills that new d2bot when its poll sees no game, so each crash costs two launches
and the inner MaxRestarts never applies (it is per `Run`). Fix: in bot-owned mode supervise the d2bot process (and its
/healthz) and let d2bot own game liveness; or require N consecutive misses spanning > GameWait.

**M2. `CleanGame` = `Runtime.Stop` also kills the Xvfb and helper the fleet prepared.** V (read), U (live effect).
`cmd/d2fleet/main.go:209` and `internal/fleet/runtime.go:81`. The restarted d2bot then creates Xvfb
(`cmd/d2bot/game.go:121-124`) and the helper (`helper.Ensure`) as its own children, a different process model from the
first start and from what the tests cover. Fix: a `StopGame` that kills only Game.exe/wineserver for CleanGame.

**M3. Normal templates do not pin the difficulty.** V. `internal/goals/goal.go` `templateDifficulty` returns "" for
`normal_*`, so a wizard user who picks `campaign:normal_all` for a character whose last game was Nightmare plays the
Nightmare quests with Normal floors (no EntryGate), and `ensureDifficulty` never switches (`want == ""`). Fix: return
"normal" for the normal templates.

**M4. Iron Maiden "retreat" only drinks; the fighter keeps swinging.** V (read), S (IM reflects melee damage).
`internal/combat/fighter.go:219-229`: below `CurseRetreatPct` (60) it drinks whenever a potion is ready, and portals
only once the belt is empty. Under Iron Maiden a melee character (or a caster using the new weak-pack staff swing,
`internal/combat/resource.go:47-49`) keeps attacking and drinking until the belt is gone, then dies if it has no scroll.
Fix: under Iron Maiden suppress melee (and the weak-pack swing) and step back until the curse ends; drink only below
the normal drink line.

**M5. Weak-pack swing uses base monster level and fail-open affixes.** V. `weakHostiles`
(`internal/combat/resource.go:104-117`) compares `monstats` level (no champion +2 / unique +3) and lifts the
`MeleeMaxPack` cap; the only brake is `PackDanger < 2`, whose affix part reads `EnchantIDs`, which returns "no affixes"
while the monster-kind offsets are UNVERIFIED (`internal/d2/monflags.go:24,44-61`, `internal/combat/target.go:47`). A
low clvl sorceress can walk into melee with a Lightning/Fire Enchanted champion pack of any size. Fix: exclude
champions/uniques/minions from "weak", keep a pack-size cap, and treat unknown kind as not weak.

**M6. Placing a quest item can leave it on the cursor.** V. `cmd/d2bot/objects.go:318-384` (`placeItem`): after five
failed clicks it returns an error with the Staff or Soulstone still held; the next walk click drops it on the floor.
The retry then fails "no hst in the pack" (`KindPrereq`) and the quest needs the item again. Fix: on failure put the
cursor item back into the inventory (`hand`), and verify the cursor is empty before returning.

**M7. A step that overruns the 40 min `r.do` budget retries forever without a task-level count.** V.
`internal/bot/task.go:182` (40 min) cancels Tick's ctx; `internal/task/executor.go:268-270` then reports
`OutInterrupted`, which counts nothing. Baal: travel + waves (30 min default, `internal/boss/waves.go:55-56`) + Baal
(20 min, `internal/boss/act5.go:285`) can exceed it. Fix: give boss steps a `Step.Budget` below the guard budget, so the
executor sees KindTimeout and counts it.

**M8. Only the failsafe-ladder pause auto-resumes.** V. `internal/bot/loop.go:519-520` vs task Stuck
(`task.go:203`), reentry (`goal.go:864`), leveler (`loop.go:611`). Any of these ends an unattended soak. Fix: a common
auto-resume (5-15 min) for every self pause unless the user paused.

**M9. d2bot under the fleet has no `D2BOT_INSTANCE` in its environment.** V. `internal/fleet/bot.go:390` inherits the
fleet's env; `--instance` sets only the flag (`cmd/d2bot/main.go:65`). `cheat.DefaultEnv` (`internal/cheat/cheat.go:91-107`,
used by the save guard `cmd/d2bot/behaviors.go:90`) then opens `mem.OpenGame()` over *all* games: with two slots the
guard of i2 attaches to i1's game. Writes still need an allow-listed name and a god marker, so the harm is small, but it
is a wrong-game access on every Save and Exit. Fix: set `cmd.Env = append(os.Environ(), "D2BOT_INSTANCE="+a.ID)` in
ProcBots, and give the guard the managed game's source instead of `OpenGame`.

## Low

- **L1** `cmd/d2soak/d2soak`: a 10 MB binary is committed (0b06c6a); `.gitignore` `/d2*` only covers the root. `git rm
  --cached` it and ignore `cmd/*/` build outputs. V.
- **L2** `internal/goals/chain.go:57-75`: if the reach_level add succeeds and the campaign add fails, `NextIDs` stays
  empty and `resumeChains` adds another reach_level on the next start. Set `NextIDs` after each successful add. V.
- **L3** `internal/fleet/bot.go:249`: the d2bot log is rotated only when a `Run` starts; a long-running d2bot grows
  `d2bot.log` without bound (boss view logs every unique every 3 s). Rotate in `lineTap` by size. V.
- **L4** `internal/bot/task.go:832`: `w.WPs[cur] || s.Player.InTown() && w.WPs[cur]` is just `w.WPs[cur]`; the in-town
  case was likely meant without the WP check. A town whose WP is not in the reader refuses every long route. V.
- **L5** `internal/bot/task.go:574-584`: Anya's pseudo item is decided by scanning the pack *after* the click; if the
  thaw consumes Malah's potion (S) the bot records "touched", not "thawed". Proxies recover via the Scroll of
  Resistance; decide from the pack before the click. V (order), S (consumption).
- **L6** Hell floor 75 for every Act 1 quest (`internal/task/nightmare.go:52-58` with dungeon=true for all of Act 1)
  makes `ReadyForNext("hell")` 75, so the chained reach_level farms Nightmare to 75, a very long grind. S.
- **L7** The save guard calls `cheat.Env.Do` on every Save and Exit for every character, appending a failed ledger line
  for non-allow-listed (real user) characters. Skip the guard when the character is not allow-listed. V.

## Cheat package (unchanged since ffecf16)

V: `attach` checks allow-list and offline markers before any memory op (`internal/cheat/memory.go:57-81`); every op
requires a reason and a writable ledger (`internal/cheat/cheat.go:121-162`). Concerns: M9 and L7 above; the allow list
includes a user-editable file (`~/.config/d2lod/cheat-allow.txt`) that, inside a user's container, would let any
character name be cheated on; ship the container with the built-in list only (or `D2CHEAT_ALLOWFILE=/dev/null`). S.

## What a 24 h unattended soak hits first (S, ranked)

1. A self pause with no auto resume (M8): reentry pause after 4 chickens in 15 min at a quick boss (Andariel/Countess),
   or task Stuck once every remaining group is blocked. Progress stops; d2soak flags a stall.
2. If boss attempts are slow: the endless chicken loop of H3 (Andariel/Duriel/Diablo) with gold and potions drained.
3. If the run reaches the end of Normal: H2 + H1 (Nightmare "done" instantly, or a permanent pause after Baal).
4. M1 at the first Recycler run (500 cycles, roughly 20 h) if it runs under d2fleet.

## What the first container user test hits first (S, ranked)

1. H5: the first d2bot start fails or times out on a fresh prefix; the goals are lost, the slot shows running, idle.
2. H4: the user removes/restarts the slot or stops the container; Save and Exit is killed, progress since the last
   exit is lost.
3. M1/M2: any game hiccup becomes a full slot restart, after which Xvfb/helper are d2bot's children.
4. M3: a character last played in Nightmare picks "Normal campaign" and plays Nightmare ungated.

## Recommended order

H1 and H2 together (small, pure, testable offline), H5, H4, H3 plus M8, then M1/M2 before the container test, M3,
then the combat items (M4, M5) before Act 4 runs.
