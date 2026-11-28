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

## Acceptance (fork differential, to be run by the fork owner)

- equipdirect with `order=a`, drop the reply, `item order a` returns the same `removed=` bytes; a second equipdirect with
  `order=a` answers `replayed` and the body is unchanged.
- The same after a d2d restart between the two calls.
- wearset of a two-hander over a weapon and a shield: `item order` returns both removed items.
- A measured run refuses all of it.
