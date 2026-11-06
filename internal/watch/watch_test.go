package watch

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"regexp"
	"strings"
	"testing"
	"time"

	"d2lod-bot/internal/decide"
)

const origin = "http://viewer.example:8891"

type rig struct {
	t    *testing.T
	sim  *Sim
	view http.Handler
	ans  http.Handler
}

func newRig(t *testing.T, build string) *rig {
	t.Helper()
	s, err := NewSim(build, 1)
	if err != nil {
		t.Fatal(err)
	}
	return &rig{t: t, sim: s, view: s.Hub.ViewHandler(), ans: s.Hub.AnswerHandler(AnswerOptions{Origins: []string{origin}})}
}

func (r *rig) get(path string, v any) *httptest.ResponseRecorder {
	r.t.Helper()
	w := httptest.NewRecorder()
	r.view.ServeHTTP(w, httptest.NewRequest("GET", path, nil))
	if v != nil && w.Code == 200 {
		if err := json.Unmarshal(w.Body.Bytes(), v); err != nil {
			r.t.Fatalf("GET %s: %v\n%s", path, err, w.Body.String())
		}
	}
	return w
}

func (r *rig) postRaw(h http.Handler, path string, body string, hdr map[string]string) *httptest.ResponseRecorder {
	req := httptest.NewRequest("POST", path, strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	for k, v := range hdr {
		req.Header.Set(k, v)
	}
	w := httptest.NewRecorder()
	h.ServeHTTP(w, req)
	return w
}

// post sends a JSON body to the answer route and returns the status and decoded reply.
func (r *rig) post(path string, body any) (int, reply) {
	r.t.Helper()
	b, _ := json.Marshal(body)
	w := r.postRaw(r.ans, path, string(b), map[string]string{"Origin": origin})
	var rp reply
	if err := json.Unmarshal(w.Body.Bytes(), &rp); err != nil {
		r.t.Fatalf("POST %s: %v\n%s", path, err, w.Body.String())
	}
	return w.Code, rp
}

func (r *rig) must(path string, body any) reply {
	r.t.Helper()
	code, rp := r.post(path, body)
	if code != 200 || !rp.OK {
		r.t.Fatalf("POST %s %v: %d %+v", path, body, code, rp)
	}
	return rp
}

func (r *rig) thoughts() ThoughtsView { var v ThoughtsView; r.get("/api/thoughts", &v); return v }
func (r *rig) rules() RulesView       { var v RulesView; r.get("/api/rules", &v); return v }
func (r *rig) journal(q string) JournalView {
	var v JournalView
	r.get("/api/journal"+q, &v)
	return v
}
func (r *rig) uptime() UptimeView { var v UptimeView; r.get("/api/uptime", &v); return v }

// fight puts the hero to work: a long "fight here" order.
func (r *rig) fight() {
	r.t.Helper()
	r.must("/api/orders", map[string]any{"kind": "fight", "minutes": 200})
	r.sim.Step(time.Second)
}

// nextThought steps the sim until a thought is open and returns it.
func (r *rig) nextThought() ThoughtItem {
	r.t.Helper()
	for i := 0; i < 200; i++ {
		if v := r.thoughts(); len(v.Thoughts) > 0 {
			return v.Thoughts[0]
		}
		r.sim.Step(5 * time.Second)
	}
	r.t.Fatal("no thought opened")
	return ThoughtItem{}
}

var builds = []string{"sorceress", "paladin"}

func TestEveryFieldValueHasAPhrase(t *testing.T) {
	cat, err := decide.Shipped()
	if err != nil {
		t.Fatal(err)
	}
	for name, f := range cat.Fields {
		vals := f.Values
		if f.Kind != "enum" {
			vals = f.Labels
		}
		for _, v := range vals {
			if fieldText[name][v] == "" {
				t.Errorf("field %s value %s has no plain-language phrase", name, v)
			}
		}
	}
	if got := PlainContext("t1.repair|act=1,diff=normal,durability=20-30,gold=500-1500,hc=normal"); got !=
		"in act 1, on Normal, gear worn, gold 500 to 1500, softcore hero" {
		t.Errorf("plain context: %q", got)
	}
}

func TestThoughtShowsFixedButtonsAndContext(t *testing.T) {
	for _, b := range builds {
		t.Run(b, func(t *testing.T) {
			r := newRig(t, b)
			r.fight()
			th := r.nextThought()
			cat, _ := decide.Shipped()
			p, _ := cat.Point(pointRetreat)
			if th.Kind != "question" || th.Question != p.Question || th.Point != pointRetreat {
				t.Fatalf("thought %+v", th)
			}
			if len(th.Choices) != len(p.Answers)+1 || th.Choices[len(th.Choices)-1].ID != decide.NotSure {
				t.Fatalf("choices %+v", th.Choices)
			}
			for i, a := range p.Answers {
				if th.Choices[i].ID != a.ID || th.Choices[i].Label != a.Label {
					t.Errorf("choice %d: %+v", i, th.Choices[i])
				}
			}
			if !strings.Contains(th.Context, "life") || th.Urgency == "" || !strings.HasPrefix(th.Meanwhile, "Meanwhile I: ") {
				t.Errorf("context/urgency/meanwhile missing: %+v", th)
			}
			if th.RemainingSec <= 0 || th.RemainingSec > float64(r.thoughts().FadeSec) {
				t.Errorf("remaining %v", th.RemainingSec)
			}
		})
	}
}

func TestFadeWritesMissedToJournal(t *testing.T) {
	r := newRig(t, "sorceress")
	r.fight()
	th := r.nextThought()
	// Time passes without a Tick: a GET shows the thought gone but writes nothing.
	logLen := len(r.sim.Store.Log())
	r.sim.Clock.Advance(time.Duration(th.RemainingSec+1) * time.Second)
	if v := r.thoughts(); len(v.Thoughts) != 0 {
		t.Fatalf("expired thought still in the feed: %+v", v.Thoughts)
	}
	r.journal("")
	if n := len(r.sim.Store.Log()); n != logLen {
		t.Fatalf("a GET wrote to the store: %d -> %d", logLen, n)
	}
	if got := r.journal("?kind=missed"); len(got.Lines) != 0 {
		t.Fatalf("missed before the fade was recorded: %+v", got.Lines)
	}
	if n, err := r.sim.Hub.Tick(); err != nil || n != 1 {
		t.Fatalf("tick: %d %v", n, err)
	}
	got := r.journal("?kind=missed")
	if len(got.Lines) != 1 || !strings.HasPrefix(got.Lines[0].Text, "Missed: "+th.Question) || got.Lines[0].Thought != th.ID {
		t.Fatalf("missed line: %+v", got.Lines)
	}
	// The next occurrence makes a new thought; nothing was counted.
	if rec, _ := r.sim.Store.Record(th.Key); rec.Streak != 0 {
		t.Errorf("a faded thought counted: %+v", rec)
	}
	if nt := r.nextThought(); nt.ID == th.ID {
		t.Error("same thought id again")
	}
}

func TestAnswerRouteValidation(t *testing.T) {
	r := newRig(t, "sorceress")
	r.fight()
	th := r.nextThought()
	id := th.ID

	for _, c := range []struct {
		name, path string
		body       any
		code       int
		errCode    string
	}{
		{"not an offered answer", "/api/answer", map[string]any{"thought": id, "answer": "cast-nova"}, 400, "bad-choice"},
		{"free text", "/api/answer", map[string]any{"thought": id, "answer": "<script>alert(1)</script>"}, 400, "bad-choice"},
		{"empty answer", "/api/answer", map[string]any{"thought": id, "answer": ""}, 400, "bad-choice"},
		{"unknown thought", "/api/answer", map[string]any{"thought": 9999, "answer": "drink"}, 404, "not-found"},
		{"no thought id", "/api/answer", map[string]any{"answer": "drink"}, 400, "bad-request"},
		{"confirm on a question", "/api/confirm", map[string]any{"thought": id, "yes": true}, 400, "wrong-kind"},
		{"confirm without yes", "/api/confirm", map[string]any{"thought": id}, 400, "bad-request"},
		{"revoke of nothing", "/api/revoke", map[string]any{"key": "t1.repair|act=1"}, 404, "not-found"},
		{"revoke-all needs confirm", "/api/revoke-all", map[string]any{}, 400, "bad-request"},
		{"unknown order kind", "/api/orders", map[string]any{"kind": "dance"}, 400, "bad-order"},
		{"unknown area", "/api/orders", map[string]any{"kind": "go", "area": "Moo Moo Farm", "minutes": 5}, 400, "bad-order"},
		{"minutes too long", "/api/orders", map[string]any{"kind": "fight", "minutes": 9999}, 400, "bad-order"},
		{"home with minutes", "/api/orders", map[string]any{"kind": "home", "minutes": 5}, 400, "bad-order"},
		{"cancel unknown order", "/api/orders/cancel", map[string]any{"id": 77}, 404, "not-found"},
	} {
		code, rp := r.post(c.path, c.body)
		if code != c.code || rp.OK || rp.Code != c.errCode {
			t.Errorf("%s: got %d %+v, want %d %s", c.name, code, rp, c.code, c.errCode)
		}
	}
	if _, rp := r.post("/api/answer", map[string]any{"thought": id, "answer": "x"}); len(rp.Allowed) != 6 {
		t.Errorf("a refused answer lists the allowed ones: %v", rp.Allowed)
	}
	if n := len(r.sim.Store.Log()); n != 1 { // only the open line: nothing refused reached the store
		t.Errorf("refused requests wrote to the log: %d lines", n)
	}

	// A valid answer, then the same thought again: double answer.
	r.must("/api/answer", map[string]any{"thought": id, "answer": "drink"})
	if code, rp := r.post("/api/answer", map[string]any{"thought": id, "answer": "drink"}); code != 409 || rp.Code != "already-answered" {
		t.Errorf("double answer: %d %+v", code, rp)
	}
	if rec, _ := r.sim.Store.Record(th.Key); rec.Streak != 1 {
		t.Errorf("double answer counted twice: %+v", rec)
	}

	// A faded thought (recorded by Tick) answers 410.
	nt := r.nextThought()
	r.sim.Clock.Advance(time.Duration(nt.RemainingSec+1) * time.Second)
	r.sim.Hub.Tick()
	if code, rp := r.post("/api/answer", map[string]any{"thought": nt.ID, "answer": "drink"}); code != 410 || rp.Code != "faded" {
		t.Errorf("faded answer: %d %+v", code, rp)
	}
}

func TestLateAnswerIsATooLateLineAndNotCounted(t *testing.T) {
	r := newRig(t, "sorceress")
	r.fight()
	th := r.nextThought()
	r.sim.Clock.Advance(time.Duration(th.RemainingSec+1) * time.Second) // faded, nobody ticked yet
	code, rp := r.post("/api/answer", map[string]any{"thought": th.ID, "answer": "drink"})
	if code != 410 || rp.Code != "faded" {
		t.Fatalf("late answer: %d %+v", code, rp)
	}
	if rec, ok := r.sim.Store.Record(th.Key); ok && (rec.Streak != 0 || rec.Last != "") {
		t.Errorf("late answer counted: %+v", rec)
	}
	if len(r.journal("?kind=late").Lines) != 1 || len(r.journal("?kind=missed").Lines) != 1 {
		t.Errorf("late and missed lines expected: %+v", r.journal("").Lines)
	}
}

func TestAnswerRouteGuards(t *testing.T) {
	r := newRig(t, "sorceress")
	r.fight()
	th := r.nextThought()
	good := fmt.Sprintf(`{"thought":%d,"answer":"drink"}`, th.ID)
	h := r.sim.Hub.AnswerHandler(AnswerOptions{Token: "s3cret", Origins: []string{origin}})
	for _, c := range []struct {
		name string
		hdr  map[string]string
		body string
		code int
	}{
		{"no token", map[string]string{}, good, 401},
		{"wrong token", map[string]string{"Authorization": "Bearer nope"}, good, 401},
		{"foreign origin", map[string]string{"Authorization": "Bearer s3cret", "Origin": "http://evil.example"}, good, 403},
		{"wrong content type", map[string]string{"Authorization": "Bearer s3cret", "Content-Type": "text/plain"}, good, 415},
		{"unknown field", map[string]string{"Authorization": "Bearer s3cret"}, `{"thought":1,"answer":"drink","x":1}`, 400},
		{"two documents", map[string]string{"Authorization": "Bearer s3cret"}, good + good, 400},
		{"too large", map[string]string{"Authorization": "Bearer s3cret"}, `{"answer":"` + strings.Repeat("a", 5000) + `"}`, 400},
		{"not json", map[string]string{"Authorization": "Bearer s3cret"}, `answer=drink`, 400},
	} {
		w := r.postRaw(h, "/api/answer", c.body, c.hdr)
		if w.Code != c.code {
			t.Errorf("%s: %d %s", c.name, w.Code, w.Body.String())
		}
	}
	if len(r.thoughts().Thoughts) != 1 {
		t.Fatal("a refused request answered the thought")
	}
	for _, m := range []string{"GET", "PUT", "DELETE", "PATCH"} {
		w := httptest.NewRecorder()
		h.ServeHTTP(w, httptest.NewRequest(m, "/api/answer", nil))
		if w.Code != 405 {
			t.Errorf("%s on the answer route: %d", m, w.Code)
		}
	}
	// Preflight: allowed origin gets the CORS headers, a foreign one does not.
	pre := httptest.NewRequest("OPTIONS", "/api/answer", nil)
	pre.Header.Set("Origin", origin)
	w := httptest.NewRecorder()
	h.ServeHTTP(w, pre)
	if w.Code != 204 || w.Header().Get("Access-Control-Allow-Origin") != origin || w.Header().Get("Access-Control-Allow-Credentials") != "true" {
		t.Errorf("preflight: %d %v", w.Code, w.Header())
	}
	pre.Header.Set("Origin", "http://evil.example")
	w = httptest.NewRecorder()
	h.ServeHTTP(w, pre)
	if w.Code != 403 || w.Header().Get("Access-Control-Allow-Origin") != "" {
		t.Errorf("foreign preflight: %d %v", w.Code, w.Header())
	}
	// With the right token the answer goes through (cookie and bearer both work).
	w = r.postRaw(h, "/api/answer", good, map[string]string{"Authorization": "Bearer s3cret"})
	if w.Code != 200 {
		t.Errorf("good request: %d %s", w.Code, w.Body.String())
	}
	w = r.postRaw(h, "/api/answer", fmt.Sprintf(`{"thought":%d,"answer":"drink"}`, th.ID), map[string]string{"Cookie": "d2dview=s3cret"})
	if w.Code != 409 {
		t.Errorf("cookie auth reaches the store (double answer expected): %d %s", w.Code, w.Body.String())
	}
}

// answerUntilRule answers the thought of the context n times (a new thought opens 30 s after each), returning the
// number of answers it took.
func answerUntilRule(r *rig, answer string) int {
	r.t.Helper()
	for n := 1; n <= 12; n++ {
		th := r.nextThought()
		if th.Kind == "confirm" {
			r.t.Fatalf("confirm too early (answer %d)", n)
		}
		r.must("/api/answer", map[string]any{"thought": th.ID, "answer": answer})
		if v := r.thoughts(); len(v.Thoughts) > 0 && v.Thoughts[0].Kind == "confirm" {
			return n
		}
		if r.rules().Live > 0 {
			return n
		}
	}
	r.t.Fatal("no rule after 12 answers")
	return 0
}

func TestRuleLearnedAfterNAnswers(t *testing.T) {
	cat, _ := decide.Shipped()
	for _, b := range builds {
		t.Run(b, func(t *testing.T) {
			r := newRig(t, b)
			r.fight()
			n := answerUntilRule(r, "portal")
			th := r.thoughts()
			var live int
			if r.sim.Build.Hardcore { // risky: N=5, then the hero asks "from now on, OK?"
				if n != cat.Config.NByRisk["risky"] {
					t.Fatalf("hardcore learned after %d answers, want %d", n, cat.Config.NByRisk["risky"])
				}
				c := th.Thoughts[0]
				if c.Kind != "confirm" || !strings.Contains(c.Learn, "From now on I will always") || len(c.Choices) != 2 {
					t.Fatalf("confirm thought: %+v", c)
				}
				if rp := r.rules(); rp.Live != 0 {
					t.Fatal("rule live before the confirm")
				}
				// "No" leaves it unlearned; the streak restarts.
				r.must("/api/confirm", map[string]any{"thought": c.ID, "yes": false})
				if r.rules().Live != 0 {
					t.Fatal("a no taught a rule")
				}
				if n2 := answerUntilRule(r, "portal"); n2 != 5 {
					t.Fatalf("after a no the streak restarts: %d answers", n2)
				}
				c = r.thoughts().Thoughts[0]
				r.must("/api/confirm", map[string]any{"thought": c.ID, "yes": true})
			} else if n != cat.Config.NByRisk["normal"] {
				t.Fatalf("softcore learned after %d answers, want %d", n, cat.Config.NByRisk["normal"])
			}
			rp := r.rules()
			live = rp.Live
			if live != 1 || len(rp.Rules) != 1 {
				t.Fatalf("rules: %+v", rp)
			}
			ru := rp.Rules[0]
			if ru.Answer != "portal" || ru.Label == "" || !strings.Contains(ru.Context, "life 26 to 49%") || !strings.HasPrefix(ru.Text, "When asked ") {
				t.Errorf("rule in plain language: %+v", ru)
			}
			// The hero now acts alone: no new thought opens for that context.
			r.sim.Step(2 * time.Minute)
			if v := r.thoughts(); len(v.Thoughts) != 0 {
				t.Errorf("a learned rule still asks: %+v", v.Thoughts)
			}
			if len(r.journal("?kind=learned").Lines) != 1 {
				t.Errorf("learned line missing")
			}
			// "why did you do that?"
			var w WhyView
			if c := r.get("/api/why?key="+urlEscape(ru.Key), &w); c.Code != 200 || w.Source != "rule" || !w.Revoke || !strings.Contains(w.Text, "rule you taught me") {
				t.Errorf("why: %d %+v", c.Code, w)
			}
		})
	}
}

func urlEscape(s string) string {
	return strings.NewReplacer("|", "%7C", "=", "%3D", ",", "%2C").Replace(s)
}

func TestRevokeOneAndAll(t *testing.T) {
	r := newRig(t, "sorceress")
	r.fight()
	answerUntilRule(r, "drink")
	rule := r.rules().Rules[0]
	r.sim.Step(time.Minute)
	if code, _ := r.post("/api/revoke", map[string]any{"key": rule.Key}); code != 200 {
		t.Fatal("revoke failed")
	}
	rp := r.rules()
	if rp.Live != 0 || !rp.Rules[0].Revoked || rp.Rules[0].RevokedAt <= 0 {
		t.Fatalf("after revoke: %+v", rp)
	}
	if code, ans := r.post("/api/revoke", map[string]any{"key": rule.Key}); code != 404 {
		t.Errorf("second revoke: %d %+v", code, ans)
	}
	if n := len(r.journal("?kind=revoked").Lines); n != 1 {
		t.Errorf("revoked lines: %d", n)
	}
	// The hero asks again and the streak restarts at 0.
	th := r.nextThought()
	if th.Key != rule.Key {
		t.Fatalf("asks about another context: %s", th.Key)
	}
	if rec, _ := r.sim.Store.Record(rule.Key); rec.Streak != 0 {
		t.Errorf("streak after revoke: %+v", rec)
	}
	var w WhyView
	r.get("/api/why?key="+urlEscape(rule.Key), &w)
	if w.Source != "revoked" || w.Revoke || !strings.Contains(w.Text, "You revoked the rule") {
		t.Errorf("why after revoke: %+v", w)
	}
	// Learn again, then revoke all (needs the confirm flag).
	answerUntilRule(r, "drink")
	if code, rp := r.post("/api/revoke-all", map[string]any{"confirm": true}); code != 200 || rp.Revoked != 1 {
		t.Fatalf("revoke-all: %d %+v", code, rp)
	}
	if v := r.rules(); v.Live != 0 {
		t.Errorf("live after revoke-all: %d", v.Live)
	}
	if len(r.journal("?kind=revoked-all").Lines) != 1 {
		t.Error("revoke-all line missing")
	}
}

func TestDeathNamesTheRulesAndOffersRevoke(t *testing.T) {
	r := newRig(t, "sorceress")
	r.fight()
	answerUntilRule(r, "drink")
	r.sim.Step(10 * time.Second)
	r.sim.Kill()
	v := r.thoughts()
	var inc *FeedItem
	for i := range v.Feed {
		if v.Feed[i].Kind == "incident" {
			inc = &v.Feed[i]
		}
	}
	if inc == nil || len(inc.Rules) != 1 || !inc.Rules[0].Live || inc.Rules[0].Label == "" || !strings.Contains(inc.Text, "died") {
		t.Fatalf("incident: %+v", v.Feed)
	}
	// The rule stays until the player decides; the revoke is offered, not applied.
	if r.rules().Live != 1 {
		t.Fatal("a death revoked a rule")
	}
	r.must("/api/revoke", map[string]any{"key": inc.Rules[0].Key})
	if r.rules().Live != 0 {
		t.Fatal("revoke from the incident failed")
	}
	if got := r.journal("?kind=incident").Lines; len(got) != 1 || len(got[0].Rules) != 1 || got[0].Rules[0].Live {
		t.Errorf("journal incident after revoke: %+v", got)
	}
}

func TestUptimeLowersOnDeathRecoveryAndStall(t *testing.T) {
	for _, b := range builds {
		t.Run(b, func(t *testing.T) {
			r := newRig(t, b)
			if u := r.uptime(); u.UptimePct != nil || u.State != KindIdle {
				t.Fatalf("before any order: %+v", u)
			}
			r.must("/api/orders", map[string]any{"kind": "go", "area": "Blood Moor", "minutes": 30})
			r.sim.Step(100 * time.Second)
			u := r.uptime()
			if u.UptimePct == nil || *u.UptimePct != 100 || u.State != KindProgress || !strings.HasPrefix(u.Label, "fighting in Blood Moor") {
				t.Fatalf("working: %+v", u)
			}
			r.sim.Kill() // 20 s dead, 20 s recovery
			r.sim.Step(40 * time.Second)
			r.sim.Stall(30 * time.Second)
			r.sim.Step(30 * time.Second)
			u = r.uptime()
			if u.Deaths != 1 || u.DeathSec != 20 || u.RecoverySec != 20 || u.StallSec != 30 {
				t.Fatalf("lost time: %+v", u)
			}
			// 100 s of progress before; progress resumes after the stall.
			want := 100 / (100 + 20.0 + 20 + 30)
			if got := *u.UptimePct / 100; got < want-0.01 || got > want+0.01 {
				t.Errorf("uptime %.3f, want about %.3f (%+v)", got, want, u)
			}
			if u.WindowPct == nil || u.Schema != "watch.uptime/1" {
				t.Errorf("window/schema: %+v", u)
			}
			// Idle time with no order is not counted at all.
			before := *r.uptime().UptimePct
			r.must("/api/orders/cancel", map[string]any{"id": 1})
			r.sim.Step(10 * time.Minute)
			after := r.uptime()
			if after.State != KindIdle || *after.UptimePct != before {
				t.Errorf("idle counted: before %.1f after %+v", before, after)
			}
		})
	}
}

func TestOrdersLifecycle(t *testing.T) {
	r := newRig(t, "sorceress")
	rp := r.must("/api/orders", map[string]any{"kind": "go", "area": "blood moor", "minutes": 1})
	if rp.Order == nil || rp.Order.Area != "Blood Moor" || rp.Order.Text != "Farm Blood Moor for 1 min" || rp.Order.ID != 1 {
		t.Fatalf("order: %+v", rp.Order)
	}
	r.must("/api/orders", map[string]any{"kind": "home"})
	var ov OrdersView
	r.get("/api/orders", &ov)
	if len(ov.Orders) != 2 || ov.Orders[0].State != OrderActive || ov.Orders[1].State != OrderQueued || len(ov.Areas) != len(OrderableAreas()) || ov.Kinds[0] != OrderGo {
		t.Fatalf("orders box: %+v", ov)
	}
	r.sim.Step(70 * time.Second) // the minute ends, then "come home" runs for 10 s
	r.sim.Step(15 * time.Second)
	r.get("/api/orders", &ov)
	if ov.Orders[0].State != OrderDone || ov.Orders[1].State != OrderDone {
		t.Fatalf("after the duration: %+v", ov.Orders)
	}
	if code, _ := r.post("/api/orders/cancel", map[string]any{"id": 1}); code != 409 {
		t.Errorf("cancel of a finished order: %d", code)
	}
	for i := 0; i < MaxOpenOrders; i++ {
		r.must("/api/orders", map[string]any{"kind": "fight", "minutes": 5})
	}
	if code, rp := r.post("/api/orders", map[string]any{"kind": "fight", "minutes": 5}); code != 409 || rp.Code != "order-state" {
		t.Errorf("sixth open order: %d %+v", code, rp)
	}
}

func TestOrderToGoal(t *testing.T) {
	g, err := OrderToGoal(Order{Kind: OrderGo, Area: "Blood Moor", Minutes: 10}, "")
	if err != nil || g.Type != "farm_area" || g.Params.Area != "Blood Moor" || g.Params.UntilMins != 10 || g.Title != "Farm Blood Moor until 10 min" {
		t.Errorf("go: %+v %v", g, err)
	}
	g, err = OrderToGoal(Order{Kind: OrderHere, Minutes: 5}, "Cold Plains")
	if err != nil || g.Params.Area != "Cold Plains" || g.Params.UntilMins != 5 {
		t.Errorf("fight here: %+v %v", g, err)
	}
	if _, err = OrderToGoal(Order{Kind: OrderHere, Minutes: 5}, ""); err == nil {
		t.Error("fight here without a known area")
	}
	if g, err = OrderToGoal(Order{Kind: OrderHome}, ""); err != nil || g.Type != "idle" {
		t.Errorf("home: %+v %v", g, err)
	}
	if _, err = OrderToGoal(Order{Kind: "x"}, ""); err == nil {
		t.Error("unknown kind")
	}
}

func TestViewerIsReadOnly(t *testing.T) {
	r := newRig(t, "sorceress")
	r.fight()
	th := r.nextThought()
	log := len(r.sim.Store.Log())
	for _, path := range append(append([]string{}, ViewRoutes...), "/watch/app.js", "/api/answer", "/api/revoke", "/api/orders") {
		for _, m := range []string{"POST", "PUT", "DELETE", "PATCH"} {
			w := httptest.NewRecorder()
			body, _ := json.Marshal(map[string]any{"thought": th.ID, "answer": "drink"})
			req := httptest.NewRequest(m, path, bytes.NewReader(body))
			req.Header.Set("Content-Type", "application/json")
			r.view.ServeHTTP(w, req)
			if w.Code != http.StatusMethodNotAllowed || w.Header().Get("Allow") != "GET, HEAD" {
				t.Errorf("%s %s on the viewer: %d", m, path, w.Code)
			}
		}
	}
	// Every GET leaves the store and the Hub unchanged.
	seq := r.sim.Hub.Seq()
	for _, path := range ViewRoutes {
		if path == "/api/events" {
			continue
		}
		r.get(path, nil)
	}
	r.get("/api/why?key="+urlEscape(th.Key), nil)
	if len(r.sim.Store.Log()) != log || r.sim.Hub.Seq() != seq {
		t.Error("a GET changed state")
	}
	if len(r.thoughts().Thoughts) != 1 {
		t.Error("the POSTs on the viewer answered the thought")
	}
}

func TestSinkHoldsThoughtsDuringAFight(t *testing.T) {
	r := newRig(t, "sorceress")
	r.must("/api/orders", map[string]any{"kind": "fight", "minutes": 200})
	r.sim.SetFight(true)
	r.sim.Step(40 * time.Second)
	if n := len(r.sim.Store.OpenThoughts()); n != 1 {
		t.Fatalf("store thoughts: %d", n)
	}
	if v := r.thoughts(); len(v.Thoughts) != 0 {
		t.Fatalf("a held thought is on the screen: %+v", v.Thoughts)
	}
	r.sim.SetFight(false) // the ladder shows it now (Sink "thought")
	v := r.thoughts()
	if len(v.Thoughts) != 1 || !strings.Contains(v.Thoughts[0].Meanwhile, "Meanwhile I:") {
		t.Fatalf("after the fight: %+v", v.Thoughts)
	}
}

func TestXSSIsInertText(t *testing.T) {
	r := newRig(t, "sorceress")
	evil := `<script>alert(1)</script><img src=x onerror=alert(2)>` + "\n\x00"
	r.sim.Hub.Say(evil)
	r.sim.Hub.Incident(evil)
	r.sim.Hub.Activity(KindProgress, evil)
	for _, p := range []string{"/api/thoughts", "/api/journal", "/api/uptime"} {
		w := r.get(p, nil)
		body := w.Body.String()
		if strings.Contains(body, "<script") || strings.Contains(body, "<img") {
			t.Errorf("%s carries raw markup: %s", p, body)
		}
		if !strings.Contains(body, "alert(1)") && p != "/api/uptime" {
			t.Errorf("%s lost the text: %s", p, body)
		}
		if ct := w.Header().Get("Content-Type"); !strings.HasPrefix(ct, "application/json") || w.Header().Get("X-Content-Type-Options") != "nosniff" {
			t.Errorf("%s headers: %v", p, w.Header())
		}
	}
	var v ThoughtsView
	r.get("/api/thoughts", &v)
	for _, f := range v.Feed {
		if strings.ContainsAny(f.Text, "\x00\n") {
			t.Errorf("control characters kept: %q", f.Text)
		}
	}
	// A bad answer id echoes as JSON text only, and the error is short.
	code, rp := r.post("/api/answer", map[string]any{"thought": 1, "answer": strings.Repeat("<b>", 100)})
	if code != 404 || strings.Contains(rp.Error, "<b><b><b><b><b><b><b><b><b><b><b><b><b><b><b>") {
		t.Errorf("echo: %d %q", code, rp.Error)
	}
	// The page: no inline script, no HTML sinks, no CDN, CSP set.
	page := r.get("/watch", nil)
	app := r.get("/watch/app.js", nil).Body.String()
	html := page.Body.String()
	if strings.Contains(html, "<script>") || strings.Contains(html, "onclick") || strings.Contains(html, "http://") || strings.Contains(html, "https://") {
		t.Error("page has inline script or an external reference")
	}
	for _, bad := range []string{"innerHTML", "outerHTML", "insertAdjacentHTML", "document.write", "eval(", "new Function", "srcdoc"} {
		if strings.Contains(app, bad) {
			t.Errorf("app.js uses %s", bad)
		}
	}
	if csp := page.Header().Get("Content-Security-Policy"); !strings.Contains(csp, "script-src 'self'") || strings.Contains(csp, "unsafe-inline") {
		t.Errorf("CSP: %q", csp)
	}
	if strings.Contains(r.get("/watch/app.css", nil).Body.String(), "@import") {
		t.Error("css imports")
	}
}

func TestJournalKeepsEverythingInOrder(t *testing.T) {
	r := newRig(t, "sorceress")
	r.fight()
	t1 := r.nextThought()
	r.must("/api/answer", map[string]any{"thought": t1.ID, "answer": decide.NotSure})
	t2 := r.nextThought()
	r.sim.Clock.Advance(time.Minute)
	r.sim.Hub.Tick()
	_ = t2
	j := r.journal("")
	var kinds []string
	for _, l := range j.Lines {
		kinds = append(kinds, l.Kind)
	}
	got := strings.Join(kinds, ",")
	if !strings.HasPrefix(got, "order,order,question,notsure,question,missed") {
		t.Errorf("journal order: %s", got)
	}
	for i := 1; i < len(j.Lines); i++ {
		if j.Lines[i].T < j.Lines[i-1].T {
			t.Errorf("journal not in time order at %d", i)
		}
	}
	if lim := r.journal("?limit=2"); len(lim.Lines) != 2 || lim.Total != j.Total {
		t.Errorf("limit: %d of %d", len(lim.Lines), lim.Total)
	}
	// "why did you do that?" from a journal line.
	var w WhyView
	for _, l := range j.Lines {
		if l.Kind == "notsure" {
			if c := r.get("/api/why?line="+l.ID, &w); c.Code != 200 || w.Source != "default" || w.Streak != 0 {
				t.Errorf("why line: %d %+v", c.Code, w)
			}
		}
	}
	if c := r.get("/api/why?line=nope", nil); c.Code != 404 {
		t.Errorf("why unknown line: %d", c.Code)
	}
}

func TestWatchAssetsHaveNoThirdPartyURL(t *testing.T) {
	r := newRig(t, "sorceress")
	re := regexp.MustCompile(`(?i)(src|href)\s*=\s*["']?(https?:)?//|url\(\s*["']?(https?:)?//|@import`)
	for _, p := range []string{"/watch", "/watch/app.css", "/watch/app.js"} {
		if m := re.FindString(r.get(p, nil).Body.String()); m != "" {
			t.Errorf("%s has an external reference: %q", p, m)
		}
	}
}
