# Pact offer UI (TASK-168)

> status: built and tested offline (Go tests, httptest); the page script was NOT run in a browser and nothing ran against a live bot. Design: `docs/research/pact-system.md`; away rules: `docs/pact-away.md`. V = proven by a test in this tree, U = unverified.

## Routes

Viewer (GET): `/watch/pacts` (static shell, CSP `script-src 'self'`), `/watch/pacts.js` (all text via `textContent`, V by a test that forbids `innerHTML`), `/api/pacts[?propose=id:rank,id:rank]` (`-` = the empty set), `/api/pacts/offer` (away text). Answer listener (POST, same gates as `/api/policy`): `/api/pacts/select {picks:[{id,rank}]}` plus the away routes. V `TestPactUIRoutes`.

## What the page shows (`pact.View`, `internal/pact/view.go`)

- The catalogue by lever family, only Pacts for the hero's class (per-class scope; V two classes).
- Per rank: heat, the generated effect line (from the catalogue text, never typed on the page), the reward (tally percent, legacy charm points, title) and a status: `ok` or the reason it cannot be chosen now (over the cap, a declared incompatibility with its reason, two no-safety Pacts, no slot). V `TestViewProposalHeatAndReasons`.
- Heat bar against the cap (12, growing to 20) and the slots; the summed rewards of the current and the proposed set (reward delta).
- Current set and proposed set side by side, plus the levers the proposal would set (policy rows, tuner knobs, death limit) next to what is held now.
- Unlock: a Pact whose trigger has not happened is a silhouette (name `???`, no effect lines, no learn text, only the unlock hint). `Controller.Unlock` records a trigger in the state file; `Controller.Choose` refuses a locked Pact. V `TestViewLockedSilhouettesAndChoose`, `TestUnlockSurvivesRestart`. The bot unlocks `andariel_first_kill` on an Andariel kill event (the event the Hall uses) and `baal_normal` on a Normal Baal kill (`unlockOf` in `cmd/d2bot/pactwire.go`). U: the kill event is not seen on a live run; the unlock is per profile, not per hero; counted triggers (`hall_count:`, `tally:`) are never opened (U).

## Return screen: what each Pact cost and paid

The "Pacts" section (existing lines) now adds, per chosen Pact: the effect (cost), how long it acted, and for a Pact that sets a policy row, "After, X: counters. Before, X: counters." from the same counters as the policy cards (`PolicySheet.Counters`). Acting stretches are `Controller.Spans` (kept in memory, 200 at most). V `TestSpansTrackActivation`, `TestCostLineWithoutAndWithCounters`. Limits (U): a Pact that only sets tuner knobs (Blood Raven's Tithe) has no counter, the line says so; the tally bonus is shown as promised, because no code pays it yet.

## Away-consent record

Stored in the Pact state file (`pacts-state.json`, 0600) next to the choice, not in the decide store: the Pact controller owns it and the decide store holds sheet rows and rules (the third criterion is met by that record, not by the decide store; U whether the card meant the store literally).

## Not done

No browser run. No Nightmare Pact or `/players`. Choosing through the page uses the answer listener token like the policy page.
