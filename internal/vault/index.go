package vault

import (
	"sort"
	"strings"
)

// index holds posting lists (insertion positions) per searchable field. Lists only grow; a record that left
// the live set is skipped at query time, which keeps replay and insert O(1) per field.
type index struct {
	quality map[int][]int
	code    map[string][]int
	chain   map[string][]int
	sockets map[int][]int
}

func newIndex() index {
	return index{quality: map[int][]int{}, code: map[string][]int{}, chain: map[string][]int{}, sockets: map[int][]int{}}
}

func (x *index) add(r *Rec) {
	x.quality[r.Quality] = append(x.quality[r.Quality], r.seq)
	x.code[r.Code] = append(x.code[r.Code], r.seq)
	for _, c := range r.Chain {
		x.chain[c] = append(x.chain[c], r.seq)
	}
	x.sockets[r.Sockets] = append(x.sockets[r.Sockets], r.seq)
}

// Filter is the search sheet. Zero fields do not filter; pointers distinguish "false" from "unset".
type Filter struct {
	Quality    []int // any of
	Code       string
	Type       string // a member of the base type chain
	MinIlvl    int
	MaxIlvl    int // 0 = no limit
	Sockets    *int
	Eth        *bool
	Identified *bool
	Pinned     *bool
	CodePrefix []string // any of: the item code starts with one (runes r01..r33 by "r0","r1","r2","r3")
	Name       string   // case-insensitive substring
	States     []State  // default: live states
	Order      string   // "new" (default), "old", "value", "ilvl", "quality" (highest first), "name" (A to Z)
	Offset     int
	Limit      int // 0 = all
}

// Query returns the matching records and the total before Offset/Limit.
func (s *Store) Query(f Filter) ([]Rec, int) {
	s.mu.Lock()
	defer s.mu.Unlock()
	cand := s.candidates(f)
	name := strings.ToLower(f.Name)
	var hit []*Rec
	match := func(r *Rec) bool {
		if len(f.States) == 0 {
			if !r.State.Live() {
				return false
			}
		} else if !hasState(f.States, r.State) {
			return false
		}
		if len(f.Quality) > 0 && !hasInt(f.Quality, r.Quality) {
			return false
		}
		if f.Code != "" && r.Code != f.Code {
			return false
		}
		if f.Type != "" && !hasStr(r.Chain, f.Type) && r.Code != f.Type {
			return false
		}
		if r.Ilvl < f.MinIlvl || (f.MaxIlvl > 0 && r.Ilvl > f.MaxIlvl) {
			return false
		}
		if f.Sockets != nil && r.Sockets != *f.Sockets {
			return false
		}
		if f.Eth != nil && r.Eth != *f.Eth {
			return false
		}
		if f.Identified != nil && r.Identified != *f.Identified {
			return false
		}
		if f.Pinned != nil && r.Pin != *f.Pinned {
			return false
		}
		if len(f.CodePrefix) > 0 && !hasPrefix(f.CodePrefix, r.Code) {
			return false
		}
		return name == "" || strings.Contains(strings.ToLower(r.Name), name)
	}
	// A page of the default or oldest-first order reads the source from the matching end and keeps only the window it needs
	// (the source is in insertion order: seq is the slot index). The rest of the matches are only counted.
	if f.Limit > 0 && (f.Order == "" || f.Order == "new" || f.Order == "old") {
		n := len(s.recs)
		if cand != nil {
			n = len(cand)
		}
		rec := func(i int) *Rec {
			if cand != nil {
				return s.recs[cand[i]]
			}
			return s.recs[i]
		}
		want, total := f.Offset+f.Limit, 0
		for k := 0; k < n; k++ {
			i := k
			if f.Order != "old" {
				i = n - 1 - k
			}
			if r := rec(i); match(r) {
				if total >= f.Offset && total < want {
					hit = append(hit, r)
				}
				total++
			}
		}
		out := make([]Rec, len(hit))
		for i, r := range hit {
			out[i] = *r
		}
		return out, total
	}
	if cand == nil {
		for _, r := range s.recs {
			if match(r) {
				hit = append(hit, r)
			}
		}
	} else {
		for _, p := range cand {
			if r := s.recs[p]; match(r) {
				hit = append(hit, r)
			}
		}
	}
	switch f.Order {
	case "old":
		// Both scans (candidate lists and the records) run in insertion order, so the common orders need no sort.
		if !sort.SliceIsSorted(hit, func(i, j int) bool { return hit[i].seq < hit[j].seq }) {
			sort.SliceStable(hit, func(i, j int) bool { return hit[i].seq < hit[j].seq })
		}
	case "value":
		sort.SliceStable(hit, func(i, j int) bool {
			if hit[i].Value != hit[j].Value {
				return hit[i].Value > hit[j].Value
			}
			return hit[i].seq > hit[j].seq
		})
	case "ilvl":
		sort.SliceStable(hit, func(i, j int) bool {
			if hit[i].Ilvl != hit[j].Ilvl {
				return hit[i].Ilvl > hit[j].Ilvl
			}
			return hit[i].seq > hit[j].seq
		})
	case "quality":
		sort.SliceStable(hit, func(i, j int) bool {
			if hit[i].Quality != hit[j].Quality {
				return hit[i].Quality > hit[j].Quality
			}
			return hit[i].seq > hit[j].seq
		})
	case "name":
		sort.SliceStable(hit, func(i, j int) bool {
			a, b := strings.ToLower(hit[i].Name), strings.ToLower(hit[j].Name)
			if a != b {
				return a < b
			}
			return hit[i].seq > hit[j].seq
		})
	default:
		if sort.SliceIsSorted(hit, func(i, j int) bool { return hit[i].seq < hit[j].seq }) {
			for i, j := 0, len(hit)-1; i < j; i, j = i+1, j-1 {
				hit[i], hit[j] = hit[j], hit[i]
			}
		} else {
			sort.SliceStable(hit, func(i, j int) bool { return hit[i].seq > hit[j].seq })
		}
	}
	total := len(hit)
	if f.Offset > 0 {
		hit = hit[min(f.Offset, len(hit)):]
	}
	if f.Limit > 0 && len(hit) > f.Limit {
		hit = hit[:f.Limit]
	}
	out := make([]Rec, len(hit))
	for i, r := range hit {
		out[i] = *r
	}
	return out, total
}

// candidates picks the smallest posting list the filter allows, or nil for a full scan.
func (s *Store) candidates(f Filter) []int {
	var best []int
	have := false
	take := func(l []int) {
		if !have || len(l) < len(best) {
			best, have = l, true
		}
	}
	if f.Code != "" {
		take(s.ix.code[f.Code])
	}
	if f.Type != "" {
		take(s.ix.chain[f.Type])
	}
	if f.Sockets != nil {
		take(s.ix.sockets[*f.Sockets])
	}
	if len(f.Quality) == 1 {
		take(s.ix.quality[f.Quality[0]])
	}
	if !have {
		return nil
	}
	if best == nil {
		best = []int{}
	}
	return best
}

func hasInt(l []int, v int) bool {
	for _, x := range l {
		if x == v {
			return true
		}
	}
	return false
}
func hasPrefix(l []string, v string) bool {
	for _, x := range l {
		if strings.HasPrefix(v, x) {
			return true
		}
	}
	return false
}

func hasStr(l []string, v string) bool {
	for _, x := range l {
		if x == v {
			return true
		}
	}
	return false
}
func hasState(l []State, v State) bool {
	for _, x := range l {
		if x == v {
			return true
		}
	}
	return false
}

// Facets counts the live records by item type (every member of the base type chain) for the filter chips.
type Facets struct {
	Types map[string]int
}

func (s *Store) Facets() Facets {
	s.mu.Lock()
	defer s.mu.Unlock()
	f := Facets{Types: map[string]int{}}
	for t, l := range s.ix.chain {
		for _, p := range l {
			if s.recs[p].State.Live() {
				f.Types[t]++
			}
		}
	}
	return f
}
