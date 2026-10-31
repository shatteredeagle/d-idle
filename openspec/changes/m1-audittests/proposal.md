## Why

Card basicsfix (merging 2026-10-31) implements potions-first, ID scrolls on demand, the repair latch and bank-before-shop (docs/basics-audit.md G1, G4, G5, G6) but, like most behaviour cards, without a spec delta, and the audit tests TestAuditS2, S3, S7, S10, S11 still pin the OLD behaviour (basicsaudit card). Without this card the fourth fix of the same symptom is one soak away: the wanted end state must be a spec requirement and an asserting test, not a knob note.

## What Changes

- internal/town/basicsaudit_test.go: TestAuditS2, S3, S7, S10, S11 flipped from pinning the current behaviour to asserting the SME end state on the merged tree; any residual difference fixed in shop.go or chores.go (small) or reported as a gap.
- Purchase order when poor as merged by basicsfix, checked against the d2-sme rule (potions first up to 3, scroll once gold covers scroll plus 2 potions) and the user's decision D3 (DECIDED 2026-10-31: the SME order); the knob default read from docs/knobs.md.
- Follow-up of the merged scroll rule: basicsfix merged "buy a Town Portal scroll at clvl >= 4 with >= 100 gold left" (`town.tpFirstLevel` 4). Under D3 that rule changes to: potions first up to 3, then the scroll only when gold >= scroll price plus 2 potions. The change (shop.go, knob default, tests) is work of this card.
- Delta: town-basics ADDED `ID scrolls only on demand`, `Purchase order when poor`, `Bank before the shop`, `Repair failure latch` (the backfill for basicsfix).

## Dependencies

After cards xpgap and basicsfix are archived (both own internal/town/shop.go and chores.go). Decision D3 is made (SME order); the card must change basicsfix's scroll rule to it.

## Result (agent m1loot)

Implemented: `town.tpScrollPotions` (default 2) and `town.tpFirstLevel` default 0 (extra floor only); shared town code. BASICSFIX had already flipped S2, S3, S7, S10 and S11; S10 now holds a scroll so the D3 order does not mask the repair latch. Delta: MODIFIED early-game-policy `Potions before the first town portal scroll`; town-basics ADDED backfill. The stash cap, trips and pack-full requirements stay with their own cards.
