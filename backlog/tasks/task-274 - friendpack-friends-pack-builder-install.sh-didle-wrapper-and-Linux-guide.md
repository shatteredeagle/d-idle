---
id: TASK-274
title: 'friendpack: friends pack builder, install.sh, didle wrapper and Linux guide'
status: Done
assignee:
  - credit-runner
created_date: '2026-11-10T22:17:00Z'
updated_date: '2026-11-10T22:45:00Z'
labels:
  - container
  - feature
  - tester
dependencies: []
priority: medium
ordinal: 200000
card: "friendpack"
owns:
  - "tools/friendpack"
  - "docs/friend-guide.md"
exclusive: []
contracts: []
result: "Built and tested: go test ./tools/friendpack (file list, reproducible, art/key refusal, install.sh with a shim docker); install.sh, ./didle status|report|stop run once on the worker host in a scratch project with fake game files and a dev image (page URL printed, container up, cleaned). NOT verified: ./didle update against a real feed, a full wizard with real MPQs, Windows. Pack signature not built (d2sign signs manifests only)."
---

## Description

<!-- SECTION:DESCRIPTION:BEGIN -->
Size: M

PR 2 of docs/research/friend-build.md section 10 (user decision 2026-11-10). Done when merged.
<!-- SECTION:DESCRIPTION:END -->

## Acceptance Criteria
<!-- AC:BEGIN -->
- [x] #1 build.sh writes a reproducible pack with image, sources, licences, install.sh, didle, guide and checksums; refuses game, art and key files (test)
- [x] #2 install.sh checks, prints the exact page URL and token, own-MPQs read-only; tested with a shim docker and once on the worker host in a scratch project
- [x] #3 docs/friend-guide.md with the licence note; nothing published, no real key used
<!-- AC:END -->
