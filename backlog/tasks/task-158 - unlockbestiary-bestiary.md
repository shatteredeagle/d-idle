---
id: TASK-158
title: 'unlockbestiary: bestiary'
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
ordinal: 136000
card: "unlockbestiary"
owns:
  - "internal/watch/unlockbestiary.go"
  - "internal/watch/unlockbestiary_test.go"
  - "internal/watch/funwire.go"
  - "internal/watch/web/"
  - "internal/watch/testdata/bestiary-*.json"
  - "data/bestiary/"
  - "data/unlocks/ladder.json"
  - "docs/unlock-bestiary.md"
exclusive: []
contracts: []
result: "Offline proof 2026-11-08: go test ./internal/watch ./internal/unlock and ./cmd/d2bot (-run BestiaryWired|BestiaryWithout) pass; golden payloads at 10 and 50 kills, spoiler rule, replay, restart persistence (TestBestiaryGoldenTwoClasses, TestBestiarySpoilerRule, TestBestiaryReplayAndUnlisted, TestBestiaryWiredWithTableAndRestart). Fixtures only; NOT verified live or in a browser. Stays open: AC #5 (panel in a browser) and #6 (live run) unticked; kind names unchecked against the real monstats (U)."
---

## Description

<!-- SECTION:DESCRIPTION:BEGIN -->
Size: M

Unlock: 10 kills of a kind opens its page, 50 kills show resist details; silhouettes for unmet kinds. Not on the two-week playable path. Design note: docs/research/unlockables.md (a brainstorm, not a decision). Technical check: do not tie to the Cain rescue (late for a Sorceress); scouting shows categories, not spoilers.
<!-- SECTION:DESCRIPTION:END -->

## Acceptance Criteria
<!-- AC:BEGIN -->
- [x] #1 ladder rows bestiary (10 kills of any kind) and bestiary_resists (50 kills) exist and open on their trigger, offline (TestBestiaryGoldenTwoClasses, Sorceress and Paladin)
- [x] #2 GET /api/bestiary returns watch.bestiary/1 with silhouette, met, card and full states by that kind's own kill count (TestBestiaryGoldenTwoClasses, TestBestiaryNoTablesAndShippedNotes)
- [x] #3 spoiler rule: no resist or immunity field before the 50th kill of that kind and row bestiary_resists open (TestBestiarySpoilerRule on the raw JSON); replay never double-counts (TestBestiaryReplayAndUnlisted)
- [x] #4 cmd/d2bot wiring: monster table through FunConfig.Monsters and kill counts persisted across a restart (TestBestiaryWiredWithTableAndRestart, TestBestiaryWithoutTablesIsFlavourOnly; docs/unlock-ladder.md wiring table, TASK-185)
- [ ] #5 panel 'Cain's notes' renders correctly in a real browser (no browser check run yet)
- [ ] #6 LIVE check on a real run: kills accumulate and cards open, kind names match the real monstats (not run; the live panels check is pending)
<!-- AC:END -->

## Implementation Notes

<!-- SECTION:NOTES:BEGIN -->
Closed in the 2026-11-10 true-up (docs/research/project-review-2026-11-10.md section 3.1): merged, offline acceptance proven; open live checks moved to docs/live-checks.md (TASK-253).
<!-- SECTION:NOTES:END -->
