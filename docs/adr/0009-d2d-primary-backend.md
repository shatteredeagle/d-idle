---
status: accepted
date: 2026-10-30
decision-makers: user (grill 0001), grill session
supersedes: 0002
---
# 0009: d2d is the primary backend; Game.exe is frozen and stays the acceptance reference

## Context and Problem Statement

ADR 0002 made d2d (private lockstep fork of the GPL-3.0 reimplementation) the "likely" primary backend, gated on proof. The prebrief of grill 0001 lists what is shown: headless build and run without Wine (V, the worker host), 90x to 370x real time (V), deterministic output for the same seed and inputs (V), bot navigation, combat and Akara trade on d2d (V, short runs), and known gaps (tome price, cast period, potion double-fill; V-d2d, cast and tome fixed on `lab2`). What is not shown: hours of the full bot loop, the replicated soak against the original (U), merc and Act 2 (U), a container build (U). The question was whether "likely" becomes "yes", what would make us leave d2d, and what stops being done.

## Considered Options

* A: d2d primary for development, soak and the container; original game for acceptance only
* B: d2d test bench only; product stays Game.exe under Wine (Wine container work resumes)
* C: hybrid; d2d for iteration, original as acceptance, product backend decided later by the fidelity gate

## Decision Outcome

Option A, effective now.

1. Product: v1 is the d2d container; users never run Game.exe. Original-game (d2lod) support is a v2 candidate, revisited when the meta-game layer is merged (trigger), not a commitment. Decision code stays backend-agnostic behind env.Env (spec backend-contract), which keeps that door open.
2. Fidelity: only the four gated metrics of spec fidelity-gate must match the original within 20 percent (XP/h, kills/h, deaths/h, deaths per 1000 kills). Prices, cast timing, drop rates, wake radius and similar are explained known gaps, fixed on the fork when cheap. The bot needs relative behaviour, not exact rules.
3. The replicated soak is no longer the gate for making d2d primary; it is the fidelity check that finds gaps to fix and that can fire the exit trigger.
4. Original-game confirmation: the replicated soak gate metrics, plus any probe whose answer changes bot decision code (merc rule, death rule, a price the planner uses). All other probes are d2d only.
5. Game.exe/Wine is frozen now: current paths keep working, G2 finishes, no new Game.exe-only features, no new Game.exe memory-offset work, Wine container stays as deployed and paused. It leaves the dev and product path when the d2d container passes its one-slot 24 h soak (docs/d2d-container/risks.md step 10). The code stays in the repo for v2.

### Exit trigger (break glass only)

Leaving d2d is a break-glass decision by the user, never automatic. It is on the table only when BOTH hold:
* a d2d engine gap whose fix is a rewrite, not a patch (more than about 16 agent-hours for one gap), and
* that gap moves a gated metric past the 20 percent tolerance.

Attribution rule: a replicated-soak FAIL counts against d2d only when the same bot build's original-game baseline is inside the gate and the gap is traced to an engine difference; otherwise it is a bot bug, fixed in the bot. Container build failures (C++26 toolchain, vendoring) are tasks (other compiler, other base image, build outside and copy in), never an exit trigger.

## Consequences

* Container work on a d2d image restarts (risks.md steps 1 to 4 first); `--backend d2d` becomes the dev default once the bot loop runs on it.
* One backend gets new features; the frozen one only needs to keep working until the retirement trigger, which ends the two-backend cost of ADR 0002.
* Risk accepted: green d2d results may not transfer (fidelity illusion); mitigated by the gate metrics and decision-relevant probes on the original. V-d2d results are still not acceptance results.
* Still unproven, now work items rather than reasons to wait (all U): hours of the full bot loop on d2d, the replicated soak comparison, merc on d2d, Act 2 and later, the container build.
* No image leaves the NAS until the GPL question is decided (ADR 0003, grill 0002).

### Gates and triggers
* Retire Game.exe from dev and product path: d2d container passes the one-slot 24 h soak.
* Revisit original-game support (v2): meta-game layer merged.
* Exit d2d: the break-glass condition above, user decision.

### Parked (separate decisions)
* GPL review, licence route and upstream PRs for the cast and price fixes: grill 0002.
* Merc strategy and Act 2 entry: grill 0003.
* Container ports, viewer, template, real time vs lockstep in the container: grill 0004.

## Sources

docs/grills/0001-d2d-primary-backend/prebrief.md and outcome.md; ADR 0001, 0002, 0003; openspec/specs/fidelity-gate/spec.md; docs/d2d-soak.md; docs/d2d-container/risks.md; docs/d2d-fidelity.md; docs/d2d-fork.md

## Addendum (2026-10-30, user)

User, verbatim: "We decided all work is on d2d now and anything touching d2lod will be much further down the line."

* ADR accepted by the user.
* Validation of the Game.exe-only paths (kill recovery, failsafe ladder, recycle, Wine/OOM) is DEFERRED to when original-game work resumes.
* G2 and the Fresha soak currently running on Game.exe finish as the last Game.exe baselines and are not rerun on d2d.
