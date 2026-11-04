# Proposed contract text: m2-tripwp (early-game-policy)

> status: PROPOSED, not in force. Converted at OpenSpec removal stage 3 from the delta spec of card `m2-tripwp` (git tag `openspec-final`, path `openspec/changes/m2-tripwp/specs/early-game-policy/spec.md`). When the work lands, the branch that implements it folds this text into `docs/contracts/early-game-policy.md` and deletes this file. Backlog task: title starts with `m2-tripwp:`.

## Requirements (new)

### Requirement: Leave town by waypoint
When the target area's waypoint, or an activated waypoint nearer to it than the town exit, is active, the bot SHALL leave town through the waypoint and MUST NOT walk out; when an open town portal of its own exists it SHALL return through it; a chicken with a portal scroll or tome charge in the pack SHALL use the portal, not leave the game. Source: docs/d2d-xp-gap.md (2.4 minutes per walk back, 14 percent of time; 21 percent of long gaps are leave-game restarts); card m2-tripwp.

#### Scenario: Cold Plains waypoint active
- **WHEN** the next step is Cold Plains and its waypoint is activated
- **THEN** the bot takes the town waypoint to Cold Plains and the walk-out minute count is zero

#### Scenario: Chicken with a scroll
- **WHEN** life drops under the chicken line and the pack holds a portal scroll
- **THEN** the bot casts the portal and does not leave the game
