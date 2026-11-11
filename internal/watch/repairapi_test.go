package watch

import (
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"os"
	"regexp"
	"strings"
	"testing"
	"time"

	"d2lod-bot/internal/decide"
	"d2lod-bot/internal/repair"
	"d2lod-bot/internal/unlock"
)

type fakeRepair struct {
	reqs, cancels int
	err           error
}

func (f *fakeRepair) View() any { return map[string]any{"schema": 1} }
func (f *fakeRepair) Request() (any, error) {
	f.reqs++
	return map[string]any{"state": "requested"}, f.err
}
func (f *fakeRepair) Cancel() error { f.cancels++; return f.err }

func TestRepairRoutesGateAndReplyWithReasons(t *testing.T) {
	api := NewRepairAPI()
	o := AnswerOptions{Token: "tok", Origins: []string{"http://ok.test"}}
	post := api.PostHandler(o)
	auth := map[string]string{"Authorization": "Bearer tok"}

	if w := invPost(post, "/api/repair/request", `{}`, auth); w.Code != 503 {
		t.Errorf("desk not bound: %d", w.Code)
	}
	f := &fakeRepair{}
	api.Bind(f)
	if w := invPost(post, "/api/repair/request", `{}`, auth); w.Code != 200 || f.reqs != 1 {
		t.Errorf("request: %d %s", w.Code, w.Body)
	}
	if w := invPost(post, "/api/repair/cancel", `{}`, auth); w.Code != 200 || f.cancels != 1 {
		t.Errorf("cancel: %d", w.Code)
	}
	if w := invPost(post, "/api/repair/request", `{}`, nil); w.Code != 401 {
		t.Errorf("no token: %d", w.Code)
	}
	if w := invPost(post, "/api/repair/request", `{}`, map[string]string{"Authorization": "Bearer tok", "Origin": "http://evil.test"}); w.Code != 403 {
		t.Errorf("foreign origin: %d", w.Code)
	}
	if w := invPost(post, "/api/repair/request", `{"x":1}`, auth); w.Code != 400 {
		t.Errorf("unknown field: %d", w.Code)
	}
	f.err = &repair.Refusal{Code: "undamaged", Reason: "nothing worn is damaged"}
	if w := invPost(post, "/api/repair/request", `{}`, auth); w.Code != 409 || !strings.Contains(w.Body.String(), "undamaged") {
		t.Errorf("refusal: %d %s", w.Code, w.Body)
	}
	f.err = errors.New("disk full")
	if w := invPost(post, "/api/repair/cancel", `{}`, auth); w.Code != 500 || strings.Contains(w.Body.String(), "disk") {
		t.Errorf("internal error leaks detail: %d %s", w.Code, w.Body)
	}
	view := api.Handler()
	for _, p := range []string{"/api/repair", "/watch/gear.js"} {
		w := httptest.NewRecorder()
		view.ServeHTTP(w, httptest.NewRequest("GET", p, nil))
		if w.Code != 200 {
			t.Errorf("GET %s: %d", p, w.Code)
		}
	}
	w := httptest.NewRecorder()
	view.ServeHTTP(w, httptest.NewRequest("POST", "/api/repair", nil))
	if w.Code != http.StatusMethodNotAllowed {
		t.Errorf("viewer accepts POST: %d", w.Code)
	}
}

// The gear panel of the command centre replaced the strip script on /watch (gear.js stays served until the cleanup PR).
func TestGearScriptIsLinkedAndSafe(t *testing.T) {
	html, err := webFS.ReadFile("web/watch.html")
	if err != nil || strings.Contains(string(html), `src="/watch/gear.js"`) {
		t.Fatalf("watch.html must not load gear.js next to the gear panel: %v", err)
	}
	if p, err := webFS.ReadFile("web/panels/gear/panel.js"); err != nil || !strings.Contains(string(p), "/api/repair/request") {
		t.Fatalf("the gear panel lacks Repair now: %v", err)
	}
	js, _ := webFS.ReadFile("web/gear.js")
	if strings.Contains(string(js), "innerHTML") {
		t.Error("gear.js must write text through textContent only")
	}
}

// The shipped repair row: manual is the default and has no unlock; the four other modes name ladder rows that exist,
// are bought with tally and cost more as they get more eager (U: the prices are guesses).
func TestShippedRepairRowMatchesModesAndLadder(t *testing.T) {
	cat, err := decide.Shipped()
	if err != nil {
		t.Fatal(err)
	}
	tbl, err := decide.ShippedPolicy(cat)
	if err != nil {
		t.Fatal(err)
	}
	row, err := tbl.Row("repair")
	if err != nil {
		t.Fatal(err)
	}
	modes, err := repair.Shipped()
	if err != nil {
		t.Fatal(err)
	}
	if row.Default != "manual" || row.Default != modes.Default {
		t.Fatalf("default %q / %q", row.Default, modes.Default)
	}
	if len(row.Options) != len(modes.Modes) {
		t.Fatalf("row has %d options, modes %d", len(row.Options), len(modes.Modes))
	}
	l, err := unlock.Shipped()
	if err != nil {
		t.Fatal(err)
	}
	for i, o := range row.Options {
		m := modes.Modes[i]
		if o.ID != m.ID || o.Unlock != m.Unlock {
			t.Errorf("option %d: row %q/%q, modes %q/%q", i, o.ID, o.Unlock, m.ID, m.Unlock)
		}
		if o.Unlock == "" {
			continue
		}
		e, err := l.Entry(o.Unlock)
		if err != nil || e.Trigger.Kind != "tally_spend" || e.Price <= 0 || e.Group != "ui" {
			t.Errorf("unlock %q: %+v %v", o.Unlock, e, err)
		}
	}
}

// Only manual is available at the start: a locked option is refused with a code, shown as locked with its price, and
// opens once the row is bought. Both classes (the rule is class-free).
func TestPolicyRepairModesLockedUntilBought(t *testing.T) {
	for _, class := range []string{"sorceress", "paladin"} {
		cat, _ := decide.Shipped()
		tbl, _ := decide.ShippedPolicy(cat)
		st, err := decide.Open(t.TempDir(), "pol", cat, &decide.ManualClock{})
		if err != nil {
			t.Fatal(err)
		}
		sheet, err := NewPolicySheet(st, cat, tbl, func() string { return class }, func() float64 { return 1 }, NewAwayLog(func() float64 { return 1 }), nil)
		if err != nil {
			t.Fatal(err)
		}
		sheet.SetStarts(func(string) (map[string]string, error) { return nil, nil })
		l, err := unlock.Shipped()
		if err != nil {
			t.Fatal(err)
		}
		eng, err := unlock.Open(t.TempDir(), class, l, func() time.Time { return time.Unix(1_800_000_000, 0) })
		if err != nil {
			t.Fatal(err)
		}
		sheet.SetUnlocker(func(id string) (bool, int64, string) {
			if eng.Allowed(id) {
				return true, 0, ""
			}
			for _, s := range eng.States("ui") {
				if s.ID == id {
					return false, s.Price, s.Teaser
				}
			}
			return false, 0, ""
		})
		if sheet.CurrentOption("repair") != "manual" {
			t.Fatalf("%s: start option %q", class, sheet.CurrentOption("repair"))
		}
		if _, _, err := sheet.Set("repair", "hour"); !errors.Is(err, ErrPolicyLocked) {
			t.Fatalf("%s: locked option accepted: %v", class, err)
		}
		w := polPost(sheet.PolicyAnswerHandler(AnswerOptions{Token: "tok"}), map[string]string{"Authorization": "Bearer tok"}, `{"row":"repair","option":"half"}`)
		if w.Code != http.StatusForbidden || !strings.Contains(w.Body.String(), `"locked"`) {
			t.Errorf("%s: locked option over HTTP: %d %s", class, w.Code, w.Body)
		}
		v, _ := sheet.View()
		for _, r := range v.Rows {
			if r.ID != "repair" {
				continue
			}
			for _, o := range r.Options {
				if want := o.ID != "manual"; o.Locked != want || (want && o.Price <= 0) {
					t.Errorf("%s: option %s locked=%v price=%d", class, o.ID, o.Locked, o.Price)
				}
			}
		}
		if _, err := eng.Buy("repair_hour", okSpender{}); err != nil {
			t.Fatal(err)
		}
		if _, changed, err := sheet.Set("repair", "hour"); err != nil || !changed {
			t.Fatalf("%s: bought option refused: %v", class, err)
		}
		// the controller follows the row and a still-locked mode acts as manual
		c, _ := repair.New(repair.Config{Mode: func() string { return sheet.CurrentOption("repair") }, Allowed: eng.Allowed})
		if c.ModeID() != "hour" {
			t.Errorf("%s: controller mode %q", class, c.ModeID())
		}
		if _, _, err := sheet.Set("repair", "always"); !errors.Is(err, ErrPolicyLocked) {
			t.Errorf("%s: always opened by another purchase: %v", class, err)
		}
	}
}

func TestGearAlertsOncePerChange(t *testing.T) {
	r, _ := alertsRig(t, "Paladin", nil)
	a := r.fun.Alerts
	conds := []repair.Cond{}
	a.SetGear(func() []repair.Cond { return conds })
	low, _ := repair.Classify("torso", "Plate", 20, 100, 30)
	broken, _ := repair.Classify("rhand", "Sword", 0, 100, 30)
	ok, _ := repair.Classify("head", "Cap", 90, 100, 30)

	conds = []repair.Cond{ok}
	a.Tick()
	if len(alertsOf(r, 0).Alerts) != 0 {
		t.Fatal("healthy gear raised an alert")
	}
	conds = []repair.Cond{ok, low}
	a.Tick()
	a.Tick() // the same state again: nothing new
	v := alertsOf(r, 0)
	if len(v.Alerts) != 1 || v.Alerts[0].Rule != "gear_low" || !v.Alerts[0].Free || !strings.Contains(v.Alerts[0].Text, "Plate") {
		t.Fatalf("low: %+v", v.Alerts)
	}
	conds = []repair.Cond{ok, low, broken}
	a.Tick()
	v = alertsOf(r, 0)
	if len(v.Alerts) != 2 || v.Alerts[1].Rule != "gear_broken" || !strings.Contains(v.Alerts[1].Text, "Sword") {
		t.Fatalf("broken: %+v", v.Alerts)
	}
}

// The page CSP is style-src 'self': a <style> element injected by script is blocked and the strip loses its layout
// (the "Gearhead 0/18 BROKENtorso" bug). Embedded JS must not create style elements or set style attributes.
func TestEmbeddedJSInjectsNoStyle(t *testing.T) {
	ents, err := webFS.ReadDir("web")
	if err != nil {
		t.Fatal(err)
	}
	bad := regexp.MustCompile(`createElement\(\s*['"]style['"]|setAttribute\(\s*['"]style['"]|insertAdjacentHTML|<style|\sstyle\s*=`)
	n := 0
	for _, e := range ents {
		if !strings.HasSuffix(e.Name(), ".js") {
			continue
		}
		n++
		b, _ := webFS.ReadFile("web/" + e.Name())
		if m := bad.Find(b); m != nil {
			t.Errorf("web/%s: %q is blocked by the page CSP; put the rule in app.css", e.Name(), m)
		}
	}
	if n == 0 {
		t.Fatal("no embedded js found")
	}
}

// A poll must not move the page: focus restored after a redraw scrolls the element into view (the "scroll keeps
// resetting" bug), so every programmatic focus in the polling scripts passes preventScroll. The modal sheet and
// reveal dialog focus (user-initiated, once) are allowed to scroll.
func TestPollingJSFocusKeepsScroll(t *testing.T) {
	re := regexp.MustCompile(`\.focus\(([^)]*)\)`)
	for _, name := range []string{"shell.js", "inventory.js", "gear.js"} {
		b, err := webFS.ReadFile("web/" + name)
		if err != nil {
			t.Fatal(err)
		}
		for _, m := range re.FindAllSubmatch(b, -1) {
			if !strings.Contains(string(m[1]), "preventScroll") {
				t.Errorf("web/%s: %s scrolls the page on a poll; use focus({ preventScroll: true })", name, m[0])
			}
		}
	}
	b, _ := webFS.ReadFile("web/app.js")
	if !strings.Contains(string(b), "items2[keep[1]].focus({ preventScroll: true })") {
		t.Error("web/app.js: renderOrders must restore focus with preventScroll")
	}
}

// Redraws that empty and refill lists must restore the page and inner scroll offsets (#journal is a scroll box).
func TestRedrawKeepsScrollOffsets(t *testing.T) {
	app, _ := webFS.ReadFile("web/app.js")
	if !strings.Contains(string(app), "keepScroll(function () {") || !strings.Contains(string(app), "renderJournal(a[1])") {
		t.Error("web/app.js: refresh must run its renders inside keepScroll")
	}
	sh, _ := webFS.ReadFile("web/shell.js")
	if !strings.Contains(string(sh), "window.scrollTo(x, y)") || !strings.Contains(string(sh), "n.scrollTop = t") {
		t.Error("web/shell.js: keepFocus must restore page and inner scroll offsets")
	}
}

// The repair reply only grows: every field of the schema 1 reply (testdata/compat/repair-v1.json, never edited) is in
// today's reply, the old gear panel logic still finds state/note/short/need, and the new fields sit beside them.
func TestRepairReplyKeepsEverySchema1Field(t *testing.T) {
	var v1 map[string]any
	b, err := os.ReadFile("testdata/compat/repair-v1.json")
	if err != nil {
		t.Fatal(err)
	}
	if err := json.Unmarshal(b, &v1); err != nil {
		t.Fatal(err)
	}
	gear := func() []repair.Cond {
		c, _ := repair.Classify("torso", "Plate", 7, 100, 30)
		return []repair.Cond{c}
	}
	rc, err := repair.New(repair.Config{Gear: gear, Estimate: func() int { return 116 }, Gold: func() int { return 75 }})
	if err != nil {
		t.Fatal(err)
	}
	rc.Request()
	rc.Short(41, 116)
	rc.SetPhase(repair.PhaseQueued, "n")
	cur, _ := json.Marshal(rc.View())
	var now map[string]any
	if err := json.Unmarshal(cur, &now); err != nil {
		t.Fatal(err)
	}
	var walk func(path string, old, cur any)
	walk = func(path string, old, cur any) {
		switch o := old.(type) {
		case map[string]any:
			c, ok := cur.(map[string]any)
			if !ok {
				t.Errorf("%s: not an object any more", path)
				return
			}
			for k, ov := range o {
				cv, ok := c[k]
				if !ok {
					t.Errorf("%s.%s was removed (a breaking change: add a schema version instead)", path, k)
					continue
				}
				walk(path+"."+k, ov, cv)
			}
		case []any:
			c, ok := cur.([]any)
			if !ok || len(c) == 0 || len(o) == 0 {
				return
			}
			walk(path+"[0]", o[0], c[0])
		}
	}
	walk("view", v1, now)
	if now["schema"].(float64) < 2 || now["trip"] == nil {
		t.Errorf("schema 2 must carry the trip: %v", now["schema"])
	}
}

// The gear panel's repair part against the controller's own replies, both classes (node, skipped without it).
func TestGearPanelRepairTripDOMGolden(t *testing.T) {
	runNode(t, "testdata/gear_repairtrip_dom.js", "web/panels/gear/panel.js")
}

// The inventory page's repair card for the same replies.
func TestInventoryRepairCardDOMGolden(t *testing.T) {
	runNode(t, "testdata/inventory_repair_dom.js", "web/inventory.js")
}

// A refusal of the new kind answers 409 with its code, like the old ones.
func TestRepairRoutesAnswerTheNewRefusals(t *testing.T) {
	for _, code := range []string{"active", "cooldown", "pact"} {
		f := &fakeRepair{err: &repair.Refusal{Code: code, Reason: "r " + code}}
		api := NewRepairAPI()
		api.Bind(f)
		post := api.PostHandler(AnswerOptions{Token: "tok"})
		w := invPost(post, "/api/repair/request", `{}`, map[string]string{"Authorization": "Bearer tok"})
		if w.Code != 409 || !strings.Contains(w.Body.String(), code) {
			t.Errorf("%s: %d %s", code, w.Code, w.Body.String())
		}
	}
}
