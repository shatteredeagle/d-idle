package pact

import (
	"encoding/json"
	"os"
	"strings"
	"testing"
	"time"

	"d2lod-bot/internal/unlock"
)

func lockedOf(c *Controller, id string) bool {
	for _, f := range c.View(nil).Families {
		for _, p := range f.Pacts {
			if p.ID == id {
				return p.Locked
			}
		}
	}
	panic("no pact " + id)
}

// The Pacts follow the shared ladder: Andariel opens the first ones, Nightmare waits for Baal in Normal, a counted
// or undecided trigger (baal_nightmare) never opens, and the silhouette reads the ladder.
func TestLadderOpensPacts(t *testing.T) {
	for _, class := range []string{"sorceress", "paladin"} {
		r := newRig(t, class, "")
		if !lockedOf(r.c, "blood_ravens_tithe") || !lockedOf(r.c, "pact_of_nightmares") {
			t.Fatalf("%s: open before any kill", class)
		}
		r.kill("Baal", "normal") // Baal before Andariel opens only its own
		if lockedOf(r.c, "pact_of_nightmares") || !lockedOf(r.c, "blood_ravens_tithe") {
			t.Fatalf("%s: Baal normal should open Nightmares only", class)
		}
		r.kill("Andariel", "nightmare")
		if lockedOf(r.c, "blood_ravens_tithe") {
			t.Fatalf("%s: an Andariel kill opens the first Pacts", class)
		}
	}
	r := newRig(t, "paladin", "")
	r.kill("Andariel", "normal")
	r.kill("Baal", "nightmare") // not a Normal kill
	if !lockedOf(r.c, "pact_of_nightmares") {
		t.Fatal("Baal in Nightmare opened Nightmares")
	}
	if err := r.c.Choose([]Pick{{"pact_of_nightmares", 1}}); err == nil || !strings.Contains(err.Error(), "locked") {
		t.Fatalf("chose a locked Pact: %v", err)
	}
}

func TestNoGateMeansLocked(t *testing.T) {
	r := newRig(t, "sorceress", "")
	r.c.cfg.Gate = nil
	if !lockedOf(r.c, "blood_ravens_tithe") {
		t.Fatal("no ladder, yet a Pact is open")
	}
}

func TestLadderReplayIdempotentAndPersistent(t *testing.T) {
	dir := t.TempDir()
	lad, err := unlock.Shipped()
	if err != nil {
		t.Fatal(err)
	}
	e, _ := unlock.Open(dir, "p", lad, nil)
	ev := unlock.Event{Type: unlock.EvBossKill, Name: "Andariel", Sub: "normal"}
	first, _ := e.Observe(ev)
	again, _ := e.Observe(ev)
	if len(first) != 1 || first[0].ID != "pact_andariel_first_kill" || len(again) != 0 {
		t.Fatalf("first %v again %v", first, again)
	}
	e2, _ := unlock.Open(dir, "p", lad, nil) // a restart
	if !e2.Allowed("pact_andariel_first_kill") || e2.Allowed("pact_baal_normal") {
		t.Fatal("persistence")
	}
	if again, _ = e2.Observe(ev); len(again) != 0 {
		t.Fatal("a replayed kill revealed again")
	}
}

func TestMigrationAndUnknownKeys(t *testing.T) {
	path := t.TempDir() + "/p.json"
	old := `{"picks":[],"consents":[],"masteries":0,"unlocked":["andariel_first_kill","baal_normal"],"future_key":{"a":1}}`
	if err := os.WriteFile(path, []byte(old), 0o600); err != nil {
		t.Fatal(err)
	}
	r := newRig(t, "paladin", path)
	for _, trig := range r.c.Legacy() {
		if _, err := r.eng.Adopt(LadderID(trig)); err != nil {
			t.Fatal(err)
		}
	}
	if lockedOf(r.c, "blood_ravens_tithe") || lockedOf(r.c, "pact_of_nightmares") {
		t.Fatal("the migrated unlocks are not open")
	}
	for i := 0; i < 2; i++ { // idempotent
		if err := r.c.ClearLegacy(); err != nil {
			t.Fatal(err)
		}
	}
	if len(r.c.Legacy()) != 0 {
		t.Fatal("legacy field kept")
	}
	if ok, _ := r.eng.Adopt("pact_baal_normal"); ok {
		t.Fatal("adopting twice reported a change")
	}
	b, _ := os.ReadFile(path)
	var m map[string]json.RawMessage
	if err := json.Unmarshal(b, &m); err != nil {
		t.Fatal(err)
	}
	if _, ok := m["future_key"]; !ok {
		t.Fatalf("unknown key dropped: %s", b)
	}
	if _, ok := m["unlocked"]; ok {
		t.Fatalf("legacy unlocked kept: %s", b)
	}
	if _, err := r.eng.Adopt("pact_nope"); err == nil {
		t.Fatal("unknown ladder entry adopted")
	}
}

func TestConsentLines(t *testing.T) {
	r := newRig(t, "sorceress", "")
	r.kill("Andariel", "normal")
	if err := r.c.Choose([]Pick{{"blood_ravens_tithe", 1}}); err != nil {
		t.Fatal(err)
	}
	if err := r.c.GrantFor("blood_ravens_tithe", "Blood Raven's Tithe", 0, 12); err != nil {
		t.Fatal(err)
	}
	if err := r.c.Revoke("blood_ravens_tithe"); err != nil {
		t.Fatal(err)
	}
	r.c.Revoke("blood_ravens_tithe") // nothing to revoke: no second line
	if len(r.consent) != 2 || !strings.Contains(r.consent[0], "given for rank 1, up to 12 hours") || !strings.Contains(r.consent[1], "revoked") {
		t.Fatalf("%q", r.consent)
	}
	r.c.GrantFor("blood_ravens_tithe", "Blood Raven's Tithe", 0, 1)
	r.now = r.now.Add(2 * time.Hour)
	r.c.Evaluate(field)
	r.c.Evaluate(field)
	if n := len(r.consent); n != 4 || !strings.Contains(r.consent[3], "ran out") {
		t.Fatalf("%q", r.consent)
	}
}
