## Why

Card basicsfix (merging 2026-10-31) implements potions-first, ID scrolls on demand, the repair latch and bank-before-shop (docs/basics-audit.md G1, G4, G5, G6) but, like most behaviour cards, without a spec delta, and the audit tests TestAuditS2, S3, S7, S10, S11 still pin the OLD behaviour (basicsaudit card). Without this card the fourth fix of the same symptom is one soak away: the wanted end state must be a spec requirement and an asserting test, not a knob note.

## What Changes

- internal/town/basicsaudit_test.go: TestAuditS2, S3, S7, S10, S11 flipped from pinning the current behaviour to asserting the SME end state on the merged tree; any residual difference fixed in shop.go or chores.go (small) or reported as a gap.
- Purchase order when poor as merged by basicsfix, checked against the d2-sme rule (potions first up to 3, scroll once gold covers scroll plus 2 potions) and the user's decision 5; the knob default read from docs/knobs.md.
- Delta: town-basics ADDED `ID scrolls only on demand`, `Purchase order when poor`, `Bank before the shop`, `Repair failure latch` (the backfill for basicsfix).

## Dependencies

After cards xpgap and basicsfix are archived (both own internal/town/shop.go and chores.go). Decision 5 of docs/roadmap.md (purchase order) is needed only if basicsfix's order differs from the SME rule.
