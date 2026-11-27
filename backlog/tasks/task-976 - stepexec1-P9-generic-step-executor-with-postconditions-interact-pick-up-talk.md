---
id: TASK-976
title: >-
  stepexec1: P9 generic step executor with postconditions (interact, pick up,
  talk
status: Done
assignee:
  - credit-runner
created_date: '2026-11-26T03:55:00Z'
updated_date: '2026-11-26T14:54:00Z'
labels:
  - bot
  - feature
dependencies: []
priority: medium
ordinal: 464000
card: "stepexec1"
branch: "agent/stepexec2"
owns:
  - "docs/design/stepexec-p9.md"
  - "internal/questspec/"
  - "data/questspec/"
  - "internal/bot/stepexec.go"
  - "internal/bot/stepexec_test.go"
  - "internal/bot/stepexec_act2_test.go"
  - "internal/bot/task.go"
  - "internal/env/tuning_quests.go"
  - "tools/provingground/scenarios/cain.json"
  - "tools/provingground/scenarios/stepexec-act2.json"
  - "tools/provingground/scenarios/tiers.json"
  - "data/darklaunch.json"
  - "docs/knobs.md"
exclusive: []
contracts: []
result: "Engine questspec.RunStep plus exec rows as data; Cain scroll take_scroll runs on it behind quests.stepExec (default 0, dark-launch row, scenario cain.scroll.stepexec). Act 2 chunk (2026-11-27): Act 2 quest chains as data, new transmute verb; the Horadric Staff transmute and placing the Staff in the orifice run on the executor with posts read from the pack and the quest reader; the Cain hand-in keeps the old path (no game fact proves it). Unit tests only (Sorceress and Paladin, must-fail cases); NOT verified live or on the lab."
---

## Description

<!-- SECTION:DESCRIPTION:BEGIN -->
Size: L

Cause classes killed: H (per-quest special case, 45) and F (a fact the backend does not give). Roadmap row P9 (docs/research/fable-logic-review-2026-11-25.md 7.1; ADR 0021, 0022). One executor: approach, act, wait for the declared fact, bounded retries, progress is not failure; identity by id; new chunks may use ONLY it. Replaces cainNodes-style handlers one by one, each behind the scenario of its chunk. Declaration side exists in internal/questspec. Unblocks the Act II epic card (epic-act2: Duriel place_staff needs a verb path) and later act epics. Depends on P2. Risk: medium; one chunk per card. Enable criteria per chunk: its Proving Ground scenario green 2 of 3 runs with the executor on; old handler deleted after a 24 h watch with auto-revert.
<!-- SECTION:DESCRIPTION:END -->

## Acceptance Criteria
<!-- AC:BEGIN -->
- [x] #1 Executor skeleton plus the Cain scroll chunk ported behind a knob with a ledger row
- [x] #2 Rule in the card doc: no new chunk without the executor
- [x] #3 Act 2 chunk: Horadric Staff transmute and orifice placement as data rows behind quests.stepExec, tests on two classes with must-fail cases
- [ ] #4 Scenarios drill.act2.assemble.stepexec and drill.act2.duriel.stepexec green 2 of 3 runs each on the lab
<!-- AC:END -->
