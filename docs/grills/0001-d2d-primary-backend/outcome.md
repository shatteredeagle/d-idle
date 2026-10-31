# Outcome of grill 0001: d2d primary backend

Date 2026-10-30. Result: [ADR 0009](../../adr/0009-d2d-primary-backend.md) (proposed). Option A.

## Round 1
- Q1 product: yes, the final product uses d2d. User added: maybe add d2lod (original Game.exe) support in v2, after the meta-game layer is baked in.
- Q2 only the four gated metrics must match within 20 %, other gaps explained and fixed when cheap: yes.
- Q3 abandon trigger (soak FAIL after 2 fix cycles, engine rewrite gap, container build fails): user pushed back; a soak FAIL may be the bot's fault, a build failure is not d2d's fault. Only the engine-rewrite gap is a valid trigger. Reworked in round 2.

## Round 2
- Q1b v2 "maybe" = v2 candidate, revisit trigger "meta-game layer merged", env.Env keeps it open: yes.
- Q3 reworked: exit only for an engine-rewrite gap (more than about 16 agent-hours) that moves a gated metric past 20 %; attribution rule for soak FAILs; build failures are tasks: yes, with the caveat that changing engines is a break-glass fix (user decision, never automatic).
- Q4 option A now, replicated soak becomes fidelity check instead of promotion gate: yes.
- Q5 original-game confirmation = soak gate metrics plus probes that change decision code: yes.
- Q6 Game.exe frozen now, retired from dev/product path at the d2d container's one-slot 24 h soak, code kept for v2: yes.

## Comparison
The prebrief author recommended C with a named trigger to A; the user chose A directly, with break-glass exit.

## Parked
- GPL review, licence, upstream PRs (cast, price): grill 0002.
- Merc and Act 2: grill 0003.
- Container ports, viewer, template, real time vs lockstep: grill 0004.
