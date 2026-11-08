# palready: Might aura once per fight, recovery leaves a dry field (TASK-180)

> status: aura fixed and verified live 2026-11-08 (EST); dry-recovery rule proven offline only (it did not fire in the 22 minute realtime drill). Tags: V verified live (how stated), S read in code, U unverified. Follows docs/palmelee.md and docs/palpace.md.

## 1. `combat aura did not come up` (V)
Root causes, three layers (all V by drill logs `bot-pr1..pr9.log` on the worker host):
1. On d2d a hotkey cast is `cmd skill <id> x y`: it casts at the point and never changes the selected right skill (aura never selected). Replaced.
2. `cmd select <id> 0` is accepted (`ok`) but `rskill` stays 0, also with the picker open (`state picker=2`) and after 10 stepped frames (PALDBG probe, pr5/pr6). So the hotkey-less path cannot be the id verb.
3. The working path is the right skill popup that game.exe uses: `SetRight` clicks the right skill button and the icon cell, and reads `rskill` back. It was dead on d2d because the d2d stack never loaded the skilldesc ListRow table (`rows 0, cells 0` in the failure log) and `SetRight` tried the pseudo hotkey `sk98` first.
4. Spam: a failed apply set `auraChecked = now+10s`, but the dirty flag (area change, death) bypasses that check, so every combat tick pressed again (2485 lines in 6 virtual-hour drill pr1).

Fix: `Fighter.applyAura` (internal/combat/aura.go) uses `SetRight` when `Fighter.D2D`, else the hotkey as before; `SetRight` skips pseudo keys on d2d (bindleft.go); `cmd/d2bot/d2dwire.go` loads the ListRows; `auraRetryAt` backs a failure off 10 s even when dirty. Aura choice stays data (profile ladder, bands); no class literal.
Tests: `TestAuraComesUpOnceOnD2D` (Might band clvl 1, Holy Fire band clvl 13; fails before: aura 0), `TestAuraFailureBacksOff`, in internal/combat/class_test.go. Build-specific: the Paladin profile only (internal/combat/classdata).

| lockstep drill `run-a.sh TAG 7`, no setup writes | before (pr1, 6 min wall) | after (pr9, 2 min wall) |
|---|---|---|
| `aura did not come up` lines | 2485 (pr2 with only the backoff: 244) | 0 |
| `aura up` | 0 | 4 in 2 min (one per area change or respawn, tries 1) |
Realtime `run-b.sh pr10 9 22m`: 0 failure lines, `aura up` once (before pm2: 140 failure lines, 0 up).

## 2. Recovery in a dry field (S offline, U live)
Rule `dryRecovery` (internal/bot/drought.go, knobs `bot.recoveryDryMin` 5, `bot.recoveryDryKills` 5, `bot.recoveryDrySkipMin` 15; 0 min = off): while recovery farms gold, fewer than 5 kills in 5 minutes in the same field area means the area is dry. The area is skipped for 15 minutes (`skipArea`) and the planner picks the next area `recoveryArea` allows (route guard, kit gate, bans and safe tiers still decide; level proposals of internal/decide/proposezone.go are not touched). When `recoveryArea` has no other area, the skip is withdrawn and a fresh game follows (drought stage 2, so `freshAfterRecoverySec` and a missing cycler still apply). Class free.
Tests: `TestDryRecoveryLeavesTheField` (Sorceress and Barbarian; fails with the rule disabled), `TestDryRecoveryQuietCases`.
Not shown live: in pr10 the hero killed about 2 per minute (21 / 36 / 56 kills at minute 5 / 10 / 20), so no 5 minute window had fewer than 5 kills and the rule never fired; the depleted stage of pm2 (gap 141 s) starts after minute 20. The move-on branch (another eligible area exists) has no offline test either (the test rig has none).

| realtime `run-b.sh`, seed 9 | before pm2 | after pr10 |
|---|---|---|
| minute 5 / 10 / 20 | clvl 2 / 2 / 2, kills 22 / 37 / 62 | clvl 1 / 2 / 2, kills 21 / 36 / 56 |
| xp at minute 20 | 1371 | 1278 |
| gap since last kill (s) | median 6, p90 45, max 184, over 60 s: 6, over 120 s: 3 | median 4, p90 53, max 277, over 60 s: 6, over 120 s: 2 |
The two runs are equal within noise (different RNG outcome); the pacing rule is not credited with any gain. writes after start: 0 (lab verbs off, no d2cheat).

## Dead ends
`cmd select` with the picker open (above); `debug skill right` would work but is a debug write (CRUTCH RULE), not used.

## Bring it back
`docs/probes/paladinsmoke/run-a.sh TAG SEED RUNFOR` (lockstep) and `run-b.sh` (realtime) on the worker host; `python3 -I docs/probes/paladinsmoke/logstats.py LOG...` prints the numbers above.
