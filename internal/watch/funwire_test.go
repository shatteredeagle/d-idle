package watch

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"testing"
	"time"

	"d2lod-bot/internal/currency"
	"d2lod-bot/internal/decide"
	"d2lod-bot/internal/goals"
	"d2lod-bot/internal/ladder"
)

const funToken = "fun-test-token-0123456789"

type funRig struct {
	t      *testing.T
	fun    *Fun
	hub    *Hub
	strip  *TallyStrip
	clock  *SimClock
	wall   time.Time
	view   string
	answer string
	client *http.Client
	ledger *currency.Ledger
}

// newFunRig starts the real watch server on two loopback ports with the whole fun loop wired the way cmd/d2bot does:
// a store on the firsts catalogue, a tally on a d.idle ledger, a Codex tracker built once the class is known.
func newFunRig(t *testing.T, class string, level int) *funRig {
	t.Helper()
	r := &funRig{t: t, wall: time.Unix(1_800_000_000, 0), clock: &SimClock{}}
	cat, err := decide.Shipped()
	if err != nil {
		t.Fatal(err)
	}
	tbl, err := decide.ShippedFirsts()
	if err != nil {
		t.Fatal(err)
	}
	if cat, err = cat.WithFirsts(tbl); err != nil {
		t.Fatal(err)
	}
	st, err := decide.New(cat, "fun-"+strings.ToLower(class), r.clock)
	if err != nil {
		t.Fatal(err)
	}
	if r.hub, err = New(Config{Store: st, Cat: cat, Clock: r.clock, Areas: OrderableAreas(), WaitForSink: true}); err != nil {
		t.Fatal(err)
	}
	now := time.Date(2026, 11, 7, 12, 0, 0, 0, time.UTC)
	r.ledger, err = currency.Open(t.TempDir(), currency.Options{Mode: testMode, Instance: "i4", Now: func() time.Time { return now }})
	if err != nil {
		t.Fatal(err)
	}
	wallet := currency.NewWallet(testMode, r.ledger)
	r.hub.SetCurrency(wallet)
	tally, err := currency.NewTally(r.ledger, currency.DefaultTallyRows(), "Hero", "r1")
	if err != nil {
		t.Fatal(err)
	}
	if r.strip, err = NewTallyStrip(r.hub, wallet, tally, func(a string) bool { return a == "Rogue Encampment" }); err != nil {
		t.Fatal(err)
	}
	r.strip.Now = func() time.Time { return now }
	codex := decide.NewCodex("fun-" + strings.ToLower(class))
	r.fun, err = NewFun(FunConfig{Hub: r.hub, Visit: NewVisit("", func() time.Time { return r.wall }), Speed: 1, Strip: r.strip,
		Codex: codex, Table: tbl, Clock: r.clock,
		Tracker: func(c string) (*decide.Tracker, error) {
			return decide.NewTracker(tbl, cat, st, codex, r.clock, decide.Who{Class: c, Diff: "normal"})
		}})
	if err != nil {
		t.Fatal(err)
	}
	r.hub.SetHero(HeroInfo{Class: class, Level: level, Area: "Rogue Encampment"})
	ctx, cancel := context.WithCancel(context.Background())
	t.Cleanup(cancel)
	served, err := r.hub.Serve(ctx, ServeOptions{ViewAddr: "127.0.0.1:0", AnswerAddr: "127.0.0.1:0", Token: funToken, Fun: r.fun})
	if err != nil {
		t.Fatal(err)
	}
	r.view, r.answer = "http://"+served.View.String(), "http://"+served.Answer.String()
	r.client = &http.Client{CheckRedirect: func(*http.Request, []*http.Request) error { return http.ErrUseLastResponse }, Timeout: 10 * time.Second}
	return r
}

// stream is one Act 1 expedition as the bot emits it: out of camp, 26 Fallen, Corpsefire, the Den quest, a unique, a
// level, back to camp. Sequence numbers come from the caller (the manager's), so a replay repeats them.
func funStream() []goals.Event {
	evs := []goals.Event{{Type: goals.EvAreaChange, Area: "Blood Moor", AreaID: 2}}
	for i := 0; i < 26; i++ {
		evs = append(evs, goals.Event{Type: goals.EvKill, Monster: "Fallen", XP: 10})
	}
	evs = append(evs,
		goals.Event{Type: goals.EvKill, Monster: "Corpsefire", Boss: true, XP: 100},
		goals.Event{Type: goals.EvQuest, Quest: "den_of_evil", Done: true},
		goals.Event{Type: goals.EvDrop, Item: &goals.Item{Name: "Tyrael's Might", Quality: "unique"}},
		goals.Event{Type: goals.EvLevelUp, Level: 6},
		goals.Event{Type: goals.EvAreaChange, Area: "Rogue Encampment", AreaID: 1})
	for i := range evs {
		evs[i].Seq = uint64(i + 1)
	}
	return evs
}

func (r *funRig) feed(evs ...goals.Event) {
	for _, ev := range evs {
		r.fun.OnGoalEvent(ev)
	}
	r.fun.Settle()
}

func (r *funRig) do(method, base, path, token, body string) (*http.Response, []byte) {
	r.t.Helper()
	req, err := http.NewRequest(method, base+path, strings.NewReader(body))
	if err != nil {
		r.t.Fatal(err)
	}
	if token != "" {
		req.Header.Set("Authorization", "Bearer "+token)
	}
	if body != "" {
		req.Header.Set("Content-Type", "application/json")
	}
	resp, err := r.client.Do(req)
	if err != nil {
		r.t.Fatal(err)
	}
	defer resp.Body.Close()
	b, _ := io.ReadAll(resp.Body)
	return resp, b
}

func (r *funRig) getJSON(path string, v any) {
	r.t.Helper()
	resp, b := r.do("GET", r.view, path, funToken, "")
	if resp.StatusCode != 200 {
		r.t.Fatalf("GET %s: %d %s", path, resp.StatusCode, b)
	}
	if err := json.Unmarshal(b, v); err != nil {
		r.t.Fatalf("GET %s: %v\n%s", path, err, b)
	}
}

func TestFunWiringEndToEnd(t *testing.T) {
	for _, class := range []string{"Sorceress", "Paladin"} {
		t.Run(class, func(t *testing.T) {
			r := newFunRig(t, class, 5)

			// a look at the live screen, then 21 minutes away
			if resp, _ := r.do("GET", r.view, "/watch?live=1", funToken, ""); resp.StatusCode != 200 {
				t.Fatalf("live screen: %d", resp.StatusCode)
			}
			r.feed(funStream()...)
			r.fun.OnWaypoint("Cold Plains", "r1:wp:3")
			r.wall = r.wall.Add(21 * time.Minute)
			resp, _ := r.do("GET", r.view, "/watch", funToken, "")
			if resp.StatusCode != http.StatusFound || resp.Header.Get("Location") != "/watch/return" {
				t.Fatalf("/watch after 21 min: %d %v", resp.StatusCode, resp.Header)
			}
			if resp, _ := r.do("GET", r.view, "/watch/return", funToken, ""); resp.StatusCode != 200 {
				t.Fatalf("return page: %d", resp.StatusCode)
			}

			var d ReturnData
			r.getJSON("/api/return", &d)
			if len(d.Trips) != 1 || d.Trips[0].Kills != 27 || !strings.Contains(d.Headline, class) {
				t.Errorf("expedition card: %+v / %q", d.Trips, d.Headline)
			}
			secs := map[string]SectionView{}
			for _, s := range d.Sections {
				secs[s.ID] = s
			}
			tallyText := strings.Join(secs["tally"].Lines, "\n")
			if !strings.Contains(tallyText, "Tally 53") || !strings.Contains(tallyText, "Last trip: +53") {
				t.Errorf("tally section: %q", tallyText)
			}
			codexText := strings.Join(secs[CodexSectionID].Lines, "\n")
			if !strings.Contains(codexText, "Fallen") || !strings.Contains(codexText, "Act 1 entries") {
				t.Errorf("codex section: %q", codexText)
			}
			if _, ok := secs["proposals"]; !ok {
				t.Error("the proposals placeholder must stay for card 131")
			}

			// the strip equals the credited events: 1 batch of 25 kills + Fallen 5 + Corpsefire 5 + boss 15 + quest 10 +
			// Blood Moor 5 + trip 2 + Cold Plains waypoint 10
			var tv TallyView
			r.getJSON("/api/tally", &tv)
			if tv.Balance != 53 || !tv.Enabled || tv.Error != "" {
				t.Fatalf("tally: %+v", tv)
			}
			var cv CodexView
			r.getJSON("/api/codex", &cv)
			if cv.Filled < 5 || !strings.Contains(cv.Line, "Corpsefire") || !strings.Contains(cv.Line, "Cold Plains waypoint") {
				t.Errorf("codex: filled %d, %q", cv.Filled, cv.Line)
			}

			// a replay (same sequence numbers) credits nothing and adds no entry
			r.feed(funStream()...)
			r.fun.OnWaypoint("Cold Plains", "r1:wp:3")
			r.getJSON("/api/tally", &tv)
			filled := cv.Filled
			r.getJSON("/api/codex", &cv)
			if tv.Balance != 53 || cv.Filled != filled {
				t.Errorf("replay: balance %d, codex %d -> %d", tv.Balance, filled, cv.Filled)
			}
			// ... also into a fresh tally on the same ledger (a restart that replays the journal)
			again, err := currency.NewTally(r.ledger, currency.DefaultTallyRows(), "Hero", "r1")
			if err != nil {
				t.Fatal(err)
			}
			s2, _ := NewTallyStrip(r.hub, r.strip.Wallet, again, func(a string) bool { return a == "Rogue Encampment" })
			feed(s2, funStream()...)
			if b, _ := r.ledger.Balance("tally"); b.Spendable != 53 {
				t.Errorf("replay into a new tally: %d", b.Spendable)
			}

			// orders: in range is free, above the range needs the charter and an approval
			post := func(token, body string) (int, map[string]any) {
				resp, b := r.do("POST", r.answer, "/api/orders", token, body)
				var m map[string]any
				_ = json.Unmarshal(b, &m)
				return resp.StatusCode, m
			}
			if code, m := post(funToken, `{"kind":"go","area":"Blood Moor","minutes":20,"replace":true}`); code != 200 {
				t.Fatalf("in-range order: %d %v", code, m)
			}
			// Cold Plains is for clvl 6: one level above the hero (clvl 5), charter 20
			code, m := post(funToken, `{"kind":"go","area":"Cold Plains","minutes":30,"replace":true}`)
			ch, _ := m["charter"].(map[string]any)
			if code != 402 || m["code"] != "charter-needed" || ch == nil || ch["price"].(float64) != 20 {
				t.Fatalf("unapproved above-range order: %d %v", code, m)
			}
			if bal, _ := r.ledger.Balance("tally"); bal.Spendable != 53 {
				t.Errorf("a refused charter cost %d", 53-bal.Spendable)
			}
			if code, m := post(funToken, `{"kind":"go","area":"Cold Plains","minutes":30,"replace":true,"approve":true}`); code != 200 {
				t.Fatalf("approved order: %d %v", code, m)
			}
			r.getJSON("/api/tally", &tv)
			if tv.Balance != 33 {
				t.Errorf("after the charter: %d, want 33", tv.Balance)
			}

			// token gates and read-only sides
			for _, p := range []string{"/api/return", "/api/tally", "/api/codex", "/watch/return"} {
				if resp, _ := r.do("GET", r.view, p, "", ""); resp.StatusCode != http.StatusUnauthorized {
					t.Errorf("GET %s without a token: %d", p, resp.StatusCode)
				}
			}
			for _, p := range []string{"/api/return", "/api/tally", "/api/codex", "/api/orders", "/api/return/answer-kind"} {
				if resp, _ := r.do("POST", r.view, p, funToken, "{}"); resp.StatusCode != http.StatusMethodNotAllowed && resp.StatusCode != http.StatusNotFound {
					t.Errorf("POST %s on the viewer: %d", p, resp.StatusCode)
				}
			}
			if code, _ := post("", `{"kind":"home"}`); code != http.StatusUnauthorized {
				t.Errorf("order without a token: %d", code)
			}
			if resp, _ := r.do("POST", r.answer, "/api/return/answer-kind", "", `{"point":"x","answer":"y"}`); resp.StatusCode != http.StatusUnauthorized {
				t.Errorf("return answer without a token: %d", resp.StatusCode)
			}
			if resp, _ := r.do("POST", r.answer, "/api/return/answer-kind", funToken, `{"point":"nope","answer":"y"}`); resp.StatusCode == http.StatusUnauthorized {
				t.Errorf("return answer with a token reached the handler: %d", resp.StatusCode)
			}
		})
	}
}

// A key that went quiet (TASK-137) shows on the return screen as "stopped asking".
func TestFunQuietNote(t *testing.T) {
	r := newFunRig(t, "Sorceress", 5)
	key := "t0.retreat.repeat|act=1,diff=normal,hc=normal"
	r.hub.Event("quiet", ladder.ThoughtView{Point: "t0.retreat.repeat", Key: key, Default: "no"}, "no answer after repeated questions")
	var d ReturnData
	r.wall = r.wall.Add(time.Minute)
	r.getJSON("/api/return?hours=1", &d)
	var got []string
	for _, s := range d.Sections {
		if s.ID == "quiet" {
			got = s.Lines
		}
	}
	if len(got) != 1 || !strings.Contains(got[0], "stopped asking") {
		t.Errorf("quiet section: %v", got)
	}
}

// Without a Codex, a tally or a class the parts stay off and say so; nothing fails.
func TestFunOptionalParts(t *testing.T) {
	s, err := NewSim("sorceress", 1)
	if err != nil {
		t.Fatal(err)
	}
	f, err := NewFun(FunConfig{Hub: s.Hub})
	if err != nil {
		t.Fatal(err)
	}
	f.OnGoalEvent(goals.Event{Type: goals.EvKill, Monster: "Fallen", Seq: 1})
	var buf bytes.Buffer
	for _, sec := range f.Return.Sections.secs {
		if v := sec.Build(&ReturnData{}); v != nil {
			fmt.Fprintln(&buf, v.ID)
		}
	}
	if !strings.Contains(buf.String(), "tally") || strings.Contains(buf.String(), "codex") {
		t.Errorf("sections: %q", buf.String())
	}
	if _, err := NewFun(FunConfig{}); err == nil {
		t.Error("a Fun without a Hub")
	}
	if _, err := NewFun(FunConfig{Hub: s.Hub, Codex: decide.NewCodex("x")}); err == nil {
		t.Error("a Codex without its table, clock and tracker factory")
	}
}
