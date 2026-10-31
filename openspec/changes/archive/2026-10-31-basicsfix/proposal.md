## Why

BASICSAUDIT (docs/basics-audit.md) found town-basics gaps; the user (2026-10-31) asked for the fixes and decided the shop policy: potions first. Numbers are the d2-sme's EXP rule of thumb (UNVERIFIED on d2d), each behind a `town.*` knob.

## What Changes

- `town.idScrollNeed` (1): identify scrolls are bought only while an unidentified set, rare, unique or crafted item is in the pack; a plain unidentified magic item is sold as it is. 0 restores the old top-up to idScrollTarget.
- `town.repairLatch` (1): a repair that changed nothing is not retried until durability changed or gold grew by futileGoldStep. 0 retries every visit.
- `town.bankFirst` (1): the gold above the keep line (300 + 50 x clvl, never below the minimal kit) is banked before the shop. 0 banks after it.
- `town.potionsFirst` (1) and `town.tpFirstLevel` (4): kit potions (healing to hpFirst, mana to the recovery floor) come first, then the first town portal scroll only at clvl >= 4 and with gold for one left. 0 buys the scroll first.
- Skipped, left for after XPGAP merges: the second sell-trip door at `bot/goal.go` (same line as XPGAP's povertySellMin edit) and the "no town portal scroll" trip reason under potionsFirst (XPGAP's tpMinGold edit in `tripReasons`).

Delta spec: ADDED requirements under early-game-policy (no numeric gate there changes).

Shared code (`internal/town` also serves the Game.exe path): every change is a G1/G2 check item; each has a knob whose 0 restores the old behaviour.
