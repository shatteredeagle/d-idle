---
status: superseded by 0009
date: 2026-10-30
decision-makers: user (decisions made in conversation), coordinator
---
# 0002: d2d becomes the likely primary backend; the original game stays the acceptance target

## Context and Problem Statement

Wine and Proton make the container heavy. d2d (https://github.com/psi29a/d2decomped, GPL-3.0) is a native reimplementation with a control socket and runs headless 100 to 380 times faster than real time on our lockstep fork.

## Considered Options

* Stay on Wine
* Switch to d2d now
* Switch to d2d primary once it builds headless, fidelity is good enough and the channel carries what the bot needs

## Decision Outcome

Third option. Keep decision code backend-agnostic behind env.Env and the interfaces (spec backend-contract), stop investing in Wine-specific container work (paused), and gate trust on the replicated soak within 20 percent (spec fidelity-gate). Probes d2d can answer cheaply go through d2d first and are confirmed on the original when cheap.

## Consequences

Two backends to keep in step; V-d2d results are not acceptance results; the Wine container stays as deployed.

## Sources

decisions.md 2026-10-30; docs/d2d-spike.md; docs/d2d-backend.md; plan-v2 section 11.1 and 11.4
