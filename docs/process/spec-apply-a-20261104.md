# Spec apply A, 2026-11-04

Capabilities: soak-gates, fidelity-gate, death-rule, upstream-pr, replay, d2d-farm. Source: docs/process/spec-ledger-20261104.md and the user's decisions of 2026-11-04 (items 1, 2, 5, 6 apply here). One commit per capability. Only `openspec/specs/<cap>/spec.md` was edited.

## soak-gates

| What | Rows | Result |
|---|---|---|
| Folded | gate3gaps: Resource slopes, Gate 2 and Gate 3 on d2d (MODIFIED), Full bag and disk guard, Gate 0.5 smoke (ADDED) | in force; thresholds marked U stay U (95 percent over 60 lines, 5 GB, 20 GB, 45 min smoke), none invented |
| Folded | specdelta: Gate 1 belt and portal (funded window, stashGold); Gate 1 deaths and progression, rewritten per decision 2 | death rule + no state over 10 virtual min without a kill or XP gain; XP/h never a per-run verdict |
| Folded | baselinefree: the 3 band requirements | kept as ADVISORY diagnostics (FAIL/hard verdict wording replaced by "thin baseline" / "below band" flags that fail nothing); numbers 1.96 and 0.14 unchanged |
| Folded | andarielspec: G2 meaning and floors, with its 5 scenarios | interim criterion text kept; title now "Gate 2 and Gate 3 meaning and floors" (decision 6: stability and economy, G-C; progression is G-B; while boss.andarielHold is on, reaching the Andariel gate) |
| Left (pending) | d2d-container-soak ADDED (open, superseded by ADR 0015) | listed under "Pending deltas" |
| Not folded | gatedelta-style XP fail in Gate 1 scenarios (baselinefree band FAIL, specdelta "one seed, inconclusive" pass) | superseded by decision 2 and ADR 0015 item 8 |
| Cleaned | Purpose: G1/G2 names replaced by Gate 1 to 3; status line added | |
| Cleaned | Requirement text shortened under 500 characters so `validate --strict` passes (sources condensed, meaning kept) | |

Open question: Soak progress floors text lists clvl, quest step and gold as progress; code also counts XP gain of 500 or more (cmd/d2soak/progress.go). No decision covers it; text unchanged, `> OPEN QUESTION` line added.

## fidelity-gate

| What | Rows | Result |
|---|---|---|
| Folded | abtool: Verdict classes, Twenty percent tolerance (MODIFIED), Game.exe arm sample and regimes, Pair setup on both backends (ADDED) | in force |
| Folded | replicated-soak: Replicated soak is a fidelity check, Soak FAIL attribution | in force |
| Folded | latebase: Late baseline | in force |
| Dropped | gatedelta Death budget, Death rate cap and Poisson bound, Proxy metrics, Proxy definitions, Missing proxy data (decision 1) | not folded; Purpose note says soaks judge deaths by death-rule, A:B uses the 20 percent tolerance |
| Cleaned | "What d2d cannot validate": "(G1, G2)" replaced by game.exe topics covered by Gate A | |

Open question: the replicated-soak text says "fire the exit trigger" (ADR 0009 wording). Decision 4 ties leaving d2d to the A:B (G-D) verdict, user decides; wording needs the user's confirmation. `> OPEN QUESTION` line added under the requirement.

## death-rule, replay, d2d-farm

| What | Result |
|---|---|
| Folded | nothing (all deltas APPLIED) |
| Cleaned | status line under Purpose; death-rule line states it is the only authority on soak deaths |
| Open | none |

## upstream-pr

| What | Result |
|---|---|
| Folded | nothing (2 SUPERSEDED rows, no edit) |
| Cleaned | dead references `tools/pr_guard.py` and `tools/presubmit.py`/`tools/d2ddet`: now say they live in the upstream project checkout, not this repo (U: not re-checked in that checkout from here); status line |
| Open | none |

## Counts

Folded: 6 (gate3gaps 4, specdelta 2) plus 3 band requirements, 1 andarielspec, abtool 4, replicated-soak 2, latebase 1 = 17 requirements. Dropped: 6 (gatedelta). Left pending: 1 (d2d-container-soak). Open questions: 2.
