package scenario

import (
	"encoding/json"
	"fmt"
	"os"
	"regexp"
	"strings"
	"time"
)

// IncidentVersion is the version of the incident files the bot writes at a death (internal/bot/incident.go).
const IncidentVersion = "didle.incident/1"

// Incident is a recorded state of a live death: the scenario `pre` of that moment plus the facts a pre cannot hold. It is
// tagged SETUP (docs/probes/incident-scenarios.md): a row built from it tests mechanics from a recorded state, and is
// never evidence about game behaviour. Report-only tier: tools only, not the merge gate (ADR 0020 amended 2026-11-25).
type Incident struct {
	Version  string    `json:"version"`
	Kind     string    `json:"kind"`
	Time     time.Time `json:"time"`
	AreaName string    `json:"areaName"`
	Class    string    `json:"class"`
	Build    string    `json:"build,omitempty"`
	Tags     []string  `json:"tags"`
	Pre      struct {
		Start      string `json:"start"`
		Area       int    `json:"area"`
		Level      int    `json:"level"`
		Gold       int    `json:"gold"`
		Stash      int    `json:"stash"`
		Difficulty string `json:"difficulty"`
		LifePct    int    `json:"life_pct"`
	} `json:"pre"`
	Facts struct {
		BeltHP       int    `json:"beltHP"`
		BeltMP       int    `json:"beltMP"`
		PackHP       int    `json:"packHP"`
		TownPortals  int    `json:"townPortals"`
		HostilesNear int    `json:"hostilesNear"`
		Killer       string `json:"killer"`
		Summary      string `json:"summary"`
		Bug          bool   `json:"bug"`
		Invariant    string `json:"invariant"` // kind "monitor": the invariant row that fired
		Hard         bool   `json:"hard"`
	} `json:"facts"`
}

// LoadIncident reads one incident file; unknown fields are an error.
func LoadIncident(path string) (Incident, error) {
	var in Incident
	b, err := os.ReadFile(path)
	if err != nil {
		return in, err
	}
	if err := json.Unmarshal(b, &in); err != nil {
		return in, fmt.Errorf("%s: %w", path, err)
	}
	if in.Version != IncidentVersion {
		return in, fmt.Errorf("%s: version %q, want %q", path, in.Version, IncidentVersion)
	}
	return in, nil
}

// Scenario turns the incident into a behaviour scenario of the nightly tier: the hero starts where and as it was, the bot
// plays on, and the run is red when the hero dies again (or, for a potion-bug death, when the autopsy flags the bug). U:
// the potion tier is not recorded, so the belt gets the lowest tier; the monsters that killed the hero are not recorded
// as units, so the row proves the state of the hero, not the situation (add pre.spawn by hand to build it).
func (in Incident) Scenario(class string, seed int64) (Scenario, error) {
	start := StartField
	if in.Pre.Start == StartTown {
		start = StartTown // a monitor hit in town replays from town
	}
	if in.Pre.Area == 0 && start == StartField {
		return Scenario{}, fmt.Errorf("incident of %s: no area", in.AreaName)
	}
	if class == "" {
		class = in.Class
	}
	diff := in.Pre.Difficulty
	if diff == "" {
		diff = "normal"
	}
	pre := Pre{HeroProfile: HeroBanked, Start: start, Area: in.Pre.Area, Level: in.Pre.Level, Gold: in.Pre.Gold, Stash: in.Pre.Stash, Difficulty: diff, LifePct: in.Pre.LifePct}
	for i := 0; i < in.Facts.BeltHP; i++ {
		pre.Belt = append(pre.Belt, "hp1")
	}
	for i := 0; i < in.Facts.BeltMP; i++ {
		pre.Belt = append(pre.Belt, "mp1")
	}
	zero := 0
	assert := []Assert{{Kind: ALogLine, Pattern: "character died", Max: &zero, Why: "the hero does not die again from the recorded state"}}
	if in.Facts.Bug {
		assert = append(assert, Assert{Kind: ALogLine, Pattern: "BUG: .* healing potions in the belt", Max: &zero, Why: "no death with a belt of potions undrunk"})
	}
	label := in.Kind
	idTail := ""
	if in.Kind == "monitor" {
		if in.Facts.Invariant == "" {
			return Scenario{}, fmt.Errorf("incident of %s: kind monitor without an invariant name", in.AreaName)
		}
		// A monitor row is red when the same invariant fires again; the death assert stays (a monitor hit often precedes one).
		assert = append(assert, Assert{Kind: ALogLine, Pattern: "invariant " + regexp.QuoteMeta(in.Facts.Invariant) + `\b`, Max: &zero, Why: "the invariant " + in.Facts.Invariant + " does not fire again from the recorded state"})
		label, idTail = "monitor "+in.Facts.Invariant, "-"+in.Facts.Invariant
	}
	assert = append(assert, Assert{Kind: ANoWrites})
	id := "incident." + strings.ToLower(strings.NewReplacer(" ", "-", "'", "").Replace(in.AreaName)) + "-" + fmt.Sprint(in.Time.Unix()) + idTail
	s := Scenario{ID: id, Title: "Incident replay (report-only): " + label + " in " + in.AreaName + " at level " + fmt.Sprint(in.Pre.Level),
		Kind: KindBehaviour, Tier: TierNightly, Pre: pre, Trigger: Trigger{Goals: "campaign"}, Budget: Budget{VirtualMin: 15},
		Assert: assert, Matrix: Matrix{Classes: []string{class}, Seeds: []int64{seed}},
		Provenance: "SETUP from a recorded incident (" + in.Time.UTC().Format(time.RFC3339) + "): level, gold, belt counts and position are written by lab verbs; CHEATED position. Mechanics only, never evidence about game behaviour. " + in.Facts.Summary,
	}
	s.applyDefaults()
	return s, s.Validate()
}
