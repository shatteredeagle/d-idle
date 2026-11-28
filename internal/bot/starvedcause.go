package bot

import (
	"fmt"
	"sort"
	"strings"

	"d2lod-bot/internal/arbiter"
	"d2lod-bot/internal/questboard"
)

// starvedCause is the text of an arbiter-starved incident. The row used to carry only the starved intent's own reason
// ("nothing else wanted: level the hero", 32 times in 12 h on two sandboxes), which says nothing about WHY it waited.
// Now it names the intent that won every decision since, the area the arbiter stood in, and (quest board on) the quests
// the board holds parked, with their reason, because a parked chunk is the usual reason the leveling filler is all that runs.
func (r *Runner) starvedCause(d arbiter.Decision, in arbiter.Intent) string {
	return starvedText(in, d, r.boardParkedNote())
}

func starvedText(in arbiter.Intent, d arbiter.Decision, parked string) string {
	var b strings.Builder
	fmt.Fprintf(&b, "%s: %s; outranked", in.Key(), in.Reason)
	switch {
	case d.Chosen == nil:
		b.WriteString(" while the arbiter chose nothing")
	case d.Chosen.Key() == in.Key():
		b.WriteString(" (it is chosen now)")
	default:
		fmt.Fprintf(&b, " by %s (%s)", d.Chosen.Key(), d.Chosen.Reason)
	}
	if d.Area != "" {
		fmt.Fprintf(&b, ", in %s", d.Area)
	}
	if parked != "" {
		b.WriteString("; quest board: " + parked)
	}
	return b.String()
}

// boardParkedNote lists the parked chunks of the board with the code that parked them ("" when none or the board is off).
func (r *Runner) boardParkedNote() string {
	rt := r.qbRT()
	if rt == nil || !r.questBoardOn() {
		return ""
	}
	runs := rt.exec.SMRuns()
	var out []string
	for k, run := range runs {
		if run.Status == questboard.RunParked {
			out = append(out, k+" "+run.Code)
		}
	}
	sort.Strings(out)
	return strings.Join(out, ", ")
}

// standingFiller reports whether in is a filler its source proposes all the time (no TTL): "nothing else wanted".
// Losing to any objective is its normal state, not starvation.
func standingFiller(in arbiter.Intent) bool { return in.Class == arbiter.Filler && in.TTL == 0 }

// noteArbiterStarved raises the soft invariant arbiter-starved for the starved intents of d, except a standing filler:
// on two sandboxes every one of 31 WARNs in 6.5 h was level/sweep outranked by quest/step (card evid1).
func (r *Runner) noteArbiterStarved(d arbiter.Decision) {
	for _, in := range d.Starved {
		if !standingFiller(in) {
			r.inv.noteStarved(r.starvedCause(d, in))
		}
	}
}
