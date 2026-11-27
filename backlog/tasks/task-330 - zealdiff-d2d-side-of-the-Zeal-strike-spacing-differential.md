---
id: TASK-330
title: 'zealdiff: d2d side of the Zeal strike spacing differential'
status: Proposed
assignee:
  - credit-runner
created_date: '2026-11-11T19:21:00Z'
labels:
  - d2d
  - oracle
  - d2d-combat
  - fidelity
dependencies: []
priority: medium
ordinal: 224000
card: "zealdiff"
owns:
  - "tools/emu/zd_*"
exclusive: []
contracts: []
lane: next
result: "d2d side done earlier (V run, docs/research/pal-zeal-trace.md): d2d strikes Zeal 15 ticks apart at IAS 0 against game.exe 7 (24 of 24 gaps differ, 5-strike cast 75 vs 43). Harness tools/emu/zd_*. AC 3 (fork fix and zero-mismatch rerun) is fork-side work, left as a handoff in this card."
---

## Description

<!-- SECTION:DESCRIPTION:BEGIN -->
Size: M

Why: fid-zeal.md: game.exe restarts the swing after every Zeal strike (Param2 100 through FUN_00553b10), d2d waits a full swing (V read); the d2d side was not run. Done when: the differential has mismatches before and zero after.
<!-- SECTION:DESCRIPTION:END -->

## Acceptance Criteria
<!-- AC:BEGIN -->
- [x] #1 drive d2d Fight with a Zeal swing and count ticks between strikes
- [x] #2 compare with fid-zeal.md table (7 ticks at 1hs IAS 0)
- [ ] #3 if mismatching, local fork branch with the fix and a zero-mismatch rerun (FORK SIDE, left open: handoff below)
<!-- AC:END -->

## Fork-side handoff (AC 3, not done here; fork edits are out of scope for this card)

- Where (S, from pal-zeal-trace.md): `Fight::player_modes` (strike.cpp, kicks_left path): after a strike lands with `kicks_left > 0` and a target, set `pmode_until` to strike tick + ceil(256 x action_frame / rate) ticks instead of the full swing (game.exe: Skills Param2 = 100 through FUN_00553b10).
- Rerun: build with `tools/emu/run-on-worker.sh --ttl 1500 zd_build.sh`, then `zd_d2d.sh 106 4 IAS WSM TICKS`; expected gaps 7 / 6 / 6 / 6 / 5 / 9 at the six rate rows of `zd_game.py 256 299 332 307 384 204`, zero mismatches. U: Dragon Talon and Fend share the kicks_left path (fid-zeal.md D2).
- Push the fork branch the same day (AGENTS.md); no GitHub post without the user's go.
