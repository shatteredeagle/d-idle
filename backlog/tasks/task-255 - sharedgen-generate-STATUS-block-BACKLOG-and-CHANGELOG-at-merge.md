---
id: TASK-255
title: 'sharedgen: generate STATUS block, BACKLOG and CHANGELOG at merge'
status: Done
assignee:
  - credit-runner
created_date: '2026-11-10T16:55:00Z'
updated_date: '2026-11-11T16:37:00Z'
labels:
  - ops
  - chore
milestone: m-4
dependencies: []
priority: medium
ordinal: 192000
card: "sharedgen"
owns:
  - "tools/sharedgen"
  - "tools/ci"
  - "AGENTS.md"
  - "docs/agents/brief-template.md"
  - "docs/backlog-howto.md"
  - "docs/START-HERE.md"
exclusive: []
contracts: []
branch: "agent/sharedgen"
result: "Generator, merge.sh hook, dry run, rehearsal on a scratch bare repo and docs done offline (ops repo commits carry merge.sh, mergeq.py, auto_done.py, docs/merging-generated.md). Not verified on the real origin until this branch merges."
---

## Description

<!-- SECTION:DESCRIPTION:BEGIN -->
Size: M

Why: five shared files were rewritten per branch (conflicts, lines to get wrong, reading cost). Done when: the merge step generates BACKLOG, CHANGELOG, the OWNERS block, the STATUS block and INDEX from the task card, queue summary and branch name; agents write none of them; the dry run of the last 30 merges is clean and byte-stable.
<!-- SECTION:DESCRIPTION:END -->

## Acceptance Criteria
<!-- AC:BEGIN -->
- [x] #1 tools/sharedgen/sharedgen.py generates BACKLOG, CHANGELOG, OWNERS block, STATUS block and INDEX; idempotent and byte-stable; offline tests pass
- [x] #2 merge.sh calls it at merge (MERGE_SHAREDGEN=0 off); legacy merge-lines files still accepted; test_merge_sh.sh case 19 passes
- [x] #3 dry run of the last 30 merges reported against the committed files
- [x] #4 AGENTS.md, brief template, backlog-howto one-pager, START-HERE tell agents not to write the shared files
- [ ] #5 live check: the first real merge through the generator (the merge of this branch) shows its generated lines on main
<!-- AC:END -->
