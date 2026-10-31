package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"os/exec"
	"path/filepath"
	"regexp"
	"strings"
	"time"
)

// The JSON shape is read by tools/agentstatus/changes.html; keep the two in step.
type jsonDoc struct {
	Generated  string         `json:"generated"`
	Commit     string         `json:"commit"`
	Counts     map[string]int `json:"counts"`
	Active     []jsonActive   `json:"active"`
	Archived   []jsonDone     `json:"archived"`
	Collisions []string       `json:"collisions"`
	Specs      []jsonSpec     `json:"specs"`
	Gates      []jsonGate     `json:"gates"`
	Merges     []jsonMerge    `json:"merges"`
	Warnings   []Finding      `json:"warnings"`
	NeedsYou   []jsonNeed     `json:"needs_you"`
}

type jsonActive struct {
	ID         string   `json:"id"`
	Status     string   `json:"status"`
	Agent      string   `json:"agent"`
	Branch     string   `json:"branch"`
	Worktree   string   `json:"worktree"`
	Why        string   `json:"why"`
	Owns       []string `json:"owns"`
	Exclusive  []string `json:"exclusive"`
	Gates      []string `json:"gates"`
	TasksDone  int      `json:"tasks_done"`
	TasksTotal int      `json:"tasks_total"`
	Path       string   `json:"path"`
}

type jsonDone struct {
	ID     string `json:"id"`
	Date   string `json:"date"`
	Merge  string `json:"merge"`
	Result string `json:"result"`
	Path   string `json:"path"`
}

type jsonSpec struct {
	Name         string `json:"name"`
	Purpose      string `json:"purpose"`
	Requirements int    `json:"requirements"`
	Path         string `json:"path"`
}

// jsonGate is one acceptance number: a card gate or a requirement of the gate specs. Value stays "-"
// until the board has live data for it.
type jsonGate struct {
	Gate      string `json:"gate"`
	Threshold string `json:"threshold"`
	Source    string `json:"source"`
	Value     string `json:"value"`
}

type jsonMerge struct {
	Hash    string `json:"hash"`
	Date    string `json:"date"`
	Subject string `json:"subject"`
	Change  string `json:"change"`
}

var (
	taskRe     = regexp.MustCompile(`(?m)^\s*- \[([ xX])\] `)
	gateSpecs  = []string{"soak-gates", "fidelity-gate"}
	sourceRe   = regexp.MustCompile(`\s*Source:\s*(.+)$`)
	sentenceRe = regexp.MustCompile(`^(.+?[.;])(\s|$)`)
)

func gitHead(root string) string {
	out, err := exec.Command("git", "-C", root, "rev-parse", "--short", "HEAD").Output()
	if err != nil {
		return ""
	}
	return strings.TrimSpace(string(out))
}

// gitMergesISO is gitMerges with an ISO date (the STATUS list drops the year).
var gitMergesISO = func(root string, n int) ([]Merge, error) {
	out, err := exec.Command("git", "-C", root, "log", "--merges", fmt.Sprintf("-n%d", n),
		"--format=%h%x1f%aI%x1f%s%x1f%b%x1e").Output()
	if err != nil {
		return nil, err
	}
	var ms []Merge
	for _, rec := range strings.Split(string(out), "\x1e") {
		if f := strings.SplitN(strings.TrimLeft(rec, "\n"), "\x1f", 4); len(f) == 4 {
			ms = append(ms, Merge{f[0], f[1], f[2], f[3]})
		}
	}
	return ms, nil
}

func renderJSON(root string, cards []*Card, merges []Merge, fs []Finding, now time.Time) ([]byte, error) {
	d := jsonDoc{Generated: now.UTC().Format(time.RFC3339), Commit: gitHead(root),
		Active: []jsonActive{}, Archived: []jsonDone{}, Collisions: []string{}, Specs: []jsonSpec{},
		Gates: []jsonGate{}, Merges: []jsonMerge{}, Warnings: []Finding{}, NeedsYou: loadNeeds(root)}
	rel := func(p string) string { r, _ := filepath.Rel(root, p); return filepath.ToSlash(r) }
	for _, c := range cards {
		switch {
		case c.Active():
			tasks, _ := readFile(filepath.Join(c.Dir, "tasks.md"))
			all := taskRe.FindAllStringSubmatch(tasks, -1)
			done := 0
			for _, m := range all {
				if m[1] != " " {
					done++
				}
			}
			d.Active = append(d.Active, jsonActive{c.ID, c.Status(), c.Agent(), c.Branch(), c.one("worktree"), c.Why,
				nz(c.Owns()), nz(c.Exclusive()), nz(c.Gates()), done, len(all), rel(c.Dir)})
			for _, g := range c.Gates() {
				d.Gates = append(d.Gates, jsonGate{g, g, "change " + c.ID, "-"})
			}
		case c.Archived:
			date := ""
			if m := archiveRe.FindStringSubmatch(filepath.Base(c.Dir)); m != nil {
				date = filepath.Base(c.Dir)[:10]
			}
			d.Archived = append(d.Archived, jsonDone{c.ID, date, c.one("merge"), c.one("result"), rel(c.Dir)})
		}
	}
	// newest archive first
	for i, j := 0, len(d.Archived)-1; i < j; i, j = i+1, j-1 {
		d.Archived[i], d.Archived[j] = d.Archived[j], d.Archived[i]
	}
	names, counts := specSummaries(root)
	for _, n := range names {
		p := "openspec/specs/" + n + "/spec.md"
		md, _ := readFile(filepath.Join(root, filepath.FromSlash(p)))
		purpose, _ := section(md, "Purpose")
		d.Specs = append(d.Specs, jsonSpec{n, purpose, counts[n], p})
		if contains(gateSpecs, n) {
			bl, _ := parseRequirements(md)
			for _, b := range bl {
				d.Gates = append(d.Gates, specGate(n, b))
			}
		}
	}
	for _, m := range merges {
		d.Merges = append(d.Merges, jsonMerge{m.Hash, m.Date, m.Subject, changeID(m)})
	}
	for _, f := range fs {
		if f.Check == "overlap" {
			d.Collisions = append(d.Collisions, f.Msg)
		} else {
			d.Warnings = append(d.Warnings, f)
		}
	}
	d.Counts = map[string]int{"active": len(d.Active), "archived": len(d.Archived), "specs": len(d.Specs),
		"collisions": len(d.Collisions), "warnings": len(d.Warnings), "needs_you": len(d.NeedsYou)}
	var buf bytes.Buffer
	enc := json.NewEncoder(&buf)
	enc.SetEscapeHTML(false)
	enc.SetIndent("", " ")
	err := enc.Encode(d)
	return buf.Bytes(), err
}

// specGate turns a requirement into a table row: first sentence is the threshold, the Source: tail the source.
func specGate(spec string, b *reqBlock) jsonGate {
	text := strings.Join(strings.Fields(b.text), " ")
	src := "spec " + spec
	if m := sourceRe.FindStringSubmatch(text); m != nil {
		src += ": " + m[1]
		text = strings.TrimSpace(text[:len(text)-len(m[0])])
	}
	if m := sentenceRe.FindStringSubmatch(text); m != nil && len(m[1]) < len(text) && len(text) > 300 {
		text = m[1]
	}
	return jsonGate{b.name, text, src, "-"}
}

// nz makes nil slices print as [] so the page never sees null.
func nz(v []string) []string {
	if v == nil {
		return []string{}
	}
	return v
}
