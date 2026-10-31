# Charms, runes and gems on a sell visit

> status: current (card m1-loot, agent m1loot, 2026-10-31). Rule source: user decision D4 and the d2-sme; shared loot code, not d2d-only.

- **Runes**: always kept (pickit rule `runes`, stash).
- **Small charms** (scha): unidentified ones are kept to be identified (`charms to id`); once identified a charm is kept only with life 20+, a resist 5+, FHR 5+, FCR, MF or skills (`charms`), else it falls to `magic and rare` and is sold.
- **Large and grand charms** (mcha, lcha): the same stat test, but only while 3 pack cells are free (needed kind `charmroom`, counted from the item sizes against the 10 x 4 pack). Without room they are sold.
- **Gems**: chipped and flawed gems (codes gc*, gf*, skc, skf) are sold while no Horadric Cube (`box`) is in the pack or stash; with the Cube they are kept up to `GemKeep` per code for upgrades, as before. Standard grade gems and better are always kept (`mid gems`, `top gems`). Knob `town.sellLowGems` (default 1; 0 = old behaviour).
- **UNVERIFIED**: "Cube available" is the Cube item held, not the Act 2 quest state (no state helper exists in loot; the quest chain returns the Cube only after the Horadric Cube chest). Unidentified charm sell prices on d2d are U (basics-audit.md Unknowns). Item stat ids 105 (FCR) and 80 (MF) follow the item-stat table used by the other rules.
- Tests: `TestCharmsRunesGemsOnSellVisit`, `TestBigCharmsNeedThreeFreeCells`, `TestSellLowGemsKnobOffKeepsThem` (internal/loot), `TestAuditS6RuneCharmAndGems` (internal/town).
