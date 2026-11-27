// Package charms holds the charm facts as data (charms.json): which item codes are charms, how many pack cells each
// takes and which qualities exist. Nothing here reads the game; the sizes are U until checked against misc.txt.
package charms

import (
	_ "embed"
	"encoding/json"
	"strings"
)

//go:embed charms.json
var raw []byte

// Base is one charm base item.
type Base struct {
	Code string `json:"code"`
	Type string `json:"type"`
	Name string `json:"name"`
	W    int    `json:"w"`
	H    int    `json:"h"`
}

// Cells is the pack cells the base takes.
func (b Base) Cells() int { return b.W * b.H }

type table struct {
	Legal  []string `json:"legalQuality"`
	LegalN []int    `json:"legalQualityNum"` // the d2 quality numbers of the same words (4 magic, 7 unique)
	Bases  []Base   `json:"bases"`
}

var tab = func() table {
	var t table
	if err := json.Unmarshal(raw, &t); err != nil {
		panic("charms: bad charms.json: " + err.Error())
	}
	return t
}()

// Lookup finds a charm base by item code ("cm1") or by type code ("scha").
func Lookup(code string) (Base, bool) {
	code = strings.ToLower(code)
	for _, b := range tab.Bases {
		if b.Code == code || b.Type == code {
			return b, true
		}
	}
	return Base{}, false
}

// Is says whether the code is a charm.
func Is(code string) bool { _, ok := Lookup(code); return ok }

// LegalQuality says whether a charm of that quality word (magic, unique, ...) exists in the game.
func LegalQuality(q string) bool {
	for _, l := range tab.Legal {
		if l == strings.ToLower(q) {
			return true
		}
	}
	return false
}

// LegalQualityNum is LegalQuality for the game's quality number.
func LegalQualityNum(q int) bool {
	for _, l := range tab.LegalN {
		if l == q {
			return true
		}
	}
	return false
}
