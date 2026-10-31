# Death rule specification

## Purpose
How deaths are judged in soaks and what the bot does after a death. The rule was APPROVED by the user on 2026-10-30 (plan-v2 section 10.2; docs/soak-analysis.md section 12) and is implemented in cmd/d2soak and internal/bot/deathrule.go. Evidence: docs/plan/death-data.md, docs/deathautopsy.md, docs/deathspiral.md.

## Requirements

### Requirement: Death is normal
Dying SHALL NOT fail a run by itself; only the rules below fail it. A careful caster is expected at 0 to 1 deaths per hour at clvl 10 or higher, 1 to 3 per hour is sloppy, 5 or more per hour is a sick bot. Source: plan-v2 section 10.2 (EXP, d2-sme).

#### Scenario: Single death
- **WHEN** one death occurs in a two hour run and no other rule trips
- **THEN** the run is not failed for it

### Requirement: Death streak (FAIL-1)
A run MUST fail on 3 deaths in 30 minutes with no clvl gain in that span, or on more than 4 deaths in 60 minutes (5 or more in 60 minutes fails regardless of level gain). This rule needs no minimum sample. Source: plan-v2 section 10.2.

#### Scenario: Three deaths without a level
- **WHEN** 3 deaths happen within 30 minutes and clvl did not change
- **THEN** d2soak reports `death_streak` FAIL

### Requirement: Recovery stuck (FAIL-2)
A run MUST fail when time in death-recovery states exceeds 10 percent of the rolling 60 minutes, or when any single death-to-back-in-field span exceeds 5 minutes. The recovery states are `Recovering after death`, `Restocking after death`, `Retrieving the corpse` and `Chores in town (poverty rebuild after failed retrieval)`; the window ends at the first combat or non-town area change. `Recovery: farming gold` and `Waiting: banned` never count. Source: plan-v2 section 10.2.

#### Scenario: Slow recovery
- **WHEN** one death takes 6 minutes to get back to the field
- **THEN** d2soak reports `death_recovery_stuck` FAIL

### Requirement: Death rate (FAIL-3)
A long clean run MUST fail above 15 deaths per 1000 kills, evaluated only with at least 150 kills in the window and after 30 minutes; above 5 per 1000 is a WARN. Kills undercount by 16 to 37 percent today. Source: plan-v2 section 10.2.

#### Scenario: Too few kills
- **WHEN** the window holds 100 kills and 3 deaths
- **THEN** the rate rule is not evaluated

### Requirement: Warnings
The judge SHALL WARN when the recovery share exceeds 5 percent or when 2 deaths occur in 60 minutes; a gold-lost share is report-only. Source: plan-v2 section 10.2.

#### Scenario: Two deaths in an hour
- **WHEN** 2 deaths fall inside 60 minutes
- **THEN** the report shows a WARN and the run is not failed

### Requirement: Repeat-site action
After 2 deaths in one area within 30 minutes, or a death within 180 seconds of the previous one, the bot SHALL ban that area for 30 minutes and write a planner dump; bans escalate x4 and x12 (cap 6 h) for the same area within 12 hours, and a ban MUST NOT remove the only viable farm site (fallback in NOIDLE). Source: plan-v2 sections 10.2 and 11.2.

#### Scenario: Fast second death
- **WHEN** a second death occurs 150 s after the first in the same area
- **THEN** the area is banned for 30 minutes and a planner dump is written

### Requirement: Bug tag
A death with at least 1 healing potion in the belt and no potion use SHALL be tagged BUG and counted separately. Source: plan-v2 section 10.2.

#### Scenario: Unused potion
- **WHEN** the bot dies with 3 healing potions in the belt and none drunk in the last 60 s
- **THEN** deaths.jsonl marks the death as BUG

### Requirement: Gold carry and corpse
The bot SHALL carry at most about 50 x clvl + 300 gold (850 at clvl 11) and bank the rest, sweep the death-spot gold pile on the in-field retrieval path, and retrieve the corpse in place only when it is a short walk away, the area is clear and it is the first attempt; otherwise it SHALL use the quit-and-restart path once M11 is verified. Source: plan-v2 sections 10.2, 10.3, 10.6 and 11.3 (death gold V on d2d only; the original game is U).

#### Scenario: Big purse
- **WHEN** the character dies with 1100 gold at clvl 12
- **THEN** the pile at the death spot is swept before a corpse trip is judged
