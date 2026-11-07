---
id: TASK-121
title: 'updrelease: signed app release manifest, d2sign tool and release workflow'
status: Proposed
assignee:
  - credit-runner
created_date: '2026-11-07T23:28:00Z'
labels:
  - container
  - feature
  - tester
milestone: m-1
dependencies:
  - TASK-124
  - TASK-120
  - TASK-25
priority: high
ordinal: 114000
card: "updrelease"
owns:
  - "cmd/d2sign"
  - ".github/workflows/app-release.yml"
  - "docs/self-update-release.md"
exclusive: []
contracts: []
---

## Description

<!-- SECTION:DESCRIPTION:BEGIN -->
Size: M

Tier 2 part 1 of docs/research/self-update-design.md sections 3.2, 3.3. Needs the key custody decision (updkey).
<!-- SECTION:DESCRIPTION:END -->

## Acceptance Criteria
<!-- AC:BEGIN -->
- [ ] #1 MANIFEST.json schema and asset layout written as a contract-style doc; cmd/d2sign (stdlib ed25519) keygen, sign, verify with a review summary; tests
- [ ] #2 app-release.yml builds assets from a tag and creates a DRAFT release; signing stays manual; nothing run or pushed
<!-- AC:END -->
