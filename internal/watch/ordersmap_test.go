package watch

import (
	"strings"
	"testing"
	"time"

	"d2lod-bot/internal/goals"
)

func (r *rig) omap() MapView { var v MapView; r.get("/api/orders/map", &v); return v }

func nodeOf(t *testing.T, v MapView, id string) MapNodeView {
	t.Helper()
	for _, n := range v.Nodes {
		if n.ID == id {
			return n
		}
	}
	t.Fatalf("no node %q", id)
	return MapNodeView{}
}

// Schema: every edge joins two nodes, the box holds every node, ids are unique, acts 2 to 5 are empty data.
func TestOrdersMapSchema(t *testing.T) {
	r := newRig(t, "sorceress")
	v := r.omap()
	if v.Schema != mapSchema || v.Act != 1 || !v.Populated || v.Town != "camp" || len(v.Acts) != 5 || v.MaxOpen != MaxOpenOrders {
		t.Fatalf("header: %+v", v)
	}
	seen := map[string]bool{}
	for _, n := range v.Nodes {
		if seen[n.ID] {
			t.Errorf("duplicate node %s", n.ID)
		}
		seen[n.ID] = true
		if n.X-nodeW/2 < v.Box.X || n.X+nodeW/2 > v.Box.X+v.Box.W || n.Y-nodeH/2 < v.Box.Y || n.Y+nodeH/2 > v.Box.Y+v.Box.H {
			t.Errorf("node %s outside the box %+v", n.ID, v.Box)
		}
	}
	for _, e := range v.Edges {
		if !seen[e.From] || !seen[e.To] {
			t.Errorf("edge %+v joins an unknown node", e)
		}
	}
	// every node is reachable from the town: the breadcrumb always has a route
	reach := map[string]bool{v.Town: true}
	for changed := true; changed; {
		changed = false
		for _, e := range v.Edges {
			if reach[e.From] != reach[e.To] {
				reach[e.From], reach[e.To], changed = true, true, true
			}
		}
	}
	for id := range seen {
		if !reach[id] {
			t.Errorf("node %s is not connected to the town", id)
		}
	}
	for _, a := range v.Acts {
		if a.Populated != (a.Act == 1) {
			t.Errorf("act %d populated=%v", a.Act, a.Populated)
		}
	}
	var a3 MapView
	r.get("/api/orders/map?act=3", &a3)
	if a3.Act != 3 || a3.Populated || len(a3.Nodes) != 0 || a3.Nodes == nil || a3.Edges == nil {
		t.Errorf("act 3 must be empty data, not null: %+v", a3)
	}
	// an orderable node always has an order name the order route accepts
	for _, n := range v.Nodes {
		if n.Orderable && n.Kind != "town" && !containsFold(r.sim.Hub.cfg.Areas, n.Order) {
			t.Errorf("%s orderable but %q is not in the area list", n.ID, n.Order)
		}
	}
}

// Scenario A (mock): in camp, nothing running. Golden JSON.
func TestOrdersMapScenarioA(t *testing.T) {
	r := newRig(t, "sorceress")
	v := r.omap()
	camp := nodeOf(t, v, "camp")
	if !camp.Status.Here || camp.Status.Order || !v.Hero.Known || v.Hero.Level != 6 {
		t.Errorf("camp: %+v hero %+v", camp.Status, v.Hero)
	}
	for _, id := range []string{"line", "cell", "up", "mg", "oc"} {
		if nodeOf(t, v, id).Orderable {
			t.Errorf("%s must not be orderable yet", id)
		}
	}
	cp := nodeOf(t, v, "cp")
	if !cp.Orderable || cp.Advice != "" {
		t.Errorf("Cold Plains at clvl 6 (range starts at 6): %+v", cp)
	}
	sf := nodeOf(t, v, "sf")
	if !strings.Contains(sf.Advice, "Advice, not a guard") || !strings.Contains(sf.Advice, "clvl 10") || sf.Status.Hard {
		t.Errorf("advice must be advice, never a guard: %+v", sf)
	}
	golden(t, "ordersmap-a", r.get("/api/orders/map", nil).Body.String())
}

// Scenario B (mock): the hero works on Cold Plains, one order waits. here, order and queued marks sit on the right nodes.
func TestOrdersMapScenarioB(t *testing.T) {
	r := newRig(t, "sorceress")
	r.must("/api/orders", map[string]any{"kind": "go", "area": "Cold Plains", "minutes": 30})
	r.must("/api/orders", map[string]any{"kind": "go", "area": "Stony Field", "minutes": 30})
	r.sim.Step(20 * time.Second) // the walk (15 s) ends: the hero stands in Cold Plains
	v := r.omap()
	cp, sf, camp := nodeOf(t, v, "cp"), nodeOf(t, v, "sf"), nodeOf(t, v, "camp")
	if !cp.Status.Here || !cp.Status.Order || cp.Status.Queued != 0 {
		t.Errorf("Cold Plains: %+v", cp.Status)
	}
	if sf.Status.Here || sf.Status.Order || sf.Status.Queued != 1 {
		t.Errorf("Stony Field: %+v", sf.Status)
	}
	if camp.Status.Here || v.OpenOrder != 2 {
		t.Errorf("camp %+v open %d", camp.Status, v.OpenOrder)
	}
	var ov OrdersView
	r.get("/api/orders", &ov)
	if ov.Orders[0].State != OrderActive || ov.Orders[0].StartedAt != 0 || ov.Orders[0].Kills == 0 || ov.Orders[0].XPGained == 0 {
		t.Errorf("progress of the active order: %+v", ov.Orders[0])
	}
	golden(t, "ordersmap-b", r.get("/api/orders/map", nil).Body.String())
}

// Scenario C (mock): the Forgotten Tower is too hard. The reason is the guard's own text, never a claim of the screen;
// the same on two builds with their own floors. An open guard leaves the node unmarked; Tamoe Highland etc. stay open.
func TestOrdersMapScenarioCTooHard(t *testing.T) {
	for _, c := range []struct{ build, want string }{
		{"sorceress", "no mercenary and clvl 6 is below 16"},
		{"paladin", "no mercenary and clvl 12 is below 14"},
	} {
		r := newRig(t, c.build)
		v := r.omap()
		ft, cell := nodeOf(t, v, "ftw"), nodeOf(t, v, "cell")
		if !ft.Status.Hard || ft.Status.HardWhy != c.want || !ft.Orderable {
			t.Errorf("%s Forgotten Tower: %+v", c.build, ft)
		}
		if !cell.Status.Hard || cell.Orderable {
			t.Errorf("%s cellars: %+v", c.build, cell)
		}
		if nodeOf(t, v, "bm").Status.Hard {
			t.Errorf("%s: Blood Moor marked hard", c.build)
		}
		// the order is accepted: the order asks, the hero's guard decides later
		if code, rp := r.post("/api/orders", map[string]any{"kind": "go", "area": "Forgotten Tower", "minutes": 30}); code != 200 || !rp.OK {
			t.Errorf("%s: order to a too-hard area refused by the screen: %d %+v", c.build, code, rp)
		}
	}
	r := newRig(t, "sorceress")
	r.sim.Hub.cfg.AreaGuard = func(int) string { return "" }
	if nodeOf(t, r.omap(), "ftw").Status.Hard {
		t.Error("an open guard must not mark the tower")
	}
	r.sim.Hub.cfg.AreaGuard = func(id int) string {
		if id == 7 {
			return "monastery road: clvl 6 < 16"
		}
		return ""
	}
	if tam := nodeOf(t, r.omap(), "tam"); !tam.Status.Hard || !strings.Contains(tam.Status.HardWhy, "monastery") {
		t.Errorf("Tamoe Highland: %+v", tam.Status)
	}
	golden(t, "ordersmap-c", newRigC(t).get("/api/orders/map", nil).Body.String())
}

func newRigC(t *testing.T) *rig {
	r := newRig(t, "sorceress")
	r.must("/api/orders", map[string]any{"kind": "go", "area": "Black Marsh", "minutes": 30})
	r.sim.Step(20 * time.Second)
	return r
}

// Names come from the bot and the player: the map JSON escapes markup and the page never parses it.
func TestOrdersMapTextIsPlain(t *testing.T) {
	r := newRig(t, "sorceress")
	r.sim.Hub.cfg.AreaGuard = func(int) string { return "<img src=x onerror=alert(1)>\n" + strings.Repeat("x", 400) }
	r.sim.Hub.SetHero(HeroInfo{Area: "<script>alert(1)</script>", Level: 6, Activity: "<b>fighting</b>\x00"})
	w := r.get("/api/orders/map", nil)
	body := w.Body.String()
	if strings.Contains(body, "<script>") || strings.Contains(body, "<img") || strings.Contains(body, "<b>") {
		t.Errorf("markup in the body: %s", body[:200])
	}
	v := r.omap()
	if why := nodeOf(t, v, "ftw").Status.HardWhy; len(why) > 160 || strings.ContainsAny(why, "\n") {
		t.Errorf("reason not plain/short: %q", why)
	}
	// a viewer cannot write
	if w := r.postRaw(r.view, "/api/orders/map", "{}", nil); w.Code != 405 {
		t.Errorf("POST on the viewer: %d", w.Code)
	}
}

// Order route: forever, replace ("Go now"), and the refusals.
func TestOrderForeverAndReplace(t *testing.T) {
	r := newRig(t, "paladin")
	r.must("/api/orders", map[string]any{"kind": "go", "area": "Blood Moor", "minutes": 10})
	rp := r.must("/api/orders", map[string]any{"kind": "go", "area": "Dark Wood", "until": "forever", "replace": true})
	if rp.Order == nil || !rp.Order.Forever || rp.Order.Minutes != 0 || rp.Order.Text != "Farm Dark Wood until you cancel" {
		t.Fatalf("forever order: %+v", rp.Order)
	}
	var ov OrdersView
	r.get("/api/orders", &ov)
	if len(ov.Orders) != 2 || ov.Orders[0].State != OrderCancelled || ov.Orders[1].State != OrderActive {
		t.Fatalf("replace: %+v", ov.Orders)
	}
	r.sim.Step(30 * time.Minute) // no end: it runs on
	r.get("/api/orders", &ov)
	if ov.Orders[1].State != OrderActive {
		t.Errorf("an until-I-cancel order ended: %+v", ov.Orders[1])
	}
	for _, c := range []struct {
		name string
		body map[string]any
	}{
		{"level", map[string]any{"kind": "go", "area": "Dark Wood", "until": "level", "minutes": 9}},
		{"forever with minutes", map[string]any{"kind": "go", "area": "Dark Wood", "until": "forever", "minutes": 5}},
		{"home forever", map[string]any{"kind": "home", "until": "forever"}},
		{"unknown area", map[string]any{"kind": "go", "area": "Tower Cellar 1", "minutes": 5}},
		{"no minutes", map[string]any{"kind": "go", "area": "Dark Wood"}},
	} {
		if code, rp := r.post("/api/orders", c.body); code != 400 || rp.Code != "bad-order" {
			t.Errorf("%s: %d %+v", c.name, code, rp)
		}
	}
	// a refused replacement cancels nothing
	r.get("/api/orders", &ov)
	if ov.Orders[1].State != OrderActive {
		t.Errorf("bad replace cancelled the order: %+v", ov.Orders)
	}
	code, _ := r.post("/api/orders/cancel", map[string]any{"id": ov.Orders[1].ID})
	if code != 200 {
		t.Errorf("cancel: %d", code)
	}
}

// The real path: order POST -> farm_area goal -> goal events -> order state, kills, xp, journal lines and uptime, against a
// fake queue; the hero's guard refusal ends the order failed with the guard's reason (who wins: the guard).
func TestOrderToGoalToUptime(t *testing.T) {
	for _, build := range []string{"sorceress", "paladin"} {
		r := newRig(t, build)
		q := &fakeQueue{}
		b := &GoalsOrders{Q: q, Hub: r.sim.Hub, Here: func() string { return "Rogue Encampment" }}
		r.sim.Hub.cfg.Orders = b
		r.sim.Hub.cfg.AreaGuard = func(id int) string {
			if id == 20 {
				return "no merc, clvl 6 < 16"
			}
			return ""
		}
		r.must("/api/orders", map[string]any{"kind": "go", "area": "Cold Plains", "until": "forever"})
		if len(q.added) != 1 || q.added[0].Type != goals.FarmArea || q.added[0].Params.Area != "Cold Plains" ||
			q.added[0].Params.UntilMins != 0 || q.added[0].Params.UntilKills != 0 || q.added[0].Params.UntilDrops != 0 {
			t.Fatalf("%s: forever must be an endless farm_area: %+v", build, q.added)
		}
		// "Go now" replaces: the first goal is removed before the second is added
		r.must("/api/orders", map[string]any{"kind": "go", "area": "Forgotten Tower", "minutes": 20, "replace": true})
		if len(q.removed) != 1 || q.removed[0] != "g1" || len(q.added) != 2 {
			t.Fatalf("%s: replace: removed %v added %d", build, q.removed, len(q.added))
		}
		b.OnGoalEvent(goals.Event{Type: goals.EvGoalRemoved, GoalID: "g1"})
		b.OnStatus(goals.Status{State: goals.RunnerRunning, Activity: "Walking to Forgotten Tower", Character: goals.Character{Class: "Sorceress", Level: 6, XP: 1000, Area: "Black Marsh", AreaID: 6}})
		b.OnGoalEvent(goals.Event{Type: goals.EvGoalStarted, GoalID: "g2"})
		b.OnGoalEvent(goals.Event{Type: goals.EvKill})
		b.OnGoalEvent(goals.Event{Type: goals.EvKill})
		b.OnStatus(goals.Status{State: goals.RunnerRunning, Activity: "Fighting", Character: goals.Character{Level: 6, XP: 1180, Area: "Black Marsh", AreaID: 6}})
		var ov OrdersView
		r.get("/api/orders", &ov)
		o := ov.Orders[1]
		if o.State != OrderActive || o.Kills != 2 || o.XPGained != 180 {
			t.Errorf("%s: progress %+v", build, o)
		}
		if up := r.uptime(); up.State != KindProgress {
			t.Errorf("%s: uptime %+v", build, up)
		}
		if v := r.omap(); !nodeOf(t, v, "bmarsh").Status.Here || !nodeOf(t, v, "ftw").Status.Order || !nodeOf(t, v, "ftw").Status.Hard {
			t.Errorf("%s: map after the walk starts: bmarsh here=%v ftw=%+v", build, nodeOf(t, v, "bmarsh").Status.Here, nodeOf(t, v, "ftw").Status)
		}
		// the guard refuses on the walk: the goal fails with the runner's message
		b.OnGoalEvent(goals.Event{Type: goals.EvGoalFailed, GoalID: "g2", Message: "area not allowed now: Forgotten Tower (20): route guard (no merc, clvl 6 < 16)"})
		r.get("/api/orders", &ov)
		if ov.Orders[1].State != OrderFailed || !strings.Contains(ov.Orders[1].Note, "no merc, clvl 6 < 16") || ov.Orders[1].XPGained != 180 {
			t.Errorf("%s: refused order: %+v", build, ov.Orders[1])
		}
		var lines []string
		for _, l := range r.journal("").Lines {
			lines = append(lines, l.Text)
		}
		j := strings.Join(lines, "\n")
		for _, want := range []string{
			"Order given: Farm Cold Plains until you cancel",
			"Order cancelled: Farm Cold Plains until you cancel",
			"Order started: Farm Forgotten Tower for 20 min",
			"Order refused by the hero's guard: Forgotten Tower (20): route guard (no merc, clvl 6 < 16)",
		} {
			if !strings.Contains(j, want) {
				t.Errorf("%s: journal lacks %q:\n%s", build, want, j)
			}
		}
		// a quest finish marks the area cleared; an unknown quest marks nothing
		b.OnGoalEvent(goals.Event{Type: goals.EvQuest, Quest: "den_of_evil", Done: true})
		b.OnGoalEvent(goals.Event{Type: goals.EvQuest, Quest: "den_of_evil", Done: true, Diff: 1})
		b.OnGoalEvent(goals.Event{Type: goals.EvQuest, Quest: "unknown_quest", Done: true})
		v := r.omap()
		if !nodeOf(t, v, "den").Status.Cleared || nodeOf(t, v, "bm").Status.Cleared || nodeOf(t, v, "bg").Status.Cleared {
			t.Errorf("%s: cleared marks wrong", build)
		}
	}
}

// Done order: the journal line carries kills and xp.
func TestOrderDoneLineHasProgress(t *testing.T) {
	r := newRig(t, "sorceress")
	r.must("/api/orders", map[string]any{"kind": "go", "area": "Blood Moor", "minutes": 1})
	r.sim.Step(90 * time.Second)
	var found string
	for _, l := range r.journal("").Lines {
		if strings.HasPrefix(l.Text, "Order done:") {
			found = l.Text
		}
	}
	if !strings.Contains(found, "Farm Blood Moor for 1 min (") || !strings.Contains(found, "kills, +") {
		t.Errorf("done line: %q", found)
	}
}

// The page: no third-party URL in html, script or css, no inline script or style, the new elements exist, the map script
// reads the svg namespace from the page.
func TestOrdersPageIsSelfContained(t *testing.T) {
	r := newRig(t, "sorceress")
	for _, p := range []string{"/watch", "/watch/app.js", "/watch/app.css"} {
		body := r.get(p, nil).Body.String()
		for _, bad := range []string{"http://", "https://", "//cdn", "@import", "style=\"", "<style"} {
			if strings.Contains(body, bad) {
				t.Errorf("%s contains %q", p, bad)
			}
		}
	}
	html := r.get("/watch", nil).Body.String()
	for _, id := range []string{`id="now"`, `id="svg"`, `id="maplist"`, `id="queue"`, `id="sheet"`, `role="dialog"`, `aria-modal="true"`, `id="scrim"`, `id="toast"`, `role="status"`, `id="viewtoggle"`} {
		if !strings.Contains(html, id) {
			t.Errorf("page lacks %s", id)
		}
	}
	if strings.Contains(html, `id="orderform"`) {
		t.Error("the old orders form is still on the page")
	}
	app := r.get("/watch/app.js", nil).Body.String()
	for _, want := range []string{"/api/orders/map", "namespaceURI", "aria-label", "aria-pressed", "role", "Escape"} {
		if !strings.Contains(app, want) {
			t.Errorf("app.js lacks %s", want)
		}
	}
}
