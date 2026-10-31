package main

import (
	"testing"

	"d2lod-bot/internal/d2dctl"
)

func TestCount(t *testing.T) {
	m := map[int]d2dctl.Monster{
		1: {Name: "Fallen", Boss: "-"}, 2: {Name: "Fallen", Boss: "-"}, 3: {Name: "Fallen", Boss: "minion"},
		4: {Name: "Snot Wing", Boss: "unique"}, 5: {Name: "Bishibosh", Boss: "superunique"}, 6: {Name: "Champion Fallen", Boss: "champion"},
	}
	r := Count(3, 7, m)
	if r.Normal != 2 || r.Minion != 1 || r.Unique != 1 || r.SuperUnique != 1 || r.Champion != 1 {
		t.Fatalf("%+v", r)
	}
	want := `3,7,2,1,1,1,1,6,"Fallen x3","champion:Champion Fallen;superunique:Bishibosh;unique:Snot Wing"`
	if got := r.CSV(); got != want {
		t.Fatalf("got %s want %s", got, want)
	}
}
