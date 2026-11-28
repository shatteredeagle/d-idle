# Fork change spec: order id on the direct item verbs

> status: proposal (card fix1, 2026-11-27). Nothing here is built; no fork edit was made. Needs the fork owner and the user's go.
> Source: docs/design/fable-review-2026-11-28/01-item-flows.md, findings F1 and F2.

## Why

`item equipdirect`, `item wearset`, `item unequipdirect` and `item merc-equip` remove an item from the game and hand its bytes
back in the reply. The reply is the only copy. The bot now journals the reply before any other step and reads the slot before
the call (internal/vaultkeep/equipdirect.go, `wornBefore`, `resolveDirect`), but a reply that never arrives can only be judged
by looking, and a wearset piece that pushes a second item off (a two-hander and the shield) cannot be judged that way at all.
The wire has no free field today (V: read in internal/d2dctl/direct_equip.go, the lines are `item equipdirect <hex> <slot>
[identified=1]` and `item wearset <slot>=<hex> ... [identified=1]`; whether the fork ignores an unknown `key=value` is U), so
the bot does not send the id yet.

## The change (fork side)

1. All four verbs accept one optional argument `order=<id>`, 1 to 64 of `[A-Za-z0-9._:-]` (the bot's `orderIDRe`). Without
   it the verbs behave as today.
2. The fork keeps the last 64 answered orders: id, verb, the full reply text. The table is written into the character's
   side file at the same moment the save is written, so it survives a d2d restart together with the swap it describes.
3. A call whose `order=` is in the table does NOT run again: it answers the stored reply with the extra word `replayed`.
4. New read verb `item order <id>`: answers the stored reply, or `err unknown-order` when the id was never applied. It is a
   read (ADR 0022): it never makes a run CHEATED.
5. `caps` lists `item.order` when 1 to 4 are present.
6. Measured-run refusal is unchanged and comes before the table lookup.

## The bot side once the capability is listed

- `DirectEquip.run` appends `order=<id>` when `caps` has `item.order`.
- `resolveDirect` asks `item order <id>` first: a stored reply is parsed and filed like a live one; `unknown-order` ends the
  order as "nothing changed". The look at the hero stays as the fallback for an older build.
- `EquipMerc` writes an order-book entry before the create and resolves it the same way; its create key stays `vk-eqm-<id>`.

## What the bot does until the fork has it (card vault2, 2026-11-28; offline fakes only, NOT verified live)

The requirement stands: without `order=` and `item order <id>` the bot can only judge a lost reply by looking, and looking
has two blind spots that only the fork can close. Until then:

- **Two-hander and the shield.** Before `item equipdirect` or `item wearset` into a hand slot the bot reads BOTH hands
  (unit and bytes; `EquipOrder.before_off`, internal/vaultkeep/equipdirect.go `otherHand`, `resolveDirect`). After a lost
  reply the other hand's item that is gone from its slot AND from the game is filed from that copy next to the weapon; one
  that is off the body but still readable keeps the order open (`order_in_flight`). V offline
  (`TestVault2TwoHanderLostReplyFilesWeaponAndShield`). Blind spot (U, needs the fork): the engine changed the item's
  bytes between the read and the swap (durability), or a second process moved it; `item order` returns the engine's own
  bytes and ends the guess.
- **Mercenary.** `item merc-equip` has no slot read on the bot side, so the bot writes an intent before the create with
  the bytes of every hero item (`merc-intents.json` next to `equip.json`, internal/vaultkeep/mercjournal.go). At the next
  Run (and at once after a lost reply) it reads the hero again: the created copy still there means not given (the copy
  goes back, the record stays); the copy gone means given (the record leaves) and every item that is new on the hero and
  that his kind can wear came off him: it is recorded first, then removed, and the record carries the order id in
  `Found.Area` so a repeated pass files nothing twice. V offline (`TestVault2MercStopAfterEveryStep`, a stop or a lost
  reply after each of the five game steps, two hero classes, two merc kinds). Blind spots (U, need the fork): whether
  `item full -1` lists the mercenary's own items on the real build (the fake does not; if it does, a given copy looks
  "still there"); an item the hero picked up between the intent and a restart that the mercenary could wear is filed as
  displaced (never lost: it becomes a vault record). `item order <id>` with the displaced unit and bytes removes both.
- **Fork ask added by this card:** point 2's table must also keep, per `item merc-equip` order, the displaced unit id
  and its bytes, and `item full` should say in `caps` whether it lists mercenary items (`item.full.merc`).

## Acceptance (fork differential, to be run by the fork owner)

- equipdirect with `order=a`, drop the reply, `item order a` returns the same `removed=` bytes; a second equipdirect with
  `order=a` answers `replayed` and the body is unchanged.
- The same after a d2d restart between the two calls.
- wearset of a two-hander over a weapon and a shield: `item order` returns both removed items.
- A measured run refuses all of it.
