---
id: TASK-159
title: 'unlockrisk: risk meter'
status: Done
assignee:
  - credit-runner
created_date: '2026-11-08T03:16:00Z'
updated_date: '2026-11-10T16:54:00Z'
labels:
  - watch
  - feature
milestone: m-4
dependencies:
  - TASK-153
priority: low
ordinal: 137000
card: "unlockrisk"
owns:
  - "internal/watch/unlockrisk.go"
  - "internal/watch/unlockrisk_test.go"
  - "internal/watch/risk.json"
  - "internal/watch/funwire.go"
  - "internal/watch/testdata/risk-*.json"
  - "data/unlocks/ladder.json"
  - "docs/unlock-risk.md"
exclusive: []
contracts: []
result: "Offline proof 2026-11-08: go test ./internal/watch ./internal/unlock and ./cmd/d2bot (-run RiskMeterWired) pass; verdict against the bot's guards, two classes, unknown class/zone errors (TestRiskVerdictAgainstTheGuards, TestRiskUnlockByNearDeathGoldenTwoClasses, TestRiskUnknownClassAndZoneAreErrors, TestRiskMeterWiredToTheBotFacts). Fixtures only; NOT verified live. No viewer panel is built (route only). Stays open: AC #5 (live run, thresholds in risk.json are U) unticked."
---

## Description

<!-- SECTION:DESCRIPTION:BEGIN -->
Size: M

Unlock: first near-death survived; ready / marginal / turn back before a boss. Not on the two-week playable path. Design note: docs/research/unlockables.md (a brainstorm, not a decision). Technical check: the meter must use the same numbers the bot uses (potions, life ratio, resists vs the zone).
<!-- SECTION:DESCRIPTION:END -->

## Acceptance Criteria
<!-- AC:BEGIN -->
- [x] #1 ladder row risk opens on near_death_survived (life under 25 percent then back to 50), offline (TestRiskUnlockByNearDeathGoldenTwoClasses)
- [x] #2 verdict ready / marginal / turn back uses the bot's own numbers: guards, potions, life ratio, level, resist gap (TestRiskVerdictAgainstTheGuards, TestRiskVerdictLifeLevelResistAndCloseCalls)
- [x] #3 GET /api/risk returns watch.risk/1, teaser only while locked; unknown class, zone or missing table row is an error naming it (TestRiskUnknownClassAndZoneAreErrors, TestRiskTableErrors, TestRiskMissingRowIsAnError)
- [x] #4 cmd/d2bot wiring: Runner.RiskFacts feeds the meter and Fun.Settle opens the near-death moment (TestRiskMeterWiredToTheBotFacts, TASK-185); ResistGap is still always 0
- [ ] #5 LIVE check on a real run: verdict matches what the bot does before a boss, thresholds checked (not run; the live panels check is pending)
<!-- AC:END -->

## Implementation Notes

<!-- SECTION:NOTES:BEGIN -->
Closed in the 2026-11-10 true-up (docs/research/project-review-2026-11-10.md section 3.1): merged, offline acceptance proven; open live checks moved to docs/live-checks.md (TASK-253).
<!-- SECTION:NOTES:END -->
