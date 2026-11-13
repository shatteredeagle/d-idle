---
id: TASK-393
title: 'cut011: friends release 0.1.1 built, verified and staged (not published)'
status: Proposed
assignee:
  - credit-runner
created_date: '2026-11-13T02:12:00Z'
labels:
  - container
  - chore
dependencies: []
priority: medium
ordinal: 246000
card: "cut011"
result: "Friends beta 0.1.1 built from main edc25c56 and staged, NOT published. Changes for friends: no more warp bounce loops, gold/loot/potion handling, belt refill, Den memory, level plan data, Paladin Sacrifice (ON, guarded, not verified live), map fog, Skills page, Gear Sets with drag and drop, safe-spot check before updates. d2d unchanged (fork acadbe47f989, same binary). Pack sha256 7e56b482703c8db09ad350478dcd9ccf424f782ae554f70c7ef64dc303a91567. Verified live on the worker host: checklist 7/7, D1/D2, 10 minute health run, update 0.1.0 to 0.1.1 with a fighting hero on the production trust path, safe-spot refusal, bad-release rollback. Details docs/releases/0.1.1.md."
result: "Friends beta 0.1.1 built, verified and staged, NOT published or sent. Cut from main edc25c56bc32 (CI green). Pack sha256 7e56b482703c8db09ad350478dcd9ccf424f782ae554f70c7ef64dc303a91567, checklist 7/7, signed beta set app-v0.1.1 seq 2, update 0.1.0 to 0.1.1 and rollback of a bad 0.1.2 verified live on the production trust path, 10 minute health run clean. For friends: fewer warp bounce loops, gold and potion handling, Paladin Sacrifice (on, guarded, not verified live), map fog, Skills page, Gear Sets, safer self-update. d2d unchanged (art-clean fork commit). Publish: bash ~/code/d2lod-ops/tools/publish-release.sh 0.1.1."
owns:
  - "docs/releases/0.1.1.md"
  - "docs/friend-checklist-0.1.1.md"
exclusive: []
contracts: []
---

## Description

<!-- SECTION:DESCRIPTION:BEGIN -->
Size: M

Why: TODO. Done when: TODO.
<!-- SECTION:DESCRIPTION:END -->

## Acceptance Criteria
<!-- AC:BEGIN -->
- [x] #1 Image, pack and signed set 0.1.1 built from main edc25c56 and staged
- [x] #2 Checklist 7/7, D1/D2, health run, update 0.1.0 to 0.1.1 and rollback drills
- [x] #3 docs/releases/0.1.1.md written
<!-- AC:END -->
