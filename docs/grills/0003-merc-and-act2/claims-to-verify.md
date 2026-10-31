# Claims to verify (grill 0003 fact base)

Feeds a later research agent. Source of every claim: the SME interview in [sme-grill.md](sme-grill.md), 2026-10-30. SME tags: SRC (named source, but mostly our own docs/tables, so S), EXP (veteran memory), UNSURE. Priority: P1 = changes a policy number or a go/no-go; P2 = tuning; P3 = nice to know. Verify source types: DATA = extracted 1.14d tables (hireling, monstats, levels, weapons), WIKI = Arreat/Amazon Basin/PureDiablo, LIVE = original Game.exe probe on a throwaway character, D2D = d2d source or probe (not acceptance for the original), CODE = decompiled or open reimplementation.

| # | Claim | SME tag | Why it matters to our policy numbers | Verify with | Prio |
|---|---|---|---|---|---|
| 1 | Blood Raven Normal HP is 323 (level 10, 50 % fire/cold/lightning/magic resist), not the wiki's 113 | SRC our monstats; wiki conflicts; SME 70 % | kit gate clvl 9 / 2 mana (spec) vs SME clvl 10-11 / 6 mana | DATA monstats + one LIVE read of her max life | P1 |
| 2 | A dead owned merc still forfeits the free Rogue | UNSURE (d2d source says yes, V-d2d) | ShouldClaimFreeRogue, the dead-Rogue replacement path | LIVE: let the Rogue die, talk to Kashya; CODE | P1 |
| 3 | Revive cost = min(int(lvl^2/2)*15, 50000) in 1.14d | EXP 55 %, D2R page; d2d source agrees | reviveGoldMult, 30 % of gold rule, re-hire vs revive | LIVE: read one price at Kashya/Greiz at a known merc level; CODE | P1 |
| 4 | Hire price formula (d2d: 130-240 at clvl 8, 295-510 at clvl 20) | UNSURE 30 % | swap gold gate, price x hireGoldMult | DATA hireling.tsv Gold column + LIVE window rows at 3 clvls | P1 |
| 5 | Andariel Normal: level 12, 2562 HP, fire -50, poison 80 (resist, not immune), melee 76-238 | SRC our monstats + Amazon Basin | Fire Bolt vs Ice Bolt, boss.andarielMinLevel | DATA + LIVE one attempt | P1 |
| 6 | A no-merc Sorceress needs clvl 18 / life 180+ for Andariel (floor 15 only with a living merc) | EXP 60 % | boss.andarielMinLevel 15 and the +2 dead-merc rule, M9 | LIVE attempt with life-per-phase and poison logging; pace tables | P1 |
| 7 | Monastery road Normal monster levels 8-10; clvl 20 is too high; real killers are ranged (archers, lightning mage), kit-bound not level-bound | SRC levels table; interpretation EXP/UNSURE | ROUTEGUARD route.monasteryMinLevel 20 | DATA levels/monstats + deaths.jsonl monster names (Soaky 3 deaths) | P1 |
| 8 | Merc Str/Dex check uses the merc's OWN stats; Rogue lvl 10 about Str 44 / Dex 59, Desert lvl 15 about Str 67 / Dex 49 (per-level in 8ths) | SRC Arreat + hireling.tsv; per-level maths UNSURE | merc.reqStrDex, MeetsReq, placeholder gear | DATA hireling.tsv + LIVE equip probe (docs/merc.md probe 3) | P1 |
| 9 | Offered merc level is within 5 of the player in 1.11+ (1.10 ranges Desert 9-13); aura level not shown in the window | S Amazon Basin; 35 % | Defiance vs Prayer value, swap rule | LIVE window rows at clvl 15/20/25; DATA | P2 |
| 10 | Desert aura order: Defiance first for Sorc/Necro/Druid/Paladin/Barb; Blessed Aim FIRST for Amazon bow/jav and Assassin (our table: Blessed Aim last) | EXP 40 % | AuraPrefs table | WIKI AB "What merc for build X"; low stakes, playtest | P2 |
| 11 | Free Rogue's variant (Fire/Cold) and starting level; whether the reward fixes the variant | UNSURE | what the Rogue gives, swap timing | LIVE after BR kill; D2D src kashya_merc | P2 |
| 12 | Hire window opens at clvl 8 OR after quest 2 (Blood Raven) | SRC Arreat (S) | paid-Rogue fallback timing, BR-skipped rule | LIVE: Kashya at clvl 8 with BR alive | P2 |
| 13 | Normal Desert merc survives Act 2 poorly (about 120 HP at lvl 9, bosses deal 10x); swap worth it only if Rogue dead or under level 12 and gold >= price + 600 | EXP 45 % | ShouldReplace rule, merc.replaceAtAct2 | LIVE Act 2 run counting merc deaths | P1 |
| 14 | Boss kill then Save and Exit keeps the quest flag and Warriv's Act 2 offer; Kashya reward survives a quit | EXP; V-d2d for Blood Raven and Andariel | quit-to-town shortcut, M9 flow | LIVE read quest word before/after reload (docs/plan 10.5 probe) | P1 |
| 15 | After reload the merc stands beside you in town; a dead merc stays dead; field quit returns to town | EXP / UNSURE | merc-missing handling, ROUTEGUARD merc state after reload | LIVE: reload with Rogue alive/dead, read merc unit | P2 |
| 16 | Duriel Normal: level 22, 4757 HP, cold 50 (not immune), fire/lightning 20; clvl 22-24, life 300+; TP inside the lair unknown | SRC table; minimum EXP | Act 2 readiness bar, Duriel floor | DATA + LIVE; Amazon Basin Duriel page | P2 |
| 17 | Act 2 Normal area levels (Sewers 13-14 ... Maggot 17, Valley 18) and no-merc dangers (Claw Vipers poison, Sand Raiders, Maggot eggs) | SRC levels table; danger EXP | Act 2 area order, antidote count | DATA levels + WIKI; LIVE death logs | P2 |
| 18 | Real Tal Rasha tomb identification | UNSURE | Act 2 flow (`FindTomb`) | WIKI (Arreat/AB Tal Rasha tombs), CODE preset | P3 |
| 19 | A dead merc's gear can be stripped | UNSURE | ShouldReplace StripFirst on a dead merc | LIVE | P3 |
| 20 | Normal rune drop sources for Tal, Eth, Ral, Tir, Sol, Shael, Lem | UNSURE | whether any merc runeword is realistic offline | DATA treasureclassex + droprate sim | P3 |

Notes for the researcher: items 1, 3, 4, 8 can mostly be closed from our extracted tables without the game; 2, 6, 7, 13, 14 need a live probe or soak data. Do not treat the SME's "SRC our table" as independent confirmation of a wiki claim.
