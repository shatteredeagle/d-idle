package watch

import (
	"encoding/json"
	"strings"
	"testing"
	"time"

	"d2lod-bot/internal/goals"
	"d2lod-bot/internal/unlock"
)

// shippedRig is the fun rig over the shipped ladder (data/unlocks/ladder.json).
func shippedRig(t *testing.T, class string) (*funRig, *unlock.Engine) {
	t.Helper()
	l, err := unlock.Shipped()
	if err != nil {
		t.Fatal(err)
	}
	eng, err := unlock.Open(t.TempDir(), strings.ToLower(class), l, func() time.Time { return time.Unix(1_800_000_000, 0) })
	if err != nil {
		t.Fatal(err)
	}
	return newFunRigWith(t, class, 1, func(_ *funRig, fc *FunConfig) { fc.Unlock = eng }), eng
}

func chronicleOf(r *funRig, query string) ChronicleView {
	var v ChronicleView
	r.getJSON(ChronicleRoute+query, &v)
	return v
}

func TestChronicleGoldenTwoClasses(t *testing.T) {
	for _, class := range []string{"Sorceress", "Paladin"} {
		t.Run(class, func(t *testing.T) {
			r, eng := shippedRig(t, class)
			// before the first kill: a silhouette, no lines, even though an area change was already recorded
			r.feed(goals.Event{Type: goals.EvAreaChange, Area: "Blood Moor", AreaID: 2, Seq: 1})
			v := chronicleOf(r, "")
			if v.Open || len(v.Lines) != 0 || v.Teaser == "" || v.Tape.Open || v.Tape.Teaser == "" {
				t.Fatalf("locked view: %+v", v)
			}
			stream := funStream()
			for i := range stream {
				stream[i].Seq += 1 // after the area change above
			}
			r.feed(stream...)
			if !eng.Allowed(ChronicleID) {
				t.Fatal("first kill did not open the Chronicle")
			}
			b, _ := json.Marshal(chronicleOf(r, ""))
			golden(t, "chronicle-"+strings.ToLower(class), string(b)+"\n")
			// the opening is never silent: a Chronicle story line and an away event exist
			found := false
			for _, l := range storyLines(r) {
				found = found || strings.Contains(l, "Unlocked: Chronicle.")
			}
			if !found {
				t.Fatalf("no reveal line: %v", storyLines(r))
			}
		})
	}
}

func TestChronicleReplayAndSince(t *testing.T) {
	r, _ := shippedRig(t, "Sorceress")
	r.feed(funStream()...)
	n := len(chronicleOf(r, "").Lines)
	if n < 20 {
		t.Fatalf("only %d lines", n)
	}
	r.feed(funStream()...) // the same events again: nothing new
	if m := len(chronicleOf(r, "").Lines); m != n {
		t.Fatalf("replay added lines: %d -> %d", n, m)
	}
	if got := chronicleOf(r, "?since=3").Lines; len(got) != n-3 || got[0].N != 4 {
		t.Fatalf("since=3: %d lines, first %+v", len(got), got)
	}
	// no quality word, no colour: a unique drop is "Picked up: <name>", the reveal owns the rest
	for _, l := range chronicleOf(r, "").Lines {
		if l.Kind == "drop" && strings.Contains(strings.ToLower(l.Text), "unique") {
			t.Errorf("drop line names its quality: %q", l.Text)
		}
	}
}

func TestChronicleTapeOpensOnTheFirstCardRead(t *testing.T) {
	r, eng := shippedRig(t, "Paladin")
	r.feed(funStream()...)
	if eng.Allowed(ChronicleTapeID) {
		t.Fatal("tape opened before any card was read")
	}
	if v := chronicleOf(r, ""); v.Tape.Open || len(v.Tape.Trips) != 0 {
		t.Fatalf("tape leaks: %+v", v.Tape)
	}
	// the JSON data route is not a read of the card; the page is
	var d ReturnData
	r.getJSON("/api/return", &d)
	if eng.Allowed(ChronicleTapeID) {
		t.Fatal("the JSON route counted as a card read")
	}
	r.wall = r.wall.Add(time.Minute)
	if resp, _ := r.do("GET", r.view, "/watch/return", funToken, ""); resp.StatusCode != 200 {
		t.Fatalf("return page: %d", resp.StatusCode)
	}
	v := chronicleOf(r, "")
	if !eng.Allowed(ChronicleTapeID) || !v.Tape.Open || len(v.Tape.Trips) == 0 {
		t.Fatalf("tape after the read: allowed %v %+v", eng.Allowed(ChronicleTapeID), v.Tape)
	}
	r.do("GET", r.view, "/watch/return", funToken, "") // again: no second reveal
	n := 0
	for _, l := range storyLines(r) {
		if strings.Contains(l, "Run timeline") {
			n++
		}
	}
	if n != 1 {
		t.Fatalf("reveal lines for the tape: %d", n)
	}
}

func TestChronicleFirstSightsFromTheCodex(t *testing.T) {
	r, _ := shippedRig(t, "Sorceress")
	r.feed(funStream()...)
	var first []string
	for _, l := range chronicleOf(r, "").Lines {
		if l.Kind == "first" {
			first = append(first, l.Text)
		}
	}
	if len(first) == 0 {
		t.Fatal("no first-sight line although the Codex fired")
	}
}

func TestChronicleUnknownRowAndWording(t *testing.T) {
	l, err := unlock.Parse([]byte(`{"schema":"unlock.ladder/1","entries":[
 {"id":"chronicle","group":"ui","trigger":{"kind":"kills_of_kind","args":{"kind":"*","min":1}},"reveal":{"title":"C","text":"c"}}]}`))
	if err != nil {
		t.Fatal(err)
	}
	eng, _ := unlock.Open("", "p", l, nil)
	if _, err := NewChronicle(&Unlocks{Engine: eng}, nil, nil, nil); err == nil || !strings.Contains(err.Error(), `"chronicle_tape"`) {
		t.Fatalf("missing tape row: %v", err)
	}
	if _, err := ParseWording([]byte(`{"schema":"chronicle.wording/1","kinds":{"nonsense":["a","b","c"]}}`)); err == nil || !strings.Contains(err.Error(), `"nonsense"`) {
		t.Fatalf("unknown kind: %v", err)
	}
	if _, err := ParseWording([]byte(`{"schema":"chronicle.wording/1","kinds":{"kill":["a"]}}`)); err == nil || !strings.Contains(err.Error(), `"kill"`) {
		t.Fatalf("too few variants: %v", err)
	}
}

func TestChronicleWordingShipped(t *testing.T) {
	c, err := NewChronicle(mustUnlocks(t), nil, nil, nil)
	if err != nil {
		t.Fatal(err)
	}
	c.OnGoalEvent(goals.Event{Type: goals.EvKill, Monster: "Fallen", Seq: 7})
	c.OnGoalEvent(goals.Event{Type: goals.EvKill, Monster: "Fallen", Seq: 7}) // replay
	c.OnGoalEvent(goals.Event{Type: goals.EvKill, Monster: "Fallen", Seq: 7 + 1})
	if len(c.lines) != 2 || !strings.Contains(c.lines[0].Text, "Fallen") {
		t.Fatalf("%+v", c.lines)
	}
	a, b := c.lines[0].Text, ""
	c2, _ := NewChronicle(mustUnlocks(t), nil, nil, nil)
	c2.OnGoalEvent(goals.Event{Type: goals.EvKill, Monster: "Fallen", Seq: 7})
	b = c2.lines[0].Text
	if a != b {
		t.Fatalf("the same event printed two lines: %q %q", a, b)
	}
}

func mustUnlocks(t *testing.T) *Unlocks {
	t.Helper()
	l, err := unlock.Shipped()
	if err != nil {
		t.Fatal(err)
	}
	eng, err := unlock.Open("", "p", l, nil)
	if err != nil {
		t.Fatal(err)
	}
	return &Unlocks{Engine: eng}
}
