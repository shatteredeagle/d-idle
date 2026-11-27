# Hand in to Akara (Search for Cain): why a click can be a silent no-op (2026-11-27)

> status: read-only code diagnosis, nothing run, sandbox untouched. Tags: V = read in the code at main 388dcc3da (file:line named), S = from a probe doc, U = unverified inference. UI click feedback is out of scope (another card).

## Path of the click (V)

1. `internal/watch/web/panels/quest-board/panel.js` `act()` sets `local.handing` and POSTs `/api/quests/search_for_cain/handin` `{chunk}`. A refusal comes back as `ok:false` with `reason` and is toasted (`Reply.OK` has no omitempty, so `ok:false` is sent).
2. `internal/watch/questsapi.go` PostHandler: `policyGate` (token, origin, content-type), desk present, then `Desk.HandIn(quest, chunk)`. Every POST is audited by `logQuestAction`.
3. `internal/questboard/desk.go` `Desk.HandIn`: refuses with a reason unless the chunk is `Awaiting` (409 `not_awaiting`), bot not paused (409 `paused`), board on (409 `off`), hero loaded (503 `no_hero`). If a hand-in for that chunk key is already in `d.hand` it answers 202 "already running" (no new work). Otherwise it records `d.hand[k]`, sets `HandInView{state:"running"}` and starts `runHandIn` with a 5 minute timeout (`handInTimeout`). The reply is 202: nothing has happened in the game yet.
4. `Exec.HandIn` (`exec.go:851`): up to `MaxHandIn` (3) attempts of `act.HandIn`, then read-back: Done T1/T2 = `done`, Done T3 only = `confirming`, else park `handin_failed`.
5. `boardActions.HandIn` (`internal/bot/questboard.go:619`) pushes a request into `rt.reqs`, a channel of capacity 1, and waits for `req.done` or ctx. The dialogue itself only runs when the runner loop reaches `boardServeHandIn` (questboard.go:961), called at the top of `boardStep` (line 701), and only when the snapshot says the hero is in town. It then runs the chunk's turn-in step (`read`: `StepTalk Akara pick "bark"`; `report`: pick "cain report") via `Dialog.Talk`.

So the 202 means "queued for the loop", never "the loop took it". There is no log line when the tap is received, queued or picked up.

## Ranked causes of a silent click

1. U/V-by-code, most likely: the tap is queued but the loop never calls `boardStep`, or calls it only after other steps. Before `boardStep` (goal.go) the tick may be taken by `repairTripStep`, `choresStep`, `lowLifeGate`, `gearStep`, `savePointStep`, `claimTripStep`, the Farm gold order, `recoveryStep`, `arbHoldsQuest` (goal.go:60-110). A goal kind of `idle` (goal.go:51) or `runs` (goal.go:91) never reaches `boardStep`, and for other goals `boardPreemptsWhy` returns false while the recovery (purse and kit) is active (questboard.go:1144). Symptom: `handIn.state "running"` for up to 5 minutes, no "talk" line, hero idle or doing chores. The reason text exists only for start (`Queued`), not for a waiting hand-in.
2. V by code: a stale request after a timeout. When the 5 minute ctx ends, `boardActions.HandIn` returns but the request stays in `rt.reqs` (nobody drains it). Every later tap then gets `"a hand-in is already waiting for the bot"` three times in a row at once (non-blocking send, questboard.go:625), so `Exec.HandIn` parks the chunk: `hand-in failed 3 times: a hand-in is already waiting for the bot`; while the old request, when the hero finally stands in town, runs a talk nobody is waiting for. Also, a second tap during the first 5 minutes answers 202 "already running" and does nothing new.
3. S: hero not in town. L3 run 7 (`docs/probes/cainl3/testdata/run7-handin-not-in-town.txt`) parked the chunk in one second; fixed in `boardServeHandIn` (waits for town), so now it is case 1 (silent wait) when the hero is out hunting. Applies if the hero is in the field or in a town-portal trip.
4. V by code: the talk runs and the board reads `confirming`, not done. `read` is T3 (bark gone, key present); the pay and the "done" word wait for the save (`QuestsLive` false, docs/questboard.md line 79). The tile shows "Checking that the game agrees"; to the player the click looks like nothing. For `report`, L3 run 10 showed Akara's shop greeting already finished the quest before the chunk was offered (S, cainl3 open item 2), so `report` may never be Awaiting.
5. V by code: talk attempts fail 3 times and the chunk parks: `quest board: hand-in failed search_for_cain/read ... err=...` x3 then `quest board: parked search_for_cain/read ... why=hand-in failed 3 times: ...`. Errors from `Talk`: `no NPC class known for Akara` (not in Act 1 town), `open Akara: ...` (NPC not found or not reached), `dialog option "bark" ... not implemented`, or `the reward did not register after the dialogue`.
6. V: refusals the player can read in the toast: not awaiting (the chunk is `running`/`accepted`, e.g. the board already treated the scroll as done), bot paused (409 `paused`), board knob off, `measured_run` is only for `try`. Not silent unless the toast is missed.

## Confirm on the sandbox (read-only calls)

- `GET /api/quests`: find `search_for_cain` chunk `read` (or `report`) and its `state`, `reason.code` (`handin_wanted` = waiting for the tap; `handin_in_town` = in flight), top-level `awaiting`, `handIn{state,reason,attempt,at}`, `paused`, `blocked`.
- Right after the tap: `handIn.state "running"` that does not move for minutes = case 1 or 2.
- `GET /api/explain`: `decisions` (is the hero doing gear, chores, recovery, a farm step?), `refusals`, `journey`. Look at the current goal kind in the goals view.
- Bot log lines to grep: `quest board: awaiting hand-in of search_for_cain/read`, `quest board: search_for_cain/read is ready to hand in to Akara` (approach talk done), `npc: talk` with `npc=Akara pick=bark`, `quest board: hand-in failed search_for_cain/read` (attempt, err), `quest board: handed in search_for_cain/read` (outcome, tier), `quest board: parked search_for_cain/read`, `quest board: start gate: waiting for the recovery (purse and kit) to finish`, and the audit line from `logQuestAction` for the POST (status 202, `ok`, `already`). Expected healthy sequence after a tap: audit 202, `npc: talk`, `quest board: handed in ... outcome=confirming`. No `npc: talk` after the audit line means the loop never served the request.
- Check `rt.reqs` fullness indirectly: a second tap after a timeout failing with "a hand-in is already waiting for the bot" confirms case 2.

## Proposed fix card (not created; coordinator decides)

Card `akarahand`: "Hand-in tap is observable and cannot go stale". owns: `internal/bot/questboard.go`, `internal/questboard/desk.go`, `internal/questboard/exec.go`, their tests, `docs/questboard.md`.
ACs:
1. The tap logs one line when queued and one when the loop picks it up (`quest board: hand-in queued/served search_for_cain/read`), both with the chunk key.
2. A request whose ctx ended is dropped: `boardServeHandIn` skips a request whose ctx is cancelled (carry ctx in `boardHandInReq`), and `boardActions.HandIn` removes its own request on timeout; test: tap, timeout, second tap works (must fail on main).
3. `HandInView` while running carries the holder in words when the loop is not serving (not in town, recovery, chores, goal kind), via the existing reason table; test per holder.
4. A goal kind that cannot reach `boardStep` (idle, runs) serves a pending hand-in in town, or the desk refuses with a reason; test with a Sorceress and one other class.
5. `confirming` after a T3 read-back says in the tile reason that the dialogue ran (words only; wording left to the UI card).
6. Live drill on a lab slot or Proving Ground `cain.read` with `writes after start: 0`; no verification claimed without it.
