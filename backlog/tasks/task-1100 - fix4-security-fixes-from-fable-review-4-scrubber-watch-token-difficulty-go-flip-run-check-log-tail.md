---
id: TASK-1100
title: 'fix4: security fixes from fable review 4 (scrubber, watch token, difficulty go, flip run check, log tail)'
status: In Progress
assignee:
  - credit-runner
created_date: '2026-11-28T03:55:00Z'
labels:
  - watch
  - bug
dependencies: []
priority: medium
ordinal: 524000
card: "fix4"
lane: next
owns:
  - "deploy/d2d-container/tester-report.sh"
  - "deploy/d2d-container/tester_report_test.go"
  - "internal/watch/serve.go"
  - "internal/watch/serve_test.go"
  - "internal/watch/gate.go"
  - "internal/watch/web/shell.js"
  - "internal/watch/difficultydesk.go"
  - "internal/watch/difficultydesk_test.go"
  - "internal/diffready/"
  - "internal/bot/diffready.go"
  - "internal/bot/diffready_test.go"
  - "tools/flip/flip.py"
  - "tools/flip/test_flip.py"
  - "internal/logx/sink.go"
  - "internal/logx/sink_test.go"
  - "docs/ops/docker-volumes.md"
  - "docs/diffready.md"
exclusive: []
contracts: []
result: "scrubber, watch token redirect and headers, per-hero expiring difficulty go with audit, budget and measured-run refusal, flip container run check, NUL-safe log tail, volume procedure doc; offline tests only, NOT verified live. Build-specific: nothing (the go rule is tested for two classes; GoTTL lives in internal/diffready). Hub link fix is in the ops repo, branch agent/fix4."
---

## Description

<!-- SECTION:DESCRIPTION:BEGIN -->
Size: M

Fixes for findings 1, 3, 4, 5, 7 of docs/design/fable-review-2026-11-28/04-security-privacy.md and the volume procedure of finding 2. Done when the tests of the touched packages pass.
<!-- SECTION:DESCRIPTION:END -->

## Acceptance Criteria
<!-- AC:BEGIN -->
- [x] #1 tester report scrubber hides passcode/secret/apikey/auth values, 16+ keys, all non-loopback IPv4, IPv6, /mnt /root /Users paths, home with a space; README claims only that; fixtures both ways
- [x] #2 watch tokenGate: 303 to the clean address after a valid ?t=, no cookie for a wrong token, no-referrer and no-store, shell.js builds no ?t= address
- [x] #3 POST /api/difficulty/go: audit line, hourly budget, measured-run refusal; diffgo.json carries hero and a 7-day end, a foreign or ended go is ignored with a log line (two classes)
- [x] #4 flip.py reads D2BOT_RUN from the container and refuses a measured run
- [x] #5 ops log Tail survives a NUL run over 4 MiB
- [x] #6 docs/ops/docker-volumes.md: remove by name, label and back up the hero volumes, never prune -a
- [ ] #7 live check: open /watch?t=<token> in a browser on a sandbox and see the clean address; run tester-report.sh in a tester container
<!-- AC:END -->
