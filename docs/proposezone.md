# The hero proposes the next zone (TASK-131)

> status: built and tested offline on fixtures 2026-11-07 (ADR 0018, decision F3). NOT verified live and NOT wired into `cmd/d2bot` yet (see "Not done"). Tags: **V** verified by a test in this repo (named), **U** unverified or a guess.

Code: `internal/decide/proposezone.go` (+ `proposezone.json`, the table), `internal/watch/proposals.go` (desk, section, routes), wiring in `internal/watch/funwire.go` (`FunConfig.Proposals`). Tests: `internal/decide/proposezone_test.go`, `internal/watch/proposals_test.go`, goldens `internal/watch/testdata/proposals/`.

## The rule

- The proposer looks (`Proposer.Look`, called by `ProposalDesk.Tick` from `Fun.Settle`, every ~2 s) at the field the hero farms. For each next zone of that field (table order) it asks one range function (the tally zone table, `currency.TallyRows.Zones`: ONE table, not a second one) and proposes the first zone where `clvl >= min + class lead` and `clvl <= max`, the kit is `OK` and the belt holds at least `minPotions` (2) potions.
- The class lead is data (`proposezone.json` `classes`): sorceress 1, paladin 0 (U: guesses about fragility). An unknown class is an error naming it (V `TestProposerUnknownClassIsAnError`), never the Sorceress default.
- The zone graph (Act 1 Normal) is data too: `zones[].next`. U: the ranges themselves are the veteran ranges tagged UNSURE in [tally-strip.md](tally-strip.md); TASK-147 classpacing will supply per-class pacing, then replace `classes` (swap the table with `ParseProposeTable`).
- ONE question per pair: key `<from>><to>`. Text: "I am level N and ready for <Zone> (recommended L1 to L2, kit ok). Go there?" with three reasons (level, potions, deaths) and options go / stay / later. Default when nobody answers: stay (the hero keeps farming).
- A Codex line is added when the zone's area first is not in the Codex: "First time in Cold Plains: <tip of the firsts row>" (`CodexIntro`; V `TestProposalAwayWaitsThenAccept`).

## Repeat and away rules

- Copies the store's semantics (the key is a free zone pair, so it does not live in the catalogue): after an answer of `stay` or `later` the pair is not asked again for `ladder.repeatCooldownMin` (30); after `ladder.quietAfter` (3) stays in a row it is quiet for good; `later` never counts toward quiet; `go` resets. State is a JSON file and survives a reload (V `TestProposerStayCooldownQuietAndReload`, `TestProposalStayPersistsAndStaysQuiet`).
- Nothing fades: an unanswered proposal stays pending for hours (V `TestProposerWaitsWithoutFading`). If the hero leaves the field by the player's own order, the old question closes without a count (V `TestProposerClosesWhenTheHeroLeft`).
- AWAY: the viewer is away when the last look (`Visit`) is older than `AwayMin` (20; the desk takes it as a config field, the knob row `ladder.awayMin` is NOT added yet). Away or not, the proposer and the desk never place an order: the only code path to an order is the answer "go" (V `TestProposalAwayWaitsThenAccept`: 10 hours away, 0 orders, hero still in Blood Moor, the proposal waits, two classes).

## Answering

- `GET /api/proposals` (viewer, token gate): pending list with `farmingMins`. `POST /api/proposals/answer {id, answer, approve}` on the answer listener, same gate as the other POST routes (origin list, token, JSON only, small body; V token and method tests).
- `go` places `OrderGo` (until you cancel, replacing open orders) through the same placer as `POST /api/orders`, so the charter flow applies: above range the answer is 402 `charter-needed` with the price, the proposal stays pending, nothing is charged (V `TestProposalAcceptAboveRangeCostsACharter`; a proposal is in range when raised, so this only happens if the hero lost levels). The bot's route guard keeps the last word on arrival (unchanged, `internal/bot/routeguard.go`). `stay` and `later` record the answer.
- Return screen: the `proposals` section lists pending proposals, the Codex line and "The hero has been farming <Zone> for N min since it was ready", with buttons Accept, Decline, Ask me later (`SectionButton.Act = "proposal"`, `return.js` posts them). Golden: `testdata/proposals/*-section.json`.

## Not done / not verified

- Not wired in `cmd/d2bot`: the bot must pass `FunConfig.Proposals` with a `Facts` reader (kit verdict, potion count, recent deaths from the runner) and a state file path next to the other ladder state. Until then the placeholder stays.
- Knob `ladder.awayMin` is not in `internal/env`.
- Above-range orders from the map already cost a charter (TASK-129); the hero's warning text is the quote's. No change here.
- Live: nothing. The ranges, leads and the 20 minute window are U.
