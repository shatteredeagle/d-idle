package decide

import (
	"strings"
	"testing"
)

func TestChampModsTable(t *testing.T) {
	rows, err := ChampMods()
	if err != nil || len(rows) == 0 {
		t.Fatalf("%d rows, %v", len(rows), err)
	}
	for _, r := range rows {
		if r.Tag != "S" && r.Tag != "U" {
			t.Fatalf("id %d: tag %q must be S or U", r.ID, r.Tag)
		}
		if r.ID < 1 || r.ID > 40 {
			t.Fatalf("id %d outside the monumod range the reader accepts", r.ID)
		}
	}
	if ModName(5) != "Extra Strong" || ModName(7) != "Cursed" {
		t.Fatal("known ids")
	}
	if got := ModName(99); got != "unknown mod #99" {
		t.Fatalf("unknown id: %q", got)
	}
}

// Every mod the Codex champion row names has an id in the table, spelled the same (the Codex and the panel must agree).
func TestChampModsCoverCodexRow(t *testing.T) {
	rows, err := ChampMods()
	if err != nil {
		t.Fatal(err)
	}
	have := map[string]bool{}
	for _, r := range rows {
		have[strings.ToLower(r.Name)] = true
	}
	var notable First
	fs, err := ShippedFirsts()
	if err != nil {
		t.Fatal(err)
	}
	for _, f := range fs.Firsts {
		if f.ID == "champion_notable" {
			notable = f
		}
	}
	if len(notable.Match) == 0 {
		t.Fatal("champion_notable row missing")
	}
	for _, m := range notable.Match {
		if !have[strings.ToLower(m)] {
			t.Errorf("Codex mod %q has no id in champmods.json", m)
		}
	}
}
