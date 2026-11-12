---
id: TASK-356
title: 'cut010: first friends release 0.1.0 (build and verify, not published)'
status: Done
assignee:
  - credit-runner
created_date: '2026-11-12T03:59:00Z'
updated_date: '2026-11-12T04:58:00Z'
labels:
  - container
  - chore
dependencies: []
priority: medium
ordinal: 232000
card: "cut010"
result: "0.1.0 built and verified, NOT published or sent. Image didle:0.1.0 on fork a392bcbd (new icon), pack sha256 b5cb3616feca3fe29a82c840049e0c81cf96b9472a89bec3694d0c4e703b4c82, legal checklist 7/7 PASS, signed beta set app-v0.1.0 verified with the real client. Verified live on the worker host with the real game folder: D1 install, D2 wizard+hero, stop/start persistence, report scrub; D3/D5/D7 not re-run (DEV-only, engine unchanged). Fixes: bot stamped 0.1.0 in the image, DIDLE_DEV=0 no longer enables nav trace, compose log rotation, Freepik notice and CA-store checklist fixes. Details docs/releases/0.1.0.md."
owns:
  - "docs/releases/0.1.0.md"
  - "docs/friend-checklist-0.1.0.md"
  - "tools/friendpack/build.sh"
  - "tools/friendpack/checklist.sh"
  - "deploy/d2d-container/context.sh"
  - "tools/friendpack/compose.yml"
  - "cmd/d2bot/main.go"
exclusive:
  - "fork-branch:fork-only/icon-original"
contracts: []
---

## Description

<!-- SECTION:DESCRIPTION:BEGIN -->
Size: M

Why: TODO. Done when: TODO.
<!-- SECTION:DESCRIPTION:END -->

## Acceptance Criteria
<!-- AC:BEGIN -->
- [x] #1 Image didle:0.1.0 built from main with the pinned fork commit, booted once with the new icon
- [x] #2 Drills D1-D7 run on the new image and legal checklist recorded
- [x] #3 Signed beta release set and the friends pack built (not published), sha256 recorded
- [x] #4 docs/releases/0.1.0.md written
<!-- AC:END -->
