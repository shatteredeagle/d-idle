# T0 drill: phase B commands (NOT run in phase A)

Card TASK-21 (`docs/t0-slice.md`). The live run belongs to a d2-live agent after the Gate 2 g2d runs end (4:05 PM EST, 2026-11-05); until then do not start it. Run only after the Gate 2 live runs are over, with the lab rules (`docs/lab-rules`, one d2d at a time, never the game instance i0, a fresh character). Nothing below was run in phase A; `ladder.enabled` is installed by the drill itself, the knob is not needed.

## Offline check (any box)

```
go test -p 1 -count=1 ./internal/ladder ./internal/drill ./cmd/d2drill ./internal/town
```

## Build and run on the worker host

```
labrun <worker> -- sh -c 'go build -o /tmp/d2drill-t0 ./cmd/d2drill'
```

Then, in a d2d slot (the d2drill pattern of the other drill groups; the slot starts its own headless d2d, no game instance of the worker host):

```
D2D_AGENT=t0drill tools/agentrun.sh t0drill --ttl 3600 -- tools/d2dslot.sh run -n 0 -t 3600 -- /tmp/d2drill-t0 -group T0 -label t0-all -repeat 1
```

Single profile or the scripted file (JSON, see `ladder.ScriptFile`):

```
... -- /tmp/d2drill-t0 -group T0 -only T0.cautious -label t0-cautious -repeat 1
... -- /tmp/d2drill-t0 -group T0 -t0-profiles file -t0-answers docs/probes/t0-answers.json -label t0-file -repeat 1
```

Flags: `-t0-profiles` (default `cautious,bold,inconsistent,holdout:101,holdout:202`), `-t0-minutes` (default 10), `-t0-answers`, `-t0-dir` (default: answers in memory). Via `tools/live.sh t0drill d2drill -group T0 ...` if the ship-and-run wrapper is wanted; it must use a hidden instance (i1..i5), never i0.

## What it asserts (phase A2)

The drill is a town drill. Five points are wired (`t1.repair`, `t1.tp.lost`, `t1.return`, `t1.town.trigger`, `t1.restock`); the town chores the lab drives exercise three of them live (`t1.repair`, `t1.tp.lost`, `t1.restock`); `t1.return` and `t1.town.trigger` sit in the bot runner's trip logic, which the town lab does not drive, and are proven offline only (`internal/bot/thoughts_test.go`). Per answer profile (the default `-t0-profiles` list: the 3 known profiles cautious, bold, inconsistent plus 2 seeded holdouts 101 and 202, so 5 runs; the scripted `file` profile on request) on a fresh clvl 1 hero:
- the setup (empty pack, gold 1000, worn gear to durability 1) is written before `d2cheat runstart`; the bot plays through its own code with scripted answers after it;
- every pass returned, no ladder fault, no chore error, answer log and open thoughts bounded;
- with nothing learned the bot's actions equal the legacy ones (parity by construction, `AskWithDefault`); what a learned answer did shows in `rulesLearned` and the log lines `ladder: thought/answer/learned`;
- `writes after start: 0` (non-zero is CHEATED and never counts).

Exact command (all 5 runs; needs the build line above first):

```
D2D_AGENT=t0drill tools/agentrun.sh t0drill --ttl 3600 -- tools/d2dslot.sh run -n 0 -t 3600 -- /tmp/d2drill-t0 -group T0 -label t0-all -repeat 1
```

## What to read

Result files `docs/probes/drills/<date>-<label>.{txt,json}`. Each profile line carries the record: `ruleFileHash`, `rulesLearned`, `answersGiven`, `thoughtsFaded`, `thoughtsOpened`, `faults` and `writes after start: 0`. PASS requires: every pass returned, no ladder fault, no chore error, bounded log, writes after start 0. A non-zero count is CHEATED and never counts.

## Fight-time points (A3) and what the live run needs

Four more points are wired (`t0.retreat.drink`, `t0.unique.fight`, `t0.area.toohard`, `t2.mercenary`; `docs/t0-slice.md`). Offline the drill covers them through the optional `drill.T0FightLab` hook: the fake lab in `internal/drill/t0_test.go` implements it, and `go test -p 1 -count=1 -run T0Drill ./internal/drill` runs 5 profiles x 2 builds x 10 virtual minutes with 80 fight passes per run (`fightPasses` in the record). The real call sites are covered by `internal/combat/ladder_t0_test.go` and `internal/bot/ladder_t0_test.go` (legacy answer untaught, learned answer replaces it). The live lab does not implement the hook, so the live run as written still exercises the town points only. A live run of the fight points needs, in `cmd/d2drill/t0_lab.go`:
1. `FightPoints` on the lab: for `t0.retreat.drink` and `t0.unique.fight` a Blood Moor pass with the bot's `combat.Fighter` against a real pack (the drill's field lab has the fighter; the lab must install the ladder before the fight and keep `InFight` true), for `t0.area.toohard` one `fillRoute` read of a clvl 14 character, for `t2.mercenary` the Lut Gholein hire chore (needs an act 2 character, so it stays offline-only on a clvl 1 act 1 hero).
2. A fresh clvl 1 hero cannot reach the elite and low-life states by itself: low life is a SETUP write before `runstart` (d2cheat SetLife) or a real fight; nothing may write after the start.
3. `ladder.enabled` stays off in `cmd/d2bot` unless `-ladder` is given; `cmd/d2drill` installs the ladder itself (the integrated path is below).
Until then the live acceptance (card items 2 and 3) is met for the town points only.

## Phase B, integrated path (the real bot, d2bot -ladder -watch)

Not run. After the Gate 2 live runs end, a d2-live agent can drive the REAL bot with the ladder on, hidden instance only (i1..i5, never i0), a fresh character, under `tools/live.sh` with the bot's own instance id (build `cmd/d2bot`):
```
tools/live.sh t0drill d2bot --attach --ladder --ladder-profile holdout:101 --watch --goals 'farm:Blood Moor' --run-for 10m
```
(instance iN of the agent through `D2BOT_INSTANCE`; viewer 127.0.0.1:8891 and answer 127.0.0.1:8892 are loopback on the worker host; view them through an ssh tunnel, or give `--token` and LAN addresses as in `docs/t0-slice.md`.) Declare the setup writes before `d2cheat --reason R runstart <run>` and set `D2BOT_RUN=<run>`. Read: the log lines `ladder on`, `ladder: thought/answer/learned`, `GET /api/uptime` (deaths, recovery), `GET /api/orders` after `POST /api/orders` `{"kind":"go","area":"Blood Moor","minutes":5}` on the answer port, and `d2cheat ledger --since-run <run>` = 0 writes. Pass: every ladder call returned (no `ladder fault` line), no rule learned that the script did not teach, the hero kept fighting while a thought was open (kills continue), the order ran for its minutes and finished, `writes after start: 0`. U until run: that thoughts open during real fights, that `farm_area` with `UntilMins` ends on time, the Hub clock against a 10 minute run.

## Cleanup

`tools/agentreap.sh t0drill` and `tools/agentreap.sh --list`; the character stays in town (the drill leaves the UI clean, the P35 gate checks it).

## Unverified until phase B (U)

Ticks per second (25 assumed from the lab's 40 ms pause); whether the repair chore asks the store at every pass on a live character; that the d2d gold, durability and pack cheats leave the character fresh enough for a second profile (each profile re-runs the setup before its own run-start marker); the field (Blood Moor) variant, which needs a wired T0 point.
