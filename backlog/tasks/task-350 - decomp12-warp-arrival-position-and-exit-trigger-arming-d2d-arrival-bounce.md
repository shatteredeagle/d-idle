---
id: TASK-350
title: 'decomp12: warp arrival position and exit trigger arming (d2d arrival bounce)'
status: Done
assignee:
  - credit-runner
created_date: '2026-11-11T23:29:00Z'
updated_date: '2026-11-12T01:51:00Z'
labels:
  - d2d
  - research
dependencies: []
priority: high
ordinal: 228000
card: "decomp12"
owns:
  - "docs/probes/fid/fid-warp-arrival.md"
  - "tools/emu/d12_arrive_probe.cpp"
  - "tools/emu/d12_build.sh"
  - "tools/emu/d12_fix.py"
  - "tools/emu/d12_fixrun.sh"
  - "tools/emu/d12_run.sh"
  - "tools/emu/d12_setup.sh"
  - "tools/emu/d12_commitmsg.txt"
exclusive: []
contracts: []
result: "Offline, not verified live. D1 proven by differential: d2d arrival spot search (nearest_free) differs from game.exe FUN_0064dea0 in 10265 of 20000 synthetic cases and 25-26 of 52 real Act 1 arrivals per seed; fix candidate fid/warp-arrival-r1 feba9539 (private fork, pushed) gives 0 differ. The arrival bounce itself is NOT explained (click box equal in both; 18 percent of clicks vs 88 percent observed): U, drill in the doc."
---

## Description

<!-- SECTION:DESCRIPTION:BEGIN -->
Size: M

Consumer: passagebounce TASK-349.
<!-- SECTION:DESCRIPTION:END -->

## Acceptance Criteria
<!-- AC:BEGIN -->
- [x] #1 docs/probes/fid/fid-warp-arrival.md answers Q1..Q4 with tags
- [x] #2 Arrival-spot differential game.exe vs d2d (V run) and fix candidate on private fork branch fid/warp-arrival-r1 pushed
<!-- AC:END -->
