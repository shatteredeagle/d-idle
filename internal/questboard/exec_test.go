package questboard

import (
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// testSM runs the whole executor suite on the run machine: QUESTBOARD_SM=1 go test ./internal/questboard. A test that
// states an old behaviour the machine changes on purpose (design section 3.7) skips itself with skipSM.
var testSM = os.Getenv("QUESTBOARD_SM") == "1"

func skipSM(t *testing.T, why string) {
	t.Helper()
	if testSM {
		t.Skip("changed on purpose by the run machine: " + why)
	}
}

type rig struct {
	t    *testing.T
	b    *Board
	f    *Fake
	dir  string
	hero string
	x    *Exec
	sm   bool   // run this rig's Exec on the run machine whatever QUESTBOARD_SM says
	old  bool   // run this rig's Exec on the legacy board whatever QUESTBOARD_SM says (wins over sm)
	led  Ledger // the account wallet of the rig (nil: none); see ledger_test.go
}

func newRig(t *testing.T, class string, clvl int) *rig { return newRigSM(t, class, clvl, false) }

func newRigSM(t *testing.T, class string, clvl int, sm bool) *rig {
	t.Helper()
	b := mustBoard(t)
	r := &rig{t: t, b: b, f: NewFake(b, class, clvl), dir: t.TempDir(), hero: "Testy", sm: sm}
	r.restart()
	return r
}

// restart drops the Exec (the process) and builds a new one over the same world and plan file.
func (r *rig) restart() {
	st, err := Open(r.b, r.dir, r.hero)
	if err != nil {
		r.t.Fatal(err)
	}
	r.x = NewExec(r.b, st, r.f, r.f, Options{Enabled: true, SM: (testSM || r.sm) && !r.old, Ledger: func() Ledger { return r.led }})
}

func (r *rig) tick(n int) Decision {
	r.t.Helper()
	var d Decision
	for i := 0; i < n; i++ {
		var err error
		if d, err = r.x.Tick(context.Background(), r.f.Step); err != nil {
			r.t.Fatal(err)
		}
	}
	return d
}

func (r *rig) chunk(q, c string) ChunkView {
	r.t.Helper()
	st, err := r.x.State()
	if err != nil {
		r.t.Fatal(err)
	}
	_, cv, ok := chunkView(st.Views, q, c)
	if !ok {
		r.t.Fatalf("no chunk %s/%s", q, c)
	}
	return cv
}

func (r *rig) next() Decision {
	r.t.Helper()
	d, err := r.x.Next()
	if err != nil {
		r.t.Fatal(err)
	}
	return d
}

func offered(offers []Offer, q, c string) bool {
	for _, o := range offers {
		if o.Quest == q && o.Chunk == c {
			return true
		}
	}
	return false
}

const (
	den = "den_of_evil"
	clr = "clear_den"
	brq = "sisters_burial_grounds"
	brc = "blood_raven"
)

// two builds: a melee Paladin and a caster Sorceress (families in data/questboard/families.json).
var builds = []struct {
	class      string
	burialSeen bool // offered Blood Raven at clvl 5 (Paladin floor 7: risky at 5; Sorceress floor 8: too hard)
}{{"Paladin", true}, {"Sorceress", false}}

func TestOffersDifferPerBuild(t *testing.T) {
	for _, bd := range builds {
		r := newRig(t, bd.class, 5)
		off, err := r.x.Offers()
		if err != nil {
			t.Fatal(err)
		}
		if !offered(off, den, clr) {
			t.Errorf("%s: Den not offered at clvl 5", bd.class)
		}
		if got := offered(off, brq, brc); got != bd.burialSeen {
			t.Errorf("%s: Blood Raven offered = %v, want %v", bd.class, got, bd.burialSeen)
		}
	}
}

func TestAcceptRunAwaitHandIn(t *testing.T) {
	for _, bd := range builds {
		t.Run(bd.class, func(t *testing.T) {
			r := newRig(t, bd.class, 6)
			if d := r.next(); d.Kind != DecIdle {
				t.Fatalf("nothing accepted: %v", d)
			}
			if err := r.x.Accept(den, clr); err != nil {
				t.Fatal(err)
			}
			if got := r.chunk(den, clr).State; got != Accepted {
				t.Fatalf("after Start: %s", got)
			}
			// one chunk at a time: a second Start is refused and names the running one
			err := r.x.Accept("search_for_cain", "scroll")
			if err == nil || !strings.Contains(err.Error(), den+"/"+clr) {
				t.Fatalf("second concurrent chunk: %v", err)
			}
			off, _ := r.x.Offers()
			for _, o := range off {
				if o.Startable {
					t.Errorf("%s/%s startable while another chunk runs", o.Quest, o.Chunk)
				}
			}
			r.tick(1)
			if got := r.chunk(den, clr); got.State != InProgress || got.Step != 1 {
				t.Fatalf("after 1 slice: %+v", got)
			}
			r.tick(3)
			// the bot stops at the NPC: awaiting, and a tick runs no more handler work
			if d := r.next(); d.Kind != DecAwait {
				t.Fatalf("objective met: %v", d)
			}
			before := r.f.Count("step")
			r.tick(5)
			if r.f.Count("step") != before || r.f.Count("handin") != 0 {
				t.Fatalf("the bot acted while awaiting: %v", r.f.Calls)
			}
			// handing in by itself never happens; the tap does
			res, err := r.x.HandIn(context.Background(), den, clr)
			if err != nil || res.Outcome != HandInDone || res.Attempts != 1 {
				t.Fatalf("hand in: %+v %v", res, err)
			}
			if len(res.Awards) != 1 || res.Awards[0].Key != "quest/den_of_evil/-/normal" {
				t.Fatalf("awards: %+v", res.Awards)
			}
			again, err := r.x.HandIn(context.Background(), den, clr)
			if err != nil || !again.Already || r.f.Count("handin") != 1 {
				t.Fatalf("repeated tap: %+v %v calls %v", again, err, r.f.Calls)
			}
			if d := r.next(); d.Kind != DecIdle {
				t.Fatalf("after the hand-in: %v", d)
			}
			// the finished record no longer blocks the next Start
			if err := r.x.Accept("search_for_cain", "scroll"); err != nil && !strings.Contains(err.Error(), "not ready") {
				t.Fatalf("next chunk: %v", err)
			}
		})
	}
}

func TestRestartAtEveryStep(t *testing.T) {
	for _, bd := range builds {
		for k := 0; k <= 4; k++ {
			r := newRig(t, bd.class, 6)
			if err := r.x.Accept(den, clr); err != nil {
				t.Fatal(err)
			}
			r.tick(k)
			want := r.chunk(den, clr)
			r.restart() // handler progress is never persisted: the world is the evidence
			got := r.chunk(den, clr)
			if want.State != got.State || want.Step != got.Step {
				t.Fatalf("%s restart after %d slices: %s/%d became %s/%d", bd.class, k, want.State, want.Step, got.State, got.Step)
			}
			r.tick(4 - k)
			if d := r.next(); d.Kind != DecAwait {
				t.Fatalf("%s k=%d: %v", bd.class, k, d)
			}
			if n := r.f.Count("step"); n != 4 {
				t.Fatalf("%s k=%d: %d slices in total, want 4 (no redo)", bd.class, k, n)
			}
			r.restart() // and again while awaiting
			if r.next().Kind != DecAwait {
				t.Fatalf("%s k=%d: restart while awaiting lost the stop", bd.class, k)
			}
			if res, err := r.x.HandIn(context.Background(), den, clr); err != nil || res.Outcome != HandInDone {
				t.Fatalf("%s k=%d: %+v %v", bd.class, k, res, err)
			}
			r.restart()
			if got := r.chunk(den, clr); got.State != Done || !got.Confirmed {
				t.Fatalf("%s k=%d: restart after hand-in: %+v", bd.class, k, got)
			}
		}
	}
}

func TestHandInReadBack(t *testing.T) {
	ctx := context.Background()
	ready := func(t *testing.T) *rig {
		r := newRig(t, "Paladin", 6)
		if err := r.x.Accept(den, clr); err != nil {
			t.Fatal(err)
		}
		r.tick(4)
		return r
	}
	t.Run("not awaiting is refused without a dialogue", func(t *testing.T) {
		r := newRig(t, "Paladin", 6)
		if err := r.x.Accept(den, clr); err != nil {
			t.Fatal(err)
		}
		r.tick(2)
		if _, err := r.x.HandIn(ctx, den, clr); err == nil || r.f.Count("handin") != 0 {
			t.Fatalf("err %v calls %v", err, r.f.Calls)
		}
	})
	t.Run("silent dialogue is retried then lands", func(t *testing.T) {
		r := ready(t)
		r.f.HandInSilent, r.f.HandInErrs = 1, 1
		res, err := r.x.HandIn(ctx, den, clr)
		if err != nil || res.Outcome != HandInDone || res.Attempts != 3 {
			t.Fatalf("%+v %v", res, err)
		}
	})
	t.Run("proxy only is confirming and pays nothing until exact", func(t *testing.T) {
		r := ready(t)
		r.f.HandInTier = TierProxy
		res, err := r.x.HandIn(ctx, den, clr)
		if err != nil || res.Outcome != HandInConfirming || len(res.Awards) != 0 {
			t.Fatalf("%+v %v", res, err)
		}
		r.f.DoneQuest[den] = TierSave // the save catches up
		cv := r.chunk(den, clr)
		if cv.State != Done || !cv.Confirmed {
			t.Fatalf("%+v", cv)
		}
		st, _ := r.x.State()
		for _, v := range st.Views {
			if v.ID == den && len(v.Awards) != 1 {
				t.Fatalf("award after confirmation: %+v", v.Awards)
			}
		}
	})
	t.Run("reward item in the pack but memory says not done: still awaiting", func(t *testing.T) {
		r := ready(t)
		r.f.ChunkDone[ChunkKey(den, clr)] = TierProxy
		if got := r.chunk(den, clr).State; got != Awaiting {
			t.Fatalf("T2 'not done' must beat the proxy: %s", got)
		}
		if res, err := r.x.HandIn(ctx, den, clr); err != nil || res.Outcome != HandInDone {
			t.Fatalf("%+v %v", res, err)
		}
	})
	t.Run("every attempt fails: parked with a reason, try again clears it", func(t *testing.T) {
		r := ready(t)
		r.f.HandInSilent = 9
		res, err := r.x.HandIn(ctx, den, clr)
		if err != nil || res.Outcome != HandInBlocked || res.Attempts != 3 || !strings.Contains(res.Why, "3 times") {
			t.Fatalf("%+v %v", res, err)
		}
		if d := r.next(); d.Kind != DecBlocked || d.Why == "" {
			t.Fatalf("%v", d)
		}
		if got := r.chunk(den, clr); got.State != Blocked {
			t.Fatalf("%+v", got)
		}
		r.restart() // a restart forgets the park and the chunk is tried again from the evidence
		r.f.HandInSilent = 0
		if testSM { // Q5: with the run machine a park survives the restart until the player retries
			if d := r.next(); d.Kind != DecBlocked {
				t.Fatalf("after restart: %v", d)
			}
			r.x.Unblock(den, clr)
		}
		if d := r.next(); d.Kind != DecAwait {
			t.Fatalf("after restart: %v", d)
		}
		r.f.HandInSilent = 9
		r.x.HandIn(ctx, den, clr)
		r.x.Unblock(den, clr)
		r.f.HandInSilent = 0
		if res, err := r.x.HandIn(ctx, den, clr); err != nil || res.Outcome != HandInDone {
			t.Fatalf("%+v %v", res, err)
		}
	})
}

func TestFailureDeathParkResume(t *testing.T) {
	fail := StepResult{StepFailed, "no path to the Den", ""}
	t.Run("failures park with a visible reason, no silent loop", func(t *testing.T) {
		r := newRig(t, "Sorceress", 6)
		r.x.Accept(den, clr)
		r.f.Script = []StepResult{fail, fail, fail}
		r.tick(3)
		d := r.next()
		if d.Kind != DecBlocked || !strings.Contains(d.Why, "gave up after 3 tries: no path to the Den") {
			t.Fatalf("%v", d)
		}
		n := r.f.Count("step")
		r.tick(5)
		if r.f.Count("step") != n {
			t.Fatal("a parked chunk kept running")
		}
		st, _ := r.x.State()
		if st.Active == nil || st.Active.State != Blocked || st.Active.Why == "" {
			t.Fatalf("Needs-you row: %+v", st.Active)
		}
		r.x.Unblock(den, clr)
		r.tick(4)
		if r.next().Kind != DecAwait {
			t.Fatal("did not resume after Unblock")
		}
	})
	t.Run("a success resets the failure count", func(t *testing.T) {
		r := newRig(t, "Paladin", 6)
		r.x.Accept(den, clr)
		r.f.Script = []StepResult{fail, fail, {Outcome: StepRan}, fail, fail}
		r.tick(5)
		if r.next().Kind != DecRun {
			t.Fatal("parked although never 3 in a row")
		}
	})
	t.Run("death resumes after recovery, a second death parks", func(t *testing.T) {
		r := newRig(t, "Paladin", 6)
		r.x.Accept(den, clr)
		r.f.Script = []StepResult{{Outcome: StepRan}, {StepDied, "Fallen pack", ""}}
		r.tick(2)
		if d := r.next(); d.Kind != DecRun {
			t.Fatalf("after one death: %v", d)
		}
		r.restart() // the bot restarts while recovering: progress comes from the evidence
		r.f.Script = []StepResult{{StepDied, "Fallen pack", ""}}
		r.tick(1)
		if !testSM { // the machine kept the first death over the restart (card fix2): the second one parked already
			r.x.Report(Decision{Quest: den, Chunk: clr}, StepResult{StepDied, "again", ""})
		}
		d := r.next()
		if d.Kind != DecBlocked || !strings.Contains(d.Why, "died 2 times") {
			t.Fatalf("%v", d)
		}
	})
	t.Run("waiting is not a failure", func(t *testing.T) {
		r := newRig(t, "Paladin", 6)
		r.x.Accept(den, clr)
		r.f.Script = []StepResult{{Outcome: StepWaiting}, {Outcome: StepWaiting}, {Outcome: StepWaiting}, {Outcome: StepWaiting}}
		r.tick(4)
		if r.next().Kind != DecRun {
			t.Fatal("waiting parked the chunk")
		}
	})
}

func TestResetPauseCancel(t *testing.T) {
	r := newRig(t, "Paladin", 6)
	r.x.Accept(den, clr)
	r.f.Script = []StepResult{{Outcome: StepRan}, {StepFailed, "x", ""}, {StepFailed, "x", ""}}
	r.tick(3)
	r.x.Reset()
	if r.f.Count("stop") != 1 {
		t.Fatalf("Reset did not stop the running chunk: %v", r.f.Calls)
	}
	if got := r.chunk(den, clr); got.State != Accepted && got.State != InProgress {
		t.Fatalf("Reset must keep the record accepted: %+v", got)
	}
	st, _ := r.x.State()
	if st.Active == nil || st.Active.Failures != 0 {
		t.Fatalf("Reset keeps counters: %+v", st.Active)
	}
	r.x.Pause(true)
	n := r.f.Count("step")
	if d := r.tick(3); d.Kind != DecPaused || r.f.Count("step") != n {
		t.Fatalf("paused: %v", d)
	}
	r.x.Pause(false)
	r.tick(3)
	if r.next().Kind != DecAwait {
		t.Fatal("did not continue after Resume")
	}
	r.x.Pause(true) // a chunk awaiting its tap stays awaiting
	if r.next().Kind != DecAwait {
		t.Fatal("pause hid the awaiting state")
	}
	r.x.Pause(false)

	// Cancel is the player's Stop: record gone, back to ready, one stop call
	if err := r.x.Cancel(den, clr); err != nil {
		t.Fatal(err)
	}
	if got := r.chunk(den, clr); got.State != Ready {
		t.Fatalf("after Stop: %+v", got)
	}
	if r.next().Kind != DecIdle {
		t.Fatal("not idle after Stop")
	}
}

func TestKnobOffRefusesEverything(t *testing.T) {
	r := newRig(t, "Paladin", 6)
	r.x.SetEnabled(false)
	os.Remove(filepath.Join(r.dir, r.hero+".quests.json")) // the rig's own first derive wrote the baseline
	if _, err := r.x.Offers(); err != ErrOff {
		t.Fatalf("offers: %v", err)
	}
	if err := r.x.Accept(den, clr); err != ErrOff {
		t.Fatalf("accept: %v", err)
	}
	if _, err := r.x.HandIn(context.Background(), den, clr); err != ErrOff {
		t.Fatalf("handin: %v", err)
	}
	if _, err := r.x.State(); err != ErrOff {
		t.Fatalf("state: %v", err)
	}
	if _, err := r.x.Next(); err != ErrOff {
		t.Fatalf("next: %v", err)
	}
	if _, err := r.x.Tick(context.Background(), r.f.Step); err != ErrOff || len(r.f.Calls) != 0 {
		t.Fatalf("tick: %v %v", err, r.f.Calls)
	}
	if _, err := os.Stat(filepath.Join(r.dir, r.hero+".quests.json")); err == nil {
		t.Fatal("a plan file was written with the knob off")
	}
}

func TestSwitchOffKeepsRecords(t *testing.T) {
	r := newRig(t, "Paladin", 6)
	r.x.Accept(den, clr)
	r.tick(2)
	r.x.SetEnabled(false)
	r.x.SetEnabled(true)
	if got := r.chunk(den, clr); got.State != InProgress && got.State != Accepted || got.Step != 2 {
		t.Fatalf("%+v", got)
	}
}

// Sandbox 2026-11-20: a plan file with two accepted Cain chunks parked both. The board now runs them one at a
// time in chain order whatever the tap order, and the second waits with a visible reason.
func TestTwoAcceptedRecordsRunInChainOrder(t *testing.T) {
	for _, class := range []string{"Paladin", "Sorceress"} {
		t.Run(class, func(t *testing.T) {
			const cain = "search_for_cain"
			r := newRig(t, class, 17)
			if err := r.x.Accept(cain, "scroll"); err != nil {
				t.Fatal(err)
			}
			p, _, _ := r.x.st.Load()
			// stones first in the file: the file order must not decide
			p.Accepted = append([]Record{{Quest: cain, Chunk: "stones", Diff: "normal"}}, p.Accepted...)
			if err := r.x.st.file.Save(p); err != nil {
				t.Fatal(err)
			}
			d, err := r.x.Next()
			if err != nil || d.Kind != DecRun || d.Quest != cain || d.Chunk != "scroll" {
				t.Fatalf("%+v %v", d, err)
			}
			st, err := r.x.State()
			// with the machine the missing middle part (read) is added at load (D4), so two chunks wait
			wantExtra := 1
			if testSM {
				wantExtra = 2
			}
			if err != nil || st.Active == nil || st.Active.Chunk != "scroll" || len(st.Extra) != wantExtra || st.Extra[wantExtra-1].Chunk != "stones" {
				t.Fatalf("%+v %v", st, err)
			}
			if w := WaitsFor(Ref{st.Active.Quest, st.Active.Chunk}); w != "waits for "+cain+"/scroll" {
				t.Fatal(w)
			}
		})
	}
}

func TestStartCoversOnlyTheCurrentChunk(t *testing.T) {
	r := newRig(t, "Paladin", 14)
	const cain = "search_for_cain"
	// boardprereq: a tap on a later part accepts the earlier steps it waits for, in order (never a lone later record)
	early := newRig(t, "Paladin", 14)
	if err := early.x.Accept(cain, "read"); err != nil {
		t.Fatalf("a later chunk accepts its prerequisite: %v", err)
	}
	if d := early.next(); d.Kind != DecRun || d.Chunk != "scroll" {
		t.Fatalf("the prerequisite runs first: %+v", d)
	}
	if err := r.x.Accept(cain, "scroll"); err != nil {
		t.Fatal(err)
	}
	r.tick(2) // item chunk: done when the scroll is in the pack
	if got := r.chunk(cain, "scroll").State; got != Done {
		t.Fatalf("scroll: %s", got)
	}
	if r.next().Kind != DecIdle {
		t.Fatal("the bot ran on into the next chunk without a tap")
	}
	off, _ := r.x.Offers()
	if !offered(off, cain, "read") {
		t.Fatalf("the next part raises its own offer: %+v", off)
	}
	if err := r.x.Accept(cain, "read"); err != nil {
		t.Fatalf("the finished first part must not block the second: %v", err)
	}
}

func TestUnknownIDsNameThemselves(t *testing.T) {
	r := newRig(t, "Paladin", 6)
	if err := r.x.Accept("nope", "x"); err == nil || !strings.Contains(err.Error(), "nope") {
		t.Fatalf("%v", err)
	}
	if _, err := r.x.HandIn(context.Background(), den, "nope"); err == nil || !strings.Contains(err.Error(), "nope") {
		t.Fatalf("%v", err)
	}
	if err := r.x.Cancel("nope", "x"); err == nil {
		t.Fatal("cancel of an unknown quest")
	}
	// an unknown class never falls back to the Sorceress
	f := NewFake(r.b, "Bard", 6)
	if _, err := r.b.LegacyReadiness(f.Class, 6); err == nil || !strings.Contains(err.Error(), "Bard") {
		t.Fatalf("%v", err)
	}
}

// Sandbox 2026-11-20 (boardprereq): stones accepted, its prerequisite scroll step not (the player stopped it). The
// board must not park for ever: it starts the scroll step itself, and a tap on the later chunk accepts the chain.
func TestPrerequisiteChainIsAccepted(t *testing.T) {
	const cain = "search_for_cain"
	accepted := func(r *rig) []string {
		p, _, _ := r.x.st.Load()
		var out []string
		for _, a := range p.Accepted {
			out = append(out, a.Chunk)
		}
		return out
	}
	for _, class := range []string{"Paladin", "Sorceress"} {
		t.Run(class+"/tap", func(t *testing.T) {
			r := newRig(t, class, 17)
			if err := r.x.Accept(cain, "stones"); err != nil {
				t.Fatal(err)
			}
			if got := accepted(r); strings.Join(got, ",") != "scroll,read,stones" {
				t.Fatalf("records %v", got)
			}
			if d := r.next(); d.Kind != DecRun || d.Chunk != "scroll" {
				t.Fatalf("%+v", d)
			}
		})
		t.Run(class+"/repair", func(t *testing.T) {
			r := newRig(t, class, 17)
			p, _, _ := r.x.st.Load()
			p.Accepted = append(p.Accepted, Record{Quest: cain, Chunk: "stones", Diff: "normal"})
			if err := r.x.st.file.Save(p); err != nil {
				t.Fatal(err)
			}
			d := r.next()
			if d.Kind != DecRun || d.Chunk != "scroll" {
				t.Fatalf("not repaired: %+v", d)
			}
			if got := accepted(r); strings.Join(got, ",") != "stones,scroll,read" {
				t.Fatalf("records %v", got)
			}
			if d := r.next(); d.Kind != DecRun || d.Chunk != "scroll" || len(accepted(r)) != 3 {
				t.Fatalf("second call changed it: %+v %v", d, accepted(r))
			}
		})
		t.Run(class+"/stop", func(t *testing.T) {
			r := newRig(t, class, 17)
			if err := r.x.Accept(cain, "stones"); err != nil {
				t.Fatal(err)
			}
			if deps := r.x.Dependants(cain, "scroll"); len(deps) != 2 || deps[0] != "read" || deps[1] != "stones" {
				t.Fatalf("dependants %v", deps)
			}
			if err := r.x.Cancel(cain, "stones"); err != nil || len(accepted(r)) != 2 {
				t.Fatalf("stones stop: %v %v", err, accepted(r))
			}
			if err := r.x.Accept(cain, "stones"); err != nil {
				t.Fatal(err)
			}
			if len(accepted(r)) != 3 {
				t.Fatalf("%v", accepted(r))
			}
			if err := r.x.Cancel(cain, "scroll"); err != nil || len(accepted(r)) != 0 {
				t.Fatalf("scroll stop must withdraw stones: %v %v", err, accepted(r))
			}
			if d := r.next(); d.Kind != DecIdle {
				t.Fatalf("%+v", d)
			}
		})
		t.Run(class+"/single", func(t *testing.T) {
			r := newRig(t, class, 17)
			if err := r.x.Accept(cain, "scroll"); err != nil || len(accepted(r)) != 1 {
				t.Fatalf("%v %v", err, accepted(r))
			}
			if d := r.next(); d.Kind != DecRun || d.Chunk != "scroll" {
				t.Fatalf("%+v", d)
			}
		})
	}
}
