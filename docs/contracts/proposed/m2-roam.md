# Proposed contract text: m2-roam (early-game-policy)

> status: PROPOSED, not in force. Converted at OpenSpec removal stage 3 from the delta spec of card `m2-roam` (git tag `openspec-final`, path `openspec/changes/m2-roam/specs/early-game-policy/spec.md`). When the work lands, the branch that implements it folds this text into `docs/contracts/early-game-policy.md` and deletes this file. Backlog task: title starts with `m2-roam:`.

## Requirements (new)

### Requirement: Leave an empty area
When a sweep has gone `bot.emptyAreaSec` seconds without a kill in an area it already crossed, the bot SHALL mark the area cleared and move to the next area of the plan instead of roaming it, and MUST NOT re-enter an area it left in the last 10 minutes unless the plan names it; the knob default is on (value in docs/knobs.md). Source: docs/killfix.md (measurement), docs/d2d-soak-full-fix.md (walk 77 percent of kill time), card m2-roam.

#### Scenario: Swept Blood Moor
- **WHEN** a clvl 3 character has crossed the Blood Moor and no kill happened for the knob's seconds
- **THEN** the area is marked cleared and the next step is Cold Plains, not another Blood Moor sweep

#### Scenario: Border ping-pong
- **WHEN** the character left Cold Plains for the Cave 2 minutes ago and the Cave is not cleared
- **THEN** it does not walk back to Cold Plains
