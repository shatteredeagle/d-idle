---
id: TASK-139
title: 'imgrepro: image build must not need the local fork build dir'
status: Proposed
assignee:
  - credit-runner
created_date: '2026-11-08T00:43:00Z'
labels:
  - container
  - chore
  - tester
milestone: m-1
dependencies: []
priority: medium
ordinal: 115000
card: "imgrepro"
owns:
  - "deploy/d2d-container/context.sh"
exclusive: []
contracts: []
---

## Description

<!-- SECTION:DESCRIPTION:BEGIN -->
Size: S

Found by TASK-25: context.sh now takes the d2d source from the published fork tag (FORK_URL), but CLI11, sentry-native and the FFmpeg 9.0.2 tarball still come from the build dir of the local fork checkout, so a clean machine cannot build the image from public inputs. Done when: the vendored inputs are fetched from their public upstream at pinned versions and hashes (or named in the fork's pack), and the image builds on a machine without the local checkout.
<!-- SECTION:DESCRIPTION:END -->

## Acceptance Criteria
<!-- AC:BEGIN -->
- [ ] #1 build from public inputs only, hashes checked
<!-- AC:END -->
