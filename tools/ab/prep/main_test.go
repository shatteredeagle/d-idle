package main

import (
	"reflect"
	"testing"
)

func TestVerbs(t *testing.T) {
	got := Verbs(Setup{Clvl: 11, Gold: 500, Stash: 0, MercType: 0})
	want := []string{"debug stat 12 11", "debug statpts 50", "debug skillpts 10", "debug stat 14 500", "debug stat 15 0", "debug merc hire 0"}
	if !reflect.DeepEqual(got, want) {
		t.Errorf("got %v", got)
	}
	if v := Verbs(Setup{Clvl: 5, MercType: -1}); len(v) != 5 {
		t.Errorf("no merc: %v", v)
	}
}
