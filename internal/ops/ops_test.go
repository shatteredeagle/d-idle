package ops

import (
	"errors"
	"fmt"
	"net"
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"sync"
	"testing"
	"time"

	"d2lod-bot/internal/questboard"
)

// fakeDesk records the calls and answers from canned data. The quest board is class-agnostic (it reads the hero's
// class only through the board data), so no second class is needed here.
type fakeDesk struct {
	mu    sync.Mutex
	calls []string
	view  questboard.View
	rep   questboard.Reply
	rec   []questboard.AuditEvent
}

func (f *fakeDesk) call(s string) questboard.Reply {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.calls = append(f.calls, s)
	r := f.rep
	if r.Status == 0 {
		r.Status = 200
	}
	return r
}
func (f *fakeDesk) View() questboard.View              { f.mu.Lock(); defer f.mu.Unlock(); return f.view }
func (f *fakeDesk) Start(q, c string) questboard.Reply { return f.call("start " + q + " " + c) }
func (f *fakeDesk) StartAnyway(q, c string) questboard.Reply {
	r := f.call("try " + q + " " + c)
	r.Override = "too hard for now"
	return r
}
func (f *fakeDesk) HandIn(q, c string) questboard.Reply { return f.call("handin " + q + " " + c) }
func (f *fakeDesk) Cancel(q, c string) questboard.Reply { return f.call("cancel " + q + " " + c) }
func (f *fakeDesk) Retry(q, c string) questboard.Reply  { return f.call("retry " + q + " " + c) }
func (f *fakeDesk) SetAuto(q string, on bool) questboard.Reply {
	if on {
		return f.call("auto " + q + " on")
	}
	return f.call("auto " + q + " off")
}
func (f *fakeDesk) Record(e questboard.AuditEvent) {
	f.mu.Lock()
	f.rec = append(f.rec, e)
	f.mu.Unlock()
}

func newSrv(d *fakeDesk, env map[string]string) (*Server, *time.Time) {
	if len(d.view.Quests) == 0 { // quests a and b are parked unless a test sets its own board
		blocked := []questboard.ChunkRow{{ID: "c", State: questboard.Blocked}}
		d.view.Quests = []questboard.QuestRow{{ID: "a", Chunks: blocked}, {ID: "b", Chunks: blocked}}
	}
	now := time.Date(2026, 11, 23, 3, 0, 0, 0, time.UTC)
	s := New(Config{Desk: func() Desk { return d }, Getenv: func(k string) string { return env[k] }, Now: func() time.Time { return now }})
	return s, &now
}

func TestEnabled(t *testing.T) {
	cases := []struct {
		env map[string]string
		on  bool
	}{
		{nil, false},
		{map[string]string{"D2BOT_OPERATOR": "0"}, false},
		{map[string]string{"D2BOT_OPERATOR": "1"}, true},
		{map[string]string{"D2BOT_OPERATOR": "1", "D2BOT_RUN": "g2"}, false},
	}
	for i, c := range cases {
		on, why := Enabled(func(k string) string { return c.env[k] })
		if on != c.on || (!on && why == "") {
			t.Errorf("case %d: on=%v why=%q", i, on, why)
		}
	}
}

func TestOffByDefaultNoSocket(t *testing.T) {
	dir := t.TempDir()
	stop, err := Start(dir, func() Desk { return &fakeDesk{} }, func(string) string { return "" }, nil)
	if err != nil || stop == nil {
		t.Fatalf("Start: %v", err)
	}
	stop()
	if _, err := os.Stat(filepath.Join(dir, SockName)); !os.IsNotExist(err) {
		t.Fatalf("socket exists without the env: %v", err)
	}
	// and refused during a measured run even with the env
	env := map[string]string{"D2BOT_OPERATOR": "1", "D2BOT_RUN": "x"}
	stop, _ = Start(dir, func() Desk { return &fakeDesk{} }, func(k string) string { return env[k] }, nil)
	stop()
	if _, err := os.Stat(filepath.Join(dir, SockName)); !os.IsNotExist(err) {
		t.Fatalf("socket exists in a measured run: %v", err)
	}
}

func TestSocketMode0600AndRoundTrip(t *testing.T) {
	dir := t.TempDir()
	d := &fakeDesk{rep: questboard.Reply{OK: true, Reason: "started"}}
	env := map[string]string{"D2BOT_OPERATOR": "1"}
	stop, err := Start(dir, func() Desk { return d }, func(k string) string { return env[k] }, nil)
	if err != nil {
		t.Fatal(err)
	}
	defer stop()
	p := filepath.Join(dir, SockName)
	fi, err := os.Stat(p)
	if err != nil {
		t.Fatal(err)
	}
	if fi.Mode().Perm() != 0o600 || fi.Mode()&os.ModeSocket == 0 {
		t.Fatalf("mode %v", fi.Mode())
	}
	resp, err := Call(p, Request{Verb: "start", Quest: "den_of_evil", Why: "overnight test"})
	if err != nil || !resp.OK {
		t.Fatalf("%+v %v", resp, err)
	}
	if len(d.calls) != 1 || d.calls[0] != "start den_of_evil " {
		t.Fatalf("calls %v", d.calls)
	}
	if _, err := net.Dial("unix", p); err != nil {
		t.Fatal(err)
	}
}

func TestAllowlistRefusesEverythingElse(t *testing.T) {
	d := &fakeDesk{}
	s, _ := newSrv(d, nil)
	for _, v := range []string{"sell", "vault", "hero", "character", "tuning", "respec", "order", "mode", "dismiss", "wait", "", "START", "start;sell", "../start"} {
		r := s.Handle(Request{Verb: v, Quest: "a", Why: "x"})
		if r.OK || r.Code != "verb" {
			t.Errorf("verb %q: %+v", v, r)
		}
	}
	if len(d.calls) != 0 || len(d.rec) != 0 {
		t.Fatalf("desk touched: %v %v", d.calls, d.rec)
	}
}

func TestWhyRequiredAndBounded(t *testing.T) {
	d := &fakeDesk{rep: questboard.Reply{OK: true}}
	s, _ := newSrv(d, nil)
	for _, why := range []string{"", "   ", strings.Repeat("x", 201), "a\nb"} {
		if r := s.Handle(Request{Verb: "retry", Quest: "a", Why: why}); r.OK || r.Code != "why" {
			t.Errorf("why %q: %+v", why, r)
		}
	}
	if r := s.Handle(Request{Verb: "retry", Quest: "a", Why: strings.Repeat("x", 200)}); !r.OK {
		t.Errorf("200 chars refused: %+v", r)
	}
	// status needs no reason
	if r := s.Handle(Request{Verb: "status"}); !r.OK {
		t.Errorf("status: %+v", r)
	}
}

func TestBadIDs(t *testing.T) {
	d := &fakeDesk{}
	s, _ := newSrv(d, nil)
	for _, q := range []string{"", "A b", "a/b", strings.Repeat("a", 65)} {
		if r := s.Handle(Request{Verb: "start", Quest: q, Why: "x"}); r.OK || r.Code != "quest" {
			t.Errorf("quest %q: %+v", q, r)
		}
	}
	if r := s.Handle(Request{Verb: "start", Quest: "a", Chunk: "x y", Why: "x"}); r.OK || r.Code != "chunk" {
		t.Errorf("chunk: %+v", r)
	}
	if r := s.Handle(Request{Verb: "auto", Quest: "a", Why: "x"}); r.OK || r.Code != "body" {
		t.Errorf("auto without on: %+v", r)
	}
}

func TestMeasuredRunRefusesMutatingAtRequestTime(t *testing.T) {
	d := &fakeDesk{rep: questboard.Reply{OK: true}}
	env := map[string]string{}
	s, _ := newSrv(d, env)
	env["D2BOT_RUN"] = "gate"
	r := s.Handle(Request{Verb: "start", Quest: "a", Why: "x"})
	if r.OK || r.Code != "measured_run" || len(d.calls) != 0 {
		t.Fatalf("%+v %v", r, d.calls)
	}
	if r := s.Handle(Request{Verb: "status"}); !r.OK {
		t.Fatalf("status must stay readable: %+v", r)
	}
}

func TestRateLimitPerQuestHour(t *testing.T) {
	d := &fakeDesk{rep: questboard.Reply{OK: true}}
	s, now := newSrv(d, nil)
	for i := 0; i < 6; i++ {
		*now = now.Add(2 * time.Minute) // past the retry cooldown
		if r := s.Handle(Request{Verb: "retry", Quest: "a", Why: "x"}); !r.OK {
			t.Fatalf("retry %d: %+v", i, r)
		}
	}
	*now = now.Add(2 * time.Minute)
	r := s.Handle(Request{Verb: "retry", Quest: "a", Why: "x"})
	if r.OK || r.Code != "rate_limited" || r.Reason == "" {
		t.Fatalf("7th: %+v", r)
	}
	if len(d.calls) != 6 {
		t.Fatalf("calls %d", len(d.calls))
	}
	// another quest is unaffected; after an hour the first is free again
	if r := s.Handle(Request{Verb: "retry", Quest: "b", Why: "x"}); !r.OK {
		t.Fatalf("other quest: %+v", r)
	}
	*now = now.Add(time.Hour)
	if r := s.Handle(Request{Verb: "retry", Quest: "a", Why: "x"}); !r.OK {
		t.Fatalf("after an hour: %+v", r)
	}
}

func TestRateLimitTotalHour(t *testing.T) {
	d := &fakeDesk{rep: questboard.Reply{OK: true}}
	s, now := newSrv(d, nil)
	for i := 0; i < 30; i++ {
		*now = now.Add(time.Second)
		q := string(rune('a'+i/5)) + "q"
		if r := s.Handle(Request{Verb: "cancel", Quest: q, Why: "x"}); !r.OK {
			t.Fatalf("%d: %+v", i, r)
		}
	}
	if r := s.Handle(Request{Verb: "cancel", Quest: "zq", Why: "x"}); r.OK || r.Code != "rate_limited" {
		t.Fatalf("31st: %+v", r)
	}
}

func TestRetryCooldown(t *testing.T) {
	d := &fakeDesk{rep: questboard.Reply{OK: true}}
	s, now := newSrv(d, nil)
	if r := s.Handle(Request{Verb: "retry", Quest: "a", Why: "x"}); !r.OK {
		t.Fatal(r)
	}
	*now = now.Add(30 * time.Second)
	for _, v := range []string{"retry", "start"} {
		if r := s.Handle(Request{Verb: v, Quest: "a", Why: "x"}); r.OK || r.Code != "cooldown" {
			t.Errorf("%s in cooldown: %+v", v, r)
		}
	}
	if r := s.Handle(Request{Verb: "cancel", Quest: "a", Why: "x"}); !r.OK {
		t.Errorf("cancel must pass: %+v", r)
	}
	*now = now.Add(31 * time.Second)
	if r := s.Handle(Request{Verb: "retry", Quest: "a", Why: "x"}); !r.OK {
		t.Errorf("after cooldown: %+v", r)
	}
}

func TestAuditLine(t *testing.T) {
	d := &fakeDesk{rep: questboard.Reply{OK: false, Code: "busy", Reason: "another chunk runs", Status: 409}}
	s, _ := newSrv(d, nil)
	r := s.Handle(Request{Verb: "start", Quest: "den_of_evil", Chunk: "c1", Why: "overnight retry 2"})
	if r.OK || r.Code != "busy" {
		t.Fatalf("the desk refusal must pass through: %+v", r)
	}
	if len(d.rec) != 1 {
		t.Fatalf("audit lines: %d", len(d.rec))
	}
	e := d.rec[0]
	if e.Remote != "operator" || e.Agent != "ops-cli" || e.Why != "overnight retry 2" || e.Action != "start" ||
		e.Quest != "den_of_evil" || e.Chunk != "c1" || e.Code != "busy" || e.Status != 409 || e.OK {
		t.Fatalf("%+v", e)
	}
	// a refusal before the desk is not an audit line of the desk (it is logged); no ring entry
	s.Handle(Request{Verb: "sell", Quest: "a", Why: "x"})
	if len(d.rec) != 1 {
		t.Fatal("allowlist refusal wrote to the ring")
	}
}

func TestStatusRows(t *testing.T) {
	d := &fakeDesk{view: questboard.View{Enabled: true, Difficulty: "normal",
		Quests: []questboard.QuestRow{{ID: "den", Chunks: []questboard.ChunkRow{
			{ID: "c1", State: questboard.Done},
			{ID: "c2", State: questboard.Blocked, Reason: &questboard.ReasonView{Code: "gave_up", Text: "could not find it"}},
			{ID: "c3", State: questboard.InProgress},
		}}},
		Running: &questboard.RunView{Quest: "den", Chunk: "c3", Step: "walk_to_cave"},
	}}
	s, _ := newSrv(d, nil)
	r := s.Handle(Request{Verb: "status"})
	if !r.OK || r.Board == nil || len(r.Board.Rows) != 3 {
		t.Fatalf("%+v", r)
	}
	rows := r.Board.Rows
	if rows[1].Code != "gave_up" || rows[1].Text != "could not find it" || rows[2].Step != "walk_to_cave" || rows[2].State != "in_progress" {
		t.Fatalf("%+v", rows)
	}
}

func TestWait(t *testing.T) {
	board := func(states ...string) func() (*Board, error) {
		i := 0
		return func() (*Board, error) {
			st := states[min(i, len(states)-1)]
			i++
			row := Row{Quest: "den", Chunk: "c1", State: st, Step: "s" + st}
			if st == "blocked" {
				row.Code, row.Text = "gave_up", "stuck"
			}
			return &Board{Rows: []Row{row}}, nil
		}
	}
	nosleep := func(time.Duration) {}
	clock := func() func() time.Time {
		t0 := time.Unix(0, 0)
		return func() time.Time { t0 = t0.Add(time.Second); return t0 }
	}
	cases := []struct {
		name    string
		states  []string
		timeout time.Duration
		want    string
	}{
		{"finished", []string{"accepted", "in_progress", "done"}, time.Hour, OutFinished},
		{"parked", []string{"in_progress", "blocked"}, time.Hour, OutBlocked},
		{"stopped", []string{"in_progress", "ready"}, time.Hour, OutStopped},
		{"timeout", []string{"in_progress"}, 5 * time.Second, OutTimeout},
		{"idle", []string{"ready"}, time.Hour, OutIdle},
	}
	for _, c := range cases {
		res := Wait(board(c.states...), "den", "", c.timeout, time.Second, nosleep, clock())
		if res.Outcome != c.want {
			t.Errorf("%s: %+v", c.name, res)
		}
		if c.want == OutBlocked && (res.Code != "gave_up" || res.Step == "") {
			t.Errorf("blocked evidence: %+v", res)
		}
	}
}

func TestProgressCounter(t *testing.T) {
	d := &fakeDesk{view: questboard.View{Enabled: true, Quests: []questboard.QuestRow{{ID: "q", Chunks: []questboard.ChunkRow{{ID: "c", State: questboard.InProgress}}}},
		Running: &questboard.RunView{Quest: "q", Chunk: "c", Step: "a"}}}
	s, _ := newSrv(d, nil)
	get := func() int { return s.Handle(Request{Verb: "status"}).Board.Rows[0].Progress }
	if get() != 0 || get() != 0 {
		t.Fatal("unchanged board must not count")
	}
	d.mu.Lock()
	d.view.Running = &questboard.RunView{Quest: "q", Chunk: "c", Step: "b"}
	d.mu.Unlock()
	if get() != 1 {
		t.Fatal("step change must count")
	}
}

func TestRetryOnlyWhenParked(t *testing.T) {
	d := &fakeDesk{rep: questboard.Reply{OK: true}, view: questboard.View{Quests: []questboard.QuestRow{
		{ID: "run", Chunks: []questboard.ChunkRow{{ID: "m", State: questboard.InProgress}}},
		{ID: "acc", Chunks: []questboard.ChunkRow{{ID: "m", State: questboard.Accepted}}},
		{ID: "rdy", Chunks: []questboard.ChunkRow{{ID: "m", State: questboard.Ready}}},
		{ID: "two", Chunks: []questboard.ChunkRow{{ID: "x", State: questboard.Done}, {ID: "y", State: questboard.Blocked}}},
	}}}
	s, _ := newSrv(d, nil)
	for _, q := range []string{"run", "acc", "rdy", "gone"} {
		r := s.Handle(Request{Verb: "retry", Quest: q, Why: "x"})
		if r.OK || r.Code != "not_parked" || !strings.Contains(r.Reason, "not blocked") {
			t.Errorf("%s: %+v", q, r)
		}
	}
	if len(d.calls) != 0 {
		t.Fatalf("the desk was called: %v", d.calls)
	}
	if r := s.Handle(Request{Verb: "retry", Quest: "two", Why: "x"}); !r.OK {
		t.Fatalf("a parked chunk of the quest: %+v", r)
	}
	if r := s.Handle(Request{Verb: "retry", Quest: "two", Chunk: "x", Why: "x"}); r.OK || r.Code != "not_parked" {
		t.Fatalf("a done chunk named: %+v", r)
	}
	// the refusal books nothing: six real retries still pass later
}

// The owner's try anyway through the socket: its own audit path with the stated reason and the skipped refusal, the same
// rate limits as every start (6 per quest per hour), never on another verb, never in a measured run. Two quests stand in
// for two builds: the verb is quest and class generic.
func TestTryAnywayIsAuditedAndRateLimited(t *testing.T) {
	for _, quest := range []string{"a", "b"} {
		d := &fakeDesk{rep: questboard.Reply{OK: true, Reason: "started anyway"}}
		s, now := newSrv(d, nil)
		r := s.Handle(Request{Verb: "start", Quest: quest, Why: "owner wants to see", TryAnyway: true})
		if !r.OK || len(d.calls) != 1 || !strings.HasPrefix(d.calls[0], "try "+quest) {
			t.Fatalf("%s: %+v calls %v", quest, r, d.calls)
		}
		if len(d.rec) != 1 || d.rec[0].Path != "operator/try-anyway" || !strings.Contains(d.rec[0].Why, "override: try anyway: owner wants to see") ||
			!strings.Contains(d.rec[0].Why, "refused: too hard for now") {
			t.Fatalf("%s audit: %+v", quest, d.rec)
		}
		if r := s.Handle(Request{Verb: "retry", Quest: quest, Why: "x", TryAnyway: true}); r.Code != "body" {
			t.Errorf("try_anyway on retry: %+v", r)
		}
		// six tries within the hour pass the per-quest limit, the seventh is refused; a minute between them for the cooldown
		for i := 0; i < PerQuestHour-1; i++ {
			*now = now.Add(2 * time.Minute)
			if r := s.Handle(Request{Verb: "start", Quest: quest, Why: "again", TryAnyway: true}); !r.OK {
				t.Fatalf("%s try %d: %+v", quest, i+2, r)
			}
		}
		*now = now.Add(2 * time.Minute)
		if r := s.Handle(Request{Verb: "start", Quest: quest, Why: "again", TryAnyway: true}); r.Code != "rate_limited" {
			t.Errorf("%s seventh try: %+v", quest, r)
		}
	}
	d := &fakeDesk{}
	s, _ := newSrv(d, map[string]string{"D2BOT_RUN": "g2"})
	if r := s.Handle(Request{Verb: "start", Quest: "a", Why: "x", TryAnyway: true}); r.Code != "measured_run" || len(d.calls) != 0 {
		t.Errorf("measured run: %+v", r)
	}
}

type fakeOrders struct {
	calls []string
	err   error
}

func (o *fakeOrders) Give(kind, area string, minutes int, replace bool) (string, error) {
	o.calls = append(o.calls, fmt.Sprintf("%s|%s|%d|%t", kind, area, minutes, replace))
	return "order " + kind, o.err
}

func newOrderSrv(d *fakeDesk, o *fakeOrders, env map[string]string) (*Server, *time.Time) {
	s, now := newSrv(d, env)
	s.cfg.Orders = func() Orders { return o }
	s.cfg.Explain = func() any { return map[string]string{"decisions": "ring"} }
	return s, now
}

func TestWhyIsReadOnlyAndNeedsNoReason(t *testing.T) {
	d := &fakeDesk{}
	s, _ := newOrderSrv(d, &fakeOrders{}, map[string]string{"D2BOT_RUN": "x"}) // a read even in a measured run
	r := s.Handle(Request{Verb: "why"})
	if !r.OK || r.Explain == nil || len(d.calls) != 0 || len(d.rec) != 0 {
		t.Fatalf("%+v %v %v", r, d.calls, d.rec)
	}
	s.cfg.Explain = nil
	if r := s.Handle(Request{Verb: "why"}); r.OK || r.Code != "off" {
		t.Fatalf("%+v", r)
	}
}

func TestOrderVerbsMapOntoOrdersAndAreAudited(t *testing.T) {
	// two hero shapes: an act 1 area and an act 2 area (areas are data, the verbs know no class)
	for _, area := range []string{"Cold Plains", "Lut Gholein"} {
		d := &fakeDesk{}
		o := &fakeOrders{}
		s, _ := newOrderSrv(d, o, nil)
		for _, rq := range []Request{
			{Verb: "hold", Why: "stand down"},
			{Verb: "home", Why: "rest"},
			{Verb: "go", Area: area, Why: "farm"},
			{Verb: "go", Area: area, Minutes: 15, Why: "short"},
		} {
			if r := s.Handle(rq); !r.OK {
				t.Fatalf("%s %+v: %+v", area, rq, r)
			}
		}
		want := []string{"home||0|true", "home||0|false", "go|" + area + "|60|false", "go|" + area + "|15|false"}
		if !reflect.DeepEqual(o.calls, want) {
			t.Fatalf("%s: calls %v, want %v", area, o.calls, want)
		}
		if len(d.rec) != 4 || d.rec[0].Remote != "operator" || d.rec[0].Path != "operator/hold" || !strings.Contains(d.rec[2].Why, "farm") {
			t.Fatalf("%s: audit %+v", area, d.rec)
		}
		if r := s.Handle(Request{Verb: "hold"}); r.OK || r.Code != "why" {
			t.Fatalf("no reason: %+v", r)
		}
		o.err = errors.New("area is not orderable")
		if r := s.Handle(Request{Verb: "go", Area: "Nowhere", Why: "x"}); r.OK || r.Code != "order_refused" {
			t.Fatalf("refused order: %+v", r)
		}
	}
}

func TestOrderVerbsRefusedInMeasuredRunAndWithoutOrders(t *testing.T) {
	d := &fakeDesk{}
	o := &fakeOrders{}
	s, _ := newOrderSrv(d, o, map[string]string{"D2BOT_RUN": "g2"})
	if r := s.Handle(Request{Verb: "hold", Why: "x"}); r.OK || r.Code != "measured_run" || len(o.calls) != 0 {
		t.Fatalf("%+v %v", r, o.calls)
	}
	s, _ = newOrderSrv(d, o, nil)
	s.cfg.Orders = nil
	if r := s.Handle(Request{Verb: "home", Why: "x"}); r.OK || r.Code != "off" {
		t.Fatalf("%+v", r)
	}
}

func TestOrderVerbsRateLimited(t *testing.T) {
	s, _ := newOrderSrv(&fakeDesk{}, &fakeOrders{}, nil)
	for i := 0; i < PerQuestHour; i++ {
		if r := s.Handle(Request{Verb: "home", Why: "x"}); !r.OK {
			t.Fatalf("%d: %+v", i, r)
		}
	}
	if r := s.Handle(Request{Verb: "home", Why: "x"}); r.OK || r.Code != "rate_limited" {
		t.Fatalf("%+v", r)
	}
}

func TestParkCancelsAndHoldsOffStartUntilWake(t *testing.T) {
	for _, quest := range []string{"a", "b"} {
		d := &fakeDesk{}
		s, now := newOrderSrv(d, &fakeOrders{}, nil)
		if r := s.Handle(Request{Verb: "park", Quest: quest, Hours: 25, Why: "x"}); r.Code != "hours" {
			t.Fatalf("25h: %+v", r)
		}
		if r := s.Handle(Request{Verb: "park", Quest: quest, Hours: 2, Why: "wait for gold"}); !r.OK {
			t.Fatalf("%+v", r)
		}
		if len(d.calls) != 1 || !strings.HasPrefix(d.calls[0], "cancel "+quest) {
			t.Fatalf("calls %v", d.calls)
		}
		for _, v := range []string{"start", "retry"} {
			if r := s.Handle(Request{Verb: v, Quest: quest, Why: "x"}); r.OK || r.Code != "parked" {
				t.Fatalf("%s while parked: %+v", v, r)
			}
		}
		if b := s.Handle(Request{Verb: "status"}).Board; len(b.Parked) != 1 || b.Parked[0].Quest != quest {
			t.Fatalf("status parked: %+v", b.Parked)
		}
		*now = now.Add(3 * time.Hour)
		if r := s.Handle(Request{Verb: "start", Quest: quest, Why: "x"}); r.Code == "parked" {
			t.Fatalf("still parked after the wake time: %+v", r)
		}
	}
}

type fakeDiff struct {
	ready bool
	got   []string
}

func (f *fakeDiff) RequestDifficultyGo(t string) (string, error) {
	f.got = append(f.got, t)
	if !f.ready {
		return "", errors.New("not ready for " + t + ": clvl 20, need 38")
	}
	return "go recorded for " + t, nil
}

func diffSrv(d *fakeDesk, g *fakeDiff, env map[string]string) *Server {
	s, _ := newSrv(d, env)
	s.cfg.Difficulty = func() DifficultyGo { return g }
	return s
}

// go-difficulty: a reason and a known target are needed, a not-ready hero is refused with the reason and audited,
// a ready one is recorded, a measured run and a missing wire say no.
func TestGoDifficulty(t *testing.T) {
	d, g := &fakeDesk{}, &fakeDiff{}
	s := diffSrv(d, g, nil)
	if r := s.Handle(Request{Verb: DifficultyVerb, Target: "nightmare"}); r.Code != "why" {
		t.Fatalf("no reason: %+v", r)
	}
	if r := s.Handle(Request{Verb: DifficultyVerb, Target: "normal", Why: "x"}); r.Code != "target" {
		t.Fatalf("bad target: %+v", r)
	}
	r := s.Handle(Request{Verb: DifficultyVerb, Target: "nightmare", Why: "ready by my look"})
	if r.OK || r.Code != "order_refused" || !strings.Contains(r.Reason, "clvl 20, need 38") {
		t.Fatalf("not ready: %+v", r)
	}
	if len(d.rec) != 1 || d.rec[0].OK || d.rec[0].Remote != "operator" {
		t.Fatalf("audit %+v", d.rec)
	}
	g.ready = true
	if r := s.Handle(Request{Verb: DifficultyVerb, Target: "hell", Why: "go"}); !r.OK || len(g.got) != 2 {
		t.Fatalf("ready: %+v %v", r, g.got)
	}
	if r := diffSrv(d, g, map[string]string{"D2BOT_RUN": "g1"}).Handle(Request{Verb: DifficultyVerb, Target: "hell", Why: "go"}); r.Code != "measured_run" {
		t.Fatalf("measured run: %+v", r)
	}
	s2, _ := newSrv(d, nil)
	if r := s2.Handle(Request{Verb: DifficultyVerb, Target: "hell", Why: "go"}); r.Code != "off" {
		t.Fatalf("unwired: %+v", r)
	}
}
