# Fable review 2026-11-28, part 1: item flows through the Vault and the fork verbs

> status: review, read-only (no code changed). Base: `origin/main` at 415b11f25. Scope: `internal/vaultkeep`, `internal/vault`,
> `internal/d2dctl` item verbs, `internal/watch` vault API and cache, `internal/charms`, the town stash chore skip.

Tags: **V** = read in the repo at the cited line (nothing was run, no live game), **S** = inferred from the code or from game
knowledge, **U** = depends on fork or game behaviour this repo cannot show. Every scenario below is V for the bot-side control
flow and U for the fork side unless it says otherwise.

## 0. Summary

The old town-visit flows (Keep, queue Sell, Reveal) are write-ahead: a log line before the game step, a look at the hero after
a restart. They hold up, with one counting bug (F6).

The newer one-call flows (direct Equip, Wear set, Equip on merc, Carry, Stow) do not have that shape. They share three
structural gaps:

1. **The reply is the only copy.** `item equipdirect`, `item wearset` and `item vault` remove an item from the game and hand
   its bytes back in the reply. Nothing is durable until the bot has parsed the reply and written a file. A lost reply, a
   failed ledger append, a parse error or a crash in that window destroys a player's item.
2. **No reservation.** The record stays `recorded` while the game step runs. There is no lock across HTTP requests
   (`watch.VaultAPI.mu` only guards the desk pointer, `internal/watch/vaultapi.go:304-314`; `stepMu` is the bot loop's and is
   not taken by any vault handler, `internal/bot/runner.go:220`). Two POSTs, or a POST and the loop's `Keeper.Run`, interleave
   between "check state" and "retire record".
3. **No idempotency key that survives.** The direct verbs send no order id on the wire (`internal/d2dctl/direct_equip.go:160`,
   `:195-199`), `EquipMerc` creates with no key at all (`internal/vaultkeep/equipmerc.go:107`), and the fork's key set is per
   process ("a restarted d2d forgets its keys", `internal/d2dctl/vault_verbs.go:429-430`).

Measured and lockstep runs: every player-ordered flow refuses a measured run in the keeper and again in the d2dctl client (V:
`carry.go:108`, `equipdirect.go:33`, `sale.go:146` and `:512`, `direct_equip.go:88`). The one leak is the repair pass (F12).
So none of F1 to F11 can skew a gate; they hit the player's real hero.

## 1. Flows and their crash points

| Flow | Steps (in order) | Durable before the game step? | Repair pass |
|---|---|---|---|
| Keep (`keep.go:626`) | kbegin log, create key=vk-id, read-back, save, kept log | yes (`BeginKeep`) | yes, looks at the hero (`keep.go:472-498`); see F6 |
| Queue Sell (`sell.go:175`) | rstage log + purse, create key, read-back, vendor sell, sold log | yes | yes (`resumeSell`) |
| Instant Sell / Sell all (`sale.go:389`, `sellall.go:179`) | order file, credit by order id, archive + soldnow batch | order file, not fsynced | yes (`resumeSale`), gated; see F5, F8 |
| Gold reveal (`gold.go:350`) | order file, unseal log, pay by order id | yes | yes (`resumeGold`) |
| Equip order, old (`equip.go:553`) | Keep to pack, Wear (clicks), Add record of displaced, `item vault` displaced, save | partly | none for "worn, displaced not filed"; see F7 |
| Direct Equip / Wear set (`equipdirect.go:188`, `:340`) | order file (not applied), engine swap, order file (applied + removed hex), discard record, decode + Add removed, save | **no**: removed bytes are only in the reply | `repairDirect`, only for orders already marked applied; see F1 |
| Equip on merc (`equipmerc.go:71`) | create (no key), merc-equip, discard record, `item vault` displaced, Add | **no**, no order file at all | none (the header says so, `equipmerc.go:19-20`); see F2 |
| Carry (`carry.go:118`) | create key=vk-carry-id, keep mark, carried.json, discard record, save | no | none; see F3 |
| Stow (`carry.go:170`) | `item full` unit, `item vault` unit, decode, Add record, forget, save | **no**: bytes only in the reply | none; see F3, F4 |

## 2. Findings, ranked

### F1 (blocker, item loss + duplication): direct Equip loses the displaced item whenever the reply is not processed

Evidence (V):
- The engine has already taken the worn item off and removed it from the game when it answers; the bytes come back as
  `removed=` (`direct_equip.go:19-27`, `:54-55`).
- `DirectEquip.run` returns an error with a valid result when the second ledger append fails (`direct_equip.go:133-135`), when
  the reply does not parse (`:117-118`), when a wearset answers a different count (`:200-202`), or on any transport error after
  the engine ran (`:119-121`).
- `directOne` treats every error as "nothing happened": `failOrder`, return (`equipdirect.go:216-227`). The removed hex is
  dropped. The vault record stays `recorded` although the item is now worn.
- A crash between the engine's answer and `setOrder(... Applied ...)` (`equipdirect.go:228`) leaves the order not applied;
  `repairDirect` then ends it "interrupted: check the Inventory page" (`equipdirect.go:463-465`).
- The order id never reaches the fork (`direct_equip.go:160`), so the bot cannot ask the fork what the order did.

Scenario: hero wears a unique ring R. Player taps Equip on vault ring N while the control socket times out on the reply (the
loop shares the socket, `equipdirect.go:21-23`). Result: N is worn, N's record is still in the vault (duplicate, sellable for
gold), R exists nowhere. Same for up to twelve pieces on Wear set.

Second scenario (V, no socket fault needed): the bot stops after the applied order is saved and before `finishDirect`. On
restart the record is still `recorded` and the hero is in the field, so `Run` has not called `repairDirect` yet. The player
taps Equip again: `putEq` replaces the order unconditionally (`equipdirect.go:330-336`, called at `:214`) and the saved
`Removed` hex is overwritten. R is gone; the second swap displaces the first copy of N and files it: N is duplicated.

Repair: partial (only applied orders). Measured run: refused.

### F2 (blocker, duplication): Equip on merc and direct Equip can be doubled by two POSTs

Evidence (V): `EquipMerc` reads `rec.State == Recorded` (`equipmerc.go:91`), creates with `CreateOpts{Pack: true}` and no key
(`:107`), and retires the record only after the give (`:129-131`). `directOne` does the same between `precheck`
(`equipdirect.go:202`) and `finishDirect` (`:236-238`). No lock is held across these steps; the page's `S.busy` flag is client
side only (`internal/watch/web/vault.js:244-255`).

Scenario: two POSTs for the same id (double tap on a slow link, two tabs, a retried request). Both pass the state check. Merc:
two copies are created; the first is given, the second is given and displaces the first, which is then filed as "previously
worn". Direct: the second swap displaces the first copy. Either way the item now exists on the body and in the vault, and the
vault copy sells for gold.

Also on the merc path: a transport error on `MercEquipDirect` after the engine applied it runs the "nothing moved" branch
(`equipmerc.go:114-126`): it tries to vault the created unit, which is on the mercenary now, tells the player to "drop it",
leaves the record in the vault (duplicate) and leaves the piece the mercenary took off in the backpack with no keep mark, where
the sell chore may take it (S). The header already admits the create-then-give crash (`equipmerc.go:19-20`).

Repair: none. Measured run: refused.

### F3 (high, loss / duplication / stuck): Carry and Stow have no journal

Evidence (V):
- Carry: create (`carry.go:147`), then mark, file, `Discard` (`:158`). A stop between create and discard leaves the charm in
  the backpack and the record live. `carried.json` is never compared with the store at start (only `Carry`, `Stow` and
  `CarriedList` read it). In the same d2d process a second Carry answers "created already: check your backpack"
  (`carry.go:245-246`) and the record stays live for ever; after a d2d restart the key is forgotten and a second charm is
  created. In both cases Sell of the still-live record credits gold for a charm the player also carries.
- Carry: a refused `Discard` only raises a note and still answers OK (`carry.go:158-162`). `Discard` refuses a record with a
  stage set (`store.go:702-708`), and Carry never checks `rec.Stage`.
- Stow: `item vault` removes the charm (`carry.go:203`), the record is added afterwards (`:208`). A stop in between destroys
  the charm.
- `carrySave` and `saveBookLocked` ignore every error and do not fsync (`carry.go:80-88`, `equip.go:193-201`).

Repair: none. Measured run: refused (`carry.go:108`).

### F4 (high, wrong item moved): Stow trusts a unit id that a game restart reassigns

Evidence (V): `Carried.Unit` is stored in `carried.json` and used as is (`carry.go:193`, `:203`). The inventory desk documents
that "a game restart gives new ids" and re-attaches its own marks (`internal/invdesk/desk.go:109`, `:127-132`); nothing
re-attaches the carried list. Stow checks only that some item has that id (`carry.go:198`), not its code or bytes.

Scenario: carry a charm, restart the game, tap Stow. Either the id is free ("the charm is not in the game any more", the entry
is forgotten, the charm stays in the pack and can no longer be stowed from the page), or the id now belongs to another stash or
pack item, which is pulled into the vault under the charm's name. Not a loss, but the wrong item moves and the carried list is
wrong from then on. U: how the fork numbers units after a load.

### F5 (high, gold for nothing): an instant sale does not reserve its records

Evidence (V): `SellNow` quotes, saves the order, credits (`sale.go:443-445`, `:516`) and only then retires the records
(`:550`). During the credit round trip the records are still `recorded`. `saleSkip` checks state, pin and stage only
(`sale.go:262-271`); it does not consult `managed()` as Sell all does (`sellall.go:44-70`). Keep, Equip, Carry, Pin and Discard
do not take `smu`.

Scenario: Sell and Carry (or Keep, or Pin) of the same id race. The stash is credited, `SellNowBatch` then refuses the record
("N item(s) stayed in the vault after the gold was credited", `sale.go:563-565`) and the player has the gold and the item. The
note is honest but nothing takes the gold back.

Related (V): `resumeSale` is gated by `SaleOn` (`sale.go:578-580`). An order already `credited` needs no game call to finish,
yet it waits while the knob is off, the fork lacks the flag or a measured run is on; all that time the paid-for records are
live and can be kept, equipped or sold through the old queue.

### F6 (high, duplication): `presentIn` miscounts after a kept item has left the hero

Evidence (V): `presentIn` says an in-flight Keep was created when the hero holds more items with those bytes than there are
other `kept` records with the same bytes (`keep.go:585-599`). A `kept` record is never reduced when its item later leaves the
hero. Records with equal bytes are normal here: a kept ring that is later displaced by a direct Equip comes back as a second
record `v-<hash>~1` (`store.go:554-557`), and rings, amulets and charms have no durability to change their bytes (S).

Scenario: keep ring X (record A, kept). Wear it. Vault-equip another ring: X is filed again as record B. Keep B; the create is
unanswered (timeout) or the bot restarts in flight. Next `Run`: the hero holds one X, one other kept record has X's bytes,
1 > 1 is false, so B "was not created" and is queued again (`keep.go:491-497`). In the same d2d process the retry hits the key,
`presentIn` is false again and the keep fails with `ReasonDup` (`keep.go:669-677`): record back in the vault, item in the stash.
After a d2d restart the key is gone and a second X is created.

### F7 (medium, loss or duplication): old Equip order between Wear and filing the displaced item

Evidence (V): after `Hand.Wear` the displaced item is in the backpack with no keep mark until `vaultBack` runs
(`equip.go:615-637`; `Protect` is only called on failure paths, `:672`, `:699`, `:708`). `vaultBack` adds the record first and
removes the item second (`equip.go:688`, `:703`). A stop after Wear leaves an unprotected former worn item for the chores (S:
whether the sell policy takes it); a stop between Add and `item vault` leaves record plus item. On restart `equipOne` starts
again from `Pick` with the unit that is already worn; there is no "already worn" branch (`equip.go:582-626`).

### F8 (medium, double credit): order files are not durable and the credit relies on the fork remembering the id

Evidence (V): `putSale` and `putOrder` write a temp file and rename with no fsync (`sale.go:219-230`); `saveSale` only notes a
failure and the sale goes on in memory (`sale.go:233-237`, `:444-445`). If the order is lost after the credit, the records are
still live and sell again. A credit whose reply was lost is retried with the same id (`sale.go:533-535`), which is right only
while the fork still knows the id; whether credit ids survive a d2d restart is U, and the key set does not
(`vault_verbs.go:429-430`). The order does not record the stash gold before the credit, so the bot cannot tell by itself.

### F9 (medium, silent loss): `ErrDuplicate` is treated as "filed"

Evidence (V): `recordItem` and `vaultBack` accept `vault.ErrDuplicate` as success (`equipdirect.go:310`, `equip.go:689`), and
`vaultBack` then removes the item from the hero (`equip.go:703`). `Add` refuses any item whose bytes equal a live record
(`store.go:549-552`). d2d items carry no unique id ("`uid` is 0 on every d2d item", `vault_verbs.go:25-26`), so two real items
can have equal bytes (S: fixed-stat set and unique jewellery of one item level, plain bases).

Scenario: one Angelic ring is in the vault, an identical one is worn. A vault Equip into that finger displaces it; `Add`
answers duplicate; the displaced ring is gone. The same rule heals the F2 and F3 duplicates by accident, which is why it must
be replaced by an explicit count, not removed blindly.

### F10 (medium, lost event after a write error): the log is not truncated after a short write

Evidence (V): `appendEv` and `commitBatch` add the written byte count and return on error (`store.go:179-189`, `:1117-1130`)
without cutting the fragment. The torn-tail repair only runs at `Open` (`store.go:238-262`), and the crash tests reopen right
after the short write (`vault_test.go:359-395`). If the process keeps running (disk full, then space again), the next good
line is glued to the fragment; replay counts one anomaly and drops that good event. If it was a `soldnow` or `discard`, the
record comes back after a restart; if it was an `add`, a drop is lost. Also (V): an fsync error returns without applying an
event that is in the file (`store.go:190-193`), so memory and disk disagree until the restart; `finishDirect` and `EquipMerc`
ignore the `Discard` error (`equipdirect.go:237`, `equipmerc.go:130`).

### F11 (low, wrong price): the price cache has no difficulty or game in its key

Evidence (V): `value` caches by record hash and identified flag for the life of the process (`sale.go:241-259`). S: the
vendor's sell cap depends on the difficulty; U: whether `item price` applies it. If it does, a price read in one difficulty is
credited in another. Also low (V): a gold reveal unseals before it pays (`gold.go:370`, `:385`); while the payment is
unanswered the record sells at the identified price and a later refusal cannot reseal a sold record (`gold.go:426-430`).

### F12 (low, measured-run hygiene): the direct repair can write the game during a measured run

Evidence (V): `runEquip` calls `repairDirect` before any gate (`equip.go:510`); `finishDirect` calls `Save` and, when decode or
`Add` fails, `createBack`, which is `item create` (`equipdirect.go:258-262`, `:319-327`). An applied order left over from before
the run start would produce a gameplay write after `runstart`. Narrow, but it is the one path in this scope that is not gated.

### Checked and found sound (V)

- `Store.Version` is bumped in `apply`, so every committed event and every replayed one changes it (`store.go:268`). The page
  cache keys on the store pointer, the version and the price inputs (`vaultcache.go:35`, `:58-61`); a sale bumps the version
  through `soldnow`. There is no ETag on `/api/vault`. The carried list, equip orders and sale view are computed per poll, not
  cached. No stale preview after a sell was found.
- `SellNowBatch`: archive before log before memory, duplicates in the list counted once (`store.go:1048-1084`); a re-run of
  the same order skips records already sold by it (`sale.go:542-548`).
- Sell all: the client order id makes a double tap settle one order (`sellall.go:190-195`); a selection sale is serialised by
  `smu` and the second POST finds the records sold.
- Keep and queue Sell log before the create and resolve in-flight records by looking (`keep.go:641-646`, `sell.go:193-196`).
- Charm data: three bases, magic and unique only (`internal/charms/charms.json`; cell sizes are tagged U there). The stash
  chore leaves a keep-marked charm in the pack on both of its paths (`internal/town/stash.go:91`, `:103`, `:307-313`).
- Rate limits: none on the vault POST routes (`policysheet.go:733-770` is origin, token, JSON, small body). Not a finding by
  itself once F2 and F5 are fixed; batch sizes are capped (`saleMax` 150, `maxEquipSet` 12).

## 3. Smallest fixes, ranked

1. **One action lock in the keeper (fixes F2, F5 race, half of F1's second scenario).** A `Keeper.actMu` taken by `Carry`,
   `Stow`, `EquipMerc`, `directOne`, `directSet`, `SellNow`, `SellAll`, `resumeSale` and by `Run` around each errand. The
   control socket already serialises the game side, so this costs no throughput. About 20 lines.
   Test sketch: `DirectFake` and `VaultFake` with a hook that blocks the first engine call on a channel; start two `Carry` (and
   two `EquipMerc`, two `EquipEnqueue`, `SellNow` plus `Carry`) goroutines for one id; release; assert exactly one item in the
   fake, one retired record, the second result `state`, and `PayFake.Calls` credited only what left the vault. Run with
   `-race`.
2. **Journal before the engine, order id on the wire (fixes F1).** (a) Never drop `res` when `run` returns an error: if any
   piece is OK, persist `Applied` and `Removed` first, then report. (b) Run `repairDirect` when the keeper is bound and at the
   top of `directOne`/`directSet`, and refuse a new order while one for the same id is active. (c) Fork: accept
   `order=<id>` on `item equipdirect`/`wearset`/`merc-equip` and answer a repeated id with the stored result (needs the fork
   owner; spec change in the ops repo). (d) On a transport error keep the order `equipping` and ask again by order id instead
   of failing it.
   Test sketch: a `DirectFake` option `DropReply` that applies the swap and returns `context.DeadlineExceeded`; assert the old
   item ends as a vault record, the new record is retired, and a second tap does not swap again. A second test: ledger whose
   second `Append` fails.
3. **Carry, Stow and merc as store-backed steps (fixes F3, F2 crash half).** Add a `hold` event (recorded to held, with the
   order id and purpose) logged before the create, completed by `discard`, undone by `release`. At start, a held record is
   resolved by looking at the hero, exactly like an in-flight Keep. For Stow and merc-displaced items add the record first from
   `item full` (the bytes are readable before removal), then `item vault`, then on failure discard the new record: the order
   `vaultBack` already uses (`equip.go:688-712`). Give `EquipMerc` a key (`vk-eqm-<id>`).
   Test sketch: table test over "stop after step k" for k in each flow using the store `hook` and a fake that panics at call
   k; reopen the store and a new keeper on the same files; assert item count in fake plus live records equals one.
4. **Re-attach carried charms by bytes (fixes F4).** Store the norm hex in `Carried`; on Stow find the pack unit by bytes and
   code, never by the saved id. Test: `VaultFake` renumbers units between Carry and Stow; assert the charm, not unit N, moves.
5. **Count, do not compare (fixes F6, F9).** `presentIn`: count the copies the hero holds against the copies the vault expects
   there (kept records minus those whose item was filed again), or drop the heuristic and trust a persisted bot-side key set
   (`d2dctl.Dedupe`, `vault_verbs.go:431`, is written and unused by the keeper). `recordItem`/`vaultBack`: on `ErrDuplicate`
   add with a generation suffix instead of dropping (the store needs an "identical twin" add), or keep the item in the hero.
   Test: the Angelic ring scenario and the keep, wear, displace, keep-again scenario of F6, each against the Sorceress default
   and one other class fixture.
6. **Sale order hardening (fixes F5 tail, F8).** Finish a `credited` order regardless of `SaleOn`; fsync the order file and
   refuse the sale when it cannot be saved; store the stash gold read before the credit and, on a retry after an unanswered
   credit, compare before resending. Mark the lines `sellq` + stage in one batch before the credit so other flows see them as
   busy. Test: `PayFake` that credits and returns an error, then "forgets" its ids; assert one credit.
7. **Truncate on a short write (fixes F10).** In `appendEv` and `commitBatch`, on a write error `Truncate` back to the size
   before the write (or fail closed: mark the store read-only until reopened). Test: hook returns a prefix and an error, then
   the next commit succeeds; reopen; assert zero anomalies and equal state.
8. **Gate the repair (F12)** with `measured()` for the `createBack` and `Save` branches; key the price cache by difficulty or
   drop it at game change (F11).

## 4. Questions for the coordinator

- F1(c) and the merc key need a fork change; which card owns the devp29 spec?
- Is the "identical twin" store change (fix 5) acceptable under the Vault contract, or should a displaced twin stay in the
  hero under the keep mark?
- The brief named `internal/watch` `vaultpoll`; no file or symbol of that name exists on this base. The cache reviewed is
  `internal/watch/vaultcache.go`.
