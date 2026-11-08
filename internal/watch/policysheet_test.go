package watch

import (
	"bytes"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"os"
	"strings"
	"testing"

	"d2lod-bot/internal/decide"
	"d2lod-bot/internal/env"
)

var updatePolicy = os.Getenv("UPDATE_POLICY") != ""

func polGolden(t *testing.T, name string, got []byte) {
	t.Helper()
	path := "testdata/policysheet/" + name
	if updatePolicy {
		os.MkdirAll("testdata/policysheet", 0o755)
		if err := os.WriteFile(path, got, 0o644); err != nil {
			t.Fatal(err)
		}
	}
	want, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	if string(want) != string(got) {
		t.Errorf("%s differs from the golden file (UPDATE_POLICY=1 rewrites it)\n got: %s\nwant: %s", name, got, want)
	}
}

type polRig struct {
	t      *testing.T
	cat    *decide.Catalogue
	tbl    *decide.PolicyTable
	store  *decide.Store
	sheet  *PolicySheet
	events *AwayLog
	now    float64
	class  string
	calls  []map[string]*float64
	dir    string
}

func newPolRig(t *testing.T, class string, withKnobs bool) *polRig {
	t.Helper()
	cat, err := decide.Shipped()
	if err != nil {
		t.Fatal(err)
	}
	tbl, err := decide.ShippedPolicy(cat)
	if err != nil {
		t.Fatal(err)
	}
	r := &polRig{t: t, cat: cat, tbl: tbl, class: class, dir: t.TempDir()}
	r.store, err = decide.Open(r.dir, "pol", cat, &decide.ManualClock{})
	if err != nil {
		t.Fatal(err)
	}
	r.events = NewAwayLog(func() float64 { return r.now })
	var k KnobSetter
	if withKnobs {
		k = func(o map[string]*float64) error { r.calls = append(r.calls, o); return nil }
	}
	r.sheet, err = NewPolicySheet(r.store, cat, tbl, func() string { return r.class }, func() float64 { return r.now }, r.events, k)
	if err != nil {
		t.Fatal(err)
	}
	return r
}

func (r *polRig) set(row, opt string) {
	r.t.Helper()
	if _, _, err := r.sheet.Set(row, opt); err != nil {
		r.t.Fatalf("set %s/%s: %v", row, opt, err)
	}
}

func (r *polRig) feed(lines string) {
	r.t.Helper()
	evs, err := ReadAwayEvents(strings.NewReader(lines))
	if err != nil {
		r.t.Fatal(err)
	}
	for _, e := range evs {
		r.events.Add(e)
	}
}

func viewJSON(t *testing.T, s *PolicySheet) []byte {
	t.Helper()
	v, err := s.View()
	if err != nil {
		t.Fatal(err)
	}
	b, _ := json.MarshalIndent(v, "", " ")
	return append(b, '\n')
}

// A journal: before the change at t=1000 one potion kind and no deaths; after it two deaths, one with a potion just before.
const polEvents = `{"t":100,"kind":"area_change","area":"Blood Moor","areaId":2}
{"t":200,"kind":"potion","potion":"Minor Healing Potion"}
{"t":300,"kind":"potion","potion":"Minor Mana Potion"}
{"t":400,"kind":"kill","monster":"Fallen","xp":10}
{"t":500,"kind":"area_change","area":"Rogue Encampment","areaId":1}
{"t":600,"kind":"drop","item":"Short Sword","quality":"normal","type":"sword"}
{"t":1100,"kind":"potion","potion":"Minor Mana Potion"}
{"t":1150,"kind":"death","monster":"Fallen","area":"Blood Moor"}
{"t":1200,"kind":"potion","potion":"Mana Potion"}
{"t":1210,"kind":"potion","potion":"Mana Potion"}
{"t":1700,"kind":"death","monster":"Corpsefire","area":"Den of Evil"}
{"t":1800,"kind":"area_change","area":"Rogue Encampment","areaId":1}
`

func TestPolicySheetGoldenTwoBuilds(t *testing.T) {
	for _, class := range []string{"sorceress", "paladin"} {
		r := newPolRig(t, class, true)
		polGolden(t, "sheet-"+class+".json", viewJSON(t, r.sheet))
	}
	r := newPolRig(t, "sorceress", true)
	v, _ := r.sheet.View()
	polGolden(t, "sheet-sorceress.html", RenderPolicy(v))
}

func TestPolicyEffectLineAndCardGolden(t *testing.T) {
	for _, class := range []string{"sorceress", "paladin"} {
		r := newPolRig(t, class, true)
		r.now = 1000
		r.feed(polEvents)
		r.set("potionmix", "allmana")
		r.now = 2000
		polGolden(t, "sheet-changed-"+class+".json", viewJSON(t, r.sheet))
		sv := r.sheet.Section().Build(nil)
		if sv == nil || len(sv.Buttons) != 1 || sv.Buttons[0].Option != "default" || sv.Buttons[0].Row != "potionmix" {
			t.Fatalf("%s: section %+v", class, sv)
		}
		b, _ := json.MarshalIndent(sv, "", " ")
		polGolden(t, "card-"+class+".json", append(b, '\n'))
		// change it back: the old card loses its button; the new card offers to undo the undo
		r.set("potionmix", sv.Buttons[0].Option)
		sv = r.sheet.Section().Build(nil)
		if sv == nil || len(sv.Buttons) != 1 || sv.Buttons[0].Option != "allmana" || len(sv.Lines) < 3 {
			t.Fatalf("%s: after change back: %+v", class, sv)
		}
	}
}

func TestPolicyCounters(t *testing.T) {
	r := newPolRig(t, "sorceress", false)
	r.feed(polEvents)
	all := r.events.Events()
	cnt := func(name string, from, to float64) (int, bool) { return policyCounters[name](all, from, to) }
	for _, c := range []struct {
		name     string
		from, to float64
		n        int
		ok       bool
	}{
		{"potions", 0, 2000, 5, true},
		{"potions_mana", 0, 2000, 4, true},
		{"potions_health", 0, 2000, 1, true},
		{"potions_rejuv", 0, 2000, 0, true},
		{"deaths", 1000, 2000, 2, true},
		{"deaths", 0, 1000, 0, true},
		{"deaths_dry", 1000, 2000, 1, true}, // the second death had no potion in the 120 s before it
		{"trips", 0, 2000, 2, true},
		{"whites", 0, 2000, 1, true},
		{"kills", 0, 2000, 1, true},
		{"repairs", 0, 2000, 0, false},
		{"gold_in", 0, 2000, 0, false},
	} {
		if n, ok := cnt(c.name, c.from, c.to); n != c.n || ok != c.ok {
			t.Errorf("%s [%v,%v] = %d,%v want %d,%v", c.name, c.from, c.to, n, ok, c.n, c.ok)
		}
	}
	r.events.Add(AwayEvent{T: 700, Kind: AwayGold, Gold: 40})
	all = r.events.Events()
	if n, ok := cnt("gold_in", 0, 2000); n != 40 || !ok {
		t.Errorf("gold_in = %d,%v", n, ok)
	}
}

func TestPolicySetAppliesRuleAndSurvivesReload(t *testing.T) {
	for _, class := range []string{"sorceress", "paladin"} {
		r := newPolRig(t, class, true)
		r.set("repair", "never") // a suboptimal value is accepted
		v, _ := r.sheet.View()
		var row *PolicyRowView
		for i := range v.Rows {
			if v.Rows[i].ID == "repair" {
				row = &v.Rows[i]
			}
		}
		if row == nil || row.Current != "never" || !row.Changed {
			t.Fatalf("%s: row %+v", class, row)
		}
		flagged := false
		for _, o := range row.Options {
			if o.ID == "never" {
				flagged = o.Suboptimal && o.Risk != "" && o.Current
			}
		}
		if !flagged {
			t.Fatalf("%s: the poor option is not flagged with its risk", class)
		}
		ctx, err := r.cat.Build("t1.repair", decide.Obs{Class: class, Cat: map[string]string{"act": "1", "diff": "normal", "hc": "normal"}, Num: map[string]float64{"gold": 50, "durability": 3}}, 1, nil)
		if err != nil {
			t.Fatal(err)
		}
		if d, _ := r.store.AskWithDefault(ctx, "repair"); d.Answer != "skip" || d.Source != "policy" || d.Thought != nil {
			t.Fatalf("%s: %+v", class, d)
		}
		// reopen: a new process
		st2, err := decide.Open(r.dir, "pol", r.cat, &decide.ManualClock{})
		if err != nil {
			t.Fatal(err)
		}
		s2, _ := NewPolicySheet(st2, r.cat, r.tbl, func() string { return class }, func() float64 { return 5 }, r.events, nil)
		if d, _ := st2.AskWithDefault(ctx, "repair"); d.Answer != "skip" || d.Source != "policy" {
			t.Fatalf("%s after reload: %+v", class, d)
		}
		v2, _ := s2.View()
		for _, rv := range v2.Rows {
			if rv.ID == "repair" && (rv.Current != "never" || rv.Effect != nil || !strings.Contains(rv.EffectNote, "before this run")) {
				t.Fatalf("%s reload row: %+v", class, rv)
			}
		}
	}
}

func TestPolicyKnobPath(t *testing.T) {
	r := newPolRig(t, "sorceress", true)
	r.set("potionmix", "allmana")
	if len(r.calls) != 1 || r.calls[0]["combat.drinkLifePct"] == nil || *r.calls[0]["combat.drinkLifePct"] != 35 {
		t.Fatalf("calls %v", r.calls)
	}
	r.set("potionmix", "allhealth") // takes the first option's knob back and sets the new ones
	last := r.calls[len(r.calls)-1]
	if v, ok := last["combat.drinkLifePct"]; !ok || v != nil {
		t.Fatalf("old knob not reset: %v", last)
	}
	if v := last["town.tripMinMPPotions"]; v == nil || *v != 0 {
		t.Fatalf("new knob not set: %v", last)
	}
	r.set("potionmix", "default")
	last = r.calls[len(r.calls)-1]
	if v, ok := last["combat.drinkManaPct"]; !ok || v != nil {
		t.Fatalf("default does not reset: %v", last)
	}
	n := len(r.calls)
	r.set("potionmix", "default") // same option: nothing happens
	if len(r.calls) != n {
		t.Fatal("an unchanged row touched the knobs")
	}
	// Reapply puts the stored option's knobs back
	r.set("belt", "half")
	r.calls = nil
	if err := r.sheet.Reapply(); err != nil {
		t.Fatal(err)
	}
	if len(r.calls) != 1 || *r.calls[0]["town.beltFillPct"] != 50 {
		t.Fatalf("reapply %v", r.calls)
	}

	// no tuner: a knob option is refused and nothing is logged; a rule-only option works
	nk := newPolRig(t, "sorceress", false)
	_, _, err := nk.sheet.Set("belt", "half")
	if !errors.Is(err, ErrPolicyNoKnobs) || nk.store.PolicyOption("belt") != "" || len(nk.store.PolicyLog()) != 0 {
		t.Fatalf("no tuner: %v log %d", err, len(nk.store.PolicyLog()))
	}
	nk.set("repair", "never")

	// a tuner that refuses: nothing is logged
	fk := newPolRig(t, "sorceress", true)
	fk.sheet.knobs = func(map[string]*float64) error { return errors.New("invalid tuning edit") }
	if _, _, err := fk.sheet.Set("belt", "half"); err == nil || len(fk.store.PolicyLog()) != 0 {
		t.Fatalf("refusing tuner: %v", err)
	}
}

func TestPolicyKnobKeysAreRealKnobs(t *testing.T) {
	cat, _ := decide.Shipped()
	tbl, err := decide.ShippedPolicy(cat)
	if err != nil {
		t.Fatal(err)
	}
	byKey := map[string]env.Knob{}
	for _, k := range env.Knobs() {
		byKey[k.Key] = k
	}
	n := 0
	for _, r := range tbl.Rows {
		for _, o := range r.Options {
			for k, v := range o.Knobs {
				n++
				kn, ok := byKey[k]
				if !ok {
					t.Errorf("%s/%s: %q is not a knob", r.ID, o.ID, k)
					continue
				}
				if v < kn.Min || v > kn.Max {
					t.Errorf("%s/%s: %s = %v outside %v..%v", r.ID, o.ID, k, v, kn.Min, kn.Max)
				}
			}
		}
	}
	if n == 0 {
		t.Fatal("no knob options in the table")
	}
}

func TestPolicyErrorsNameTheThing(t *testing.T) {
	r := newPolRig(t, "sorceress", true)
	if _, _, err := r.sheet.Set("nosuchrow", "x"); !errors.Is(err, ErrPolicyRow) || !strings.Contains(err.Error(), "nosuchrow") {
		t.Errorf("row: %v", err)
	}
	if _, _, err := r.sheet.Set("repair", "melt"); !errors.Is(err, ErrPolicyOption) || !strings.Contains(err.Error(), "melt") || !strings.Contains(err.Error(), "repair") {
		t.Errorf("option: %v", err)
	}
	r.class = "necromancer"
	if _, err := r.sheet.View(); err == nil || !strings.Contains(err.Error(), `"necromancer"`) {
		t.Errorf("class: %v", err)
	}
	r.class = ""
	if _, _, err := r.sheet.Set("repair", "never"); !errors.Is(err, ErrPolicyNoClass) {
		t.Errorf("no class: %v", err)
	}
	// a row for one class only refuses the other
	cat, _ := decide.Shipped()
	tbl, err := decide.ParsePolicy([]byte(`{"version":1,"rows":[{"id":"pal","label":"L","help":"H","classes":["paladin"],"default":"d","options":[{"id":"d","label":"D"},{"id":"x","label":"X","knobs":{"town.sellJunk":0}}]}]}`), cat)
	if err != nil {
		t.Fatal(err)
	}
	for class, wantErr := range map[string]bool{"sorceress": true, "paladin": false} {
		st, _ := decide.New(cat, "p", &decide.ManualClock{})
		ps, _ := NewPolicySheet(st, cat, tbl, func() string { return class }, func() float64 { return 0 }, NewAwayLog(nil), func(map[string]*float64) error { return nil })
		_, _, err := ps.Set("pal", "x")
		if wantErr != errors.Is(err, ErrPolicyScope) || (!wantErr && err != nil) {
			t.Errorf("%s: %v", class, err)
		}
	}
	bad := &decide.PolicyTable{}
	json.Unmarshal([]byte(`{"version":1,"rows":[{"id":"r","label":"L","help":"H","default":"d","options":[{"id":"d","label":"D"}],"effect":[{"counter":"mystery","label":"M"}]}]}`), bad)
	st, _ := decide.New(cat, "p", &decide.ManualClock{})
	if _, err := NewPolicySheet(st, cat, bad, func() string { return "sorceress" }, func() float64 { return 0 }, NewAwayLog(nil), nil); err == nil || !strings.Contains(err.Error(), `"mystery"`) {
		t.Errorf("counter: %v", err)
	}
}

func polPost(h http.Handler, hdr map[string]string, body string) *httptest.ResponseRecorder {
	req := httptest.NewRequest("POST", "/api/policy", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	for k, v := range hdr {
		req.Header.Set(k, v)
	}
	w := httptest.NewRecorder()
	h.ServeHTTP(w, req)
	return w
}

func TestPolicyHTTPGates(t *testing.T) {
	r := newPolRig(t, "sorceress", true)
	opts := AnswerOptions{Token: "tok", Origins: []string{"http://ok.example"}}
	ans := r.sheet.PolicyAnswerHandler(opts)
	auth := map[string]string{"Authorization": "Bearer tok"}
	good := `{"row":"repair","option":"never"}`
	for name, c := range map[string]struct {
		hdr  map[string]string
		body string
		code int
	}{
		"no token":      {nil, good, 401},
		"wrong token":   {map[string]string{"Authorization": "Bearer nope"}, good, 401},
		"bad origin":    {map[string]string{"Authorization": "Bearer tok", "Origin": "http://evil.example"}, good, 403},
		"content type":  {map[string]string{"Authorization": "Bearer tok", "Content-Type": "text/plain"}, good, 415},
		"unknown field": {auth, `{"row":"repair","option":"never","x":1}`, 400},
		"two objects":   {auth, good + good, 400},
		"unknown row":   {auth, `{"row":"zzz","option":"never"}`, 404},
		"unknown opt":   {auth, `{"row":"repair","option":"melt"}`, 400},
	} {
		if w := polPost(ans, c.hdr, c.body); w.Code != c.code {
			t.Errorf("%s: %d %s, want %d", name, w.Code, w.Body, c.code)
		}
	}
	if len(r.store.PolicyLog()) != 0 {
		t.Fatal("a refused request changed the sheet")
	}
	w := polPost(ans, map[string]string{"Authorization": "Bearer tok", "Origin": "http://ok.example"}, `{"row":"repair","option":"melt"}`)
	var rep policyReply
	json.Unmarshal(w.Body.Bytes(), &rep)
	if len(rep.Allowed) != 4 || !strings.Contains(rep.Error, "melt") {
		t.Errorf("allowed %v error %q", rep.Allowed, rep.Error)
	}
	if w := polPost(ans, auth, good); w.Code != 200 || !strings.Contains(w.Body.String(), `"changed":true`) {
		t.Fatalf("good: %d %s", w.Code, w.Body)
	}
	if w := polPost(ans, auth, good); w.Code != 200 || strings.Contains(w.Body.String(), `"changed"`) {
		t.Fatalf("same again: %d %s", w.Code, w.Body)
	}
	if r.store.PolicyOption("repair") != "never" {
		t.Fatal("not applied")
	}
	req := httptest.NewRequest("GET", "/api/policy", nil)
	req.Header.Set("Authorization", "Bearer tok")
	wg := httptest.NewRecorder()
	ans.ServeHTTP(wg, req)
	if wg.Code != 405 {
		t.Errorf("GET on the answer route: %d", wg.Code)
	}

	// the viewer is read-only and serves page, JSON and script
	view := r.sheet.PolicyHandler()
	for _, p := range PolicyRoutes {
		w := httptest.NewRecorder()
		view.ServeHTTP(w, httptest.NewRequest("GET", p, nil))
		if w.Code != 200 || w.Header().Get("Content-Security-Policy") == "" && p != "/api/policy" {
			t.Errorf("GET %s: %d", p, w.Code)
		}
	}
	w = httptest.NewRecorder()
	view.ServeHTTP(w, httptest.NewRequest("POST", "/api/policy", strings.NewReader(good)))
	if w.Code != 405 {
		t.Errorf("POST on the viewer: %d", w.Code)
	}
	r.class = ""
	w = httptest.NewRecorder()
	view.ServeHTTP(w, httptest.NewRequest("GET", "/watch/policy", nil))
	if w.Code != 503 {
		t.Errorf("no class page: %d", w.Code)
	}
}

func TestPolicySectionInReturnScreen(t *testing.T) {
	r := newPolRig(t, "sorceress", true)
	r.now = 1000
	r.feed(polEvents)
	r.set("repair", "never")
	r.now = 1900
	reg := DefaultSections()
	reg.Register(PolicySectionID, r.sheet.Section())
	d := BuildReturn(ReturnInput{Events: r.events.Events(), Since: 0, Now: 1900, Hero: HeroInfo{Class: "Sorceress"}, Sections: reg})
	html := string(RenderReturn(d, false))
	for _, want := range []string{`data-section="policy"`, "Policy changes", `data-act="policy"`, `data-row="repair"`, `data-option="default"`, "Change it back"} {
		if !strings.Contains(html, want) {
			t.Errorf("page lacks %q", want)
		}
	}
	if !strings.Contains(string(RenderReturn(BuildReturn(ReturnInput{Now: 10}), false)), "data-act=\"policy\"") == false {
		t.Error("a page without a change shows a policy button")
	}
	if !bytes.Contains([]byte(returnJS), []byte("a==='policy'")) {
		t.Error("the return script does not post policy changes")
	}
}
