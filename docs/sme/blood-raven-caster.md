# Blood Raven for casters (Sorceress, Necromancer, Druid), classic 1.14d Normal

Written 2026-10-29 by SMEKB-D. Target: classic LoD 1.14d offline, Normal. Not D2R; D2R-only items are marked `[D2R]`.
Tags: **V** = fetched this session (URL given), **S** = snippet/secondary, **U** = memory or inference.

Sources fetched:
- DW-BR = Diablo Wiki Blood Raven (Wayback of the 2011 page): https://web.archive.org/web/2023/https://diablo2.diablowiki.net/Blood_Raven (V)
- DW-FQ = Diablo Wiki fandom "Sisters' Burial Grounds (Quest)": https://web.archive.org/web/2024/https://diablo.fandom.com/wiki/Sisters'_Burial_Grounds_(Quest) (V)
- DW-FB = same wiki, Blood Raven page: https://web.archive.org/web/2024/https://diablo.fandom.com/wiki/Blood_Raven (V)
- SW = StrategyWiki Sisters' Burial Grounds: https://strategywiki.org/wiki/Diablo_II/Sisters%27_Burial_Grounds (V)
- AG = Almar's Guides: https://almarsguides.com/Computer/Games/Diablo2/Quests/Act1/2SistersBurialGrounds/ (V)
- GB = GameBanshee walkthrough: https://www.gamebanshee.com/diabloii/walkthrough/act1quests.php (V)
- NB = nabou.com walkthrough: http://www.nabou.com/diablo_2/act_1-quest_2.html (V)
- A1Q = Arreat Summit mirror, Act 1 quests: https://classic.hiveworkshop.com/diablo2exp/quests/act1.shtml (V)
- HIR = Arreat Summit mirror, hirelings: https://classic.hiveworkshop.com/diablo2exp/basics/hirelings.shtml (V)
- RB = RankedBoost Blood Raven (D2R-flavored, tables copy the classic data): https://rankedboost.com/diablo-2/bosses/blood-raven/ (V)
- D2DATA = blizzhackers/d2data JSON dumps of the game tables, https://raw.githubusercontent.com/blizzhackers/d2data/master/json/monstats.json (V as data; patch U)
- Our class docs: `docs/sme/class-sorceress.md`, `class-necromancer.md`, `class-druid.md` (skill numbers, V where they say so), `play-economy.md` (merc, prices), `docs/g1-log.md` (our bot's Blood Raven history).

Not reachable: Maxroll classic/D2R class guides mention nothing about her (their leveling pages are generic, V fetched sorc, necro, druid leveling: no "Raven" text), GameFAQs, Fandom direct and PureDiablo/Basin wikis (Cloudflare, V tried). So there is no single caster-specific guide behind this file: tactics are assembled from the boss facts plus general community advice (S) and flagged accordingly.

## 1. The boss, in numbers (Normal)

| Fact | Value | Tag |
|---|---|---|
| Location | Burial Grounds, off the Cold Plains; she stands near the centre, north of the tree; fence has gates NW and SE and many gaps | V SW, DW-BR |
| Monster level (N/NM/H) | 10 / 43 / 88 | V DW-BR, D2DATA |
| HP, Normal | 113 per DW-BR and RB (their stat table); D2DATA `monstats.json` shows `minHP=maxHP=323` for `bloodraven`; I could not reconcile (likely a table scaled by the level multiplier). Plan for 113 to ~330 | V both, conflict U |
| XP, Normal | 181 | V DW-BR, RB |
| Speed | walk 6, run 8 (fast; no knockback needed to keep her at range) | V D2DATA, DW-BR |
| Attacks | ranged bow 1-4 physical (random rare bow); Fire Arrow adds fire; Normal fire arrow about 1-3 phys + 5-11 fire, i.e. roughly 6-14 per hit (NM 19-21+41-47) | V DW-BR |
| Hungry Dead | she spawns at least a dozen zombies, and her skill `Nest` raises more during the fight; Hungry Dead give no XP | V DW-BR, D2DATA (skill `Nest`), mechanics-progression.md |
| Resistances (N) | physical 0%, fire/cold/lightning/poison/magic 50% each | V DW-BR (matches AG: "high resists to all magic types") |
| Chill effectiveness | 50% in Normal (40% NM, 33% H): cold slows her half as long as a normal monster | V DW-BR |
| Stun | "almost impossible to stun" (fast recovery) | V DW-BR, DW-FB says stunning helps if she stops; not reliable |
| Drain effectiveness | 100% | V DW-BR |
| No random affixes; does not regen HP (superunique) | | V DW-BR |
| On death | her spirit sends lightning that kills nearby undead inside the cemetery (not those outside the fence) | V DW-FQ, NB |
| Reward | Kashya offers a free Rogue merc if you have none and lets you hire any time; also unlocks at clvl 8 regardless (HIR "second quest or level 8"); DW-FQ says level 9.  | V HIR, DW-FQ |

Consequences:
- **Cold is NOT better than any other element** against her: 50% resist to everything magical, 50% chill effectiveness. The folk advice "a weapon or charm that adds cold slows her" (DW-FQ, S) is true but weak. Pure physical (0% resist) is her softest spot in Normal: Rogue arrows, a bow, melee, and Necromancer skeleton hits (physical) all do full damage. This is the reason a summoner Necromancer is good against her and a pure-fire druid is not.
- Elemental casters do half damage; Ice Bolt/Fire Bolt at level 3-5 is 3-6 damage per cast (V class-sorceress.md table) so 113 HP at 50% resist is about 30-40 bolts and 100+ mana. This is why a low-level caster cannot burst her.
- Her fire arrow is mild (6-14), so character fire resistance is irrelevant in Normal (no resistance penalty in Normal, U from general D2 knowledge). What kills characters is the arrow count, zombies surrounding them, and the long fight without mana.

## 2. Minimum level and attempts

| Question | Answer | Tag |
|---|---|---|
| Minimum clvl for a caster | clvl 8-10 with ranged or minions; below 6 it is a death trap for a Sorceress/Druid. Humans often skip until Rogue hire at 8-9 or until Cave/Stony Field has given the levels. Our own gate: floor 8 plus kit | S (play-act-route.md: "fight at 6-10"), V gate.go via audit doc, U numerics |
| Necromancer | clvl 7-8 with Raise Skeleton 3+ and Skeleton Mastery 2+ (3 skeletons) and Clay Golem | U (class-necromancer.md entry-gate table) |
| Sorceress | clvl 6-8 with Ice Bolt 4+, 4 mana + 4 health potions | U (class-sorceress.md) |
| Druid | clvl 8-10; summoner plan (Raven, Spirit Wolf at clvl 6) or Firestorm | U |
| Is she required? | No. Not on the critical path to Andariel/Act 2 (Warriv takes you without her); only the Rogue hire and XP | V play-act-route.md (U for the Catacombs claim), A1Q |
| How many attempts | Human norm is 1 to 3 tries with a retreat to town between; a clean first attempt is possible once clvl >= 9 and supplies are ready. Over 3 failed attempts mean "level elsewhere". There is no fetched source with a statistic: U. | U |
| Our bot's history | 181 aborted attempts in a short soak without kills; later gate: 0 attempts when gated; attempt 8 drained gold 3670 to 15 via retreats (fixed by GOLDBURN) | V play-bot-pitfalls.md, g1-log.md |

Practical gate for the bot (recommendation): attempt only with clvl >= 9, 4+ healing potions, 1+ mana potions (caster), a TP tome with 2+ charges, and gold >= the cost of the potion refill that a retreat trip would trigger. If either the first attempt retreats at the same HP threshold twice, set a cooldown of 2 clvls rather than re-entering.

## 3. Common tactics (all classes)

1. **Thin the outer skeletons and zombies first** (outside the fence), so fewer monsters surround you in the cemetery (V DW-FQ, SW, AG, GB).
2. **Lure Hungry Dead out of the fence and fight at the gap**; use bushes and walls (the walls outside the NW entrance) as line-of-sight blockers against her arrows (V SW).
3. **Ignore the Hungry Dead once you are at her** and focus on her: she keeps raising more and they give no XP (V SW, DW-FQ, AG "should be ignored"). Her death kills them anyway (V NB, DW-FQ), so killing zombies is wasted mana.
4. **Keep moving if you need a heal**: the graveyard "has a lot of areas that can confuse the pathfinding", so running around a tombstone cluster breaks her aim (V NB).
5. **Corner her** in the gates or stone walls outside the fence (V DW-FB, SW); once she stops, she dies fast (V DW-FB: "taken down quite quickly if she stops moving").
6. **Hit and run**: she shoots a few times and runs; do not stand in open ground between her and you (V DW-BR "hit and run tactics").
7. **Supplies:** plenty of health potions; potions "can be restocked from Akara" (S, search snippet of a forum). Count 4-8 minor healing (30 gold each), mana 2-4 for casters.
8. **Town Portal:** open a portal near the NW entrance before engaging (U, standard practice) so a retreat is one click, not a walk through zombies. Burial Grounds is next to the Cold Plains, about one area from town; walking is feasible but passes the Hungry Dead.

## 4. Sorceress

Source basis: boss facts above (V), skill numbers from our class doc (V/T there), human practice U/S.

- **Position:** stay in the cemetery's gap or just outside the fence with a wall or the fence line between you and her, not in the open centre. Keep the Hungry Dead behind you in the corridor instead of around you.
- **Skills and damage:** Ice Bolt (3-5 damage at level 1, +15% per level) and Fire Bolt (3-6) both lose half to her 50% resistance; Ice Bolt's chill is half as long (50% chill effectiveness). Frost Nova (clvl 6) does 2-4 at level 1, AoE, and freezes/chills the zombies that reach you; it does not hurt her much but buys breathing room. Charged Bolt is the odd one out: it homes, ignores the aiming problem when she is fast, and does 2-4 x several bolts, but also at 50% resist. None of these one-shots her; she is a mana race. Expect to spend 100+ mana; 4+ mana potions (60 gold each) are the real kit.
- **Static Field** (a Sorceress skill that removes a percentage of the target's current life per cast, with a smaller effect on bosses) is the usual human trick to shorten boss fights, but its clvl and Normal-difficulty effect on a superunique are U here; check `class-sorceress.md` before relying on it.
- **Kiting:** Teleport is not available (clvl 18). Without it, kite by walking around tombstone clusters and the tree (V NB), using cooldown-free bolts; Frost Nova when two or three zombies are adjacent.
- **Rogue merc:** if you hire the Rogue first (clvl 8+, price from the dialog; our clvl 13 Desert reading was 360), she draws arrows and does full physical damage; the Rogue is the human answer to Blood Raven for a Sorceress, so many humans do Cave 1 for levels then hire, then BR (S play-safety.md section 7, V HIR). Catch-22: the free Rogue hire comes after killing her, so humans pay the hire fee at clvl 8 or fight her without one.
- **Minimum:** clvl 8-9, Ice Bolt 4+, 4 mana + 4 health potions, TP tome, belt of 8 slots (Sash 64 gold).

## 5. Necromancer (best case)

- **Why it works:** skeletons do **physical** damage (0% resist in Normal), so BR's elemental resistances are irrelevant; they also draw her arrows (V class-necromancer.md: "weak vs BR's fire arrows if skeletons are not between you and her").
- **Setup:** Raise Skeleton 3+ with Skeleton Mastery 2+ (3 skeletons), Clay Golem at clvl 6 (also physical; golem soaks arrows), Amplify Damage 1 (curse: +damage taken, also lets physical beat resistances; one point is enough, S Icy Veins/Odealo [D2R guides]). Raise Skeleton costs 5 + level mana per skeleton (V class doc).
- **Position:** stay behind the minions at a gap in the fence; let skeletons engage the zombies and BR chase them; curse BR when she stops. Resummon dead skeletons by walking to a corpse.
- **Corpse supply:** the cemetery is full of zombies and skeletons; the dead create corpses (U, standard). Do not use Corpse Explosion yet (not at this level).
- **Minimum:** clvl 7-8, Raise Skeleton >= 3, Mastery >= 2, Clay Golem 1, 4 health potions (U class doc), 3 summons of mana.
- **Blood Golem:** not recommended (S, 1.10 nerf; its HP link makes you take damage). Clay is the standard.
- **Failure mode:** skeletons die to the zombie crowd at the centre (U), so recast between waves; do not run in and make them follow in a conga line.

## 6. Druid (weakest early caster)

- **Elemental trap:** Firestorm/Fire Claws/Molten Boulder are fire (BR 50% resist, but she is not immune). Raven/Spirit Wolf/Oak Sage are physical or summoned.
- **Ravens:** the Raven skill's blind does not work on bosses and superuniques (S, search snippet of a druid skill description), so Ravens are only a distraction/damage source.
- **Spirit Wolf (clvl 6):** physical damage (U), tanky wolves draw arrows; the summoner plan resembles the Necromancer's. Wolves are the better choice than the elemental line for BR.
- **Firestorm:** 4 mana at every level, but 1-2 fire damage per second per tick at level 1 (V class-druid.md) halved by resistance: too weak. Not enough damage until Molten Boulder/Fissure (clvl 6/12 skills).
- **Minimum:** clvl 8-10 with Spirit Wolf; otherwise pay for a Rogue first or level in Cave 1/Stony Field.
- **Weakness:** low mana (+2 per clvl) means the potion bill is higher than a Sorceress's.

## 7. What a bot should do (recommendations)

1. Gate BR on clvl >= 9 (Sorceress/Druid) or 8 (Necromancer with 3+ skeletons), potions >= 4, TP tome >= 2 charges, and an HP/mana reserve. The dominating failure is the repeated retreat loop, not the boss: attempt 8 ate its gold in retreats (V g1-log.md).
2. Approach through the NW or SE gate, clear the outer ring first (the Hungry Dead are slow, V GB "enemies around BR are slow"), open a TP at the gate, then go for her with a plan: skeletons/Clay Golem lead (Necro), or fight at the gate line and kite around the tree (Sorc/Druid).
3. Target selection: BR first, ignore zombies once she is in sight (V SW, AG). `killFirstNames` already has "bloodraven" first (V audit-bot-vs-human.md Z2).
4. Retreat rule: the first HP < 40% -> TP out, rest, return; do not loop more than twice per clvl (the g1 attempt 8 gold bleed).
5. After the kill: talk to Kashya (reward), hire the Rogue if gold allows (the hire is free if you have no merc: V DW-FQ), then Akara is not needed.
6. Alternative: skip BR entirely until clvl 10+ and a merc; she is optional. Her XP (181 at level 10, V RB) is about a normal monster level kill-chunk, not an XP jackpot.

## 8. Open questions

- Real Normal HP (113 vs 323 vs something else): read from memory (`mem` live) or a kill log.
- Exact attack rate (frames) and arrow damage for the specific rare bow roll.
- Whether the free hire at clvl 8 vs 9 differs in 1.14d (HIR vs DW-FQ conflict).
- A real caster-focused guide from the Amazon Basin or PureDiablo forums; they were blocked/unfindable in this session.
