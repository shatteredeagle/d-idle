---
id: TASK-150
title: 'sensesvideo: research overlay aligned to the video (projection and camera)'
status: Deferred
assignee:
  - credit-runner
created_date: '2026-11-08T03:12:00Z'
updated_date: '2026-11-08T03:49:00Z'
labels:
  - watch
  - research
  - parked
milestone: m-4
dependencies:
  - TASK-148
priority: low
ordinal: 133000
card: "sensesvideo"
owns:
  - "docs/research/senses-video.md"
exclusive: []
contracts: []
---

## Description

<!-- SECTION:DESCRIPTION:BEGIN -->
Size: M

Design: docs/research/info-unlocks.md. Not on the two-week playable path (returnscreen, policysheet, tallystrip, codexfirsts, playsession2); after it. Deferred: v1 is the side-panel minimap. Done when the research doc gives a verdict and a probe plan.
<!-- SECTION:DESCRIPTION:END -->

## Acceptance Criteria
<!-- AC:BEGIN -->
- [ ] #1 Source of camera centre and capture crop/scale found or ruled out (d2d tap and x11grab); isometric 2:1 projection checked against screenshots
- [ ] #2 Verdict: overlay feasible with calibration or not; error in pixels; frame delay of the stream measured
- [ ] #3 Live steps written as a ready-to-run checklist (no game used by the agent)
<!-- AC:END -->

## Implementation Notes

<!-- SECTION:NOTES:BEGIN -->
side-panel minimap is the primary design (user 2026-11-07); video overlay deferred
<!-- SECTION:NOTES:END -->
