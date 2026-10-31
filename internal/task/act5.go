package task

import (
	"context"
	"time"
)

// Normal Act 5 quest handlers (docs/act5.md). Same shape as act4.go. Tags: area ids and boss classes are V (read from
// the real 1.14d tables, quest/act45_real_test.go re-checks them); every object name the adapter must understand, every
// talk option, the Anya/Malah flow, Larzuk's socket UI and the Rescue mechanics are U until the live checks A5-6..A5-8.

// Act 5 areas (V levels.txt). Halls of Pain is 123 and Vaught 124 (older docs said 122/123).
const (
	Harrogath           Area = 109
	BloodyFoothills     Area = 110
	FrigidHighlands     Area = 111
	CrystallinePassage  Area = 113
	FrozenRiver         Area = 114
	AncientsWay         Area = 118
	ArreatSummit        Area = 120
	NihlathakTemple     Area = 121
	HallsOfPain         Area = 123
	HallsOfVaught       Area = 124
	WorldstoneKeep2     Area = 129
	ThroneOfDestruction Area = 131
	WorldstoneChamber   Area = 132
)

// Act 5 quest ids. They are the save-word ids; the Rescue and mercenary groups are ours.
const (
	QSiege    = "siege_harrogath"
	QRescue   = "rescue_arreat"
	QPrison   = "prison_of_ice"
	QBetrayal = "betrayal_harrogath"
	QAncients = "rite_of_passage"
	QBaal     = "eve_of_destruction"
	QMercAct5 = "merc_act5"

	NPCLarzuk   = "Larzuk"
	NPCMalah    = "Malah"
	NPCAnya     = "Anya"
	NPCQualKehk = "Qual-Kehk"

	BossShenk     = "Shenk the Overseer"
	BossNihlathak = "Nihlathak"
	BossAncients  = "The Ancients"
	BossBaal      = "Baal"

	// Item display names (U: not dumped; codes ice/tr2 are V in quest/act5.go).
	ItemMalahPotion = "Malah's Potion"
	ItemResScroll   = "Scroll of Resistance"
	// Pseudo items the adapter sets (U), like ItemOrbSmashed: Anya touched, thawed with the potion, scroll used.
	ItemAnyaTouched = "Anya touched"
	ItemAnyaThawed  = "Anya thawed"
	ItemScrollRead  = "Scroll of Resistance read"
)

// Soft floors of the Normal Act 5 campaign. Source: the work list in docs/act5.md (runs.json R: Shenk 25, Ancients
// 20/28, Baal 32). The same doc notes progression-normal.json wants 36 for the summit and 38 for the throne and that the
// area levels are 29..43 (V), so a weak build may want these raised; they are constants for that reason.
const (
	floorShenk     = 25
	floorRescue    = 25
	floorPrison    = 25
	floorNihlathak = 30
	floorAncients  = 36
	floorBaal      = 38
)

// ItemUser is the optional capability behind reading a scroll: right-click the named pack item (U: Act2Env has it).
type ItemUser interface {
	UseItem(ctx context.Context, item string) error
}

func init() {
	handlers[QSiege] = siegeNodes
	handlers[QRescue] = rescueNodes
	handlers[QPrison] = prisonNodes
	handlers[QBetrayal] = betrayalNodes
	handlers[QAncients] = ancientsNodes
	handlers[QBaal] = baalNodes
	handlers[QMercAct5] = merc5Nodes
}

// a5Stage is the reader's stage raised by what the world proves.
func a5Stage(w World, q string) Stage {
	s := w.Quest(q).Stage
	raise := func(p Stage) {
		if p > s {
			s = p
		}
	}
	switch q {
	case QSiege:
		if w.Killed[BossShenk] {
			raise(StageCleared)
		}
	case QPrison:
		if w.Items[ItemAnyaTouched] > 0 || w.Items[ItemMalahPotion] > 0 {
			raise(StageStarted)
		}
		if w.Items[ItemAnyaThawed] > 0 || w.Items[ItemResScroll] > 0 {
			raise(StageCleared)
		}
		if w.Items[ItemScrollRead] > 0 || w.Killed[BossNihlathak] || w.Killed[BossAncients] || w.Killed[BossBaal] {
			raise(StageDone)
		}
	case QBetrayal:
		if w.Killed[BossNihlathak] {
			raise(StageCleared)
		}
	case QAncients:
		if w.Killed[BossAncients] {
			raise(StageCleared)
		}
		if w.Killed[BossBaal] || w.Area == WorldstoneChamber {
			raise(StageDone)
		}
	case QBaal:
		if w.Killed[BossBaal] {
			raise(StageDone)
		}
	}
	return s
}

func a5At(q string, s Stage) Pred { return func(w World) bool { return a5Stage(w, q) >= s } }
func a5Is(q string, s Stage) Pred { return func(w World) bool { return a5Stage(w, q) == s } }

// Siege on Harrogath: Shenk in the Bloody Foothills (no waypoint, V: the walk starts in town), then Larzuk adds sockets
// to one item (the item pick is the adapter's, U). Shenk's area level is 24 (V); floor 25 (R).
func siegeNodes() []Node {
	q := QSiege
	killed := a5At(q, StageCleared)
	return []Node{
		{Step: withReady(StepTravelA2(id(q, "to"), "Go to the Bloody Foothills", BloodyFoothills, killed), gatePotions(floorShenk, "Shenk needs clvl 25", minPotions, q))},
		req(Node{Step: StepKill(id(q, "kill"), BossShenk, BloodyFoothills, killed, 12*time.Minute)}, id(q, "to")),
		{Step: home2(id(q, "home"), BloodyFoothills, a5Is(q, StageCleared))},
		{Step: StepTalk(id(q, "reward"), "Larzuk adds sockets", NPCLarzuk, "sockets", not(a5Is(q, StageCleared)), nil)},
	}
}

// Rescue on Mount Arreat (optional, can fail, V text): open the three cages in the Frigid Highlands, then Qual-Kehk.
// Without a reader nothing shows the work was done, so the group runs once per session (Session.Finished).
func rescueNodes() []Node {
	q := QRescue
	done := a5At(q, StageCleared)
	cages := Step{ID: id(q, "cages"), Title: "Free the three prisoners", Phase: "fight", Budget: 10 * time.Minute, Est: 4 * time.Minute, Optional: true,
		Need: needIf(done, func(w World) bool { return w.Area == FrigidHighlands }),
		Do: func(ctx context.Context, x Actions) error {
			var first error
			for i := 0; i < 3; i++ {
				if err := x.Interact(ctx, "cage"); err != nil {
					if i == 0 {
						first = err
					}
					break
				}
			}
			return first
		}}
	return []Node{
		{Step: StepWaypoint(id(q, "wp"), FrigidHighlands, done)},
		{Step: withReady(StepTravelA2(id(q, "to"), "Go to the Frigid Highlands", FrigidHighlands, done), gatePotions(floorRescue, "Rescue needs clvl 25", minPotions, q))},
		req(Node{Step: cages}, id(q, "to")),
		{Step: home2(id(q, "home"), FrigidHighlands, func(w World) bool { return !done(w) })},
		{Step: optional(StepTalk(id(q, "reward"), "Talk to Qual-Kehk", NPCQualKehk, "rescue", not(a5Is(q, StageCleared)), nil))},
	}
}

// Prison of Ice (S Kolbot Questing.anya): touch frozen Anya in the Frozen River (off the Crystalline Passage), Malah gives
// the potion, thaw her, Malah gives the Scroll of Resistance, read it (+10 all resists), Anya opens the portal to
// Nihlathak. Every step's trigger is a pseudo item the adapter sets (U).
func prisonNodes() []Node {
	q := QPrison
	done := a5At(q, StageDone)
	touched := func(w World) bool { return a5Stage(w, q) >= StageStarted }
	thawed := func(w World) bool { return a5Stage(w, q) >= StageCleared }
	inRiver := func(w World) bool { return w.Area == FrozenRiver }
	touch := StepInteract(id(q, "touch"), "Touch frozen Anya", "Frozen Anya", touched, inRiver)
	thaw := StepInteract(id(q, "thaw"), "Thaw Anya with the potion", "Frozen Anya", thawed,
		func(w World) bool { return inRiver(w) && w.Items[ItemMalahPotion] > 0 })
	read := Step{ID: id(q, "read"), Title: "Read the Scroll of Resistance", Phase: "chore", Budget: 2 * time.Minute, Est: 10 * time.Second,
		Need: needIf(done, func(w World) bool { return w.Items[ItemResScroll] > 0 }),
		Do: func(ctx context.Context, x Actions) error {
			u, ok := x.(ItemUser)
			if !ok {
				return Errorf(KindPrereq, "the bot cannot right-click a pack item")
			}
			return u.UseItem(ctx, ItemResScroll)
		}}
	toRiver := func(sid string, done Pred) Step {
		return withReady(StepTravelA2(id(q, sid), "Go to the Frozen River", FrozenRiver, done), gatePotions(floorPrison, "Anya needs clvl 25", minPotions, q))
	}
	return []Node{
		{Step: StepWaypoint(id(q, "wp"), CrystallinePassage, done)},
		{Step: toRiver("to", func(w World) bool { return done(w) || touched(w) })},
		req(Node{Step: touch}, id(q, "to")),
		{Step: home2(id(q, "home1"), FrozenRiver, func(w World) bool { return touched(w) && !done(w) && w.Items[ItemMalahPotion] == 0 && !thawed(w) })},
		{Step: StepTalk(id(q, "potion"), "Malah gives the potion", NPCMalah, "potion",
			func(w World) bool { return done(w) || thawed(w) || w.Items[ItemMalahPotion] > 0 || !touched(w) }, nil)},
		{Step: toRiver("to2", func(w World) bool { return done(w) || thawed(w) || w.Items[ItemMalahPotion] == 0 })},
		req(Node{Step: thaw}, id(q, "to2")),
		{Step: home2(id(q, "home2"), FrozenRiver, func(w World) bool { return thawed(w) && !done(w) && w.Items[ItemResScroll] == 0 })},
		{Step: StepTalk(id(q, "scroll"), "Malah gives the Scroll of Resistance", NPCMalah, "scroll",
			func(w World) bool { return done(w) || w.Items[ItemResScroll] > 0 || !thawed(w) }, nil)},
		{Step: read},
	}
}

// Betrayal of Harrogath (optional): Anya's red portal in town leads to Nihlathak's Temple; the waypoint of the Halls of
// Pain is activated on the way, then Vaught (Nihlathak, SU 60, class 526). Anya personalizes an item afterwards (the
// item pick is the adapter's, U). The portal object name is U (docs/act5.md open question 1).
func betrayalNodes() []Node {
	q := QBetrayal
	killed := a5At(q, StageCleared)
	portal := StepInteract(id(q, "portal"), "Take Anya's red portal", "Anya's portal", killed,
		func(w World) bool { return w.InTown && !w.WPs[HallsOfPain] && a5At(QPrison, StageDone)(w) })
	return []Node{
		{Step: portal},
		{Step: withReady(StepTravelA2(id(q, "to-pain"), "Go to the Halls of Pain", HallsOfPain, func(w World) bool { return killed(w) || w.Area == HallsOfVaught }), gatePotions(floorNihlathak, "Nihlathak needs clvl 30", minPotions, q))},
		{Step: StepWaypoint(id(q, "wp"), HallsOfPain, killed)},
		{Step: StepTravelA2(id(q, "to"), "Go to the Halls of Vaught", HallsOfVaught, killed)},
		req(Node{Step: StepKill(id(q, "kill"), BossNihlathak, HallsOfVaught, killed, 12*time.Minute)}, id(q, "to")),
		{Step: home2(id(q, "home"), HallsOfVaught, a5Is(q, StageCleared))},
		{Step: StepTalk(id(q, "reward"), "Anya personalizes an item", NPCAnya, "personalize", not(a5Is(q, StageCleared)), nil)},
	}
}

// Rite of Passage: the Ancients' Way waypoint, the Arreat Summit, boss.RunAncients touches the altar and kills Talic,
// Madawc and Korlic with no town portal and without leaving the area (V text), so the kill step carries both
// constraints and the potion floor is checked in Ready, never mid-fight. Credit needs clvl 20 (V text).
func ancientsNodes() []Node {
	q := QAncients
	killed := a5At(q, StageCleared)
	kill := StepKill(id(q, "kill"), BossAncients, ArreatSummit, killed, 20*time.Minute)
	kill.Constraints = Constraints{NoTownPortal: true, NoLeaveArea: true}
	return []Node{
		{Step: StepWaypoint(id(q, "wp"), AncientsWay, killed)},
		{Step: withReady(StepTravelA2(id(q, "to"), "Go to the Arreat Summit", ArreatSummit, killed), gatePotions(floorAncients, "the Ancients need clvl 36", 6, q))},
		req(Node{Step: kill}, id(q, "to")),
		{Step: home2(id(q, "home"), ArreatSummit, a5Is(q, StageCleared))},
		{Step: optional(StepTalk(id(q, "reward"), "Talk to Qual-Kehk", NPCQualKehk, "rite of passage", not(a5Is(q, StageCleared)), nil))},
	}
}

// Eve of Destruction: the Worldstone Keep 2 waypoint, the Throne, boss.RunBaal fights the waves, takes the portal and
// kills Baal, so the kill step spans two areas (a restart inside the chamber must still run it). The summit stairs open
// only after the Ancients (S); the campaign order guarantees that, a Prereq check would break a proxy-only restart.
func baalNodes() []Node {
	q := QBaal
	killed := a5At(q, StageDone)
	toThrone := withReady(StepTravelA2(id(q, "to"), "Go to the Throne of Destruction", ThroneOfDestruction,
		func(w World) bool { return killed(w) || w.Area == WorldstoneChamber }), gatePotions(floorBaal, "Baal needs clvl 38", 5, q))
	kill := StepKill(id(q, "kill"), BossBaal, ThroneOfDestruction, killed, 45*time.Minute)
	kill.Need = needIf(killed, func(w World) bool { return w.Area == ThroneOfDestruction || w.Area == WorldstoneChamber })
	return []Node{
		{Step: StepWaypoint(id(q, "wp"), WorldstoneKeep2, killed)},
		{Step: toThrone},
		req(Node{Step: kill}, id(q, "to")),
	}
}

// Qual-Kehk's barbarian hire (price factor 9000, V hireling.txt: expensive; the row and the price floor are U).
func merc5Nodes() []Node {
	q := QMercAct5
	hire := Step{ID: id(q, "hire"), Title: "Hire a Barbarian", Phase: "chore", Budget: 3 * time.Minute, Est: 30 * time.Second,
		Need: func(w World) (bool, Evidence) {
			return w.Items[ItemMercenary] == 0 && !w.HasMerc && w.Act >= 5, EvMemory
		},
		Ready: func(w World) *Fault {
			switch {
			case !w.InTown:
				return &Fault{Kind: KindPrereq, Why: "Qual-Kehk is in town", Insert: []Step{StepGoHome(id(q, "hire/home"))}}
			case w.Gold < 20000:
				return &Fault{Kind: KindResource, Why: "not enough gold to hire (U)"}
			}
			return nil
		},
		Do: func(ctx context.Context, x Actions) error {
			h, ok := x.(MercHirer)
			if !ok {
				return Errorf(KindPrereq, "the bot has no mercenary hire capability")
			}
			return h.HireMerc(ctx, 5, "Bash")
		}}
	return []Node{{Step: hire}}
}
