# abmerc: paired A/B of the merc revive reserve (late-start variant)

> status: harness proven by a dry run (2026-11-06); the full 16-run A/B has NOT been run. Design: docs/ab-design.md "Variants". Tags: V = seen in the dry run, S = from a contract or earlier log, U = not measured.

## What is compared
Arm A `new` = the SME rule of #276 (`merc.reviveReserveLow` 0, `merc.reviveReserveHigh` 50, `merc.reviveReserveClvl` 13). Arm B `old` = fixed reserves 155 and 310 in the same knobs (an emulation of the 2026-11-02 kit reserve; see docs/ab-design.md for what it does not reproduce). Everything else is identical: binary, seed, prepared clvl 11 hero (500 gold, free Rogue), horizon.

## Files
- `tools/ab/prep/` makes the hero (lab d2d with `--lab-verbs`, ledgered SETUP verbs, save).
- `tools/ab/mercrun.sh TAG ARM SEED HOURS [SLOT]` is one run in one slot (prep, then the bot on a d2d without `--lab-verbs`).
- `tools/ab/mercab.sh HOURS SEED...` deals the runs onto `AB_SLOTS` lanes (default `3 4`), refuses fewer than 8 seeds, then runs `mercan.py` and `paired.py` (XP/h, kills/h; base old, arm new).
- `tools/ab/mercan.py` per-run metrics (tests: `python3 -I tools/ab/test_mercan.py`, `go test ./tools/ab/prep`).

## Dry run (V, 2026-11-06, slot 4, one seed pair run one after the other, 0.5 virtual hour)
Command on the worker, from `~/d2d-lab/abmerc` (binaries built from this branch into `bin/`): `AB_DRY=1 AB_SLOTS=4 AB_TAGPFX=dry D2D_BIN=<lab binary with --lab-verbs> bin/agentrun.sh abmerc --ttl 3000 -- ./mercab.sh 0.5 11`.
- Prep worked: 6 ledger verbs (level, points, gold, stash, merc hire), save, quit; the bot loaded the save (`merc: seen ... alive=true` on the first zone entry, clvl 11, 500 gold used for the belt).
- The merc died about 10 virtual minutes in (`merc: died`), the revive was skipped for gold in both arms (`revive skipped` x2), no revive within 30 minutes (so the first death at this purse does exercise the reserve question).
- Both arms gave identical kills (133) and XP/h (8486): the lockstep fork is deterministic and a purse of about 100 gold never reaches either reserve, as expected. The alive share differed (6.7 against 10.0 percent) only because the log clock puts the death line on a different minute line; U: read the alive share as +-1 minute per event.
- `writes after start: 0` was printed in both logs.
- Wall time: 30 virtual minutes in about 4.5 wall minutes under host load 12 to 21 on 16 cores = about 6.7 times real time (the brief assumes 8).
- NOT run: the prep with the `debug stat 13` experience line (added after the dry run: the first prep left the hero at xp 0 of clvl 11). First live check on the full run's first pair; a failure shows as `prep failed` and `exit prep-failed` in the logs.
- The dry run was stopped by the load rule (load1 per core above 1.1) before a re-check.

## Run plan (U for the totals, V for the 6.7 times figure)
- One run: 4 virtual hours = 240 minutes; at 6.7 to 8 times that is 30 to 36 wall minutes, plus about 1 minute of prep and the slot wait. Wall cap per run in the script: `H*60/8*1.6+10` = 58 minutes.
- 16 runs (8 seeds x 2 arms) = about 8 to 10 slot-hours.
- Two lanes (slots 3 and 4, both arms of a seed side by side): about 4 to 5 hours wall. Four lanes (`AB_SLOTS="3 4 5 6"`, slots 4 and up need the host admission check): about 2 to 2.5 hours.
- Earliest start: when the g2g gate runs release slots 0 to 3 (the brief says about 01:30 EST) and the host is under load1 1.1 per core. Command, on the worker: `AB_SLOTS="0 1 2 3" bin/agentrun.sh abmerc --ttl 21600 -- ./mercab.sh 4 11 13 17 19 21 23 25 27` (ship the binaries first, see the build line in docs/ab-design.md notes below). With the four gate slots free: about 2.5 hours wall.
- Build and ship (worker, in a worktree of this branch): `go build -o ~/d2d-lab/abmerc/bin/d2bot.abmerc ./cmd/d2bot && go build -o ~/d2d-lab/abmerc/bin/prep.abmerc ./tools/ab/prep && cp tools/d2dslot.sh tools/agentrun.sh ~/d2d-lab/abmerc/bin/ && cp tools/ab/*.sh tools/ab/*.py ~/d2d-lab/abmerc/`.
- Reading the result: `result-<pid>.txt` in `~/d2d-lab/abmerc` plus the two paired.py verdicts at the end of the driver output. With the pilot SD unknown, 8 seeds only show an effect of about 15 percent or more; if the interval includes 0, report `no effect shown` with the seeds the SD requires (docs/ab-design.md sample-size rule).
