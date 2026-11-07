---
id: TASK-67
title: 'g2gfixmouth: nav mouth gate no safe click loop at repeated spots'
status: Done
assignee: []
created_date: '2026-11-07T07:05:00Z'
updated_date: '2026-11-07T15:14:00Z'
labels:
  - bot
  - bug
  - gate
milestone: m-0
dependencies: []
priority: high
ordinal: 65000
card: "g2gfixmouth"
owns:
  - "internal/nav/mouthgate.go"
  - "internal/nav/nav.go"
  - "internal/nav/g2gmouth_test.go"
  - "cmd/d2drill/e1_mouth.go"
  - "cmd/d2drill/e1_mouth_test.go"
  - "docs/probes/mouthfix-run.sh"
exclusive: []
contracts: []
---

## Description

<!-- SECTION:DESCRIPTION:BEGIN -->
Size: M

docs/research/g2g-diagnosis.md finding N1: 379/468/187 click held (no safe click) lines per seed, the same 3 spots repeated 38 to 56 times; detour fired 10/17/2 times.

Done when: inside-disc and outside-disc holds at the recorded spots resolve or escalate within 3 holds; offline test from the recorded from/to pairs (acceptance criteria below).
<!-- SECTION:DESCRIPTION:END -->

## Acceptance Criteria
<!-- AC:BEGIN -->
- [x] #1 inside-disc and outside-disc holds at the recorded spots resolve or escalate within 3 holds
- [x] #2 offline test from the recorded from/to pairs
<!-- AC:END -->

result: Escalation added (hold counter, pocket window, internal/nav/mouthgate.go hold/pocketClick): the same click refused 3 times within 120 s from inside a trigger disc (or at the rewrite cap) is rewritten to a non-approaching cell or sent as is. Offline test from the logged from/to pairs (exit and walls assumed, U). Hold of a target inside the bound from outside the disc stays a refusal. Live drill E1.mouth-pocket ran on slot 3: pocket NOT reproduced on that seed (inconclusive), so not verified live.
