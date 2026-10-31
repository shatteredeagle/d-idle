# SME-grill digest, topic 0003: merc strategy and Act 2 entry

How: the coordinator-side agent (smegrill0003) interviewed the `d2-sme` agent in rounds, no human in the loop, one SME agent continued across rounds. Grilling style: pointed questions, numbers demanded, vague answers challenged once, recommended answer per question. Tags on SME claims: SRC (a named source, repeated here; NOT re-fetched by us unless marked V), EXP (veteran memory), UNSURE. Classic 1.14d; D2R-only facts are flagged. Nothing here is verified by us: see [claims-to-verify.md](claims-to-verify.md). Written 2026-10-30.

Caveat on SRC: the SME's "SRC" mostly points at our own docs/sme/*.md (which cite wiki pages, themselves S/EXP). Treat SRC-via-our-docs as S, not V.

## Round 1: the free Blood Raven Rogue

Verdict: our gate is a sound floor, but clvl 9 is soft; retreat loop matters more than the number; the paid-hire ban is sensible; claim at once.

| Q | SME answer | Tag |
|---|---|---|
| Kit gate (clvl 9 / 4 HP / 2 MP / tome 2+ / 250 g) | Fine as a floor. SME's own numbers: clvl 10 and 6 mana potions, because a fully elemental Sorceress does half damage to her (50 % to all magic, 50 % chill) and her physical resist is 0 %. | SRC docs/sme/blood-raven-caster.md s1-2 (which cites Diablo Wiki, Wayback; itself EXP) |
| What kills a Sorceress | The Hungry Dead crowd boxing you in plus a long mana race, not her Fire Arrow (about 6-14 a hit). SME doubts the "skeleton archers" premise: the doc says she raises zombies via Nest. | SRC same doc; archers UNSURE |
| Tactics | Fence gap NW or SE with a wall between you and her; ignore zombies once she is in sight; Frost Nova (clvl 6) only to peel; retreat below 40 % life. Static Field on a superunique in Normal: UNSURE. | EXP |
| What makes SME wrong | Her real HP is 323, not 113 (our doc has both): then a clvl 9 Sorc needs 150+ mana and the gate should be clvl 11. | UNSURE, our doc conflicts |
| Bot rule | Attempt 1 at clvl 10; a retreat means cooldown +2 clvl, not a retry; log her LifePct at each retreat so the HP question is answered by data. | EXP |
| Rogue skills | Fire Arrow or Cold Arrow plus Inner Sight; Inner Sight is a cast, not an aura; the Rogue has NO aura. | SRC docs/sme/mercenary.md s1 (hireling.tsv) |
| Reward Rogue variant and level | UNSURE: whether the reward fixes the variant or lets you pick. Bot should read rows live and take Cold if offered. Starting level UNSURE. | UNSURE |
| Rogue value | Small: level 3-7 does 1-3 base plus bow, about 45 HP at level 3, about 8 HP per level; a decoy plus 5-15 per arrow; roughly doubles a clvl 10-13 Sorc's early output. Useful from the reward to about clvl 14 or Andariel; swap to the Desert Merc on Act 2 arrival (human norm). | SRC mercenary.md s2/s4/s5 (S) |
| Paid Kashya hire before reward | Sensible; hiring a Rogue first forfeits the free one. Cost of waiting is about zero: levels 1-7 have no merc for anyone. Whether a DEAD owned merc forfeits: UNSURE. | SRC mercenary.md s2 (Amazon Basin Rogue Scout); dead-merc UNSURE |
| Safest no-merc path | Blood Moor, Den of Evil, Cold Plains, Cave 1 to clvl 9-10; BR attempt at clvl 10 with Fire Bolt 4+. Fallback: stuck at clvl 8-9 with 4+ failed runs and gold over 500, then a paid Rogue beats a death loop. | EXP |
| Claim and gear | Claim at once (menu opens straight after the kill; risk is dying or leaving first). Do not pay revive for a Rogue under level 15, re-hire in Act 2. Placeholder gear: Cap, Quilted or Leather Armor, Short Bow or Hunter's Bow; any socketed bow bonus; Str/Dex reqs per item UNSURE, read the merc inventory. | EXP; revive formula SRC mercenary.md s3 (S, D2R-ish) |

Gaps named by the SME: reward Rogue variant/level, dead-merc forfeit, real BR HP, clvl 8 vs 9 hire opening.

Coordinator note (challenge pending, put in round 2): d2d probe saw 8 hire rows at clvl 8 after the kill (docs/d2d-probes.md), so "hire opens at clvl 8" needs a source.

## Round 1 challenges (answered in round 2)

| Challenge | SME answer | Tag | Status |
|---|---|---|---|
| Blood Raven HP 113 vs 323 | Our doc cites 113 (Diablo Wiki) and 323 (blizzhackers d2data monstats.json); cannot reconcile; read max life from the live unit. | UNSURE | UNRESOLVED, cheap live probe |
| "Hire opens at clvl 8" | Rule is "second Act 1 quest done OR clvl 8" (Arreat hirelings page via mercenary.md s2). Our d2d sample (clvl 8 + BR dead) cannot separate them; test Kashya at clvl 8 with BR alive. Treat the quest as the proven path. | SRC Arreat via our doc (S) | partly resolved |
| Dead merc forfeits reward | Believes the reward is for "no merc employed" and a dead merc still counts, never saw it confirmed. d2d source says yes in LoD (docs/d2d-probes.md), original unverified. | UNSURE (d2d source V-d2d) | UNRESOLVED on original |

## Round 2: Act 2 Desert Mercenary and economics

Note: the SME fetched nothing this round; "SRC" is our own docs/sme/mercenary.md. I pushed it to fetch real sources from round 3 on.

| Q | SME answer | Tag |
|---|---|---|
| Aura order, Normal (rows Prayer / Defiance / Blessed Aim) | Sorc, Necro, Druid: Defiance first, then Prayer (the merc is the frontline, his defence decides how long he lives; Prayer's heal is tiny at low aura level; Prayer first only if the bot is short of potions). Paladin: Defiance. Barbarian: Defiance in Normal, Might in NM. Amazon bow/jav and Assassin: Blessed Aim first, then Defiance. | EXP |
| Contradiction with us | Our table puts Blessed Aim LAST for bow/jav Amazons ("survival first"); SME puts it FIRST for Amazon bow/jav and Assassin. | judgment, see prebrief |
| NM Holy Freeze / Might | Sound in principle, but NM hire costs about 7900 x factor and the Normal merc levels with you; swap only with gold over hire + 1.5 x revive. | SRC mercenary.md s2 (hireling.tsv factor, V in our doc) + EXP |
| Merc level and aura level at clvl 15-25 | Merc level UNSURE: 1.10 ranges 9-13 Normal; 1.11+ "within 5 of the player" (Amazon Basin, S). Expect about 10-25. Aura level: no source, the window does not show it. | UNSURE |
| Hire price formula | UNSURE; do not assume level^2; fit price against merc level from d2d rows (d2d: 130-240 at clvl 8, 295-510 at clvl 20). | UNSURE |
| Revive price | min(int(lvl^2/2)*15, 50000) from a D2R page; believes it matches 1.14d, medium confidence. 750 / 1680 / 3000 / 4680 at merc level 10 / 15 / 20 / 25. d2d source agrees (docs/d2d-probes.md, V-d2d), never seen on the original. | EXP (SRC mercenary.md s3, D2R page) |
| Swap Rogue to Desert | Conditions: gold >= price + 600 reserve, reached Lut Gholein and past Andariel, and Rogue dead or under level 12. Not rushed: Desert is a frontline tank that suits a Sorc better. Strip the Rogue first; helm and armor transfer, the bow does not. Rogue (Cold Arrow + Inner Sight) probably kills better up to ~clvl 15, worse after. Andariel is before Act 2 and bosses do 10x damage to mercs: neither merc survives her well. | EXP; 10x SRC ARR via mercenary.md s3 |
| Iron Wolf (Act 3) | Normal rows Fire/Cold/Lightning cast spells, no aura, factor 1000-1500; never beats the Desert merc for a bot; do not plan for it. | SRC mercenary.md s1/s5; EXP |
| Levelling and death | Levels by XP, capped at the player's level, no XP at the cap. Level loss on death UNSURE for 1.14d; PD says no XP loss in NM/Hell 1.09/1.10; revive keeps level, XP, gear. Assume none. | SRC ARR (S) / UNSURE |

## Round 2 challenges, settled by fetching (answered in round 3-4)

The SME read Amazon Basin (Hireling, Rogue Scout, Desert Mercenary, Blood Raven), the Arreat mirror (classic.hiveworkshop.com/diablo2exp/basics/hirelings.shtml), PureDiablo (purediablo.com/strategy/diablo-2-guide-the-mercenary-guide), a Daum hireling.txt column guide (m.cafe.daum.net/rpgguild/ISk/26) and our extracted tables (monstats, superuniques, hireling, weapons, armor, runes, levels under /tmp/claude-1000/gx/cache). We did not re-fetch any of it.

| Question | Result | Tag | Status |
|---|---|---|---|
| Blood Raven Normal HP | Table monstats: minHP = maxHP = 323, level 10, exp 151, 50 % fire/cold/lightning/magic resist. The wiki's 113 and 181 XP are another version. Whether 1.14d scales on top: unknown; read max life live. | SRC our monstats table (V-table per our docs) | mostly resolved: 323, so the gate should assume the hard version (clvl 10-11, 6 mana) |
| Dead merc forfeits reward | Amazon Basin says only "not if you already employ a mercenary"; no source on dead. | UNSURE | UNRESOLVED on original |
| Hire price formula | Daum: Gold column "compounded by mlvl", no formula. | UNSURE | UNRESOLVED (fit from d2d rows or read live) |
| Revive formula | Only the 50,000 cap (Arreat, AB) and "peak about level 80" (PD). lvl^2/2*15 is still a D2R page plus d2d source. | UNSURE for 1.14d | UNRESOLVED: read one live price |
| Merc Str/Dex | Arreat: hirelings must meet all Str, Dex and Level requirements; hireling.tsv has Str, Str/Lvl, Dex, Dex/Lvl per merc, so the merc's OWN stats. Rogue at lvl 10 about Str 44 / Dex 59, Desert lvl 15 about Str 67 / Dex 49 (SME applied Daum's "per-level columns are in 8ths"; UNSURE for aura growth). | SRC Arreat + table; per-level maths UNSURE | resolved on principle (matches our merc.reqStrDex rule), numbers to verify live |

## Round 3: merc gear

| Q | SME answer | Tag |
|---|---|---|
| Item requirements (Normal bases, from tables, no clvl req) | Short Bow Dex 15; Hunter's Bow Dex 28; Long Bow Str 22 / Dex 19; Skull Cap Str 15; Helm Str 26; Quilted Armor Str 12; Leather Armor Str 15; Hard Leather Str 20; Ring Mail Str 36; Spear Dex 20; Trident Str 38 / Dex 24; Bardiche Str 40; Voulge Str 50; Brandistock Str 40 / Dex 50; Pike Str 60 / Dex 45; Partizan Str 113 / Dex 67 clvl 23. | SRC our weapons/armor tables; Partizan row looks wrong to the coordinator (our docs/merc.md uses Partizan as a strip example; Arreat lists Partizan Str 80), verify |
| Placeholder gear | Rogue: Hunter's Bow (socketed), Hard Leather, Skull Cap. Desert: socketed Bardiche or Voulge, Ring Mail, Helm. Chipped/flawed gem or rune in a socket; for mercs life steal beats damage. Do not give a lvl-12 Desert a Pike or Brandistock. A bow on a Desert is useless, Rogue uses bows only. | EXP; PD for life steal (S) |
| Long-term runewords | Stealth (Tal Eth, armor); Insight (Ral Tir Tal Sol, polearm; Meditation aura to the party; req clvl 27 that the merc must meet); Treachery (Shael Thul Lem); Infinity (Ber Mal Ber Ist) not realistic early; Fortitude, Duress, Peace not realistic in Normal. Rune drop sources in Normal: UNSURE, treasureclassex not checked; Sol, Shael, Lem too rare to plan around. | SRC runes table for recipes; drops UNSURE |
| Bot inventory rules | Never sell items on the merc; strip before replacing (re-hire loses gear); dead merc strip: UNSURE; before Act 2 only cap plus armor are worth the effort; equip only when Str/Dex/level met; helm and armor first, weapon second. | SRC Arreat/AB (S); EXP |

## Round 4: death, revive, route guard

| Q | SME answer | Tag |
|---|---|---|
| Revive policy | Because the price is unverified: reviveGoldMult 2.0 (gold >= 2 x price + potion kit); max 2 revives per hour; stop reviving above 2 deaths per hour; too expensive when price > 40 % of gold at clvl 12-25; re-hire (lose gear) when revive > about 3 x hire price and the gear is worth no more than a Hunter's Bow plus Ring Mail. At merc lvl 20 revive ~3000 if the formula holds, re-hire (Normal Desert about 360 live) is far cheaper: so read ONE real revive price first. | EXP; 360 V-live per mercenary.md |
| Boss damage | Bosses deal 10x damage to a merc; Andariel, Duriel, Radament will kill him, that is normal. Mercs follow portals, waypoints, acts, difficulties; missing after TP/WP: wait 10 s, then walk to the portal or change area. | SRC Arreat via mercenary.md s3 |
| ROUTEGUARD clvl 20 | Normal monster levels from levels table: Tamoe Highland 8, Monastery Gate 8, Outer Cloister 9, Barracks 9, Jail 10. Gate monsters (bighead, corruptrogue, sk_archer, skmage_ltng): ranged archers and the lightning mage are what a no-merc caster dies to. clvl 20 looks TOO HIGH; the axis is probably ranged hits plus lightning, not clvl. Verify from death logs (monster at the last hit). Human at clvl 14 no merc: Jail levels or Countess runs for XP, then Monastery with 4 healing, TP, lightning resist in mind. Minimum: life 150+, tome, 4 healing, 2 mana. | SRC levels table; interpretation EXP/UNSURE ("bighead" unidentified) |
| Act 2 Normal danger (no merc) | Levels: Sewers 13-14, Halls of the Dead 12-13, Dry Hills 15, Far Oasis 16, Lost City 17, Valley of Snakes 18, Claw Viper Temple 14, Maggot Lair 17, Arcane Sanctuary 14. Dangerous: Claw Vipers (poison), Valley of Snakes, Sand Raiders, Maggot Lair (egg spawns). Safer: Dry Hills, Halls of the Dead, Sewers. Needs Frozen Armor, 8-slot belt, rejuv/antidote against poison. | SRC levels table; EXP; not source-checked |

Contradiction with our plan: ROUTEGUARD (plan-v2 11.2) uses clvl 20 or a merc as the entry bar for the Monastery road; SME says the monsters are level 8-10 and clvl 20 is probably too high. The counter-evidence is our own data: Soaky died 3x there at clvl ~14 without merc and with a thin kit (plan-v2 11.2). The grill should take the SME hypothesis (ranged/lightning, kit-bound, not level-bound) as a probe, not a fact.

## Round 5: Act 2 entry and Andariel

Fetched by the SME: Arreat Act 1 and Act 2 quest pages (classic.hiveworkshop.com/diablo2exp/quests/act1.shtml, act2.shtml), Amazon Basin Andariel and Duriel pages, our monstats/levels tables. Challenge answered: Partizan Str 113 / Dex 67 / clvl 23 is the exceptional-tier base and matches our weapons table; it is not an error (my earlier doubt withdrawn).

| Q | SME answer | Tag |
|---|---|---|
| Andariel numbers (Normal) | Level 12, HP 2562 (min = max), Fire -50, Lightning 50, Cold 50, Poison 80 (resistance, not immunity in the Normal row). Fire Bolt does 150 %; Ice Bolt is the weak choice. Melee 76-238 per hit. Whether 1.14d scales the HP: UNSURE. | SRC our monstats table; Amazon Basin confirms fire -50 and level 12 |
| Gate | Knob 15 is too low for a merc-less Sorceress: SME says clvl 18, life 180+, Fire Bolt 10+, Frozen Armor; clvl 15 only with a living merc. Kit: ask for 3 antidotes and a tome with 2+ charges (our kit: 4 healing, TP, 3 mana, 2 antidotes is sound). No resist threshold in Normal (no penalty). | EXP (clvl number); the SME gave no source for 18 |
| Tricks | Fight from the stairs and step up to heal; kite along walls; Static Field and Frost Nova peel mobs, not for her. Catacomb mobs level 11-12. | EXP; levels table |
| Skip Andariel? | No: her death opens Warriv's Act 2 trip (EXP, the Arreat page does not say). Our own d2d probes agree that Warriv "go east" opens after her kill (docs/d2d-probes.md, V-d2d). Optional for Act 2 progress: Den, Blood Raven, Cain, Forgotten Tower, Countess, Tools of the Trade. Rewards: Den skill point (UNSURE), Blood Raven free Rogue, Tools of the Trade imbue (Arreat). | EXP + V-d2d |
| Act 2 waypoints | Lut Gholein, Sewers 2, Dry Hills, Halls of the Dead 2, Far Oasis, Lost City, Palace Cellar 1, Arcane Sanctuary, Canyon of the Magi. | EXP |
| Quest flow | Duriel needs the Cube (Halls of the Dead), shaft (Maggot Lair), amulet (Claw Viper Temple), assemble in the Cube, open the real tomb; the Summoner (Arcane Sanctuary) gives the Canyon and the real tomb. Radament optional (skill point). SME order for XP and safety: Sewers, Halls of the Dead, Far Oasis, Maggot Lair, Viper Temple, Palace, Arcane, tomb. How to find the real tomb: UNSURE (our docs/act2.md handles it through `FindTomb`). Arcane death: re-enter by waypoint, recover the corpse. Coordinator note: in real D2 the Staff is the Horadric Staff made from shaft + amulet (no Cube needed for the Staff itself; the Cube is a separate quest needed to combine them): the SME's wording is loose; verify against Arreat. | EXP; wording flagged |
| Duriel (Normal) | Level 22, HP 4757, cold 50 (not immune), fire 20, lightning 20, melee 140-180. Use fire/lightning. Sorc minimum clvl 22-24, life 300+. TP in the lair: UNSURE, plan for none. 1-3 tries expected. | SRC our monstats table; EXP for the minimum |

## Round 6: quit/reload and the merc (no source found)

| Q | SME answer | Tag |
|---|---|---|
| After Save and Exit | Merc loads with you in town; a dead merc stays dead until revived; quitting in the field returns you to your act's town; whether the merc is at full HP: UNSURE. | EXP / UNSURE |
| Boats and acts | Merc follows through (Arreat). Kashya reward survives a quit (quest state saves): UNSURE, test. Our d2d probe says the reward works after reload (V-d2d). | SRC Arreat; V-d2d |
| Boss kill then quit | Quest flag saved at the kill; Warriv's Act 2 offer available after reload (EXP). d2d probe agrees (V-d2d). | EXP + V-d2d |
| Poison, dead merc on exit | Poison can kill an unhealed merc in town (Arreat); Save and Exit with a dead merc costs nothing (EXP). | SRC Arreat; EXP |

Coordinator note: no round-6 claim has a real source, d2d is the only data we have, and the original-game behaviour is open. The quit-safety reference gap in docs/sme is still there.

## Round 7: the SME's own UNSURE list (self-audit, no fetches; confidence figures are the SME's gut numbers, EXP)

| # | Claim | Matters because | What settles it | SME confidence |
|---|---|---|---|---|
| 1 | Blood Raven Normal HP is 323 (table), not 113 (wiki) | decides the kit gate (323 HP at 50 % resist is a 150+ mana fight for a Sorc) | read max life from the live unit | 70 % |
| 2 | A dead owned merc forfeits the free reward | whether a dead Rogue can still be replaced free | live test: let the Rogue die, talk to Kashya | 50 % |
| 3 | Revive = min(int(lvl^2/2)*15, 50000) in 1.14d | reviveGoldMult, when to re-hire | read one real price at a known merc level | 55 % |
| 4 | Hire price formula | the swap gold gate | fit price vs merc level from d2d rows and one live window | 30 % |
| 5 | Offered merc level at clvl 15-25 ("within 5 levels", 1.11+) and aura level | Defiance vs Prayer value | read the window level; aura level needs a tooltip or stat read | 35 % / 25 % |
| 6 | Andariel needs clvl 18 and life 180+ for a no-merc Sorc | whether the floor 15 is lethal | one live attempt with life-per-phase logging | 60 % |
| 7 | Static Field helps on a superunique; skeleton archers in BR's pack | wasted code if wrong | skills.tsv and monster list | 25 % / 40 % |
| 8 | Desert aura per class | tuning, low stakes | playtest | 40 % |
| 9 | The Normal Desert merc survives Act 2 at all | the swap's value | live Act 2 run counting deaths | 45 % (about 120 HP at lvl 9, bosses deal 10x) |

EXP items the SME would NOT bet money on: Den of Evil gives a skill point; the real tomb is identifiable on arrival; Duriel's lair TP rule; "desert heat does nothing"; which monster kills at the Monastery Gate (names from the table, the attribution is a guess); rune drop locations in Normal; reload puts the merc beside you at full HP.

## Round 8: what a pro would change in our plan (the 5 weakest points, ranked)

Policy given to the SME: merc-policy spec plus plan-v2 11.2 route guard plus the Andariel floor and M9.

| # | Weak point | Pro does instead | Evidence that would convince |
|---|---|---|---|
| 1 | ROUTEGUARD clvl 20 or a merc: the Monastery road is monster level 8-10, clvl 20 is far above the need, the no-merc bot grinds ~5 extra levels where XP is already thin | clvl 14 with life 150+, 4 healing, TP, 2 mana; fight at the gate corner; retreat at 40 % life | death log with monster names and damage before death |
| 2 | Andariel floor 15 with a 2-death cap is too low unless a merc lives | wait for clvl 18-20, a living merc or full potions, 3 antidotes | one live attempt, life per phase |
| 3 | "Never pay Kashya first" has an uncertain upside: the Rogue is a decoy (45 HP at lvl 3) | policy stays, add a bailout: after 3 failed BR attempts or clvl 12, pay ~100-150 gold | the dead-merc reward test |
| 4 | Revive multiplier rests on an unverified formula | revive only when the price is under 30 % of gold; re-hire when revive > 3 x hire (Desert hire about 360) | one real revive price |
| 5 | Act 2 last in the work order while Andariel is the gate: Act 2 entry never ran live | one early Act 2 probe on a throwaway character | one live probe |

SME's three JUDGMENT calls for the user: risk appetite (deaths per boss before fallback; Andariel cap 2), merc timing (paid Rogue at clvl 8-9 vs wait for the free one), readiness bar (clvl 15 vs 18-20 for Andariel). Everything else is a fact a probe or fit can settle.

## Resolved vs unresolved (end of interview)

Resolved enough to plan on (S, from our extracted tables and two wiki pages): Blood Raven HP 323 / level 10 / 50 % elemental resist in our monstats (but a wiki says 113); Andariel level 12, 2562 HP, fire -50, poison 80; Duriel level 22, 4757 HP, cold 50 not immune; Normal Desert aura rows are Prayer, Defiance, Blessed Aim and NM adds Thorns, Holy Freeze, Might (hireling.tsv); Rogue has Inner Sight (cast), no aura; merc Str/Dex use the merc's own stats (Arreat plus the table); Iron Wolf not worth planning.
Unresolved (all in [claims-to-verify.md](claims-to-verify.md)): dead merc and the reward, hire and revive formulas, offered merc level and aura level, the no-merc Andariel number, what kills at the Monastery Gate, quit/reload merc state, Act 2 real tomb, Normal rune sources.

## Process notes (honesty)

- The SME answered rounds 1-2 without fetching anything; I challenged once, and it then fetched Amazon Basin, Arreat mirror, PureDiablo, a Daum column guide and read our TSVs. Rounds 6-8 are EXP only. Nothing here is verified by us.
- Disagreement with our own docs: Blessed Aim position (SME first for bow/jav Amazon and Assassin; spec: last); ROUTEGUARD number (20 vs SME 14); Andariel floor (15 vs SME 18 no-merc); the 2-mana kit (SME 6 mana potions for BR); clvl 9 vs 10-11 for BR.
- The SME's Act 2 "Cube then Staff" wording was loose; docs/act2.md has the coded flow (staff from shaft + amulet, Cube separately).
