package campaign

import (
	"fmt"
	"strings"
)

// Band is how ready the hero is for a chapter. Never a lock: a too-early chapter can still be started with Go now.
type Band string

const (
	TooEarly    Band = "too_early"   // below the floor
	Ready       Band = "ready"       // at or above the floor, below the recommended level (or a readiness check fails)
	Comfortable Band = "comfortable" // at or above the recommended level and no readiness check fails
)

// Hero is what the bands read.
type Hero struct {
	Class string // lowercased d2.Class name ("paladin")
	Level int
	Merc  bool // a living merc
}

// Check is one readiness check line (internal/readiness Result: id, state pass|warn|fail|skip, reason).
type Check struct{ ID, State, Reason string }

// Readiness is the band of one chapter with its numbers and reasons.
type Readiness struct {
	Chapter     string   `json:"chapter"`
	Band        Band     `json:"band"`
	Floor       int      `json:"floor"`
	Recommended int      `json:"recommended"`
	Reasons     []string `json:"reasons,omitempty"`
}

// Rate computes the band of c for h. checks are the readiness results for the chapter's area; only the ids the chapter
// lists count, a fail caps the band at Ready and a warn only adds its reason.
func Rate(c *Chapter, h Hero, checks []Check) (Readiness, error) {
	l, err := c.LevelsFor(h.Class)
	if err != nil {
		return Readiness{}, err
	}
	floor, rec := l.For(h.Merc)
	r := Readiness{Chapter: c.ID, Floor: floor, Recommended: rec}
	switch {
	case h.Level < floor:
		r.Band = TooEarly
		r.Reasons = append(r.Reasons, fmt.Sprintf("clvl %d < floor %d", h.Level, floor))
	case h.Level < rec:
		r.Band = Ready
		r.Reasons = append(r.Reasons, fmt.Sprintf("clvl %d < recommended %d", h.Level, rec))
	default:
		r.Band = Comfortable
	}
	want := map[string]bool{}
	for _, id := range c.Checks {
		want[id] = true
	}
	for _, k := range checks {
		if !want[k.ID] {
			continue
		}
		switch k.State {
		case "fail":
			if r.Band == Comfortable {
				r.Band = Ready
			}
			r.Reasons = append(r.Reasons, k.ID+": "+k.Reason)
		case "warn":
			r.Reasons = append(r.Reasons, k.ID+": "+k.Reason)
		}
	}
	return r, nil
}

// withWall raises the floor (and the recommended level) to a safety wall and recomputes the level part of the band.
func (r Readiness) withWall(wall, lvl int) Readiness {
	if wall <= r.Floor {
		return r
	}
	r.Floor, r.Recommended = wall, max(r.Recommended, wall)
	reasons := []string{fmt.Sprintf("safety wall: clvl %d", wall)}
	for _, s := range r.Reasons {
		if !strings.HasPrefix(s, "clvl ") {
			reasons = append(reasons, s)
		}
	}
	switch {
	case lvl < r.Floor:
		r.Band = TooEarly
	case lvl < r.Recommended:
		r.Band = Ready
	case r.Band == TooEarly:
		r.Band = Comfortable
	}
	r.Reasons = reasons
	return r
}

// GoAt is the player's policy row: the band from which Auto goes.
type GoAt string

const (
	GoRecommended GoAt = "recommended" // default: Auto goes when Comfortable
	GoFloor       GoAt = "floor"       // Auto goes when Ready
)

// enough reports whether band b is enough to go under policy g.
func (g GoAt) enough(b Band) bool {
	if g == GoFloor {
		return b == Ready || b == Comfortable
	}
	return b == Comfortable
}
