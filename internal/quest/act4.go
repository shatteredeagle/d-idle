package quest

// Normal Act 4 (Pandemonium Fortress) as data. Tags follow docs/act4.md: V = read from the real 1.14d tables on
// 2026-10-29 by ACT3PLAN (levels, monstats, objects, items; act4_real_test.go re-checks them where /tmp/d2mpq
// exists), S = Kolbot constants / Arreat Summit, U = unverified in the running client. No live game was used.

// Act 4 area ids (V levels.txt). The Fortress (AreaFortress = 103) is in hops.go.
const (
	AreaOuterSteppes    = 104
	AreaPlainsOfDespair = 105 // Izual, no waypoint
	AreaCityOfTheDamned = 106 // waypoint 28
	AreaRiverOfFlame    = 107 // Hellforge, waypoint 29
	AreaChaosSanctuary  = 108 // data name "Chaos Sanctum"
)

// Act 4 NPC classes (V monstats hcIdx): Tyrael is in dialog.go (367), Jamella 405, Halbu 257, Izual 256 (boss),
// izualghost 406 (appears after the kill, S), Hephasto 409 (SU 41). Which Cain class stands in the Fortress is U:
// cain4 246 and cain5 265 both exist.
const (
	Jamella     = 405
	Halbu       = 257
	CainA4      = 246
	CainA4b     = 265
	IzualGhost  = 406
	ClassIzual  = 256
	ClassHephas = 409
	ClassDiablo = 243
)

// Act 4 objects. Names V (objects.txt), roles S (Kolbot sdk.js).
const (
	ObjHellforge     = 376 // V exists; S Hephasto.js
	ObjSealInfector1 = 392 // V all five named "Seal"; roles S Common.js
	ObjSealInfector2 = 393
	ObjSealSeis      = 394
	ObjSealVizier1   = 395
	ObjSealVizier2   = 396
	ObjSealGlow      = 131 // V "Dummy", S SealGlow
	ObjDiabloStar    = 255 // V "Dummy", S DiabloStar
	ObjWaypointA4    = 398 // V (town waypoint)
	ObjHarrogath     = 566 // V name "Harrogath", S RedPortalToAct5
)

// Quest item codes (V items.txt). ItemSoulstone is in act3.go.
const (
	ItemHellforgeHammer = "hfh" // 2x3, V quest item, dropped by Hephasto (S)
)

// Extended step kinds for Acts 4 and 5, continuing past Act 3's KSmashOrb (110).
//
//	KOpenSeals  Area = Chaos Sanctuary; the adapter runs the seal executor (boss.OpenSeals): layout detection from the
//	            preset coordinates, five seals in order, clearing around each node. Success = Diablo spawned (U)
//	KPlaceOnForge  Obj = forge, Item = the item to place; U: whether this is a cursor drop or an equip-and-hit
//	KWaves      Area = Throne of Destruction; the adapter runs the Baal wave loop (boss.Waves)
const (
	KOpenSeals Kind = 111 + iota
	KPlaceOnForge
	KWaves
)

// Seal object ids as data, in the order the executor opens them (Vizier, Seis, Infector; S Kolbot).
var Seals = []int{ObjSealVizier1, ObjSealVizier2, ObjSealSeis, ObjSealInfector1, ObjSealInfector2}

// Act4 lists the Normal Act 4 handlers in play order. Req marks the critical path to Act 5 (Diablo). Every dialog row
// and claim mechanic is U; Izual and the Hellforge are independent of Diablo (docs/act4.md section 2).
var Act4 = []Handler{
	{Quest: "act4_intro", Name: "Spoke to Tyrael", Tag: "U",
		Steps: []Step{
			{Kind: KGoNPC, NPC: Tyrael},
			{Kind: KTalk, NPC: Tyrael},
			{Kind: KClose},
		}},
	{Quest: "izual", Name: "The Fallen Angel (2 skill points)", Tag: "U (Izual preset anchor S)",
		Steps: []Step{
			{Kind: KGoArea, Area: AreaCityOfTheDamned, Note: "waypoint 28, then walk 106 -> 105 (level crossing, not in Vis, V)"},
			{Kind: KGoArea, Area: AreaPlainsOfDespair, Tag: "no waypoint (V)"},
			{Kind: KKill, Target: "Izual", Area: AreaPlainsOfDespair, Tag: "V class 256, all resists 30, cold 75", Note: "boss.Izual"},
			{Kind: KToTown},
			{Kind: KGoNPC, NPC: Tyrael},
			{Kind: KTalk, NPC: Tyrael, Note: "U: reward 2 skill points"},
			{Kind: KClose},
		}},
	{Quest: "hellforge", Name: "The Hellforge (gems and a rune)", Skip: true, Tag: "U (placement procedure unknown)",
		Steps: []Step{
			{Kind: KGoArea, Area: AreaRiverOfFlame, Note: "waypoint 29"},
			{Kind: KKill, Target: "Hephasto the Armorer", Area: AreaRiverOfFlame, Tag: "V SU 41 class 409", Note: "boss.Hephasto; needs the Soulstone in the pack"},
			{Kind: KPickup, Item: ItemHellforgeHammer, Tag: "V code"},
			{Kind: KPlaceOnForge, Obj: ObjHellforge, Item: ItemSoulstone, Area: AreaRiverOfFlame, Tag: "U"},
			{Kind: KToTown},
		}},
	{Quest: "terrors_end", Name: "Terror's End (Diablo)", Req: true, Tag: "U",
		Steps: []Step{
			{Kind: KGoArea, Area: AreaRiverOfFlame, Note: "waypoint 29, then the Vis link 107 -> 108 (V)"},
			{Kind: KGoArea, Area: AreaChaosSanctuary},
			{Kind: KOpenSeals, Area: AreaChaosSanctuary, Tag: "S layouts and node paths (boss/seals.go), U in 1.14d"},
			{Kind: KKill, Target: "Diablo", Area: AreaChaosSanctuary, Tag: "V class 243", Note: "boss.Diablo"},
			{Kind: KToTown},
			{Kind: KGoNPC, NPC: Tyrael},
			{Kind: KTalk, NPC: Tyrael, Note: "U: Tyrael after Diablo"},
			{Kind: KClose},
		}},
	{Quest: "act4_done", Name: "Travel to Harrogath", Req: true, Tag: "U",
		Steps: []Step{
			{Kind: KGoNPC, NPC: Tyrael},
			{Kind: KTalk, NPC: Tyrael},
			{Kind: KPick, NPC: Tyrael, Labels: []string{"Travel to Harrogath"}, Tag: "U"},
			{Kind: KWaitArea, Area: AreaHarrogath, Tag: "V area id"},
		}},
}

// Act4Handler returns the handler for a quest id.
func Act4Handler(quest string) (Handler, bool) {
	for _, h := range Act4 {
		if h.Quest == quest {
			return h, true
		}
	}
	return Handler{}, false
}

// Act4Order: the cheap talk, Izual (independent, 2 skill points), the optional Hellforge, Diablo, the hop.
var Act4Order = []string{"act4_intro", "izual", "hellforge", "terrors_end", "act4_done"}
