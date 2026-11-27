---
id: TASK-542
title: 'support: send a support bundle to the updates site'
status: Done
assignee:
  - credit-runner
created_date: '2026-11-16T04:02:00Z'
updated_date: '2026-11-16T14:26:00Z'
labels:
  - watch
  - feature
dependencies: []
priority: medium
ordinal: 286000
card: "support"
owns:
  - "internal/support"
  - "internal/watch/supportapi.go"
  - "internal/watch/supportapi_test.go"
  - "internal/watch/web/supportui.js"
  - "internal/watch/web/shell.css"
  - "internal/watch/web/shell.js"
  - "internal/watch/funwire.go"
  - "internal/watch/http_view.go"
  - "internal/watch/supportui_test.go"
  - "cmd/d2bot/supportwire_test.go"
  - "cmd/d2bot/ladder.go"
  - "cmd/d2bot/live.go"
  - "cmd/d2bot/d2dbackend.go"
  - "tools/uishots"
  - "tools/uishots.sh"
  - "tools/friendpack/compose.yml"
  - "tools/friendpack/install.sh"
  - "docs/friend-guide.md"
  - "docs/process/leakcheck-allow.txt"
  - "cmd/d2supportrecv"
  - "cmd/d2bot/supportwire.go"
  - "docs/support-bundle.md"
exclusive: []
contracts: []
result: "Offline only, NOT verified live: bundle builder + redaction + fail-closed scan (internal/support), receiver cmd/d2supportrecv (auth, rate limits, quota, traversal tests), watch routes, UI sheet (node test), fit pages (browser run not done), docs/support-bundle.md with the threat model. Not deployed; deploy steps in cmd/d2supportrecv/README.md; ops fetch script written in the ops repo, uncommitted. closecards3 Remaining (live): #5 need the live game, a browser or a worker-host run; not verified here."
---

## Description

<!-- SECTION:DESCRIPTION:BEGIN -->
Size: L

Why: friends cannot easily send the owner a clean log; a button builds a redacted, scanned, previewed bundle and uploads it to the updates site (receiver on the file server) or saves it. Done when: the code, tests and docs are merged; the server deploy and the live checklist in docs/support-bundle.md are the owner's.
<!-- SECTION:DESCRIPTION:END -->

## Acceptance Criteria
<!-- AC:BEGIN -->
- [x] #1 Bundle builder, redaction, fail-closed scan, tests
- [x] #2 Receiver (cmd/d2supportrecv) with auth, quota, purge, traversal tests; README, nginx snippet, deploy steps
- [x] #3 Watch API + UI sheet + node tests + fit fixtures
- [x] #4 Docs: support-bundle.md, friend guide section, knobs
- [ ] #5 Live check: real upload on the updates site (owner, after deploy)
<!-- AC:END -->
