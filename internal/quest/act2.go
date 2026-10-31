package quest

// Normal Act 2 (Lut Gholein) as data. Tags: V verified against the real game tables on 2026-10-28 (levels.txt,
// monstats.txt, objects.txt, items.txt read from the install through internal/data; act2_real_test.go re-checks
// them where /tmp/d2mpq exists), S secondary source (Kolbot constants, Arreat Summit), U unverified in the running
// client. Nothing here was run against a live game; docs/act2.md lists the live checks.

// Act 2 area ids. V (levels.txt rows 40..74). Note: docs/goals/execution.md and catalog.md list Halls of the Dead 3
// as 58 and Claw Viper 2 as 61 / Maggot Lair 3 as 64 from memory; the table says 60 and 61 and 64, and 58 is Claw Viper
// Temple 1. These constants are the table's.
const (
	AreaRockyWaste    = 41
	AreaDryHills      = 42
	AreaFarOasis      = 43
	AreaLostCity      = 44
	AreaValleySnakes  = 45
	AreaCanyon        = 46 // Canyon of the Magi (waypoint; seven tomb entrances)
	AreaSewers1       = 47
	AreaSewers2       = 48 // waypoint
	AreaSewers3       = 49 // Radament
	AreaHarem1        = 50
	AreaHarem2        = 51
	AreaPalaceCellar1 = 52 // waypoint
	AreaPalaceCellar2 = 53
	AreaPalaceCellar3 = 54 // dead end: the Arcane Sanctuary portal (S)
	AreaStonyTomb1    = 55
	AreaHallsDead1    = 56
	AreaHallsDead2    = 57 // waypoint
	AreaClawViper1    = 58
	AreaStonyTomb2    = 59
	AreaHallsDead3    = 60 // Horadric Cube chest
	AreaClawViper2    = 61 // Amulet of the Viper altar, Fangskin
	AreaMaggotLair1   = 62
	AreaMaggotLair2   = 63
	AreaMaggotLair3   = 64 // Staff of Kings chest
	AreaAncientTunnel = 65
	AreaTalTomb1      = 66 // Tal Rasha's Tomb, seven areas 66..72, one holds the orifice (per game, U)
	AreaTalTomb7      = 72
	AreaDurielsLair   = 73
	AreaArcane        = 74 // Arcane Sanctuary (waypoint)
)

// Act 2 town and quest NPC classes. V: monstats hcIdx (Warriv 175, Atma 176, Drognan 177, Fara 178, Greiz 198, Elzix 199,
// Geglash 200, Jerhyn 201, Lysander 202, Meshif 210 for the A2 harbour and 264 for the A3 one, Tyrael 251). Greiz also V
// live (town/merc.go). Which Cain class stands in Lut Gholein after the rescue is U (cain2 244 .. cain6 520 all exist).
const (
	Atma              = 176
	Drognan           = 177
	Fara              = 178
	Greiz             = 198
	Elzix             = 199
	Geglash           = 200
	Lysander          = 202
	TyraelTomb        = 251 // tyrael1: appears in Tal Rasha's Chamber after Duriel (S)
	MeshifA3          = 264
	CainA2            = 244 // V live (i1): a class 244 NPC stands in Lut Gholein at (5134,5085) beside Fara; role Cain by position/class, menu not opened
	ObjOrifice        = 152 // V objects.txt "orifice": the Horadric Staff holder in the true tomb
	ObjTaintedSun     = 149 // V "taintedsunaltar": the Amulet of the Viper altar in Claw Viper Temple 2
	ObjDurielDoor     = 100 // V "Duriel's Lair": the object that leads to area 73 once the staff is placed
	ObjTomeJournal    = 357 // V "Tome"; S (Kolbot) says it is the Summoner's journal: touching it opens the red portal
	ObjCubeChest      = 354 // V live: opened at (15363,12623) in Halls of the Dead 3 (60) gives the Cube `box` (class 549); not in Presets until the room streams in
	ObjScrollChest    = 355 // V live: at (7832,8027) in Sewers 3 (49), 5 subtiles from Radament; opening drops the Horadric Scroll `tr1` (class 550)
	ObjStaffChest     = 356 // S: Staff of Kings chest, Maggot Lair 3
	ObjArcanePortal   = 298 // U: the Palace Cellar 3 portal to the Arcane Sanctuary (V: object "portal" act 2 exists)
	ObjWaypointArcane = 323 // U: "Waypoint qm" act 2, probably the Arcane Sanctuary one
)

// Quest item codes. V items.txt: Cube box (2x2, quest), Staff of Kings msf (display "Shaft of the Horadric Staff", 1x3),
// Amulet of the Viper vip (display "Top of the Horadric Staff", 1x1), Horadric Staff hst (1x4), Book of Skill ass (2x2,
// useable), Horadric Scroll tr1 (2x2).
const (
	ItemCube           = "box"
	ItemShaft          = "msf"
	ItemViperAmulet    = "vip"
	ItemHoradricStaff  = "hst"
	ItemBookOfSkill    = "ass"
	ItemHoradricScroll = "tr1"
)

// Extended step kinds for Act 2. They continue the Kind space of steps.go far away from its iota, so a future
// merge of both lists cannot collide. Parameters ride in the existing Step fields:
//
//	KUseItem   Item = code to right-click (Book of Skill)
//	KTransmute Labels = input item codes, Item = expected output code
//	KPlaceItem Item = code to put into the object, Obj = object class, Area where it stands
//	KOpenChest Obj = chest class, Area; the loot drops to the ground, a following KPickup takes it
//	KFindTomb  Area = Canyon; walks the seven tomb entrances until the orifice object is present
//	KHire      Labels[0] = aura name, Obj = max gold (0 = what gold allows)
const (
	KUseItem Kind = 100 + iota
	KTransmute
	KPlaceItem
	KOpenChest
	KFindTomb
	KHire
)

// Act2 lists the Normal Act 2 handlers in play order. Req marks the critical path to Act 3 (Staff chain and Duriel);
// the Summoner chain is optional for progression but is where the real tomb is revealed in the original game (S),
// Skip says NM/Hell may leave a quest out (policy). Pre uses saves quest ids.
//
// Every dialog row, the order of rewards and the claim-by-talking mechanics are U; the fetch legs end with the
// quest item in the pack, which RunAct2 proves (T3) before moving on.
var Act2 = []Handler{
	{Quest: "act2_intro", Name: "Spoke to Jerhyn", Req: true, Tag: "U (the word is the intro talk; position of Jerhyn in the palace is found by scan)",
		Steps: []Step{
			{Kind: KGoNPC, NPC: Jerhyn},
			{Kind: KTalk, NPC: Jerhyn},
			{Kind: KClose},
		}},
	{Quest: "radament", Name: "Radament's Lair", Skip: false, Tag: "U",
		Steps: []Step{
			{Kind: KGoNPC, NPC: Atma},
			{Kind: KTalk, NPC: Atma, Note: "U: first talk gives the quest, no topic row (Atma's Talk submenu is not recorded)"},
			{Kind: KClose},
			{Kind: KGoArea, Area: AreaSewers3, Note: "Sewers 2 waypoint is only active after walking Sewers 1 -> 2 once; nav falls back to stairs"},
			{Kind: KKill, Target: "Radament", Area: AreaSewers3, Tag: "V class 229, super unique 10"},
			{Kind: KPickup, Item: ItemBookOfSkill, Tag: "V code"},
			{Kind: KUseItem, Item: ItemBookOfSkill, Note: "right-click: +1 skill point (stat 5). The book picks a skill, the leveler spends the point"},
			{Kind: KOpenChest, Obj: ObjScrollChest, Area: AreaSewers3, Tag: "S", Note: "the Horadric Scroll chest next to the boss"},
			{Kind: KPickup, Item: ItemHoradricScroll, Tag: "V code"},
			{Kind: KToTown},
			{Kind: KGoNPC, NPC: Atma},
			{Kind: KTalk, NPC: Atma, Note: "U: completes the quest (reward is the book already read)"},
			{Kind: KClose},
		}},
	{Quest: "horadric_cube", Name: "Horadric Cube (Halls of the Dead 3)", Req: true, Tag: "U",
		Steps: []Step{
			{Kind: KGoArea, Area: AreaHallsDead3, Note: "Halls of the Dead 2 waypoint (57), then down; Dry Hills (42) is the way in"},
			{Kind: KOpenChest, Obj: ObjCubeChest, Area: AreaHallsDead3, Tag: "S"},
			{Kind: KPickup, Item: ItemCube, Tag: "V code"},
			{Kind: KToTown},
		}},
	{Quest: "staff_of_kings", Name: "Staff of Kings (Maggot Lair 3)", Req: true, Tag: "U",
		Steps: []Step{
			{Kind: KGoArea, Area: AreaMaggotLair3, Note: "Far Oasis (43) -> Maggot Lair 1..3; Coldworm the Burrower lives on level 3"},
			{Kind: KOpenChest, Obj: ObjStaffChest, Area: AreaMaggotLair3, Tag: "S"},
			{Kind: KPickup, Item: ItemShaft, Tag: "V code"},
			{Kind: KToTown},
		}},
	{Quest: "tainted_sun", Name: "The Tainted Sun (Amulet of the Viper)", Req: true, Tag: "U",
		Steps: []Step{
			{Kind: KGoArea, Area: AreaClawViper2, Note: "Lost City (44) -> Valley of Snakes (45) -> Claw Viper Temple 1 (58) -> 2 (61)"},
			{Kind: KKill, Target: "Fangskin", Area: AreaClawViper2, Tag: "V super unique 12, clawviper3", Note: "guards the altar"},
			{Kind: KUseObject, Obj: ObjTaintedSun, Area: AreaClawViper2, Tag: "V object class"},
			{Kind: KPickup, Item: ItemViperAmulet, Tag: "V code"},
			{Kind: KToTown},
		}},
	{Quest: "horadric_staff", Name: "The Horadric Staff (assemble and place)", Req: true,
		Pre: []Cond{}, Tag: "U",
		Steps: []Step{
			{Kind: KTransmute, Labels: []string{ItemShaft, ItemViperAmulet}, Item: ItemHoradricStaff, Tag: "V recipe (cubemain: msf + vip -> hst)", Note: "Cube in the pack; open it from the inventory, anywhere in town"},
			{Kind: KGoArea, Area: AreaCanyon},
			{Kind: KFindTomb, Area: AreaCanyon, Tag: "U", Note: "enter tombs 66..72 until the orifice object is present"},
			{Kind: KPlaceItem, Item: ItemHoradricStaff, Obj: ObjOrifice, Tag: "U (cursor then click, or click with the staff in the pack)"},
			{Kind: KUseObject, Obj: ObjDurielDoor, Area: AreaDurielsLair, Tag: "V object class", Note: "the object that appears once the staff is in; leads to area 73"},
		}},
	{Quest: "arcane_sanctuary", Name: "The Arcane Sanctuary (Drognan, Jerhyn, Palace Cellar 3)", Skip: true, Tag: "U",
		Steps: []Step{
			{Kind: KGoNPC, NPC: Drognan},
			{Kind: KTalk, NPC: Drognan},
			{Kind: KClose},
			{Kind: KGoNPC, NPC: Jerhyn},
			{Kind: KTalk, NPC: Jerhyn},
			{Kind: KClose},
			{Kind: KGoArea, Area: AreaPalaceCellar3, Note: "Palace Cellar 1 waypoint (52) is behind the Harem; the Harem door opens after Jerhyn (U)"},
			{Kind: KUseObject, Obj: ObjArcanePortal, Area: AreaPalaceCellar3, Tag: "U"},
			{Kind: KWaitArea, Area: AreaArcane, Tag: "V area id"},
			{Kind: KTouchWP, Note: "Arcane waypoint: object class U (323?)"},
		}},
	{Quest: "summoner", Name: "The Summoner", Skip: true, Tag: "U",
		Steps: []Step{
			{Kind: KGoArea, Area: AreaArcane},
			{Kind: KKill, Target: "The Summoner", Area: AreaArcane, Tag: "V class 250", Note: "needs Teleport or a long walk through the arms (docs/progression/bosses.md)"},
			{Kind: KUseObject, Obj: ObjTomeJournal, Area: AreaArcane, Tag: "S", Note: "touching the journal opens the red portal to the Canyon (U)"},
			{Kind: KUseObject, Obj: ObjRedPortal, Area: AreaArcane, Tag: "U"},
			{Kind: KWaitArea, Area: AreaCanyon, Tag: "V area id"},
			{Kind: KToTown},
		}},
	{Quest: "seven_tombs", Name: "The Seven Tombs (Duriel)", Req: true, Tag: "U",
		Steps: []Step{
			{Kind: KGoArea, Area: AreaDurielsLair, Note: "boss.RunWithEntry enters through the tomb when the player is not already in the lair"},
			{Kind: KKill, Target: "Duriel", Area: AreaDurielsLair, Tag: "V class 211"},
			{Kind: KGoNPC, NPC: TyraelTomb},
			{Kind: KTalk, NPC: TyraelTomb, Note: "S: Tyrael opens the portal to town and ends the quest"},
			{Kind: KClose},
			{Kind: KToTown},
			{Kind: KGoNPC, NPC: Jerhyn},
			{Kind: KTalk, NPC: Jerhyn, Note: "S: Jerhyn's talk records act 2 complete (A2->A3 hop leg)"},
			{Kind: KClose},
		}},
}

// Act2Proxy says, per handler id, that the work is plainly done when the pack holds the item: a fetch leg cannot be
// repeated (the chest is empty) and an assembled Staff cannot be made twice. T3 evidence, so Result.Confirmed stays
// false until the save or memory reader catches up.
var Act2Proxy = map[string]func(Has) bool{
	"horadric_cube":  func(h Has) bool { return h.Has(ItemCube) },
	"staff_of_kings": func(h Has) bool { return h.Has(ItemShaft) || h.Has(ItemHoradricStaff) },
	"tainted_sun":    func(h Has) bool { return h.Has(ItemViperAmulet) || h.Has(ItemHoradricStaff) },
}

// Act2Handler returns the handler for a quest id.
func Act2Handler(quest string) (Handler, bool) {
	for _, h := range Act2 {
		if h.Quest == quest {
			return h, true
		}
	}
	return Handler{}, false
}

// Act2Order is the order the campaign runs the handlers in. Radament first (the scroll starts the Staff chain), then
// the three fetches (cheap to expensive), the assembly, the optional Summoner pair, and Duriel. The Summoner pair
// sits before Duriel because the original game tells you the real tomb there; the bot finds the tomb by scanning,
// so the pair may be skipped (Skip) without blocking Duriel.
var Act2Order = []string{
	"act2_intro", "radament", "horadric_cube", "tainted_sun", "staff_of_kings", "horadric_staff",
	"arcane_sanctuary", "summoner", "seven_tombs",
}

// Act2Merc is the Normal Act 2 Desert Mercenary hire as a handler. The aura is advisory: hireling.txt (V) lists
// Prayer (Combat), Defiance (Defensive, Id 7) and Blessed Aim (Offensive) for Normal, and the hire window shows
// only the type text, so the town side picks the row (town.MercWant.Row or Pick). Defiance is the default because
// it helps every build (docs/progression/merc.md). Not a quest id: Reader.Skippable is false, so the caller
// decides when to run it (no living merc, gold >= the quoted cost).
var Act2Merc = Handler{Quest: "act2_merc", Name: "Hire the Desert Mercenary", Tag: "U",
	Steps: []Step{
		{Kind: KHire, Labels: []string{"Defiance"}, Tag: "V dialog (Talk / Hire / Cancel on Greiz), row choice U",
			Note: "town.Hire clicks Greiz from where the player stands (no nav walk: the stairs killed a character twice)"},
	}}
