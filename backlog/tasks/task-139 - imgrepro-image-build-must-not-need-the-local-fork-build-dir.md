---
id: TASK-139
title: 'imgrepro: image build must not need the local fork build dir'
status: Done
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
  - "deploy/d2d-container/Dockerfile.d2d"
  - "deploy/d2d-container/tester_image_test.go"
  - "docs/tester-live.md"
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
- [x] #1 build from public inputs only, hashes checked
<!-- AC:END -->

result: verified live yes (2026-11-07): image built on the worker host from a clean scratch copy with no local fork checkout; CLI11 2.6.2, sentry-native 0.17.3 and FFmpeg 9.0.2 come from public archives pinned by sha256 in context.sh; 409 MB, static self-test PASS, wizard 401 without and 200 with the token, binary differs from the local-build one only in the build timestamp (25 bytes). Details docs/tester-imgrepro.md.
