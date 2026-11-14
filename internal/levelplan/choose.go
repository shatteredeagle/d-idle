package levelplan

import "fmt"

// Hero is what the slice gates read.
type Hero struct {
	Level   int
	Merc    bool // a living merc
	FullKit bool // a full belt and a town portal (the bot's fullKit)
}

// Gate says whether h may farm s and why not. It chooses a farm place; it never walls a chapter (the journey's bands
// and readiness checks stay the only word on quests).
func Gate(s Slice, h Hero) (bool, string) {
	if s.Kind != KindFarm {
		return false, "not a farm slice"
	}
	if e := s.EntryFor(h.Merc); h.Level < e {
		if !h.Merc && h.Level >= s.Entry {
			return false, fmt.Sprintf("clvl %d < %d without a merc", h.Level, e)
		}
		return false, fmt.Sprintf("clvl %d < entry %d", h.Level, e)
	}
	if h.Level < s.KitClvl && !h.FullKit {
		return false, fmt.Sprintf("clvl %d < %d needs a full kit", h.Level, s.KitClvl)
	}
	return true, ""
}

// Choice is the farm place Choose picks.
type Choice struct {
	Slice  string
	Area   int
	Fresh  bool
	PerMin float64 // XP per minute expected there (re-clear factor applied)
}

// Choose picks where h farms: the best XP per minute among the fresh areas of the slices whose band holds the level;
// when none is fresh, the best of (a) a fresh area of any slice h may enter past its band and (b) a re-clear of an
// in-band area at reclear times its rate. That is the "advance before re-clearing" rule: a cleared band never holds the
// hero while an open later field pays more. Areas missing from the table rank last; ok is false when no gate passes.
func (c *ClassPlan) Choose(h Hero, fresh func(area int) bool, t Table, reclear float64) (Choice, bool) {
	var best Choice
	found := false
	consider := func(s Slice, a int, f bool, factor float64) {
		x, ok := t[a]
		rate := -1.0
		if ok {
			rate = x.PerMin(h.Level) * factor
		}
		if !found || rate > best.PerMin {
			best, found = Choice{Slice: s.ID, Area: a, Fresh: f, PerMin: rate}, true
		}
	}
	for pass := 0; pass < 2 && !found; pass++ {
		for _, s := range c.Farms() {
			if ok, _ := Gate(s, h); !ok {
				continue
			}
			inBand := h.Level <= s.Exit
			gotFresh := false // areas are in route order: only the first fresh one is open (Cave 2 lies behind Cave 1)
			for _, a := range s.Areas {
				f := fresh(a) && !gotFresh
				gotFresh = gotFresh || fresh(a)
				switch {
				case pass == 0 && inBand && f:
					consider(s, a, true, 1)
				case pass == 1 && f:
					consider(s, a, true, 1)
				case pass == 1 && inBand && !fresh(a):
					consider(s, a, false, reclear)
				}
			}
		}
	}
	if !found { // every gated slice is past its band and cleared: re-clear the last one
		for _, s := range c.Farms() {
			if ok, _ := Gate(s, h); ok {
				for _, a := range s.Areas {
					consider(s, a, fresh(a), reclear)
				}
			}
		}
	}
	return best, found
}
