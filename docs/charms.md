# Charms in the Vault: record, Carry, Stow

> status: built offline 2026-11-27 (card charmvault). Design background: `docs/design/charm-management-2026-11-27.md`. Nothing here was run against a live game. Tags: V verified in this tree by test, U unverified live.

Decision (user, 2026-11-27): a charm is handled like any other item. The bot records it into the Vault, the PLAYER identifies it and decides to carry it. The bot never carries or packs charms by itself; inventory management beyond starter gear and potions is player-driven.

## What exists
- Recording (V, `internal/lootrules/vaultpreset_test.go` `TestVaultPresetCharmsAllRecorded`): under `loot.vault` every small, large and grand magic or unique charm, identified or not and with or without pack room, is a `vault` action. No new knob.
- Data (V): `internal/charms/charms.json` holds the codes (`cm1`..`cm3`, types `scha`/`mcha`/`lcha`), the cells and the legal qualities (magic 4, unique 7; rare, set and crafted charms do not exist). The cell sizes 1x1, 1x2, 1x3 are U until read from `misc.txt`.
- Carry (V offline, `internal/vaultkeep/carry.go`): `POST /api/vault/carry {ids:[id]}` for an identified charm record creates the item in the backpack at the first free cell (`item create` pack, idempotency key `vk-carry-<id>`), puts the keep mark on it, writes it to `vaults/<hero>/vault/carried.json` and retires the vault record ("carried to the backpack"). Refusals: not a charm (`kind`), not identified (`reveal`), no free cell (`room`), dead or no game, a measured run (`off`). One call, no town trip, like the direct Equip. No knob: it is gated by `loot.vault` and a measured run.
- Stow (V offline): `POST /api/vault/stow {ids:[id]}` (the id is the old record id the carried list shows) reads the unit, removes it from the hero (`item vault`) and records it again. When the vault has no room the call is refused first; when the item cannot be decoded it is created into the stash instead and the answer says where.
- UI (`vault.js`, `vault.json`): a Carry button on the detail of a charm record (disabled with the reason while unidentified or off), and a "Carried charms" strip with one Stow button per charm. `GET /api/vault` carries `carry:{enabled,why,carried,cells}`; a row carries `charm:true`. The identified stats of a charm are the normal item lines. Button feedback comes from the shared busy-button helper (`toast.js` `dBusy`, branch agent/vaultperf): it wraps every POST made from a click, so it needs no code here and applies once that branch is merged.
- Pack chores (V, `internal/town/stash.go`, test `TestCarriedCharmStaysInThePack`, needs the game tables to run): the pickit stash chore and the valuables insurance skip a charm that carries the keep mark. The tidy policy and the sell chores already honour the mark (`internal/tidy/tidy.go` Decide, `town/shop.go`). Any other kept item may still be stashed.
- Sell all (V, `TestSellAllLeavesCarriedAndPinnedCharms`): a carried charm is no vault record, so it is never in a sale list; a pinned charm (the vault's keep mark) is left out by the existing plan.

## Not done / lists
- Dead pickit clauses: `quality: rare` and `stat:fcr` on the charm rules never match a real charm (V Arreat, no rare charms, no FCR affix). Left as is: `internal/loot/charms_test.go` pins an FCR charm as a stash on purpose and the legacy mirror in `pickit_starter.json` has the same lines; not trivial to remove without touching two files outside this card.
- Crash between `item create` and the record retire in Carry leaves a copy in the vault and one in the pack (the create key stops a second create; no repair pass yet).
- No card for build value, pack budget or suggestions (design sections 4 to 7).

## Live checklist (not run; needs one hidden instance and a throwaway character)
1. `item create` with `pack=1` for a charm: does the engine place a 1x2 and 1x3 at the first free cell and answer ErrNoSpace when none fits? Compare the cell sizes with `misc.txt` `invwidth`/`invheight`.
2. Does d2d count a carried charm in the character stats (life, resists) when it sits in the pack? Read `stats` before and after a Carry of a life charm.
3. Does `item vault <unit>` take a unit from the PACK (the verb is documented for stash and belt items)? If not, Stow needs the town path.
4. Does the keep mark survive a game restart (unit ids change; `ReconcileMarks` re-attaches by item)? Check a Carry, restart, one town visit: the charm stays.
5. Does the `d2d` Stow of a charm created by Carry decode (`item decode`) and re-record with its stats?
