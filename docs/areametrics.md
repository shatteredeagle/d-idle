# Per-area gold and XP log (change areametrics)

Logging only, no behaviour change. Tags: V = verified offline (unit tests named below), S = secondary/derived, U = unverified live. Nothing was run live.

| Line | New fields | Written by | Read by | Tags |
|---|---|---|---|---|
| `minute` | `goldPickup`: positive gold change between two snapshots in the same non-town area since the last minute line | `noteAreaMetrics` (internal/bot/areametrics.go), `minuteLine` resets it | `ParseAreas` | V by `TestMinuteLineGoldPickupResets`. S: a field shrine/refund would count, a town sale never does |
| `area summary` (info) | `area`, `minutes` (rounded wall/virtual minutes in the area), `xp` (sum of positive exp changes), `kills` (observer kill counter delta), `goldPickup` | `noteAreaMetrics`, once per area change | `ParseAreas` (counts them) | V by `TestAreaSummaryAndGoldPickup`. Kills of the first snapshot after a border can land in the new area (S) |
| `combat kill` | `class`: monster id from the unit (data row, no literal) next to `monster` name and `xp` | `noteKills` | `ParseAreas` | V by `TestCastLogDoesNotChangeActions` (same actions with the log on and off). `xp` is the pooled exp gain since the last tick split evenly over the monsters that died in that tick (S): exact for single kills, a mean for packs |
| `merc: gold farm off` | `reason=` `merc-back` / `gold-reached` / `other` (clvl cap or knob 0) | `mercWire.GoldFarm` via `merc.LogGoldFarmReason` | humans | V by `TestLogLines`. The on/off line with need/gold/clvl already existed (mercwire.go); no second line. mercrevive2 adds only a budget line, not touched |

## Report
`d2dsoak report LOG...` prints after the per-skill section: per area minutes, XP/h, kills, XP/kill, goldPickup/h, then XP/kill per monster. XP/kill and XP/h come from the differences of consecutive minute lines (booked to the area of the later line, gaps capped at 2 min); old logs without `goldPickup` or `class` print n/a (`TestAreasOldLogNA`).

## What it cannot know
- Which kill gave which gold: pickups are per area, not per monster.
- Gold from sales, quest rewards or stash moves is excluded only through the town test (a field-area gold change from other sources counts).
- XP per kill by monster is a share of the pool (see above), not the game's exp formula.
- Not measured live (U): the real size of a pickup per minute.
