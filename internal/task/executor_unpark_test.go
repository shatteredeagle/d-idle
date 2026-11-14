package task

import "testing"

func TestUnparkLiftsTheBlock(t *testing.T) {
	e := NewExecutor(nil, nil)
	e.blocked["den"] = "too weak"
	e.strikes["den"] = &strikeRec{parked: true}
	e.fails["quest.den/to-den"] = 3
	e.Unpark("den")
	if _, ok := e.Blocked("den"); ok || len(e.strikes) != 0 || len(e.fails) != 0 {
		t.Fatalf("blocked %v strikes %v fails %v", e.blocked, e.strikes, e.fails)
	}
}
