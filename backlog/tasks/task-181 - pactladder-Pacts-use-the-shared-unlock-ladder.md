---
id: TASK-181
title: 'pactladder: Pacts use the shared unlock ladder'
status: Done
assignee:
  - credit-runner
created_date: '2026-11-08T09:11:00Z'
updated_date: '2026-11-08T10:30:00Z'
labels:
  - bot
  - chore
  - G-C
milestone: m-4
dependencies:
  - TASK-153
  - TASK-168
priority: medium
ordinal: 151000
card: "pactladder"
owns:
  - "internal/pact/"
  - "internal/unlock/"
  - "data/unlocks/ladder.json"
  - "internal/watch/pactapi.go"
  - "internal/watch/unlock_test.go"
  - "cmd/d2bot/pactwire.go"
  - "cmd/d2bot/pactwire_test.go"
  - "cmd/d2bot/funwire.go"
  - "docs/pact-ui.md"
  - "docs/unlock-ladder.md"
  - "backlog/merge-lines/agent-pactladder.txt"
exclusive: []
contracts: []
result: "Built and tested offline (unit and wiring tests, Sorceress and Paladin); not verified live or in a browser. Widens the owns of TASK-153 (internal/unlock, ladder.json, docs/unlock-ladder.md) and TASK-168 (pact, pactwire, pactapi). baal_nightmare Pact is NOT an entry: proposal undecided. Hardcore stays a separate flag."
---

## Description

<!-- SECTION:DESCRIPTION:BEGIN -->
Size: S

Why: TODO. Done when: TODO.
<!-- SECTION:DESCRIPTION:END -->

## Acceptance Criteria
<!-- AC:BEGIN -->
- [x] #1 Pact unlocks are ladder entries (Andariel kill, Baal in Normal); Hell/Hardcore not entries
- [x] #2 Controller.Unlock replaced by the ladder; silhouettes read the ladder; reveal fires through it
- [x] #3 Old pacts-state.json unlocked field migrated, unknown keys kept
- [x] #4 Away-consent grant/revoke/expiry visible on the return screen
- [x] #5 Tests: Andariel kill, Nightmare locked until Baal Normal, replay, persistence, two classes
<!-- AC:END -->
