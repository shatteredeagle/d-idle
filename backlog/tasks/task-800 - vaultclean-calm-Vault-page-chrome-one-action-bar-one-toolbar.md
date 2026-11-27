---
id: TASK-800
title: 'vaultclean: calm Vault page chrome (one action bar, one toolbar)'
status: Done
assignee:
  - credit-runner
created_date: '2026-11-22T03:48:00Z'
updated_date: '2026-11-27T02:03:00Z'
labels:
  - watch
  - feature
dependencies: []
priority: medium
ordinal: 367000
card: "vaultclean"
owns:
  - "internal/watch/web/vault.html"
  - "internal/watch/web/vault.js"
  - "internal/watch/web/vault-logic.js"
  - "internal/watch/web/vault.css"
  - "internal/watch/web/vault.json"
  - "internal/watch/vaultclean_test.go"
  - "internal/watch/testdata/vault_clean.mjs"
  - "internal/watch/testdata/vault_reveal.mjs"
  - "tools/desktopscroll/vault.mjs"
  - "tools/uishots/fit.js"
  - "docs/ui/vault-cleanup"
  - "docs/vault.md"
  - "tools/uishots/vaultshots.sh"
exclusive: []
contracts: []
result: 'built and tested offline (go test ./internal/watch ./tools/uishots, node harnesses with fake timers, tools/desktopscroll/run.sh green on the builder, fixture-viewer pictures at 1366x768, 1920x1080, 390x844 in docs/ui/vault-cleanup/); not seen on the live page (no deploy by this card). Build-specific: none. Not touched: the API, the bot. closecards3: ticked #4 (merged through the merge gate (merge commit on main)).'
lane: next
---

## Description

<!-- SECTION:DESCRIPTION:BEGIN -->
Size: M

Why: user feedback 2026-11-21 on the live Vault page ("very confusing/messy"): an empty Reveal fold, three stacked bands above the grid, a framed grid with empty space, a ? on every tile. Done when: one toolbar, one contextual action bar, transient status chips, one Marked for sale chip, a frameless grid, a ? badge on unidentified tiles only, an item card docked only while open.
<!-- SECTION:DESCRIPTION:END -->

## Acceptance Criteria
<!-- AC:BEGIN -->
- [x] #1 No empty fold; one action bar
- [x] #2 Status chip expires; marked chip; no grid frame; unid badge only
- [x] #3 Screenshots 3 sizes in docs/ui/vault-cleanup
- [x] #4 Gates green and merged
<!-- AC:END -->
