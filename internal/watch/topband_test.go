package watch

import (
	"encoding/json"
	"os"
	"regexp"
	"testing"
)

// /api/hero carries the stats block for the hero stats tile: null until the bot reports it, then the resistances and
// the readiness light with plain, bounded text. Both test classes: nothing here knows a class.
func TestHeroStatsBlock(t *testing.T) {
	for _, build := range []string{"sorceress", "paladin"} {
		r := newRig(t, build)
		var raw map[string]json.RawMessage
		r.get(HeroRoute, &raw)
		if string(raw["stats"]) != "null" {
			t.Errorf("%s: stats without a source: %s", build, raw["stats"])
		}
		r.sim.Hub.SetHeroStatsSource(func() (HeroStats, bool) { return HeroStats{}, false })
		r.get(HeroRoute, &raw)
		if string(raw["stats"]) != "null" {
			t.Errorf("%s: stats before the first read: %s", build, raw["stats"])
		}
		long := "<b>resist</b>\n"
		for len(long) < 400 {
			long += "x"
		}
		r.sim.Hub.SetHeroStatsSource(func() (HeroStats, bool) {
			return HeroStats{Resists: HeroResists{Fire: 12, Cold: 30, Light: -8, Poison: 5}, Readiness: &HeroReadiness{Light: "amber", Area: "Blood Moor", Reason: long}}, true
		})
		var v HeroView
		r.get(HeroRoute, &v)
		if v.Stats == nil || v.Stats.Resists.Light != -8 || v.Stats.Readiness == nil || v.Stats.Readiness.Light != "amber" {
			t.Fatalf("%s: stats %+v", build, v.Stats)
		}
		if n := len(v.Stats.Readiness.Reason); n > 160 {
			t.Errorf("%s: reason not bounded: %d", build, n)
		}
		r.sim.Hub.SetHeroStatsSource(func() (HeroStats, bool) { return HeroStats{Resists: HeroResists{Fire: 1}}, true })
		r.get(HeroRoute, &v)
		if v.Stats == nil || v.Stats.Readiness != nil {
			t.Errorf("%s: stats without a readiness model: %+v", build, v.Stats)
		}
	}
}

// DOM goldens of the three top-band panels in node (ready, unreported, empty, error).
func TestTopBandPanelDOMGoldens(t *testing.T) {
	runNode(t, "testdata/topband_dom.js", "web/panels")
}

// The new panels keep the page rules: no timers, requests or queries of their own, text through textContent, no
// focus calls, no style writes, and the existing panels they sit beside are unchanged (their v1 manifests still match).
func TestTopBandPanelsFollowTheContract(t *testing.T) {
	bad := regexp.MustCompile(`setInterval|setTimeout|fetch\(|EventSource|document\.|innerHTML|\.style\b|\.focus\(|window\.`)
	for _, id := range []string{"hero-meters", "alerts-count", "hero-detail"} {
		b, err := webFS.ReadFile("web/panels/" + id + "/panel.js")
		if err != nil {
			t.Fatal(err)
		}
		if m := bad.Find(b); m != nil {
			t.Errorf("%s: %q breaks the panel contract", id, m)
		}
	}
	for _, id := range []string{"hero-status", "hero-vitals", "activity", "why", "gear", "tally", "currency", "map", "video"} {
		cur, err := webFS.ReadFile("web/panels/" + id + "/panel.json")
		if err != nil {
			t.Fatal(err)
		}
		old, err := os.ReadFile("testdata/compat/panels-v1/" + id + ".json")
		if err != nil {
			t.Fatal(err)
		}
		var a, b PanelManifest
		json.Unmarshal(cur, &a)
		json.Unmarshal(old, &b)
		if a.Band != b.Band || a.Endpoint != b.Endpoint || a.Title != b.Title {
			t.Errorf("%s changed its band, endpoint or title", id)
		}
	}
}
