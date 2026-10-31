# Class research index (docs only)

Each class file follows `docs/plan/leveling.md`: build choice, level ranges, skills/stats, gear, merc, dangers, mana, and a "what the bot needs" table compared with the Sorceress. Evidence tags: (V) fetched, (S) snippet, (U) unverified. Most guides are Maxroll D2R Season 14, so D2R-only items are flagged inside each file.

- [amazon.md](amazon.md): javelin line (Poison Javelin to Lightning Fury); ranged and easy to cast, but slow, lightning-immune Hell, ammo and pets. Rank 5.
- [assassin.md](assassin.md): Fire Blast/Wake of Fire then Lightning Sentry traps; safe at range, needs a trap planner and lightning-immune fixes. Rank 4.
- [barbarian.md](barbarian.md): War Cry melee build; low mana needs but inverts the ranged design, physical immunes, stamina and weapon-swap buffs. Rank 6.
- [druid.md](druid.md): Fire (Fissure/Volcano) with pets and buffs; gear- and cooldown-heavy, fire immunes. Rank 7.
- [necromancer.md](necromancer.md): Teeth/Bone Spear then Summoner; safe minions, curses fix immunes, but corpse and summon upkeep is complex. Rank 3.
- [paladin.md](paladin.md): Holy Fire then Blessed Hammer (Hammerdin); safest and few immunities, needs aura switching and directional casts. Rank 2, **recommended second class**.
- [ranking.md](ranking.md): criteria matrix for all seven classes and the recommendation.
- [party.md](party.md): party mechanics (XP, monster scaling, caps), synergies, compositions, `party.Role` mapping and additions.

## Resolved in the follow-up (see tags in each file)
- Players X: life and XP `(n+1)/2` (V Arreat), damage +6.25% per step (V D2R Maxroll; S for 1.14d), `/players` rules (V). party.md section 1.
- Party auras and warcries reach party members, with radii (V Arreat). paladin.md, barbarian.md, party.md.
- Charged items recharge at the town blacksmith (V Arreat). Mercenary revival cap 50,000 (V Arreat). assassin.md, ranking.md.
- Amazon leveling bands (V Maxroll amazon-leveling-guide, respec at 38). amazon.md.

## Still open
1. Is the 6.25%/player damage step in 1.14d, and does Players X change monster density? Needs a two-instance live test (U).
2. Exact charges and Teleport level of an Ormus Staff of Teleportation, and the mercenary hire fee formula (U; read from the game when hiring/buying).
3. Whether aura/warcry effects cross a party member in a different area; actual in-game radius check (U).
4. D2R-only items (Sunder charms, Flame Rift, Bone Break, Crack of the Heavens sunder, Patch 2.4 Unraveler change) are flagged in the class files; none are verified for 1.14d, plan without them.
5. Ranking scores remain judgement (U); revisit after the Sorceress runs live and skills.txt/monstats.txt are dumped.
