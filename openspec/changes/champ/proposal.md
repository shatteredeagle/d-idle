## Why

The d2-sme verdict (2026-10-31): champions (flag 0x0c, no minions, about 3x life, +50% damage, faster, 3x XP, groups of 3-5) are ELITE, not true bosses. Reading the code showed champions are already Elite=true, Boss=false in combat and in the kill and death records (Boss only for named uniques and monstats bosses), but they get less caution than Elite packs deserve early in Normal, and a champion death was not labelled and could feed the danger table of its base monster.

## What Changes

- combat: `Target.Champion`; a group of 3+ champions in view at clvl < 8 is Avoid.
- resource: `Input.ChampionPack`, knob `combat.championEarlyPct` (default 10 pp) raising the healing and rejuvenation (boss) lines in Normal below clvl 15 while a champion is in range.
- bot: additive `killerChampion` in deaths.jsonl; `killerBoss` stays named boss / superunique only; a champion death adds no danger-table weight.
- No change to the death-rule or soak-gate specs (no number there changes).

## Impact

Shared with the Game.exe path (internal/combat, internal/bot, internal/resource): all changes are stricter-only, so each needs a G1/G2 check. Notes: docs/champions.md.
