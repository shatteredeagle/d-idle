---
status: accepted
date: 2026-10-31
decision-makers: user (grill 0003)
---
# 0011: Readiness is character power against zone danger; merc-split clvl floors are the interim rule and the safety net

## Context and Problem Statement

The route guard (plan-v2 11.2, merc-policy `Route guard`) keeps a caster off the Monastery road and the Forgotten Tower until clvl 20 or any living merc; Andariel's floor is clvl 15, +2 with a dead merc and gold < 1000 (merc-policy `Andariel floor`). The d2-sme disputed the 20 (roadmap section 6, D6). Cards m3-routeguard and m3-merc wait on this grill.

Facts used (tags from the grill's prebrief and verified-claims.md):
* The 9 Monastery road deaths were at clvl 11-12 with a 2-potion kit; killers melee Dark Stalker packs (6), Steel Grumble (2), Dark Lancer (1); no ranged death (V, docs/plan/death-data.md via verified-claims claim 7). The plan's "clvl ~14" was wrong.
* Road monster level 8-10 to the Barracks, 12 in Catacombs 3-4 (V, levels.txt).
* Andariel Normal in game: 1024 HP, melee 6-19, poison is the danger, fire -50 (V, data + Blizzard bestiary). The SME's floor 18 rested on the raw 2562 column (refuted).
* Act 1 XP drops off from clvl 17-18 (V, table math, docs/sme/audit-bot-vs-human.md A1).
* Revive price level^2/2*15 (DERIVED from the decompile via d2d, never seen live): 540-600 at merc lvl 7-9; a paid Rogue at clvl 10-12 costs about 100-240 (DERIVED).
* Hiring any merc, even a dead one, forfeits the free Rogue (DERIVED d2d; LIKELY on the original).
* No power or danger rating exists in the bot today (U: none found in the docs read).

## Considered Options

* A: plan as is (clvl 20 or any living merc; Andariel 15 +2)
* B: SME kit-and-life guard (road 18/14, tower 16/12, Andariel 18 without a merc)
* C: cautious (20 kept, Andariel 18-20, no Act 2 work before a live win)
* D: probes first, numbers later
* E (user, in the grill): fixed clvl floors are too prescriptive; area level is a general guide; rate zone danger and character power and progress when power covers danger

## Decision Outcome

Chosen: E, with interim floors from a lowered B, set now as provisional numbers (not D: no extra probe round before m3-routeguard).

Target rule (new work, its own change card after this ADR):
* Character power = survival (max life, resists, potions in the belt, town portal, merc alive) + offense (estimated kill speed from our skill damage against the area's real monster HP after resists and immunities) + clvl as a minor input.
* Zone danger = area monster level (the general guide) + real monster HP and damage (monlvl scaling) + crowd size (pack size, density) + threats the build handles badly (immunities to our damage type, poison, ranged or lightning casters, uniques and bosses) + our own deaths and chicken exits per hour in that area, weighted more as data comes in.
* Ready = power >= danger x a safety margin. Weights and margin are set by that card.
* The model takes over only after an offline calibration: replayed on our logs it rates the 9 road deaths (clvl 11-12, 2-potion kit) as not ready, and it agrees with the m3-andariel result. After that the floors below are a safety net: the model may go at most 2 clvl below them.

Interim floors (provisional; m3-routeguard writes them now, does not wait for the model):

| gate | no merc | living merc |
|---|---|---|
| Monastery road (areas 7, 26-37) | 16 | 14 |
| Forgotten Tower (areas 20-25), `route.towerNeedsMerc` 0 | 16 | 14 |
| Andariel, kit gate plus >= 2 antidote potions | 17 | 15 |

The Andariel +2 dead-merc rule is replaced by the no-merc floor. The existing supply kit gate stays on every guard.

Merc rules:
* Revive only when gold minus the kit reserve >= `merc.reviveGoldMult` (2) x price; `merc.reviveMaxPerHour` (2), `merc.stopGoldMult` (5) and `merc.stopDeathsPerHour` (3) stay.
* Blood Raven bailout: after 3 failed attempts or at clvl 12 without the kill, she counts as skipped for good (`BloodRavenSkipped`) and the paid Rogue fallback of the spec applies.
* Keeper vs throwaway: each build names a target merc (type, aura for Act 2) and target gear. A merc that is the build's target or wears target gear is a keeper: always revived, never replaced to save gold. Any other merc is throwaway: re-hired when the hire price is below the revive price (same gold-minus-kit rule). Until builds define a target merc, the bot always revives.

### Gates and triggers

1. m3-andariel (cheat-levelled throwaway fight) and m3-act1run confirm or move the floors.
2. Road floor trigger (user asked to pin it for follow-up): if Monastery road deaths fail the death rule (ADR 0005, soak-gates) at the clvl 16 floor in m3-act1run, raise it to 18 and re-grill; if m3-act1run shows no road deaths, try 14.
3. No unattended Andariel attempt before m3-andariel's throwaway fight has won once.
4. The power/danger model replaces the floors only after the calibration above.

## Consequences

* m3-routeguard can launch: MODIFIED deltas on merc-policy `Route guard` and `Andariel floor` with the interim table, `route.towerNeedsMerc` 0, a with-merc floor per guard, the antidote item in the Andariel kit.
* m3-merc: revive rule counts gold above the kit reserve; Blood Raven bailout; always-revive until build merc targets exist.
* A new card is needed for the power/danger model and its offline calibration (not created here).
* Good: less grinding in the low-XP Act 1 band; the gate follows the real cause of deaths (crowds, kit, life) instead of a clvl.
* Bad: the floors are below the plan's 20 on evidence from one character's logs; the trigger above catches it if wrong. Revive and hire prices are not yet seen live.

### Parked

* Build merc targets (each build's target merc and gear): needed for the keeper rule.
* Desert merc in Normal: buy or not, at what gold, survival in Act 2 (M4).
* Max-life floor for Andariel: waits for m3-andariel's life-per-phase numbers.
* Open live probes: dead-merc forfeit on the original, one revive quote, 2-3 hire windows, merc Str/Dex own-stats rule, merc state after quit and reload.
* The model's weights and safety margin (its card).

## Sources

docs/grills/0003-merc-and-act2/prebrief.md, verified-claims.md, sme-grill.md, outcome.md; ADR 0005, ADR 0006; openspec/specs/merc-policy/spec.md; plan-v2 11.2; docs/roadmap.md sections 6 and 8 (D6); docs/knobs.md
