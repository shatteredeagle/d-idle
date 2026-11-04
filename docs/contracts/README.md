# Contracts

> status: current since OpenSpec removal stage 3 (`docs/process/openspec-removal-plan.md`); format work follows in stage 3b (`docs/process/contracts-after-openspec.md`).

A contract is a promise the bot, a judge or the lab keeps: rules and the numbers that decide a gate or a policy. It is not evidence (that is `docs/*.md` with V/S/U tags), not a decision record (`docs/adr/`) and not a work item (`backlog/`).

## Files

One file per capability, `docs/contracts/<capability>.md`. Gates: `soak-gates`, `fidelity-gate`, `death-rule`. Policies: `merc-policy`, `early-game-policy`, `act1-campaign`, `town-basics`, `combat-loop`, `bot-loop`. Interfaces: `backend-contract`, `replay`, `d2d-farm`, `dashboard-pub`. Standing rules: `lab-rules`, `process-hygiene`, `public-split`, `licence-policy`, `upstream-pr`.

`proposed/<card>.md` holds requirement text that is NOT in force yet (the delta specs of the cards that were open at the tag, written out as plain requirements). The branch that implements the work folds it into the main file and deletes the proposal.

## Form

`### Requirement: <name>` with SHALL/MUST text, then `#### Scenario: <name>` with WHEN/THEN lines. Every number carries a `Source:` (user decision with date, ADR, doc, or measurement). Never copy a number that has an owner (a knob default lives in `docs/knobs.md`): cite it.

## Changing a contract

Edit the file in the same branch as the code that changes the behaviour; no delta file, no apply step. A line whose `Source:` is a user decision changes only with the user's go, cited on the line with its date. The guard rule for `docs/adr/*`, `GLOSSARY.md` and `docs/decisions.md` is unchanged.

## History

Up to git tag `openspec-final` the contracts lived in `openspec/specs/<capability>/spec.md` and changed through delta specs in `openspec/changes/`. The reconciliation is recorded in `docs/process/spec-ledger-20261104.md` and `docs/process/spec-ledger-resolutions-20261104.md`; all old cards are indexed in `docs/archive/openspec-index.md`.
