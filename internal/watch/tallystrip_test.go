package watch

import (
	"encoding/json"
	"errors"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"d2lod-bot/internal/currency"
	"d2lod-bot/internal/goals"
)

func tallyRig(t *testing.T, build string, level int) (*rig, *TallyStrip, *time.Time) {
	t.Helper()
	r := newRig(t, build)
	now := time.Date(2026, 11, 6, 12, 0, 0, 0, time.UTC)
	l, err := currency.Open(t.TempDir(), currency.Options{Mode: testMode, Instance: "i4", Now: func() time.Time { return now }})
	if err != nil {
		t.Fatal(err)
	}
	w := currency.NewWallet(testMode, l)
	r.sim.Hub.SetCurrency(w)
	ta, err := currency.NewTally(l, currency.DefaultTallyRows(), "Hero", "r1")
	if err != nil {
		t.Fatal(err)
	}
	s, err := NewTallyStrip(r.sim.Hub, w, ta, func(a string) bool { return a == "Rogue Encampment" })
	if err != nil {
		t.Fatal(err)
	}
	s.Now = func() time.Time { return now }
	r.sim.Hub.SetHero(HeroInfo{Class: build, Level: level, Area: "Rogue Encampment"})
	return r, s, &now
}

func feed(s *TallyStrip, evs ...goals.Event) {
	for i := range evs {
		if evs[i].Seq == 0 {
			evs[i].Seq = uint64(i + 1)
		}
		s.OnEvent(evs[i])
	}
}

func trip() []goals.Event {
	evs := []goals.Event{{Type: goals.EvAreaChange, Area: "Blood Moor"}}
	for i := 0; i < 26; i++ {
		evs = append(evs, goals.Event{Type: goals.EvKill, Monster: "Fallen"})
	}
	evs = append(evs, goals.Event{Type: goals.EvKill, Monster: "Corpsefire", Boss: true},
		goals.Event{Type: goals.EvQuest, Quest: "den_of_evil", Done: true},
		goals.Event{Type: goals.EvAreaChange, Area: "Rogue Encampment"})
	for i := range evs {
		evs[i].Seq = uint64(i + 1)
	}
	return evs
}

func TestTallyStripGolden(t *testing.T) {
	for _, build := range []string{"sorceress", "paladin"} {
		_, s, _ := tallyRig(t, build, 5)
		feed(s, trip()...)
		// replaying the same journal changes nothing (idempotent)
		feed(s, trip()...)
		w := httptest.NewRecorder()
		s.Handler().ServeHTTP(w, httptest.NewRequest("GET", "/api/tally", nil))
		if w.Code != 200 {
			t.Fatalf("%d %s", w.Code, w.Body)
		}
		var v TallyView
		if err := json.Unmarshal(w.Body.Bytes(), &v); err != nil {
			t.Fatal(err)
		}
		// 1 batch of 25 kills (1) + Fallen 5 + Corpsefire first 5 + boss 15 + quest 10 + area 5 + trip 2 = 43
		if v.Balance != 43 || v.Last == nil || v.Last.Net != 43 || v.Expedition != nil || v.Error != "" {
			t.Errorf("%s: %+v", build, v)
		}
		golden(t, "tally-"+build, w.Body.String())
	}
}

func TestTallyVanillaAndPost(t *testing.T) {
	r := newRig(t, "sorceress")
	w := currency.NewWallet(currency.VanillaMode, nil)
	l, _ := currency.Open(t.TempDir(), currency.Options{Mode: currency.VanillaMode})
	ta, _ := currency.NewTally(l, currency.DefaultTallyRows(), "Hero", "r1")
	s, _ := NewTallyStrip(r.sim.Hub, w, ta, func(string) bool { return false })
	feed(s, goals.Event{Type: goals.EvKill, Monster: "Fallen"})
	if v := s.View(); v.Enabled || !strings.Contains(v.Strip, "Tally is off") || v.Error != "" {
		t.Errorf("vanilla view %+v", v)
	}
	rr := httptest.NewRecorder()
	s.Handler().ServeHTTP(rr, httptest.NewRequest("POST", "/api/tally", nil))
	if rr.Code != 405 {
		t.Errorf("POST: %d", rr.Code)
	}
	if _, err := NewTallyStrip(r.sim.Hub, w, ta, nil); err == nil {
		t.Error("nil IsTown accepted")
	}
}

func TestEventWithoutSeqIsReported(t *testing.T) {
	_, s, _ := tallyRig(t, "sorceress", 5)
	s.OnEvent(goals.Event{Type: goals.EvKill, Monster: "Fallen"})
	if v := s.View(); !strings.Contains(v.Error, "sequence") {
		t.Errorf("error %q", v.Error)
	}
}

func TestCharterFlow(t *testing.T) {
	for _, build := range []string{"sorceress", "paladin"} {
		r, s, _ := tallyRig(t, build, 5)
		go1 := OrderRequest{Kind: OrderGo, Area: "Cold Plains", Minutes: 30}
		// Cold Plains is meant for level 6: one level above, charter 20.
		_, q, err := s.PlaceOrder(go1, false)
		if !errors.Is(err, currency.ErrCharterNeeded) || q.Price != 20 || !strings.Contains(q.Text, "20 tally") {
			t.Fatalf("%s: unapproved: %+v %v", build, q, err)
		}
		if _, _, err := s.PlaceOrder(go1, true); !errors.Is(err, currency.ErrInsufficient) {
			t.Fatalf("%s: no funds: %v", build, err)
		}
		if n := len(r.sim.Hub.openIDs()); n != 0 {
			t.Fatalf("%s: a refused charter placed %d orders", build, n)
		}
		if _, err := s.Wallet.Ledger.Grant("tally", "grant", "g1", 25, "test"); err != nil {
			t.Fatal(err)
		}
		o, _, err := s.PlaceOrder(go1, true)
		if err != nil || o.ID == 0 {
			t.Fatalf("%s: approved: %v", build, err)
		}
		if b, _ := s.Wallet.Ledger.Balance("tally"); b.Spendable != 5 {
			t.Errorf("%s: balance %d, want 5", build, b.Spendable)
		}
		// in range is free: hero level 6
		r.sim.Hub.SetHero(HeroInfo{Class: build, Level: 6, Area: "Rogue Encampment"})
		if _, q, err := s.PlaceOrder(OrderRequest{Kind: OrderGo, Area: "Cold Plains", Minutes: 10}, false); err != nil || q.Price != 0 {
			t.Errorf("%s: in range: %+v %v", build, q, err)
		}
		if b, _ := s.Wallet.Ledger.Balance("tally"); b.Spendable != 5 {
			t.Errorf("%s: a free trip cost tally", build)
		}
		if _, _, err := s.PlaceOrder(OrderRequest{Kind: OrderGo, Area: "Moo Moo Farm", Minutes: 10}, true); err == nil {
			t.Errorf("%s: unknown area accepted", build)
		}
	}
}
