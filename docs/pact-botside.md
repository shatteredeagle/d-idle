# Bot-side Pacts (TASK-167)

> status: built and tested offline with fakes; NOT run against a live bot. Design: `docs/research/pact-system.md` (a design note); catalogue: `docs/pact-catalogue.md`. Tags: V = proven by a test in this tree, U = unverified.

## What it does

`internal/pact/botside.go` is a `Controller`. The chosen Pacts (`pacts-state.json` in the state dir, 0600) are put through the existing levers and nothing else (no memory write):

- knobs through the tuner's edit (`bot.Tuner.Edit`, all or nothing, nil resets to the profile value), policy rows through the sheet (`PolicySheet.SetByPact`: same knobs, rules and log line as a player's change, but no recorded change, so no "change it back" card for a held row; the player's earlier option is put back when the Pact stops).
- Blood Raven's Tithe: ranks 1 / 3 / 5 empty the four belt columns (`resource.beltCol1..4` = 4, nothing bought or belted) and zero the potion trip knobs below clvl 8 / 14 / 99 (`until_clvl` in the data). V `TestTithePresentConsentAndSafeZones`, `TestRankLadderForPotions` (Paladin, Sorceress). U: the bot still drinks a potion it finds in the pack; "never drink" at rank 5 is not enforced (no knob switches drinking off).
- Andariel's Shadow: death limit 3 / 2 / 1 (`death_limit`). Each counted death raises a count; at the limit the hero is paused through the goal manager once and waits. V `TestShadowDeathLimitPausesAndResumes`. The count restarts when the player takes the hero back (status Paused to Running) or on a new process (U: "per expedition" is not defined further; a death returns the hero to town, so a town visit cannot reset it).
- Rakanishu's Bargain rank 3 and 5 use the same generic path (row `tp` walk, knob); rank 1 ("no portal after the Den quest") and the waypoint ban have no code (U). V `TestBargainRank3SetsRowAndKnob`. Griswold's Debt and Cain's Silence are row-only and work through the same path.

## When a Pact acts (live)

All of: not off (below), out of the safe zones (town; the Den of Evil, U: the quest state is not read; the first 10 minutes AND below clvl 5 count as safe until both are over; numbers are data in `limits`), not paused, and either a human is connected or that Pact has a valid away-consent. Presence is not detected yet (the wiring passes none), so today only a consent turns a Pact on.

## Away consent (minimal; the UI is TASK-168/169)

`Controller.Grant(pact, typed, difficulty)`: the player types the Pact's name (case and spacing aside); the record (pact, rank, difficulty, time, typed text) is stored in the state file. It ends at `Revoke`, when the Pact leaves the set or changes rank (`Select`), when the difficulty changes, and after `limits.consent_days` (7, U). V in `TestTithePresentConsentAndSafeZones`. No route or page calls it yet: until `pactui` the only way to choose Pacts or give a consent is the state file (U).

## Off in gates and measured runs

`pact.Off`: `D2BOT_PACTS=off`, or `D2BOT_RUN` set (the runstart protocol of gates, soaks and baselines sets it, AGENTS.md CRUTCH RULE). `startPacts` then builds nothing (V `TestOffInGatesAndMeasuredRuns`, `TestStartPactsOffInGatesAndMeasuredRuns`). A process with no `pacts-state.json` also does nothing.

## Return screen

Section "Pacts" (registered by `cmd/d2bot/pactwire.go`): one line per chosen Pact, "Name (rank r): effect line [active | why not]", the effect line from the catalogue. Not the before/after counter cards of the policy section (U: those need the change record this path deliberately skips).

## Not done

- No tally multiplier, no unlock events, no mastery counting, no low-life floor, no Hardcore, no Nightmare Pact, no `/players` (other cards).
- The bot-side wiring (`cmd/d2bot/pactwire.go`, one call in `startLadder`) is not run live. The sheet refuses rows for classes the decide catalogue lacks (only sorceress and paladin today), so for the other five classes a row-based Pact reports an error once; knob-only Pacts work.
