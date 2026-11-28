# Fable review 2026-11-28, round 2 part A: the fixes for the item flows (fix1) and the quest board (fix2)

> status: review, read-only (no code changed). Base: `origin/main` at 7ec65e945. Scope: merge 95520db57 (fix1, TASK-1095:
> `internal/vaultkeep`, `internal/d2dctl/direct_equip.go`, `internal/watch/vaultapi.go`) and merge b0844c390 (fix2, TASK-1099:
> `internal/questboard`, `internal/bot` questboard.go, goal.go, diffready.go, invariants.go), read against the original
> reviews `01-item-flows.md` and `02-quest-board.md`.

Tags: **V** = read in the repo at the cited line, **S** = inferred from the code, **U** = depends on fork or game behaviour
this repo cannot show. Nothing was run: no Go changes were allowed, so every "must-fail test" is a sketch, not a result. No
live game was used.

## 0. Verdict

fix2 closes what it set out to close; what is left there is medium or lower. fix1 closes the common paths (a late error
after the swap, the double POST, the Stow order of steps), but the new "judge by looking" code decides by unit id, and unit
ids do not survive a new game. That makes three of its repairs wrong in exactly the situations they were written for (N1,
N2), and an order that is open blocks only a second Equip, not a Sell (N3). None of this can skew a gate (every flow still
refuses a measured run); it hits the player's real items. The knob behind direct Equip is on by default
(`VaultDirectEquip: 1`, `internal/env/tuning_loot.go:60`), so these paths are live for every hero on a fork build with
the verbs. Recommendation: set it to 0 and hide Carry and Stow until N1 to N3 are fixed (owner decision).

## 1. Was each original finding closed?

| Original | State after the fix | Evidence |
|---|---|---|
| 01 F1 lost reply, direct Equip | partly: late error and parse error closed; unanswered call still wrong in four cases | N2, N4 |
| 01 F1 second scenario (restart, second tap) | closed (V): `put` moves an unfiled order aside, `directOpen` refuses, repair runs first (`equip.go:163-170`, `equipdirect.go:206-211`) | |
| 01 F2 double POST | closed for Equip, Wear set, Carry, Stow, Sell (V: `actMu`, `vaultapi.go:1149-1160`); merc crash journal left (card says so) | N5 |
| 01 F3 Carry/Stow journal | closed for one charm; wrong with two or more carried charms after a new game | N1 |
| 01 F4 Stow by bytes | same: the by-bytes search is defeated by the stale ids of the other entries | N1 |
| 01 F5 sale reservation | partly: Carry, Stow, direct Equip serialised; Keep, Pin, Discard not (card says so); open orders not respected | N3 |
| 01 F5 credited order with the sale off | closed (V: `sale.go:584-588`); no test in the fix1 diff | T-gaps |
| 01 F6 presentIn twins | not closed, made visible: the doubtful Keep is failed back into the vault while its item may be on the hero | N8 |
| 01 F9 duplicate Add | closed (V: `equipdirect.go:474-489`, `equip.go:713-718`) | N6 for the tag rule |
| 01 F12 repair in a measured run | closed (V: `quiet`, `equipdirect.go:379`, `:495`) | |
| 02 F1 counters, legacy honours parks | closed (V: `exec.go:786-796`, `:218-222`); side effects | B1, B2, B3 |
| 02 F2 rung 4 | closed (V: `invariants.go:196-210`, `:606-610`; guard refusals all carry "not ready", so `fail` does not mark them a fault, `gate.go:309-345`, `invariants.go:530`) | T-gaps |
| 02 F3 resume for every goal | closed (V: `goal.go:52`, `questboard.go:781-795`); it only lifts when calm and above the life floor (`questauto.go:66-68`), so no mid-fight lift | B2 |
| 02 F4 difficulty go | closed for "eaten before the hop"; taken before the hop is verified | B4 |
| 02 F5 SetSM live | closed (V: `questboard.go:165`, `:806`) | |
| 02 F6 release retry | closed (V: `exec_sm.go:302`); Retry zeroes the stored counters (`sm.go:437`), so the retry gets its full tries | |
| 02 F7 board off keeps parks | closed (V: `exec.go:157-159`, `:625`) | B3 |

## 2. New findings, fix1 (item flows)

### N1 (high, charm stranded; loss with a twin): `packUnit` skips units by the STALE ids of the other carried charms
- V: `packUnit` builds `taken` from `Unit` of every other settled entry (`carry.go:125-130`) and skips any hero item with
  such an id before it compares bytes (`:134`). `Unit` of a settled entry is written at Carry time and never refreshed
  (only the entry being stowed or repaired is rewritten, `:186`, `:196`, `:367`). The struct comment says itself that unit
  ids are not stable across a game (`:42-44`).
- Scenario: charms A, B, C carried in game 1 as units 41, 42, 43. The next game numbers them 42, 43, 44 (one more item
  loaded before them; S: how the fork numbers, the review-1 F4 note is the same U). Stow A: `taken` = {42, 43}; A's real
  unit 42 is skipped; `packUnit` returns 0; Stow answers "the charm is not in the game any more" and `forgetCarried`
  (`:356-359`). A stays in the backpack, is gone from the carried list and can no longer be stowed from the page. This is
  outcome 1 of the original F4, now for every hero with two or more carried charms.
- Worse with equal bytes: `have` is `ItemFull(-1)`, the whole hero, not the backpack (`:172`, `:350`). In `repairCarry` a
  stow-intent whose charm left the game is "still carried" when any other item with the same bytes exists (stash twin):
  the new record is discarded (`:190-197`): the charm is lost. A carry-intent whose create never ran is "completed" on a
  twin: the record is discarded (`:179-187`): the charm is lost.
- Fix: never use stored unit ids of other entries. Match all entries to units by bytes in one pass (multiset), restrict to
  the backpack location, and refresh `Unit` of every entry when the game changed.
- Must-fail test: copy of `TestFix1StowFindsCharmByBytes` with two carried charms of different bytes; renumber so that
  A's new unit equals B's stored unit; `Stow(A)` must answer OK and leave B carried. Second: stow-intent, the fake removed
  the charm, an identical charm lies in the stash; `repairCarry`; want the tagged record still live.

### N2 (high, loss plus duplicate): an open direct order is judged by unit id in a later game
- V: `resolveDirect` decides with `Hand.WornUnit(slot) == o.BeforeUnit` (nothing moved, order failed, `equipdirect.go:353-357`)
  and "unit `BeforeUnit` is not in the game" (the swap ran: the record is discarded and the copy `Before` is filed,
  `:358-365`). It is called right after an unanswered call and again by `repairDirectLocked` at every later `Run` and before
  every later order (`:207`, `:548`, `:679`). The order carries no game identity.
- An order stays open when the look itself fails (`ItemFull` error, `:358-361`), which is likely in the case the code is
  for: the control socket just timed out. The next look is in town, often in another game or after the failsafe restarted
  the game process.
- Scenario A (ids shifted): nothing was swapped; in the new game the old ring R has another id; the slot differs from
  `BeforeUnit` and `BeforeUnit` is free: verdict "swapped". Record N is discarded although N was never worn (N lost), and
  R is filed as a record while R is still worn (R doubled, sellable).
- Scenario B (ids equal): the swap did run; after the reload the worn N has R's old number: verdict "nothing changed", the
  order is failed, N's record stays (N doubled) and the `Before` copy sits in a failed order that the next order
  overwrites (`unfiled` needs `active`, `equip.go:158-161`): R lost. This is the original F1.
- Same game, S: the client timed out but the engine has the command queued; the look says "nothing changed", then the
  engine runs it. Only the order id on the wire closes that (the card lists it as left); until then an unanswered order
  should stay open for a settle time, not be failed on the first look.
- Fix: decide by bytes, never by id: read the slot's item and compare with `Before` (nothing moved) and with the order's
  create hex (worn). Store the hex of the new item in the order. Anything else stays open.
- Must-fail test: `TestFix1DroppedReplyResolvedByLooking` variant: `DropReply`, `ItemFull` fails once (order stays open),
  the fake renumbers all units, `Run`. Want: one N (worn), no N record, exactly one R record. Variant without the swap:
  want N's record live and no R record.

### N3 (high, gold for a worn item): an open order or intent blocks Equip only
- V: `saleSkip` checks pin, state and stage (`sale.go:262-272`); it does not ask `managed()` (the Sell-all plan does,
  `sellall.go:44-70`), the carried journal or `directOpen`. A record under an open direct order or a carry-intent is still
  `recorded` (it is retired only in `finishDirect` or after the create, `equipdirect.go:374-378`, `carry.go:296`).
  `Carry` and `EquipMerc` do not check `directOpen` either; `Keep.Enqueue` does not check the carried journal.
- Scenario: Equip N, the call is unanswered and stays open ("order_in_flight" for a second Equip). The player taps Sell on
  N: the stash is credited. The repair later finds the swap ran; `finishDirect` skips the discard (not `recorded`). N is
  worn and was paid for. Same with a carry-intent whose `Discard` failed (`carry.go:296-301`; the comment there promises
  "no Sell or Equip of it is offered meanwhile", nothing enforces it).
- Fix: one `held(id)` rule (active direct order, carry or stow intent, sale line, gold order) used by `saleSkip`, `Carry`,
  `EquipMerc`, `precheck` and the Keep enqueue.
- Must-fail test: `TestFix1UnknownOutcomeBlocksSecondTap` plus `SellNow([id])`: want a skip with code busy and
  `PayFake.Calls` empty.

### N4 (medium, the original F1 in two remaining shapes)
- V: `wornBefore` returns (0, "") both for an empty slot and for a worn item it could not read (`equipdirect.go:320-330`);
  the order is placed anyway (`:225-228`). With `BeforeUnit == 0` an unanswered call ends "interrupted" and the order is
  failed (`:348-352`). Slot empty and swap ran: N worn, N's record live (duplicate) although `WornUnit(slot) != 0` proves
  the swap. Slot occupied but unreadable and swap ran: R lost, N doubled. Fix: refuse the order when the slot is occupied
  and its bytes cannot be read; for an empty slot judge by the slot being filled with N's bytes.
- V: `wornBefore` reads the target slot only. A two-handed weapon also pushes the shield off (`DirectFake.Pushes`,
  `direct_equip.go:336`); after an unanswered call only `Before` is filed (`:363`): the shield is lost. Fix: read every
  slot the piece can empty (the preview knows the hands rule), or refuse a two-hander until the order id is on the wire.
- V: an answer that did not parse is kept in `o.Reply` "never ended or overwritten" (`:288-289`, `:675`), but in the same
  call `resolveDirect` runs with no look at `Reply` (`:250`) and can `failOrder` it; a failed order is not `unfiled` and
  the next order replaces it. And an order with `Reply` that does survive stays active for ever: the record answers
  `order_in_flight` to every Equip and `EquipCancel` refuses a direct order (`equip.go:437-441`). There is no way out but
  editing the file. Fix: `resolveDirect` returns early when `Reply != ""`; add an owner action "I checked, close it".
- V: `putEq`, `journalReply` and `markApplied` go through `saveBookLocked`, which ignores every write error and does not
  fsync (`equip.go:212-221`). The journal the fix relies on can silently not exist (disk full). Carry refuses when its
  journal cannot be written (`carry.go:279-282`); the direct path should do the same before the engine call.
- Must-fail tests: `DropReply` with an empty slot, want N's record retired; `DropReply` with a two-hander and `Pushes`,
  want two filed records; book path in a read-only directory, want the order refused with code journal and no call in
  `DirectFake.Calls`.

### N5 (medium, regression, U on the fork): the new merc key makes a second attempt answer "dup" for the life of the game process
- V: `EquipMerc` creates with `Key: vk-eqm-<record id>` (`equipmerc.go:112`) and answers `dup` when the key was seen
  (`:113-117`). The refusal path takes the created copy away and leaves the record in the vault (`:137-149`); the key stays
  claimed (the fake never releases a key, `internal/d2dctl/vault_fake.go:238`, `:270`; the fork keeps keys per process,
  review 1 section 0, U whether `item vault` releases one).
- Scenario: Equip on mercenary; he refuses (requirement, busy, dead). The player fixes the cause and taps again: "an
  earlier order already took this item out of the vault: check your backpack and the mercenary", until the game process
  restarts. Before the fix the second tap worked. The note is also untrue (the copy was removed).
- The unanswered branch (`:124-136`) with an `ItemFull` error keeps the record live and tells the player to look; the
  record can then be sold while the mercenary wears the item (N3).
- Fix: key by order (`vk-eqm-<order>`), and keep a small journal entry (record id, created unit, order) so a retry knows
  whether a copy exists; the lock already stops the double tap the key was added for.
- Must-fail test: merc fake refuses once, then accepts; second `EquipMerc` must be OK with one item on the mercenary and
  the record retired.

### N6 (low, duplicate after a crash): the "this order filed it" rule looks at live records only
- V: `recordItem` treats a duplicate as filed only when a `recorded` record carries the order tag (`equipdirect.go:474-479`).
  If the bot stops between `Add` and the order going `done`, and the player sells or keeps that record before the next
  repair, the second pass adds R again. Fix: match the tag in any state. `createBack` keys are per game process (`:498`),
  so the same stop plus a game restart makes a second copy; store "placed" in the order before returning.
- V: `finishDirect` on the resolve path calls `Pinner(slot, 0, name)` (`:407-409`, `Unit` is 0 in `:363`): the slot pin
  gets unit 0 (S: what `PinSlot` does with it).

### N8 (medium, F6 not closed): the twin Keep ends as "record in the vault, item maybe on the hero"
- V: `twinDoubt` fails the Keep back into the vault (`keep.go:497-503`). In the F6 scenario (A kept, its item displaced
  and filed as B, Keep B created but unanswered) the one item on the hero is B's: the result is B live in the vault and
  on the hero, with a note. Visible, but still a sellable duplicate, and `back` counts records in any state including the
  record under judgement (`keep.go:611-613`). Fix as review 1 said: reduce the kept count when a displaced item with the
  same bytes is filed (mark record A "returned"), then `presentIn` is exact.
- No test names `twinDoubt` (V: grep over `internal/vaultkeep/*_test.go`).

## 3. New findings, fix2 (quest board)

### B1 (medium, behaviour change): deaths now add up over the whole life of a run
- V: `rs.deaths` only rises (`exec.go:826`); a slice that ran resets failures only (`:804`). Since the fix the count is
  stored and seeded (`exec_sm.go:266-297`). Before, a restart cleared it, and the sandboxes restart at every self-update.
- Scenario: a long chunk, one death on Monday, one on Wednesday with hours of clean play between: parked
  `died_too_often`. With `quests.resumeAfterDeaths` 0 that needs the player. Expect more parks than the baseline the knob
  rows were judged on. S: whether that is wanted. Recommendation: either say so in docs/questboard.md and the dark-launch
  row, or count deaths in a window (drop the count after N clean slices or M minutes).
- Must-fail test (if a window is wanted): StepDied, 50 times StepRan with the clock advanced, StepDied; want DecRun.

### B2 (medium): a stored park on the legacy board is never lifted by the policy, and the page still says it will be
- V: the legacy board now shows stored parks (`exec.go:218-222`). `ResumeTick` returns at once without the machine
  (`resume.go:156`); `ResumeNote` has no such check (`:141-149`). So after a flip back to `quests.boardSM` 0 the page says
  "Resumes by itself in about N min" for a park only Try again lifts: finding 3 of review 2, in the new state the fix made.
- V: the player's Reset on the legacy board does not clear a stored park (`exec.go:625` needs the machine) although the
  comment at `exec_sm.go:384` calls Reset the one explicit way; only Try again does (`exec.go:655-657`).
- S: a chunk cancelled and accepted again on the legacy board is blocked at once: the legacy Cancel removes the record
  (`plan.go:345-351`), not the parked run (not traced whether a later save reconciles it), and `storedPark` finds it again with the new record (`store_sm.go:36-48`).
- Fix: `ResumeNote` answers the "only you" sentence when `!e.o.SM`; legacy Reset and legacy Cancel clear the stored park
  of the chunk.
- Must-fail test: park with SM, `SetSM(false)`; want `ResumeNote` without "by itself"; `Reset()`; want DecRun.

### B3 (low): two places write counters without the seed
- V: `HandIn` creates the run state with `e.state` and parks with it (`exec.go:971-973`); `smPark` then stores
  `Failures, Deaths = 0, 0` over the stored counters (`exec_sm.go:249`). If the hand-in park is lifted the death count
  starts again. A `known` state created there also skips the seed in `Report` (`exec.go:786-790`). Fix: seed in `state()`.
- V: when `smPark` cannot save, `rs.blocked` is set and the deferred `smCount` is skipped (`exec.go:793`): after a restart
  the last count is missing. Fix: store the counters when the park was not saved.

### B4 (low, opinion): the go is taken before the hop is verified
- V: `diffHopStarted` runs before `verifyDifficulty` (`goal.go:1257-1258`). A new game that was made in the old difficulty
  (the game refused the difficulty) eats the go and sets the ban; the owner has to tap again. Review 2 asked for "Take only
  after verify succeeded". No double take and no path that never takes was found (V: the only hop is `ensureDifficulty`;
  a go for a passed difficulty is dropped by `Keep`, `consent.go:101-110`).
- Must-fail test: `hopCycler` that makes the game in the old difficulty; want `Granted("nightmare")` still true.

## 4. Gaps the tests do not cover (V: test lists of `fix1_test.go`, `vaultflight_test.go`, both `fix2_test.go`)

- No test runs two keeper calls at once (review 1 asked for goroutines and `-race`): the lock is tested through
  `VaultAPI.enter` alone, not through a POST (no test sees an HTTP 409) and not in the keeper.
- No test for Wear set with a dropped reply, for `EquipMerc` at all, for the sale reservation or the credited order with
  the sale off (acceptance criteria 2 and 4 of TASK-1095 have no test in the diff), for `twinDoubt`, for `journalReply`
  with an answer that does not parse.
- Carry and Stow are tested with one charm; the renumbering test adds 1000 to the one unit (`fix1_test.go:190`), which
  cannot collide (N1).
- Rung 4: the test feeds `noteStepRefused` from a side goroutine (`failsafe_test.go:331-341`); no result travels through
  `do` and `fail`, so the pairing "coded refusal and no fault from the same result" holds by the wording "not ready"
  only. A guard that words a coded refusal otherwise turns the skip off silently.
- Resume under every goal kind is tested for the lift, not for "no lift while the fight is on" from `runGoal`.

## 5. Checked and found sound (V)

- Lock order `actMu`, then `smu`, then `mu` / `cb.mu`; `Run` takes `actMu` only inside `repairDirect` and `resumeSale`
  with no other lock held; `journalReply` takes `mu` while the caller holds `actMu` only. No cycle found. `enter` releases
  by `defer`. S, not traced to the end: `Save`, `Pinner` and the hand are called under `actMu` from an HTTP goroutine while
  the loop may wait for `actMu` in `Run`; they are a socket call, a desk call and a memory read (`cmd/d2bot/vaultkeep.go:83`,
  `:139`, `vaultequip.go:60`), none takes the loop's step lock as far as read.
- Replay of an applied order is idempotent while the filed record is live (tag rule) and `Removed` shrinks to what is
  left (`equipdirect.go:400-405`).
- Stow writes the record before `item vault` and rolls back on a refusal (`carry.go:373-389`).
- Retry (player, policy, release) zeroes the stored counters, so the seed cannot re-park at the first failure
  (`sm.go:437`).
- `boardResumeTick` runs after the pause, recall and vault steps and before the goal branches, throttled to 10 s, and
  refreshes the world first (`questboard.go:781-795`).
- The go file survives a failed new game and a hero in the field (`diffready_test.go:168`).

## 6. Recommended order

1. N2 and N1 (decide by bytes, not by unit id) and N3 (one held rule), in one card; keep direct Equip, Carry and Stow off
   for real heroes until then.
2. N5 (merc key per order) and N4 (refuse an unreadable slot, two-hander, journal write errors).
3. B2 (page sentence and legacy Reset), B1 as a decision for the owner.
4. The test gaps of section 4 with the fixes above; B3, B4, N6, N8 after.
