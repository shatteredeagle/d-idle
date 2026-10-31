package main

import (
	"encoding/json"
	"strings"
	"testing"
	"time"
)

func TestRenderJSON(t *testing.T) {
	root := t.TempDir()
	addCard(t, root, "alpha", "in-progress", "\n  - internal/town/**", "\n  - d2d-control-socket")
	addCard(t, root, "beta", "in-progress", "\n  - internal/town/x.go", "\n  - d2d-control-socket")
	write(t, root, "openspec/changes/alpha/tasks.md", strings.Replace(card("alpha", "in-progress", "\n  - internal/town/**", "\n  - d2d-control-socket"),
		"- [ ] 1.1 x", "- [x] 1.1 x\n- [ ] 1.2 y\n- [X] 1.3 z", 1))
	write(t, root, "openspec/changes/archive/2026-10-29-old/tasks.md", strings.Replace(card("old", "archived", "", " []"), "merge:\nresult:", "merge: abc123\nresult: \"merged, verified live\"", 1))
	write(t, root, "openspec/changes/archive/2026-10-29-old/proposal.md", why)
	write(t, root, "openspec/specs/soak-gates/spec.md", "# Soak\n\n## Purpose\nNumeric pass criteria for the soaks, so a run is judged by numbers and not by opinion.\n\n## Requirements\n\n### Requirement: G1 belt\nA G1 run SHALL have no window longer than 5 minutes with zero potions. Source: docs/review-strategy.md.\n\n#### Scenario: Empty\n- **WHEN** x\n- **THEN** y\n")
	cards, fs := loadCards(root)
	fs = append(fs, checkOverlap(cards)...)
	merges := []Merge{{"h1", "2026-10-30T09:00:00-04:00", "Merge agent/alpha: done", ""}}
	b, err := renderJSON(root, cards, merges, fs, time.Date(2026, 10, 30, 12, 0, 0, 0, time.UTC))
	if err != nil {
		t.Fatal(err)
	}
	var d jsonDoc
	if err := json.Unmarshal(b, &d); err != nil {
		t.Fatal(err)
	}
	if d.Generated != "2026-10-30T12:00:00Z" || len(d.Active) != 2 || len(d.Archived) != 1 {
		t.Fatalf("counts: %s", b)
	}
	a := d.Active[0]
	if a.ID != "alpha" || a.TasksDone != 2 || a.TasksTotal != 3 || a.Owns[0] != "internal/town/**" {
		t.Errorf("alpha: %+v", a)
	}
	if o := d.Archived[0]; o.ID != "old" || o.Date != "2026-10-29" || o.Merge != "abc123" || o.Result != "merged, verified live" {
		t.Errorf("archived: %+v", o)
	}
	if len(d.Collisions) != 2 || d.Counts["collisions"] != 2 || d.Counts["warnings"] != 0 {
		t.Errorf("collisions: %v %v", d.Collisions, d.Counts)
	}
	if len(d.Merges) != 1 || d.Merges[0].Change != "alpha" {
		t.Errorf("merges: %+v", d.Merges)
	}
	if len(d.Specs) != 1 || d.Specs[0].Requirements != 1 || d.Specs[0].Path != "openspec/specs/soak-gates/spec.md" {
		t.Errorf("specs: %+v", d.Specs)
	}
	var gate *jsonGate
	for i, g := range d.Gates {
		if g.Gate == "G1 belt" {
			gate = &d.Gates[i]
		}
	}
	if gate == nil || gate.Source != "spec soak-gates: docs/review-strategy.md." || strings.Contains(gate.Threshold, "Source") || gate.Value != "-" {
		t.Errorf("spec gate: %+v", gate)
	}
	if strings.Contains(string(b), "null") {
		t.Error("null in output, the page expects arrays")
	}
}

func TestParseNeeds(t *testing.T) {
	md := "## Open\n\n- [ ] a | now | do it | yes | since 2026-10-30 | docs/x.md\n- [ ] b | bogus | ask | rec | since 2026-10-29\n- [ ] bad | now | short\n- [x] c | done 2026-10-30 | n\n"
	n := parseNeeds(md)
	if len(n) != 2 || n[0].ID != "a" || n[0].Priority != "now" || n[0].Since != "2026-10-30" || n[0].Link != "docs/x.md" {
		t.Fatalf("%+v", n)
	}
	if n[1].Priority != "whenever" || n[1].Link != "" {
		t.Errorf("%+v", n[1])
	}
	if got := loadNeeds(t.TempDir()); got == nil || len(got) != 0 {
		t.Errorf("missing file must give an empty slice: %v", got)
	}
}
