## Why

Baseline set `kit` (lane i2, Sorceress) stalled at clvl 3: the town walk to the Blood Moor looped on a dead gate, and a stray Short Bow made the Sorceress refuse every attack and spend all her gold on Arrows. Evidence and analysis: docs/kitstall.md.

## What Changes

- nav: a gate that stalled every approach and forced the town-portal escape is marked dead for 30 min; it survives the stalled-gate TTL, `learn()` after a crossing elsewhere and a re-entered game. If only dead gates remain they are still tried.
- gear: `WeaponProblem` ignores a launcher without quiver for Sorceress and Necromancer (spells need no ammo).
- town: `AmmoPlan` buys ammo for the Amazon only.

## Impact

internal/nav (exit.go, route.go, stuckmem.go), internal/gear/gear.go, internal/town/ammo.go, tests, docs/kitstall.md. No spec delta.

## Overlap (declared)

internal/nav, internal/gear and internal/town files are owned by other (archived) cards; the edits are small and listed above.
