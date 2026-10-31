# Prebrief for grill 0003: merc strategy and Act 2 entry

status: ready (SME-grilled, not yet user-grilled). Written 2026-10-30 for the interviewee (the user) and the grilling model. How the session runs: [../../process/grill-workflow.md](../../process/grill-workflow.md). Fact base and full interview: [sme-grill.md](sme-grill.md); load-bearing claims queued for source checks: [claims-to-verify.md](claims-to-verify.md). Tags: V = read in a linked doc or table (that doc has its own tags), S = secondary (wiki, SME citing our docs), EXP = veteran memory of the d2-sme agent, U = unverified. Classic 1.14d only; D2R facts are flagged. NOTHING below about the original game is verified by us live.

## 1. The decision in one sentence

How does the bot handle mercenaries and when is it allowed into Act 2 (Monastery road, Andariel, Act 2 entry): free Rogue then Desert Mercenary with a revive economy, or play unmerc'd until a higher clvl, and what readiness bar (clvl, life, kit) opens Andariel and the Monastery?

Existing record: [ADR 0006](../../adr/0006-free-rogue-merc.md) (take the free Rogue, never pay Kashya first, V), [merc-policy spec](../../../openspec/specs/merc-policy/spec.md), plan-v2 11.2 route guard. This grill confirms or changes the numbers in them.

## 2. Already DECIDED (do not re-ask)

- Take the free Blood Raven Rogue, never pay Kashya before the quest (user, 2026-10-30; ADR 0006, V).
- Merc policy code exists and is wired behind `merc.enabled` (default 0) ([merc.md](../../merc.md), V).
- Classic 1.14d, offline single player; cheats only on throwaway characters via the ledger.

## 3. Options (the real choices)

| | Option | Meaning |
|---|---|---|
| A | CURRENT plan | ROUTEGUARD clvl 20 or a living merc for the Monastery road; Andariel floor 15 (+2 if the merc is dead and gold < 1000), kit gate on top; Act 2 last in the work order. |
| B | SME-leaning | Guard by kit and life rather than clvl (clvl 14, life 150+, 4 healing, TP, 2 mana, retreat at 40 %); Andariel floor 18 without a merc, 15 only with a living merc; bail out of Blood Raven after 3 failures or clvl 12 by paying Kashya ~100-150; one early Act 2 probe on a throwaway. |
| C | CAUTIOUS | Keep A's clvl 20 guard but fix the XP gap (more Act 1 areas), Andariel at 18-20 always, no Act 2 work until a live Andariel attempt succeeded. |
| D | PROBE FIRST | Do not pick numbers now: run 5 cheap live probes (section 5) and let data set the knobs; keep A until then. |

## 4. Verified-ish facts

| Claim | Number | Source | Tag |
|---|---|---|---|
| Free Rogue is granted when Kashya's menu opens after BR dies, gold unchanged; hiring any merc first forfeits it (d2d: even a dead merc) | 0 gold | [d2d-probes.md](../../d2d-probes.md) s1, d2d source `kashya_merc` | V-d2d; original U |
| Reward survives a reload without re-killing BR (BR respawns) | - | d2d-probes.md s3 | V-d2d |
| Blood Raven Normal: level 10, 323 HP, 50 % fire/cold/lightning/magic resist (wiki says 113) | 323 | our extracted monstats via SME | S (table), conflicts with a wiki; gate depends on it |
| She kills a Sorceress by the zombie crowd and a long mana race, not her arrows; SME: clvl 10, 6 mana | - | sme-grill.md round 1 | EXP |
| The Rogue has no aura (Inner Sight is a cast); Fire/Cold Arrow; about 45 HP at lvl 3, +8 per level; a decoy plus 5-15 per arrow | - | [sme/mercenary.md](../../sme/mercenary.md) (hireling.tsv) | V-table / EXP |
| Normal Desert hire: aura rows Prayer, Defiance, Blessed Aim; Nightmare adds Thorns, Holy Freeze, Might; Hell has no Might/Holy Freeze (1.14d; D2R differs) | 360 gold seen at clvl 13 live | hireling.tsv, [town.md](../../town.md) | V |
| Hire price per row (not level^2): clvl 8 130-240, clvl 20 295-510 on d2d | - | d2d-probes.md s4 | V-d2d |
| Revive cost min(level^2/2*15, 50000): 750 / 1680 / 3000 / 4680 at merc lvl 10/15/20/25 | - | d2d source + a D2R page; SME 55 % | S, never seen on either backend |
| Merc must meet Str/Dex/level on his OWN stats (Rogue lvl 10 about Str 44 / Dex 59) | - | Arreat + hireling.tsv via SME | S; live probe pending |
| Placeholder gear fits a lvl 12-20 merc: Hunter's Bow (Dex 28), Hard Leather (Str 20), Skull Cap (Str 15), Ring Mail (Str 36), Bardiche (Str 40) | - | our weapons/armor tables via SME | S |
| Andariel Normal: level 12, 2562 HP, fire -50 (Fire Bolt 150 %), cold/lightning 50, poison 80; Duriel level 22, 4757 HP, cold 50 | - | our monstats + Amazon Basin | S |
| Monastery road Normal monster level 8-10 (Gate: archers, lightning mage); Soaky died 3x there at clvl ~14 with no merc and a thin kit | - | levels table via SME; [plan-v2](../../plan/plan-v2.md) 11.2 | S / V (deaths, our logs) |
| Andariel's death opens Warriv's Act 2 trip, also after reload | - | d2d-probes.md s3 | V-d2d |
| Death toll: no floor, a clvl 12 death with 1100 gold drops all of it; stash is safe | - | plan-v2 section 10.6 | V-d2d |
| Quit/reload merc state, boat and act follow, dead-merc strip | - | SME | EXP / U |

## 5. NOT yet proven (settle by probes or research, not by judgment)

1. BR real HP and clvl gate (probe: read her life live).
2. Dead merc forfeits the reward on the ORIGINAL (probe: let the Rogue die, talk to Kashya).
3. One real revive price and one hire window per clvl (fits the formulas).
4. Merc Str/Dex own-stats rule (one equip probe).
5. What actually kills at the Monastery Gate (monster name from deaths.jsonl).
6. A no-merc Andariel attempt with life-per-phase logging (the plan's M9).
7. Merc position and state after reload; Act 2 flow never run live; Desert merc survival in Act 2.

## 6. What the user must JUDGE (research cannot settle these)

1. RISK APPETITE: how many bot deaths per boss attempt are acceptable before falling back (Andariel cap is 2) and is a death-loop in Catacombs 4 worse than slow XP? A death costs a corpse run plus gold; XP is cheap.
2. MERC TIMING: wait for the free Rogue (cost about zero, risk a thin clvl 8-10 with no merc) or pay about 100-150 at clvl 8-9 (loses the reward forever). The SME's bailout (3 failed BR attempts or clvl 12) is a compromise.
3. ACT 2 READINESS BAR: clvl 15 with a merc and kit, 18-20 without one, or "kit and life, not clvl"? Slow-and-safe (more XP, fewer deaths, lower throughput) against fast-and-learn (earlier data).
4. EFFORT PRIORITY: Act 2 stays last in the work order, or an early throwaway probe (SME says do the probe early because Andariel is the gate)? And how much merc-gear work is worth it before Act 2 (SME: cap and armor only)?
5. HOW MUCH LIVE-PROBE BUDGET to spend on claims 1-6 of section 5 before the numbers are trusted (each is 20-60 minutes of live time).

## 7. Questions the grill must force

1. What would make you reopen ROUTEGUARD's clvl 20? Name a measurable trigger (deaths per 1000 min at the Monastery with the kit met?).
2. What is the lowest clvl at which you accept the first unattended Andariel attempt without a merc, and who watches it?
3. If the free Rogue dies at clvl 11, do we revive (about 1000 gold if the formula holds), re-hire, or play without?
4. At what gold level does the bot buy the Desert merc in Normal, and is a Normal Desert merc even worth it for a Sorceress (SME: 45 % that he survives)?
5. Which unproven item in section 5 do you want probed BEFORE the policy numbers are fixed?

## 8. Risks

- The SME's numbers (clvl 18, clvl 14, 6 mana, floor 18) are experience, not data; the SME answered the first two rounds without fetching and has no source for most Round 6 claims.
- Fidelity: free Rogue, revive and quit behaviour are V-d2d only; d2d is not acceptance for the original.
- Over-guarding costs throughput (clvl 20 gate grinds thin Act 1 XP); under-guarding costs death loops.

## 9. Recommendation of the prebrief author (read after you answer)

U, judgment of the agent that ran the SME-grill, not a fact. Read this only after you answered section 6.

- Pick D then B: run probes 1-5 of section 5 (all cheap, throwaway character), then apply the SME's kit-and-life guard and floor 18 without a merc, 15 with. Keep the Rogue policy; add the 3-attempt/clvl-12 bailout only if probe 2 shows a dead Rogue still forfeits (then a paid Rogue is the only way to get a merc when BR is stuck).
- Do not lower ROUTEGUARD on SME opinion alone: the evidence is monster levels from a table against 3 real deaths at clvl ~14; get the killer's name from the logs first.
- Do not spend effort on Iron Wolf, merc runewords or NM aura swaps now.

## 10. Route guard, Andariel floor and revive budget (added 2026-10-31 by the roadmap follow-up)

The user decided on 2026-10-31 (docs/roadmap.md decision D6) to run this grill now on the numbers below. It is the user's interactive session; nothing here is decided. Cards m3-routeguard and m3-merc wait for it (m3-routeguard writes no delta until the answers exist). Tags as above; EXP = d2-sme experience, U = unverified.

### 10.1 The open numbers, side by side

| number | plan / code today (source) | d2-sme view (source) |
|---|---|---|
| Monastery road floor, no merc | clvl 20 (`route.monasteryMinLevel`, docs/knobs.md; plan-v2 11.2; merc-policy `Route guard`) | clvl 18 (docs/roadmap.md section 6; EXP) |
| Monastery road floor, living merc | merc alone opens it (any clvl; plan-v2 11.2) | clvl 14 with the merc (docs/roadmap.md section 6; EXP) |
| Forgotten Tower floor | clvl 20, `route.towerNeedsMerc` 1 (`route.towerMinLevel`, docs/knobs.md) | clvl 16 without a merc, 12 with (docs/roadmap.md section 6; EXP) |
| Andariel floor | clvl 15, +2 if the merc is dead and gold < 1000 (`boss.andarielMinLevel`, merc-policy `Andariel floor`; docs/bossgate.md) | clvl 15 only with a living merc, 18 without (section 3 option B; EXP); the SME audit A1 puts the XP cliff at clvl 17 to 18 in Act 1 (docs/sme/audit-bot-vs-human.md A1, V table math: 5 % XP at clvl 22 vs monster level 12, 62 % at -7) |
| Revive budget | revive only with gold >= `merc.reviveGoldMult` (2) x price, at most `merc.reviveMaxPerHour` (2); stop bar `merc.stopGoldMult` (5) x price after `merc.stopDeathsPerHour` (3) deaths (docs/knobs.md; merc-policy `Revive and stop rule`); price level^2/2*15 (S, never observed) | cost <= half of gold on hand plus the kit, stop after 2 deaths per hour or when Andariel is dead (docs/roadmap.md section 8 as first proposed; EXP) |

Evidence both sides rest on: the Soaky deaths on the Monastery road at clvl ~14 without a merc and with a thin kit (V, our logs, plan-v2 11.2), monster level 8 to 10 there (S table), and the XP total to clvl 15 of 176,132 (act1-campaign spec, V table). The SME numbers are experience (section 8); the plan's 20 rests on three deaths at clvl 14 in a run with an empty belt, which says more about the kit than the level.

### 10.2 Questions the grill should settle

1. Monastery road without a merc: floor 20 (plan), 18 (SME) or a kit-and-life rule instead of a clvl (option B of section 3)? Recommended default: 18, with the kit gate unchanged, and reopen if deaths per 1000 minutes on the road exceed the soak-gates death rule at that floor.
2. Monastery road and tower with a living merc: allow from 14 (road) and 12 (tower) as the SME says, or keep "any clvl with a merc"? Recommended default: 14 and 12 (a merc is not a license for clvl 8); this adds a `withMerc` floor per guard.
3. Tower floor: 20 (plan) or 16 without a merc? The tower is a no-reward quest in Normal (plan-v2 10.5). Recommended default: 16, and skip the tower unless the Countess is wanted (runes are not a v1 goal).
4. Andariel floor: 15 with a living merc and 18 without (SME), 15 flat (today) or 18 always (option C, plus a live attempt first)? Recommended default: 15 with a living merc, 18 without, the dead-merc +2 rule dropped in favour of the 18; the m3-andariel throwaway fight confirms or moves it.
5. Revive budget and stop rule: keep 2 x price and 5 x price after 3 deaths, or the SME's budget (cost at most half of the gold on hand plus the kit, stop after 2 deaths per hour or when Andariel is dead)? Recommended default: the SME budget, since price is unobserved (S): one live revive price probe (section 5 item 3) before the numbers are fixed, then the knobs follow.

If the user wants probes first (option D), the answer to 1 to 5 is "the defaults above, provisional" and probes 1 to 5 of section 5 run before m3-routeguard launches.

