## Why

SKMANA findings B1-B3 (docs/research/sk-mana-trace.md) and DRILLS13 M1, DRILLS15 M2/M4: the bot's mana model is off. Regen is a straight line of 100 s per pool (game.exe adds a whole number of 256ths per frame: 1.2x to 1.8x slower on early pools), the ManaBurnTracker books a cast when it is sent (game.exe pays at the action frame and not for a cut cast, d2d pays 1 tick after the command and still charges a cut cast), the sim drinks a mana potion at once.

## What Changes

- internal/resource/manamodel.go: `ManaPerFrame256`, `ManaPerSec` (per-frame whole-256ths rule, one place, named constants with evidence tags), `ManaBook` with `BookD2D` / `BookGameExe`; types.go loses `manaFullSecs`.
- internal/combat: the fight state books casts through `ManaBook` (model from `Fighter.D2D`), player mode hit recovery cuts pending bookings on game.exe.
- sim.go: mana potion arrives over 128 frames.
- No threshold or skill-choice change. Two policy tests needed new inputs because the thresholds now meet slower regen (listed as follow-ups in docs/fixmanamodel.md).

## Impact

Offline only, not verified live. No spec names the regen model (grep openspec/specs), so no delta. Overlap (declared): internal/resource/** and internal/combat/{fight,castfirst,hard}.go, resource_test.go are owned by main (fight.go was fixbuff's, now archived); this card edits them (small edits) and does not list them in owns.
