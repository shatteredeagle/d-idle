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

## Cleanup

`tools/agentreap.sh t0drill` and `tools/agentreap.sh --list`; the character stays in town (the drill leaves the UI clean, the P35 gate checks it).

## Unverified until phase B (U)

Ticks per second (25 assumed from the lab's 40 ms pause); whether the repair chore asks the store at every pass on a live character; that the d2d gold, durability and pack cheats leave the character fresh enough for a second profile (each profile re-runs the setup before its own run-start marker); the field (Blood Moor) variant, which needs a wired T0 point.
