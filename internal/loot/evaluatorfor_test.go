package loot_test

import (
	"testing"

	"d2lod-bot/internal/d2"
)

// EvaluatorFor gives the verdict EvaluateUnit gives, from one snapshot read, and reads the game no more.
func TestEvaluatorForMatchesEvaluateUnit(t *testing.T) {
	l, r := newLooter(t)
	w := r.W
	w.Player.Level = 10
	snaps := 0
	l.UseSource(func() (*d2.Snapshot, error) { snaps++; return w.Snapshot() }, w.UnitStats)
	var items []d2.Unit
	for i, code := range []string{"hp1", "mp1", "hax", "cap"} {
		u := w.NewItem(code, 2)
		u.Addr = uint32(0xC0DE + i)
		items = append(items, u)
	}
	want := make([]string, len(items))
	for i, u := range items {
		want[i] = l.EvaluateUnit(u).String()
	}
	snaps = 0
	s, _ := w.Snapshot()
	ev := l.EvaluatorFor(s)
	for i, u := range items {
		if got := ev(u).String(); got != want[i] {
			t.Errorf("item %d: %q want %q", i, got, want[i])
		}
	}
	if snaps != 0 {
		t.Errorf("EvaluatorFor read the game %d times", snaps)
	}
	if ev2 := l.EvaluatorFor(nil); ev2(items[0]).String() == "" {
		t.Error("nil snapshot")
	}
}
