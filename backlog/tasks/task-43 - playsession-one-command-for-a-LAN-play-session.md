---
id: TASK-43
title: 'playsession: one command for a LAN play session'
status: Proposed
assignee: []
created_date: '2026-11-06T16:02:00Z'
labels:
  - T0
  - tooling
dependencies: []
priority: medium
ordinal: 43000
card: "playsession"
owns:
  - "tools/play-session.sh"
  - "docs/play-session.md"
exclusive: []
contracts: []
---

## Description

<!-- SECTION:DESCRIPTION:BEGIN -->
The 'level 1 friend on the LAN' item: bring up a hidden instance and start the bot with -ladder -watch -watch-video so the user can watch and answer from another device. The instance bring-up goes through tools/ops.sh (private stubs).
<!-- SECTION:DESCRIPTION:END -->

## Acceptance Criteria
<!-- AC:BEGIN -->
- [ ] #1 tools/play-session.sh: one command; prints the watch URL and token file path (never the token); refuses without the ops checkout with a one-line message
- [ ] #2 A dry-run mode prints the exact commands; a shell test covers dry-run and the refusal; docs/play-session.md has stop and cleanup
- [ ] #3 Says verified live or not; any live check is a ready-to-run checklist
<!-- AC:END -->
