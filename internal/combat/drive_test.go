package combat

import (
	"slices"
	"testing"

	"d2lod-bot/internal/d2"
)

// Fighter.Drive (agency.planner 2): nil or empty changes nothing, a named attack the hero can use goes first, one it
// cannot use (or that does not exist) leaves the profile's list as it is.
func TestDriveAttackReordersOnlyUsableSkills(t *testing.T) {
	h, g := newHandGame(t)
	g.p.Level = 12
	g.p.Skills = []d2.Skill{{ID: 0, Level: 1}, {ID: 98, Level: 8}, {ID: 107, Level: 1}, {ID: 106, Level: 1}, {ID: 112, Level: 1}}
	base := attackNamesOf(h.f, g.p)
	if len(base) < 2 || base[0] != "Zeal" {
		t.Fatalf("profile order: %v", base)
	}
	h.f.Drive = nil
	if got := attackNamesOf(h.f, g.p); !slices.Equal(got, base) {
		t.Fatalf("nil Drive changed the list: %v vs %v", got, base)
	}
	for _, d := range []func() (string, string){func() (string, string) { return "", "" }, func() (string, string) { return "Nonexistent", "" }} {
		h.f.Drive = d
		if got := attackNamesOf(h.f, g.p); !slices.Equal(got, base) {
			t.Fatalf("empty or unknown drive changed the list: %v vs %v", got, base)
		}
	}
	other := base[1]
	h.f.Drive = func() (string, string) { return other, "" }
	got := attackNamesOf(h.f, g.p)
	if got[0] != other || len(got) != len(base) {
		t.Fatalf("drive %q: %v (profile %v)", other, got, base)
	}
}

// The driven aura goes first as a plain entry and the profile's ladder stays behind it.
func TestDriveAuraPrepends(t *testing.T) {
	h := &Fighter{}
	ladder := []AuraPref{{Name: "Might"}}
	if got := h.driveAura(ladder); len(got) != 1 {
		t.Fatalf("no Drive: %v", got)
	}
	h.Drive = func() (string, string) { return "", "Concentration" }
	got := h.driveAura(ladder)
	if len(got) != 2 || got[0].Name != "Concentration" || got[1].Name != "Might" {
		t.Fatalf("got %v", got)
	}
	h.Drive = func() (string, string) { return "", "" }
	if got := h.driveAura(ladder); len(got) != 1 {
		t.Fatalf("empty aura: %v", got)
	}
}
