# SME review: Druid, Barbarian, Amazon, Assassin (2026-11-27)

> status: advice only, no code or data changed. Tags: SRC = read in this checkout this turn (path given), EXP = veteran 1.14d experience, not looked up, UNSURE. No web fetch this turn; skill prerequisites and levels come from `docs/sme/class-<class>.md` (skills.txt-derived tables). Shape follows the Necromancer and Paladin/Sorceress reviews.

## Verdicts

| Class / table | Verdict | One line |
|---|---|---|
| Druid fire (`druid_fire_normal30`, `druid_fire_guide`) | sensible, one ordering defect | Maxroll order (Firestorm 1-5, Boulder 6-7, Firestorm to 11, Fissure at 12); the default profile prefers Molten Boulder over Firestorm |
| Druid tornado (`druid_tornado_endgame90`) | sensible plan, two unreachable skills | Arctic Blast with Cyclone Armor synergy is a clever no-respec bridge; Hurricane (about 20 points) and Grizzly/Dire Wolf have no key or buff slot |
| Barbarian `barbarian_melee_normal30` / `frenzy_guide` | depends | Right skills; default profile lists passive Iron Skin as a buff; Frenzy needs two weapons the bot may not hold |
| Barbarian `barbarian_warcry_endgame90` | NOT sensible as shipped | War Cry is `kind: buff` in `skills.json`, so the cast loop skips it: 20+ points never fire |
| Amazon bow (`amazon_bow_normal30`) | NOT sensible clvl 18+ | Default profile puts Multiple Shot (1 point) before Exploding Arrow (19 points) |
| Amazon multishot endgame / javelin | multishot: 12 points unused clvl 6-17; javelin sensible | MS band starts at clvl 18; javelin plan matches its profile |
| Assassin fireblast / sentry | sensible plan, Death Sentry unreachable | Fire Blast then Wake of Fire is the classic order; Death Sentry has no layout key so the planner never lays it |

## Druid (2a)

- Tables (SRC): fire 12 skill rows follow Maxroll; tornado has 3 banked rows (3-5) and spends 6-9 on Arctic Blast, 12-20 on Cyclone Armor, 21-26 Twister, Tornado from 26. Cyclone Armor x9 is not a waste: Arctic Blast gets +15 percent per Cyclone Armor level and Tornado +9 percent (`docs/sme/class-druid.md` table). Prereqs legal: Arctic Blast no prereq, Cyclone Armor needs it (clvl 12), Twister needs Cyclone Armor (18), Tornado needs Twister (24), Hurricane needs Tornado (30) (SRC).
- Unreachable (SRC `profiles.json` druid/tornado): layout keys f1-f8 are Arctic Blast, Twister, Tornado, Cyclone Armor, Oak Sage, Raven, Spirit Wolf, Summon Fenris. Hurricane (about 20 points, clvl 44-60) and Summon Grizzly have no key, and `buffs` holds only Cyclone Armor, so Hurricane is never cast. It still feeds Tornado synergy, so the points are not dead, but Hurricane is a real damage source. Eight F-keys are all the game gives; Raven (1 point, a clvl 1-5 summon) wastes f6.
- Early weak point (EXP): clvl 1-5 the tornado druid has only Raven plus a staff and 3 banked points; life is about 55-80 and mana about 20. Blood Moor is survivable with potions and the merc. A human would also be here at the weakest point of the class. Firestorm 1 point at clvl 2 would give a ranged hit but is a lasting waste in a no-respec table (UNSURE worth it).
- Default profile order (SRC): druid default band 12 is `Eruption, Molten Boulder, Firestorm`. `druid_fire_normal30` has no `combat` key so it uses it, while the `fire` profile (Eruption, Firestorm, Boulder) is right. Firestorm is 4 mana flat and gets +23 percent per Boulder/Fissure level (SRC class-druid.md), Boulder is 10+ mana.
- Mana (SRC class-druid.md): pool about 64 at clvl 20; Fissure 15, Volcano 25. `mana_heavy: true` for tornado only; fire tables are not flagged and use one mp column. A human belts 2 mana columns for a fire druid from clvl 12 (EXP).
- Merc (EXP): Act 1 Rogue early (ranged cover), Act 2 Desert merc Defiance or Prayer in Normal; Might in Nightmare is UNSURE. Druid has no block, so merc life matters.
- Gaps: no druid row in `levelplan-normal.json` or `act2plan.json` (SRC); `act2plan.go` returns an error naming the class, not a fallback.

## Barbarian (3)

- Tables (SRC): `melee_normal30` Bash 2-4, Axe Mastery at 5 (start weapon is a hand axe, `docs/process/barbarian-support.md`), Double Swing 6-10, Frenzy from 24. `frenzy_guide` is Double Swing 13 then Iron Skin, Frenzy at 25+. Prereqs legal (Frenzy needs Double Throw, SRC).
- `barbarian/default` buffs `Battle Orders, Shout, Iron Skin` and layout f3 Iron Skin (SRC `profiles.json`). Iron Skin is a passive (SRC barbarian-support.md, backlog B8), so the third group does nothing. The table also lists hotkey F3 Iron Skin at clvl 18 (`melee_normal30`).
- War Cry (SRC `classdata/skills.json`): `{"name":"War Cry","kind":"buff","range":0}`. `castfirst.go` line 161 and `rules.go` line 202 skip `KindBuff` when picking an attack. The warcry profile's band `minLevel 30 requires War Cry` therefore lists a skill the fighter never casts; the clvl 31-90 table (about 60 War Cry/Battle Cry points) plays as Bash (UNSURE no other code path casts it; check a log for `cast War Cry`). `profileskills_test.go` only checks `AttackNames`, not the cast loop. War Cry is a physical-plus-stun area attack with its own synergies (Howl, Taunt, Battle Cry, SRC class-barbarian.md), not a buff. Its prereqs are Battle Cry and Battle Orders (SRC), table legal.
- Warcry clvl 1-30 is Bash only, 8 points. Leap 1 at clvl 15 (prereq of a Leap Attack the table never buys) is a wasted point; Find Potion/Find Item are Grim Ward prerequisites, Grim Ward is unsupported (barbarian-support.md). Battle Command is never bought, so the build loses +1 all skills (EXP).
- Melee block (SRC): surround 4 within reach, retreat at 40 percent, surround retreat at 65 percent life. War Cry wants the hero inside the pack; this block was copied from the Zealot and will make the hero retreat exactly when War Cry works (barbarian-support.md B11).
- Frenzy (SRC gearcheck, barbarian-support.md B2): dropped when the off-hand is not a melee weapon; the bot starts with axe + shield. Table clvl 24-30 spends Frenzy x5 plus Double Throw; plan and profile "agree" (agreement.json) only because the profile uses Bash/Double Swing first. A second Hand Axe costs a few hundred gold; it is a policy gap, not a data one.
- Whirlwind/Leap Attack wave-3 rows (SRC): the `whirl` move and the `Whirlwind` press row are on the default profile only; no barbarian table buys Leap Attack, Concentrate or Whirlwind, so they are inert. Safe: they act only once learned. Leap Attack escape (surround 4, life 55) is fine as a planning value (EXP) but unreachable. Whirlwind costs 25+ mana per spin (barbarian-support.md); a Normal pool is 35, so even when taken it needs mana potions, which the Barbarian kit says it does not buy (`classtraits.json` manaNeed 0).
- Early game (EXP): the Barbarian is the easiest class clvl 1-12 (life +2/lvl, 1-hander + shield). Weak spot is stamina: 2 Increased Stamina points only (table row 15), nothing reads stamina (B7).
- Merc (EXP): Act 2 Desert merc Prayer or Defiance in Normal, keep through Nightmare; Barbarian needs no mana-aura merc until War Cry.
- Belt (SRC chores-defaults): health 1, mana 0 for the class. Right for `melee_normal30`; wrong for warcry (`mana_heavy_builds` lists it, so two mana columns from the table flag).

## Amazon (4)

- Profile bug (SRC `profiles.json` amazon default): bands are `minLevel 18 requires Multiple Shot: [Multiple Shot, Exploding Arrow, Fire Arrow, ...]`, `minLevel 12 requires Exploding Arrow: [Exploding Arrow, ...]`. `amazon_bow_normal30` buys Multiple Shot 1 at clvl 7 and Exploding Arrow x19 at 12-30. From clvl 18 the first learned skill wins, i.e. one-point Multiple Shot beats a 19-point Exploding Arrow with Fire Arrow 8 behind it. The table's `main_attack` says Exploding Arrow 12-30.
- `amazon_multishot_endgame90` has no `combat` key either. Its 12 Multiple Shot points at clvl 6-15 are unused until the band at 18; clvl 6-17 the profile uses Magic Arrow 1 (third band). That is the weakest 12 levels of the build.
- Early game (EXP): Amazon is average, bow Magic Arrow/Fire Arrow is fine. The multishot table spends clvl 3-5 on Critical Strike/Inner Sight/Critical Strike (3 points, no damage) while Magic Arrow sits at 1.
- Javelin (SRC): Poison Javelin x12 clvl 6-13, Lightning Bolt x17 to 30 (rows 2-5 banked). Profile bands match the plan (V by agreement of bands and table). Fine for Normal. `chores-defaults.json` has ammo only for arrows; "javelins are picked up from the floor" (SRC). Whether javelin skills consume quantity is UNSURE; read the stack size after 20 casts live.
- Buffs: profile lists Valkyrie; the javelin table never buys it (Valkyrie is clvl 30, plus Decoy chain, SRC class-amazon.md). Inert, harmless.
- Merc (EXP): Rogue Act 1 from Kashya as soon as clvl 5-8, then Act 2 Desert merc; Amazon is fragile (life +2/lvl, no block without a shield).
- Gaps: no amazon rows in `levelplan-normal.json` or `act2plan.json`.

## Assassin (5)

- Plan (SRC `build-assassin_fireblast_normal30.json`): Fire Blast 1-5 (`Fire Trauma` in skills.txt), Claw Mastery at the Den, Burst of Speed (`Quickness`) at 6, Fire Blast to 9, Wake of Fire x20 from 12. Prereqs legal (Wake needs Fire Blast, SRC class-assassin.md). This is the Maxroll order; sensible.
- Death Sentry (SRC `trap.go`, `assassin.go`): `planTrap` only lays skills for which `ready` is true, i.e. `sp.Key != ""` or the skill is the right-hand skill. `profiles.json` assassin layout has no Death Sentry (and no Wake of Inferno, Cloak of Shadows or Mind Blast). The sentry endgame table buys Death Sentry x20 at 56-72 and plans hotkey F5; the profile layout wins, so it is never laid.
- Buff group `["Quickness","Fade"]` is a one-of group (SRC rotation.go: one group index). If the first known skill wins, Fade (bought at 20 in the sentry table, gives resists and damage reduction) is never cast once Burst of Speed is known. A trapper gains nothing from speed (trap speed is cast rate); Fade is the survival tool (EXP, UNSURE on group semantics).
- Mana (SRC class-assassin.md): traps cost 13-20 each and `planTrap` needs 2x cost in pool; `assassin_*` are not `mana_heavy`, belt stays 1 mana column. `classtraits.json` manaNeed 1.
- Weak spot clvl 12-23 on the sentry table: Charged Bolt Sentry x8 (1-7 at L1, weak) replaces Wake of Fire; Fire Blast is still the real damage. Maxroll's Wake leveling does not carry (it respecs at 36-38); a no-respec trapper table is inherently soft here (EXP).
- Merc (EXP): essential: traps need time to place and a trap caster dies when surrounded. Act 2 Desert merc (Defiance or Prayer in Normal) hired at clvl 12 per `act2plan.json` merc row; Act 1 Rogue before.
- Gaps: no assassin rows in `levelplan-normal.json` or `act2plan.json`; the trap planner is UNVERIFIED LIVE (trap.go header).

## Ranked data changes (at most 8 per class)

### Druid
1. `internal/combat/classdata/profiles.json` druid/tornado: `layout` `Raven: f6` -> `Hurricane: f6`; add `["Hurricane"]` to `buffs` after Cyclone Armor. Why: Hurricane (about 20 points) is never cast; Raven is one point. Confidence medium (the Hurricane state id for the recast clock is UNSURE; verify `buff Hurricane` log line).
2. `profiles.json` druid default (id none), band minLevel 12: `Eruption, Molten Boulder, Firestorm, Attack` -> `Eruption, Firestorm, Molten Boulder, Attack`; or add `"combat": "fire"` to `build-druid_fire_normal30.json`. Why: Firestorm 9 points at 4 mana beats a 2-point Boulder at 10. Confidence medium-high.
3. `build-druid_fire_normal30.json` / `druid_fire_guide` clvl 17-18: Arctic Blast 1 + Cyclone Armor 1 -> Eruption +2 (keep Oak Sage). Why: Cyclone Armor 1 absorbs about 40 damage, 2 points of Fissure are worth more at 15-20 mana. Confidence low-medium.
4. `docs/plan/data/potions-belt.json` `mana_heavy_builds.builds`: add `druid_fire_normal30` (and `druid_fire_guide`), `mana_heavy: true` in the files. Why: Fissure 15 + Volcano 25 on a 64 pool. Confidence medium.
5. `build-druid_tornado_endgame90.json` hotkeys: drop F6 Raven and F8 Fenris (not in table) or rebind F6 to Hurricane; keep table and layout in step. Confidence low (cosmetic).
6. `levelplan-normal.json` / `act2plan.json`: add a druid class row (caster archetype like the Sorceress rows). Confidence medium (EXP slices).

### Barbarian
1. `internal/combat/classdata/skills.json` War Cry: `"kind":"buff","range":0` -> `"kind":"melee","range":6` (right hand). Why: `castfirst.go` skips buff kinds, so War Cry is never cast; it is a stun-and-damage nova. Confidence medium-high; verify by `cast War Cry` count above 0 in a clvl 31+ run.
2. `profiles.json` barbarian default: remove `["Iron Skin"]` from `buffs`, `Iron Skin: f3` from layout; remove F3 Iron Skin from `build-barbarian_melee_normal30.json` hotkeys. Why: passive (backlog B8). Confidence high.
3. `profiles.json` barbarian/warcry `melee`: `surround` 4 -> 6, `surroundLifePct` 65 -> 50. Why: War Cry needs the pack on top of the hero; the Zealot numbers retreat early. Confidence low-medium (needs the survival model).
4. `build-barbarian_warcry_endgame90.json` row 15: Leap -> Taunt (or Howl); row 30: Natural Resistance -> Battle Command is not legal before 30 so put Battle Command 1 at clvl 49 and add `["Battle Command"]` to the profile `buffs` (key f5 exists). Why: +1 to all skills including War Cry. Confidence medium; Battle Command needs two casts per barbarian-support.md B3.
5. `build-barbarian_warcry_endgame90.json` rows 2-8: Bash x8 -> Bash x1, Double Swing from 6 (Bash is its prereq), and add `Double Swing` hotkey; the profile's first band already lists it. Why: Double Swing is the better early attacker (EXP); 8 Bash points are only worth it as a stun. Confidence low-medium.
6. `build-barbarian_melee_normal30.json` `main_attack`/profile: either set `main_attack` Bash for 6-23 or put Double Swing before Bash in the default profile band. Why: today the table says Double Swing, the profile fights with Bash. Confidence low (which is better is U).
7. `act2plan.json` barbarian melee row (an archetype copy of the Paladin melee row with lower life floors: about 140 at Halls, 200 at Tombs). Confidence low-medium.

### Amazon
1. `profiles.json` amazon default band minLevel 18: `[Multiple Shot, Exploding Arrow, Fire Arrow, ...]` -> `[Exploding Arrow, Multiple Shot, Fire Arrow, Magic Arrow, Jab, Attack]`. Why: bow table's 19-point Exploding Arrow must win. Confidence high.
2. New `profiles.json` amazon/multishot: bands `minLevel 6 requires Multiple Shot: [Multiple Shot, Magic Arrow, Attack]`, layout from the default; `"combat": "multishot"` in `build-amazon_multishot_endgame90.json`. Why: 12 points at clvl 6-15 unused until 18. Confidence medium-high.
3. `build-amazon_multishot_endgame90.json` rows 3-5: Critical Strike, Inner Sight, Critical Strike -> Magic Arrow +3 (or bank). Why: early damage. Confidence low.
4. `chores-defaults.json` amazon `ammo`: add a javelin trigger for the javelin build (kind javelins, restock under 20 percent) after one live read of whether skills consume quantity. Confidence low (UNSURE).
5. `build-amazon_javelin_normal30.json`: Valkyrie chain (Inner Sight 1 at 6, Slow Missiles, Decoy, Valkyrie 1 at 30) is mostly out of reach at 30; leave. Confidence n/a.
6. `act2plan.json` and `levelplan-normal.json`: add an amazon row (ranged archetype). Confidence medium.

### Assassin
1. `profiles.json` assassin `layout`: add `"Death Sentry": "f8"` (and `Wake of Inferno` if the table adds it). Why: `planTrap` requires a key; the sentry endgame buys Death Sentry x20. Confidence high (SRC trap.go ready()).
2. `profiles.json` assassin `buffs`: `["Quickness","Fade"]` -> `["Fade","Quickness"]`. Why: Fade (resists, damage reduction) is the survival buff for a trapper; speed is not. Confidence medium (group semantics UNSURE).
3. `potions-belt.json` `mana_heavy_builds` and `mana_heavy: true` in `build-assassin_sentry_endgame90.json`. Why: Lightning/Death Sentry 20 mana each, 5-cap per type. Confidence medium.
4. `build-assassin_sentry_endgame90.json` clvl 12-23: Charged Bolt Sentry x8 -> keep 4, Shock Web +2, Fire Blast +2 (Fire Blast gets synergy from Charged Bolt Sentry). Why: soft clvl 12-23 gap. Confidence low.
5. `profiles.json` assassin default: add `Fire Trauma` band with `minLevel 12` giving the planner a melee/standoff stance row (trap lay needs about 20 subtiles); data field exists only on `paladin/hammerdin` today (stance). Confidence low (no stance block for casters yet).
6. `act2plan.json` and `levelplan-normal.json`: add an assassin row (caster archetype, life floor like the Sorceress). Confidence medium.

## Top 5 overall

1. Barbarian: `skills.json` War Cry kind buff -> melee (War Cry never fires).
2. Amazon: default profile band 18 puts Multiple Shot before Exploding Arrow.
3. Assassin: add `Death Sentry` layout key f8 (never laid today).
4. Barbarian: remove passive Iron Skin from the default buff ladder and hotkeys.
5. Druid: tornado profile has no key or buff for Hurricane, and the fire default prefers Boulder over Firestorm.

## Verify live

- War Cry: `cast War Cry` lines in a Barbarian clvl 31+ soak; stun on a pack.
- Amazon: cast log at clvl 18+ shows Exploding Arrow, not Multiple Shot.
- Assassin: `trap skill=Death Sentry` after clvl 56 and `trap learned` lines.
- Druid: `buff Hurricane` line and cast counts; belt layout line after the first shop trip.
- Keep `writes after start: 0` on every record (AGENTS.md crutch rule).

## Gaps in docs/sme

- No verified Barbarian Frenzy rule (does it work with one weapon and a shield); no javelin quantity rule; no Hurricane/Cyclone Armor state ids; Act 2 slices for four classes; FCR/FHR breakpoints per class in a data file the gear scorer reads (EXP: Druid FCR 0/4/10/19/30/46/63/99, Assassin 0/8/16/27/42/65/102/174, Amazon 0/7/14/22/32/48/68/99, Barbarian 0/9/20/37/63/105/200; all UNSURE here, no source fetched). Queue research.
