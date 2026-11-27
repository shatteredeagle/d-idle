package main

import "testing"

func TestInvariantCountsAndPolicy(t *testing.T) {
	log := []byte(`{"time":"2026-11-26T00:00:00Z","level":"WARN","msg":"invariant potion-not-drunk","detail":"x"}
{"time":"2026-11-26T00:00:01Z","level":"WARN","msg":"invariant potion-not-drunk","detail":"x"}
{"time":"2026-11-26T00:00:02Z","level":"WARN","msg":"invariant town-thrash","detail":"y"}
{"time":"2026-11-26T00:00:03Z","level":"INFO","msg":"zone entry"}
`)
	h := invariantCounts(log)
	if h["potion-not-drunk"] != 2 || h["town-thrash"] != 1 || len(h) != 2 {
		t.Fatalf("counts %v", h)
	}
	p := loadInvariantPolicy()
	a := p.invariantAsserts("drill.x", h)
	if len(a) != 1 || a[0].OK || a[0].Kind != "invariant potion-not-drunk" {
		t.Fatalf("hard hit must be red, soft ignored: %+v", a)
	}
	a = p.invariantAsserts("incident.questy", h)
	if len(a) != 1 || !a[0].OK {
		t.Fatalf("report-only must not be red: %+v", a)
	}
	if len(p.invariantAsserts("drill.x", nil)) != 0 {
		t.Fatal("no hits no asserts")
	}
	if a := p.invariantAsserts("drill.x", map[string]int{"mode-no-exit": 3}); len(a) != 0 {
		t.Fatalf("mode-no-exit is soft in drills: %+v", a)
	}
	if a := p.invariantAsserts("town.x", map[string]int{"mode-no-exit": 3}); len(a) != 1 || a[0].OK {
		t.Fatalf("mode-no-exit stays hard elsewhere: %+v", a)
	}
}

func TestInvariantFragmentsMerge(t *testing.T) {
	p := loadInvariantPolicy()
	if len(p.Hard) == 0 {
		t.Fatal("base policy empty")
	}
}
