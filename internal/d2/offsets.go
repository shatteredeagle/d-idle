// Package d2 reads typed Diablo II LoD 1.14d state from a mem.Source. Every read is defensive:
// pointers are range-checked, lists are length-bounded, and the Snapshot retries torn reads.
//
// Tag legend used in comments: VERIFIED LIVE = checked against the running 1.14d client on the worker
// (method in docs/mem.md); UNVERIFIED = taken from D2BS/docs and not yet confirmed.
package d2

// RVAs from the Game.exe image base 0x400000 (add mem.Base).
const (
	rvaPlayerUnit   = 0x3A6A70 // UnitAny*          VERIFIED LIVE
	rvaFirstControl = 0x3D55BC // Control*          VERIFIED LIVE (non-zero in menus)
	rvaGameInfo     = 0x3A0438 // GameStructInfo*   VERIFIED LIVE (char name at +0xBD)
	rvaExpCharFlag  = 0x3A04F4 // DWORD             VERIFIED LIVE (1 for the LoD char)
	rvaServerUnits  = 0x3A5E70 // UnitHashTable[6]  VERIFIED LIVE
	rvaClientUnits  = 0x3A5270 // UnitHashTable[6]  UNVERIFIED (used for missiles only)
	rvaMouseX       = 0x3A6AB0 // VERIFIED LIVE
	rvaMouseY       = 0x3A6AAC // VERIFIED LIVE
	rvaViewportX    = 0x3A520C // see docs/mem.md for verification state
	rvaViewportY    = 0x3A5208
	rvaScreenSizeX  = 0x31146C
	rvaScreenSizeY  = 0x311470
)

// Unit types (UnitAny.dwType).
const (
	TypePlayer  = 0
	TypeMonster = 1
	TypeObject  = 2
	TypeMissile = 3
	TypeItem    = 4
	TypeTile    = 5
)
