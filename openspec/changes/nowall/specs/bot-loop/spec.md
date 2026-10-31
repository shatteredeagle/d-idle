## ADDED Requirements

### Requirement: Campaign stall escalation
While a goal is active and not paused, the bot SHALL measure campaign progress (a kill or xp gain, a non-town area entered, a purchase, a quest step) on the virtual clock. After `bot.stallWarnMin` (default 30) virtual minutes without it the bot MUST log one WARN `campaign stalled: no progress for X virtual min` with the last progress time and state, and start a fresh game through the shared path (newGameRun plus forgetGame). If still no progress after `bot.stallExitMin` (default 30) more virtual minutes it MUST end Run with ErrCampaignStalled and d2bot MUST exit with status 5 and the message `campaign stalled, exiting for a restart`, so the supervisor restarts the process. Any progress resets the ladder; `bot.stallWarnMin` 0 turns it off. The default exceeds the longest legitimate no-progress stretch in the navsoak logs (19 virtual min). Source: card nowall; docs/nowall.md; tests TestStallLadder, TestStallLegitimateWaitsQuiet, TestStallProgressResets, TestHaltErrorCampaignStalled.

#### Scenario: Legitimate wait
- **WHEN** the bot waits 28 virtual minutes (for example area bans of 3 min each) without progress
- **THEN** nothing is logged and no game is started

#### Scenario: Stall through both windows
- **WHEN** no progress for 31 virtual minutes, then 31 more after the fresh game
- **THEN** the first window logs the WARN and starts a fresh game, the second ends Run with ErrCampaignStalled (exit status 5)

#### Scenario: Progress resets
- **WHEN** a kill or purchase happens after the fresh game
- **THEN** the ladder returns to stage 0 and a full window must pass before the next WARN
