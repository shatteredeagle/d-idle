---
id: TASK-120
title: 'ghcrpublish: publish the tester image to ghcr.io from the public bot repo'
status: Proposed
assignee:
  - credit-runner
created_date: '2026-11-07T23:28:00Z'
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
exclusive: []
contracts: []
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
