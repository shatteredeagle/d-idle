package quest

// Normal Act 5 (Harrogath) as data. Tags follow docs/act5.md: V = read from the real 1.14d tables on 2026-10-29 by
// ACT3PLAN (act5_real_test.go re-checks them where /tmp/d2mpq exists), S = Kolbot / Arreat Summit, U = unverified
// in the running client. No live game was used. AreaHarrogath = 109 is in hops.go.
//
// Correction (V levels.txt): Halls of Anguish is 122, Halls of Pain 123 (waypoint 35), Halls of Vaught 124; older
// docs say 122/123 for the last two.
const (
	AreaBloodyFoothills   = 110 // Shenk, no waypoint
	AreaFrigidHighlands   = 111 // data name "Rigid Highlands", waypoint 31
	AreaArreatPlateau     = 112
	AreaCrystalPassage    = 113 // data name "Crystalized Cavern 1"
	AreaFrozenRiver       = 114 // data name "Cellar of Pity", Anya
	AreaGlacialTrail      = 115
	AreaFrozenTundra      = 117
	AreaAncientsWay       = 118 // data name "Glacial Caves 1", waypoint 37
	AreaIcyCellar         = 119
	AreaArreatSummit      = 120 // data name "Rocky Summit"
	AreaNihlathakTemple   = 121
	AreaHallsOfAnguish    = 122
	AreaHallsOfPain       = 123 // waypoint 35
	AreaHallsOfVaught     = 124
	AreaWorldstoneKeep1   = 128
	AreaWorldstoneKeep2   = 129 // waypoint 38
	AreaWorldstoneKeep3   = 130
	AreaThrone            = 131
	AreaWorldstoneChamber = 132
)

// Act 5 NPC classes (V monstats hcIdx). Tyrael's Act 5 form `tyrael3` 521 and Cain `cain6` 520 stand in Harrogath
// after the hop (which one is present is U).
const (
	Larzuk       = 511
	AnyaNPC      = 512 // `drehya`; frozen form `drehyaiced` 527
	Malah        = 513
	NihlathakNPC = 514 // the town NPC; the boss is 526
	QualKehk     = 515
	CainA5       = 520
	TyraelA5     = 521
)

// Boss and monster classes (V monstats): Shenk `overseer1` 479, Nihlathak `nihlathakboss` 526, the Ancients 540/541/542
// (Talic, Madawc, Korlic), Baal throne form 543, Baal in the chamber `baalcrab` 544, `baalclone` 570, Lister 571.
const (
	ClassShenk      = 479
	ClassNihlathak  = 526
	ClassTalic      = 540
	ClassMadawc     = 541
	ClassKorlic     = 542
	ClassBaalThrone = 543
	ClassBaal       = 544
	ClassBaalClone  = 570
	ClassCouncilA5  = 557 // wave 3 council member (S)
	ClassLister     = 571
	ClassPrisonDoor = 434 // V; Rescue on Mount Arreat
	ClassCaptive    = 534 // V
)

// Act 5 objects. Names V (objects.txt), roles S Kolbot sdk.js.
const (
	ObjWaypointA5     = 429
	ObjAnyaPlatform   = 460 // V "dummy"
	ObjNihlathakPlat  = 462 // V "dummy"
	ObjCage           = 473 // V "cagedwussie1"
	ObjAncientsAltar  = 546 // V "ancientsaltar"
	ObjStatueTalic    = 474 // V "Ancient Statue 3/1/2", which is which is S
	ObjStatueMadawc   = 475
	ObjStatueKorlic   = 476
	ObjToWSK1         = 547 // V "To The Worldstone Keep Level 1"
	ObjBaalPortal     = 563 // V "The Worldstone Chamber"
	ObjThroneEntrance = 569 // V "Throne of Destruction"
	ObjFrozenAnya     = 558 // V "fana"
)

// Quest item codes (V items.txt).
const (
	ItemMalahPotion  = "ice" // "Keep it to thaw Anya"
	ItemResistScroll = "tr2" // Scroll of Resistance, "Right Click to Cast"
)

// Act5 lists the Normal Act 5 handlers in play order. Req marks the critical path to Baal. Every dialog row,
// claim mechanic and the Anya/Larzuk UI flows are U (docs/act5.md).
var Act5 = []Handler{
	{Quest: "act5_intro", Name: "Spoke to Larzuk", Tag: "U",
		Steps: []Step{
			{Kind: KGoNPC, NPC: Larzuk},
			{Kind: KTalk, NPC: Larzuk},
			{Kind: KClose},
		}},
	{Quest: "siege_harrogath", Name: "Siege on Harrogath (sockets)", Tag: "U (socket UI unknown)",
		Steps: []Step{
			{Kind: KGoArea, Area: AreaBloodyFoothills, Note: "waypoint 111 then west; no waypoint here (V)"},
			{Kind: KKill, Target: "Shenk the Overseer", Area: AreaBloodyFoothills, Tag: "V SU 42 class 479", Note: "boss.Shenk"},
			{Kind: KToTown},
			{Kind: KGoNPC, NPC: Larzuk},
			{Kind: KTalk, NPC: Larzuk, Note: "U: reward adds sockets to one chosen item"},
			{Kind: KClose},
		}},
	{Quest: "rescue_arreat", Name: "Rescue on Mount Arreat", Skip: true, Tag: "U (can fail, V text)",
		Steps: []Step{
			{Kind: KGoArea, Area: AreaFrigidHighlands, Note: "waypoint 31"},
			{Kind: KUseObject, Obj: ObjCage, Area: AreaFrigidHighlands, Tag: "V object; Prison Door 434 monsters next to each cage"},
			{Kind: KToTown},
			{Kind: KGoNPC, NPC: QualKehk},
			{Kind: KTalk, NPC: QualKehk, Note: "U: reward barbarian mercenaries, runes"},
			{Kind: KClose},
		}},
	{Quest: "prison_of_ice", Name: "Prison of Ice (Scroll of Resistance)", Req: true, Tag: "U (flow S Kolbot Questing.anya)",
		Steps: []Step{
			{Kind: KGoArea, Area: AreaFrozenRiver, Note: "Crystalline Passage 113 -> Frozen River 114 (Vis, V)"},
			{Kind: KUseObject, Obj: ObjFrozenAnya, Area: AreaFrozenRiver, Tag: "V object, behaviour S"},
			{Kind: KToTown},
			{Kind: KGoNPC, NPC: Malah},
			{Kind: KTalk, NPC: Malah, Note: "U: Malah gives the potion (ice)"},
			{Kind: KClose},
			{Kind: KGoArea, Area: AreaFrozenRiver},
			{Kind: KUseObject, Obj: ObjFrozenAnya, Area: AreaFrozenRiver, Note: "U: thaws her with the potion in the pack"},
			{Kind: KToTown},
			{Kind: KGoNPC, NPC: Malah},
			{Kind: KTalk, NPC: Malah, Note: "U: Malah gives the Scroll of Resistance (tr2)"},
			{Kind: KClose},
			{Kind: KUseItem, Item: ItemResistScroll, Tag: "U right click"},
			{Kind: KGoNPC, NPC: AnyaNPC},
			{Kind: KTalk, NPC: AnyaNPC, Note: "U: Anya opens the red portal to Nihlathak and personalizes later"},
			{Kind: KClose},
		}},
	{Quest: "betrayal_harrogath", Name: "Betrayal of Harrogath (Nihlathak)", Skip: true, Tag: "U",
		Steps: []Step{
			{Kind: KGoArea, Area: AreaHallsOfPain, Note: "waypoint 35 (needs the temple levels once)"},
			{Kind: KGoArea, Area: AreaHallsOfVaught},
			{Kind: KKill, Target: "Nihlathak", Area: AreaHallsOfVaught, Tag: "V SU 60 class 526", Note: "boss.Nihlathak"},
			{Kind: KToTown},
		}},
	{Quest: "rite_of_passage", Name: "Rite of Passage (the Ancients)", Req: true, Tag: "U",
		Steps: []Step{
			{Kind: KGoArea, Area: AreaAncientsWay, Note: "waypoint 37"},
			{Kind: KGoArea, Area: AreaArreatSummit},
			{Kind: KUseObject, Obj: ObjAncientsAltar, Area: AreaArreatSummit, Tag: "V object; altar spot (10047,12622) S"},
			{Kind: KKill, Target: "The Ancients", Area: AreaArreatSummit, Tag: "V classes 540/541/542, clvl 20/40/60 for credit", Note: "boss.Ancients; no town portal, never leave the area"},
		}},
	{Quest: "eve_of_destruction", Name: "Eve of Destruction (Baal)", Req: true, Tag: "U",
		Steps: []Step{
			{Kind: KGoArea, Area: AreaWorldstoneKeep2, Note: "waypoint 38 once the summit stairs are open"},
			{Kind: KGoArea, Area: AreaThrone, Note: "130 -> 131"},
			{Kind: KWaves, Area: AreaThrone, Tag: "S wave list (boss/waves.go)"},
			{Kind: KUseObject, Obj: ObjBaalPortal, Area: AreaThrone, Tag: "V object"},
			{Kind: KWaitArea, Area: AreaWorldstoneChamber},
			{Kind: KKill, Target: "Baal", Area: AreaWorldstoneChamber, Tag: "V class 544", Note: "boss.Baal"},
		}},
}

// Act5Handler returns the handler for a quest id.
func Act5Handler(quest string) (Handler, bool) {
	for _, h := range Act5 {
		if h.Quest == quest {
			return h, true
		}
	}
	return Handler{}, false
}

// Act5Order: Larzuk and Shenk (sockets), optional Rescue, Prison of Ice, optional Nihlathak, the Ancients, Baal.
var Act5Order = []string{
	"act5_intro", "siege_harrogath", "rescue_arreat", "prison_of_ice", "betrayal_harrogath", "rite_of_passage", "eve_of_destruction",
}

// Act5Merc is Qual-Kehk's barbarian hire (price factor 9000, V hireling.txt; row U). Act 3's Iron Wolf is cheaper.
var Act5Merc = Handler{Quest: "act5_merc", Name: "Hire a barbarian", Tag: "U",
	Steps: []Step{{Kind: KHire, Labels: []string{"Bash"}, Tag: "U"}}}
