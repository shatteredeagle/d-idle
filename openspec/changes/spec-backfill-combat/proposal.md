## Why

Card navstall (2026-10-30) fixed a lockstep hang in the combat loop and carried skip_specs, and no capability spec covers combat. The spec-sme review (2026-10-31) listed it as behaviour-bearing without a spec; the failed-walk tick, the pinned retreat and the no-progress watchdog are contracts a later change must not break.

## What Changes

- New capability `combat-loop` (delta ADDED, applied to openspec/specs/combat-loop/spec.md at merge): failed walk costs a poll tick, no retreat while pinned, fight-loop watchdog, and the champion caution rules of card champ (group avoid below clvl 8, potion lines +combat.championEarlyPct, champion death label).
- Specs only; no code is touched.
