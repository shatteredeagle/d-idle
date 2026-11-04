> status: REWRITTEN 2026-11-04 for ADR 0015 and the Current truth block in docs/decisions.md. The old framing (retire Game.exe at a d2d container soak, "G1/G2 criteria") is superseded (ADR 0013 retired the G1/G2 names). This card is now the Gate A card.

## Why

Gate A (fidelity acceptance) is the release gate for the product (ADR 0015). Acceptance path (all must hold):

1. A fresh, uncheated character plays 24 h unattended on d2d and clears Act 1 (Andariel dead, Act 2 entered), with writes after start = 0 (the Gate 3 run of the d2d ladder: Gate 0 drills, Gate 1 2 h, Gate 2 8 h, Gate 3 24 h).
2. The A:B comparison (goal G-D; fresh clvl 1-6 cell, 6 game.exe lanes + 6 d2d lanes) says d2d and game.exe are EQUIVALENT on XP/h and kills/h at 20 percent.

game.exe 2 h / 8 h / 24 h runs are the ORACLE side: they supply the reference lanes for A:B and fidelity evidence. They are not the release gate, and no game.exe soak is needed beyond the A:B reference lanes.

Old game.exe-only paths (kill recovery, recycle, Wine/OOM) stay parked, not Gate A criteria. [S: ADR 0015]

## What Changes

- tasks: the d2d 24 h Gate 3 run record (fresh, uncheated, writes after start 0) and the G-D A:B verdict at 20 percent are the two Gate A inputs
- list which game.exe runs (2 h / 8 h / 24 h) serve as oracle reference lanes for G-D; drop the "regression soak on Game.exe" item
- Gate 3 needs the user's go; leaving d2d after a DIFFERENT verdict is the user's call, never automatic

## Not in scope

Tier B release, the d2d container, game.exe-only failure paths.
