# Champions are elite, not bosses

Decision (d2-sme, user go 2026-10-31): a champion (monster flag bit 2, no minions; about 3x life, +50% damage, faster, 3x XP, groups of 3-5) is ELITE, never a boss. Tags: the numbers below are the SME's rules of thumb (EXP), not measured; the flag bits are UNVERIFIED live (docs/d2-1.14d-memory.md).

## What `Boss` gates (read before the change)
- `combat.TargetDanger`: Boss is pack danger 2 (DangerElite), a plain Elite only 1 (DangerShaman). Danger 2 stops melee fallback and the 'weak pack' staff swing (resource/attack.go), triggers merc-dead abort, retreat-rest (RetreatDanger) and the mana-potion 'Best' tier. Danger 1 already arms the boss-rejuv line.
- `Hostiles`: Boss is exempt from the Hell Avoid rules and the death-weight Avoid; resource thin-out and the merc 'Skip' use Elite||Boss; necro/assassin/static-field logic use Boss||Elite.
- bot: kill events (`EvKill.Boss`, goal stats bossKills), `killerBoss`, `combat.DeathInfo.Boss`.

## What is true now
- `Target.Boss` only for a named unique or monstats boss; a champion is `Elite` + `Champion`. Kill events and `killerBoss` already follow the same rule (flag bit 3 / monstats), so no kill or boss-stat change was needed.
- New: `Target.Champion`; `Input.ChampionPack`; `DeathInfo.Champion`; additive `killerChampion` in deaths.jsonl (never with `killerBoss`). A champion death adds no death-spiral danger weight for its base monster.
- Extra caution (Boss caution Elite lacks): knob `combat.championEarlyPct` (default 10 pp, 0 = off) adds to the healing line (55) and the boss-rejuv line (50) while a champion is in range, Normal, clvl < 15. A group of 3+ champions in view at clvl < 8 is Avoid.
- Not done: random rares without a name still count as `IsUnique` (boss in kill events and `killerBoss`); left as is.

## Shared code, G1/G2
internal/combat, internal/resource, internal/bot are shared with the Game.exe path. All changes are stricter-only (earlier drinks, avoid a group, extra labels) and need a G1/G2 check: soak a low-level Normal run through a champion pack (the avoid rule can skip a pack that blocks a route).
