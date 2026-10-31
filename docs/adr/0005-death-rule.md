---
status: accepted
date: 2026-10-30
decision-makers: user (decisions made in conversation), coordinator
---
# 0005: Deaths are judged by streak, recovery share and rate, not by a fixed count

## Context and Problem Statement

The first proposal (3 deaths in 20 minutes or 5 per hour) was not approved: it let 24 deaths in 8 hours pass. The d2-sme and logged death data (docs/plan/death-data.md) gave a better set.

## Considered Options

* Fixed deaths per hour
* Streak only
* Streak plus recovery share plus rate, with repeat-site action

## Decision Outcome

Approved by the user 2026-10-30 as plan-v2 section 10.2: FAIL-1 streak, FAIL-2 recovery stuck, FAIL-3 rate, WARNs, repeat-site ban, BUG tag. Implemented in cmd/d2soak and internal/bot/deathrule.go; stated as numbers in spec death-rule.

## Consequences

Rates need at least 150 kills in the window; kills undercount by 16 to 37 percent today.

## Sources

plan-v2 section 10.2; docs/soak-analysis.md section 12
