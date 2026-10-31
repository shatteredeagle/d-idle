# bot-loop Specification

## Purpose
What the bot main loop must keep true across games: how bot-started restarts are counted on the minute line, what per-game memory is forgotten on a new game, and how monsters are classified from their flag byte.

## Requirements

### Requirement: Restarts counts every bot-started game
The minute line `restarts` value SHALL be the maximum of (a) menu-observed game entries after the first that were not planned save points and (b) the games the bot started itself (newGameRun: farm, difficulty switch, fresh game, failsafe rung 3; the between-goals recycle), and MUST never be negative; planned save points do not count. Known limit: d2d crash re-entries are NOT counted, nothing on that backend sees them. Source: card botsmall; docs/bot-small.md; test TestRestartCount.

#### Scenario: d2d recycles without a menu poll
- **WHEN** the menu observer saw 1 game entry (no menu poll on d2d), 0 planned save points and the bot started 17 games
- **THEN** restartCount(1, 0, 17) is 17

#### Scenario: Planned save points
- **WHEN** the observer saw 5 game entries of which 4 were planned save points and the bot started 0 games
- **THEN** restartCount(5, 4, 0) is 0

#### Scenario: Fresh game without an observer
- **WHEN** freshGameToLevel starts three fresh games and the observer saw no menus
- **THEN** the recycle count is 3 and the restarts value is at least 3 (TestFreshGameCountsRecycleWithoutObserver)

#### Scenario: Never negative
- **WHEN** planned save points exceed the observed entries (0 entries, 3 planned) and no recycle ran
- **THEN** restartCount(0, 3, 0) is 0

### Requirement: Per-game memory is forgotten on every new game
On every new game the bot loop SHALL drop its per-game memory through the one shared helper forgetGame: the cleared-area set, sweepKills, needBuff (set true again), stepArea (0) and waitNote (false); both the observer path (syncGames, a menu-observed new game) and the bot-started fresh-game path (freshGameToLevel, d2d) MUST use it. Source: cards botsmall and smallfixes; docs/bot-small.md; test TestSyncGamesForgetsViaSharedHelper.

#### Scenario: New game seen through the menus
- **WHEN** the observer's game count moves while the loop holds a cleared area, sweep kills 5, stepArea 7, needBuff false and waitNote true
- **THEN** after syncGames the cleared set is empty, sweepKills is nil, stepArea is 0, needBuff is true and waitNote is false

#### Scenario: Bot-started fresh game on d2d
- **WHEN** freshGameToLevel starts a new game successfully with no menu poll
- **THEN** forgetGame runs, so the old game's cleared areas do not trigger another fresh game

### Requirement: Monster classification by flag bits
The monster kind SHALL come from the MonsterData flag byte with the champion bit (0x04) tested before the boss bit (0x08): a champion reads 0x0c and is Champion, never Unique (IsUnique false); 0x08 and 0x0a are Unique, 0x10 Minion, a bare 0x02 Normal, 0x00 Unknown. The bits are VERIFIED LIVE only by the original-game census (docs/orig-spawns.md), not by d2d; the +0x1C enchant offset is UNVERIFIED. Source: card botsmall; TestMonKindRealFlagBytes.

#### Scenario: Champion byte
- **WHEN** the flag byte is 0x0c
- **THEN** Kind is Champion, IsChampion is true and IsUnique is false

#### Scenario: Rare and superunique
- **WHEN** the flag byte is 0x08 or 0x0a
- **THEN** Kind is Unique and IsUnique is true

#### Scenario: Ordinary monster
- **WHEN** the flag byte is 0x00
- **THEN** Kind is Unknown, not Normal

### Requirement: Recovery without the kit stays near town
While recovery runs and the character has no healing potion in the belt or no town portal scroll (Normal act 1), the bot SHALL farm only Blood Moor (2) and Cold Plains (3), wait at home when neither is open, and MUST keep its walks out of the dungeon areas (8, 9, 13, 17, 18, 19). The knob `recoveryKitGate` (default 1) at 0 SHALL restore the old choice.

#### Scenario: Recovery area without the kit
- **WHEN** a recovering character has an empty belt or no portal scroll at clvl 7 or 14
- **THEN** the area is Blood Moor or Cold Plains, or none (TestKitGateRecoveryArea, TestKitGateNoSafeFieldWaits)

#### Scenario: Kit present keeps the old choice
- **WHEN** a recovering character has a healing potion and a portal scroll, or the knob is 0
- **THEN** the area equals the one chosen before the change (TestKitGateKeepsOldChoices)

#### Scenario: Walks avoid the dungeons
- **WHEN** the gate turns on or off
- **THEN** the navigator gets the six dungeon areas, or none; Route prefers a direct exit over a seen path through an avoided level, and an avoided destination stays routable (TestKitGateNavAvoid, TestRouteAvoidLevels)

### Requirement: Campaign stall escalation
While a goal runs, the bot SHALL escalate when the campaign makes no progress (kill, non-town area entered, purchase, quest step) on the virtual clock: after `bot.stallWarnMin` (30) minutes one WARN `campaign stalled: no progress for X virtual min` and a fresh game; after `bot.stallExitMin` (30) more, exit status 5 with `campaign stalled, exiting for a restart`. Progress resets it; 0 turns it off. Source: card nowall, docs/nowall.md.

#### Scenario: Legitimate wait
- **WHEN** the bot waits 28 virtual minutes (for example area bans of 3 min each) without progress
- **THEN** nothing is logged and no game is started

#### Scenario: Stall through both windows
- **WHEN** no progress for 31 virtual minutes, then 31 more after the fresh game
- **THEN** the first window logs the WARN and starts a fresh game, the second ends Run with ErrCampaignStalled (exit status 5)

#### Scenario: Progress resets
- **WHEN** a kill or purchase happens after the fresh game
- **THEN** the ladder returns to stage 0 and a full window must pass before the next WARN
