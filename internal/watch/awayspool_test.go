package watch

import (
	"os"
	"path/filepath"
	"testing"
	"time"
)

func TestAwaySpoolSurvivesRestart(t *testing.T) {
	path := filepath.Join(t.TempDir(), "away-spool.jsonl")
	wall := time.Unix(1_000_000, 0)
	mk := func(now float64) *AwayLog {
		l := NewAwayLog(func() float64 { return now })
		l.wall = func() time.Time { return wall }
		return l
	}
	a := mk(500)
	if err := a.AttachSpool(path); err != nil {
		t.Fatal(err)
	}
	a.Add(AwayEvent{T: 400, Kind: AwayClose, LifePct: 33, Sec: 8})
	a.Add(AwayEvent{T: 450, Kind: AwayDrop, Item: "Cap", Quality: "normal"})
	a.Add(AwayEvent{T: 460, Kind: AwayKill, Monster: "Fallen"}) // not spooled
	// ten minutes later, new process, new time base; one event is too old and one has no wall time
	wall = wall.Add(10 * time.Minute)
	old := `{"t":1,"kind":"close","lifePct":5,"wall":1}` + "\n" + `{"t":1,"kind":"close","lifePct":6}` + "\n"
	f, _ := os.OpenFile(path, os.O_APPEND|os.O_WRONLY, 0)
	f.WriteString(old)
	f.Close()
	b := mk(30)
	if err := b.AttachSpool(path); err != nil {
		t.Fatal(err)
	}
	evs := b.Events()
	if len(evs) != 2 || evs[0].Kind != AwayClose || evs[0].LifePct != 33 || evs[0].Sec != 8 || evs[1].Item != "Cap" {
		t.Fatalf("%+v", evs)
	}
	if evs[0].T != 30-600 || evs[1].T != 30-600 {
		t.Errorf("events not moved onto the new time base: %v %v", evs[0].T, evs[1].T)
	}
	b.Add(AwayEvent{Kind: AwayClose, LifePct: 50})
	c := mk(40)
	if err := c.AttachSpool(path); err != nil {
		t.Fatal(err)
	}
	if n := len(c.Events()); n != 3 {
		t.Errorf("third start has %d events, want 3", n)
	}
}
