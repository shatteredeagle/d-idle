// Package readiness answers "is this hero ready for that area" as a set of per-threat checks (pass, warn, fail) rolled up
// to a traffic light. Checks are data rows (checks.json), never class or skill literals in Go. Design and open points:
// docs/readiness.md, docs/research/player-power.md. Nothing here changes a gate or a retreat (cards TASK-219, TASK-222).
package readiness

import (
	"encoding/json"
	"errors"
	"fmt"
	"slices"
	"strconv"
	"strings"

	"d2lod-bot/internal/data"
)

// AreaTableSchema is the schema tag of the generated per-area table.
const AreaTableSchema = "readiness.areas/1"

// Difficulty names, in data.Difficulty order.
var diffNames = [3]string{"normal", "nightmare", "hell"}

// Elements are the damage types the checks reason about, in data.DamageType order.
var elementNames = [6]string{"physical", "magic", "fire", "lightning", "cold", "poison"}

// Affix names an elite pack can carry (docs/difficulty.md section 4, V from the Arreat bonus page). The table lists the
// ones that can spawn in the area's difficulty.
var randomAffixes = []string{
	"extra_strong", "extra_fast", "cursed", "magic_resistant", "fire_enchanted", "lightning_enchanted",
	"cold_enchanted", "mana_burn", "teleportation", "spectral_hit", "stone_skin", "aura_enchanted",
}

// AreaDiff is what one area holds at one difficulty.
type AreaDiff struct {
	MLvl int `json:"mlvl"`
	// Immune lists the elements some regular spawn is immune to (resist 100 or more, V: data.Monster.Immune).
	Immune []string `json:"immune,omitempty"`
	// EliteImmune lists the elements an elite modifier could make a spawn immune to: fire, cold and lightning with an
	// elemental enchant (+75 to that resist) or magic resistant (+40), poison with spectral hit (+40). Rules: docs/difficulty.md section 4.
	EliteImmune []string `json:"eliteImmune,omitempty"`
	Affixes     []string `json:"affixes,omitempty"`
	// TopLife and TopHit are the largest regular spawn life and single hit (physical or elemental) and TopToHit its
	// attack rating. U: the monlvl.txt scaling is applied as percent of the monstats value, which is the reading of
	// the columns, not a live measurement.
	TopLife  int `json:"topLife,omitempty"`
	TopHit   int `json:"topHit,omitempty"`
	TopToHit int `json:"topToHit,omitempty"`
	Spawns   int `json:"spawns"`
}

// Area is one levels.txt row with its three difficulties (normal, nightmare, hell).
type Area struct {
	ID   int         `json:"id"`
	Name string      `json:"name"`
	Act  int         `json:"act"` // 1-based
	Diff [3]AreaDiff `json:"diff"`
}

// AreaTable is the generated table. It is derived from the install tables at run time (data.DB); the game data itself
// is never committed (docs/decisions.md), only this derivation and its fixtures.
type AreaTable struct {
	Schema string  `json:"schema"`
	Areas  []*Area `json:"areas"`
}

func monstersByKey(db *data.DB) map[string]*data.Monster {
	m := make(map[string]*data.Monster, len(db.Monsters))
	for _, mon := range db.Monsters {
		m[strings.ToLower(mon.Key)] = mon
	}
	return m
}

// BuildAreaTable derives the table from the game tables. A nil db or one without levels is an error; towns and areas
// without spawns are left out.
func BuildAreaTable(db *data.DB) (*AreaTable, error) {
	if db == nil || len(db.Levels) == 0 || len(db.Monsters) == 0 {
		return nil, errors.New("readiness: area table needs the level and monster tables")
	}
	byKey := monstersByKey(db)
	t := &AreaTable{Schema: AreaTableSchema}
	for _, l := range db.Levels {
		a := &Area{ID: l.ID, Name: l.Name, Act: l.Act + 1}
		if a.Name == "" {
			a.Name = l.Key
		}
		spawns := false
		for d := 0; d < 3; d++ {
			pool := l.Mon
			if d > 0 {
				pool = l.NMon
			}
			ad := areaDiff(db, byKey, pool, l.UMon, l.MonLvl[d], d)
			if ad.Spawns > 0 {
				spawns = true
			}
			a.Diff[d] = ad
		}
		if spawns {
			t.Areas = append(t.Areas, a)
		}
	}
	if len(t.Areas) == 0 {
		return nil, errors.New("readiness: no area has spawns in the level table")
	}
	return t, nil
}

func areaDiff(db *data.DB, byKey map[string]*data.Monster, pool, elite []string, mlvl, d int) AreaDiff {
	out := AreaDiff{MLvl: mlvl}
	var scale *data.MonLevel
	if mlvl >= 0 && mlvl < len(db.MonLevels) {
		scale = db.MonLevels[mlvl]
	}
	pct := func(v int, p [3]int) int {
		if scale == nil || p[d] == 0 {
			return v
		}
		return v * p[d] / 100
	}
	immune := map[int]bool{}
	elitePossible := map[int]bool{}
	ranged := false
	for _, k := range pool {
		m := byKey[strings.ToLower(k)]
		if m == nil || !m.Enabled || m.NPC || !m.Killable {
			continue
		}
		out.Spawns++
		ranged = ranged || m.Ranged
		for e := 0; e < len(elementNames); e++ {
			if m.Res[d][e] >= 100 {
				immune[e] = true
			}
		}
		if m.Boss {
			continue
		}
		scaleLife, scaleDM, scaleTH := scaleOf(scale)
		out.TopLife = max(out.TopLife, pct(m.HPMax[d], scaleLife))
		hit := max(m.A1Max[d], m.A2Max[d], m.S1Max[d])
		for _, el := range m.Elem {
			hit = max(hit, el.Max)
		}
		out.TopHit = max(out.TopHit, pct(hit, scaleDM))
		out.TopToHit = max(out.TopToHit, pct(max(m.A1TH[d], m.A2TH[d], m.S1TH[d]), scaleTH))
	}
	for _, k := range append(slices.Clone(pool), elite...) {
		m := byKey[strings.ToLower(k)]
		if m == nil || !m.Enabled || m.NPC || !m.Killable {
			continue
		}
		for _, e := range []data.DamageType{data.Fire, data.Lightning, data.Cold} {
			if r := m.Res[d][e]; r < 100 && r+75 >= 100 { // the enchant bonus (+75) covers magic resistant (+40)
				elitePossible[int(e)] = true
			}
		}
		if r := m.Res[d][data.Poison]; r < 100 && r+40 >= 100 {
			elitePossible[int(data.Poison)] = true
		}
	}
	out.Immune = elementList(immune)
	out.EliteImmune = elementList(elitePossible)
	if out.Spawns > 0 {
		for _, a := range randomAffixes {
			if a == "magic_resistant" && d == 0 {
				continue // cannot be picked in Normal (V Arreat bonus page, docs/difficulty.md section 4)
			}
			out.Affixes = append(out.Affixes, a)
		}
		if ranged {
			out.Affixes = append(out.Affixes, "multiple_shots")
		}
	}
	return out
}

func scaleOf(s *data.MonLevel) (life, dm, th [3]int) {
	if s == nil {
		return
	}
	return s.HP, s.DM, s.TH
}

func elementList(set map[int]bool) []string {
	var out []string
	for e := 0; e < len(elementNames); e++ {
		if set[e] {
			out = append(out, elementNames[e])
		}
	}
	return out
}

// Marshal writes the table as indented JSON.
func (t *AreaTable) Marshal() ([]byte, error) { return json.MarshalIndent(t, "", " ") }

// ParseAreaTable reads and checks a table. Errors name the field.
func ParseAreaTable(b []byte) (*AreaTable, error) {
	var t AreaTable
	dec := json.NewDecoder(strings.NewReader(string(b)))
	dec.DisallowUnknownFields()
	if err := dec.Decode(&t); err != nil {
		return nil, fmt.Errorf("area table: %w", err)
	}
	if t.Schema != AreaTableSchema {
		return nil, fmt.Errorf("area table: schema %q, want %q", t.Schema, AreaTableSchema)
	}
	seen := map[int]bool{}
	for _, a := range t.Areas {
		if a == nil || a.Name == "" || seen[a.ID] {
			return nil, fmt.Errorf("area table: area %d is empty or repeated", a.ID)
		}
		seen[a.ID] = true
		for d := range a.Diff {
			for _, e := range append(slices.Clone(a.Diff[d].Immune), a.Diff[d].EliteImmune...) {
				if !slices.Contains(elementNames[:], e) {
					return nil, fmt.Errorf("area table: area %q %s: unknown element %q", a.Name, diffNames[d], e)
				}
			}
		}
	}
	return &t, nil
}

// Find looks an area up by id, display name or key (case-insensitive). An unknown area is an error naming it.
func (t *AreaTable) Find(ref string) (*Area, error) {
	ref = strings.TrimSpace(ref)
	if id, err := strconv.Atoi(ref); err == nil {
		for _, a := range t.Areas {
			if a.ID == id {
				return a, nil
			}
		}
	}
	for _, a := range t.Areas {
		if strings.EqualFold(a.Name, ref) {
			return a, nil
		}
	}
	return nil, fmt.Errorf("readiness: area %q is not in the area table", ref)
}
