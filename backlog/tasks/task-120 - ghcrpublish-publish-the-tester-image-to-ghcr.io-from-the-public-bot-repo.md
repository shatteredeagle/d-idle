---
id: TASK-120
title: 'ghcrpublish: publish the tester image to ghcr.io from the public bot repo'
status: Done
assignee:
  - credit-runner
created_date: '2026-11-07T23:28:00Z'
updated_date: '2026-11-10T16:54:00Z'
labels:
  - container
  - feature
  - tester
milestone: m-1
dependencies:
  - TASK-25
  - TASK-31
priority: high
ordinal: 113000
card: "ghcrpublish"
owns:
  - ".github/workflows/tester-image.yml"
  - "docs/registry-publish.md"
  - "deploy/d2d-container/compose.friends.yml"
  - "docs/tester-registry.md"
  - "deploy/d2d-container/compose.tester.yml"
  - "deploy/d2d-container/context.sh"
  - "deploy/d2d-container/tester_image_test.go"
exclusive: []
contracts: []
result: "PREP only (2026-11-08, agent/ghcrprep): inert workflow_dispatch workflow + environment gate written, compose.tester.yml takes D2_IMAGE, static tests, worker-host dry run (smoke PASS, cold build numbers in docs/tester-registry.md). Nothing pushed or run on GitHub; AC 1 and 3 partly (no live registry, no scan on a hosted runner), AC 2 uses compose.tester.yml instead of compose.friends.yml; task stays open for the first publish. Status sync 2026-11-08 (stays open): PREP only, registry publish not done."
---

## Description

<!-- SECTION:DESCRIPTION:BEGIN -->
Size: M

Tier 1 of docs/research/self-update-design.md (user decision 2026-11-07). Friends update with docker compose pull and up -d. Publishing needs the user's go (first outside copy, package visibility).
<!-- SECTION:DESCRIPTION:END -->

## Acceptance Criteria
<!-- AC:BEGIN -->
- [ ] #1 workflow file written (tag tester-v*, workflow_dispatch dry run, amd64 only, tags X.Y.Z latest d2d-<sha> sha-<sha>, labels incl. source and fork commit, no-Blizzard-data scan of the built image, GITHUB_TOKEN only); NOTHING pushed or run
- [ ] #2 compose.friends.yml with image tag variable, explicit pull_policy, optional Watchtower label documented with its risks; docs/registry-publish.md holds the publishing plan, the one-time package-public step and the release body template with the GPL source offer
- [ ] #3 dry run on the worker host proves the public-source build (fork tag checkout) gives a working image (TASK-25 smoke); result says verified live or not
<!-- AC:END -->

## Implementation Notes

<!-- SECTION:NOTES:BEGIN -->
Closed in the 2026-11-10 true-up (docs/research/project-review-2026-11-10.md section 3.1): merged, offline acceptance proven; open live checks moved to docs/live-checks.md (TASK-253).
<!-- SECTION:NOTES:END -->
