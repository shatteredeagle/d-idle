package questboard

import (
	"sort"
	"strings"
)

// Reason codes and their wording (design docs/design/fable-review-2026-11-21 appendix B.1). Go passes a code and
// arguments, never a sentence; one table builds the sentence for the page, the log and the scenario assertions.

// Remedy says who acts on a reason.
type Remedy string

const (
	RemedyChore  Remedy = "chore"  // a town visit fixes it: the bot goes now
	RemedyWait   Remedy = "wait"   // time or another run fixes it
	RemedyTrain  Remedy = "train"  // levels or gold fix it
	RemedyPlayer Remedy = "player" // a tap fixes it: Needs-you row
	RemedyNone   Remedy = "none"   // nothing fixes it here
)

// ReasonKind sorts the codes into the three groups of B.1.
type ReasonKind string

const (
	KindHold   ReasonKind = "hold"   // the bot acts or waits, the run stays live, no failure counted
	KindNeeds  ReasonKind = "needs"  // the run is awaiting or parked, a Needs-you row is raised
	KindNobody ReasonKind = "nobody" // the game failed it, or no handler exists
)

// ReasonInfo is one row of the closed reason table. BudgetMin is the virtual-minute budget of a hold (0: none).
type ReasonInfo struct {
	Code      string
	Kind      ReasonKind
	Remedy    Remedy
	BudgetMin int
	Text      string // {name} placeholders are filled by Sentence
}

var reasonTable = []ReasonInfo{
	{"waits_for", KindHold, RemedyWait, 0, "Starts after {part}."},
	{"fight_first", KindHold, RemedyWait, 0, "Starts when the fight is over."},
	{"recovery_first", KindHold, RemedyWait, 60, "Starts after the hero has refilled the purse and restocked."},
	{"gold_order_first", KindHold, RemedyWait, 0, "Your Farm gold order comes first."},
	{"order_first", KindHold, RemedyWait, 0, "Your order comes first; {quest} continues after it."},
	{"needs_potions", KindHold, RemedyChore, 20, "Going to town for potions: {have} of {need} {kind} potions."},
	{"needs_gold_withdraw", KindHold, RemedyChore, 20, "Taking gold from the stash to restock."},
	{"needs_repair", KindHold, RemedyChore, 20, "Going to town to repair."},
	{"needs_kit", KindHold, RemedyChore, 20, "Going to town for {item}."},
	{"needs_gold_farm", KindHold, RemedyTrain, 60, "Not enough gold to restock: farming gold first."},
	{"too_weak", KindHold, RemedyTrain, 60, "Too hard for now: {why}. Training toward it."},
	{"route_closed", KindHold, RemedyTrain, 60, "The road to {area} is closed for now: {why}."},
	{"area_banned", KindHold, RemedyWait, 0, "{area} is closed after {n} deaths; trying again later."},
	{"handin_in_town", KindHold, RemedyWait, 5, "Hand-in is on its way: the hero is going to town."},
	{"handin_waiting", KindHold, RemedyWait, 5, "Hand-in is queued: {why}"},
	{"confirming", KindHold, RemedyWait, 0, "Handed in. Checking that the game agrees."},
	{"loading", KindHold, RemedyWait, 2, "Waiting for the game."},

	{"handin_wanted", KindNeeds, RemedyPlayer, 0, "{npc} is waiting: hand in {quest}."},
	{"paused", KindNeeds, RemedyPlayer, 0, "Paused: the hero continues when you resume."},
	{"gave_up", KindNeeds, RemedyPlayer, 0, "{quest} is stuck: gave up after {n} tries: {why}"},
	{"no_path", KindNeeds, RemedyPlayer, 0, "{quest} is stuck: no walkable way after {n} tries: {why}"},
	{"refused_repeatedly", KindNeeds, RemedyPlayer, 0, "{quest} is paused: the same step was refused {n} times in a row ({code}): {why}"},
	{"died_too_often", KindNeeds, RemedyPlayer, 0, "{quest} is stuck: the hero died {n} times on this part."},
	{"paused_after_deaths", KindNeeds, RemedyPlayer, 0, "{quest} is paused: the hero died {n} times on this part."},
	{"handin_failed", KindNeeds, RemedyPlayer, 0, "The hand-in did not go through: {why}"},
	{"prereq_regressed", KindNeeds, RemedyPlayer, 0, "{part} no longer reads as done, and {quest} needs it."},
	{"prereq_missing", KindNeeds, RemedyPlayer, 0, "{part} has to be done first and cannot start now: {why}"},
	{"quest_locked", KindNeeds, RemedyPlayer, 0, "{quest} is locked again: {other} no longer reads as done."},
	{"hold_timeout", KindNeeds, RemedyPlayer, 0, "Still waiting after {n} minutes: {hold}"},

	{"game_failed", KindNobody, RemedyNone, 0, "Failed in this game. It can be done again in the next difficulty."},
	{"unsupported", KindNobody, RemedyNone, 0, "Not supported yet: {why}"},
}

var reasonByCode = func() map[string]ReasonInfo {
	m := map[string]ReasonInfo{}
	for _, r := range reasonTable {
		m[r.Code] = r
	}
	return m
}()

// ReasonOf looks a code up; an unknown code is not a reason.
func ReasonOf(code string) (ReasonInfo, bool) { r, ok := reasonByCode[code]; return r, ok }

// ReasonCodes lists every code, sorted.
func ReasonCodes() []string {
	var out []string
	for c := range reasonByCode {
		out = append(out, c)
	}
	sort.Strings(out)
	return out
}

// Sentence builds the player's text of a code; a placeholder without an argument stays visible so a missing
// argument shows in a screenshot instead of reading as a complete sentence.
func Sentence(code string, args map[string]string) string {
	r, ok := reasonByCode[code]
	if !ok {
		return "No reason was given (report this)"
	}
	s := r.Text
	for k, v := range args {
		s = strings.ReplaceAll(s, "{"+k+"}", v)
	}
	return s
}
