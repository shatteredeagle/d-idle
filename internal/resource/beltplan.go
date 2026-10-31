package resource

import "fmt"

// Soak numbers (docs/soak-analysis.md, run 104840, V from the log): 22 healing potions drunk, 19 of them at 34 to 45 percent
// life (the drink line is 55 but the fight loop reacts late), 3 at 16 to 19. Lowering the line would only drink later; the
// waste is the second and third potion of one fight in an area next to town, so with a thin belt we leave instead.

// thinBeltRetreat reports whether a fight with life under the drink line should be left (portal, or the walk home from the
// first fields) instead of spending one of the last healing potions. Never near the chicken line: there the drink is faster.
func thinBeltRetreat(in Input, t Tuning) bool {
	if t.RetreatBelt <= 0 || in.Hostiles <= 0 || !(in.HasTP || in.NearTown) {
		return false
	}
	if TripBudgetSpent(in, t) != "" {
		return false
	}
	lp := in.LifePct()
	if lp >= t.DrinkLifePct || lp < max(t.DrinkRejuvPct, t.Chicken(in.Diff)+8) {
		return false
	}
	have := in.Belt.HP + in.Belt.RV
	if t.RetreatCountPack != 0 {
		// Spares in the pack are one belt fill away (beltFillStep, between fights): drink and fight on instead of
		// running home with a full backpack.
		have += in.Belt.Spare()
	}
	return have <= t.RetreatBelt
}

// TripBudgetSpent says why a thin belt must not send the bot home again ("" = the trip is affordable). A caster that
// portals out every minute pays a scroll and a few potions each time (V, G1 attempt 8 log: 14 trips, 2160 gold in 15
// minutes against Blood Raven at clvl 9); past the budget, or with too little gold to restock, it drinks what is left
// and fights on.
func TripBudgetSpent(in Input, t Tuning) string {
	switch {
	case t.TripBudget > 0 && in.TripsRecent >= t.TripBudget:
		return fmt.Sprintf("trip budget: %d trips in %d min (limit %d)", in.TripsRecent, t.TripWindowMin, t.TripBudget)
	case t.TripGoldFloor > 0 && in.Gold < t.TripGoldFloor:
		return fmt.Sprintf("trip budget: %d gold under the floor %d, a trip could not restock", in.Gold, t.TripGoldFloor)
	}
	return ""
}

// FieldVerdict is the FieldGate answer.
type FieldVerdict struct {
	Go        bool
	Restock   bool // not ready and not poor: shop again before leaving
	BloodMoor bool // poor: only the Blood Moor (next to the camp) is allowed
	HaveHP    int
	WantHP    int
	Reason    string
}

// FieldGate decides whether the character may leave town for the field. With MinFieldBeltPct set it refuses a belt below
// that share of the planned healing potions (and a missing town portal): a character that can pay shops again, a poor one
// farms the Blood Moor only (the walk home works there). Off (0) always says Go.
func FieldGate(in Input, t Tuning) FieldVerdict {
	plan := PlanForBuild(in.Class, in.Level, in.Diff, in.Build)
	want := plan.capacity(PotionLife, in.Belt.Rows)
	v := FieldVerdict{HaveHP: in.Belt.HP + in.Belt.RV, WantHP: want}
	if t.MinFieldBeltPct <= 0 || want == 0 {
		v.Go, v.Reason = true, "belt gate off"
		return v
	}
	thin := v.HaveHP*100 < want*t.MinFieldBeltPct
	if !thin && in.HasTP {
		v.Go, v.Reason = true, "belt ready"
		return v
	}
	why := fmt.Sprintf("%d of %d healing potions", v.HaveHP, want)
	if !in.HasTP {
		why += ", no town portal"
	}
	// Poor: the gold cannot buy even the cheapest healing potion above the reserve.
	if in.Poor || in.Gold < t.ReserveGold+t.KitPotionGold || (!thin && in.Gold < t.KitTPGold) {
		v.BloodMoor, v.Reason = true, "poor ("+why+"): Blood Moor only"
		return v
	}
	v.Restock, v.Reason = true, "belt thin ("+why+"): restock before the field"
	return v
}
