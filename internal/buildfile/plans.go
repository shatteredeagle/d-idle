package buildfile

import (
	"encoding/json"
	"fmt"
	"os"
	"sort"
	"strings"

	"d2lod-bot/internal/leveler"
	"d2lod-bot/internal/plandata"
)

// The shipped plan tables (internal/plandata, build-*.json) as build files and back: a guide track is a file
// (ADR 0020, docs/agency.md). The table format stays what the leveler reads; the build file is the shareable form.

// PlanNames lists the shipped plan tables ("paladin-normal30", ...), sorted.
func PlanNames() []string {
	ents, _ := plandata.FS.ReadDir(".")
	var out []string
	for _, e := range ents {
		if n := e.Name(); strings.HasPrefix(n, "build-") && strings.HasSuffix(n, ".json") && n != plandata.CatalogFile {
			out = append(out, strings.TrimSuffix(strings.TrimPrefix(n, "build-"), ".json"))
		}
	}
	sort.Strings(out)
	return out
}

// ExportPlan turns a shipped table into a build file. The notes say what FromRows had to infer (quest points the table
// assumes, rows it could not express); the file is returned even when it does not validate under rules, so the caller
// can show the problems.
func ExportPlan(name string, rules Rules) (*Build, []string, error) {
	raw, err := plandata.FS.ReadFile("build-" + name + ".json")
	if err != nil {
		return nil, nil, fmt.Errorf("buildfile: no shipped plan table %q", name)
	}
	lb, err := leveler.ParseBuild(raw)
	if err != nil {
		return nil, nil, fmt.Errorf("buildfile: plan table %q: %w", name, err)
	}
	b, notes := FromRows(lb.Rows, lb.Class, rules)
	if b.Name == "" {
		b.Name = name
	}
	b.Source = "plandata:build-" + name + ".json"
	return b, notes, nil
}

// planFile is the plan table JSON the leveler's ParseBuild reads (its rawBuild), written by ImportPlan.
type planFile struct {
	Build  string      `json:"build"`
	Class  string      `json:"class"`
	Source string      `json:"source,omitempty"`
	Rows   []planRow   `json:"rows"`
	Respec *planRespec `json:"respec,omitempty"`
}

type planRow struct {
	Clvl      int         `json:"clvl"`
	Strength  int         `json:"strength,omitempty"`
	Dexterity int         `json:"dexterity,omitempty"`
	Vitality  int         `json:"vitality,omitempty"`
	Energy    int         `json:"energy,omitempty"`
	Skills    []planSkill `json:"skills,omitempty"`
}

type planRespec struct {
	AtClvl int         `json:"at_clvl"`
	Pool   []planSkill `json:"respend_pool"`
}

type planSkill struct {
	Skill  string `json:"skill"`
	Points int    `json:"points"`
	Quest  string `json:"quest,omitempty"`
}

// ImportPlan validates a build file under rules, expands it and writes the plan table JSON the leveler reads
// (build-*.json shape, no combat line or hotkeys: those stay engine data). Problems are returned as an error naming
// them all; a file that does not validate is never expanded.
func ImportPlan(b *Build, rules Rules) ([]byte, error) {
	if ps := Validate(b, rules); len(ps) > 0 {
		var m []string
		for _, p := range ps {
			m = append(m, p.String())
		}
		return nil, fmt.Errorf("buildfile: %s does not validate: %s", b.Name, strings.Join(m, "; "))
	}
	e, err := Expand(b, rules)
	if err != nil {
		return nil, err
	}
	lb := e.ToLeveler(b)
	var pf planFile
	pf.Build, pf.Class, pf.Source = b.Name, b.Class, "didle.build/1"
	for _, r := range lb.Rows {
		var row planRow
		row.Clvl, row.Strength, row.Dexterity, row.Vitality, row.Energy = r.Clvl, r.Attr[leveler.Str], r.Attr[leveler.Dex], r.Attr[leveler.Vit], r.Attr[leveler.Ene]
		for _, s := range r.Skills {
			row.Skills = append(row.Skills, planSkill{s.Skill, s.Points, s.Quest})
		}
		pf.Rows = append(pf.Rows, row)
	}
	if lb.Respec > 0 {
		pf.Respec = &planRespec{AtClvl: lb.Respec}
		for _, s := range lb.RespecPool {
			pf.Respec.Pool = append(pf.Respec.Pool, planSkill{Skill: s.Skill, Points: s.Points})
		}
	}
	return json.MarshalIndent(pf, "", " ")
}

// LoadFile reads, strictly parses and validates a build file from disk. A parse error or any validation problem is an
// error; the Build is returned with it so the caller can still show the file.
func LoadFile(path string, rules Rules) (*Build, error) {
	raw, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}
	b, err := Parse(raw)
	if err != nil {
		return nil, fmt.Errorf("buildfile: %s: %w", path, err)
	}
	if ps := Validate(b, rules); len(ps) > 0 {
		var m []string
		for _, p := range ps {
			m = append(m, p.String())
		}
		return b, fmt.Errorf("buildfile: %s: %s", path, strings.Join(m, "; "))
	}
	return b, nil
}

// SaveFile writes the canonical JSON of a validated build file.
func SaveFile(path string, b *Build, rules Rules) error {
	if ps := Validate(b, rules); len(ps) > 0 {
		return fmt.Errorf("buildfile: %s does not validate: %s", b.Name, ps[0].String())
	}
	raw, err := Marshal(b)
	if err != nil {
		return err
	}
	return os.WriteFile(path, append(raw, '\n'), 0o644)
}
