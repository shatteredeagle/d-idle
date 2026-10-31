# Act 1 chain to Andariel: what each step needs, code vs d2-sme (ANDYCHAIN, 2026-10-31)

> status: research input for grill 0003 and cards m3-merc, m3-routeguard, m3-act1chain, m3-andariel. Decides nothing. Tags: V = repo code, docs or logs; SME = d2-sme 2026-10-31 (from memory, did not read our docs; EXP/U inside it); U = unverified. Numbers with an owner (knobs, gates) are cited, not owned here: docs/knobs.md, openspec/specs/merc-policy, soak-gates.

## 1. The chain as the code has it (V: internal/task/campaign.go NormalAct1, gate.go, route.go, act1.go, plandata progression-normal.json)

Order: Den, Blood Raven, [Tools, Twr: optional], Andariel, Warriv hop; a free-Rogue claim step is added (`WithRogueClaim`). The Cain chain exists (`cainNodes`) but is off (`Act1Options.Cain`, not needed for Act 2). The planner takes the first unsatisfied entry whose floor is met, else levels.

| step | requirement in code today | on d2d today |
|---|---|---|
| Den of Evil | floor 1 (5 without scroll and potions), 4 potions + TP; Akara talk, clear area 8, reward | works to clvl 10 in soaks (V: Viewy, STATUS); skill point reward flow V offline |
| Blood Raven | clvl 9 caster (`boss.brCasterLevel`), 4 healing, 2 mana, tome >= 2 charges, GOLD >= 250 (`brGold`), 3 attempts then wait +1 to +2 clvl | kill + Kashya grant V on d2d via debug kill (docs/d2d-probes.md); never fought by the bot: Viewy had 15-100 gold, so the gold gate holds her for ever |
| free Rogue | claim after BR dead; merc policy off by default (`merc.enabled` 0, tuning_merc.go); revive price never observed | UNVERIFIED end to end; revive S only |
| Tools (Smith) | optional; skipped for a Sorceress (`ImbueSkipWhy`: class has no use) | n/a, skipped |
| Forgotten Twr / Countess | optional; clvl 14 (gate) and tower guard clvl 20 unless a living merc (`route.towerMinLevel` 20, `towerNeedsMerc` 1) | blocked (Viewy: "no merc, clvl 10 < 20") |
| Cain / Inifuss / Cairn Stones | off (`Cain` false); would need the tree (class U), stone order (brute force, U), `cmd interact` on objects | Dark Wood "unreachable/nav-stuck" in Viewy (nav, not quest); not needed |
| Monastery road | areas 7 and 26-37: clvl 20 without a living merc, nothing with a merc (`route.monasteryMinLevel` 20) plus kit (4 healing, 2 portal charges, 2 mana) | blocked (Viewy) |
| Andariel | knob floor 15 (+2 dead merc and gold < 1000); gate has `Route: true`, so the effective floor is `max(15, route floor)` = 20 for a character with no merc; kit 4 healing (`minPotions`), TP, 3 mana, 2 antidotes; script in internal/boss/act1.go | never fought on either backend (STATUS); Catacombs 4 reached only by the road |
| Warriv hop | talk "go east" (option name U), done when act >= 2 | not exercised |

Chain deadlock (V by reading the code, U live): the Rogue needs Blood Raven, Blood Raven needs 250 gold, gold income is near zero, and without the Rogue the road to Andariel needs clvl 20, so the real Andariel floor today is 20 (the knob's 15 only applies with a living merc). Viewy at clvl 10 after 10.6 h shows the first link failing.

## 2. d2-sme view for a clvl 10 Sorceress (SME; consulted 2026-10-31, answered from memory, our docs not read)

| step | SME minimum |
|---|---|
| Den | clvl 6, 4+ minor healing, 2 mana, TP; ~100 gold |
| Blood Raven | clvl 11-12 (EXP), Frozen Armor 1 point, 6 healing, 3 mana, tome; merc preferred; ~300-400 gold (U) |
| Cain, Inifuss, Cairns, Cow | skip all (Act 2 needs only Andariel dead) |
| Forgotten Twr / Countess | skip (runes not needed); if kept: clvl 18 and a merc |
| Smith | only if met on the road |
| Monastery road | clvl 16-18, vitality, 8+ potions, antidotes, merc alive, ~800 gold |
| Andariel | clvl 18, life >= 250 (U), 8 healing, 3-4 mana, 3 antidotes, merc alive, ~600 gold |

Other SME points: XP cliff at clvl 14-16 (Act 1 mlvl tops at 10-12, Andariel 12; consistent with docs/sme/audit-bot-vs-human.md A1: clvl 17-18); killers: Countess archers and fire arrows at 0 mana, Twr ghosts/hawks in corridors, Catacombs stalkers, gargoyles and archers at 0 mana, Andariel poison nova plus melee; the Rogue is not needed to survive but for an unattended bot effectively required (a dead one stays dead; revive roughly 100-500 gold, U); Static Field leaves Andariel at about 1/4-1/3 life (U), lightning is her weakest resist. Build: add 1 point each Frozen Armor and Warmth, rest into Fire Bolt, Static Field, Vitality; Teleport waits for clvl 18. Gold: 15-100 at clvl 10 means the bot is not selling drops (a human has 1-2k); gate Blood Raven on potions rather than a gold figure.

Caution: this SME answer is one pass from memory; its numbers are EXP/U, and its BR floor of 11-12 contradicts our merged 9 (docs/sme/blood-raven-caster.md, brkill 1477431, which the repo tags U too).

## 3. Agreement and disagreement (the grill numbers)

| gate | code | SME | verdict |
|---|---|---|---|
| Monastery road, no merc | 20 | 20 (SME earlier in the roadmap: 18) | agree (this pass); the roadmap row says 18: SME is not stable, the grill decides |
| Monastery road, merc alive | any clvl | 18 | DISAGREE: no floor with a merc is too loose (prebrief 10: 14) |
| Twr | 20 or merc | skip; else 18 + merc | disagree on merc-only; agree it should be skipped for a Sorceress |
| Andariel | 15 (+2) | 18 with merc, 21 without, life >= 250 | DISAGREE, and the effective no-merc floor in code is already 20 |
| Blood Raven | clvl 9 + 250 gold | clvl 11-12, gate on potions not gold | DISAGREE on level (U) and on a gold gate; the gold gate is what blocks M3 today |
| Andariel kit | 4 healing, 3 mana, 2 antidotes | 8, 3-4, 3 | disagree (kit thin) |
| revive | gold >= 2 x price | revive only if affordable | agree in kind; price unobserved (probe) |

## 4. Table and smallest cards

| step | code requirement | SME requirement | blocked today by | smallest fix card |
|---|---|---|---|---|
| Den | clvl 1-5, kit | clvl 6, kit | nothing | none |
| Blood Raven + Rogue | clvl 9, kit, 250 gold | clvl 11-12, kit, ~300 gold | no gold (items not sold or piles not taken), gold gate, merc policy off | gold income card; BR gate on kit with gold as soft rule |
| Twr | 20 or merc | skip | guard (fine) | skip by default for non-melee: no card needed |
| Monastery road | 20 or any merc | 18 with merc | no merc, so 20 | m3-routeguard (after grill) |
| Andariel | 15 +2, effective 20 | 18 / 21, life 250 | road guard, never fought | m3-andariel on a throwaway |
| hop | act >= 2 | - | unexercised | m3-act1chain |

Five smallest cards, in order (one agent each, not written):
1. Gold income: sell every non-junk drop on each town trip and take gold piles, with a log of gold and items sold per trip (SME gold point; audit M1 area; unblocks the BR gold gate, Viewy 15-100 gold).
2. Blood Raven gate: make gold a restock-trip trigger rather than a hard gate, level and potion count from the grill (m3-merc input), so the Rogue is actually reached.
3. Merc on by default on d2d (`merc.enabled`), claim, revive budget with the probed price (m3-merc).
4. Route guard numbers with a `withMerc` floor per guard, Twr skipped for a Sorceress, effective Andariel floor shown in one log line (m3-routeguard, after grill 0003).
5. Andariel fight on a cheat-levelled throwaway, 3 with a Rogue, 1 without, antidote on poison (m3-andariel, unchanged), then Act 1 order and Warriv hop evidence (m3-act1chain).

## 5. Unknowns for a hidden d2d probe (U; each is half a slot-hour at most)

1. Gold per hour at clvl 8-12 with selling on: log gold and items sold per town trip over 2 virtual hours (d2dsoak, one seed).
2. Merc revive price at Kashya for the free Rogue and Rogue level at clvl 10 (debug kill BR, claim, kill the merc, read the dialog price).
3. Free Rogue grant with a fresh no-merc character through the bot's claim step (only the already-granted state was run live, docs/d2d-town.md).
4. Andariel HP, resists, poison dose in d2d Normal and Static Field's actual floor (SME U): m3-andariel records them.
5. Catacombs 4 reachability by the bot (waypoints at Catacombs 2; Jail, Cathedral stairs) and Warriv "go east" option row.
6. Dark Wood "unreachable" is a nav bug, not needed for Andariel; probe only if Cain is wanted.
