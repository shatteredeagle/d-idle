package decide

import (
	"fmt"
	"math/rand"
)

// Profile is a simulated player: it decides what to reply to a thought. It knows nothing of the store's internals,
// so drills and tests share one generator (cautious, bold, inconsistent and seeded holdouts).
type Profile struct {
	Name string
	// Reply returns the answer id for an ask thought of point p whose untrained default is def. "" means the player
	// looked away (no answer), NotSure is the built-in extra answer.
	Reply func(rng *rand.Rand, p *Point, def string) string
	// Confirm returns "yes", "no" or "" (ignored) for a confirm thought.
	Confirm func(rng *rand.Rand) string
}

// ProfileNames are the three named profiles; "holdout" is the seeded fourth kind.
var ProfileNames = []string{"cautious", "bold", "inconsistent"}

func lastAnswer(p *Point) string { return p.Answers[len(p.Answers)-1].ID }

// KnownProfiles returns the three named profiles.
func KnownProfiles() []Profile {
	return []Profile{
		{"cautious", func(_ *rand.Rand, _ *Point, def string) string { return def },
			func(*rand.Rand) string { return "no" }},
		{"bold", func(rng *rand.Rand, p *Point, _ string) string {
			if rng.Intn(10) == 0 {
				return "" // looked away
			}
			return lastAnswer(p)
		}, func(*rand.Rand) string { return "yes" }},
		{"inconsistent", func(rng *rand.Rand, p *Point, _ string) string {
			if rng.Intn(5) == 0 {
				return NotSure
			}
			return p.Answers[rng.Intn(len(p.Answers))].ID
		}, func(rng *rand.Rand) string { return []string{"yes", "no", ""}[rng.Intn(3)] }},
	}
}

// Holdout builds a profile from a seed: a preferred answer per point, a flip, ignore and notsure rate. The same
// seed always gives the same player.
func Holdout(seed int64) Profile {
	g := rand.New(rand.NewSource(seed))
	flip, ignore, unsure, yes := g.Float64()*0.5, g.Float64()*0.6, g.Float64()*0.2, g.Float64()
	pref := map[string]int{}
	return Profile{
		Name: "holdout",
		Reply: func(rng *rand.Rand, p *Point, _ string) string {
			i, ok := pref[p.ID]
			if !ok {
				i = g.Intn(len(p.Answers))
				pref[p.ID] = i
			}
			switch x := rng.Float64(); {
			case x < ignore:
				return ""
			case x < ignore+unsure:
				return NotSure
			case x < ignore+unsure+flip:
				return p.Answers[rng.Intn(len(p.Answers))].ID
			}
			return p.Answers[i].ID
		},
		Confirm: func(rng *rand.Rand) string {
			if rng.Float64() < yes {
				return "yes"
			}
			return "no"
		},
	}
}

// NewProfile returns a named profile (cautious, bold, inconsistent) or the seeded holdout.
func NewProfile(name string, seed int64) (Profile, error) {
	if name == "holdout" {
		return Holdout(seed), nil
	}
	for _, p := range KnownProfiles() {
		if p.Name == name {
			return p, nil
		}
	}
	return Profile{}, fmt.Errorf("decide: unknown answer profile %q", name)
}
