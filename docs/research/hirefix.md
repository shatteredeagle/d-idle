# hirefix: bot-side HIREREADER (2026-11-02)

> status: current. Design: hirereader-design.md (branch agent/hirereader). Tags: V verified (test), U unverified live.

## Done (offline; fake-dump tests, each fails when its fix is reverted)
- B2: `internal/d2dsrc/player.go` reads `dead` before the `present` return; `present:false,dead:true` sets `Extras.Merc{Dead:true}` (no unit), so `d2dmerc.FromResult` gives Owned, not Alive. Test `TestDeadAndGoneIsOwnedDead` (V offline; the dump shape is S live, drill item 1 records it). Class stays 0 for that state (the code is gone from the dump).
- B1: `GU`, `IW`, `0A` map to 338, 359, 561 (`tables.go`). Test `TestHirelingCodesMapToClasses`. Codes IW and 0A are U live (drill item 2).
- B4: `d2dmerc.ParseQuote`, `Reviver.LastQuote/Quote()`; the click is skipped with `ErrCantAfford` when gold is below the quote; `merc: revived ... estimate=` logs the quote when read. `ParseQuote` is tested; the Revive flow itself has no fake (the NPC walk needs the live loop): U.
- Judge (ops repo commit 66da100): `HIRE_RE` did not match the bot's real `merc: hired class=338 ...` (colon), so a paid hire never counted. Added; test added.

## What the gate can now confirm
Claim or paid hire (`merc: claimed free rogue` plus a later `(merc=alive)` read, or `merc: hired class=`), for every class once B1 is live (route-guard `merc=alive` now true for Desert, Iron Wolf, Barbarian). A death plus revive shows as `merc: revived price=N ... estimate=Q` (N paid, Q quoted).

## Cannot confirm
Aura/level/name (needs the d2d dump keys, private fork), hire-versus-claim by dump alone, replacement of a merc, the revive result cells (D1). The minute line has no merc field; I added none (the log lines carry it). B3 (Hirer over `menu`) deferred: not small, needs the window flow live.
