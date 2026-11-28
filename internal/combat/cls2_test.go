package combat

import (
	"context"
	"encoding/json"
	"strings"
	"testing"

	capdata "d2lod-bot/data/capabilities"
	"d2lod-bot/internal/d2"
	"d2lod-bot/internal/plandata"
)

type tableKeys struct {
	Build      string `json:"build"`
	Class      string `json:"class"`
	Combat     string `json:"combat"`
	MainAttack []struct {
		Skill    string `json:"skill"`
		FromClvl int    `json:"from_clvl"`
	} `json:"main_attack"`
}

// tablesOf reads the class, combat and main_attack keys of every shipped plan table of a class.
func tablesOf(t *testing.T, class string) []tableKeys {
	t.Helper()
	ents, err := plandata.FS.ReadDir(".")
	if err != nil {
		t.Fatal(err)
	}
	var out []tableKeys
	for _, e := range ents {
		if !strings.HasPrefix(e.Name(), "build-") || e.Name() == plandata.CatalogFile {
			continue
		}
		raw, _ := plandata.FS.ReadFile(e.Name())
		var k tableKeys
		if err := json.Unmarshal(raw, &k); err != nil {
			t.Fatalf("%s: %v", e.Name(), err)
		}
		if strings.EqualFold(k.Class, class) {
			out = append(out, k)
		}
	}
	if len(out) < 2 {
		t.Fatalf("%s: %d plan tables, the check needs two builds", class, len(out))
	}
	return out
}

// Card cls2: the combat key of a table, not the class, picks the profile. The Amazon's bow, multishot and javelin
// tables land on three different profiles, and every main attack a table names is an attack of a band of the profile
// it lands on (a javelin table on the bow profile would shoot arrows it does not have). The Assassin's two tables
// share one profile whose trap ladder holds every sentry they name. Two builds per class, all read from the data.
func TestCombatKeySelectsTheProfilePerTable(t *testing.T) {
	// The generic attacks of the capability table are the weapon's basic attack: no band entry, no key.
	var caps struct {
		Generic map[string]json.RawMessage `json:"generic"`
	}
	if err := json.Unmarshal(capdata.JSON, &caps); err != nil {
		t.Fatal(err)
	}
	classOf := map[string]d2.Class{"amazon": d2.Amazon, "assassin": d2.Assassin}
	for _, class := range []string{"amazon", "assassin"} {
		byProfile := map[string][]string{}
		for _, tb := range tablesOf(t, class) {
			if tb.Combat == "" {
				t.Errorf("%s: no combat key", tb.Build)
				continue
			}
			p, err := LookupProfile(classOf[class], Build(tb.Combat))
			if err != nil {
				t.Errorf("%s: %v", tb.Build, err)
				continue
			}
			byProfile[p.ProfileID()] = append(byProfile[p.ProfileID()], tb.Build)
			inBand := map[string]bool{}
			for _, b := range p.Bands {
				for _, a := range b.Attack {
					inBand[a] = true
				}
			}
			for _, tr := range p.Traps {
				inBand[tr] = true
			}
			for _, m := range tb.MainAttack {
				if _, generic := caps.Generic[m.Skill]; !generic && !inBand[m.Skill] {
					t.Errorf("%s: main attack %s (from clvl %d) is in no band and no trap ladder of profile %s", tb.Build, m.Skill, m.FromClvl, p.ProfileID())
				}
			}
		}
		switch class {
		case "amazon":
			if len(byProfile) < 3 {
				t.Errorf("amazon tables share profiles: %v", byProfile)
			}
		case "assassin":
			if len(byProfile) != 1 {
				t.Errorf("assassin tables split over profiles: %v", byProfile)
			}
		}
	}
	if _, err := LookupProfile(d2.Amazon, "crossbow"); err == nil {
		t.Error("an unknown Amazon combat key resolved to a profile")
	}
}

// Card cls2: Death Sentry and the Necromancer's corpse spells read one reservation (aimcorpse.go). A corpse another
// corpse spell took is not the trap's corpse, and the corpse the trap is laid on is taken for the next spell.
func TestDeathSentrySharesTheCorpseReservation(t *testing.T) {
	h, w := trapWorld(t, map[int]int{idDeathSent: 10})
	w.snap.Monsters = []d2.Unit{mon(2, 1, 112, 100), mon(3, 1, 113, 101)}
	w.corpse(20, 110, 100)
	ctx := context.Background()

	h.f.takenCorpses(w.snap.Player.Area) // the reservation is per area; the first read binds it to this one
	h.f.takeCorpse(20)
	if acted, _ := h.f.assassinTick(ctx, w.snap); acted {
		t.Fatalf("Death Sentry on a reserved corpse: %v", w.castAt)
	}
	w.corpse(21, 111, 101)
	acted, err := h.f.assassinTick(ctx, w.snap)
	if !acted || err != nil || len(w.castAt) != 1 || w.castAt[0].X != 111 {
		t.Fatalf("Death Sentry on the free corpse: %v %v %v", acted, err, w.castAt)
	}
	if !h.f.takenCorpses(w.snap.Player.Area)[21] {
		t.Error("the corpse under the new Death Sentry is not reserved")
	}
}
