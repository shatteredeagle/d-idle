# Decision points T0 to T2: d2-sme review (answer from memory)

> status: input for the decision-point store spec ([decision-point-store-spec.md](decision-point-store-spec.md), catalogue in its section 5) and the gate "decision-point catalogue final" of [ADR 0016](../adr/0016-automation-ladder.md). Asked on 2026-11-05 as the grill 0005 follow-up ([outcome](../grills/0005-automation-ladder/outcome.md)).

Tags: **SME (U)** = answer of the d2-sme agent from veteran memory; it looked nothing up and did not read docs/sme/. Every number is unverified and must be checked (monster levels per area, TP and potion prices, tome capacity, life of a clvl 5 hero) before the band edges are frozen. **U** = my wording. Classic 1.14d, Act 1 Normal.

Question put to the SME: for each T0 to T2 decision point, which game-state fields would a veteran weigh, which bands make two situations the same decision, which risk class fits; add the escape-kit safety field. Context given: streak rule (N 2/3/5 by risk), mid-fight questions never pause the flow (act now, ask after), questions fade, fixed answers with "not sure", confirm on risky points.

## 1. Escape-kit field `kit` (SME (U))

One banded field, three values, never raw counts:

| value | meaning |
|---|---|
| OK | at least 1 TP scroll, or a tome with 3 or more scrolls, AND at least 2 healing potions in the belt (any tier) |
| THIN | a TP is available but only 1 scroll, OR exactly 1 healing potion, OR gold under 150 with no TP |
| NONE | no TP and no healing potion, or no way out; with no scroll, gold under 100 (one scroll costs about 100) is NONE |

Mana is not folded into `kit` (it would split situations too finely). Casters (Sorceress, Paladin profiles) get a separate mana-potion band, defined in the class profile data, not in code (U, build-flexibility rule).

## 2. Bands (SME (U); Act 1 Normal)

| field | bands |
|---|---|
| life | above 70 / 40 to 70 / 25 to 40 / below 25 percent. Unsure: a clvl 5 sorceress has about 60 life, so one hit can skip a band. Hardcore: 25 to 40 is the "act now" band |
| gold | below 100 / 100 to 500 / 500 to 1500 / above 1500. 100 = one TP; 500 = roughly a tome plus potions |
| level gap (area monster level minus hero level) | 0 or less / +1 to +3 / +4 to +6 / +7 or more. Edges matter only under clvl 10 |
| free pack rows (of 4) | 0 / 1 / 2 or more. Town trigger at 1 or fewer unless a TP is held |
| stash | under 50 percent / 50 to 85 / over 85 / full |
| death streak | 1 / 2 / 3 or more |
| run time vs expected | under 0.7x / 0.7 to 1.3x / over 1.3x |
| belt state | full / half / under 2 healing potions |
| gear band (boss safety) | one composite of resists and life, 3 values |

## 3. Table (SME (U) for all cells; "change" marks what differs from spec section 5)

| decision point | context fields | bands | risk class | default if unanswered (act now) | escape-kit field |
|---|---|---|---|---|---|
| t0.retreat.drink | life band, area class, pack danger class; ADD kit | life 4, kit 3 | normal (hardcore: risky) | drink if a potion is held, else portal if kit OK, else run | high: decides drink vs portal |
| t0.retreat.lethal | pack type (closed list, e.g. Fallen shaman, Gargantuan Beast, Bone Archer cluster); ADD kit | list | risky | avoid | medium |
| t0.area.toohard | area class (not raw area name), level gap band; ADD kit | gap 4, kit 3 | risky | skip, go to the next lower area | high: NONE forces "lower area" |
| t0.retreat.repeat | area class; ADD kit | area class | normal | leave | high |
| t1.restock | gold band, belt state | gold 4, belt 3 | safe | yes if gold allows | it is the kit refill itself |
| t1.return | area class; ADD kit | kit 3 | safe if kit OK, else normal | no return until kit is not NONE | high |
| t1.keep.kind | item kind class; ADD gold band; no exact item names | kind class | normal; charm or unique: risky (selling is irreversible) | keep | none |
| t1.town.trigger | free-rows band; ADD kit | rows 3, kit 3 | safe | go if rows 1 or fewer OR kit THIN | high |
| t1.tp.lost | gold band | gold 4 | normal (risky in a dangerous area) | buy if gold 100 or more, else walk | defines kit |
| t1.stash.full | stash band; ADD gold band | stash 4 | normal | sell, never drop | none |
| t2.repeat | run kind; ADD kit | kit 3 | safe | again if kit OK | high |
| t2.failstreak | run kind, death streak band | streak 3 | risky | stop at 3 or more (hardcore: stop at 1) | medium |
| t2.slowrun | run kind; ADD time-vs-expected band | time 3 | normal | continue on the first slow run, switch on the second | none |
| t2.bossunsafe | boss, gear band | gear 3 | risky | skip | medium |

## 4. Missing points a veteran expects (SME (U))

- t0.unique.fight: a champion or unique pack met mid-area, fight or skip (the most common non-boss killer in Normal Act 1).
- t0.leave.safe: a pack class the hero cannot see coming (fast chaser, ranged): leave by portal or run. Unsure whether `t0.retreat.lethal` covers it.
- t0.shrine: take or avoid a shrine in the path (unsure).
- t1.repair (safe, nearly always yes), t1.identify (scroll or Cain on a magic item), t1.gamble (rare, T2 or later), t1.potions.buy (only if a tier choice exists).
- t2.reset.game: leave and restart for a fresh area (free offline, safe).
- t2.mercenary: hire or revive (unsure; belongs to merc policy).

Fields NOT to use (split situations too finely): raw life, mana, gold, level; exact monster names outside the lethal list; time of day, game or run count; individual potion counts; exact inventory; item affixes.

## 5. Top 3 findings (SME (U))

1. Add `kit` to t0.retreat.drink, t0.area.toohard, t0.retreat.repeat, t1.return, t1.town.trigger, t2.repeat. The same life band is a different situation with 0 potions and no TP.
2. Hardcore raises the risk class one step on the retreat and area points; t2.failstreak stops at 1 death there (a death ends the hero). Softcore classes are fine as proposed.
3. Mid-fight points need an "act now" default that depends on kit alone, never on a rule being learned; the answers feed later contexts, not the live decision. This matches ADR 0016 item 10 and 11.

## 6. Notes for the spec (U, mine)

- Risk class is fixed per point (ADR 0016 item 13), but the SME wants it to depend on hardcore and on kit. Both are not allowed to vary one point's class; the way to keep the class fixed is a separate point id per mode, or a hardcore flag in the context key with the stricter class chosen for that point. Decision for the user or the store card.
- Hardcore is unbuilt until the T4 acceptance (ADR 0016 item 9), so the hardcore cells are noted, not built.
- Band edges are SME (U); verification list: monster levels per Act 1 area, TP and potion prices, tome capacity, life of a low-level hero (data from the repo's tables, not memory).
