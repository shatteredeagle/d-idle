package watch

import (
	"bytes"
	"encoding/json"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"d2lod-bot/internal/campaign"
)

// journeyDesk is a real controller on the shipped spine after one decision for class at clvl lvl with done quests.
func journeyDesk(t *testing.T, class string, lvl int, done ...string) *campaign.Controller {
	t.Helper()
	s, err := campaign.Shipped(campaign.Validator{})
	if err != nil {
		t.Fatal(err)
	}
	now := time.Date(2026, 11, 11, 12, 0, 0, 0, time.UTC)
	jc, err := campaign.NewController(s, "", func() string { return "Hero" }, func() time.Time { return now })
	if err != nil {
		t.Fatal(err)
	}
	jc.Pace = func(int) (float64, time.Duration, bool) { return 0.62, 25 * time.Minute, true }
	p := campaign.Progress{Hero: campaign.Hero{Class: class, Level: lvl}, Quests: map[string]bool{}, Waypoints: map[int]bool{3: true}}
	for _, q := range done {
		p.Quests[q] = true
		p.MercOwned = p.MercOwned || q == "sisters_burial" // the Rogue was claimed and is dead (the live hero, 2026-11-11)
	}
	o := jc.Opts()
	d, err := s.Next(p, jc.State(), o)
	if err != nil {
		t.Fatal(err)
	}
	jc.Decided(d, p, o)
	return jc
}

// GET /api/campaign for a Paladin and a Sorceress against goldens (UPDATE=1 writes them; goldens are only added).
func TestCampaignViewGoldens(t *testing.T) {
	for _, c := range []struct {
		name, class string
		lvl         int
		done        []string
	}{
		{"view-paladin-clvl12", "paladin", 12, []string{"den_of_evil", "sisters_burial"}},
		{"view-sorceress-clvl7", "sorceress", 7, []string{"den_of_evil"}},
	} {
		api := NewCampaignAPI()
		api.Bind(journeyDesk(t, c.class, c.lvl, c.done...))
		w := httptest.NewRecorder()
		api.Handler().ServeHTTP(w, httptest.NewRequest("GET", CampaignRoute, nil))
		if w.Code != 200 {
			t.Fatalf("%s: %d", c.name, w.Code)
		}
		var buf bytes.Buffer
		if err := json.Indent(&buf, w.Body.Bytes(), "", " "); err != nil {
			t.Fatal(err)
		}
		golden := filepath.Join("testdata", "campaign", c.name+".json")
		if os.Getenv("UPDATE") != "" {
			os.WriteFile(golden, buf.Bytes(), 0o644)
		}
		want, err := os.ReadFile(golden)
		if err != nil {
			t.Fatal(err)
		}
		if !bytes.Equal(bytes.TrimSpace(buf.Bytes()), bytes.TrimSpace(want)) {
			t.Errorf("%s differs from %s:\n%s", c.name, golden, buf.String())
		}
	}
}

func TestCampaignIntentGate(t *testing.T) {
	api := NewCampaignAPI()
	o := AnswerOptions{Token: "tok", Origins: []string{"http://ok.test"}}
	post := api.PostHandler(o)
	auth := map[string]string{"Authorization": "Bearer tok"}
	if w := invPost(post, CampaignIntentRoute, `{"chapter":"countess","intent":"skip"}`, auth); w.Code != 503 {
		t.Errorf("desk not bound: %d", w.Code)
	}
	jc := journeyDesk(t, "paladin", 12, "den_of_evil", "sisters_burial")
	api.Bind(jc)
	if w := invPost(post, CampaignIntentRoute, `{"chapter":"countess","intent":"skip"}`, auth); w.Code != 200 || !strings.Contains(w.Body.String(), `"state":"skipped"`) {
		t.Errorf("skip: %d %s", w.Code, w.Body)
	}
	if w := invPost(post, CampaignIntentRoute, `{"chapter":"andariel","goNow":true}`, auth); w.Code != 200 || jc.State().GoNow != "andariel" {
		t.Errorf("go now: %d", w.Code)
	}
	for name, c := range map[string]struct {
		body string
		hdr  map[string]string
		code int
	}{
		"no token":        {`{"mode":"free"}`, nil, 401},
		"foreign origin":  {`{"mode":"free"}`, map[string]string{"Authorization": "Bearer tok", "Origin": "http://evil.test"}, 403},
		"unknown field":   {`{"mode":"free","x":1}`, auth, 400},
		"unknown chapter": {`{"chapter":"diablo","intent":"skip"}`, auth, 400},
		"bad intent":      {`{"chapter":"countess","intent":"maybe"}`, auth, 400},
		"empty":           {`{}`, auth, 400},
	} {
		if w := invPost(post, CampaignIntentRoute, c.body, c.hdr); w.Code != c.code {
			t.Errorf("%s: %d, want %d", name, w.Code, c.code)
		}
	}
	w := httptest.NewRecorder()
	api.Handler().ServeHTTP(w, httptest.NewRequest("POST", CampaignRoute, nil))
	if w.Code != 405 {
		t.Errorf("viewer accepts POST: %d", w.Code)
	}
}

// The journey panel against a stub DOM (node): golden testdata/campaign/journey_dom.txt, UPDATE=1 rewrites it.
func TestJourneyPanelDOM(t *testing.T) {
	for _, c := range []struct {
		class string
		lvl   int
		done  []string
	}{{"paladin", 12, []string{"den_of_evil", "sisters_burial"}}, {"sorceress", 7, []string{"den_of_evil"}}} {
		b, _ := json.Marshal(journeyDesk(t, c.class, c.lvl, c.done...).View())
		os.WriteFile(filepath.Join("testdata", "campaign", "view-"+c.class+".tmp.json"), b, 0o644)
	}
	defer os.Remove(filepath.Join("testdata", "campaign", "view-paladin.tmp.json"))
	defer os.Remove(filepath.Join("testdata", "campaign", "view-sorceress.tmp.json"))
	runNode(t, "testdata/campaign/journey_dom.js", "web/panels/journey/panel.js")
}

// The top band chip of the journey (node, skipped without it).
func TestJourneyChipLine(t *testing.T) {
	runNode(t, "testdata/campaign/journey_chip.js", "web/panels/journey-chip/panel.js")
}
