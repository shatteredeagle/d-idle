package questboard

import (
	"encoding/json"
	"net/http"
	"strings"
	"testing"
	"time"
)

func (r *rig) desk(paused *bool) *Desk {
	st, err := Open(r.b, r.dir, r.hero)
	if err != nil {
		r.t.Fatal(err)
	}
	return NewDesk(r.b, DeskConfig{
		Exec:   func() (*Exec, *Store) { return r.x, st },
		Paused: func() bool { return paused != nil && *paused },
		Class:  func() string { return r.f.Class },
	})
}

func waitHand(t *testing.T, d *Desk) HandInView {
	t.Helper()
	for i := 0; i < 200; i++ {
		d.invalidate()
		if h := d.View().HandIn; h != nil && h.State != "running" {
			return *h
		}
		time.Sleep(10 * time.Millisecond)
	}
	t.Fatal("hand-in never finished")
	return HandInView{}
}

func TestDeskStartRunHandIn(t *testing.T) {
	for _, bd := range builds {
		t.Run(bd.class, func(t *testing.T) {
			r := newRig(t, bd.class, 6)
			d := r.desk(nil)
			v := d.View()
			if !v.Enabled || len(v.Offers) == 0 || v.Mode != "train" || !v.ModeActs {
				t.Fatalf("view: %+v", v)
			}
			for _, o := range v.Offers {
				if o.Pays != nil || !o.Startable || o.Readiness == "" {
					t.Errorf("offer %+v", o)
				}
			}
			a := d.Start(den, "")
			if !a.OK || a.Already || a.Chunk != clr || a.State != "accepted" || a.Status != 200 {
				t.Fatalf("start: %+v", a)
			}
			// double press: a no-op with already
			if a := d.Start(den, ""); !a.OK || !a.Already {
				t.Fatalf("second press: %+v", a)
			}
			// another quest while one runs: a refusal that names the running one
			b := d.Start("search_for_cain", "")
			if b.OK || b.Code != CodeBusy || !strings.Contains(b.Reason, den) || b.Status != 409 {
				t.Fatalf("busy: %+v", b)
			}
			if run := d.View().Running; run == nil || run.Quest != den {
				t.Fatalf("running: %+v", run)
			}
			// not awaiting yet: refuse with the state
			if h := d.HandIn(den, ""); h.OK || h.Code != CodeNotAwait {
				t.Fatalf("early hand-in: %+v", h)
			}
			r.tick(4)
			v = d.viewFresh()
			if v.Awaiting == nil || v.Strip == nil || v.Strip.NPC == "" || len(v.Needs) != 1 || v.Needs[0].Kind != "quest_handin" {
				t.Fatalf("awaiting: %+v", v)
			}
			if v.Needs[0].Actions[0].Post != "/api/quests/"+den+"/handin" {
				t.Fatalf("need action: %+v", v.Needs[0])
			}
			h := d.HandIn(den, "")
			if !h.OK || !h.Pending || h.Status != http.StatusAccepted {
				t.Fatalf("hand-in: %+v", h)
			}
			if got := waitHand(t, d); got.State != "done" {
				t.Fatalf("hand-in result: %+v", got)
			}
			if again := d.HandIn(den, ""); !again.OK || !again.Already {
				t.Fatalf("repeat hand-in: %+v", again)
			}
			if r.f.Count("handin") != 1 {
				t.Fatalf("dialogue ran %d times", r.f.Count("handin"))
			}
			v = d.viewFresh()
			if len(v.History) != 1 || v.History[0].Quest != den || v.Awaiting != nil {
				t.Fatalf("history: %+v", v.History)
			}
		})
	}
}

func (d *Desk) viewFresh() View { d.invalidate(); return d.View() }

func TestDeskRefusals(t *testing.T) {
	r := newRig(t, "Sorceress", 5)
	paused := true
	d := r.desk(&paused)
	if a := d.Start(den, ""); a.OK || a.Code != CodePaused || !strings.Contains(a.Reason, "paused") {
		t.Fatalf("paused: %+v", a)
	}
	if a := d.Start("nope", ""); a.OK || a.Code != CodeUnknown || a.Status != 404 {
		t.Fatalf("unknown quest: %+v", a)
	}
	if a := d.Start(den, "zzz"); a.OK || a.Code != CodeUnknown {
		t.Fatalf("unknown chunk: %+v", a)
	}
	paused = false
	// Blood Raven is too hard for a clvl 5 Sorceress (floor 8): refused with the reason, nothing accepted
	a := d.Start(brq, "")
	if a.OK || a.Code != CodeNotReady || a.Reason == "" {
		t.Fatalf("not ready: %+v", a)
	}
	if st, _ := r.x.State(); st.Active != nil {
		t.Fatalf("a refused start left a record: %+v", st.Active)
	}
	// a cancel with nothing running is a no-op
	if a := d.Cancel(den, ""); !a.OK || !a.Already {
		t.Fatalf("cancel idle: %+v", a)
	}
	if a := d.Start(den, ""); !a.OK {
		t.Fatalf("start: %+v", a)
	}
	paused = true
	r.tick(4)
	if a := d.HandIn(den, ""); a.OK || a.Code != CodePaused {
		t.Fatalf("hand-in while paused: %+v", a)
	}
	paused = false
	if a := d.Cancel(den, ""); !a.OK || a.Already || a.State != "ready" {
		t.Fatalf("cancel: %+v", a)
	}
	if a := d.Cancel(den, ""); !a.OK || !a.Already {
		t.Fatalf("second cancel: %+v", a)
	}
}

func TestDeskOffAndNoHero(t *testing.T) {
	b := mustBoard(t)
	none := NewDesk(b, DeskConfig{Exec: func() (*Exec, *Store) { return nil, nil }})
	if a := none.Start(den, ""); a.OK || a.Code != CodeNoHero || a.Status != 503 {
		t.Fatalf("no hero: %+v", a)
	}
	if v := none.View(); v.Enabled || v.Reason == "" || len(v.Offers) != 0 {
		t.Fatalf("no hero view: %+v", v)
	}
	r := newRig(t, "Paladin", 6)
	r.x.SetEnabled(false)
	d := r.desk(nil)
	a := d.Start(den, "")
	if a.OK || a.Code != CodeOff || !strings.Contains(a.Reason, "quests.board") {
		t.Fatalf("off: %+v", a)
	}
	if v := d.View(); v.Enabled || !strings.Contains(v.Reason, "quests.board") {
		t.Fatalf("off view: %+v", v)
	}
	if a := d.SetMode("farm"); !a.OK {
		t.Fatalf("mode while off: %+v", a) // the mode is the player's setting, kept while the board is off
	}
}

func TestDeskModeAndRestartPersistence(t *testing.T) {
	r := newRig(t, "Paladin", 6)
	d := r.desk(nil)
	if a := d.SetMode("sprint"); a.OK || a.Code != CodeBadMode || a.Status != 400 {
		t.Fatalf("bad mode: %+v", a)
	}
	if a := d.SetMode("farm"); !a.OK || a.Mode != "farm" {
		t.Fatalf("mode: %+v", a)
	}
	if a := d.SetMode("farm"); !a.OK || !a.Already {
		t.Fatalf("same mode: %+v", a)
	}
	if a := d.Start(den, ""); !a.OK {
		t.Fatal(a)
	}
	// the process restarts: a new Exec and desk over the same files see the accepted record and the mode
	r.restart()
	d2 := r.desk(nil)
	v := d2.View()
	if v.Mode != "farm" || v.Running == nil || v.Running.Quest != den {
		t.Fatalf("after restart: mode %s running %+v", v.Mode, v.Running)
	}
	if a := d2.Start(den, ""); !a.OK || !a.Already {
		t.Fatalf("press after restart: %+v", a)
	}
	if a := d2.Cancel(den, ""); !a.OK {
		t.Fatal(a)
	}
	if a := d2.Start(den, ""); !a.OK {
		t.Fatal(a)
	}
	// the new record carries the stored mode
	st, _ := Open(r.b, r.dir, r.hero)
	p, _, _ := st.Load()
	if p.Mode != "farm" || len(p.Accepted) != 1 || p.Accepted[0].Mode != "farm" {
		t.Fatalf("plan mode %q records %+v", p.Mode, p.Accepted)
	}
}

func TestDeskViewJSONShape(t *testing.T) {
	r := newRig(t, "Paladin", 6)
	raw, err := json.Marshal(r.desk(nil).View())
	if err != nil {
		t.Fatal(err)
	}
	var m map[string]any
	_ = json.Unmarshal(raw, &m)
	for _, k := range []string{"schema", "enabled", "mode", "modeActs", "autoStart", "offers", "quests", "history", "needs"} {
		if _, ok := m[k]; !ok {
			t.Errorf("missing %q", k)
		}
	}
	if strings.Contains(string(raw), `"pays"`) {
		t.Error("pays must be omitted until the wallet exists")
	}
	if s := string(raw); strings.Contains(s, "margin") || strings.Contains(s, "chapter ") {
		t.Errorf("numeric margin or chapter text in the view: %s", s)
	}
}

func TestDeskViewCached(t *testing.T) {
	r := newRig(t, "Paladin", 6)
	now := time.Unix(1000, 0)
	st, _ := Open(r.b, r.dir, r.hero)
	d := NewDesk(r.b, DeskConfig{Exec: func() (*Exec, *Store) { return r.x, st }, Now: func() time.Time { return now }})
	v1 := d.View()
	r.f.DoneQuest[den] = TierMemory
	if v2 := d.View(); len(v2.Quests) != len(v1.Quests) || v2.Generated != v1.Generated {
		t.Fatal("the second read within 1 s must be the cached one")
	}
	now = now.Add(2 * time.Second)
	if d.View().Generated.Equal(v1.Generated) {
		t.Fatal("the cache must expire")
	}
}

// An accepted chunk the hero has not started never reads as silent: the strip and the card say why (the bot's reason
// when it has one, else the generic queued line).
func TestDeskQueuedReason(t *testing.T) {
	for _, bd := range builds {
		t.Run(bd.class, func(t *testing.T) {
			r := newRig(t, bd.class, 6)
			d := r.desk(nil)
			if a := d.Start(den, ""); !a.OK {
				t.Fatalf("start: %+v", a)
			}
			v := d.viewFresh()
			if v.Running == nil || v.Strip == nil || !strings.Contains(v.Strip.Queued, "Queued") || v.Running.Queued != v.Strip.Queued {
				t.Fatalf("queued line missing: %+v %+v", v.Running, v.Strip)
			}
			d.cfg.Queued = func() string { return "Starts when the fight is over." }
			if v := d.viewFresh(); v.Strip.Queued != "Starts when the fight is over." {
				t.Fatalf("bot reason: %+v", v.Strip)
			}
		})
	}
}

// Sandbox 2026-11-20: Stop then Confirm on a chunk of a plan file with two accepted chunks changed nothing.
func TestDeskStopOfOneOfTwoAcceptedChunks(t *testing.T) {
	const cain = "search_for_cain"
	for _, class := range []string{"Paladin", "Sorceress"} {
		t.Run(class, func(t *testing.T) {
			r := newRig(t, class, 17)
			paused := false
			d := r.desk(&paused)
			if a := d.Start(cain, "scroll"); !a.OK {
				t.Fatalf("%+v", a)
			}
			p, _, _ := r.x.st.Load()
			p.Accepted = append(p.Accepted, Record{Quest: cain, Chunk: "stones", Diff: "normal"})
			if err := r.x.st.file.Save(p); err != nil {
				t.Fatal(err)
			}
			a := d.Cancel(cain, "stones")
			if !a.OK || a.Already || a.State != "ready" {
				t.Fatalf("stop stones: %+v", a)
			}
			st, _ := r.x.State()
			wantExtra := 0
			if testSM { // D4: the load added the missing middle part read
				wantExtra = 1
			}
			if st.Active == nil || st.Active.Chunk != "scroll" || len(st.Extra) != wantExtra {
				t.Fatalf("%+v", st)
			}
			if a := d.Cancel(cain, "scroll"); !a.OK || a.Already {
				t.Fatalf("stop scroll: %+v", a)
			}
			if st, _ := r.x.State(); st.Active != nil {
				t.Fatalf("still accepted: %+v", st.Active)
			}
		})
	}
}

// A chunk the readiness word holds back (too hard) shows the model's sentence on the sheet row, not only "too hard for
// now" (card andariel1: Andariel was never offered and the board said nothing useful).
func TestDeskTooHardRowSaysWhy(t *testing.T) {
	const quest = "sisters_to_the_slaughter"
	for _, bd := range builds {
		t.Run(bd.class, func(t *testing.T) {
			r := newRig(t, bd.class, 6)
			d := r.desk(nil)
			row := func() ChunkRow {
				for _, q := range d.viewFresh().Quests {
					if q.ID == quest {
						return q.Chunks[0]
					}
				}
				t.Fatal("no quest row")
				return ChunkRow{}
			}
			c := row()
			if c.Readiness != ReadyTooHard || c.Why != "too hard for now" {
				t.Fatalf("without the model: %+v", c)
			}
			d.cfg.Explain = func(q Quest, c Chunk) (Readiness, bool) {
				return Readiness{Word: ReadyTooHard, Sentence: "You have no merc and would likely lose to Andariel: too hard"}, true
			}
			if c := row(); !strings.Contains(c.Why, "would likely lose to Andariel") {
				t.Fatalf("with the model: %+v", c)
			}
		})
	}
}

// akarafix: a running hand-in the loop has not served says why (reason.code handin_waiting); served: the generic code.
func TestDeskHandInWaitReason(t *testing.T) {
	d := &Desk{cfg: DeskConfig{HandInWhy: func() string { return "the hero is out in the field." }}}
	r := d.handInWaitReason()
	if r == nil || r.Code != CodeHandInWaiting || !strings.Contains(r.Text, "out in the field") {
		t.Fatalf("reason: %+v", r)
	}
	d.cfg.HandInWhy = func() string { return "" }
	if d.handInWaitReason() != nil {
		t.Fatal("a reason for a tap that is being served")
	}
}
