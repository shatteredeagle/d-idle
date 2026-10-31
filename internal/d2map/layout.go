package d2map

// Struct offsets, from docs/d2-1.14d-memory.md section 3.2 (D2BS D2Structs.h). Each is tagged
// VERIFIED LIVE (read from the running 1.14d client, see docs/map.md) or UNVERIFIED.
const (
	rvaPlayerUnit = 0x3A6A70 // VERIFIED LIVE (cmd/spike, MEM)

	offUnitAct  = 0x1C // VERIFIED LIVE: UnitAny.pAct
	offUnitPath = 0x2C // VERIFIED LIVE: UnitAny.pPath

	offPathX     = 0x02 // VERIFIED LIVE (word)
	offPathY     = 0x06 // VERIFIED LIVE (word)
	offPathRoom1 = 0x1C // VERIFIED LIVE

	actSize       = 0x4C
	offActSeed    = 0x0C // VERIFIED LIVE (dword, nonzero, stable)
	offActRoom1   = 0x10 // VERIFIED LIVE (head of the active Room1 list)
	offActIndex   = 0x14 // VERIFIED LIVE (0 in act 1)
	offActMisc    = 0x48 // VERIFIED LIVE
	miscSize      = 0x480
	offMiscLevel1 = 0x47C // VERIFIED LIVE (pLevelFirst)

	levelSize        = 0x228
	offLvlRoom2First = 0x10  // VERIFIED LIVE
	offLvlPosX       = 0x1C  // VERIFIED LIVE (tiles)
	offLvlPosY       = 0x20  // VERIFIED LIVE (tiles)
	offLvlSizeX      = 0x24  // VERIFIED LIVE (tiles)
	offLvlSizeY      = 0x28  // VERIFIED LIVE (tiles)
	offLvlNext       = 0x1AC // VERIFIED LIVE
	offLvlMisc       = 0x1B4 // VERIFIED LIVE (== Act.pMisc)
	offLvlNo         = 0x1D0 // VERIFIED LIVE
	offLvlWarpX      = 0x1E0 // UNVERIFIED semantics (9 dwords)
	offLvlWarpY      = 0x204 // UNVERIFIED semantics (9 dwords)

	room2Size      = 0x70
	offR2Near      = 0x08 // VERIFIED LIVE (pointer to array of Room2*)
	offR2Next      = 0x24 // VERIFIED LIVE
	offR2Flags     = 0x28 // UNVERIFIED meaning
	offR2NearCount = 0x2C // VERIFIED LIVE
	offR2Room1     = 0x30 // VERIFIED LIVE
	offR2PosX      = 0x34 // VERIFIED LIVE (tiles)
	offR2PosY      = 0x38 // VERIFIED LIVE (tiles)
	offR2SizeX     = 0x3C // VERIFIED LIVE (tiles)
	offR2SizeY     = 0x40 // VERIFIED LIVE (tiles)
	offR2Tiles     = 0x4C // see docs/map.md
	offR2Level     = 0x58 // VERIFIED LIVE
	offR2Preset    = 0x5C // VERIFIED LIVE

	room1Size      = 0x80
	offR1Near      = 0x00 // VERIFIED LIVE (pointer to array of Room1*)
	offR1Room2     = 0x10 // VERIFIED LIVE
	offR1Coll      = 0x20 // VERIFIED LIVE
	offR1NearCount = 0x24 // VERIFIED LIVE
	offR1XStart    = 0x4C // VERIFIED LIVE (subtiles)
	offR1YStart    = 0x50 // VERIFIED LIVE (subtiles)
	offR1XSize     = 0x54 // VERIFIED LIVE (subtiles)
	offR1YSize     = 0x58 // VERIFIED LIVE (subtiles)
	offR1UnitFirst = 0x74 // plausible, see docs/map.md
	offR1Next      = 0x7C // VERIFIED LIVE

	collSize     = 0x28
	offCollX     = 0x00 // VERIFIED LIVE (subtiles)
	offCollY     = 0x04
	offCollW     = 0x08
	offCollH     = 0x0C
	offCollRoomX = 0x10 // tile units
	offCollRoomY = 0x14
	offCollRoomW = 0x18
	offCollRoomH = 0x1C
	offCollStart = 0x20 // VERIFIED LIVE (WORD*)
	offCollEnd   = 0x24 // doc said +0x22; natural layout and live reads say +0x24

	presetSize = 0x1C
	offPreTxt  = 0x04
	offPrePosX = 0x08
	offPreNext = 0x0C
	offPreType = 0x14
	offPrePosY = 0x18

	tileSize     = 0x14
	offTileRoom2 = 0x00
	offTileNext  = 0x04
	offTileNum   = 0x10
)

// Sanity limits so torn or garbage reads cannot make us allocate or loop without end.
const (
	maxList     = 16384 // entries walked in any linked list
	maxNear     = 64    // entries in a near-room array
	maxCollDim  = 256   // subtiles per side of one room's collision map
	maxLevelDim = 4096  // tiles
	// maxLevelTiles bounds a Level struct's own size: real levels are under 400 tiles a side (S, the 2000 subtile
	// figure of docs/map.md); a torn read must not yield a rectangle of hundreds of millions of cells (OOM, G1 attempt 16).
	maxLevelTiles = 600
)

func validPtr(p uint32) bool { return p >= 0x10000 && p < 0x7fff0000 }
