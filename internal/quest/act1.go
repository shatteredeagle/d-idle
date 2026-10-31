package quest

import (
	"context"
	"fmt"
)

// Act 1 area ids (levels.txt; V only for 1, the rest S/U). The nav layer owns the walking.
const (
	AreaBloodMoor      = 2
	AreaStonyField     = 4
	AreaDarkWood       = 5
	AreaBlackMarsh     = 6
	AreaDenOfEvil      = 8
	AreaBurialGrounds  = 17
	AreaForgottenTower = 20
	AreaTowerCellar5   = 25
	AreaBarracks       = 28
	AreaCatacombs4     = 37
	AreaTristram       = 38
)

// Quest item codes (S: items.txt codes as remembered; confirm with a pack dump, U).
const (
	ItemInifussScroll = "bks" // Scroll of Inifuss, from the Tree of Inifuss
	ItemCairnKey      = "bkd" // Key to the Cairn Stones, what Akara hands back after reading the scroll
	ItemMalus         = "hdm" // Horadric Malus, dropped by the Smith
)

// Handler is one quest as data. Pre are conditions to start, Steps the work. Req marks quests the critical path
// needs (the planner decides whether optional ones run). Skip says when a quest may be left out in
// Nightmare and Hell (policy, docs/progression/quests.md section 4).
type Handler struct {
	Quest string
	Name  string
	Req   bool
	Skip  bool // policy: skippable in NM/Hell
	Pre   []Cond
	Steps []Step
	Tag   string
}

// Act1 lists the Act 1 handlers in play order. Every dialog row and the claim-by-talking mechanics are U
// except Akara's Den of Evil row (V-04); see docs/quest.md for the live checks.
var Act1 = []Handler{
	{Quest: "den_of_evil", Name: "Den of Evil", Skip: true, Tag: "U (reward claim by re-talking Akara)",
		Steps: []Step{
			{Kind: KGoNPC, NPC: Akara},
			{Kind: KTalk, NPC: Akara},
			{Kind: KPick, NPC: Akara, Labels: []string{"Talk", "Den of Evil"}, Tag: "V", Note: "starts the quest: word 0x0004 (V-04)"},
			{Kind: KClose},
			{Kind: KGoArea, Area: AreaDenOfEvil},
			{Kind: KClear, Area: AreaDenOfEvil, Note: "every monster incl. champions and uniques"},
			{Kind: KToTown},
			{Kind: KGoNPC, NPC: Akara},
			{Kind: KTalk, NPC: Akara, Note: "U: the reward is granted by the Talk itself"},
			{Kind: KClose},
		}},
	{Quest: "sisters_burial_grounds", Name: "Sisters' Burial Grounds (Blood Raven)", Skip: false, Tag: "U",
		Steps: []Step{
			{Kind: KGoArea, Area: AreaBurialGrounds},
			{Kind: KKill, Target: "Blood Raven", Area: AreaBurialGrounds},
			{Kind: KToTown},
			{Kind: KGoNPC, NPC: Kashya},
			{Kind: KTalk, NPC: Kashya, Note: "U: Kashya's reward (merc hire) is offered by the Talk"},
			{Kind: KClose},
		}},
	{Quest: "search_for_cain", Name: "The Search for Cain (Scroll of Inifuss)", Skip: true, Tag: "U",
		Steps: []Step{
			{Kind: KGoArea, Area: AreaDarkWood},
			{Kind: KUseObject, Obj: ObjTreeInifuss, Area: AreaDarkWood, Note: "touch the tree; the scroll drops (S)"},
			{Kind: KPickup, Item: ItemInifussScroll},
			{Kind: KToTown},
			{Kind: KGoNPC, NPC: Akara},
			{Kind: KTurnIn, NPC: Akara, Item: ItemInifussScroll, Note: "S: Talk with the scroll in the pack makes Akara read it and return the Cairn key"},
			{Kind: KGoArea, Area: AreaStonyField},
			{Kind: KUseObject, Obj: ObjCairnStone, Area: AreaStonyField, Note: "U: the five stones must be touched in a fixed order; needs its own handler once the order is verified"},
			{Kind: KWaitArea, Area: AreaTristram},
			{Kind: KUseObject, Obj: ObjCainGibbet, Area: AreaTristram, Note: "U: free Cain after killing the Tristram monsters"},
			{Kind: KToTown},
			{Kind: KGoNPC, NPC: CainTown},
			{Kind: KTalk, NPC: CainTown, Note: "U: talking to Cain in town ends the quest"},
			{Kind: KClose},
		}},
	{Quest: "forgotten_tower", Name: "The Forgotten Tower (Countess)", Skip: true, Tag: "U",
		Steps: []Step{
			{Kind: KGoArea, Area: AreaForgottenTower},
			{Kind: KGoArea, Area: AreaTowerCellar5, Note: "stairs down; no waypoint in the tower (execution.md)"},
			{Kind: KKill, Target: "The Countess", Area: AreaTowerCellar5},
			{Kind: KToTown},
		}},
	{Quest: "tools_of_the_trade", Name: "Tools of the Trade (the Smith)", Skip: true, Tag: "U",
		Steps: []Step{
			{Kind: KGoArea, Area: AreaBarracks},
			{Kind: KKill, Target: "The Smith", Area: AreaBarracks},
			{Kind: KPickup, Item: ItemMalus},
			{Kind: KToTown},
			{Kind: KGoNPC, NPC: Charsi},
			{Kind: KTurnIn, NPC: Charsi, Item: ItemMalus, Note: "S: handing over is the Talk; the imbue pick (Imbue row, then an item) is a separate step"},
		}},
	{Quest: "sisters_to_the_slaughter", Name: "Sisters to the Slaughter (Andariel)", Req: true, Tag: "U",
		Steps: []Step{
			{Kind: KGoArea, Area: AreaCatacombs4},
			{Kind: KKill, Target: "Andariel", Area: AreaCatacombs4},
			{Kind: KToTown},
			{Kind: KGoNPC, NPC: Warriv},
			{Kind: KTalk, NPC: Warriv, Note: "U: the Talk records the quest; the ride is the A1->A2 hop"},
			{Kind: KClose},
		}},
}

// Handler1 returns the Act 1 handler for a quest id.
func Handler1(quest string) (Handler, bool) {
	for _, h := range Act1 {
		if h.Quest == quest {
			return h, true
		}
	}
	return Handler{}, false
}

// RunHandler runs one handler: skip on T1/T2 Done, check Pre, run Steps, re-read. A quest that still reads
// not-Done afterwards is NOT an error: the save lags until the next Save and Exit, so Confirmed is false and
// the task layer records the completion as T3 evidence and re-checks later (quests.md section 1).
func RunHandler(ctx context.Context, env Env, r Reader, diff int, h Handler) (Result, error) {
	if r.Skippable(diff, h.Quest) {
		return Result{Skipped: true, After: r.Quest(diff, h.Quest), Confirmed: true}, nil
	}
	for _, c := range h.Pre {
		if r.Status(diff, c.Quest) < c.Want {
			return Result{}, fmt.Errorf("%w: %s needs %s %s", ErrPrereq, h.Name, c.Quest, c.Want)
		}
	}
	if err := RunSteps(ctx, env, statusFn(r, diff), h.Steps); err != nil {
		return Result{Ran: true}, fmt.Errorf("%s: %w", h.Name, err)
	}
	after := r.Quest(diff, h.Quest)
	return Result{Ran: true, After: after, Confirmed: after.Status == Done && after.Evidence >= EvSave}, nil
}
