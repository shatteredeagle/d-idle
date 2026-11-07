---
id: TASK-124
title: 'updkey: signing key custody and rotation setup'
status: Proposed
assignee:
  - user
created_date: '2026-11-07T23:28:00Z'
labels:
  - ops
  - chore
  - tester
milestone: m-1
dependencies: []
priority: high
ordinal: 117000
card: "updkey"
owns:
  - "docs/self-update-keys.md"
exclusive: []
contracts: []
---

## Description

<!-- SECTION:DESCRIPTION:BEGIN -->
Size: S

Decision card for docs/research/self-update-design.md section 8. Blocks updrelease signing and Tier 2.
<!-- SECTION:DESCRIPTION:END -->

## Acceptance Criteria
<!-- AC:BEGIN -->
- [ ] #1 user decides key custody, signing tool and repo owner (design doc section 8); key pair generated offline, public key handed to the image build, backup place named (no secret in the repo)
<!-- AC:END -->
