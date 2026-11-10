---
id: TASK-19
title: "decision-point-store: learn-by-doing store: answers, streaks, rules, fading thoughts"
status: Deferred
assignee:
  - credit-runner
created_date: '2026-11-05T00:00:00Z'
updated_date: '2026-11-10T16:54:00Z'
labels:
  - bot
  - feature
  - ladder
  - T0
  - parked
dependencies: []
priority: high
ordinal: 19000
card: "decision-point-store"
owns:
  - "internal/decide/**"
  - "docs/decision-points.md"
  - "docs/research/decision-point-store-spec.md"
  - "docs/research/decision-point-sme-review.md"
exclusive: []
branch: "agent/decision-point-store"
agent: "decision-point-store"
result: "dpgaps (offline tested, not live): store behaviours (banked skips, LateAnswer, confirm blocks asks), catalogue v2 with migration, default-parity 1168 match / 0 mismatch / 192 band-interior splits in 4 points (gate #4 stays open), public Profiles, Thought and AskWithDefault API; see docs/research/decision-point-store-notes.md"
live: ""
merge_hash: ""
overlap: "New package internal/decide; no overlap with open cards. The call sites in internal/bot, internal/town, internal/combat belong to t0-slice."
source: "grill 0005 outcome (docs/grills/0005-automation-ladder/outcome.md), ADR 0016 item 24"
contracts: ["bot-loop"]
next: "tier order: after T4/Gate A cards per ADR 0016"
---

## Description

<!-- SECTION:DESCRIPTION:BEGIN -->
Size: L

Build the decision-point store of docs/research/decision-point-store-spec.md (already updated to ADR 0016 with fading thoughts, the SME's kit field and bands). Scope: the catalogue as data (T0 to T2 points, fixed risk class and fixed answers, a built-in 'not sure'), the context key and band table, the append-only answer log, the streak algorithm (N 2/3/5 by risk, reset on a different answer, a faded thought neither counts nor resets, confirm on risky points), the rules list with revoke one or all and no self-revoking rule, per-profile persistence beside the goals state file (atomic write, version, refuse newer), all offline. Also check that a rule file can express today's defaults: spec section 5.1 maps the Tuning knobs (docs/knobs.md) to default rules and lists the gaps (life band edges 55 and 35, the lethal-pack list, a per-area retreat counter, a stash-full default, a run-time measure); this card closes or documents each and ticks the open gate of ADR 0016. Build-specific data (class mana band, plan-keyed points) lives in the class profile and plan tables, not in Go; the store must error on an unknown plan or class. Not in this card: wiring into combat (t0-slice), the screen (watch-screen), the altar. Order note: tier order: after T4/Gate A cards per ADR 0016 (the four T4 and Gate A cards m3-act1run, m3-andariel, m3-merc, gameexe-retire-soak wait their turn). Ladder work is product work, outside the tooling freeze (ADR 0016 item 3). Tags: V read in repo, U unverified.

Done when: Failing tests first: streak table (N by risk, reset on a different answer, skip, new context is a new...; Replay test: the same answer log gives the same rules; atomic write and refused newer version like...; Two builds per the build-flexibility rule: the default Sorceress and one other class or build; no point id... (and 2 more criteria below)
<!-- SECTION:DESCRIPTION:END -->

## Acceptance Criteria
<!-- AC:BEGIN -->
- [x] #1 Failing tests first: streak table (N by risk, reset on a different answer, skip, new context is a new streak), fade (a faded thought neither counts nor resets), 'not sure' (no count, no reset), confirm (risky: no resets the streak, a faded confirm leaves it unlearned), revoke one and all, no self-revoking rule
- [x] #2 Replay test: the same answer log gives the same rules; atomic write and refused newer version like internal/goals/store_test.go
- [x] #3 Two builds per the build-flexibility rule: the default Sorceress and one other class or build; no point id or band names a class; an unknown plan or class returns an error naming it
- [ ] #4 Rule-file test: the T0 to T2 default rule file reproduces the untrained default of spec section 5.1 on the listed contexts, gaps listed; the ADR 0016 gate 'a rule file can express today's defaults' is ticked or its remaining gaps named
- [ ] #5 docs/decision-points.md: catalogue, bands and defaults as shipped data; go vet and go test -p 1 of the package; leakcheck 0; contracts and backlog checks clean
<!-- AC:END -->

## Implementation Notes

<!-- SECTION:NOTES:BEGIN -->
Offline slice built in internal/decide (Pro job dpstore; notes in docs/research/decision-point-store-notes.md). Tests prove AC 1 to 3 (streaks, fades, not sure, confirm, revoke, replay, persistence, two classes, errors naming the culprit, 5 answer profiles, random-script invariants). AC 4: the default rule file covers 9 points fully, 4 partly, 10 not; gaps named in the notes, so the ADR 0016 gate stays open. AC 5: docs/decision-points.md is not written yet. Stage-2 urgency is only an interface. Nothing wired into combat; no live proof yet, so status stays In Progress until the T0 slice proves it live. Not verified live.

Catalogue v3 (job dpsplits): default parity closed for t0.retreat.drink, t0.unique.fight and t2.mercenary, proven by TestDefaultParityNumericPoints which calls resource.PotionPolicy and AbortFight (and the minEngageLifePct and mercHireMinGold knob values) at every integer life percent and gold 0..3000, both builds: 0 mismatches, 0 band-interior splits. t0.area.toohard clvl 14 (72 contexts) stays open: adding merc takes the key from 216 to 432 contexts, so it waits for an owner decision (drop levelgap or accept); the ADR 0016 gate stays open until then. Not verified live.

Parked in the 2026-11-10 true-up: ADR 0017 D2 replaced the fade with the queue; re-scope to a question queue for the return screen or keep parked.
<!-- SECTION:NOTES:END -->
