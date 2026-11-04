## Why

docs/research/sorc-act-gate.md section 3 row 4 (SKSMOKE-A1): the Act 1 Sorceress gate needs live "SK" drills for Fire Bolt and Charged Bolt damage (levels 1, 4, 8, 13, 20), kills per cast on a pack, and the Frost Nova ring. Every lab instance was stopped when this card started (devhealth and game.sh status, 2026-11-03), and agents may not start games, so the live part is BLOCKED; this card holds the drill design and the oracle expected values so the run can be done in one session.

## What Changes

- docs/probes/drills/2026-11-03-SK-design.txt: expected per-hit damage (min, max, mean) per level from the patched tables, the measurement method, PASS conditions, the setup cheats, the run steps for instance i2.
- Tools for the drill (offline, tests only): `d2cheat skilllevel` (cmd/d2cheat/skilllevel.go) and `combatlive sk` (cmd/combatlive/sk.go), documented in docs/sksmoke-tools.md. Each needs a 4 to 10 line case in the existing main.go of its command (cmd/d2cheat/main.go, cmd/combatlive/main.go).

## Impact

Docs plus two small tools. Frost Nova ring: no harness exists in internal/drill or cmd/d2drill (no "nova" there), so it is reported as not done.

## Overlap (declared)

m1-liveproof owns docs/probes/drills/**; this card only adds one new file there (2026-11-03-SK-design.txt) and edits no existing drill file.

Overlap (declared): cmd/d2cheat/main.go is owned by snaplib/ledgerkey; this card adds one `case "skilllevel"` and one usage line there, nothing else.
