package main

import (
	"context"
	"path/filepath"
	"strings"
	"sync/atomic"
	"testing"
	"time"

	"d2lod-bot/internal/data"
	"d2lod-bot/internal/goals"
	"d2lod-bot/internal/task"
	"d2lod-bot/internal/watch"
)

// The bestiary, the risk meter, the order queue and the atlas on the real wiring (TASK-185): startLadder + feedHub with a
// fake runner, a fake monster table and a fake facts reader, two classes. Fixtures only; nothing here ran against a game.

type fakeTable map[string]*data.Monster

func (f fakeTable) MonsterByName(n string) *data.Monster { return f[strings.ToLower(n)] }

func wireTable(t *testing.T, skip string) fakeTable {
	t.Helper()
	notes, _, err := watch.FilterBestiaryNotes(nil)
	if err != nil {
		t.Fatal(err)
	}
	m := fakeTable{}
	for _, n := range notes {
		if n.Name != skip {
			m[strings.ToLower(n.Name)] = &data.Monster{Name: n.Name, Type: "demon", Melee: true}
		}
	}
	f := m["fallen"]
	f.Res[data.Normal][data.Fire], f.Res[data.Nightmare][data.Fire], f.Res[data.Hell][data.Fire] = 0, 50, 100
	return m
}

type wireRig struct {
	rig        *ladderRig
	m          *goals.Manager
	view, ans  string
	cancel     context.CancelFunc
	life       atomic.Int32
	state      string
	class      string
	statusFunc func() goals.Status
}

const wireTok = "wire-watch-token-1234"

func newWireRig(t *testing.T, class, state string, table watch.MonsterTables) *wireRig {
	t.Helper()
	ctx, cancel := context.WithCancel(context.Background())
	t.Cleanup(cancel)
	w := &wireRig{m: goalManager(t), cancel: cancel, state: state, class: class}
	w.life.Store(100)
	lo := ladderOptions{on: true, watch: true, viewAddr: "127.0.0.1:0", answerAddr: "127.0.0.1:0", watchToken: wireTok,
		currencyDir: filepath.Join(state, "currency"), monsters: table,
		riskFacts: func(zone string) watch.RiskFacts {
			return watch.RiskFacts{Zone: zone, Kit: "OK", Potions: 4, LifePct: int(w.life.Load()), EngagePct: 65, MinEngagePct: 40, Ok: true}
		}}
	status := func() goals.Status {
		return goals.Status{State: goals.RunnerRunning, Character: goals.Character{Name: "Hero", Class: class, Level: 12, Area: "Blood Moor", AreaID: 2}}
	}
	rig, err := startLadder(ctx, lo, "", state, fakeEnableFirsts(t, &watch.SimClock{}), w.m, status, func() string { return "Blood Moor" }, nil)
	if err != nil || rig == nil || rig.Fun == nil {
		t.Fatalf("%v %v", rig, err)
	}
	w.rig = rig
	w.view, w.ans = "http://"+rig.Served.View.String(), "http://"+rig.Served.Answer.String()
	return w
}

func (w *wireRig) get(t *testing.T, path string, out any) int {
	t.Helper()
	return httpDoH(t, "GET", w.view+path, nil, map[string]string{"Authorization": "Bearer " + wireTok}, out)
}

func (w *wireRig) post(t *testing.T, path string, body any, out any) int {
	t.Helper()
	return httpDoH(t, "POST", w.ans+path, body, map[string]string{"Authorization": "Bearer " + wireTok}, out)
}

func waitUntil(t *testing.T, what string, ok func() bool) {
	t.Helper()
	for end := time.Now().Add(8 * time.Second); time.Now().Before(end); time.Sleep(50 * time.Millisecond) {
		if ok() {
			return
		}
	}
	t.Fatalf("timeout: %s", what)
}

func (w *wireRig) openOrders(t *testing.T) []watch.Order {
	t.Helper()
	var ov watch.OrdersView
	w.get(t, "/api/orders", &ov)
	var out []watch.Order
	for _, o := range ov.Orders {
		if o.State == watch.OrderQueued || o.State == watch.OrderActive {
			out = append(out, o)
		}
	}
	return out
}

func questEv(id string, stage int, seq uint64) goals.Event {
	return goals.Event{Type: goals.EvQuest, Quest: id, Stage: stage, Done: stage >= 3, Seq: seq}
}

// A near-death dip that recovers opens the risk row; the verdict then comes from the runner's facts. Token gates apply.
func TestRiskMeterWiredToTheBotFacts(t *testing.T) {
	for _, class := range []string{"Sorceress", "Paladin"} {
		t.Run(class, func(t *testing.T) {
			w := newWireRig(t, class, t.TempDir(), nil)
			if code := httpDo(t, "GET", w.view+"/api/risk", nil, nil); code != 401 {
				t.Errorf("risk without a token: %d", code)
			}
			var rv watch.RiskView
			if w.get(t, "/api/risk?zone=Blood+Moor", &rv) != 200 || rv.Open || rv.Teaser == "" {
				t.Fatalf("locked view: %+v", rv)
			}
			w.life.Store(20) // below 25 percent, then back to 50 or more: one close call
			w.rig.Fun.Settle()
			w.rig.Fun.Settle()
			w.get(t, "/api/risk", &rv)
			if rv.Open {
				t.Fatalf("opened while still down: %+v", rv)
			}
			w.life.Store(60)
			w.rig.Fun.Settle()
			w.rig.Fun.Settle() // the same episode does not count twice
			waitUntil(t, "a risk verdict from the bot facts", func() bool {
				rv = watch.RiskView{}
				w.get(t, "/api/risk?zone=Blood+Moor", &rv)
				return rv.Open && rv.Verdict != nil
			})
			if rv.Recent != 1 || rv.Closest == nil || rv.Closest.LowPct != 20 {
				t.Errorf("close calls: %+v", rv)
			}
			if rv.Verdict.Zone != "Blood Moor" || len(rv.Verdict.Checks) == 0 {
				t.Errorf("verdict: %+v", rv.Verdict)
			}
			w.get(t, "/api/risk?zone=Nowhere", &rv)
			if !strings.Contains(rv.Error, "Nowhere") {
				t.Errorf("an unknown zone is an error naming it: %q", rv.Error)
			}
		})
	}
}

// The Den clears slot one; the sequencer places the orders in turn through Hub.place; a death stops it until resume.
func TestOrderQueueWiredThroughTheOrdersPath(t *testing.T) {
	for _, class := range []string{"Sorceress", "Paladin"} {
		t.Run(class, func(t *testing.T) {
			w := newWireRig(t, class, t.TempDir(), nil)
			var qv watch.OrderQueueView
			if code := httpDo(t, "GET", w.view+"/api/orderqueue", nil, nil); code != 401 {
				t.Errorf("queue without a token: %d", code)
			}
			if code := httpDo(t, "POST", w.ans+"/api/orderqueue", map[string]any{"action": "clear"}, nil); code != 401 {
				t.Errorf("queue post without a token: %d", code)
			}
			if w.get(t, "/api/orderqueue", &qv); qv.Open {
				t.Fatalf("open before the Den: %+v", qv)
			}
			if code := w.post(t, "/api/orderqueue", map[string]any{"action": "add", "kind": "go", "area": "Blood Moor", "minutes": 5}, nil); code != 403 {
				t.Fatalf("add while locked: %d", code)
			}
			w.m.Handle(questEv(task.QDen, 2, 0)) // Den of Evil cleared: slot one
			waitUntil(t, "slot one", func() bool { w.get(t, "/api/orderqueue", &qv); return qv.Open && qv.Capacity == 1 })
			w.m.Handle(questEv(task.QBurial, 3, 0)) // Kashya's quest done: slot two
			waitUntil(t, "slot two", func() bool { w.get(t, "/api/orderqueue", &qv); return qv.Capacity == 2 })
			for _, a := range []string{"Blood Moor", "Cold Plains"} {
				// the first status poll (2 s tick) tells the hero's level; until then an add is a 400
				waitUntil(t, "add "+a, func() bool {
					return w.post(t, "/api/orderqueue", map[string]any{"action": "add", "kind": "go", "area": a, "minutes": 5}, nil) == 200
				})
			}
			waitUntil(t, "the first order placed", func() bool { return len(w.openOrders(t)) == 1 })
			if got := w.openOrders(t); got[0].Area != "Blood Moor" {
				t.Fatalf("first order: %+v", got)
			}
			w.rig.Fun.Settle() // still one open order
			w.rig.Fun.Settle()
			if n := len(w.openOrders(t)); n != 1 {
				t.Fatalf("%d open orders", n)
			}
			w.rig.Hub.OrderState(w.openOrders(t)[0].ID, watch.OrderDone, "")
			w.m.Handle(goals.Event{Type: goals.EvDeath}) // the death stops the queue before the second order
			waitUntil(t, "stopped on death", func() bool { w.get(t, "/api/orderqueue", &qv); return qv.Stopped != "" })
			w.rig.Fun.Settle()
			w.rig.Fun.Settle()
			if n := len(w.openOrders(t)); n != 0 {
				t.Fatalf("an order was placed while stopped: %d", n)
			}
			if code := w.post(t, "/api/orderqueue", map[string]any{"action": "resume"}, nil); code != 200 {
				t.Fatalf("resume: %d", code)
			}
			waitUntil(t, "the second order", func() bool { o := w.openOrders(t); return len(o) == 1 && o[0].Area == "Cold Plains" })
		})
	}
}

// Without a Pact controller the halt signal is empty; a missing rig part never panics.
func TestPactHaltWithoutController(t *testing.T) {
	if s := pactHalt(&ladderRig{}); s != "" {
		t.Errorf("halt %q", s)
	}
}

// Bestiary cards come from the monster table; resists appear at 50 kills; a kind the table lacks is skipped and logged;
// the counts survive a restart (same state dir); the atlas needs no config and answers.
func TestBestiaryWiredWithTableAndRestart(t *testing.T) {
	for _, class := range []string{"Sorceress", "Paladin"} {
		t.Run(class, func(t *testing.T) {
			state := t.TempDir()
			w := newWireRig(t, class, state, wireTable(t, "Returned"))
			if code := httpDo(t, "GET", w.view+"/api/bestiary", nil, nil); code != 401 {
				t.Errorf("bestiary without a token: %d", code)
			}
			errs := strings.Join(w.rig.Fun.Errors(), "|")
			if !strings.Contains(errs, `"Returned"`) {
				t.Errorf("the missing kind is not logged: %q", errs)
			}
			for i := 0; i < 50; i++ {
				w.m.Handle(goals.Event{Type: goals.EvKill, Monster: "Fallen", XP: 1})
			}
			var bv watch.BestiaryView
			waitUntil(t, "50 kills", func() bool { w.get(t, "/api/bestiary", &bv); return cardOf(bv, "fallen").Kills == 50 })
			c := cardOf(bv, "fallen")
			if !bv.Open || c.State != "full" || c.Resists == nil || !bv.Tables {
				t.Fatalf("fallen card: %+v (open %v tables %v)", c, bv.Open, bv.Tables)
			}
			if cardOf(bv, "returned").ID != "" {
				t.Errorf("a kind absent from the table has a card")
			}
			var av watch.AtlasView
			if code := w.get(t, "/api/atlas", &av); code != 200 {
				t.Errorf("atlas: %d", code)
			}
			w.cancel()
			time.Sleep(300 * time.Millisecond)

			w2 := newWireRig(t, class, state, wireTable(t, "Returned"))
			var bv2 watch.BestiaryView
			w2.get(t, "/api/bestiary", &bv2)
			if c := cardOf(bv2, "fallen"); c.Kills != 50 || c.State != "full" || !bv2.Open {
				t.Fatalf("after the restart: %+v open %v", c, bv2.Open)
			}
		})
	}
}

// With no monster table the cards carry the flavour only and the bot still starts.
func TestBestiaryWithoutTablesIsFlavourOnly(t *testing.T) {
	w := newWireRig(t, "Sorceress", t.TempDir(), nil)
	for i := 0; i < 10; i++ {
		w.m.Handle(goals.Event{Type: goals.EvKill, Monster: "Fallen", XP: 1})
	}
	var bv watch.BestiaryView
	waitUntil(t, "10 kills", func() bool { w.get(t, "/api/bestiary", &bv); return cardOf(bv, "fallen").Kills == 10 })
	if bv.Tables || cardOf(bv, "fallen").Resists != nil {
		t.Errorf("tables %v card %+v", bv.Tables, cardOf(bv, "fallen"))
	}
}

func cardOf(v watch.BestiaryView, id string) watch.BestiaryCard {
	for _, c := range v.Cards {
		if c.ID == id {
			return c
		}
	}
	return watch.BestiaryCard{}
}
