package decide

import (
	"strings"
	"testing"
	"time"
)

func firstsRig(t *testing.T, class, dir string) (*Tracker, *Store, *Codex, *ManualClock) {
	t.Helper()
	tbl, err := ShippedFirsts()
	if err != nil {
		t.Fatal(err)
	}
	cat, err := mustCat(t).WithFirsts(tbl)
	if err != nil {
		t.Fatal(err)
	}
	clk := &ManualClock{}
	st, err := New(cat, "p1", clk)
	if err != nil {
		t.Fatal(err)
	}
	var cx *Codex
	if dir != "" {
		cx, err = OpenCodex(dir, "p1")
		if err != nil {
			t.Fatal(err)
		}
	} else {
		cx = NewCodex("p1")
	}
	tr, err := NewTracker(tbl, cat, st, cx, clk, Who{Class: class, Diff: "normal"})
	if err != nil {
		t.Fatal(err)
	}
	return tr, st, cx, clk
}

// eventFor builds an event that triggers exactly the row.
func eventFor(r First) FirstEvent {
	name := r.Match[0]
	if name == "*" {
		name = "Anything"
	}
	e := FirstEvent{Kind: r.Kind, Name: name, Area: "Blood Moor", Clvl: 4}
	if r.Kind == "champion" && name != "Anything" {
		e.Name, e.Mods = "Champion", []string{name}
	}
	return e
}

func TestShippedFirstsValid(t *testing.T) {
	tbl, err := ShippedFirsts()
	if err != nil {
		t.Fatal(err)
	}
	if len(tbl.Firsts) < 30 {
		t.Fatalf("%d firsts, the Act 1 list has at least 30", len(tbl.Firsts))
	}
	cat := mustCat(t)
	if _, err := cat.Point(FirstsPoint); err == nil {
		t.Fatal("the shipped catalogue must stay unchanged (WithFirsts adds the point)")
	}
	n := len(cat.Points)
	cat2, err := cat.WithFirsts(tbl)
	if err != nil || len(cat2.Points) != n+1 || len(cat.Points) != n {
		t.Fatalf("WithFirsts: err %v, points %d -> %d, original %d", err, n, len(cat2.Points), len(cat.Points))
	}
	if _, err := cat2.WithFirsts(tbl); err == nil {
		t.Fatal("a second WithFirsts must be refused")
	}
	for _, id := range []string{"fallen", "fallen_shaman", "corpsefire", "bishibosh", "blood_raven", "rakanishu", "countess", "andariel",
		"den_of_evil", "wp_cold_plains", "wp_stony_field", "wp_dark_wood", "wp_black_marsh", "wp_outer_cloister", "wp_jail_1",
		"wp_inner_cloister", "wp_catacombs_2", "first_shrine", "first_rune", "first_unique", "first_set", "first_rare",
		"champion_pack", "champion_notable", "first_death", "forgotten_tower"} {
		if !contains(cat2.Fields["first"].Values, id) {
			t.Errorf("first %q missing", id)
		}
	}
}

func TestFirsts_TableRejects(t *testing.T) {
	for name, mut := range map[string]func(string) string{
		"dup":     func(s string) string { return strings.Replace(s, `"id": "fallen_shaman"`, `"id": "fallen"`, 1) },
		"kind":    func(s string) string { return strings.Replace(s, `"kind": "death"`, `"kind": "dream"`, 1) },
		"default": func(s string) string { return strings.Replace(s, `"default": "no"`, `"default": "maybe"`, 1) },
		"tag":     func(s string) string { return strings.Replace(s, `"tipTag": "V"`, `"tipTag": "X"`, 1) },
	} {
		if _, err := ParseFirsts([]byte(mut(string(shippedFirsts)))); err == nil {
			t.Errorf("%s: want an error", name)
		}
	}
}

// Every first fires once, never twice, and survives a restart (per-profile persistence), for two classes.
func TestFirsts_EveryFirstOnce_TwoClasses(t *testing.T) {
	tbl, _ := ShippedFirsts()
	for _, class := range []string{"sorceress", "paladin"} {
		for _, r := range tbl.Firsts {
			t.Run(class+"/"+r.ID, func(t *testing.T) {
				dir := t.TempDir()
				tr, st, cx, clk := firstsRig(t, class, dir)
				clk.Advance(90 * time.Second)
				ev := eventFor(r)
				got, err := tr.Observe(ev)
				if err != nil {
					t.Fatal(err)
				}
				var f *Fired
				for i := range got {
					if got[i].ID == r.ID {
						f = &got[i]
					}
				}
				if f == nil {
					t.Fatalf("event %+v did not fire %s (fired %v)", ev, r.ID, got)
				}
				if f.Entry.FirstSec != 90 || f.Entry.Tip != r.Tip {
					t.Fatalf("entry %+v", f.Entry)
				}
				wantThought := r.Question != ""
				if (f.Decision != nil) != wantThought || (len(st.OpenThoughts()) > 0) != wantThought {
					t.Fatalf("question %v, decision %v, open %d", wantThought, f.Decision, len(st.OpenThoughts()))
				}
				if wantThought {
					want := tr.defaultFor(r)
					if f.Decision.Answer != want || f.Entry.Decided != want || f.Entry.Label == "" {
						t.Fatalf("default %q, entry %+v, want %q", f.Decision.Answer, f.Entry, want)
					}
					th := st.OpenThoughts()[0]
					if th.Sit["question"] != r.Question || th.Sit["consequence"] != r.Consequence || th.Sit["tiptag"] != r.TipTag {
						t.Fatalf("situation %v", th.Sit)
					}
				}
				if again, err := tr.Observe(ev); err != nil || len(again) != 0 {
					t.Fatalf("second observe fired %v (%v)", again, err)
				}
				// restart: a new store and tracker over the same codex file
				tr2, st2, cx2, _ := firstsRig(t, class, dir)
				if !cx2.Has(r.ID) || cx2.Decided(r.ID) != cx.Decided(r.ID) {
					t.Fatal("codex lost on reload")
				}
				if again, err := tr2.Observe(ev); err != nil || len(again) != 0 || len(st2.OpenThoughts()) != 0 {
					t.Fatalf("after a restart the first fired again: %v (%v)", again, err)
				}
			})
		}
	}
}

func TestFirsts_ClassDataAndErrors(t *testing.T) {
	tbl, _ := ShippedFirsts()
	var corpse First
	for _, r := range tbl.Firsts {
		if r.ID == "corpsefire" {
			corpse = r
		}
	}
	sorc, _, _, _ := firstsRig(t, "sorceress", "")
	pal, _, _, _ := firstsRig(t, "paladin", "")
	if sorc.defaultFor(corpse) != "yes" || pal.defaultFor(corpse) != "no" {
		t.Fatalf("corpsefire default: sorceress %q paladin %q", sorc.defaultFor(corpse), pal.defaultFor(corpse))
	}
	cat, _ := mustCat(t).WithFirsts(tbl)
	st, _ := New(cat, "p1", &ManualClock{})
	_, err := NewTracker(tbl, cat, st, NewCodex("p1"), &ManualClock{}, Who{Class: "necromancer", Diff: "normal"})
	if err == nil || !strings.Contains(err.Error(), "necromancer") {
		t.Fatalf("unknown class: %v", err)
	}
	if _, err := NewTracker(tbl, mustCat(t), st, NewCodex("p1"), &ManualClock{}, Who{Class: "sorceress", Diff: "normal"}); err == nil {
		t.Fatal("a catalogue without the firsts point must be refused")
	}
	bad := *tbl
	bad.Firsts = append([]First(nil), tbl.Firsts...)
	bad.Firsts[0].DefaultByClass = map[string]string{"druid": "no"}
	if _, err := NewTracker(&bad, cat, st, NewCodex("p1"), &ManualClock{}, Who{Class: "sorceress", Diff: "normal"}); err == nil || !strings.Contains(err.Error(), "druid") {
		t.Fatalf("defaultByClass with an unknown class: %v", err)
	}
	if _, err := sorc.Observe(FirstEvent{Kind: "weather", Name: "rain"}); err == nil {
		t.Fatal("unknown event kind must be an error")
	}
	hard, err := NewTracker(tbl, cat, st, NewCodex("p1"), &ManualClock{}, Who{Class: "sorceress", Diff: "easy"})
	if err != nil {
		t.Fatal(err)
	}
	if _, err := hard.Observe(FirstEvent{Kind: "monster", Name: "Fallen"}); err == nil || !strings.Contains(err.Error(), "easy") {
		t.Fatalf("unknown difficulty: %v", err)
	}
}

func TestFirsts_SettleAnsweredAndFaded(t *testing.T) {
	tr, st, cx, clk := firstsRig(t, "sorceress", "")
	tr.Observe(FirstEvent{Kind: "monster", Name: "Fallen Shaman", Area: "Blood Moor"})
	tr.Observe(FirstEvent{Kind: "monster", Name: "Corpsefire", Area: "Den of Evil"})
	var shaman int
	for _, th := range st.OpenThoughts() {
		if strings.HasSuffix(th.Key, "first=fallen_shaman") || strings.Contains(th.Key, "first=fallen_shaman") {
			shaman = th.ID
		}
	}
	if shaman == 0 {
		t.Fatalf("no shaman thought in %v", st.OpenThoughts())
	}
	if err := st.Answer(shaman, "no"); err != nil {
		t.Fatal(err)
	}
	clk.Advance(2 * time.Minute) // the Corpsefire question fades unanswered
	if _, err := st.Tick(); err != nil {
		t.Fatal(err)
	}
	if err := tr.Settle(); err != nil {
		t.Fatal(err)
	}
	if e, _ := cx.Entry("fallen_shaman"); e.Decided != "no" || e.Source != "answered" || e.Open || e.Label != "kill whatever is nearest" {
		t.Fatalf("answered entry %+v", e)
	}
	if e, _ := cx.Entry("corpsefire"); e.Decided != "yes" || e.Source != "default" || e.Open {
		t.Fatalf("faded entry %+v", e)
	}
	// a first never teaches a rule: its key is answered once
	if r, _, _ := st.Stats(); r != 0 {
		t.Fatalf("%d rules learned from firsts", r)
	}
}

// A synthesized Act 1 run: the density goal is at least 8 firsts in the first 30 virtual minutes, at least 3 in the
// first 10 (the kill criterion of TASK-130), and a replay of the same run adds nothing.
func TestFirsts_ReplayDensity(t *testing.T) {
	type step struct {
		at time.Duration
		ev FirstEvent
	}
	min := func(m float64) time.Duration { return time.Duration(m * float64(time.Minute)) }
	run := []step{
		{min(1), FirstEvent{Kind: "area", Name: "Blood Moor", Clvl: 1}},
		{min(2), FirstEvent{Kind: "monster", Name: "Fallen", Area: "Blood Moor", Clvl: 2}},
		{min(2.2), FirstEvent{Kind: "monster", Name: "Fallen", Area: "Blood Moor", Clvl: 2}}, // same kind again
		{min(4), FirstEvent{Kind: "monster", Name: "Fallen Shaman", Area: "Blood Moor", Clvl: 2}},
		{min(6), FirstEvent{Kind: "monster", Name: "Quill Rat", Area: "Blood Moor", Clvl: 3}},
		{min(8), FirstEvent{Kind: "shrine", Name: "Armor Shrine", Area: "Blood Moor", Clvl: 3}},
		{min(11), FirstEvent{Kind: "area", Name: "Den of Evil", Clvl: 3}},
		{min(13), FirstEvent{Kind: "monster", Name: "Corpsefire", Area: "Den of Evil", Clvl: 4}},
		{min(19), FirstEvent{Kind: "area", Name: "Cold Plains", Clvl: 5}},
		{min(21), FirstEvent{Kind: "monster", Name: "Bishibosh", Area: "Cold Plains", Clvl: 5}},
		{min(23), FirstEvent{Kind: "champion", Name: "Champion", Mods: []string{"Fire Enchanted"}, Area: "Cold Plains", Clvl: 5}},
		{min(25), FirstEvent{Kind: "waypoint", Name: "Cold Plains", Clvl: 5}},
		{min(28), FirstEvent{Kind: "drop", Name: "rare", Clvl: 6}},
	}
	for _, class := range []string{"sorceress", "paladin"} {
		tr, _, cx, clk := firstsRig(t, class, "")
		in10, in30 := 0, 0
		for _, s := range run {
			clk.T = s.at
			fired, err := tr.Observe(s.ev)
			if err != nil {
				t.Fatal(err)
			}
			for range fired {
				if s.at <= min(10) {
					in10++
				}
				if s.at <= min(30) {
					in30++
				}
			}
		}
		if in10 < 3 || in30 < 8 {
			t.Fatalf("%s: %d firsts in 10 min, %d in 30 min", class, in10, in30)
		}
		n := len(cx.Entries())
		for _, s := range run {
			clk.T = s.at + time.Hour
			if fired, err := tr.Observe(s.ev); err != nil || len(fired) != 0 {
				t.Fatalf("replay fired %v (%v)", fired, err)
			}
		}
		if len(cx.Entries()) != n {
			t.Fatal("replay changed the codex")
		}
		t.Logf("%s: %d firsts in 10 min, %d in 30 min, %d entries", class, in10, in30, n)
	}
}

func TestCodex_FileErrors(t *testing.T) {
	dir := t.TempDir()
	if _, err := OpenCodex(dir, "../x"); err == nil {
		t.Fatal("bad profile accepted")
	}
	cx, _ := OpenCodex(dir, "p1")
	cx.add(CodexEntry{ID: "a", Kind: "monster", Source: "quiet"})
	if _, err := OpenCodex(dir, "p2"); err != nil {
		t.Fatal(err)
	}
}

// The repeat policy (cooldown, quiet) is per key and a first asks its own key once: it must never suppress a first.
func TestFirsts_NotSuppressedByRepeatPolicy(t *testing.T) {
	tr, st, cx, clk := firstsRig(t, "sorceress", "")
	st.SetRepeatPolicy(DefaultRepeatCooldown, DefaultQuietAfter)
	for _, n := range []string{"Fallen", "Fallen Shaman", "Quill Rat", "Corpsefire", "Bishibosh"} {
		clk.Advance(10 * time.Second)
		got, err := tr.Observe(FirstEvent{Kind: "monster", Name: n})
		if err != nil || len(got) != 1 || got[0].Decision == nil || got[0].Decision.Thought == nil {
			t.Fatalf("%s: %v %v", n, got, err)
		}
	}
	if len(st.OpenThoughts()) != 5 || len(cx.Entries()) != 5 {
		t.Fatalf("%d open, %d entries", len(st.OpenThoughts()), len(cx.Entries()))
	}
}
