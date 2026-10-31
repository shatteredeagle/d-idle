package quest

import (
	"context"
	"fmt"
)

// Town and hub area ids (levels.txt): Rogue Encampment 1 (V), Lut Gholein 40, Kurast Docks 75,
// Pandemonium Fortress 103, Harrogath 109 (U except 1; town/acts.go carries the same table).
const (
	AreaRogueEncampment = 1
	AreaLutGholein      = 40
	AreaKurastDocks     = 75
	AreaFortress        = 103
	AreaHarrogath       = 109
)

// Hop is an act transition as a step list. Pre are the quest conditions that must hold (T1/T2 evidence is
// preferred by the caller; a Pre that is only met by proxy still blocks, per docs/progression/quests.md);
// FromArea is where the player must stand (0 = anywhere, the first step then is KToTown); Success is detected
// through Player.Area == To, which is V as a check method (nav.md) though the area ids are U.
type Hop struct {
	Name     string
	FromAct  int
	ToAct    int
	FromArea int
	To       int // area id on arrival
	Pre      []Cond
	Steps    []Step
	Tag      string
}

// Hops are the first-trip transitions (docs/progression/acts.md section 2). Every later trip is a waypoint
// tab (act tabs 2-5 are U, V-06), so each hop ends with KTouchWP.
var Hops = []Hop{
	{Name: "A1->A2 Warriv", FromAct: 1, ToAct: 2, FromArea: AreaRogueEncampment, To: AreaLutGholein,
		Pre: []Cond{{"sisters_to_the_slaughter", Done}},
		Steps: []Step{
			{Kind: KGoNPC, NPC: Warriv, Tag: "V", Note: "typed monster lookup: Warriv and the waypoint are both id 11 (nav.md)"},
			{Kind: KTalk, NPC: Warriv},
			{Kind: KPick, NPC: Warriv, Labels: []string{"Go East"}, Tag: "U", Note: "row index not live-verified"},
			{Kind: KWaitArea, Area: AreaLutGholein, Tag: "U"},
			{Kind: KTouchWP},
		}, Tag: "U"},
	{Name: "A2->A3 Meshif", FromAct: 2, ToAct: 3, FromArea: AreaLutGholein, To: AreaKurastDocks,
		Pre: []Cond{{"seven_tombs", Done}},
		Steps: []Step{
			{Kind: KGoNPC, NPC: Jerhyn, Note: "only if act2_done is not set yet; the Tyrael/Jerhyn talks end the quest (U)"},
			{Kind: KTalk, NPC: Jerhyn},
			{Kind: KClose},
			{Kind: KGoNPC, NPC: Meshif},
			{Kind: KTalk, NPC: Meshif},
			{Kind: KPick, NPC: Meshif, Labels: []string{"Sail East"}, Tag: "U"},
			{Kind: KWaitArea, Area: AreaKurastDocks, Tag: "U"},
			{Kind: KTouchWP},
		}, Tag: "U"},
	{Name: "A3->A4 red portal", FromAct: 3, ToAct: 4, FromArea: 102, To: AreaFortress,
		Pre: []Cond{{"guardian", Done}},
		Steps: []Step{
			{Kind: KUseObject, Obj: ObjRedPortalA3, Area: 102, Tag: "S object 342 (Kolbot RedPortalToAct4)", Note: "portal appears inside Durance of Hate 3 after Mephisto; gone if the game is left"},
			{Kind: KWaitArea, Area: AreaFortress, Tag: "U"},
			{Kind: KTouchWP},
		}, Tag: "U"},
	{Name: "A4->A5 Tyrael", FromAct: 4, ToAct: 5, FromArea: AreaFortress, To: AreaHarrogath,
		Pre: []Cond{{"terrors_end", Done}},
		Steps: []Step{
			{Kind: KGoNPC, NPC: Tyrael},
			{Kind: KTalk, NPC: Tyrael},
			{Kind: KPick, NPC: Tyrael, Labels: []string{"Travel to Harrogath"}, Tag: "U"},
			{Kind: KWaitArea, Area: AreaHarrogath, Tag: "U"},
			{Kind: KTouchWP},
		}, Tag: "U (needs the expansion flag)"},
}

// HopFor returns the hop leaving act `from`, or false.
func HopFor(from int) (Hop, bool) {
	for _, h := range Hops {
		if h.FromAct == from {
			return h, true
		}
	}
	return Hop{}, false
}

// ErrWrongPlace means the player is not where the hop starts.
var ErrWrongPlace = fmt.Errorf("quest: hop precondition: wrong area")

// RunHop checks the preconditions, runs the steps and detects success by Player.Area == h.To. The Jerhyn leg of
// A2->A3 is skipped when the Seven Tombs quest already reads Done (the talk would only repeat).
func RunHop(ctx context.Context, env Env, r Reader, diff int, h Hop) (Result, error) {
	if env.Area() == h.To {
		return Result{Skipped: true}, nil
	}
	if ok, c := r.Met(diff, h.Pre); !ok {
		return Result{}, fmt.Errorf("%w: %s needs %s %s", ErrPrereq, h.Name, c.Quest, c.Want)
	}
	if h.FromArea != 0 && env.Area() != h.FromArea {
		return Result{}, fmt.Errorf("%w: %s starts in %d, player in %d", ErrWrongPlace, h.Name, h.FromArea, env.Area())
	}
	steps := h.Steps
	if h.Name == "A2->A3 Meshif" && r.Skippable(diff, "act2_done") {
		steps = steps[3:] // drop the Jerhyn leg
	}
	if err := RunSteps(ctx, env, statusFn(r, diff), steps); err != nil {
		return Result{Ran: true}, err
	}
	if env.Area() != h.To {
		return Result{Ran: true}, fmt.Errorf("quest: %s finished but area is %d, want %d", h.Name, env.Area(), h.To)
	}
	return Result{Ran: true, Confirmed: true}, nil
}
