# Farm gold (the player's order; fix for the poverty loop)

> status: built and tested offline (fixtures, stub DOM). Nothing ran live. Tags: V proven by a test named here, S secondary (a doc or the log), U unverified. Task: see `backlog/tasks` (card goldfarm). Related: `docs/areathrash.md` (the Blood Moor finding), `docs/repair.md` (the pattern copied), `docs/whyline.md`.

## What the player sees

On `/watch`, the Gear condition panel (panel version 2, additive; the manifest id, endpoint and band are unchanged) has a second button next to Repair now: **Farm gold**. Pressed, the hero goes to a safe zone and farms until the player presses **Stop farming**. One status line under the gear chips reads "Farming gold on your order: 40 of 190 in Stony Field." (or "... 200 of 190 reached, still farming in Stony Field until you stop."). The Why line says the same ("farm_user"). The log has a distinct line `goldfarm: farming gold on user request` (once per zone change). Golden: `internal/watch/testdata/gear_farm_dom.txt` (V `TestGearPanelFarmGoldDOMGolden`, node).

## Endpoint and state

- `GET /api/farmgold` (viewer): `{schema:1, on, since, earned, area, have, want, reached, note, text}`. The same object is the additive field `farm` of `GET /api/repair`, so the panel needs no second poll.
- `POST /api/farmgold {"on": true|false}` on the answer listener: the repair endpoint's gate (origin list, token or passcode, JSON only, small body, unknown fields refused; V `TestGoldFarmRoutesGate`).
- `goldfarm-ui.json` in the state dir holds the switch, the start time and the gold the earned count starts from. It survives restarts like the repair request and the inventory marks (V `TestOrderPersistsAcrossRestart`). A damaged file is an error at start, not a silent off.
- A named run (`D2BOT_RUN`: gate, soak, drill) never farms on an order (`wireGoldFarm`), like the repair gate: measured runs stay comparable.

## Target amount

`goldfarm.Want(repairNeed, tuning) = repairNeed + resource.KitCost + PovertyMargin` (V `TestWantIsRepairPlusKitPlusMargin`). `repairNeed` is the lower-bound estimate `town.RepairEstimate` already feeds the gear panel (0 when nothing is damaged). KitCost (portal scroll plus `HPFirst` healing potions) plus `PovertyMargin` is exactly where poverty mode ends (`resource.PovertyCheck`), so reaching the target means: the repair is paid, the kit is paid, and poverty would not come back on a sale that barely covers it. No new knob. "Have" is pack plus stash gold, so banking is not counted as a loss.

The target only drives the status line. The order does not end when it is met: it keeps going until the player stops it (user spec). U: whether a higher bank floor is wanted; the town's own `StashGoldBase + level * StashGoldPerLvl` keep line decides what is banked at each visit, unchanged.

## Safe zone: one rung down, as data

`Plan.FarmRung` (`internal/bot/goldfarm.go`, V `TestFarmRungOneRungDown`): among the plan zones that may be farmed at all, ask `Pick` what the level is ready for, and farm the zone before it in the plan's order (the first zone when the pick is the first). "May be farmed" is the list a broke hero already uses (`safeTier`, Normal act 1; V reused), the allowed acts, and the thin-kit rule (`KitLvl`). The level windows and the melee entry level are the plan table's (`melee_enter_clvl`); no class or skill name is in the code (the caller passes `MeleeTank(class)`). V for a caster and a melee class: clvl 10 farms Stony Field, clvl 7 farms Cold Plains (caster) or Cave Level 1 (melee, Stony Field opens at 7 for it), clvl 1 farms Blood Moor.

Skipped zones drop one more rung: cleared in this game, banned or skipped by the death-spiral and border-loop breakers (`avoidedNow`), guarded (`guardedAreas`), the Den without a potion buffer. V: the breaker's skip moves the order to the next rung (`TestGoldFarmTargetBothClasses`). Two gates stay as they are: a gated gear verdict allows only the first field (`safeFarmArea`) and so does a hero without a town portal (the walk home works from there only); V `TestGoldFarmKeepsToTheFirstFieldWhenGatedOrWithoutPortal`.

When the level is below the first zone's entry level there is no safe zone: the order ends itself, logs `goldfarm: stopped, no safe zone`, and the panel says "Farming gold stopped: level N is below the first safe zone" (V `TestGoldFarmHaltsWhenNoSafeZone`, `TestFarmRungTooLow` for both classes). When every zone is only temporarily excluded the order stays on and waits (status note "waiting, every safe zone is cleared or avoided for now"; the normal no-target path starts a fresh game when the zone is cleared; V `TestGoldFarmWaitsWhenEverythingIsAvoided`).

U (needs a live look): that the rung is "clearly over-levelled" enough. At clvl 10 Stony Field is 2 levels under the pick; if that is too hot for a poor hero the answer is a larger step in `FarmRung`, not a class check.

## What counts as done with a trip, and the rest of the loop

The order is not a trip; it chooses the area of the leveling slice (`target0`, and `goldFarmStep` replaces a campaign quest step the way recovery does, `levelSlice` is the shared body). Everything else runs as always: sell trips and the lean play of poverty mode (`updateStarvation` is called every choice, so its hysteresis and `PovertyAdvice` keep updating; V `TestTarget0UsesTheOrder`), gold pickup, banking at the visit, the repair gate (a manual Repair now request is still honoured at the visit), the town trip trigger and chicken. The order sets none of these and bypasses none. Class-free, V with a Sorceress and a Paladin.

## Area-thrash and hysteresis

The border-loop breaker (`docs/areathrash.md`, escalating skips) keeps working: its skip is an exclusion for the rung search, so a bouncing zone is left for the next rung and comes back only when the skip ends. The chosen zone then passes `holdTarget` (3 minutes at least, longer after flipping), so a rung flip does not walk the hero around. Poverty hysteresis (band between entry and exit) is not touched.

## Not done, open questions

- Live check (checklist below). The status line's "of Y" uses the repair estimate, a lower bound.
- Does `Repair now` get paid from banked gold? The town withdraws stash gold only for the kit (U, `basicsfix.go`); the "reached" state counts the stash, a repair needs pack gold. If the live check shows a banked hero waiting for gold it cannot reach, count pack gold only.
- Open choice for the owner: one rung (this doc) versus the existing `PovertyLevels` offset. Both exist in code; `starveArea` uses the offset.

## Live checklist (nothing ran live)

1. Start the bot with `--ladder --watch`; the gear panel shows Farm gold (needs `GET /api/repair` to answer).
2. Poor hero at clvl 6 to 10 in the field: press Farm gold; the log shows `goldfarm: farming gold on user request area=..`, the Why line says "Farming gold on your order", the status line counts up; the hero stays in that zone and goes to town for its normal trips.
3. Restart the bot: the order is still on. Press Stop farming: normal planning resumes at the next area choice.
4. Break the pick: skip the farm zone with the breaker (cave shuttle): the order moves one rung down.
5. `d2cheat ledger` shows no writes from the order (it only reads).
