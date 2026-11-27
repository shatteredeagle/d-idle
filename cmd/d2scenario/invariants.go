package main

import (
	"bytes"
	"embed"
	"encoding/json"
	"fmt"
	"sort"
	"strings"

	"d2lod-bot/internal/scenario"
)

//go:embed invariants.json
var invariantsJSON []byte

// invariantFragments are per-feature additions (invariants.d/<feature>.json, same shape as invariants.json, no _doc):
// a new invariant row goes into its own file so two branches never edit the same list line.
//
//go:embed invariants.d
var invariantFragments embed.FS

type invariantPolicy struct {
	Hard       []string            `json:"hard"`
	Soft       []string            `json:"soft"`
	ReportOnly []string            `json:"report_only"`
	SoftIn     map[string][]string `json:"soft_in"`
}

func loadInvariantPolicy() invariantPolicy {
	var p invariantPolicy
	if err := json.Unmarshal(invariantsJSON, &p); err != nil {
		panic("invariants.json: " + err.Error())
	}
	ents, _ := invariantFragments.ReadDir("invariants.d")
	for _, e := range ents { // ReadDir is sorted: a stable merge order
		if !strings.HasSuffix(e.Name(), ".json") {
			continue
		}
		raw, _ := invariantFragments.ReadFile("invariants.d/" + e.Name())
		var f invariantPolicy
		if err := json.Unmarshal(raw, &f); err != nil {
			panic("invariants.d/" + e.Name() + ": " + err.Error())
		}
		p.Hard = append(p.Hard, f.Hard...)
		p.Soft = append(p.Soft, f.Soft...)
		p.ReportOnly = append(p.ReportOnly, f.ReportOnly...)
		for k, v := range f.SoftIn {
			if p.SoftIn == nil {
				p.SoftIn = map[string][]string{}
			}
			p.SoftIn[k] = append(p.SoftIn[k], v...)
		}
	}
	return p
}

// invariantCounts counts the `invariant <name>` WARN lines of the bot log (the monitor's one line per hit, invariants.go in internal/bot).
func invariantCounts(log []byte) map[string]int {
	out := map[string]int{}
	for _, l := range bytes.Split(log, []byte("\n")) {
		if len(l) == 0 || l[0] != '{' || !bytes.Contains(l, []byte(`"invariant `)) {
			continue
		}
		var m struct {
			Msg string `json:"msg"`
		}
		if json.Unmarshal(l, &m) == nil && strings.HasPrefix(m.Msg, "invariant ") {
			out[strings.TrimPrefix(m.Msg, "invariant ")]++
		}
	}
	return out
}

// invariantAsserts turns the hits of a run into assert results: a hard row not in a report-only scenario fails the run.
func (p invariantPolicy) invariantAsserts(id string, hits map[string]int) []scenario.AssertResult {
	hard := map[string]bool{}
	for _, n := range p.Hard {
		hard[n] = true
	}
	reportOnly := false
	for _, pre := range p.ReportOnly {
		if strings.HasPrefix(id, pre) {
			reportOnly = true
		}
	}
	var names []string
	for n := range hits {
		names = append(names, n)
	}
	sort.Strings(names)
	var out []scenario.AssertResult
	for _, n := range names {
		if !hard[n] || p.softFor(n, id) {
			continue
		}
		r := scenario.AssertResult{Kind: "invariant " + n, OK: reportOnly, Want: "0 hits", Got: fmt.Sprintf("%d hits", hits[n])}
		if reportOnly {
			r.Why = "report only"
		}
		out = append(out, r)
	}
	return out
}

// softFor says a hard row is only logged for this scenario id (invariants.json soft_in).
func (p invariantPolicy) softFor(name, id string) bool {
	for _, pre := range p.SoftIn[name] {
		if strings.HasPrefix(id, pre) {
			return true
		}
	}
	return false
}
