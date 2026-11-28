package supply

import (
	"bytes"
	_ "embed"
	"encoding/json"
	"fmt"
	"sort"
	"strconv"
	"strings"
	"sync"
)

// Line is one planned supply. Only the lines the first stages act on are modelled; repair, ammo, merc and keys join
// with the cards that need them.
type Line string

const (
	Healing      Line = "healing"
	Mana         Line = "mana"
	Rejuvenation Line = "rejuvenation"
	Portal       Line = "portal"
	GoldCarry    Line = "gold_carry" // gold the excursion itself needs in the purse
)

var lines = map[Line]bool{Healing: true, Mana: true, Rejuvenation: true, Portal: true, GoldCarry: true}

// Qty is one line of a need: below Min the hero does not leave, Target is what a visit fills to when gold allows.
type Qty struct {
	Min, Target int
	Max         int    // the most the hero can carry of this line (0 = unknown); a Min above it is clamped
	MinWhy      string // the rule that set the min, for the trace and the viewer
	TierFloor   int    // lowest potion tier worth counting for this line (0 = any); carried as data, the shop and counter read it later
}

// Need is the supply need of one excursion.
type Need struct {
	Excursion string
	Profile   string
	Lines     map[Line]Qty
}

// Inputs are the build's own needs, read by the caller from the build profile and the belt plan (never from a class
// list here).
type Inputs struct {
	ManaNeed   int          // 0, 1 or 2: how much the build uses mana potions
	BeltSlots  int          // potion boxes of the equipped belt (0 = unknown: the row's own min applies)
	Capacity   map[Line]int // belt capacity plus pack buffer per line, 0 or absent = unknown
	BaseTarget map[Line]int // belt capacity plus pack buffer: the target when gold allows (section 3.2 layer 1)
}

type row struct {
	Min        int `json:"min"`
	Target     int `json:"target"`
	ManaNeedAt int `json:"manaNeedAtLeast"` // the line applies only to a build whose ManaNeed is at least this
	TierFloor  int `json:"tierFloor"`       // lowest potion tier that counts toward the line (0 = any)
	// BeltSlots maps a belt size to the row's min and target for belts of at least that many potion boxes (the largest
	// size not above the hero's belt wins). A belt smaller than every key keeps the row's own min. Keys are integers.
	BeltSlots map[string]beltRow `json:"beltSlots"`
}

type beltRow struct {
	Min    int `json:"min"`
	Target int `json:"target"`
}

// forBelt resolves the BeltSlots condition: the row to use for a belt of slots boxes.
func (r row) forBelt(slots int) row {
	if len(r.BeltSlots) == 0 || slots <= 0 {
		return r
	}
	best, found := -1, false
	for k := range r.BeltSlots {
		n, _ := strconv.Atoi(k) // validated at load
		if n <= slots && n > best {
			best, found = n, true
		}
	}
	if !found {
		return r // smaller than every key: the row's own min stands and the capacity clamp lowers it
	}
	b := r.BeltSlots[strconv.Itoa(best)]
	r.Min, r.Target = b.Min, b.Target
	return r
}

//go:embed profiles.json
var profilesJSON []byte

var (
	profOnce sync.Once
	profs    map[string]map[Line]row
	profErr  error
)

func loadProfiles() (map[string]map[Line]row, error) {
	profOnce.Do(func() {
		var f struct {
			Comment  string                    `json:"_comment"`
			Profiles map[string]map[string]row `json:"profiles"`
		}
		dec := json.NewDecoder(bytes.NewReader(profilesJSON))
		dec.DisallowUnknownFields() // an unknown condition key is an error, not a silently ignored row
		if err := dec.Decode(&f); err != nil {
			profErr = fmt.Errorf("supply: profiles.json: %w", err)
			return
		}
		profs = map[string]map[Line]row{}
		for name, ls := range f.Profiles {
			m := map[Line]row{}
			for l, r := range ls {
				if !lines[Line(l)] {
					profErr = fmt.Errorf("supply: profiles.json: profile %q names unknown line %q", name, l)
					return
				}
				if r.Min < 0 || r.Target < 0 {
					profErr = fmt.Errorf("supply: profiles.json: profile %q line %q has a negative number", name, l)
					return
				}
				for k, b := range r.BeltSlots {
					if n, err := strconv.Atoi(k); err != nil || n <= 0 || b.Min < 0 || b.Target < 0 {
						profErr = fmt.Errorf("supply: profiles.json: profile %q line %q has a bad beltSlots entry %q", name, l, k)
						return
					}
				}
				m[Line(l)] = r
			}
			profs[name] = m
		}
	})
	return profs, profErr
}

// Profiles lists the profile names, sorted.
func Profiles() ([]string, error) {
	p, err := loadProfiles()
	if err != nil {
		return nil, err
	}
	names := make([]string, 0, len(p))
	for n := range p {
		names = append(names, n)
	}
	sort.Strings(names)
	return names, nil
}

// NeedFor derives the need of an excursion from a profile row and the build's inputs. An unknown profile is an error
// that names it; there is no fallback profile.
func NeedFor(excursion, profile string, in Inputs) (Need, error) {
	p, err := loadProfiles()
	if err != nil {
		return Need{}, err
	}
	rows, ok := p[profile]
	if !ok {
		names, _ := Profiles()
		return Need{}, fmt.Errorf("supply: unknown profile %q (known: %s)", profile, strings.Join(names, ", "))
	}
	n := Need{Excursion: excursion, Profile: profile, Lines: map[Line]Qty{}}
	for l, r := range rows {
		if in.ManaNeed < r.ManaNeedAt {
			continue
		}
		r = r.forBelt(in.BeltSlots)
		q := Qty{Min: r.Min, Target: max(r.Target, r.Min, in.BaseTarget[l]), Max: in.Capacity[l], MinWhy: "profile " + profile, TierFloor: r.TierFloor}
		n.Lines[l] = q
	}
	return n, nil
}
