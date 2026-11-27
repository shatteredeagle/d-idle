---
id: TASK-426
title: 'cut012: friends release 0.1.2 (installer fix, bot fixes) built, verified, staged'
status: Proposed
assignee:
  - credit-runner
created_date: '2026-11-14T15:50:00Z'
labels:
  - container
  - chore
dependencies: []
priority: medium
ordinal: 255000
card: "cut012"
result: "Installer fix (quoted .env path, Diablo II subfolder detection, README checksum line) tested offline. 0.1.2 pack (sha256 c5146213a61b952863560f7a970a5d5d132dba887de043046349b5d28de55e14), signed set and feed tree staged on the head box, checklist 7/7, NOT published. Live drills (zip target install, 20 minute health run, update 0.1.1 to 0.1.2, rollback) NOT run: they need docker with real game files and the worker host is in use. Details docs/releases/0.1.2.md. Remaining (live): ACs 2 and 3 publish, drills, docs/releases/0.1.2.md."
owns:
  - "docs/releases/0.1.2.md"
  - "docs/friend-checklist-0.1.2.md"
  - "tools/friendpack/install.sh"
  - "tools/friendpack/didle"
  - "tools/friendpack/compose.yml"
  - "tools/friendpack/README.txt"
  - "tools/friendpack/pack_test.go"
  - "docs/friend-guide.md"
exclusive: []
contracts: []
lane: next
---

## Description

<!-- SECTION:DESCRIPTION:BEGIN -->
Size: M

Why: TODO. Done when: TODO.
<!-- SECTION:DESCRIPTION:END -->

## Acceptance Criteria
<!-- AC:BEGIN -->
- [x] #1 Installer quotes the game path, finds the Diablo II subfolder, README names the tarball checksum
- [ ] #2 Image, pack, signed set 0.1.2 built and staged; checklist 7/7; drills and 20 minute health run
- [ ] #3 docs/releases/0.1.2.md written
<!-- AC:END -->
