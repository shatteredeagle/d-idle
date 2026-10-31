## Why

Both navsoak runs hit a permanent town wedge (card townwedge fixes the cause) and then ran for hours logging "no eligible reachable leveling area" 52k and 173k times with 0 area entries and 0 purchases (docs/probes/navsoak-compare.md on agent/soakcompare). The loop watchdog never fired because the virtual clock kept advancing; 60 and 167 wall minutes were lost. Any future cause of a stall needs the same generic answer.

## What Changes

- internal/bot/stall.go: campaign progress (kill or xp, non-town area entered, purchase, quest step) is tracked on the virtual clock; after bot.stallWarnMin (30) without it the bot logs one WARN and starts a fresh game, after bot.stallExitMin (30) more it ends Run with ErrCampaignStalled; progress resets.
- cmd/d2bot: ErrCampaignStalled exits with status 5 (exitStalled), restarted by the supervisor like a storm.
- internal/env: the two knobs (0 on stallWarnMin restores the old behaviour); docs/knobs.md regenerated. Shared by both backends; motivated by d2d.
- Delta: bot-loop ADDED `Campaign stall escalation`.
