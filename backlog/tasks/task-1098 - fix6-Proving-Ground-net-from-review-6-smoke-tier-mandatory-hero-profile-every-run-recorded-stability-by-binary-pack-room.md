---
id: TASK-1098
title: 'fix6: Proving Ground net from review 6: smoke tier, mandatory hero profile, every run recorded, stability by binary, pack room'
status: In Progress
assignee:
  - credit-runner
created_date: '2026-11-28T03:48:00Z'
labels:
  - bot
  - bug
dependencies: []
priority: high
ordinal: 524000
card: "fix6"
lane: next
owns:
  - "cmd/d2scenario"
  - "internal/scenario"
  - "tools/provingground"
  - "tools/ci/check.sh"
  - "cmd/stepcat"
  - "docs/process/proving-ground.md"
  - "docs/process/proving-ground-status.md"
  - "docs/probes/pg-ops-handoff-2026-11-28.md"
  - "data/stepgrounds/steps.json"
exclusive: []
contracts: []
result: "Not verified live (no lab run). Smoke tier (no game) with a CI step, pre.hero_profile mandatory in all scenario files with per-profile setup golden files, every run recorded with results-against-ledger exit 4, stability by bot binary hash with BLOCKED as a page state, belt-refill pack fixed and an overfull pack refused at validate. Verified by unit tests, d2scenario validate and smoke. Nothing build-specific: the smoke rows and tests use Paladin, Sorceress and (stability) two classes; no skill id or class check added. Ops items in docs/probes/pg-ops-handoff-2026-11-28.md."
---

## Description

<!-- SECTION:DESCRIPTION:BEGIN -->
Size: M

Why: review 6 (docs/design/fable-review-2026-11-28/06-proving-ground.md) found the 2026-11-27 break went unseen. Done when the five fixes are in with tests.
<!-- SECTION:DESCRIPTION:END -->

## Acceptance Criteria
<!-- AC:BEGIN -->
- [x] #1 d2scenario smoke: static checks plus four preps on a fake d2d, a test and a step in tools/ci/check.sh
- [x] #2 pre.hero_profile is mandatory (validate rejects a missing one), every scenario file migrated, setup golden files per profile with a test
- [x] #3 every run is recorded: a ledger refusal is spooled, run prints results N ledger M and exits non-zero when they differ
- [x] #4 stability: agent branches count, BLOCKED is a visible state and excluded from stability, the streak keys on the bot binary hash
- [x] #5 town.plan.belt-refill-from-pack no longer overfills the pack and validate rejects an overfull pack
- [ ] #6 live: one T2 run on the lab d2d records one ledger line per result
<!-- AC:END -->
