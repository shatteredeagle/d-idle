---
id: merc-policy
goal: G-B
status: in-force
enforced_by:
  - internal/merc
  - internal/d2dmerc
---
# Merc policy contract

## Purpose
How the bot gets, keeps and gears a mercenary. The user decided on 2026-10-30 to take the free Rogue (Blood Raven reward) and never pay Kashya before the quest. Source: plan-v2 sections 3, 10, 10.1, 10.6, 11.2, 11.3; docs/merc.md and docs/merc-wire.md (code, wired behind `merc.enabled`, default 1 since the mercpolicy card).

> status: reconciled 2026-11-04 against docs/process/spec-ledger-20261104.md; pending deltas: m3-merc

## Requirements

### Requirement: Free Rogue first
The bot SHALL claim the free Rogue at Kashya after Blood Raven is dead (quest state 3 to 5; the game takes the first merc list entry, any owned merc refuses, a Normal-only test was not found in the read path (U); the bot still claims in Normal only) and MUST NOT pay for any Act 1 hire before that; hiring any merc first forfeits the reward, and a dead owned merc still forfeits it. Source: plan-v2 section 10 (user decision), 10.6 and 11.3 (V-d2d: no hire window, gold unchanged).

#### Scenario: Reward available
- **WHEN** Blood Raven is dead, no merc is owned and Kashya's menu is open in Normal
- **THEN** the claim step runs and no gold is spent

### Requirement: Paid Rogue only as fallback
A paid Kashya hire SHALL happen only when the bot skipped Blood Raven for good (`BloodRavenSkipped`) and `merc.freeRogueOnly` is 0, with the price rule `price x merc.hireGoldMult <= gold - kit`. Source: docs/merc.md policy 2.

#### Scenario: Skipped Blood Raven
- **WHEN** Blood Raven is skipped for good and gold covers the price rule
- **THEN** the paid hire is allowed

### Requirement: Blood Raven kit gate
A Blood Raven attempt MUST start only with clvl 9 or higher for Sorceress, Druid and Amazon (8 for Necromancer and melee), 4 healing potions, 2 mana potions for casters, a town portal tome with 2 charges and about 250 gold; at most 3 attempts, a failed one waits for clvl +1 or +2 and a refill. The gate binds attempts only: a kill below it still makes the Rogue claimable (U how). Source: plan-v2 section 10.1; gate2-analysis 2026-11-02. Reconciled 2026-11-04 (source: mercpolicy).

#### Scenario: Thin kit
- **WHEN** a clvl 10 Sorceress has 2 healing potions
- **THEN** no Blood Raven attempt starts and the bot restocks

#### Scenario: Kill below the gate
- **WHEN** Blood Raven is dead and the character is clvl 7
- **THEN** the claim at Kashya still runs and no gold is spent

### Requirement: Revive and stop rule
The bot SHALL revive a dead merc only with a town portal and potions and gold of at least `merc.reviveGoldMult` (1) x the price plus the revive reserve (see Spending order), at most `merc.reviveMaxPerHour` (2) times an hour, and SHALL use `merc.stopGoldMult` (5) as the multiple when merc deaths exceed `merc.stopDeathsPerHour` (3). Price estimate: level squared / 2 x 15, cap 50,000 (S). Source: user decision 2026-11-02; docs/merc.md policy 5. Reconciled 2026-11-04 (source: mercpolicy).

#### Scenario: Short of gold
- **WHEN** revive costs 750, clvl is 14 (reserve 50) and gold is 799
- **THEN** the revive is skipped with reason `gold`

### Requirement: Route guard
Current implementation (interim): the Monastery road (areas 7, 26 to 37) MUST be entered only at clvl 14 or higher, merc or not (`route.monasteryMinLevel` 14, `...Merc` 14), and the Forgotten Tower (areas 20 to 25) at clvl 15 with no merc and max life 150 and the kit (4 healing, a town portal with charges, 3 mana potions for a caster), clvl 14 with a living merc, and clvl 16 when max life is unreadable (the merc is no longer required; knobs `route.towerMinLevel` 15, `route.towerMinLife` 150, `route.towerMinLevelNoLife` 16, `route.towerKitMana` 3, `route.towerMinLevelMerc` 14; SME 2026-11-04, U); each floor is logged as `route guard: <gate> floor=<n> (merc=<alive|dead|none>)`. ADR 0011; user 2026-11-02. Reconciled 2026-11-04 (source: mercpolicy).

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

### Requirement: Route guard floors as plan data (direction)
Direction (decided 2026-11-04, not implemented): each guarded zone SHALL carry its floors as plan data (`minClvl`, `minMaxLife`, `minHealPots`) with no merc split, and a RAISE-ONLY death loop SHALL lift a zone's floor after deaths once an exposure minimum is met; the merc effect feeds back through gold. The bot SHOULD stash gold before a guarded zone. Source: user decision 2026-11-04. Reconciled 2026-11-04 (source: decision 5).

#### Scenario: Death in a zone
- **WHEN** deaths in a zone meet the exposure minimum and fail the death rule
- **THEN** the zone floor is raised and is never lowered by the loop

#### Scenario: Dead merc
- **WHEN** the merc is dead and the character meets the zone floor
- **THEN** the zone stays open and the gold lost to revive costs, not a lower floor, feeds back

### Requirement: Route guard starting numbers (direction, U)
The plan data SHALL start at (U, SME design, tuned only by the death loop): road front (areas 7, 26, 27) clvl 13, road deep (Barracks to Catacombs) clvl 15, Forgotten Tower: no merc clvl 15 and max life 150 and the kit, merc clvl 14, unreadable life clvl 16 (SME 2026-11-04, U), Andariel clvl 18 flat with max life 250 and antidote potions. Not implemented: the code keeps the interim numbers. Source: user decision 2026-11-04. Reconciled 2026-11-04 (source: decision 5).

#### Scenario: Code unchanged
- **WHEN** no code card for the direction has been approved
- **THEN** the interim Route guard and Andariel floor numbers decide

### Requirement: Andariel floor
Current implementation (interim): the Normal campaign SHALL attempt Andariel from clvl `boss.andarielMinLevel` 17 with no merc and `boss.andarielMinLevelMerc` 15 with a living merc, with the kit gate plus 2 antidote potions; a dead merc uses the no-merc floor. Before the throwaway fight of card m3-andariel has won once, `boss.andarielHold` 1 makes the campaign level instead, 0 lifts it. Source: ADR 0011, docs/knobs.md. Reconciled 2026-11-04 (source: decision 5).

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
Gear swaps SHALL need at least `merc.swapGainPct` (10 percent) score gain, items must be identified, unbroken and meet the merc's own level (and Str and Dex with `merc.reqStrDex` on); potions SHALL be healing only, below `merc.potionLifePct` (50 percent). All weights and the aura order are judgments (U) until the probe list in docs/merc.md is run. Source: docs/merc.md policy 7, 8, 10; user decision 2026-11-02 (potion line 40 to 50). Reconciled 2026-11-04 (source: 2026-11-02-mercpolicy).

#### Scenario: Marginal upgrade
- **WHEN** a new helm scores 5 percent above the worn one
- **THEN** no swap happens

#### Scenario: Rogue at half life
- **WHEN** the Rogue is at 49 percent life in a non-trivial fight with a healing potion in the belt
- **THEN** she is given a healing potion

### Requirement: Spending order
The bot SHALL revive a dead merc before it buys potions or repairs (selling junk first), as soon as pack plus stash gold covers the probed real quote plus a small reserve: `merc.reviveReserveLow` (0) gold below clvl `merc.reviveReserveClvl` (13) and `merc.reviveReserveHigh` (50) gold from it, never a full potion refill. The stash shortfall is withdrawn for the payment. The price formula (level squared / 2 x 15) never gates the payment; it only decides whether to try when the quote cannot be read, and an unreadable quote is logged once per town visit and probed again on the next visit. A HIRE keeps the potions-first rule: gold of at least price x `merc.hireGoldMult` (1) plus `merc.kitReservePct` (100) percent of the full kit refill (town portal, `resource.kitMinHP` healing and `resource.kitMinMP` mana potions at the shop prices), with a town portal and potions on hand. Source: user 2026-11-06: defer to the SME (SME answer: revive ASAP with a small reserve); hires: user decision 2026-11-02 (SME order), `merc.kitReservePct` 150 became 100 (user decision 2026-11-04). Reconciled 2026-11-04 (source: 2026-11-02-mercpolicy). Replaces the 100 percent kit reserve on revives (g2f seed 17: price 1920 plus a 155 to 310 reserve kept the merc dead 7 hours; the interim cap `merc.reviveReserveMax` of #274 is retired and ignored when an old tuning file names it). Offline tested only; verify from the log lines `merc: revive skipped (gold) price=P gold=G need=N` and `merc: revived`.

#### Scenario: Reserve short
- **WHEN** a revive costs 1920, clvl is 14 and gold is 1930
- **THEN** the revive is skipped with reason `gold` and needs 1970

#### Scenario: Reserve met
- **WHEN** the same revive meets gold 1975 at clvl 14, or gold 1920 at clvl 12
- **THEN** the revive runs

#### Scenario: Hire keeps the kit reserve
- **WHEN** a hire quotes 400, the kit refill is 310 and gold is 709
- **THEN** the hire waits (it needs 710)

#### Scenario: Revive cap (Revive and stop rule)
- **WHEN** two revives already happened in the last hour
- **THEN** the third is skipped with reason `revive cap`

#### Scenario: Stop state (Revive and stop rule)
- **WHEN** a 4th merc death happens in an hour and revive costs 750
- **THEN** the revive needs 5 x 750 plus the revive reserve

### Requirement: Revive price is read, not computed
The bot SHALL pay a revive whatever the NPC takes and log `merc: revived price=<paid> level=<n> estimate=<formula>`; the formula min(level squared / 2 x 15, 50,000) is used only as the pre-check estimate (S, unconfirmed on 1.14d) and MUST NOT be treated as the price. Source: user decision 2026-11-02; docs/merc-wire.md. Reconciled 2026-11-04 (source: 2026-11-02-mercpolicy).

#### Scenario: Dialog price differs
- **WHEN** the estimate is 750 and the revive takes 600
- **THEN** the log shows `price=600` and `estimate=750`

### Requirement: Act 2 Desert Mercenary swap
At the first Lut Gholein visit per difficulty the bot SHALL read the hire window (one `merc: hire window` line per row, fields in docs/botlogs.md) and replace the Rogue with the Desert Mercenary of the best class aura (Defiance for the Sorceress), stripping her gear first, only when the price passes the spending order; a poorer purse leaves the visit open. Source: user decision 2026-11-02; docs/sme/mercenary.md. Reconciled 2026-11-04 (source: mercpolicy).

#### Scenario: Swap with gear
- **WHEN** the Rogue wears a helm and a bow and the window offers Prayer 360 and Defiance 400 with gold 5000
- **THEN** the weapon and the helm are taken off, then the Defiance row is hired and `merc: hired` is logged

#### Scenario: Too poor for the best aura
- **WHEN** gold is 709 (Defiance 400 plus a 310 reserve is 710) and Prayer would pass
- **THEN** nothing is hired and nothing is stripped

### Requirement: Gold farm while the merc is missing
While the merc is dead or missing, gold is below the next merc payment (revive estimate or Normal hire quote of 400, plus the reserve (revive reserve, or the kit reserve for a hire)) and clvl is below `merc.goldFarmMaxLevel` (17, 0 = off), a Normal Act 1 character SHALL level only in Blood Moor, Cold Plains, Stony Field, the Den of Evil, Caves 1 and 2, the Burial Grounds and the Crypt, and log `merc: gold farm on|off` once per change. Source: user decision 2026-11-02; cap raised from 16 to 17 by commit 8f4b8e3f (mercrevive2), docs/knobs.md is the owner. Reconciled 2026-11-04 (source: 2026-11-02-mercpolicy).

#### Scenario: Dead merc, thin purse
- **WHEN** a clvl 11 character has a dead merc and 300 gold
- **THEN** every other area is excluded from the level picker

#### Scenario: Past the cap
- **WHEN** the same character is clvl 16
- **THEN** no area is excluded for the farm

### Requirement: Merc policy on by default
The merc policy SHALL run by default (`merc.enabled` 1: revive, Act 2 swap, merc state for the route guards, status line); 0 leaves the bot as before. Source: user decision 2026-11-02; docs/merc-wire.md. Reconciled 2026-11-04 (source: 2026-11-02-mercpolicy).

#### Scenario: Default tuning
- **WHEN** the default tuning is loaded
- **THEN** `merc.enabled` is 1

### Requirement: Readiness model
Once the readiness model is enabled the route guard SHALL allow an area only when character power is at least zone danger times the safety margin, SHALL never allow an area more than 2 clvl below the merc-policy floors, and the model MUST be enabled only after its offline replay rates the 9 Monastery road deaths (clvl 11-12, 2-potion kit) as not ready. Source: ADR 0011 (grill 0003, 2026-10-31); card m3-readiness-model. Reconciled 2026-11-04 (source: 2026-11-04-m3-readiness-model).

#### Scenario: Road death replay
- **WHEN** the model replays a clvl 11 character with a 2-potion kit entering the Monastery road against a Dark Stalker pack
- **THEN** the verdict is not ready

#### Scenario: Floor safety net
- **WHEN** the model rates a clvl 12 character ready for the Forgotten Tower with no merc, max life 149 and the kit, and the no-merc floor is clvl 15 (max life 150 and the kit)
- **THEN** the guard still blocks the Forgotten Tower because the model may go at most 2 clvl below the floor, so 13 is the lowest allowed level

#### Scenario: Forgotten Tower floor examples
- **WHEN** a clvl 15 character with no merc, max life 149 and the kit asks for the Forgotten Tower
- **THEN** the guard blocks it (max life below 150)
- **WHEN** a clvl 15 character with no merc, max life 150 and the kit met (4 healing, a town portal with charges, 3 mana potions for a caster) asks for the Forgotten Tower
- **THEN** the guard allows it
- **WHEN** a clvl 14 character with a living merc asks for the Forgotten Tower
- **THEN** the guard allows it (living merc floor clvl 14)
- **WHEN** the max life cannot be read
- **THEN** the no-merc floor is clvl 16

#### Scenario: Model disabled
- **WHEN** the enable knob is 0
- **THEN** only the clvl floors decide

## Pending deltas (not in force)
- m3-merc (open card, proposed): ADDED "Merc uptime in Act 1", "Blood Raven bailout", "Keeper and throwaway mercs"; MODIFIED "Revive and stop rule" (reviveGoldMult 2 on gold minus the kit reserve). The MODIFIED block conflicts with the folded mercpolicy text (reviveGoldMult 1 plus the kit reserve) and its "Revive too expensive" scenario assumes the old rule, so the card needs a rebase on the folded text before it is applied.
