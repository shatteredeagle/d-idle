---
status: accepted
date: 2026-10-30
decision-makers: user (decisions made in conversation), coordinator
---
# 0006: Take the free Rogue (Blood Raven reward); never pay Kashya first

## Context and Problem Statement

A paid Act 1 hire before Blood Raven forfeits the free merc, the merc was dead code, and the route guard needs a merc or clvl 20 to reach Andariel.

## Considered Options

* Paid Kashya hire at clvl 8
* Free Rogue after Blood Raven
* No merc

## Decision Outcome

Free Rogue after the kill, paid Kashya hire only when Blood Raven is skipped for good. The bot therefore needs a real Blood Raven tactic and kit gate. Merc policy is wired behind merc.enabled (default 0); see spec merc-policy.

## Consequences

Blood Raven must be killed early; unverified on the original game (d2d says no hire window and gold unchanged).

## Sources

plan-v2 section 10, 10.1, 11.3; docs/merc.md
