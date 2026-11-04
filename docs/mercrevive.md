# Merc revive stall: knob first (card mercrevive)

> status: offline, nothing verified live. Source: user decision 2026-11-04; analysis in docs/andarielplan.md section 5.

- `merc.kitReservePct` 150 -> 100 (V: internal/env/tuning_merc.go). The revive bar is price x mult + 100 percent of one kit refill (V: `merc.ReviveNeed`, tests TestSpendingOrderReserve with two kit sizes). Gold between the new and the old bar now revives.
- `merc.goldFarmMaxLevel` 16 -> 18 (V: same file, TestGoldFarm). The farm stays on at clvl 16 and 17 while the merc is dead and gold is short.
- Hire and replace use the same reserve, so their bars drop by the same 50 percent of a kit (V: policy tests updated).
- Logs (V: tests): `merc: revive skipped (gold) price=P gold=G need=N` and, once per change of N, `merc: gold wait need=N gold=G` (counter reset when a revive proceeds).
- No merc-policy spec line states these numbers (V: grep), so no delta spec.
- Unverified live: whether the lower bar ends the Gate 2 seed 11 stall.
