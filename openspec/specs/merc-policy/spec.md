# Merc policy specification

## Purpose
How the bot gets, keeps and gears a mercenary. The user decided on 2026-10-30 to take the free Rogue (Blood Raven reward) and never pay Kashya before the quest. Source: plan-v2 sections 3, 10, 10.1, 10.6, 11.2, 11.3; docs/merc.md and docs/merc-wire.md (code, wired behind `merc.enabled`, default 0).

## Requirements

### Requirement: Free Rogue first
The bot SHALL claim the free Rogue at Kashya after Blood Raven is dead (quest state 3 to 5, Normal only) and MUST NOT pay for any Act 1 hire before that; hiring any merc first forfeits the reward, and a dead owned merc still forfeits it. Source: plan-v2 section 10 (user decision), 10.6 and 11.3 (V-d2d: no hire window, gold unchanged).

#### Scenario: Reward available
- **WHEN** Blood Raven is dead, no merc is owned and Kashya's menu is open in Normal
- **THEN** the claim step runs and no gold is spent

### Requirement: Paid Rogue only as fallback
A paid Kashya hire SHALL happen only when the bot skipped Blood Raven for good (`BloodRavenSkipped`) and `merc.freeRogueOnly` is 0, with the price rule `price x merc.hireGoldMult <= gold - kit`. Source: docs/merc.md policy 2.

#### Scenario: Skipped Blood Raven
- **WHEN** Blood Raven is skipped for good and gold covers the price rule
- **THEN** the paid hire is allowed

### Requirement: Blood Raven kit gate
A Blood Raven attempt MUST start only with clvl 9 or higher for Sorceress, Druid and Amazon (8 for Necromancer and melee), at least 4 healing potions, at least 2 mana potions for casters, a town portal tome with 2 or more charges, and gold of about 250 or more; at most 3 attempts, a failed attempt waits for clvl +1 or +2 and a refill, and a second retreat ends the attempt. Source: plan-v2 section 10.1 (EXP, merged as brkill 1477431).

#### Scenario: Thin kit
- **WHEN** a clvl 10 Sorceress has 2 healing potions
- **THEN** no Blood Raven attempt starts and the bot restocks

### Requirement: Revive and stop rule
The bot SHALL revive a dead merc only with a town portal and potions present and gold of at least `merc.reviveGoldMult` x the price (price = level squared / 2 x 15, cap 50,000), with at most `merc.reviveMaxPerHour` revives per hour, and SHALL raise the bar to `merc.stopGoldMult` x price when merc deaths exceed `merc.stopDeathsPerHour`. Source: docs/merc.md policy 5; revive formula S (d2decomped), never observed on d2d.

#### Scenario: Short of gold
- **WHEN** revive costs 600 and gold is 900 with a multiplier of 2
- **THEN** the revive is skipped with reason `gold`

### Requirement: Route guard
The Monastery road (areas 7, 26 to 37, so Andariel too) and the Forgotten Tower (areas 20 to 25) MUST be entered only at clvl 16 or higher with no merc, or 14 or higher with a living merc (`route.monasteryMinLevel*`, `route.towerMinLevel*`, `route.towerNeedsMerc` 0), plus the supply kit on every guard. Each effective floor is logged as `route guard: <gate> floor=<n> (merc=<alive|dead|none>)`. A no-merc knob at 0 turns a guard off; unknown merc is none. Interim. ADR 0011.

#### Scenario: No merc at clvl 14
- **WHEN** a clvl 14 caster without a merc picks a leveling area
- **THEN** the Monastery road and the Forgotten Tower are excluded and the planner levels in the Blood Moor to Black Marsh, caves and Crypt instead

#### Scenario: Living merc at clvl 14
- **WHEN** a clvl 14 caster with a living merc and the full supply kit picks a leveling area
- **THEN** the Monastery road and the Forgotten Tower are allowed

#### Scenario: Thin kit with a merc
- **WHEN** a clvl 15 caster with a living merc holds 2 healing potions
- **THEN** the Monastery road is excluded until the kit is refilled

#### Scenario: Effective floor logged
- **WHEN** the merc state of a clvl 12 caster changes from none to alive
- **THEN** the log shows `route guard: monastery floor=16 (merc=none)` once and then `route guard: monastery floor=14 (merc=alive)` once

#### Scenario: Road floor trigger
- **WHEN** Monastery road deaths in the act1run soak fail the death rule at the clvl 16 floor
- **THEN** the floor is raised to 18 and the decision goes back to a grill, and if the soak shows no road deaths 14 is tried

### Requirement: Andariel floor
The Normal campaign SHALL attempt Andariel from clvl `boss.andarielMinLevel` 17 with no merc and `boss.andarielMinLevelMerc` 15 with a living merc, with the kit gate on top plus at least 2 antidote potions; the former +2 for a dead merc is replaced by the no-merc floor. No unattended attempt before the throwaway fight of card m3-andariel has won once: `boss.andarielHold` 1 makes the campaign level instead, 0 lifts it. Source: ADR 0011, docs/knobs.md.

#### Scenario: Floor read from the knob
- **WHEN** an agent needs the Andariel level
- **THEN** it reads docs/knobs.md or the knob, not a number copied from a doc

#### Scenario: Dead merc
- **WHEN** a clvl 16 character has no living merc and enough gold to revive
- **THEN** Andariel is not attempted until clvl 17 or the merc is revived and clvl 15 is met

#### Scenario: No antidotes
- **WHEN** a clvl 17 character holds a full kit but 1 antidote potion
- **THEN** the Andariel attempt does not start and the bot restocks

#### Scenario: Unattended hold
- **WHEN** a clvl 20 character with a living merc and a full kit reaches Andariel in the campaign while `boss.andarielHold` is 1
- **THEN** the campaign levels instead and logs `boss: held for andariel (unattended hold ...)`, and with the knob 0 the attempt starts

#### Scenario: Old numbers by knobs
- **WHEN** the road and tower floors are set to 20 with the with-merc road floor 0 and `route.towerNeedsMerc` 1, and the Andariel floor to 15
- **THEN** the former decisions return (a living merc opens the road and tower at any level); the dead-merc +2 is not restored, the no-merc floor 17 equals it

### Requirement: Merc gear and potions
Gear swaps SHALL need at least `merc.swapGainPct` (10 percent) score gain, items must be identified, unbroken and meet the merc's own level (and Str and Dex with `merc.reqStrDex` on); potions SHALL be healing only, below `merc.potionLifePct` (40 percent). All weights and the aura order are judgments (U) until the probe list in docs/merc.md is run. Source: docs/merc.md policy 7, 8, 10.

#### Scenario: Marginal upgrade
- **WHEN** a new helm scores 5 percent above the worn one
- **THEN** no swap happens
