# Why Questy's quest chunks never start (2026-11-27)

> status: probe result, read-only on the Questy sandbox (2026-11-28 00:22 UTC). Answers the open question of [sandbox-postdeploy-audit-2026-11-27.md](sandbox-postdeploy-audit-2026-11-27.md) section 2 (row "quest board, Questy"). Tags: V = read in the container or code this run, S = secondary, U = unverified.

## Verdict

BY DESIGN, not a bug in the board. Nothing is holding the chunks; nothing is asking for them. A chunk starts only when something accepts it, and on Questy nothing does: the player's Start is absent, the per-quest auto opt-in is off for every quest, the campaign goal never accepts, and the operator quest loop stopped on 2026-11-26 03:19 UTC. The sandbox has been left in a state where the board waits for a human. A real gap remains (section 5): a board that only waits for a tap is a poor fit for an unattended sandbox, and the viewer text does not say so.

## 1. What starts a chunk (decision tree)

All starts end in `Exec.acceptLocked` (internal/questboard/exec.go:381). V `grep acceptLocked`: exactly three callers.

1. `Exec.Accept` (exec.go:370): the viewer's Start tap and the operator verb `ops quest start` (same `Desk` function, ADR 0030). Human or agent action only.
2. `Exec.TryAnyway` (try.go:53): the owner or sandbox agent `start --try-anyway`; skips a readiness refusal, one shot (ADR 0030 amendments 2026-11-24).
3. `Exec.AutoTick` (auto.go:303), called from `boardStep` (internal/bot/questboard.go:798) every tick. It starts a chunk only if ALL hold:
   a. `quests.board` on (the board is enabled; V tuning override `quests.board: 1`).
   b. The quest has a per-quest, per-difficulty opt-in record in the plan file (`plan.autoStart`, `Plan.AutoStart`, plan.go:151). Set by the viewer's auto cell or `ops quest auto <quest> --on --why`. This is NOT a knob: the `quests.auto*` knobs (`autoDelaySec` 20, `awayMin`, `autoHoldMin` 15, `autoGapMin` 30, `autoDayCap` 6; docs/knobs.md) only shape the limits. Even `quests.autoDayCap` 0 would only disable the whole thing.
   c. The quest data says `autoStart: allowed` (data/questboard/act1.json). `never` quests refuse the opt-in (`SetAutoStart`, plan.go:398).
   d. The chunk is `ready` and its readiness WORD is `safe` (auto.go:231; `risky`, `too hard`, `unknown` never auto-start).
   e. `autoGate` passes (auto.go:177): not paused, no player pause or reset within 15 min, calm (no fight), mode is not `idle`, day cap and 30 min gap not hit.
   f. A 20 s countdown ends without the player's Cancel.
4. Nothing else. `docs/knobs.md` row `quests.board`: "Nothing else starts quests: between chunks the hero acts on the player's mode (Train ...)". V code: `internal/campaign` contains no board accept call (grep), the campaign goal (`D2BOT_GOALS=campaign:normal_act1` on Questy, V env) only gives the hero an objective; `boardStep` then follows `Next()`, and with no accepted run `Next()` returns idle and `boardBetween` lets the mode decide (train: "Quest: fighting (Level up to 17)", the line the audit saw).
5. Resume (`quests.resumeAfterDeaths`, dark launch, off) only lifts a park, never starts a fresh chunk. The "operator quest loop" is `tools/questloop.sh`, an external shell driver that presses 1 on a schedule; it is not part of the bot.

## 2. Which branch Questy is on (evidence)

| Check | Evidence | Result |
|---|---|---|
| Board on | V `ops quest status --why`: `board enabled=true difficulty=normal`; tuning.json `quests.board 1, boardSM 1, stepExec 1` | on |
| Nothing blocks the two chunks | V `blood_raven ready why-not word=safe floor=7 need=no_merc wake="" admission="" last-refused=never`; `search_for_cain/scroll ready word=safe floor=11`; no `code`, no `paused`, clvl 16 vs floors 7 and 11 | ready, safe, unblocked |
| Per-quest auto opt-in | V plan file `state/quests/Questy.quests.json` has keys `accepted, awaySeen, based, runs, latches` and NO `autoStart` array | OFF for every quest (3.b fails) |
| Cain can never auto-start | V data/questboard/act1.json: `search_for_cain autoStart: never` (hazard), `sisters_burial_grounds autoStart: allowed` | scroll chunk needs a human tap by design |
| Mode | V board mode `train` (accepted records say `mode: train`); hero logs `Quest: fighting (Level up to 17)` | not idle (3.e passes if opted in) |
| Anything else that accepts | V audit `state/quests/Questy.audit.jsonl`: last line 2026-11-26 03:19:55Z, operator `cancel ... why "pause: 2 deaths in one chunk"` on cain, den, blood_raven; no start since (about 21 h). Plan `runs`: the last Cain scroll run is `stopped`. | the questloop was stopped on purpose after two deaths and never restarted |
| Log | V `ops log --grep 'auto-start|quest auto|started by itself'` over 3000 lines: 0 lines | AutoTick never even counted down |
| `need=no_merc` | V readiness.go:16 it is a `reason` beside the word, not a refusal (the earlier audit said the same) | irrelevant to the start |

So: branch 3.b (no opt-in), plus branch 1 (no human, loop stopped). Blood Raven, tools_of_the_trade and forgotten_tower would auto-start the moment they are opted in (all `allowed`, all `safe`); the gates 3.d and 3.e hold today.

Side note, V: Questy was in a "waits_for" queued record for `search_for_cain/scroll` (quests.json.bak) that the 03:19 cancel turned into `stopped`; that stop is the player's decision, not a park, which is why no `paused` code shows.

## 3. By design or bug

Design, with one weak spot. By design: ADR 0030 decision text and docs/questboard.md S8 ("a quest the player marked `auto` starts by itself"; opt-in per quest, default off; the board's default is "Start is the player's tap"). The unattended sandbox is the one place that wants the opposite default, and it only gets it by someone running `ops quest auto` or `questloop.sh`. The audit's guess (auto-start off) was right; the part it could not name is that this is a per-quest plan record, not a `quests.auto*` knob, so a tuning.json read shows nothing.

Weak spots (not bugs): (1) `ops quest status` prints no auto column, so "opted in or not" is invisible without reading the plan file; (2) with the questloop stopped and no opt-in, nothing tells the operator that the board is waiting for a human; the `why-not` lines read as if the board were choosing not to start; (3) `ops status --why` shows no "waiting for Start" line.

## 4. What a human can do now (sandbox, no code change)

Smallest change, uses the existing operator verb, no new knob, no restart (ADR 0030 verb 4, rate limit 6 per quest per hour; V usage line of `d2bot ops quest`):

```
docker exec <container> /data/didle/current/bin/d2bot ops quest auto sisters_burial_grounds --on --why "sandbox: let Blood Raven start by itself"
```

Expected: the offer counts down 20 s, then `quest board: sisters_burial_grounds/blood_raven started by itself` (auto.go:303). Repeat for `forgotten_tower` and `tools_of_the_trade` when wanted; `search_for_cain` refuses (`never`); it needs `start` (or `tools/questloop.sh <container> search_for_cain`) with the stop condition of the 2-deaths pause in mind (the same hazard class that parked it on 2026-11-26; the Cain scroll run is the one that killed the hero twice, U on the current cause). Limits: 30 min gap, 6 per day, one chunk per start. The `tools/flip/flip.py` route is NOT needed: auto-start is no tuning override; `--window 2h` does not apply. (If the owner wants the day cap or gap changed, those ARE knobs: `quests.autoGapMin`, `quests.autoDayCap` via flip.py.)

An alternative is `tools/questloop.sh <container> --all` (the existing overnight driver, it also covers Cain). Either way, record the action in the sandbox handover.

## 5. Fix proposal (for the owner; nothing built here)

Card `questauto-visible` (small, no new knob):
- owns: `internal/watch/questsapi.go` or the `ops quest status` renderer in `cmd/d2bot/ops*.go`, `internal/questboard/desk.go` (the `why` text), `docs/operator.md`.
- AC1: `ops quest status` and `--why` print for each quest `auto=on|off|never` and, when no chunk runs and no auto is on, one line `waiting for Start (no auto-start opt-in)`.
- AC2: when AutoTick holds an opted-in safe offer back, `ops status --why` prints `ax.why` (it exists in `AutoSnapshot.Why`; V auto.go).
- AC3: offline test with `mem.Fake`/plan fixture: no opt-in prints the waiting line; opt-in plus `safe` prints `counting down`.

Optional, needs owner word (a new default for the sandbox, not for players): a sandbox-only auto policy so an unattended hero does not idle on the board, for example the container deploy script runs `ops quest auto <q> --on` for every `allowed` quest once at first start (a tool step, no new knob), or a default-0 knob `quests.autoDefaultOn` [dark-launch] opting every `allowed` quest in. NEEDS OWNER WORD: user approval for any new knob; the `ops` script variant needs none but changes what the sandbox does by itself, so it is the owner's call too (the user's rule: measured runs never use writes, and auto-start is a normal path, so it does not breach the crutch rule).

Not found: any bug in `AutoTick`, `autoCandidate` or the campaign. U: whether Questy's admission floors (`admission=""`, shadow mode) would refuse once opted in; they were empty for every chunk, so none is expected.
