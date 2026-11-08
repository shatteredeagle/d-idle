# Pact away safety (TASK-169)

> status: built and tested offline with fakes and httptest; NOT run against a live bot or a browser. Design: `docs/research/pact-system.md` section 5. V = proven by a test in this tree, U = unverified.

## Presence (who counts as a human)

`pact.Presence` (`internal/pact/away.go`): present while an authenticated viewer request came in within `limits.presence_min` minutes (default 5) or a video stream is open (`Video.Viewers() > 0`). The watch server calls `Fun.OnView` for every request that passed the viewer gate (`serve.go`), so a polled page or a stream both count. V `TestPresenceWindowAndStream`, `TestPresenceGatesPact` (Sorceress, Paladin). U: a browser tab left open on a poll keeps the Pact on; that is a human-chosen state, not detected.

## Consent API

All on the answer listener, same gates as `/api/policy` (origin list, token, JSON only, small body; V `TestPactRoutesGatesAndErrors`).

- `GET /api/pacts/offer` (viewer): `{offer, hardcore}`. The offer lists, per chosen Pact, the effect line, what can be lost with numbers from the catalogue (death limit, clvl cut-off), the consent state, the phrase to type; the rules (presence minutes, life floor, safe minutes, maximum hours) and, on Nightmare or Hell, the entry warning (resist -40 / -100 and death XP loss 5 % / 10 %; V `DifficultyLevels` in `docs/probes/pactnative.md`). A gold or minutes cost per death is NOT given: no number for it exists in the data (U).
- `POST /api/pacts/consent {pact, rank, awayHours, typed}`: `typed` must be the Pact's name (case and spacing aside); `rank` must be the chosen rank; `awayHours` 1 to 24 x `consent_days` (0 = the maximum). Dated record in `pacts-state.json`.
- `POST /api/pacts/revoke {pact}`.
- A consent ends: after `awayHours`, at the 7 day cap, when the Pact's rank changes or it leaves the set, and when the difficulty changes (so it is asked again per difficulty). V `TestConsentHoursExpiryAndRankChange`.

## Pause rules

A Pact is inactive (and its levers put back) in town, in the first minutes (existing safe zones), and now under the life floor: `limits.life_floor_pct` (default 35) of max life. It resumes 10 percent above the floor (no flapping). Each change writes one journal line ("Pacts paused: life under 35% ..." / "Pacts resume"). This pauses the Pact, not the goal manager: the hero keeps playing normally (U: the card text "pauses the hero in town" is read this way). The death-limit pause of Andariel's Shadow still holds the hero. Unknown life never pauses. V `TestLowLifePauseAndResume`.

## Hardcore (decision F12)

A separate flag (`hardcore.json`, 0600), never in `pacts-state.json`, not on the Pact sheet, no heat cost. `POST /api/pacts/hardcore {action: confirm|revoke|activate, typed}`: confirm needs the HERO's name typed (a Pact name or a click-word is refused); the away allowance is off by default and needs a confirmation first. Activation always answers 501 "not active yet": the bot cannot create or play a hardcore hero. V `TestHardcoreSeparateAndStubbed`, `TestPactDeskConsentAndHardcore`.

## Files

`internal/pact/away.go`, `internal/watch/pactapi.go`, `cmd/d2bot/pactwire.go` (adapter `pactDesk`), optional limits `presence_min` and `life_floor_pct` in the catalogue.

## Not done

No page for the offer yet (TASK-168). No live run. Presence from a real browser is untested.
