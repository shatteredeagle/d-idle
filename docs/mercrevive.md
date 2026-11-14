# Merc revive stall: knob first (card mercrevive)

> status: offline, nothing verified live. Source: user decision 2026-11-04; analysis in docs/andarielplan.md section 5.

- `merc.kitReservePct` 150 -> 100 (V: internal/env/tuning_merc.go). The revive bar is price x mult + 100 percent of one kit refill (V: `merc.ReviveNeed`, tests TestSpendingOrderReserve with two kit sizes). Gold between the new and the old bar now revives.
- `merc.goldFarmMaxLevel` 16 -> 18 (V: same file, TestGoldFarm). The farm stays on at clvl 16 and 17 while the merc is dead and gold is short.
- Hire and replace use the same reserve, so their bars drop by the same 50 percent of a kit (V: policy tests updated).
- Logs (V: tests): `merc: revive skipped (gold) price=P gold=G need=N` and, once per change of N, `merc: gold wait need=N gold=G` (counter reset when a revive proceeds).
- No merc-policy spec line states these numbers (V: grep), so no delta spec.
- Unverified live: whether the lower bar ends the Gate 2 seed 11 stall.

## Update mercrevive2 (2026-11-14)
Source: docs/research/merc2-trace.md (decomp16).

- V read (FUN_00579c00): the revive handler wants gold only: no town portal, no potions, no reserve. Price = (merc level^2 / 2) x 15, capped 50,000, paid from carried gold then stash.
- Bug B1 (fixed): `merc.ShouldRevive` also demanded a portal and potions (`ReasonKit`, removed). The hero14 Paladin played hours with a dead merc. Now the bar is gold (carried + stash) >= price x `merc.reviveGoldMult` + the revive reserve.
- B2: the reserve was already a knob pair (`merc.reviveReserveLow/High/Clvl`, 0 below clvl 13, 50 from it); it is the single source for the revive bar. `town.mercReserve` (sellskip) is the separate shop-side hold of the price against optional buys; both stay.
- Order: with no healing potion carried the revive before the shop (`ReviveFirst`) waits, the shop buys potions (survival, never gated), then `ReviveStep` revives on gold alone, with the tome and spare scrolls held back by `town.mercReserve`. With a potion carried the revive still comes before the shop (SME order of g2e F5).
- Backoff: a failed attempt (no gold shown, NPC unreachable) blocks new attempts for `mercSkipGap` (2 min); the hourly cap still counts only paid revives.
- B3: contract line reworded (any owned merc refuses; Normal-only test not found, U).
- B4 (proposal, not built): the free Iron Wolf from Asheara after the Blade of the Old Religion (FUN_005b9240, any owned merc refuses it, so a hired Rogue or Desert merc forfeits it). No Act 2 quest chain exists in `data/campaign/` (Normal only has Act 1 steps), so there is nothing to hang a claim step on. When Act 2 is populated: add a chore after that quest, only when no merc is owned or the merc is dead and the class wants the Iron Wolf (data in the merc class table, not Go).
- Tests: `go test ./internal/merc ./cmd/d2bot -run 'Revive|EmptyBelt'` (Sorceress and Paladin; no kit revives; below price+reserve refuses; empty belt shops first; backoff). Not verified live (U).
