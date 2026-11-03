## ADDED Requirements

### Requirement: Spending order
The bot SHALL spend on potions before the merc: a revive or a hire is paid only with gold of at least price x `merc.reviveGoldMult` (1) or `merc.hireGoldMult` (1) plus `merc.kitReservePct` (150) percent of the full kit refill (town portal, `resource.kitMinHP` healing and `resource.kitMinMP` mana potions at the shop prices), with a town portal and potions on hand. Source: user decision 2026-11-02 (SME order).

#### Scenario: Reserve short
- **WHEN** a revive costs 750, the kit refill is 310 and gold is 1214
- **THEN** the revive is skipped with reason `gold`

#### Scenario: Reserve met
- **WHEN** the same revive meets gold 1215
- **THEN** the revive runs

### Requirement: Revive price is read, not computed
The bot SHALL pay a revive whatever the NPC takes and log `merc: revived price=<paid> level=<n> estimate=<formula>`; the formula min(level squared / 2 x 15, 50,000) is used only as the pre-check estimate (S, unconfirmed on 1.14d) and MUST NOT be treated as the price. Source: user decision 2026-11-02; docs/merc-wire.md.

#### Scenario: Dialog price differs
- **WHEN** the estimate is 750 and the revive takes 600
- **THEN** the log shows `price=600` and `estimate=750`

### Requirement: Act 2 Desert Mercenary swap
At the first Lut Gholein visit per difficulty the bot SHALL read the hire window (one `merc: hire window` line per row, once, fields in docs/botlogs.md) and replace the Rogue with the Desert Mercenary of the best class aura (Defiance for the Sorceress), stripping her gear first, only when the price passes the spending order; a poorer purse leaves the visit open and a better aura in the window is waited for. Source: user decision 2026-11-02; docs/sme/mercenary.md.

#### Scenario: Swap with gear
- **WHEN** the Rogue wears a helm and a bow and the window offers Prayer 360 and Defiance 400 with gold 5000
- **THEN** the weapon and the helm are taken off, then the Defiance row is hired and `merc: hired` is logged

#### Scenario: Too poor for the best aura
- **WHEN** gold is 864 (Defiance 400 plus a 465 reserve is 865) and Prayer would pass
- **THEN** nothing is hired and nothing is stripped

### Requirement: Gold farm while the merc is missing
While the merc is dead or missing, gold is below the next merc payment (revive estimate or Normal hire quote of 400, plus the kit reserve) and clvl is below `merc.goldFarmMaxLevel` (16, 0 = off), a Normal Act 1 character SHALL level only in Blood Moor, Cold Plains, Stony Field, the Den of Evil, Caves 1 and 2, the Burial Grounds and the Crypt, and log `merc: gold farm on|off` once per change. Source: user decision 2026-11-02.

#### Scenario: Dead merc, thin purse
- **WHEN** a clvl 11 character has a dead merc and 300 gold
- **THEN** every other area is excluded from the level picker

#### Scenario: Past the cap
- **WHEN** the same character is clvl 16
- **THEN** no area is excluded for the farm

### Requirement: Merc policy on by default
The merc policy SHALL run by default (`merc.enabled` 1: revive, Act 2 swap, merc state for the route guards, status line); 0 leaves the bot as before. Source: user decision 2026-11-02; docs/merc-wire.md.

#### Scenario: Default tuning
- **WHEN** the default tuning is loaded
- **THEN** `merc.enabled` is 1

## MODIFIED Requirements

### Requirement: Blood Raven kit gate
A Blood Raven attempt MUST start only with clvl 9 or higher for Sorceress, Druid and Amazon (8 for Necromancer and melee), 4 healing potions, 2 mana potions for casters, a town portal tome with 2 charges and about 250 gold; at most 3 attempts, a failed one waits for clvl +1 or +2 and a refill. The gate binds attempts only: a kill below it still makes the Rogue claimable (Gate 2 claimed at clvl 7, U how). Source: plan-v2 section 10.1 (brkill 1477431); gate2-analysis 2026-11-02.

#### Scenario: Thin kit
- **WHEN** a clvl 10 Sorceress has 2 healing potions
- **THEN** no Blood Raven attempt starts and the bot restocks

#### Scenario: Kill below the gate
- **WHEN** Blood Raven is dead and the character is clvl 7
- **THEN** the claim at Kashya still runs and no gold is spent

### Requirement: Revive and stop rule
The bot SHALL revive a dead merc only with a town portal and potions and gold of at least `merc.reviveGoldMult` (1) x the price plus the kit reserve, at most `merc.reviveMaxPerHour` (2) times an hour, and SHALL use `merc.stopGoldMult` (5) as the multiple when merc deaths exceed `merc.stopDeathsPerHour` (3). Price estimate: level squared / 2 x 15, cap 50,000 (S). Source: user decision 2026-11-02 (supersedes the pending m3-merc delta); docs/merc.md policy 5.

#### Scenario: Short of gold
- **WHEN** revive costs 750, the kit reserve is 465 and gold is 1199
- **THEN** the revive is skipped with reason `gold`

#### Scenario: Cap
- **WHEN** two revives already happened in the last hour
- **THEN** the third is skipped with reason `revive cap`

#### Scenario: Stop state
- **WHEN** a 4th merc death happens in an hour and revive costs 750
- **THEN** the revive needs 5 x 750 plus the reserve

### Requirement: Route guard
The Monastery road (areas 7, 26 to 37) MUST be entered only at clvl 14 or higher with or without a merc (`route.monasteryMinLevel` 14, `...Merc` 14) and the Forgotten Tower (areas 20 to 25) at 16 with no merc, 14 with a living one, plus the full supply kit on every guard. Each floor is logged as `route guard: <gate> floor=<n> (merc=<alive|dead|none>)`. A no-merc knob at 0 turns a guard off. Interim. ADR 0011; user decision 2026-11-02 (a dead merc must not block the road).

#### Scenario: No merc at clvl 14
- **WHEN** a clvl 14 caster without a merc and with the full supply kit picks a leveling area
- **THEN** the Monastery road is allowed, the Forgotten Tower is excluded, and at clvl 13 the road is excluded too

#### Scenario: Living merc at clvl 14
- **WHEN** a clvl 14 caster with a living merc and the full supply kit picks a leveling area
- **THEN** the Monastery road and the Forgotten Tower are allowed

#### Scenario: Thin kit with a merc
- **WHEN** a clvl 15 caster with a living merc holds 2 healing potions
- **THEN** the Monastery road is excluded until the kit is refilled

#### Scenario: Effective floor logged
- **WHEN** the merc state of a clvl 12 caster changes from none to alive
- **THEN** the log shows `route guard: monastery floor=14 (merc=none)` once and then `route guard: monastery floor=14 (merc=alive)` once

#### Scenario: Road floor trigger
- **WHEN** Monastery road deaths in the act1run soak fail the death rule at the clvl 14 floor
- **THEN** the floor is raised to 16 and the decision goes back to a grill

### Requirement: Merc gear and potions
Gear swaps SHALL need at least `merc.swapGainPct` (10 percent) score gain, items must be identified, unbroken and meet the merc's own level (and Str and Dex with `merc.reqStrDex` on); potions SHALL be healing only, below `merc.potionLifePct` (50 percent). All weights and the aura order are judgments (U) until the probe list in docs/merc.md is run. Source: docs/merc.md policy 7, 8, 10; user decision 2026-11-02 (potion line 40 to 50).

#### Scenario: Marginal upgrade
- **WHEN** a new helm scores 5 percent above the worn one
- **THEN** no swap happens

#### Scenario: Rogue at half life
- **WHEN** the Rogue is at 49 percent life in a non-trivial fight with a healing potion in the belt
- **THEN** she is given a healing potion
