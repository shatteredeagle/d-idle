package d2map

import "sort"

// Preset unit kinds (PresetUnit.dwType). VERIFIED LIVE for 1, 2 and 5 in the Rogue Encampment.
const (
	PresetNPC    = 1
	PresetObject = 2
	PresetTile   = 5
)

// Preset is a statically placed unit (NPC, object or level-transition tile) with its absolute
// position in subtiles. It exists in the map data before the live unit does.
type Preset struct {
	Kind  int
	ID    int // monster class (NPC), object id (object) or tile number (tile)
	Pos   Point
	Room2 uint32
}

// RoomTile links a tile number to the Room2 on the other side of a level transition.
type RoomTile struct {
	Dest uint32 // Room2 address across the transition
	Num  uint32
}

// Exit is a level transition: standing on Pos leads to level Dest.
type Exit struct {
	Dest  int
	Pos   Point
	Room2 uint32
}

// Adjacent is a neighbouring level seen through a Room2's near-room links.
type Adjacent struct {
	LevelNo int
	Rect    Rect // the neighbouring level's rectangle
	Contact Rect // part of this level within one subtile of the neighbour
}

// Presets reads a Room2's preset list (cycle-safe, positions made absolute).
func (r Reader) Presets(rm Room2) []Preset {
	var out []Preset
	seen := map[uint32]bool{}
	for p := rm.Preset; validPtr(p) && !seen[p] && len(out) < maxList; {
		seen[p] = true
		b, err := r.block(p, presetSize)
		if err != nil {
			break
		}
		out = append(out, Preset{
			Kind:  ti(b, offPreType),
			ID:    ti(b, offPreTxt),
			Pos:   Point{rm.Rect.X + ti(b, offPrePosX), rm.Rect.Y + ti(b, offPrePosY)},
			Room2: rm.Addr,
		})
		p = le32(b, offPreNext)
	}
	return out
}

// RoomTiles reads a Room2's transition tile list (cycle-safe).
func (r Reader) RoomTiles(rm Room2) []RoomTile {
	var out []RoomTile
	seen := map[uint32]bool{}
	for p := rm.Tiles; validPtr(p) && !seen[p] && len(out) < maxList; {
		seen[p] = true
		b, err := r.block(p, tileSize)
		if err != nil {
			break
		}
		t := RoomTile{Dest: le32(b, offTileRoom2)}
		if np := le32(b, offTileNum); validPtr(np) {
			if nb, err := r.block(np, 4); err == nil {
				t.Num = le32(nb, 0)
			}
		}
		out = append(out, t)
		p = le32(b, offTileNext)
	}
	return out
}

// Waypoint object ids (objects.txt; Kolbot Pather list). UNVERIFIED except where docs/map.md says so.
var waypointObjects = map[int]bool{119: true, 145: true, 156: true, 157: true, 237: true, 238: true, 288: true, 323: true, 324: true, 398: true, 402: true, 429: true, 494: true, 496: true, 511: true, 539: true}

// IsWaypointObject reports whether an object class id is a waypoint.
func IsWaypointObject(id int) bool { return waypointObjects[id] }

// sortPresets orders presets deterministically (kind, id, y, x).
func sortPresets(ps []Preset) {
	sort.Slice(ps, func(i, j int) bool {
		a, b := ps[i], ps[j]
		if a.Kind != b.Kind {
			return a.Kind < b.Kind
		}
		if a.ID != b.ID {
			return a.ID < b.ID
		}
		if a.Pos.Y != b.Pos.Y {
			return a.Pos.Y < b.Pos.Y
		}
		return a.Pos.X < b.Pos.X
	})
}

// mergePresets adds the presets of one Room2 read to the known list, dropping duplicates (same kind, id and
// position: the same unit seen through another Room2 address after the level was entered again).
func mergePresets(have, add []Preset) []Preset {
	type key struct {
		kind, id int
		pos      Point
	}
	seen := make(map[key]int, len(have)+len(add))
	for i, p := range have {
		seen[key{p.Kind, p.ID, p.Pos}] = i
	}
	for _, p := range add {
		k := key{p.Kind, p.ID, p.Pos}
		if i, ok := seen[k]; ok {
			have[i].Room2 = p.Room2
			continue
		}
		seen[k] = len(have)
		have = append(have, p)
	}
	return have
}
