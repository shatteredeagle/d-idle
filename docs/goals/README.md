# docs/goals: the next goal layer (research, docs only)

1. `catalog.md`: 127 concrete objectives in 14 families (level, quests per act, bosses, areas, waypoints, resources, items, build, runes, safety, difficulty) with parameters, tiers, prerequisites, effort (U) and risk; says which existing goal type already covers each.
2. `execution.md`: the step model, per-family decomposition onto nav/town/combat/loot/leveling, quest mechanics for all 5 acts in 1.14d (NPCs, items, areas, bosses, rewards, required vs skippable, pitfalls), failure recovery, and the missing capabilities.
3. `tracking.md`: state we can read (verified vs unverified), measurement per family, proposed Goal/Progress/Event/Observation additions, dashboard views, and a 24-item live spike checklist.
4. `planner.md`: dependency graph, campaign templates (Normal playthrough, Hell Baal runs, rune farm), a default campaign for a new character, and an auto-planner pseudo-code.
5. Sources: Blizzard Classic Battle.net quest pages (V) and PureDiablo rewards (S); the rest is recollection or repo docs and is tagged (U).

## Open questions
- Where are the quest bits and waypoint bits in 1.14d memory (PlayerData pointers are a D2BS-layout guess)? Spike items 1-7 in tracking.md.
- Is Den of Evil's skill point and respec granted in every difficulty or Normal only (S purediablo vs repo data and the "skip Den in Hell" note)?
- Is the Cairn stone order random per game? Is Cain's rescue needed for the Act 2 scroll dialog? Is the Summoner skippable for Duriel?
- Exact Act 3 Golden Bird chain (dropper, Meshif or Alkor turn-in) and exact area ids (recollection; dump levels.txt at setup).
- Cow level prerequisite (Baal or Diablo once in Normal?) and Normal-only rewards (Lam Esen +5 stats).
- Effort estimates are guesses; the first soak run should calibrate them (planner learns medians).
