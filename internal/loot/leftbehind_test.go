package loot

import "testing"

func TestNoteLeftBehindCountsOnlyValuablesAndSkipsNamedCalls(t *testing.T) {
	var l Looter
	sk := []Skipped{{Why: "no room", Prio: 90}, {Why: "no room", Prio: 5}, {Why: "filtered", Prio: 99}, {Why: "no room", Prio: LeftBehindMinPrio}}
	l.noteLeftBehind(sk, PickOpts{})
	if got := l.LeftBehind(); got != 2 {
		t.Fatalf("left behind %d, want 2", got)
	}
	l.noteLeftBehind(nil, PickOpts{Names: []string{"x"}}) // a named pickup says nothing about the rest
	if l.LeftBehind() != 2 {
		t.Fatal("named call reset the count")
	}
	l.noteLeftBehind(nil, PickOpts{})
	if l.LeftBehind() != 0 {
		t.Fatal("clean pickup did not reset")
	}
}
