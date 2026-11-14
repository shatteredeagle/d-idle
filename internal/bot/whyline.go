package bot

import (
	"strconv"

	"d2lod-bot/internal/goals"
)

// The whyline (docs/whyline.md): at the places that already decide what the hero does next, the bot names the reason as
// a code plus parameters. The watch layer renders it from its template table (internal/watch/whyline.json); the bot
// never writes the sentence. Call setWhy right before the r.do or setState that starts the activity.

// Reason codes. Each one has a template in internal/watch/whyline.json (a test keeps the two lists equal).
const (
	WhyQuestStep      = "quest_step"    // step, level
	WhyWalk           = "walk"          // area, level
	WhyFight          = "fight"         // area, life
	WhyTownTrip       = "town_trip"     // why (the chore planner's own reason text, numbers included)
	WhyTownChores     = "town_chores"   // why
	WhyHealTrip       = "heal_trip"     // life, line
	WhyWaitHeal       = "wait_heal"     // why
	WhyClaimTrip      = "claim_trip"    // none
	WhyBackToField    = "back_to_field" // none
	WhyToTown         = "to_town"       // none
	WhyAreaDone       = "area_done"     // none
	WhyGuardExit      = "guard_exit"    // why
	WhyFarmRuns       = "farm_runs"     // area
	WhyFarmGold       = "farm_gold"     // level
	WhySavingFor      = "saving_for"    // area, item, need (poverty: the first must-have the gold does not cover)
	WhyFarmUser       = "farm_user"     // area, have, want (the player's Farm gold order)
	WhyAfterDeath     = "after_death_restock"
	WhyDeathCorpse    = "after_death_corpse"
	WhyDeathGold      = "after_death_gold"
	WhyPlayerItems    = "player_items"    // none (the item actions the player queued on the inventory page)
	WhyRepairWait     = "repair_wait"     // why (the repair trip waits for a safe moment, or loots first)
	WhyRepairTrip     = "repair_trip"     // step (going home, repairing, returning)
	WhyRepairFarm     = "repair_farm"     // area, have, want (Farm until repair)
	WhyJourneyBlocked = "journey_blocked" // why (the quest step is refused; the hero farms the level plan and retries)
)

// whyCodes lists every code above (a test compares it with the template table).
var whyCodes = []string{WhyQuestStep, WhyWalk, WhyFight, WhyTownTrip, WhyTownChores, WhyHealTrip, WhyWaitHeal, WhyClaimTrip,
	WhyBackToField, WhyToTown, WhyAreaDone, WhyGuardExit, WhyFarmRuns, WhyFarmGold, WhyFarmUser, WhySavingFor, WhyAfterDeath, WhyDeathCorpse, WhyDeathGold, WhyPlayerItems, WhyRepairWait, WhyRepairTrip, WhyRepairFarm, WhyJourneyBlocked}

// setWhy announces the reason for the activity the next setLocked sets. Parameters are key, value pairs.
func (r *Runner) setWhy(code string, kv ...string) {
	w := goals.Why{Code: code}
	if len(kv) > 1 {
		w.Params = make(map[string]string, len(kv)/2)
		for i := 0; i+1 < len(kv); i += 2 {
			w.Params[kv[i]] = kv[i+1]
		}
	}
	r.mu.Lock()
	r.pendingWhy = w
	r.mu.Unlock()
}

func itoa(n int) string { return strconv.Itoa(n) }

// stepTitle is the quest step's own title, or a neutral word while the planner has no step yet.
func stepTitle(has bool, title string) string {
	if !has || title == "" {
		return "the next step"
	}
	return title
}
