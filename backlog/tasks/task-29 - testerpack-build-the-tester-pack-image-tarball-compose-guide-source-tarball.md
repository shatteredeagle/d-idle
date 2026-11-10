---
id: TASK-29
title: 'testerpack: install path, tester pack with guide and first-stranger checklist'
status: Proposed
assignee:
  - credit-runner
created_date: '2026-11-06T14:27:00Z'
updated_date: '2026-11-10T17:11:00Z'
labels:
  - container
  - feature
  - tester
milestone: m-1
dependencies:
  - TASK-31
  - TASK-28
priority: high
ordinal: 29000
card: "testerpack"
owns:
  - "tools/tester-pack.sh"
  - "docs/tester-guide.md"
exclusive: []
contracts: []
result: "Re-scoped 2026-11-10 (trueup2): install path, absorbs the tester guide and first-stranger checklist of TASK-26; docs/tester-guide.md added to owns. Not built, not verified live."
---

## Description

<!-- SECTION:DESCRIPTION:BEGIN -->
Size: M
Why: beta exit criterion 3 (docs/research/project-review-2026-11-10.md section 5.1) needs one install a Docker-literate friend can follow. This card absorbs TASK-26 (testerdocs): the tester guide and the first-stranger checklist ship inside the pack. Building the pack locally is not publishing it; no copy leaves the machine without the user's go.
Done when: tools/tester-pack.sh builds the pack (image save, compose.tester.yml, guide, source tarballs, licence files, sha256sums; no game file, save or token), and docs/tester-guide.md carries the install route, troubleshooting, known weaknesses, what to send back and the first-stranger checklist. Depends on TASK-31 and TASK-28. The version line in the watch header is TASK-47's (not owned here).
<!-- SECTION:DESCRIPTION:END -->

## Acceptance Criteria
<!-- AC:BEGIN -->
- [ ] #1 tools/tester-pack.sh builds a directory with the docker save gz of the d2d image, compose.tester.yml, the guide, the d.idle fork source tarball (pinned ref plus VERSION) with the vendor inputs, the d2lod-bot source tarball of the image commit, the licence files and sha256sums; contains no game file, save or token; a test checks the file list
- [ ] #2 docs/tester-guide.md: prerequisites, MPQ files to bring, compose route, token, wizard, watch URL and ports, what to expect, stop and update, troubleshooting, known weaknesses, what to send back (absorbed from TASK-26)
- [ ] #3 first-stranger checklist with pass criteria (compose, wizard, card, answer, order, video, restart, bundle, update) in the guide; no private hostnames or paths
<!-- AC:END -->
