# Merc revive and hire: what is wired, what is not, live checklist (card mercgold)

> status: current 2026-11-27. Tags: V verified (how), S secondary, U unverified. Source: `docs/probes/sandbox-postdeploy-audit-2026-11-27.md` problem 2.

## Why (b) a dead merc stayed dead 22 minutes with 98k in the stash
- V code read: with `merc.enabled=1` the wire wraps the town (`cmd/d2bot/mercwire.go mercTown`). The legacy chore (`bot.mercWanted`, `bot.mercLegacy`) needs `r.d.Town.(bot.MercTown)`; the wrapper embeds the `Townsman` interface only, so the assertion fails and the legacy path is silent. The wire's own revive (`reviveFirst/reviveStep`, `d2dmerc.Reviver`, gold = pack + stash, stash fetched by `fundRevive`) ran only inside `Chores`, so only a trip started for another reason revived him.
- V audit: that revive worked live once (23:27, 1680 gold). The knob `bot.mercReviveUnverified` gates only the legacy path (`mercLegacy`); it is not read by the wire. So nothing needs flipping for the wire revive.
- Fix (no new knob): `mercTown.NeedsTrip` returns `merc dead, revive due (gold G, need N)` when `merc.ShouldRevive` says yes (same bar, hourly cap, quote, failure backoff `mercSkipGap`). V offline: `TestDeadMercWithGoldIsATripReason` (Sorceress, Paladin). U live: that the trip then lands at Kashya/Greiz and pays (the audit's one revive says yes).
- Legacy path with the knob off now logs `merc dead since Nm, revive off (bot.mercReviveUnverified)` every 5 min and sets a standing refusal `merc revive` in `/api/explain`. V offline: `TestMercDeadReviveOffIsVisible` (Sorceress, Paladin).

## Why (a) Questy has no merc
- V code read: nothing can hire. `mercWire.hirer` is nil in production (`HireStep` returns "" at once; no d2d reader of the hire window exists, `d2dmerc.ErrNoOffers`). The legacy `town.Hire` path is not reachable either (see above) and needs a row picker (`ErrMercRowUnknown`).
- V policy: `merc.ShouldHire` refuses the Act 1 Rogue with `free rogue pending` until Blood Raven is done or skipped, except classes with trait `mercEarlyRogue` (only Necromancer, `internal/d2/classtraits.json`). A Paladin therefore waits for Blood Raven, and the quest board has not started that chunk (audit problem 1/2, card mercfree).
- Not fixed here: a paid Act 1 hire needs the Kashya hire window reader and a click verb (both U, `docs/merchire.md`), and the board start of Blood Raven belongs to the quest board owner. Adding a clvl row to the class traits without a hirer would be a dead letter, so none was added.
- Visible now: every 10 min the wire logs `merc: none at clvl N (Class): act 1 rogue hire <policy reason>; no hire window reader wired, so no paid hire can happen` and puts the text into `lastDecision` of `/api/v1/instances/{id}/merc/`. V offline: `TestNoMercSaysWhy` (Paladin, Necromancer).

## Live checklist (sandbox, 2 h window; nothing here ran live)
1. Revive trip (wire): merc dead, hero in the field with pack + stash gold above `merc: gold wait` need. Expect within one poll `trip: merc dead, revive due (gold ..., need ...)`, a walk home, `merc: revived price=... level=...`, merc alive in `/merc/`.
2. Poor purse: no trip, `merc: gold wait need=..` as before.
3. Failure: block the NPC (or take the gold during the dialog): `merc: revive skipped (failed: ...)`, no new trip for 2 min, no loop.
4. Legacy path (only if `merc.enabled=0`): with `bot.mercReviveUnverified=0` expect the `revive off` line and the `merc revive` refusal in `/api/explain`; flip the knob on the sandbox only after step 1 shows the same dialog click works, then expect `merc chore what=revive done=true`. The knob row in `data/darklaunch.json` stays until that is seen.
5. No merc: `merc: none at clvl ...` once per 10 min; after Blood Raven the free claim (`merc: claim verified`) must end it.
