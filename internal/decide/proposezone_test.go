package decide

import (
	"errors"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

// testRanges is the tally zone table's relevant rows (internal/currency/tallyrows.json), copied so the package stays
// free of an import; the watch tests use the real table.
func testRanges(area string) (int, int, bool) {
	m := map[string][2]int{"Cold Plains": {6, 12}, "Stony Field": {10, 16}, "Burial Grounds": {8, 14}, "Dark Wood": {10, 16}}
	r, ok := m[area]
	return r[0], r[1], ok
}

func newTestProposer(t *testing.T, path string) (*Proposer, *ManualClock) {
	t.Helper()
	tbl, err := ShippedProposeTable()
	if err != nil {
		t.Fatal(err)
	}
	clk := &ManualClock{}
	p, err := NewProposer(tbl, testRanges, clk, path)
	if err != nil {
		t.Fatal(err)
	}
	return p, clk
}

func TestProposerRule(t *testing.T) {
	cases := []struct {
		name string
		in   ProposeInput
		want string // "" = no proposal
	}{
		{"paladin at the minimum", ProposeInput{Class: "paladin", Clvl: 6, Area: "Blood Moor", Kit: "OK", Potions: 4}, "Cold Plains"},
		{"sorceress one level short of her lead", ProposeInput{Class: "sorceress", Clvl: 6, Area: "Blood Moor", Kit: "OK", Potions: 4}, ""},
		{"sorceress with her lead", ProposeInput{Class: "sorceress", Clvl: 7, Area: "Blood Moor", Kit: "OK", Potions: 4}, "Cold Plains"},
		{"thin kit", ProposeInput{Class: "paladin", Clvl: 9, Area: "Blood Moor", Kit: "THIN", Potions: 4}, ""},
		{"too few potions", ProposeInput{Class: "paladin", Clvl: 9, Area: "Blood Moor", Kit: "OK", Potions: 1}, ""},
		{"over the range of the zone", ProposeInput{Class: "paladin", Clvl: 13, Area: "Blood Moor", Kit: "OK", Potions: 4}, ""},
		{"first candidate wins", ProposeInput{Class: "paladin", Clvl: 10, Area: "Cold Plains", Kit: "OK", Potions: 4}, "Stony Field"},
		{"a zone with no next zone", ProposeInput{Class: "paladin", Clvl: 20, Area: "Forgotten Tower", Kit: "OK", Potions: 9}, ""},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			p, _ := newTestProposer(t, "")
			pr, err := p.Look(c.in)
			if err != nil {
				t.Fatal(err)
			}
			got := ""
			if pr != nil {
				got = pr.To
			}
			if got != c.want {
				t.Fatalf("proposal %q, want %q", got, c.want)
			}
			if pr != nil && (!strings.Contains(pr.Text, "ready for "+c.want) || len(pr.Why) != 3 || len(pr.Options) != 3) {
				t.Errorf("text or reasons off: %+v", pr)
			}
		})
	}
}

func TestProposerUnknownClassIsAnError(t *testing.T) {
	p, _ := newTestProposer(t, "")
	_, err := p.Look(ProposeInput{Class: "necromancer", Clvl: 9, Area: "Blood Moor", Kit: "OK", Potions: 4})
	if !errors.Is(err, ErrProposal) || !strings.Contains(err.Error(), "necromancer") {
		t.Fatalf("err = %v", err)
	}
}

// One pending proposal is raised once however often the hero looks; nothing places an order and nothing fades.
func TestProposerWaitsWithoutFading(t *testing.T) {
	p, clk := newTestProposer(t, "")
	in := ProposeInput{Class: "paladin", Clvl: 8, Area: "Blood Moor", Kit: "OK", Potions: 4, Away: true}
	first, _ := p.Look(in)
	for i := 0; i < 50; i++ {
		clk.Advance(10 * time.Hour)
		again, err := p.Look(in)
		if err != nil || again == nil || again.ID != first.ID || again.State != ProposalPending {
			t.Fatalf("look %d: %+v %v", i, again, err)
		}
	}
	if n := len(p.Pending()); n != 1 {
		t.Fatalf("%d pending", n)
	}
	if !first.Away {
		t.Error("raised while away, the flag must say so")
	}
}

func TestProposerStayCooldownQuietAndReload(t *testing.T) {
	path := filepath.Join(t.TempDir(), "proposals.json")
	p, clk := newTestProposer(t, path)
	in := ProposeInput{Class: "paladin", Clvl: 8, Area: "Blood Moor", Kit: "OK", Potions: 4}
	for round := 1; round <= DefaultQuietAfter; round++ {
		pr, _ := p.Look(in)
		if pr == nil {
			t.Fatalf("round %d: no proposal", round)
		}
		if _, err := p.Answer(pr.ID, ProposeStay); err != nil {
			t.Fatal(err)
		}
		// within the cooldown nothing is asked again, also after a restart
		p2, err := NewProposer(p.tbl, testRanges, clk, path)
		if err != nil {
			t.Fatal(err)
		}
		if again, _ := p2.Look(in); again != nil {
			t.Fatalf("round %d: re-asked inside the cooldown after reload: %+v", round, again)
		}
		clk.Advance(DefaultRepeatCooldown + time.Minute)
	}
	reloaded, _ := NewProposer(p.tbl, testRanges, clk, path)
	if again, _ := reloaded.Look(in); again != nil {
		t.Fatalf("quiet after %d stays must persist across reload, got %+v", DefaultQuietAfter, again)
	}
}

func TestProposerLaterDoesNotGoQuietAndGoResets(t *testing.T) {
	p, clk := newTestProposer(t, "")
	in := ProposeInput{Class: "paladin", Clvl: 8, Area: "Blood Moor", Kit: "OK", Potions: 4}
	for i := 0; i < DefaultQuietAfter+2; i++ {
		pr, _ := p.Look(in)
		if pr == nil {
			t.Fatalf("later %d: quiet too early", i)
		}
		if _, err := p.Answer(pr.ID, ProposeLater); err != nil {
			t.Fatal(err)
		}
		clk.Advance(DefaultRepeatCooldown + time.Minute)
	}
	pr, _ := p.Look(in)
	if _, err := p.Answer(pr.ID, "now"); err == nil {
		t.Error("a bad answer must be refused")
	}
	if got, ok := p.Get(pr.ID); !ok || got.State != ProposalPending {
		t.Errorf("a refused answer must leave it pending: %+v", got)
	}
	if _, err := p.Answer(pr.ID, ProposeGo); err != nil {
		t.Fatal(err)
	}
	if _, err := p.Answer(pr.ID, ProposeStay); err == nil {
		t.Error("an answered proposal cannot be answered again")
	}
}

// When the player's own order moves the hero, the open question is stale and closes without a count.
func TestProposerClosesWhenTheHeroLeft(t *testing.T) {
	p, _ := newTestProposer(t, "")
	pr, _ := p.Look(ProposeInput{Class: "paladin", Clvl: 8, Area: "Blood Moor", Kit: "OK", Potions: 4})
	if pr == nil {
		t.Fatal("no proposal")
	}
	if got, _ := p.Look(ProposeInput{Class: "paladin", Clvl: 8, Area: "Cold Plains", Kit: "OK", Potions: 4}); got != nil && got.ID == pr.ID {
		t.Fatal("the stale proposal must close")
	}
	if len(p.Pending()) != 0 && p.Pending()[0].From == "Blood Moor" {
		t.Fatal("still pending")
	}
}

func TestParseProposeTableErrors(t *testing.T) {
	for name, src := range map[string]string{
		"schema":  `{"schema":"x","zones":[],"classes":{"a":{}}}`,
		"nonext":  `{"schema":"decide.proposezone/1","zones":[{"area":"A","next":[]}],"classes":{"a":{}}}`,
		"unknown": `{"schema":"decide.proposezone/1","zones":[],"classes":{"a":{}},"bogus":1}`,
	} {
		if _, err := ParseProposeTable([]byte(src)); !errors.Is(err, ErrProposal) {
			t.Errorf("%s: err = %v", name, err)
		}
	}
}
