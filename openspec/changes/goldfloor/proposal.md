## Why

docs/goldfloor.md: the bot is under 100 gold in 31 to 90 percent of minutes while earning 430 to 1270 gold/h (sales 80 to 85 percent, piles 15 to 20 percent): the shop spends everything. TP scrolls are 48 to 55 percent of spend, mana potions 22 to 34 percent, healing potions about 20 percent, and 83 percent of NAVSOAK trips bought under 30 gold, each burning a scroll that the next visit rebuys.

## What Changes

- Trip floor: a non-emergency town trip starts only when the planned purchase is worth 60 gold or more, or the belt holds no healing potion; a poor character (gold under the kit line) does not rebuy a TP scroll while it holds under 2 healing potions of kit.
- Mana cap: at most 1 mana potion bought per visit below clvl 10 and no `mp belt columns` trip trigger below clvl 10 (knobs, defaults from the ledger).
- Loot bar (last, measure first): `loot.minValuePerCell` default 100 to 20.
- Not changed: boss.brGold (not binding, V), recovery.kitGold (no gold effect, I).

## Impact
Code paths: trip triggers (internal/bot/goal.go), town purchase plan (internal/town/shop.go, beltbuy.go), knobs (internal/env/tuning_town.go, tuning_loot.go), tests. Delta: town-basics ADDED requirements.
Dependencies: docs/knobs.md is owned by m3-routeguard and internal/bot/resource.go by m3-merc until archived: regenerate knobs.md after they merge and say so in the report; after m1-trips and m2-roam are archived or while they are deferred (all touch internal/bot/goal.go); needs the hidden-soak A/B (docs/ab-design.md).
