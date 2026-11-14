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
---

## Description

<!-- SECTION:DESCRIPTION:BEGIN -->
Size: L

The user asked (2026-11-14) for a whole pass on the watch web UI: usability, information density, ease of use, and one unified control surface. Design first; the user reviews; workers build in slices. Done when docs/ui/redesign-2026-11-14/ holds the deliverables A to H and the user has the mockup link. No product code in this card. Consumer: the build-slice cards S1 to S7 of docs/ui/redesign-2026-11-14/G-build-plan.md (created after the user reviews).
<!-- SECTION:DESCRIPTION:END -->

## Acceptance Criteria
<!-- AC:BEGIN -->
- [ ] #1 Audit: every panel and control with a KEEP/MERGE/DEMOTE/DROP verdict, pictures at 1440, 900 and 390 px from fixtures, and the command table
- [ ] #2 Principles and information architecture of the default view
- [ ] #3 Orders: the unified control surface, the primary-input decision with its trade-offs, the additive API draft with V/U backend support
- [ ] #4 Layout presets as watch.layout/1 drafts (default, simple, standard, stream, debug) with the alias table and rollout plan
- [ ] #5 Visual system: tokens, components, accessibility, motion
- [ ] #6 Clickable static mockup in the repo and published as a private Artifact
- [ ] #7 Build plan in mergeable slices for Sonnet workers, and the numbered open questions with a recommendation each
<!-- AC:END -->
