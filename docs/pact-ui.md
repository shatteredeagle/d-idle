# Pact offer UI (TASK-168)

> status: built and tested offline (Go tests, httptest); the page script was NOT run in a browser and nothing ran against a live bot. Design: `docs/research/pact-system.md`; away rules: `docs/pact-away.md`. V = proven by a test in this tree, U = unverified.

## Routes

Viewer (GET): `/watch/pacts` (static shell, CSP `script-src 'self'`), `/watch/pacts.js` (all text via `textContent`, V by a test that forbids `innerHTML`), `/api/pacts[?propose=id:rank,id:rank]` (`-` = the empty set), `/api/pacts/offer` (away text). Answer listener (POST, same gates as `/api/policy`): `/api/pacts/select {picks:[{id,rank}]}` plus the away routes. V `TestPactUIRoutes`.

## What the page shows (`pact.View`, `internal/pact/view.go`)

- The catalogue by lever family, only Pacts for the hero's class (per-class scope; V two classes).
- Per rank: heat, the generated effect line (from the catalogue text, never typed on the page), the reward (tally percent, legacy charm points, title) and a status: `ok` or the reason it cannot be chosen now (over the cap, a declared incompatibility with its reason, two no-safety Pacts, no slot). V `TestViewProposalHeatAndReasons`.
- Heat bar against the cap (12, growing to 20) and the slots; the summed rewards of the current and the proposed set (reward delta).
- Current set and proposed set side by side, plus the levers the proposal would set (policy rows, tuner knobs, death limit) next to what is held now.
- Unlock (TASK-181 pactladder): the Pacts use the shared unlock ladder (`docs/unlock-ladder.md`), not their own state. A Pact is open when the ladder entry `pact_<unlock_trigger>` is open (`pact.LadderID`, `Config.Gate`); no gate means every Pact is locked, and a trigger without a ladder entry (`baal_nightmare`, which is a proposal not decided, and the counted `hall_count:`/`tally:` ones) never opens. Entries in `data/unlocks/ladder.json`: `pact_andariel_first_kill` (a `boss_kill` of Andariel, any difficulty) and `pact_baal_normal` (Baal killed in Normal). A locked Pact stays a silhouette (`???`, only the unlock hint) in `/watch/pacts` and `/api/pacts`; `Controller.Choose` refuses it. The kill event goes `startPacts` -> `Unlocks.Observe(boss_kill)`, so the reveal (Chronicle line plus the "Unlocked since you left" section) fires through the ladder and nothing opens silently. `Controller.Unlock` is gone. Hardcore is a separate flag, not an entry. V `TestLadderOpensPacts`, `TestNoGateMeansLocked`, `TestLadderReplayIdempotentAndPersistent`, `TestViewLockedSilhouettesAndChoose`, `TestShippedPactUnlockReveals`, `TestChooseThroughDeskFollowsLadder` (Sorceress and Paladin). U: the kill event is not seen on a live run; the unlock is per profile, not per hero.
- Migration: an old `pacts-state.json` with an `unlocked` list is adopted into the ladder at start (`adoptLegacyUnlocks`, no reveal: the viewer saw them open), then the field is dropped. Idempotent; keys this version does not know are written back unchanged. V `TestMigrationAndUnknownKeys`.
- Away-consent lines: the record itself stays in `pacts-state.json`; each grant, revoke and expiry also writes one Chronicle incident line and an away event (`watch.AwayPactConsent`), and the Pacts return section lists those since the viewer left. A consent that ran out is dropped from the file and said once. V `TestConsentLines`. U: not seen in a browser.

## Return screen: what each Pact cost and paid

The "Pacts" section (existing lines) now adds, per chosen Pact: the effect (cost), how long it acted, and for a Pact that sets a policy row, "After, X: counters. Before, X: counters." from the same counters as the policy cards (`PolicySheet.Counters`). Acting stretches are `Controller.Spans` (kept in memory, 200 at most). V `TestSpansTrackActivation`, `TestCostLineWithoutAndWithCounters`. Limits (U): a Pact that only sets tuner knobs (Blood Raven's Tithe) has no counter, the line says so; the tally bonus is shown as promised, because no code pays it yet.

## Away-consent record

Stored in the Pact state file (`pacts-state.json`, 0600) next to the choice, not in the decide store: the Pact controller owns it and the decide store holds sheet rows and rules (the third criterion is met by that record, not by the decide store; U whether the card meant the store literally).

## Not done

No browser run. No Nightmare Pact or `/players`. Choosing through the page uses the answer listener token like the policy page.
