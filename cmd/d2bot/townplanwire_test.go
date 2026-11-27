package main

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"d2lod-bot/internal/supply"
	"d2lod-bot/internal/watch"
)

func townPlanGet(f *watch.Fun) int {
	w := httptest.NewRecorder()
	f.TownPlan.Handler().ServeHTTP(w, httptest.NewRequest("GET", watch.TownPlanRoute, nil))
	return w.Code
}

// must-fail: a Fun whose route was never bound answers 503, not a made-up plan.
func TestBindTownPlanUnboundIs503(t *testing.T) {
	f := &watch.Fun{}
	bindTownPlan(f, nil)
	if f.TownPlan == nil {
		t.Fatal("the route must be mounted even without a source")
	}
	if got := townPlanGet(f); got != http.StatusServiceUnavailable {
		t.Fatalf("unbound route: %d, want 503", got)
	}
	bindTownPlan(nil, nil) // no Fun: nothing, no panic
}

func TestBindTownPlanAnswersForTwoBuilds(t *testing.T) {
	for name, in := range map[string]supply.Inputs{
		"sorceress-like": {ManaNeed: 2, Capacity: map[supply.Line]int{supply.Healing: 4, supply.Mana: 4}},
		"paladin-like":   {ManaNeed: 0, Capacity: map[supply.Line]int{supply.Healing: 4}},
	} {
		n, err := supply.NeedFor("town", "field", in)
		if err != nil {
			t.Fatal(name, err)
		}
		c := supply.Count{Purse: 200}
		v := supply.ViewOf(supply.Solve(n, c, supply.Prices{supply.Healing: 30, supply.Mana: 30, supply.Portal: 100}, supply.Policy{}, 0), c, nil, 0)
		f := &watch.Fun{}
		bindTownPlan(f, func() supply.View { return v })
		if got := townPlanGet(f); got != 200 {
			t.Fatalf("%s: %d", name, got)
		}
	}
	// the late source: no town yet is NoView (known false), not a 503
	ref := &runnerRef{}
	if v := townPlanSource(ref)(); v.Known {
		t.Fatal("no town: the view must be NoView")
	}
	if townPlanSource(nil) != nil {
		t.Fatal("nil ref: no source")
	}
}
