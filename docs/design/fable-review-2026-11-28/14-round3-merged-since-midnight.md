# Review 14: round 3, code merged since 00:30 EST (2026-11-28)

> status: current. Reviewer run rv5, read-only. Base `27fe88d59` (first-parent tip before 00:30 EST) to `origin/main` `20d1fc70f`;
> ops repo `master` at `153b3e4`. Merges covered: `git log --since='2026-11-28 00:30' --merges --oneline origin/main`.

Tags: **V** = read in the code at the line cited (nothing was run: no test, no game). **S** = inferred from the code,
consequence not demonstrated. **U** = not checked. No finding here was reproduced by a failing test; every "test sketch"
is a test that should fail today and is not written. Areas I did not read: `internal/shadowlog`, `cmd/d2shadow`,
`tools/roadmap`, `internal/watch/serve.go` (ops2 cookie), `internal/diffready/consent.go` (sweep1), sec2, evid1, grad1,
`cmd/d2scenario`, the cls1/cls2 JSON rows themselves, ops4 queue guard and card union. Those are U, not clean.

## Ranked findings

### H1. Mercenary equip: a lost reply to "take the copy back" destroys the vault item (vault2)
- Where: `internal/vaultkeep/equipmerc.go:186`, `internal/vaultkeep/mercjournal.go:211-234`.
- V: when the mercenary refuses the piece, the copy is removed with `ItemVault(cr.ID)`. If that call is unanswered the
  function returns and the intent stays at step `give`. `settleMerc` then judges only by "is the copy still on the hero":
  not there and step `give` means "the mercenary wears it", so `mercRetire` discards the vault record.
- Scenario: refusal (level, Str), the removal runs in the game but its reply is lost. Next pass: copy gone, record
  retired. The item is in neither the vault nor the game. The same hole is inside `settleMerc` itself (line 220: an
  unanswered removal of a copy "still in the pack" leaves the intent open; the next pass reads "gone = given").
- Test sketch: fake API where `MercEquipDirect` answers a refusal and `ItemVault` removes the unit and returns a timeout;
  run `EquipMerc`, then `repairMerc`; assert the record is still `Recorded`. Fails today.
- Fix direction: journal a third step (`takeback`) before the removal; in that step "copy gone" means "back in the vault".

### H2. Mercenary repair files every new wearable item of the HERO into the vault (vault2)
- Where: `internal/vaultkeep/mercjournal.go:232-246`, gate at `:187`.
- V: after "given", every item on the hero that was not there when the intent was written and that the mercenary could
  wear (`MercPreview(code).Can`) is recorded and removed from the game (`mercFile` -> `ItemVault`). The item list is the
  whole hero, not the backpack (the carry code says so: "the item list carries no location (U)"). The repair is skipped
  for the whole of a measured run (`k.measured()`), so the window is as long as the run.
- Scenario: give unanswered just before a measured run; during the run the hero loots or buys armour, a helm or a
  polearm and wears it; after the run the repair removes those pieces, worn ones included (S: that `item vault` takes a
  worn unit is U live). An unattended hero can lose its armour in the field. Together with H1 the false "given" verdict
  (copy sold by a town chore, dropped) also triggers the sweep.
- Test sketch: intent at step `give` with pack snapshot P; fake hero now holds P minus the copy plus one new armour
  (worn) and one new helm (pack); `repairMerc`; assert at most ONE piece is filed and never a unit the hand reports worn.
- Fix direction: file only the unit the fork named as displaced, or at most one new piece of the slot the copy went to;
  never a worn unit; cap the age of an intent that may sweep.

### H3. On-road kit skip admits a hero with no potions and no portal to the boss level of the same road (nf3 x sweep2)
- Where: `internal/bot/earlyroute.go:107`, `internal/admit/admit.go:143`; rows in `docs/plan/data/admission.json`.
- V: the kit rule is skipped for every area of a road once the hero stands in any area of that road. The boss levels
  are on the same row as their approach: sewers `[47,48,49]` (Radament), halls `[56,57,60]`, claw viper `[58,61]`,
  maggot lair `[62,63,64]`, underground passage `[10,5]`. Only Duriel's Lair has its own row and keeps its kit check.
- S: the lab case that motivated the skip was "no portal, so the trip cannot be made". The skip also covers "portal in
  hand, zero potions", where the old refusal produced a restock trip. Whether the trip rules catch that hero before the
  boss is not shown; `holdForReward` (`internal/bot/goal.go:370`) can hold a restock trip for `rewardHoldMax`.
- Test sketch (two classes): `Rows.MayEnter(View{Area: 63, Healing: 0, HasTP: true}, 64, Quest)` must answer kit-only
  refusal; with `HasTP: false` it must answer a verdict that walks out, not Allow. Today both are Allow.
- Fix direction: skip only the portal half when no portal exists; keep the healing floor for a deeper area of the road.

### H4. A hole hold answers "fresh game", not "wait" (live1)
- Where: `internal/bot/zonemodel.go:271` (and `:294`), `internal/bot/plan.go:513-516`, `internal/bot/task.go` (`freshGameToLevel`).
- V: the comment says "nothing is picked (the caller waits in town; fault bans are short)" but the return is
  `0, true, false`; `Plan.Pick` turns `fresh` into "nothing eligible", and `LevelTo` then calls `freshGameToLevel`, which
  starts a new game whenever an area was cleared or ran dry in this game.
- Scenario: `Hold` (best zone has margin under 0.05, a permanent state for an undergeared hero): every slice asks for a
  fresh game; `freshAllowedFor` counts no-gain restarts up to `Route.FreshMaxNoGain` and the hero ends "blocked: needs
  input". For a fault ban the new game costs a game for a ban that lapses in minutes. S: the restart count, not run.
- Test sketch: extend `TestZoneHoleNotWalkedAfterFaultBansOrWithoutMargin`: with `cleared` non-empty and a fake Cycler,
  run `LevelTo` on the "no margin" and "unreachable ban" cases and assert the Cycler was not asked. The existing test
  only asserts that area 3 is not chosen.

### H5. A transient profile failure latches "the hero will not fight" for the rest of the game (fix8/cls)
- Where: `internal/combat/fighter.go:937` (`CompareAndSwap(nil, &err)`), `internal/combat/fight.go:118,159`.
- V: `profileFor` stores the first `SelectProfile` error and nothing clears it except `SetProfileErr(nil)` at the next
  class-aware start. `SelectProfile` fails only when the class has no profile at all, which is also what a torn or early
  snapshot with a bad class id looks like. From then on `Clear` and `Kill` return an error in the field.
- S: that such a snapshot reaches `profileFor` live is not shown. The cost if it does is a hero that stands in a pack.
- Test sketch: `f.profileFor(&d2.Player{Class: 99})`, then `f.profileFor(sorc)`, then `f.Clear(...)` for a Sorceress and a
  Paladin: must fight. Today `ProfileErr()` stays set.
- Fix direction: only `SetProfileErr` latches; `profileFor` reports the error for that call and clears it on a later success.

### M1. Declines and failures share one counter: one real failure after a row of declines jumps to BackoffMax (fix7b x sweep2)
- Where: `internal/arbiter/arbiter.go:218-237`.
- V: `Decline` and `Report(false)` both go through `fail`, which increments `b.n`; only the cap differs. After k
  declines the next real failure doubles k times and lands on `BackoffMax`, the state review 11 N1c set out to remove.
  `NoBackoff` intents are untouched by `Decline` (`backsOff()` returns first), so a declining trip intent is chosen
  again every tick and the chain runs as fallback (`internal/bot/arbiter_control.go:31-48`): no tight loop, V.
- Test sketch: six `Decline` of a claim intent 120 s apart, then `Report(in, now, false)`; assert the refusal lasts no
  longer than `BackoffBase`. Fails (it lasts BackoffMax).

### M2. Lost-reply equip is judged "swap ran" without checking that the old item left the hero (fix1b)
- Where: `internal/vaultkeep/equipdirect.go:385-427`.
- V: slot holds the new bytes -> the copy of the old item read before the call is filed. The old rule also required the
  old unit to be gone from the game. `heldBy` blocks Sell, Carry and Equip of a record under an open order but not Keep.
- Scenario: reply lost, nothing moved, order open; the record is taken out with Keep and put on by hand; the repair
  sees the new bytes in the slot and files a second copy of the old item, which is in the pack: a duplicate.
- Also V: "another item in the slot" and orders written before fix1b (`New == ""`, line 400) stay open with no expiry;
  such an order refuses Sell, Carry and Equip of its record for ever. Review 10 N4 asked for exactly this to end.
- Test sketch: open order with Before=A, New=B; hero holds B in the slot and A in the pack; `repairDirect`; assert no new
  record with A's bytes. Second: order with `New == ""` and a third item in the slot ends after N passes.

### M3. Reserved records stay held while a queued sale cannot run (vault2)
- Where: `internal/vaultkeep/sale.go:689`, `internal/vault/store.go` (`dry`, hold rule).
- V: an unanswered credit leaves the order queued and its records held. `resumeSale` returns before `settleSale` while
  `SaleOn` is false (knob off, fork flag off, measured run), so neither the retry nor the release runs; Keep, Pin,
  Discard, Reveal, Carry and Equip of those records answer "reserved" for as long as that lasts.
- Test sketch: credit returns a timeout; turn the sale knob off; `resumeSale` ten times; assert `st.Held()` is empty or
  the order ended with a note. Today the hold stays.
- Note (V, older code): a credited order whose batch partly fails ends `SaleDone` with the gold kept and the records
  released (`sale.go:670`); with reservations a state refusal can no longer cause it, a disk error still can.

### M4. Parked boss exclusion covers only the boss's own area and only after a margin read (sweep2)
- Where: `internal/bot/questboard.go:210-236`, `:356`.
- V: `bossArea` is rebuilt empty in every `assess` and filled only when the margin hook returned an encounter for a
  chunk-level boss. Before the first assessment, with the hook unset, or for a quest-level boss, nothing is excluded.
- S: the live symptom named Tower Cellar Level 4; the boss stands on Level 5. The test uses area 25 only. Whether Level
  4 can still be picked depends on the candidate list (U).
- Test sketch: park the Countess chunk, ask `guardedAreasNoFarm` for areas 21 to 25; assert every level of the tower is
  excluded; repeat before the first `assess`.

### M5. Reinserted town chore can land after a chore it must precede (nf4)
- Where: `internal/town/route.go:309-326`.
- V: `routeOrderWith` orders only the placed chores; `reinsertChores` puts an unplaced chore after the last chore it
  must follow and does not look at edges where it must come first. With A -> m -> B and the route choosing B before A,
  m lands after B. S: whether today's edge set has such a pair for a chore without a preset (Cain before the rescue).
- Test sketch: property test over random edge sets: for every edge (x, y) with both present, index(x) < index(y).

### M6. Ops pipeline status still reads the removed one-file ledger (led1 x ops)
- Where: ops repo `tools/pipeline_status.py:76`.
- V: it opens `data/darklaunch.json`, which led1 removed; the dark section now prints "not readable". After a clean
  close the status word for git is only printed by the watcher (`tools/flip/flip.py`, `_design_hint`), so an unattended
  watcher leaves git at `dark` while the sandbox runs the knob; the runtime state file is per machine
  (`ledger.default_state`), so another machine sees a dark row and would open a second window. The autodeploy skip reads
  `flip.py status` (windows), not the ledger, and the state file sits in a subdirectory the window store does not list: V, fine.
- Test sketch: `pipeline_status` against a checkout with `data/darklaunch/` only must list the dark rows.

### L1. Third full-grid search ignores stall blocks (nf3b)
`internal/nav/nav.go:420`: the seam retry with hard blocks off returns a path through cells a stall blocked, unmarked, so
the caller cannot tell; a goal that is really cut off now costs three whole-grid searches (memoised for 2 s of
navigator time, `nopath.go`). The memo key holds the COUNT of hard blocks and blacklist entries, not their content.

### L2. `rewardWalkedHome` calls `exec.Next()` from the trip hold (live1)
`internal/bot/goal.go:398`: `Next` derives the board from the plan file on every call; the hold is asked several times
a second while a reward is pending. U whether `Next` has side effects besides the read.

### L3. Heartbeat no longer counts monster life (fix7b)
`internal/bot/observe.go:797`: a single action that fights for the watchdog limit without a kill, a pickup or 20
subtiles of movement is now judged hung. Intended; U whether any boss fight of a weak hero is that long.

### L4. Class split by literal next to class split by data (cls1/cls2)
`internal/bot/plan.go:96` (`MeleeTank`: Paladin, Barbarian) feeds the level windows; `admitView` and `earlyRouteCheck`
use the traits table (`Caster()`), where every non-caster gets the melee life floors. A bow Amazon or a trap Assassin
has class data now and still takes the melee rows. Older leftovers, not introduced tonight; not re-audited.

## Checked and found sound (V by reading)
- `Reserve`, `SellNow`, `SellAll`, direct Equip, Carry, Stow and the merc path all take `actMu`; the sale also `smu`.
  No interleaving between a quote and a reservation exists inside one process. `releaseOrphans` runs under both.
- An older binary replaying a log with `hold`/`release` events skips them as anomalies; nothing is lost on a rollback.
- `vault.Inspect` and the ops vault check write nothing.
- Hand-in: `handed` is set only when the dialogue returned no error; reading the evidence after a failed attempt does
  not mark a chunk done by itself.
- `flailHold` and the cleared-area sweep now pause; the Run loop stamps `lastStep` every iteration, so holds in town do
  not trip the process watchdog.
- Autodeploy: hold by code tree survives docs-only commits; a rollback failure exits 3 with the hold already on disk.

## Unattended deploy to the sandboxes
H1 and H2 need the vault mercenary equip to be on and used (U: its knob state in the sandboxes). H3 and H5 are in the
default path of a Normal hero and are the two I would settle before leaving a sandbox alone over a night; H4 costs games,
not the hero.
