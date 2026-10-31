# docs/sme: Diablo II knowledge base (subject-matter notes)

Target: classic LoD 1.14d, offline single player, softcore. D2R-only facts are flagged `[D2R]` inside the files. Written 2026-10-29/30 by SMEKB-A..E, SMEAUDIT, MERCRESEARCH and the D2DHARVEST agents; the d2-sme agent's veteran judgment is tagged EXP. Index of everything else: [../INDEX.md](../INDEX.md).

## How to read the tags (reliability)
- **V** = a page or table the agent fetched or computed (URL or "T" = our install's MPQ tables); **V1** = carried from an earlier SME file, not re-fetched; **S** = snippet or secondary; **U** = memory or inference; **C** = computed by us from a V formula; **B** = our own bot data.
- **EXP** = veteran judgment of the d2-sme agent, not a fact. **SRC** = the user or the SME agent said so (about 85 to 99 % agreement), still needs a live check.
- Several notes cite Maxroll D2R guides (V1): D2R balance differs from 1.14d in places; the files flag it where known.

## d2decomped-derived notes (secondary sources)
[d2decomped-findings.md](d2decomped-findings.md) and [d2decomped-findings-2.md](d2decomped-findings-2.md) summarise the d2d reimplementation (https://github.com/psi29a/d2decomped, GPL-3.0). They are a strong SECONDARY source: d2d is a reimplementation checked by its authors against the original, not the original. Anything that matters (death rule, quest quit-safety, merc, waypoints) is acceptance-tested on the original game. GPL rule: paraphrased only, with file-path citations, no code or long passages copied (docs/decisions.md 2026-10-30). The Mephisto quit-safety claim in them contradicts the user's report and is unresolved ([../BACKLOG.md](../BACKLOG.md)).

## Files
| file | covers |
|---|---|
| [mechanics-combat.md](mechanics-combat.md) | damage, hit chance, resist, aura and curse mechanics (1.14d) |
| [mechanics-breakpoints.md](mechanics-breakpoints.md) | FCR/FHR/FBR/IAS tables per class, computed and cross-checked |
| [mechanics-progression.md](mechanics-progression.md) | XP, levels, area levels, difficulty, resists |
| [mechanics-items-drops.md](mechanics-items-drops.md) | treasure classes, magic find, gold, shops (Basin drop tables predate 1.13) |
| [economy-data.md](economy-data.md), [play-economy.md](play-economy.md) | vendor prices and stock, gambling, shopping and chores |
| [play-act-route.md](play-act-route.md), [play-safety.md](play-safety.md), [play-bot-pitfalls.md](play-bot-pitfalls.md) | human route through five acts, what kills characters, what automation gets wrong |
| [class-amazon.md](class-amazon.md), [class-assassin.md](class-assassin.md), [class-barbarian.md](class-barbarian.md), [class-druid.md](class-druid.md), [class-necromancer.md](class-necromancer.md), [class-paladin.md](class-paladin.md), [class-sorceress.md](class-sorceress.md) | per class: skills, no-respec build line, gear, bot notes |
| [builds-leveling.md](builds-leveling.md), [builds-gear.md](builds-gear.md), [builds-endgame-casters.md](builds-endgame-casters.md), [builds-endgame-melee.md](builds-endgame-melee.md), [builds-endgame-amazon-farming.md](builds-endgame-amazon-farming.md), [builds-open-questions.md](builds-open-questions.md) | leveling builds clvl 1-40, runewords/cube/vendors (ladder-only runewords are blocked offline), endgame builds, farming, resolved questions |
| [pace-tables.md](pace-tables.md) | human and speedrun pace, basis of the soak clvl floors (cells are U-derived) |
| [mercenary.md](mercenary.md) | hire rules, auras, revive, free Rogue (V from our hireling.tsv); conflicts with d2d notes on rejuvenation |
| [blood-raven-caster.md](blood-raven-caster.md) | how casters kill Blood Raven (plan-v2 10.1 builds on it; her HP is unsettled) |
| [early-gear-policy.md](early-gear-policy.md), [potion-policy.md](potion-policy.md) | d2-sme reviews turned into bot policy (EXP, code-cited) |
| [audit-bot-vs-human.md](audit-bot-vs-human.md) | SMEAUDIT: where our bot differs from a veteran (partly fixed since; see ../plan/STATUS.md) |
| [d2decomped-findings.md](d2decomped-findings.md), [d2decomped-findings-2.md](d2decomped-findings-2.md) | d2d notes: death/gold, quests, merc formulas, memory addresses, combat, AI, waypoints (secondary) |

Known gaps: no quit-safety reference from the original game, no death-cost-per-difficulty table, no clvl 1-8 buy order beyond early-gear-policy (plan-v2 s9).
