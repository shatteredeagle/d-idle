# Operator quest control

> status: current (card opsquest). Code: `internal/ops`, `cmd/d2bot/ops*.go`, `tools/questloop.sh`.

## What it is
A way for the person (or agent) who runs the container to press the quest board verbs without the viewer's passcode: Start, Hand in, Retry, Stop and the auto switch, plus a read of the board. It is a local unix socket in the instance state directory, served by the bot, and a client in the same binary (`d2bot ops quest ...`). It calls the same `questboard.Desk` functions the viewer's POST routes call (`internal/watch/questsapi.go`), so the refusals (busy, locked, not_ready, paused...) are the viewer's.

## Turn it on or off
- Off by default. It exists only when the container has `D2BOT_OPERATOR=1` in its environment. Not in the friends pack defaults.
- Never in a measured run: with `D2BOT_RUN` set no socket is created, and a request that arrives later is refused (`measured_run`).
- Off again: remove the variable and recreate the container (the socket file is removed at start of the next run; it is mode 0600).
- Env only; there is no knob and no flag.

## Verbs (the allowlist is in code, `internal/ops/ops.go`)
`start`, `retry`, `cancel`, `handin`, `auto` (change the board) and `status` (read). Every other verb is refused (`verb`); there is no sell, vault, hero, character, tuning, respec, order, mode or dismiss verb.

```
docker exec <container> <d2bot> ops quest status [--json] [--why]
docker exec <container> <d2bot> ops quest start|retry|cancel|handin <quest> [--chunk ID] --why "reason"
docker exec <container> <d2bot> ops quest start <quest> [--chunk ID] --try-anyway --why "reason"
docker exec <container> <d2bot> ops quest auto <quest> --on|--off --why "reason"
docker exec <container> <d2bot> ops quest wait <quest> [--chunk ID] [--timeout 600]
```
`--why` is required for every verb that changes something (1 to 200 characters, one line). Exit codes: 0 done, 1 refused or not finished, 2 usage or a `wait` timeout. The socket is found by `--sock`, then `$D2BOT_OPERATOR_SOCK`, then `/data/instances/main/state/operator.sock` (the container layout).

`wait` polls `status` every 5 s until the chunk is done (`finished`), parked (`blocked`), back to ready after running (`stopped`), was not running at all (`idle`) or the timeout passes (`timeout`). It prints one line `RESULT outcome=... quest=... chunk=... state=... step=... code=... text="..."` with the last step and the park reason.

Finding the binary in the container: `/data/didle/current/bin/d2bot` when the `current` link exists (a self-updated release), else `ls -d /data/didle/releases/*/bin/d2bot | sort | tail -1`, else `d2bot` on the PATH. `tools/questloop.sh` does this lookup itself (`--bin` or `D2BOT_BIN` overrides).

## Read the decision ring, hold, home, go, park (ADR 0030 amended 2026-11-25, card explain1)
Same socket, same guard rails (off by default, never in a measured run for the order verbs, 0600 unix socket, audited with `remote:"operator"`, `--why` required, 30 per hour for the board). These verbs are class-agnostic; areas and quests are data.

```
d2bot ops status --why                              # read only: the last 200 decisions and the standing refusals
d2bot ops hold --why "reason"                       # cancel the open orders and rest in town (a home order, replace)
d2bot ops home --why "reason"                       # queue a rest in town behind the open orders
d2bot ops go <area> [--minutes N] --why "reason"    # a go order, default 60 minutes; the hero's own guards still decide when it walks
d2bot ops park <quest> <hours 1..24> --why "reason" # stop the quest's chunk and refuse start and retry of it until the wake time
```
- `status --why` and the viewer-side route `GET /api/explain` (same access as `/api/why`) return the same view: `decisions` (oldest first: tick time, hook family, whyline `code` and params, area, ticks held), `refusals` standing now (`what`, `need`, count, last seen; a step refusal ages out after 10 minutes), the arbiter's last shadow choice (empty while `bot.arbiterShadow` is 0; `arbiterDecision` is the same choice as a record: intent, family, reason, what it displaced) and the journey block reason. `invariants` lists every invariant counter with `hard`, `count` and `since` (docs/invariants.md); `tools/flip/flip.py watch` reads it. U: refusals come from the places that log `action failed` and `area gate refused the walk` only; typed step outcomes (review P2) will fill it fully.
- The order verbs go through `Hub.Give`, the call the viewer's order box makes, so the area list, the 8 open orders limit and the guards are the viewer's. A refusal is `order_refused` with the reason.
- `park` is the operator's own wake time, kept in process memory (reset at a bot restart, like the limits); it is shown in `ops quest status` data as `parked`. It is not a board state: the viewer still shows the chunk as stopped.

## Try anyway (owner override, ADR 0030 amended 2026-11-24)
`start <quest> --try-anyway --why "..."` starts ONE chunk that the board refuses for READINESS (`not_ready`: "too hard for now", a level or life floor, the Andariel unattended hold, the Barracks or road floor) for one attempt. It skips only those words: the kit gates (potions, portal, mana, antidotes), the death-spiral guards and every combat reaction (drink, retreat, chicken, town portal) stay. It is one shot: a park, a death, a finish, a stop or a restart ends it and the chunk returns to its previous hold. A locked, done, parked or running part is refused (`not_overridable`). It counts against the limits below (6 per quest per hour), is refused in a measured run (`measured_run`), and writes the audit line `path:"operator/try-anyway"`, `why:"override: try anyway: <reason> (refused: <the skipped refusal>)"`; the bot logs `override: try anyway`. The viewer has the same thing: the `Try anyway (risk)` button on a part that waits for readiness, with the same passcode and a second tap that states the risk. The death of a try is counted as a death only.

## Audit
Every verb the desk ran adds a line to the hero's quest audit ring (`<Hero>.audit.jsonl`, the same file the viewer writes) with `remote:"operator"`, `agent:"ops-cli"`, `path:"operator/<verb>"`, the desk's code and reason, and `why` (the caller's reason; the field is new and empty for viewer lines). A refusal before the desk (bad verb, missing reason, rate limit) is logged, not written to the ring, so a runaway loop cannot push real events out of it.

## Limits (in the bot, per process)
At most 6 mutating verbs per quest per hour, 30 per hour for the whole board, and a 60 s cooldown after a `retry` during which `retry` and `start` of that quest are refused (`cooldown`; `cancel` still works). Excess is `rate_limited` with the reason in words.

A `retry` is also refused with `not_parked` (and books nothing) unless a matching chunk of the quest is `blocked`: retry clears a park and must never touch a chunk that runs. `tools/questloop.sh` only retries a chunk that is blocked with a park code, confirmed on a second status read.

## Threat model
1. Anyone with `docker exec` on the host already owns the container, its state and its save; this adds no power they lack.
2. The socket is not on the network: a unix socket, mode 0600, owned by the bot's user, created under a temporary name and renamed into place so it is never looser.
3. It is not exposed through the viewer, a port or the answer listener; the viewer's passcode and token are neither read nor needed.
4. The scope is the quest board only; the allowlist is tested, anything else is refused.
5. Every action is audited with its reason; the limits stop a runaway loop.
6. Off by default and never in measured runs, so gates and soaks cannot be touched by it.
7. Caveat: the player's own Stop/Start history shows operator lines too; the board does not distinguish them visually (only the audit does).
8. Caveat: the limits live in process memory and reset at a bot restart.
9. Caveat: the quest board is class-agnostic, so nothing here depends on the hero's class or build.
10. Do not put the socket on a shared mount; the instance state directory must stay private to the container.

## Driving quests overnight
`tools/questloop.sh <container> <quest|--all> [--max-tries 3] [--wait 300] [--stall 6] [--backoff 65] [--budget 10800] [--tries-backoff 1200] [--deadline 07:30] [--bin PATH]`.

- Per quest: status, then start (retry when parked; nothing when a chunk already runs, so restarting the loop is safe), then `wait` in `--wait` second windows with one heartbeat line per window (elapsed, try, step, progress).
- Slow progress is not a failure: `status` rows carry a `progress` counter (the server counts each change of state or step it sees for a chunk), and `wait` prints it. A window that ends with a moved step or counter just continues. `--stall` windows without any move, a park, a cancel or a refusal count as a failed try.
- After `--max-tries` tries the loop backs off `--tries-backoff` s (20 min, under the 6 per quest per hour limit) and goes on with the same quest until the per-chunk `--budget` (3 h) or the `--deadline` (HH:MM local time, default 07:30, or `@EPOCH`) passes. A refused `cooldown` or `rate_limited` waits and is not a try; `not_ready`/`locked`/`unavailable` skips the quest with a line.
- The board runs one chunk at a time, so the loop never works on a quest while another has an accepted or running chunk: at every pass, if another quest is active it logs `waiting on <quest> ...; skipping <x>: busy` and waits on that quest first (retry only when it is parked). A `busy` refusal is not a try; the next pass finds the active quest and switches to it. A chunk in `awaiting_handin` gets the `handin` verb (once, then waits), never `start`, and only when nothing else is active. Locked or unavailable quests and `not_ready` refusals are skipped without a try.
- A finished chunk moves on (the next chunk of the quest, then the next quest). `--all` walks the board in order, skips done and locked quests, repeats until nothing is startable, and exits 1 if any quest used up its budget or the deadline passed.
- Hero deaths are not exposed by the board, so they are not a stop condition. The area is not exposed either; progress is the step and state counter.
- Offline test: `bash tools/questloop_test.sh` (a fake `docker` shim and a fake clock).

- `ops quest status --why` (here `--why` is a switch) adds one `why-not` line under every chunk that does not move: readiness word, floor, need code, wake fact, guard words and the last refusal time (`whyNot` in `GET /api/quests`, drawn as `Why not:` on the quest board sheet). The need and wake come from the hero's freshest typed refusal and only for live chunks; for a waiting chunk the need is its first readiness reason. V (unit tests, no live game).
- Auto-start is visible (card questauto-visible): every `ops quest status` row carries `auto=on|off|never` (plan-file opt-in, or the quest data `autoStart: never`); `ops status --why` ends with `waiting for Start: quest/chunk (start_tap|never_auto, auto=..)` for each ready chunk nothing will start (only the Start tap, the `start` verb or an opted-in safe AutoTick start a chunk) and `auto-start held back: <why>` from the auto controller. `GET /api/quests` chunk rows add `auto` and `waitingFor`; the Return screen section "Quests paused" adds `Waiting for you to start: ...` while nothing runs. When chunks start is unchanged. V (unit tests, no live game).
