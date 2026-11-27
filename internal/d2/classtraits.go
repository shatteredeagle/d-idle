package d2

import (
	_ "embed"
	"encoding/json"
	"fmt"
	"strings"
)

// ClassTraits are the per-class facts several packages used to write out as a class list (build flexibility audit rows 27
// to 29). Caster: every attack costs mana and life is thin (gear gate, den floor, merc auras). ManaNeed ranks the mana
// dependence: 0 none, 1 occasional, 2 every attack (docs/classes/*.md section 7, S). The numbers live in classtraits.json.
type ClassTraits struct {
	Caster   bool `json:"caster"`
	ManaNeed int  `json:"manaNeed"`
	// MercEarlyRogue: the class takes the Act 1 Rogue (cold arrow, slows) before Blood Raven is done, because its own damage
	// comes late (summoners; Necromancer SME delta, docs/probes/necro-readiness-2026-11-26.md, S). Default false.
	MercEarlyRogue bool `json:"mercEarlyRogue"`
	// MercAura: the Act 2 hireling aura the class wants, by difficulty ("normal", "nightmare", "hell") to a skills.txt aura
	// name. An absent difficulty means "no class preference" (the caller's default). Design: docs/design/merc-aura-by-archetype.md.
	MercAura map[string]string `json:"mercAura"`
}

// MercAuraFor returns the class's wanted merc aura for a difficulty ("normal", "nightmare", "hell", case-insensitive).
// ok is false when the class has no preference; an unknown class is an error naming it.
func MercAuraFor(c Class, diff string) (aura string, ok bool, err error) {
	t, err := TraitsOf(c)
	if err != nil {
		return "", false, err
	}
	aura, ok = t.MercAura[strings.ToLower(diff)]
	return aura, ok, nil
}

//go:embed classtraits.json
var classTraitsJSON []byte

var classTraits = func() map[string]ClassTraits {
	m := map[string]ClassTraits{}
	if err := json.Unmarshal(classTraitsJSON, &m); err != nil {
		panic("d2: classtraits.json: " + err.Error())
	}
	for _, n := range classNames {
		if _, ok := m[n]; !ok {
			panic("d2: classtraits.json has no row for " + n)
		}
		for d, a := range m[n].MercAura {
			if (d != "normal" && d != "nightmare" && d != "hell") || a == "" {
				panic("d2: classtraits.json: bad mercAura entry for " + n + ": " + d)
			}
		}
	}
	return m
}()

// TraitsOf returns the row of a class; an unknown class is an error naming it, never another class's row.
func TraitsOf(c Class) (ClassTraits, error) {
	t, ok := classTraits[c.String()]
	if !ok {
		return ClassTraits{}, fmt.Errorf("d2: no class traits for %s", c)
	}
	return t, nil
}

// TraitsByName is TraitsOf for the class name strings some packages carry ("Sorceress").
func TraitsByName(name string) (ClassTraits, error) {
	t, ok := classTraits[name]
	if !ok {
		return ClassTraits{}, fmt.Errorf("d2: no class traits for %q", name)
	}
	return t, nil
}

// Caster reports the caster trait; false for an unknown class (a corrupt class byte is not a caster, and TraitsOf names it).
func (c Class) Caster() bool { t, _ := TraitsOf(c); return t.Caster }

// ManaNeed is the mana dependence rank; an unknown class reads 1 (occasional), the old default for classes without a row.
func (c Class) ManaNeed() int {
	if t, err := TraitsOf(c); err == nil {
		return t.ManaNeed
	}
	return 1
}
