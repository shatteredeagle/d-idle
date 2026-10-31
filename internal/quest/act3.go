package quest

// Normal Act 3 (Kurast) as data. Tags follow docs/act3.md: V = read from the real 1.14d tables on 2026-10-29 by
// ACT3PLAN (levels, monstats, objects, items, cubemain; act3_real_test.go re-checks them where /tmp/d2mpq exists),
// S = Kolbot constants / Arreat Summit, U = unverified in the running client. No live game was used.

// Act 3 area ids (V levels.txt).
const (
	AreaSpiderForest  = 76
	AreaGreatMarsh    = 77
	AreaFlayerJungle  = 78
	AreaLowerKurast   = 79
	AreaKurastBazaar  = 80
	AreaUpperKurast   = 81
	AreaCauseway      = 82
	AreaTravincal     = 83
	AreaSpiderCavern  = 85 // Khalim's Eye chest (V text, chest class S)
	AreaFlayerDung3   = 91 // Khalim's Brain chest
	AreaSewers3Kurast = 93 // Kurast Sewers 2 (data name); Khalim's Heart chest
	AreaRuinedTemple  = 94 // Lam Esen's Tome altar
	AreaDurance1      = 100
	AreaDurance2      = 101 // waypoint
	AreaDurance3      = 102 // Mephisto
)

// Act 3 NPC classes (V monstats hcIdx): Hratli 253 (const in dialog.go), Alkor 254, Ormus 255, Asheara 252.
// CainA3 is cain3; which class actually stands in Kurast is U (town.Acts lists all).
const (
	Asheara = 252
	Alkor   = 254
	Ormus   = 255
	CainA3  = 245
)

// Act 3 objects. Names V (objects.txt), roles S (Kolbot sdk.js) unless marked.
const (
	ObjLamTomeAltar  = 193 // V "LamTome"
	ObjGidbinnAltar  = 251 // V "gidbinn altar"
	ObjGidbinn       = 252 // V "gidbinn"
	ObjCompellingOrb = 404 // V "compellingorb"
	ObjRelicChestA   = 405 // V "chest"; Kolbot: Heart
	ObjRelicChestB   = 406 // V "chest"; Kolbot: Brain
	ObjRelicChestC   = 407 // V "chest"; Kolbot: Eye
	ObjSewerStairs   = 366 // S: Sewers entrance trick
	ObjSewerLever    = 367 // S
	ObjDuranceStairs = 386 // V exists as "Dummy"; S Kolbot DuranceEntryStairs
	ObjRedPortalA3   = 342 // S Kolbot RedPortalToAct4; the Act 3 to 4 portal (ObjRedPortal = 60 is the Arcane and Anya portal)
)

// Quest item codes (V items.txt, all Quest = true; sizes asserted in the real-table test).
const (
	ItemJadeFigurine  = "j34" // 1x2
	ItemGoldenBird    = "g34" // 1x2
	ItemGidbinnDecoy  = "d33"
	ItemGidbinn       = "g33"
	ItemLamTome       = "bbb" // 2x2
	ItemKhalimEye     = "qey" // 1x1
	ItemKhalimHeart   = "qhr" // 1x1
	ItemKhalimBrain   = "qbr" // 1x1
	ItemKhalimFlail   = "qf1" // 2x3
	ItemKhalimWill    = "qf2" // 2x3
	ItemSoulstone     = "mss" // Mephisto's Soulstone, 1x1 (Hellforge quest, Act 4)
	ItemBlackTowerKey = "luv" // role U
)

// Extended step kinds for Act 3, continuing past Act 2's (100..105).
//
//	KSmashOrb  Obj = orb class, Item = weapon code to equip from the pack (qf2); the adapter swaps the weapon, clicks
//	           the orb until its mode changes and re-equips the old weapon (X5 in docs/act3.md)
const KSmashOrb Kind = 110

// Act3 lists the Normal Act 3 handlers in play order. Req marks the critical path to Act 4 (Khalim's Will chain,
// Council, orb, Mephisto). Every dialog row and claim mechanic is U; fetch legs end with the item in the pack.
var Act3 = []Handler{
	{Quest: "act3_intro", Name: "Spoke to Hratli", Req: true, Tag: "U",
		Steps: []Step{
			{Kind: KGoNPC, NPC: Hratli},
			{Kind: KTalk, NPC: Hratli},
			{Kind: KClose},
		}},
	{Quest: "golden_bird", Name: "The Golden Bird (+20 life)", Skip: true, Tag: "U (dropper of j34 is U; candidates Sszark SU 21, Endugu SU 22)",
		Steps: []Step{
			{Kind: KPickup, Item: ItemJadeFigurine, Tag: "V code"},
			{Kind: KToTown},
			{Kind: KGoNPC, NPC: MeshifA3},
			{Kind: KTalk, NPC: MeshifA3, Note: "U: Meshif returns g34 for j34"},
			{Kind: KClose},
			{Kind: KGoNPC, NPC: Alkor},
			{Kind: KTalk, NPC: Alkor, Note: "U: hand over g34, Alkor brews the +20 life potion; drink it"},
			{Kind: KClose},
		}},
	{Quest: "blade_old_religion", Name: "Blade of the Old Religion", Skip: true, Tag: "U (Kolbot skips it in rushes)",
		Steps: []Step{
			{Kind: KGoArea, Area: AreaFlayerJungle, Note: "Pygmy village; altar 251/252"},
			{Kind: KOpenChest, Obj: ObjGidbinnAltar, Area: AreaFlayerJungle, Tag: "S"},
			{Kind: KPickup, Item: ItemGidbinn, Tag: "V code"},
			{Kind: KToTown},
			{Kind: KGoNPC, NPC: Hratli},
			{Kind: KTalk, NPC: Hratli, Note: "U: reward is a rare ring"},
			{Kind: KClose},
		}},
	{Quest: "lam_esen", Name: "Lam Esen's Tome (+5 stat points)", Skip: true, Tag: "U",
		Steps: []Step{
			{Kind: KGoArea, Area: AreaRuinedTemple, Note: "Kurast Bazaar (80) -> Ruined Temple; whether the altar needs the quest started is U"},
			{Kind: KOpenChest, Obj: ObjLamTomeAltar, Area: AreaRuinedTemple, Tag: "V name, S role"},
			{Kind: KPickup, Item: ItemLamTome, Tag: "V code"},
			{Kind: KToTown},
			{Kind: KGoNPC, NPC: Alkor},
			{Kind: KTalk, NPC: Alkor, Note: "U: reward 5 stat points, spent by the leveler"},
			{Kind: KClose},
		}},
	{Quest: "khalims_eye", Name: "Khalim's Eye (Spider Cavern)", Req: true, Tag: "U",
		Steps: []Step{
			{Kind: KGoArea, Area: AreaSpiderCavern, Note: "Spider Forest (76) waypoint, then Spider Cave (84) -> Cavern (85)"},
			{Kind: KOpenChest, Obj: ObjRelicChestC, Area: AreaSpiderCavern, Tag: "S (which chest class holds which relic is U)"},
			{Kind: KPickup, Item: ItemKhalimEye, Tag: "V code"},
			{Kind: KToTown},
		}},
	{Quest: "khalims_brain", Name: "Khalim's Brain (Flayer Dungeon 3)", Req: true, Tag: "U",
		Steps: []Step{
			{Kind: KGoArea, Area: AreaFlayerDung3, Note: "Flayer Jungle (78) waypoint, then Dungeon 1 (88) -> 2 (89) -> 3 (91)"},
			{Kind: KOpenChest, Obj: ObjRelicChestB, Area: AreaFlayerDung3, Tag: "S"},
			{Kind: KPickup, Item: ItemKhalimBrain, Tag: "V code"},
			{Kind: KToTown},
		}},
	{Quest: "khalims_heart", Name: "Khalim's Heart (Kurast Sewers 2)", Req: true, Tag: "U",
		Steps: []Step{
			{Kind: KGoArea, Area: AreaSewers3Kurast, Note: "Bazaar (80) or Upper Kurast (81) -> Sewers 1 (92) -> 2 (93); lever/stairs trick 366/367 is S"},
			{Kind: KOpenChest, Obj: ObjRelicChestA, Area: AreaSewers3Kurast, Tag: "S"},
			{Kind: KPickup, Item: ItemKhalimHeart, Tag: "V code"},
			{Kind: KToTown},
		}},
	{Quest: "blackened_temple", Name: "The Blackened Temple (High Council, Khalim's Flail)", Req: true, Tag: "U",
		Steps: []Step{
			{Kind: KGoArea, Area: AreaTravincal, Note: "waypoint 25"},
			{Kind: KKill, Target: "High Council", Area: AreaTravincal, Tag: "V classes 345/346/347, SU 26..31; which three spawn is U", Note: "boss.HighCouncil"},
			{Kind: KPickup, Item: ItemKhalimFlail, Tag: "V code; dropper member U"},
			{Kind: KToTown},
		}},
	{Quest: "khalims_will", Name: "Khalim's Will (cube, equip, smash the orb)", Req: true, Tag: "U",
		Steps: []Step{
			{Kind: KTransmute, Labels: []string{ItemKhalimFlail, ItemKhalimHeart, ItemKhalimEye, ItemKhalimBrain}, Item: ItemKhalimWill, Tag: "V recipe (cubemain Super Khalim Flail)", Note: "4 inputs, never run live"},
			{Kind: KGoArea, Area: AreaTravincal},
			{Kind: KSmashOrb, Obj: ObjCompellingOrb, Item: ItemKhalimWill, Area: AreaTravincal, Tag: "V object, behaviour S/U"},
		}},
	{Quest: "guardian", Name: "The Guardian (Mephisto)", Req: true, Tag: "U",
		Steps: []Step{
			{Kind: KGoArea, Area: AreaDurance2, Note: "stairs 83 -> 100 -> 101 (waypoint) once the orb is smashed"},
			{Kind: KGoArea, Area: AreaDurance3},
			{Kind: KKill, Target: "Mephisto", Area: AreaDurance3, Tag: "V class 242"},
			{Kind: KPickup, Item: ItemSoulstone, Tag: "V code; Hellforge quest"},
		}},
	{Quest: "act3_done", Name: "Red portal to the Fortress", Req: true, Tag: "U",
		Steps: []Step{
			{Kind: KUseObject, Obj: ObjRedPortalA3, Area: AreaDurance3, Tag: "S object 342, only in the game of the kill"},
			{Kind: KWaitArea, Area: AreaFortress, Tag: "V area id"},
		}},
}

// Act3Proxy: the work is plainly done when the pack holds the item.
var Act3Proxy = map[string]func(Has) bool{
	"khalims_eye":   func(h Has) bool { return h.Has(ItemKhalimEye) || h.Has(ItemKhalimWill) },
	"khalims_brain": func(h Has) bool { return h.Has(ItemKhalimBrain) || h.Has(ItemKhalimWill) },
	"khalims_heart": func(h Has) bool { return h.Has(ItemKhalimHeart) || h.Has(ItemKhalimWill) },
}

// Act3Handler returns the handler for a quest id.
func Act3Handler(quest string) (Handler, bool) {
	for _, h := range Act3 {
		if h.Quest == quest {
			return h, true
		}
	}
	return Handler{}, false
}

// Act3Order: cheap optional rewards first, then the relic legs, Council, cube and orb, Mephisto, portal.
var Act3Order = []string{
	"act3_intro", "golden_bird", "blade_old_religion", "lam_esen",
	"khalims_eye", "khalims_brain", "khalims_heart", "blackened_temple", "khalims_will", "guardian", "act3_done",
}

// Act3Merc is Asheara's Iron Wolf hire. Row choice is U (docs/act3.md section 4).
var Act3Merc = Handler{Quest: "act3_merc", Name: "Hire the Iron Wolf", Tag: "U",
	Steps: []Step{{Kind: KHire, Labels: []string{"Cold"}, Tag: "U"}}}
