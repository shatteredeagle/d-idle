## Why

Equip failures (130 on Soaky, 18 on Fresha, EQUIPLIVE) are items the character cannot wear yet: the game refuses the slot click. The town's `viable` check reads the base table only; the level requirement of a magic item is max(base levelreq, highest affix levelreq). A full belt also looked like it had room (`Free()`), so potion drags were no-ops.

## What Changes

- RequirementSource seam in internal/town (require.go): effective level/str/dex per item, affix levelreq from the affix tables by display name (d2d), conservative margin for magic and better items on the Game.exe path.
- Unidentified, above-level, small-gain items are skipped before any click; refused clicks put the item back and blacklist it by (id, clvl, str, dex).
- Belt top-up: room counted from occupied cells, aimed at an empty cell, verified by count; a failed belt is not retried until it changes.
- Right-click equip fallback disabled (1.14d equips by click only).
