---
id: TASK-1028
title: 'actpanel: Act map never blank, status blip kept, bfcache rewake'
status: Done
assignee:
  - credit-runner
created_date: '2026-11-27T01:52:00Z'
updated_date: '2026-11-27T02:03:00Z'
labels:
  - watch
  - bug
dependencies: []
priority: medium
ordinal: 496000
card: "actpanel"
lane: next
owns:
  - "internal/watch/web/panels/zone-map"
  - "internal/watch/web/panels/statusline"
  - "internal/watch/web/shell.js"
  - "internal/watch/testdata/zonemap"
  - "internal/watch/testdata/glance_dom.js"
  - "internal/watch/testdata/shell_sched.js"
result: "Not verified live. Root cause of the stalled Act map and NO SIGNAL not found offline (the panels render with fresh and advanced data); these three changes remove the blank box and the blip. Vault slowness not reproduced in the offline benchmark."
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
- [x] #1 Zone map shows a sentence, never a blank box, when its first fetch fails or a draw throws
- [x] #2 Statusline keeps its sentence over one failed poll
- [x] #3 A page restored from the bfcache polls again at once
<!-- AC:END -->
