package combat

import (
	"fmt"
	"sort"
	"strings"

	"d2lod-bot/internal/d2"
)

// Driver hold log (TASK-1127). The delivery drivers (whirl, escape, minion, corpse explosion) return silently when a
// gate holds them, so a drill whose driver never fires could not be read from the log. Each gate of a LEARNED row notes
// its reason here; the fight end (telemSummaries) logs one `combat hold` line per driver with the ticks per reason and
// clears the notes. Nothing is noted at knob combat.deliveryUpkeep 0 (the drivers do not act there).
const (
	holdWhirl  = "whirl"
	holdEscape = "escape"
	holdMinion = "minion"
	holdCorpse = "corpse-explosion"

	holdNoPack    = "no pack"
	holdLifeFloor = "life floor"
	holdCommit    = "commit gate"
	holdNoTrigger = "no surround or life above the line"
	holdNoCell    = "no cell"
	holdUnbound   = "no key and not on the right hand"
	holdNoCorpse  = "no corpse in reach"
	holdNoHostile = "no hostile"
)

// hold notes one held tick of a driver.
func (f *Fighter) hold(driver, why string) {
	if f.upkeepMode() == 0 || why == "" {
		return
	}
	if f.holds == nil {
		f.holds = map[string]map[string]int{}
	}
	if f.holds[driver] == nil {
		f.holds[driver] = map[string]int{}
	}
	f.holds[driver][why]++
}

// learned says the hero has a point in the skill (a hold of an unlearned row is not worth a line).
func (f *Fighter) learned(p *d2.Player, name string) bool {
	_, ok := KnownSkills(p, f.cat, f.keys)[name]
	return ok
}

// logHolds writes the fight's hold lines, drivers and reasons in a stable order, and clears the notes.
func (f *Fighter) logHolds() {
	drivers := make([]string, 0, len(f.holds))
	for d := range f.holds {
		drivers = append(drivers, d)
	}
	sort.Strings(drivers)
	for _, d := range drivers {
		var why []string
		ticks := 0
		for w, n := range f.holds[d] {
			why = append(why, fmt.Sprintf("%s:%d", w, n))
			ticks += n
		}
		sort.Strings(why)
		f.info("hold", "driver", d, "ticks", ticks, "why", strings.Join(why, "; "))
	}
	f.holds = nil
}
