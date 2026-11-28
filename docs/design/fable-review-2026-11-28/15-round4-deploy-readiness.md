# Round 4: deploy readiness of main for the 06:30 EST canary autodeploy (2026-11-28)

**VERDICT: GO** for code tree `6e37fec73` (candidate head `cb6cbe67a`), deployed over `42bd6d967`. Conditional: see line 4.
1. Nothing found that can wipe or corrupt the Doofer save, kill the hero by a new code path with certainty, or crash-loop the bot. No blocker.
2. The save repair cannot fire on this deploy: the canary has no `savebackups/` folder yet (V, listed in the container), and repair needs a valid backup. The first start only begins writing backups.
3. The real live `Doofer.d2s` (1625 bytes, version 96) passes the savesafe validity rule: signature, size field, checksum (V, three reads of the live file checked with an independent re-implementation). So backups will be written, not skipped.
4. Scope limit: only the tree above was read. If a later code merge (dur2, drv2, road5, zone5 were NOT in main at review time) becomes the candidate before 06:30, this verdict does not cover it.
5. Safety net that makes GO acceptable unattended: autodeploy watches 900 s and rolls back on 3 restarts or a hard invariant (V, autodeploy.py in the ops repo); free space on the canary data volume is about 149 GB (V), so the disk guard is inert.
6. To hold instead: `systemctl --user disable --now d2-autodeploy.timer` on the worker host. `autodeploy.py` has NO `--hold` option (V: only `--dry-run`, `--commit`, `--release`).
7. Brief correction: `bot.zoneModel` is 1 on the canary (V, tuning.json override), not off. The fix14 hole hold is therefore live there (finding H2).
8. Not done: no test run by this review (mergeq gate and CI green are taken from the autodeploy journal, V for the log line only); nothing verified live.

Tags: V = read in the repo or run; S = reasoned from code, not run; U = unknown.

## Question 1: what could kill, wipe, crash-loop or stall

| Risk asked | Answer | Tag |
|---|---|---|
| Repair restores an OLD backup over a good save | Only when the live file fails `Valid` AND a valid backup exists. A good file is never replaced; the replaced file is kept as `.broken-<unix>`. One real hole: H1 (repair runs before the "a d2d already answers" check). | V code, S hole |
| Backup written while the save is mid-write | The `.d2s` is read in one `ReadFile` and must pass the checksum, else skipped (`savesafe.go` Save). Side files (`.map`, `.ma0`..) are copied unchecked: a torn side file can enter a set, and is used only if the live side file is missing or empty at a repair. | V code |
| Restore loop | None in the bot: repair errors are logged, never returned (`savesafewire.go:47-60`). A restored file is valid, so the next start does nothing. | V |
| Disk guard refuses the save point and loses xp | Only under 200 MB free; canary has ~149 GB. When it does trigger, xp at risk grows without bound and only planned save points are covered: deaths, recycles and `Restart` still make d2d write on the full disk (M3). | V code |
| Backup growth | Canary save folder holds about 72 KB (V). 19 sets at most: under 2 MB. `*.d2x`/`*.sss` are capped at 16 MB each per set (worst case ~300 MB per shared file, U: no such file exists on d2d). `.broken-*` files and `.tmp-*` leftovers are never pruned (L). | V |
| Vault | savesafe does not snapshot the bot-side vault store. A repair rolls the hero back and leaves the vault as it is (M2). | S |
| Red-bar notices endpoint | Read only, 60 s poll, one statfs and one small file read per call. `withScriptTag` now buffers every `/watch*` HTML GET even when no updater is set; no streaming route sits under that prefix without a dot (V grep; SSE is under `/api/`). | V |
| sec2 token and log changes | `/api/log` now answers 405 to anything but GET/HEAD; `ops log` output is redacted and made printable; temp files removed before a 0600 write. No token check changed in this diff. Board impact: none found. | V diff |

## Question 2: GO / NO-GO by commit

| Merge | Verdict | Reason |
|---|---|---|
| savesafe `010befbd5` | GO | Inert repair on first start; validity rule matches the real save; all failures are log-only; guard goroutine runs under `goSafe`. |
| fix14 `dc5419a81` | GO with H2, H3 open | Hole hold is live on the canary (stall risk, not lethal). Objective-level kit rule is U live. Merc journal changes only keep more. |
| fix8b `6e37fec73` | GO | Doofer is a Necromancer (class 2, V wizard.env), class mana need already 2: belt columns unchanged. `colsForNeed` is equal to the old switch for needs 0, 1, 2 (V by reading). `no-combat-profile` is now a hard invariant: a profile loss rolls the deploy back, the safe direction. |
| evid1 `6a0008aab` | GO | Logging only; `cpHeld` keeps the old order of `cpDecide` (V by reading). |
| sec2 `0cdf10521` | GO | See table above. |
| ledenable, flipfix2 | already deployed (`42bd6d967` contains them) | - |

## Question 3: findings

### H1 (high consequence, low odds): repair runs before the bot knows whether a d2d is still writing
`cmd/d2bot/d2dbackend.go:251-253` calls `repairSaveAtStart` before `d2denv.Build` and before the `answers(ctx, be.Proc)` check at `:258`. If a previous d2d survived (the bot then exits with "a d2d already answers", and the launcher restarts it), every restart reads a file that a live d2d may be writing in place. A torn read fails the checksum, the live file is renamed to `.broken-*` under the running d2d and an older backup is put in its place. Loss is bounded (the `.broken` file is kept), but the hero silently loses up to ten minutes and the running d2d keeps writing to the renamed inode.
Recommendation: skip the repair when the control socket answers; do the repair only on the path that is about to start d2d.
Must-fail test: fake `answers` true, a live file with a bad checksum and one valid backup; assert the live file is byte-identical afterwards and no `.broken-*` exists.

### H2 (high for an unattended night): hole hold waits in town with no status and a fed heartbeat
`internal/bot/task.go:1232-1241`: while `holeHeld` is set, `freshGameToLevel` pauses 5 s, calls `r.Beat()` and returns done. No `setState`, no log line, no bound. The step watchdog is fed, so a hold that never ends is a silent stall. The canary has `bot.zoneModel` 1 (V).
Recommendation: one status line ("Waiting in town: no zone with margin") and a log line once per hold; a ceiling after which the old fresh-game path runs.
Must-fail test: a model that always answers the hole hold; run N steps on a fake clock past the ceiling; assert a status naming the hold and that the fresh-game path ran once.

### H3 (U): objective-level kit refusal for a hero already on the road without a portal
`internal/admit/admit.go:150-157` and `internal/bot/earlyroute.go:113-117`. The comment above it records (V, lab night) that a refusal inside the lair parked the chunk and left the hero idle. The new exception re-introduces a refusal there for the boss or chest level; without a portal it returns `KitOnly=false`, the same class as "too weak". The code comment itself says "U: not shown live". A hero standing in a dungeon level with few potions and no portal is the death case this rule wants to avoid.
Recommendation: a drill for "on the road, no portal, potions under the kit, next level is the objective": assert the hero walks out or holds in a cleared spot with a status, within a bound.
Must-fail test: `MayEnter` with `v.Area` on the road, `HasTP=false`, healing under the kit, target in `ObjectiveAreas`; then the runner step: assert it does not end in "Waiting for a fresh game" and the hero leaves the level.

### M1: a missing live file plus old backups resurrects a deleted hero
`savesafe.go` Repair, "live file missing" branch. The Characters page (`internal/charsel`) can delete a hero; the runbook says to move the backups away first, the code does not. Recommendation: charsel delete also archives `savebackups/<name>.*`. Test: delete through charsel, start, assert no `.d2s` reappears.

### M2: hero and vault are not restored together (S)
A repair goes back to the backup made at the last town arrival; the vault store keeps the later state. Items taken out of the vault after that backup are lost, items put in are doubled. Recommendation: copy the vault store into the same set and name the gap in the repair notice.

### M3: the disk guard covers only planned save points
`internal/bot/savepoint.go:74`. Deaths, recycles and `Deps.Restart` still save on a full disk. Recommendation: the same check in the recycle and restart paths, or the fork's atomic save.

### L
- `checkpoint idle` (`internal/bot/checkpoint.go` noteIdle) remembers only the last area and reason: reasons that alternate ("hostiles near" / "far from the entry") log on every change. Key the gap per area and reason.
- Notice text carries absolute folder paths to every viewer past the gate (`savesafe.go` Notices).
- `sidestepMissiles` now ranks missiles before the cheap gates on every call (`internal/combat/dodge.go:94-104`).

## What to look at in the morning
`d2bot ops save check` in the canary (expect one `live ... valid` line and backup lines), `ops status --why` for `notices`, the deploy log line for the candidate (`ok` or `ROLLED-BACK`), and the log for `save backup: skipped`.
