---
id: TASK-126
title: 'learnedbrain: tiny answer-prediction model experiment'
status: Done
assignee:
  - credit-runner
created_date: '2026-11-07T23:44:00Z'
updated_date: '2026-11-10T16:54:00Z'
labels:
  - bot
  - research
  - parked
dependencies:
  - TASK-106
priority: low
ordinal: 118000
card: "learnedbrain"
owns:
  - "docs/research/learnedbrain.md"
  - "docs/research/decision-dataset.md"
exclusive: []
contracts: []
result: "Status sync 2026-11-08 (stays open): AC1 open: no 500 labelled answers yet; merge was the decision dataset (TASK-106), not this card."
---

## Description

<!-- SECTION:DESCRIPTION:BEGIN -->
Size: L

Size: L

Why: user decision 2026-11-07 (record the tiny learned brain idea, keep the data). Blocked until TASK-106 has logged the decision dataset (docs/research/decision-dataset.md) and about 500 labelled player answers exist. Done when: the criteria below hold.

Experiment: train a small classifier (decision tree, logistic regression, small MLP) OFFLINE (Python dev tool, not shipped) on the player's answers to decision-point questions; measure agreement with the current rule defaults (the catalogue default and legacy answer) on a held-out set split by session; decide in ONE day whether a model adds value over the rule defaults. First uses: rank which questions are worth asking, pre-fill defaults for the queue while the player is away.

Stop conditions: fewer than ~500 labelled player answers (do not start); held-out agreement not above the rule defaults by a margin written before looking at the result (stop, record negative result); a day spent without a decision (stop and report); features leak the outcome window (discard the run).

Determinism: any inference that could ship is stdlib-only Go with quantised integer weights (no floats in the decision path), so lockstep runs stay reproducible; a model must give identical output for identical input on every host.

Dependency rule: no runtime dependencies in the bot (go.mod untouched); Python and any ML library are offline dev tooling only, outside the shipped binary.

Cheaper alternatives to try first, same features: decision tree and logistic regression, and Bayesian tuning of ladder knobs with the paired A/B harness (tools/ab).

Out of scope: a reinforcement-learned policy, until the fun target is set and d2/d2d parity is closer (a policy trained in d2d would exploit simulator quirks such as the chicken-rejoin full heal found by the fidelity probe).
<!-- SECTION:DESCRIPTION:END -->

## Acceptance Criteria
<!-- AC:BEGIN -->
- [ ] #1 Dataset check: count labelled player answers in the decision dataset (answered_by=player); start only at 500 or more, else report the count and stop
- [ ] #2 Baselines first: rule default, decision tree and logistic regression on the same features; held-out split by session id; agreement table in docs/research/learnedbrain.md
- [ ] #3 Only if a baseline beats the rule defaults by the pre-written margin: one small MLP, quantised to ints, with a Go stdlib inference sketch (no runtime deps) and a determinism check on two hosts
- [ ] #4 Verdict written in one day: adds value or not, with the stop condition that applied; no RL policy work; leakcheck 0
<!-- AC:END -->

## Implementation Notes

<!-- SECTION:NOTES:BEGIN -->
Closed in the 2026-11-10 true-up (docs/research/project-review-2026-11-10.md section 3.1): merged, offline acceptance proven; open live checks moved to docs/live-checks.md (TASK-253).
<!-- SECTION:NOTES:END -->
