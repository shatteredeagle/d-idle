package resource

import (
	_ "embed"
	"encoding/json"
	"fmt"
	"strings"
	"sync"

	"d2lod-bot/internal/env"
)

// The belt plan knob (TASK-164, docs/belt-plan.md): the player's choice of what the four belt columns hold. The build's own
// plan (PlanForBuild) stays the default; a template replaces all four columns and a column override replaces one.

//go:embed beltspec.json
var beltSpecJSON []byte

// Column override values of resource.beltCol1..4.
const (
	ColPlan  = 0 // keep the template's (or the build's) column
	ColLife  = 1
	ColMana  = 2
	ColRejuv = 3
	ColEmpty = 4 // nothing is bought or belted for this column
)

// SpecTemplate is one row of beltspec.json.
type SpecTemplate struct {
	ID    int      `json:"id"`
	Name  string   `json:"name"`
	Cols  []string `json:"cols"`
	Label string   `json:"label"`
	Risk  string   `json:"risk"`
}

var (
	specOnce sync.Once
	specTpl  []SpecTemplate
)

// SpecTemplates lists the templates (id 0, the build default, is not in the table).
func SpecTemplates() []SpecTemplate {
	specOnce.Do(func() {
		var f struct {
			Templates []SpecTemplate `json:"templates"`
		}
		if err := json.Unmarshal(beltSpecJSON, &f); err != nil {
			panic("resource: bad embedded beltspec.json: " + err.Error())
		}
		specTpl = f.Templates
	})
	return specTpl
}

// TemplateID resolves a template name ("default" or "" is 0). An unknown name is an error that lists the known ones.
func TemplateID(name string) (int, error) {
	n := strings.ToLower(strings.TrimSpace(name))
	if n == "" || n == "default" {
		return 0, nil
	}
	var known []string
	for _, t := range SpecTemplates() {
		if t.Name == n {
			return t.ID, nil
		}
		known = append(known, t.Name)
	}
	return 0, fmt.Errorf("resource: unknown belt plan template %q (known: default, %s)", name, strings.Join(known, ", "))
}

// PlanSpec is the player's belt plan choice. The zero value keeps the build's plan.
type PlanSpec struct {
	Template int
	Cols     [4]int
}

// SpecOf reads the spec from the policy tuning.
func SpecOf(t Tuning) PlanSpec {
	return PlanSpec{Template: t.BeltPlan, Cols: [4]int{t.BeltCol1, t.BeltCol2, t.BeltCol3, t.BeltCol4}}
}

// SpecOfEnv reads the spec from the shared tuning (town and loot read the env group directly).
func SpecOfEnv(r env.ResourceTuning) PlanSpec {
	return PlanSpec{Template: r.BeltPlan, Cols: [4]int{r.BeltCol1, r.BeltCol2, r.BeltCol3, r.BeltCol4}}
}

// IsDefault reports that the spec changes nothing.
func (s PlanSpec) IsDefault() bool { return s == PlanSpec{} }

// Apply returns the plan with the spec laid over it. MaxTier and PrefTier stay: the tier follows the level and the
// potion tier option. An unknown template or column value is an error that names it (no silent fallback).
func (s PlanSpec) Apply(p BeltPlan) (BeltPlan, error) {
	if s.Template != 0 {
		found := false
		for _, t := range SpecTemplates() {
			if t.ID != s.Template {
				continue
			}
			found = true
			for i, c := range t.Cols {
				if i >= len(p.Cols) {
					break
				}
				k, err := specKind(c)
				if err != nil {
					return p, fmt.Errorf("resource: belt plan template %q: %w", t.Name, err)
				}
				p.Cols[i] = k
			}
		}
		if !found {
			return p, fmt.Errorf("resource: unknown belt plan template id %d (resource.beltPlan)", s.Template)
		}
	}
	for i, c := range s.Cols {
		switch c {
		case ColPlan:
		case ColLife:
			p.Cols[i] = PotionLife
		case ColMana:
			p.Cols[i] = PotionMana
		case ColRejuv:
			p.Cols[i] = PotionRejuv
		case ColEmpty:
			p.Cols[i] = PotionNone
		default:
			return p, fmt.Errorf("resource: unknown belt column value %d (resource.beltCol%d: 0 plan, 1 healing, 2 mana, 3 rejuvenation, 4 empty)", c, i+1)
		}
	}
	return p, nil
}

func specKind(c string) (Potion, error) {
	switch c {
	case "hp":
		return PotionLife, nil
	case "mp":
		return PotionMana, nil
	case "rv":
		return PotionRejuv, nil
	case "", "none":
		return PotionNone, nil
	}
	return PotionNone, fmt.Errorf("unknown column kind %q", c)
}

// PlanFor returns the plan the spec makes of the build's own plan, with the error of Apply.
func (s PlanSpec) PlanFor(in Input) (BeltPlan, error) {
	return s.Apply(PlanForBuild(in.Class, in.Level, in.Diff, in.Build))
}

// HasHealing reports a healing or rejuvenation column: the potions that answer low life. A plan without one is the
// player's call (docs/belt-plan.md); the kit gate then asks for no healing potion, and the portal stays the way out.
func (p BeltPlan) HasHealing() bool {
	for _, c := range p.Cols {
		if c == PotionLife || c == PotionRejuv {
			return true
		}
	}
	return false
}

// KitHealing is the healing count the kit checks (KitShortfall, KitLow) see. A plan without a healing column asks for
// none, so an all-mana belt is not reported low for lacking what it does not plan; the emergency exits (town portal,
// the chicken line) are not part of this and stay as they are.
func KitHealing(plan BeltPlan, healing int, t Tuning) int {
	if plan.HasHealing() {
		return healing
	}
	return max(healing, t.HPFirst, t.RecoveryPotionFloor)
}

// HasMana reports a mana column.
func (p BeltPlan) HasMana() bool {
	for _, c := range p.Cols {
		if c == PotionMana {
			return true
		}
	}
	return false
}
