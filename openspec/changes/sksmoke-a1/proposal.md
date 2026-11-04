## Why

docs/research/sorc-act-gate.md section 3 row 4 (SKSMOKE-A1): the Act 1 Sorceress gate needs live "SK" drills for Fire Bolt and Charged Bolt damage (levels 1, 4, 8, 13, 20), kills per cast on a pack, and the Frost Nova ring. Every lab instance was stopped when this card started (devhealth and game.sh status, 2026-11-03), and agents may not start games, so the live part is BLOCKED; this card holds the drill design and the oracle expected values so the run can be done in one session.

## What Changes

- docs/probes/drills/2026-11-03-SK-design.txt: expected per-hit damage (min, max, mean) per level from the patched tables, the measurement method, PASS conditions, the setup cheats, the run steps for instance i2.
- No Go code (no game.exe-side drill harness for skills exists; the existing skill drills run on d2d slots, not on game.exe).

## Impact

Docs only. Frost Nova ring: no harness exists in internal/drill or cmd/d2drill (no "nova" there), so it is reported as not done.
