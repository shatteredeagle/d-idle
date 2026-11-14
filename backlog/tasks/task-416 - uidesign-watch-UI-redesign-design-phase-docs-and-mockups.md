---
id: TASK-416
title: "uiredesign: watch UI redesign, design phase (docs and mockups)"
status: In Progress
assignee:
  - user
created_date: '2026-11-14T14:26:00Z'
labels:
  - watch
  - research
dependencies: []
priority: high
ordinal: 253000
card: "uiredesign"
owns:
  - "docs/ui/redesign-2026-11-14/"
exclusive: []
contracts: []
branch: "agent/uidesign"
result: "design proposal written, not built and not verified live (2026-11-14): docs/ui/redesign-2026-11-14/ holds the audit (fixture-viewer pictures at 1440, 900, 390 px), principles, the Orders control surface (hybrid: Orders panel primary, act map as place picker; additive /api/commands and /api/glance drafts), four watch.layout/1 preset drafts with the migration table, the visual system, a clickable mockup (also a private Artifact), seven build slices and twelve open questions; no file under internal/watch changed; waits for the user's review"
---

## Description

<!-- SECTION:DESCRIPTION:BEGIN -->
Size: L

The user asked (2026-11-14) for a whole pass on the watch web UI: usability, information density, ease of use, and one unified control surface. Design first; the user reviews; workers build in slices. Done when docs/ui/redesign-2026-11-14/ holds the deliverables A to H and the user has the mockup link. No product code in this card. Consumer: the build-slice cards S1 to S7 of docs/ui/redesign-2026-11-14/G-build-plan.md (created after the user reviews).
<!-- SECTION:DESCRIPTION:END -->

## Acceptance Criteria
<!-- AC:BEGIN -->
- [x] #1 Audit: every panel and control with a KEEP/MERGE/DEMOTE/DROP verdict, pictures at 1440, 900 and 390 px from fixtures, and the command table
- [x] #2 Principles and information architecture of the default view
- [x] #3 Orders: the unified control surface, the primary-input decision with its trade-offs, the additive API draft with V/U backend support
- [x] #4 Layout presets as watch.layout/1 drafts (simple, standard as the proposed default, stream, debug) with the alias table and rollout plan
- [x] #5 Visual system: tokens, components, accessibility, motion
- [x] #6 Clickable static mockup in the repo and published as a private Artifact
- [x] #7 Build plan in mergeable slices for Sonnet workers, and the numbered open questions with a recommendation each
<!-- AC:END -->
