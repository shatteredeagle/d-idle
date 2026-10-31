---
status: accepted
date: 2026-10-30
decision-makers: user (decisions made in conversation), coordinator
---
# 0008: Agents do not kill the soak game process unless a permission rule allows it

## Context and Problem Statement

The Claude Code permission classifier denied `kill -9` of the soak own Game.exe (attempt 15), so G1 criteria 6 and 7 (kill recovery, orphan winedevice) are unverified since attempt 8. We do not work around a permission denial.

## Considered Options

* Add a permission rule allowing the kill of the soak own Game.exe
* Use a d2soak fault mode or tools/soak/failsafe.sh drill instead
* Accept unverified criteria

## Decision Outcome

Accepted 2026-10-30: the user granted a permission rule for the exact command (kill -9 of the soak Game.exe, run from the coordinator), after the classifier twice denied it without one. No agent kills a game process without such a rule. First use: G2 at 08:36:34, recovery logged `Game lost: restarting it` at 08:36:35, `recycle done` at 08:37:06 (31 s), farming again at about 08:38:06 (92 s, limit 3 min), no orphan winedevice: criteria 6 and 7 passed. The soak-gates spec keeps the 100 s requirement.

## Consequences

Recovery evidence is current (G2 injection 2026-10-30). Open note: the minute-line `restarts` counter stayed 0 after the recycle (it counts bot restarts, not game recycles).

## Sources

docs/g1-log.md attempt 15; docs/BACKLOG.md item 1; docs/plan/STATUS.md open decision 1
