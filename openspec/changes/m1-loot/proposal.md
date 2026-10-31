## Why

docs/basics-audit.md row 6 is P: a rune and a gem survive a sell visit (test), but a plain magic charm falls through to the `magic and rare` pickit rule and is sold, and low gems sell by design. The d2-sme (2026-10-31) gave the human rule: keep every small charm, identify it with a spare ID scroll or Cain and keep it only with life, a resist, FCR, FHR or MF; large and grand charms only with 3 free cells; sell chipped and flawed gems before the Cube (Act 2), keep every rune. The user's checklist row 6 says gem kept; that conflict is decision 6 of docs/roadmap.md and the card implements whichever the user picks (default: the SME rule).

## What Changes

- internal/loot/pickit_starter.json: a `charms` rule that holds every 1x1 charm until identified, keeps it with life, resist, FCR, FHR or MF, sells it otherwise; large and grand charms need 3 free cells; runes always kept; chipped and flawed gems sold before the Cube is owned (or kept, per decision 6).
- internal/loot/charms_test.go: a recorded pack with 2 charms (one with life, one plain), 1 rune, 1 chipped gem, 1 flawless gem survives a sell pass per the rule.
- docs/loot-charms.md: the rule, the decision, what is U (unidentified charm prices on d2d).
- Delta: town-basics ADDED `Charms, runes and gems on a sell visit`.
