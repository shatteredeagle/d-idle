---
id: TASK-232
title: >-
  uidesign2: inventory paper-doll mockup, tokens style guide, PNG renders of
  every mockup
status: Done
assignee:
  - credit-runner
created_date: '2026-11-09T19:32:00Z'
updated_date: '2026-11-09T19:59:00Z'
labels:
  - watch
  - research
dependencies: []
priority: high
ordinal: 177000
card: "uidesign2"
owns:
  - "docs/mockups/ui/"
  - "docs/ui-design-system.md"
exclusive: []
contracts: []
result: 'Done (docs only): inventory.html is a Diablo II paper doll (8x6 cell grid, pips, BROKEN stamp, PIN, upgrade arrow, S/K marks, detail card, responsive); tokens.html is a style guide with WCAG ratios computed at generation (all pairs pass 4.5:1 except --d-text-3 on surface-1 at 4.3:1, placeholder use only); 27 PNG renders from google-chrome-stable --headless=new under a private Xvfb on the worker host at nice 19 (1440x900, 1440 full page, 1920x1080, 390 wide), all under 265 KB; copied with the mockups to the draft dir served on port 8899 (index.html lists the renders). VERIFIED by looking at the renders: watch-wide 1440/1920, inventory 1440/1920, tokens 1440, phone 390. Gates: hygiene, merge lines, leakcheck, backlog check pass.'
---

## Description

<!-- SECTION:DESCRIPTION:BEGIN -->
Size: M

User feedback on the TASK-228 mockups: policy, return and overlay are good; the inventory paper doll must look like the in-game screen, the tokens page is messy, and the user wants PNG renders to see it in a normal browser. Done when the three are merged and served.
<!-- SECTION:DESCRIPTION:END -->

## Acceptance Criteria
<!-- AC:BEGIN -->
- [x] #1 inventory.html shows a Diablo II style paper doll (helm, amulet, armour, weapon, shield, gloves, belt, boots, two rings) above the 10x4 backpack, belt and stash, with rarity colour, durability per slot, BROKEN, sell/keep marks, pin icon, upgrade hook; responsive; docs/ui-design-system.md gains the paperdoll component
- [x] #2 tokens.html is an organised style guide: colour swatches with contrast ratios, type specimens, spacing, radius, elevation scales, tidy component gallery
- [x] #3 docs/mockups/ui/renders/ holds PNGs of every page at 1440x900 and 1920x1080 full page plus 390 wide for phone and overlay, each under 400 KB, rendered by headless Chromium under Xvfb on the worker host
- [x] #4 mockups and renders copied to the worker host draft directory with an index page listing the renders
- [x] #5 hygiene gate, merge_lines_check, leakcheck and backlog check pass
<!-- AC:END -->
