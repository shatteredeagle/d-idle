package watch

import (
	"context"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"d2lod-bot/internal/decide"
	"d2lod-bot/internal/goals"
	"d2lod-bot/internal/unlock"
)

const alertsLadder = `{"schema":"unlock.ladder/1","entries":[
 {"id":"alerts","group":"ui","trigger":{"kind":"tally_spend"},"price":10,"reveal":{"title":"Alerts and pings","text":"Pings.","teaser":"Costs 10 tally."}}
]}`

func alertsRig(t *testing.T, class string, ac *AlertsConfig) (*funRig, *unlock.Engine) {
	t.Helper()
	l, err := unlock.Parse([]byte(alertsLadder))
	if err != nil {
		t.Fatal(err)
	}
	eng, err := unlock.Open(t.TempDir(), strings.ToLower(class), l, func() time.Time { return time.Unix(1_800_000_000, 0) })
	if err != nil {
		t.Fatal(err)
	}
	r := newFunRigWith(t, class, 5, func(_ *funRig, fc *FunConfig) { fc.Unlock = eng; fc.Alerts = ac })
	return r, eng
}

func alertsOf(r *funRig, since uint64) AlertsView {
	var v AlertsView
	path := AlertsRoute
	if since > 0 {
		path += "?since=" + string(rune('0'+since))
	}
	r.getJSON(path, &v)
	return v
}

func drop(seq uint64, name, q string) goals.Event {
	return goals.Event{Seq: seq, Type: goals.EvDrop, Item: &goals.Item{Name: name, Quality: q}, Area: "Blood Moor"}
}

func death(seq uint64) goals.Event {
	return goals.Event{Seq: seq, Type: goals.EvDeath, Monster: "Fallen Shaman", Area: "Blood Moor", Level: 5}
}

func TestAlertRulesShippedAndErrors(t *testing.T) {
	rt, err := ShippedAlertRules()
	if err != nil {
		t.Fatal(err)
	}
	free := map[string]bool{}
	for _, r := range rt.Rules {
		if r.Free {
			free[r.ID] = true
		}
	}
	if !free["death"] || !free["low_life"] || len(free) != 2 {
		t.Fatalf("free rules must be exactly death and low_life: %v", free)
	}
	for _, r := range rt.Rules {
		for c := range r.ClassText {
			if !containsStr(alertClasses, c) {
				t.Errorf("%s: class %s", r.ID, c)
			}
		}
	}
	base := func(rule string) string {
		return `{"schema":"alerts.rules/1","rules":[` + rule + `]}`
	}
	for name, tc := range map[string]struct{ rule, want string }{
		"unknown event":    {`{"id":"x","event":"sneeze","severity":"info","text":"a"}`, `unknown event "sneeze"`},
		"unknown severity": {`{"id":"x","event":"town","severity":"loud","text":"a"}`, `unknown severity "loud"`},
		"empty text":       {`{"id":"x","event":"town","severity":"info","text":" "}`, "empty text"},
		"bad placeholder":  {`{"id":"x","event":"town","severity":"info","text":"{item}"}`, "{item}"},
		"drop no quality":  {`{"id":"x","event":"drop","severity":"info","text":"a"}`, "needs qualities"},
		"kill all":         {`{"id":"x","event":"kill","severity":"info","text":"a"}`, "must set boss"},
		"neg cooldown":     {`{"id":"x","event":"town","severity":"info","text":"a","cooldownSecs":-1}`, "negative cooldown"},
		"stray filter":     {`{"id":"x","event":"town","severity":"info","text":"a","filters":{"minLevel":3}}`, "minLevel"},
		"unknown class":    {`{"id":"x","event":"town","severity":"info","text":"a","classText":{"wizard":"b"}}`, `unknown class "wizard"`},
		"markup":           {`{"id":"x","event":"town","severity":"info","text":"<b>"}`, "< or >"},
		"low life range":   {`{"id":"x","event":"low_life","severity":"info","text":"a","filters":{"belowPct":0}}`, "belowPct"},
	} {
		if _, err := ParseAlertRules([]byte(base(tc.rule))); err == nil || !strings.Contains(err.Error(), tc.want) || !strings.Contains(err.Error(), `"x"`) {
			t.Errorf("%s: %v, want %q naming the rule", name, err, tc.want)
		}
	}
	dup := `{"id":"x","event":"town","severity":"info","text":"a"}`
	if _, err := ParseAlertRules([]byte(base(dup + "," + dup))); err == nil || !strings.Contains(err.Error(), "duplicate") {
		t.Errorf("duplicate: %v", err)
	}
}

// openAlerts is Alerts over the shipped rules with a manual clock and no ladder, so only the free rules fire.
func newTestAlerts(t *testing.T, class string, now *float64, u *Unlocks) *Alerts {
	t.Helper()
	a, err := NewAlerts(u, AlertsConfig{}, func() float64 { return *now }, func() string { return class })
	if err != nil {
		t.Fatal(err)
	}
	return a
}

func TestAlertRulesFiltersAndCooldowns(t *testing.T) {
	l, _ := unlock.Parse([]byte(alertsLadder))
	eng, _ := unlock.Open("", "p", l, nil)
	if _, err := eng.Buy("alerts", okSpender{}); err != nil {
		t.Fatal(err)
	}
	now := 1000.0
	a := newTestAlerts(t, "paladin", &now, &Unlocks{Engine: eng})
	ids := func(es []Alert) string {
		var s []string
		for _, e := range es {
			s = append(s, e.Rule)
		}
		return strings.Join(s, ",")
	}
	type step struct {
		name string
		at   float64
		ev   AlertEvent
		want string
	}
	for _, s := range []step{
		{"unique", 1000, AlertEvent{Kind: "drop", Item: "Shako", Quality: "unique"}, "unique_drop"},
		{"another unique has no cooldown", 1001, AlertEvent{Kind: "drop", Item: "Skullder's", Quality: "set"}, "unique_drop"},
		{"magic is below the filter", 1002, AlertEvent{Kind: "drop", Item: "Cap", Quality: "magic"}, ""},
		{"rare", 1003, AlertEvent{Kind: "drop", Item: "Crown", Quality: "rare"}, "rare_drop"},
		{"rare cooldown", 1100, AlertEvent{Kind: "drop", Item: "Crown", Quality: "rare"}, ""},
		{"rare after cooldown", 1304, AlertEvent{Kind: "drop", Item: "Crown", Quality: "rare"}, "rare_drop"},
		{"rune", 2000, AlertEvent{Kind: "drop", Item: "Ort Rune", Quality: "rune"}, "rare_drop"},
		{"boss", 2000, AlertEvent{Kind: "kill", Monster: "Andariel", Boss: true}, "boss_kill"},
		{"plain kill", 2000, AlertEvent{Kind: "kill", Monster: "Zombie"}, ""},
		{"level 1 is below minLevel", 2000, AlertEvent{Kind: "level_up", Level: 1}, ""},
		{"level 7", 2000, AlertEvent{Kind: "level_up", Level: 7}, "level_up"},
		{"town", 3000, AlertEvent{Kind: "town", Area: "Rogue Encampment"}, "back_in_town"},
		{"town again too soon", 3100, AlertEvent{Kind: "town", Area: "Rogue Encampment"}, ""},
		{"town after 15 min", 3901, AlertEvent{Kind: "town", Area: "Rogue Encampment"}, "back_in_town"},
		{"life 50 is fine", 4000, AlertEvent{Kind: "low_life", Pct: 50}, ""},
		{"life 20", 4000, AlertEvent{Kind: "low_life", Pct: 20, Area: "Blood Moor"}, "low_life"},
		{"life 10 inside cooldown", 4060, AlertEvent{Kind: "low_life", Pct: 10}, ""},
		{"life 10 later", 4121, AlertEvent{Kind: "low_life", Pct: 10}, "low_life"},
		{"death has no cooldown", 5000, AlertEvent{Kind: "death", Monster: "Andariel", Area: "Catacombs"}, "death"},
		{"second death", 5000, AlertEvent{Kind: "death", Monster: "Andariel", Area: "Catacombs"}, "death"},
	} {
		now = s.at
		if got := ids(a.Emit(s.ev)); got != s.want {
			t.Errorf("%s: raised %q, want %q", s.name, got, s.want)
		}
	}
}

type okSpender struct{}

func (okSpender) Spend(string, int64, string) error { return nil }

func TestAlertsLockedKeepsOnlyFreeRules(t *testing.T) {
	l, _ := unlock.Parse([]byte(alertsLadder))
	eng, _ := unlock.Open("", "p", l, nil)
	now := 0.0
	a := newTestAlerts(t, "sorceress", &now, &Unlocks{Engine: eng})
	a.Emit(AlertEvent{Kind: "drop", Item: "Shako", Quality: "unique"})
	a.Emit(AlertEvent{Kind: "level_up", Level: 9})
	a.Emit(AlertEvent{Kind: "death", Monster: "Mephisto", Area: "Durance of Hate"})
	a.Emit(AlertEvent{Kind: "low_life", Pct: 5})
	v := a.View(0)
	if v.Open || len(v.Alerts) != 2 || v.Alerts[0].Rule != "death" || v.Alerts[1].Rule != "low_life" || v.Teaser == "" || v.Price != 10 {
		t.Fatalf("locked view: %+v", v)
	}
	// buying later does not replay what was skipped
	if _, err := eng.Buy("alerts", okSpender{}); err != nil {
		t.Fatal(err)
	}
	if v := a.View(0); !v.Open || len(v.Alerts) != 2 || v.Teaser != "" {
		t.Fatalf("after buying: %+v", v)
	}
	// without any ladder only the free rules fire
	a2 := newTestAlerts(t, "sorceress", &now, nil)
	a2.Emit(AlertEvent{Kind: "drop", Item: "Shako", Quality: "unique"})
	a2.Emit(AlertEvent{Kind: "death", Monster: "x", Area: "y"})
	if v := a2.View(0); len(v.Alerts) != 1 || v.Open {
		t.Fatalf("no ladder: %+v", v)
	}
}

func TestAlertsGoalEventsTwoClasses(t *testing.T) {
	for _, tc := range []struct{ class, deathText, levelText string }{
		{"Sorceress", "The Sorceress is spent. Fallen Shaman ended her in Blood Moor.", "Sparks climb the staff: level 6."},
		{"Paladin", "The Paladin is down. Fallen Shaman took her in Blood Moor; the Light waits.", "The Light burns brighter: level 6."},
	} {
		t.Run(tc.class, func(t *testing.T) {
			r, eng := alertsRig(t, tc.class, nil)
			r.feed(death(1))
			if v := alertsOf(r, 0); len(v.Alerts) != 1 || v.Alerts[0].Text != tc.deathText || !v.Alerts[0].Free || v.Alerts[0].Severity != "alarm" {
				t.Fatalf("locked: %+v", v)
			}
			r.feed(goals.Event{Seq: 2, Type: goals.EvLevelUp, Level: 6}, drop(3, "Harlequin Crest", "unique"))
			if v := alertsOf(r, 0); len(v.Alerts) != 1 {
				t.Fatalf("a locked row raised a gated alert: %+v", v)
			}
			if _, err := eng.Buy("alerts", okSpender{}); err != nil {
				t.Fatal(err)
			}
			r.feed(goals.Event{Seq: 4, Type: goals.EvLevelUp, Level: 6}, drop(5, "Harlequin Crest", "unique"),
				goals.Event{Seq: 6, Type: goals.EvAreaChange, Area: "Blood Moor", AreaID: 2},
				goals.Event{Seq: 7, Type: goals.EvAreaChange, Area: "Rogue Encampment", AreaID: 1},
				goals.Event{Seq: 8, Type: goals.EvKill, Monster: "Andariel", Boss: true})
			v := alertsOf(r, 0)
			var got []string
			for _, a := range v.Alerts {
				got = append(got, a.Rule+": "+a.Text)
			}
			want := []string{"death: " + tc.deathText, "level_up: " + tc.levelText, "unique_drop: Something glints in the dust: Harlequin Crest (unique).",
				"back_in_town: The hero is back in Rogue Encampment and ready to trade.", "boss_kill: Andariel has fallen. The hero stands over the remains."}
			if strings.Join(got, "\n") != strings.Join(want, "\n") {
				t.Fatalf("got\n%s\nwant\n%s", strings.Join(got, "\n"), strings.Join(want, "\n"))
			}
			b, _ := json.Marshal(v)
			golden(t, "alerts-"+strings.ToLower(tc.class), string(b)+"\n")
			// a replayed stream raises nothing; the first area seen is not a return
			r.feed(death(1), drop(5, "Harlequin Crest", "unique"))
			if v2 := alertsOf(r, 0); len(v2.Alerts) != len(v.Alerts) {
				t.Fatalf("replay raised alerts: %+v", v2)
			}
		})
	}
}

func TestAlertsTownNeedsAFieldFirst(t *testing.T) {
	now := 0.0
	a := newTestAlerts(t, "sorceress", &now, nil)
	l, _ := unlock.Parse([]byte(alertsLadder))
	eng, _ := unlock.Open("", "p", l, nil)
	eng.Buy("alerts", okSpender{})
	a.u = &Unlocks{Engine: eng}
	a.OnGoalEvent(goals.Event{Type: goals.EvAreaChange, Area: "Rogue Encampment", AreaID: 1}) // start-up spawn
	if len(a.View(0).Alerts) != 0 {
		t.Fatal("the first area counted as a return")
	}
	a.OnGoalEvent(goals.Event{Type: goals.EvAreaChange, Area: "Blood Moor", AreaID: 2})
	a.OnGoalEvent(goals.Event{Type: goals.EvAreaChange, Area: "Rogue Encampment", AreaID: 1})
	if len(a.View(0).Alerts) != 1 {
		t.Fatal("no return alert")
	}
}

func TestAlertsTickProposalAndPactPause(t *testing.T) {
	l, _ := unlock.Parse([]byte(alertsLadder))
	eng, _ := unlock.Open("", "p", l, nil)
	eng.Buy("alerts", okSpender{})
	now := 0.0
	a := newTestAlerts(t, "paladin", &now, &Unlocks{Engine: eng})
	pend := []decide.Proposal{{ID: 3, From: "Blood Moor", To: "Cold Plains"}}
	lines := []JournalLine{{ID: "X1", Text: "Pact: Iron Oath: 3 deaths, the limit is 3"}, {ID: "X2", Text: "Pact: resumed"}, {ID: "X3", Text: "something else"}}
	a.proposals = func() []decide.Proposal { return pend }
	a.incidents = func() []JournalLine { return lines }
	a.Tick()
	a.Tick() // the same signals again raise nothing
	v := a.View(0)
	if len(v.Alerts) != 2 || v.Alerts[0].Text != "The hero would like to leave Blood Moor for Cold Plains and awaits your word." ||
		v.Alerts[1].Text != "A Pact holds the hero in town and waits for you: Iron Oath: 3 deaths, the limit is 3" {
		t.Fatalf("%+v", v.Alerts)
	}
}

func TestAlertsAPIAndTokenGates(t *testing.T) {
	r, _ := alertsRig(t, "Sorceress", nil)
	if resp, _ := r.do("GET", r.view, AlertsRoute, "", ""); resp.StatusCode != http.StatusUnauthorized {
		t.Errorf("no token: %d", resp.StatusCode)
	}
	if resp, _ := r.do("GET", r.view, AlertsRoute, "wrong", ""); resp.StatusCode != http.StatusUnauthorized {
		t.Errorf("wrong token: %d", resp.StatusCode)
	}
	if resp, _ := r.do("POST", r.view, AlertsRoute, funToken, "{}"); resp.StatusCode != http.StatusMethodNotAllowed && resp.StatusCode != http.StatusNotFound {
		t.Errorf("POST: %d", resp.StatusCode)
	}
	if resp, _ := r.do("GET", r.view, AlertsRoute+"?since=abc", funToken, ""); resp.StatusCode != http.StatusBadRequest {
		t.Errorf("bad since: %d", resp.StatusCode)
	}
	resp, b := r.do("GET", r.view, AlertsUIRoute, funToken, "")
	if resp.StatusCode != 200 || !strings.Contains(string(b), "requestPermission") || strings.Contains(string(b), "requestPermission();") && !strings.Contains(string(b), "click") {
		t.Errorf("ui script: %d", resp.StatusCode)
	}
	if resp, _ := r.do("GET", r.view, AlertsUIRoute, "", ""); resp.StatusCode != http.StatusUnauthorized {
		t.Errorf("ui script without token: %d", resp.StatusCode)
	}
	// the page: permission is asked on a click only, never at load
	js := string(b)
	i := strings.Index(js, "requestPermission")
	if i < 0 || !strings.Contains(js[max(0, i-120):i], "addEventListener('click'") {
		t.Error("Notification.requestPermission must sit inside the click handler")
	}
	if !strings.Contains(js, "NOT built") {
		t.Error("the script must say background push is not built")
	}
	v := alertsOf(r, 0)
	if !strings.Contains(v.Notifications.Background, "not built") || v.Outbound.Enabled || v.Outbound.Consent != "Outbound alerts are off. Nothing leaves this machine." {
		t.Errorf("view: %+v", v)
	}
	// buying through the existing route opens the row
	r.feed(funStream()...) // 53 tally, price 10
	resp, body := r.do("POST", r.answer, UnlocksBuyRoute, funToken, `{"id":"alerts"}`)
	if resp.StatusCode != 200 {
		t.Fatalf("buy: %d %s", resp.StatusCode, body)
	}
	if v := alertsOf(r, 0); !v.Open {
		t.Errorf("not open after buying: %+v", v)
	}
}

func TestAlertsSincePaging(t *testing.T) {
	now := 0.0
	a := newTestAlerts(t, "sorceress", &now, nil)
	for i := 0; i < 130; i++ {
		a.Emit(AlertEvent{Kind: "death", Monster: "m", Area: "a"})
	}
	v := a.View(0)
	if len(v.Alerts) != alertsKeep || v.Seq != 130 || v.Alerts[0].Seq != 31 {
		t.Fatalf("ring: %d alerts, seq %d, first %d", len(v.Alerts), v.Seq, v.Alerts[0].Seq)
	}
	if v := a.View(128); len(v.Alerts) != 2 {
		t.Fatalf("since 128: %+v", v.Alerts)
	}
}

// --- outbound ---

type hit struct {
	body []byte
	hdr  http.Header
}

func recorder(status int) (*httptest.Server, *[]hit, *sync.Mutex) {
	var mu sync.Mutex
	var hits []hit
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		b, _ := io.ReadAll(r.Body)
		mu.Lock()
		hits = append(hits, hit{b, r.Header.Clone()})
		mu.Unlock()
		w.WriteHeader(status)
	}))
	return srv, &hits, &mu
}

func TestOutboundDefaultOffNeverContactsTheNetwork(t *testing.T) {
	srv, hits, mu := recorder(200)
	defer srv.Close()
	// a URL on file but not enabled: nothing is built, nothing is sent
	off := OutboundConfig{Enabled: false, Kind: "webhook", URL: srv.URL}
	if s, err := NewSender(off, srv.Client()); s != nil || err != nil {
		t.Fatalf("sender for an off setting: %v %v", s, err)
	}
	if s, err := NewSender(OutboundConfig{}, nil); s != nil || err != nil {
		t.Fatalf("zero value: %v %v", s, err)
	}
	cfgs := []*AlertsConfig{nil, {}, {Outbound: off}, {Outbound: off, Sender: &WebhookSender{URL: srv.URL, Client: srv.Client()}}}
	for i, ac := range cfgs {
		r, _ := alertsRig(t, "Sorceress", ac)
		r.feed(death(1))
		r.fun.Alerts.Flush()
		if r.fun.Alerts.sender != nil {
			t.Errorf("config %d: a sender exists while off", i)
		}
		if v := alertsOf(r, 0); v.Outbound.Enabled || v.Outbound.Target != "" || len(v.Alerts) != 1 {
			t.Errorf("config %d: %+v", i, v)
		}
	}
	mu.Lock()
	defer mu.Unlock()
	if len(*hits) != 0 {
		t.Fatalf("%d requests while the channel is off", len(*hits))
	}
	// the shipped profile config is off too
	if c, err := ParseOutboundConfig(nil); err != nil || c.Enabled {
		t.Fatalf("empty config: %+v %v", c, err)
	}
}

func TestOutboundConfigErrorsAndConsent(t *testing.T) {
	for in, want := range map[string]string{
		`{"enabled":true}`: "no url",
		`{"enabled":true,"url":"ftp://x/y","kind":"webhook"}`:                  "http or https",
		`{"enabled":true,"url":"https://x/y"}`:                                 `unknown kind ""`,
		`{"enabled":true,"url":"https://x/y","kind":"sms"}`:                    `unknown kind "sms"`,
		`{"enabled":true,"url":"https://x/y","kind":"ntfy","secret":"s"}`:      "secret only fits kind webhook",
		`{"enabled":true,"url":"https://x/y","kind":"ntfy","minSeverity":"x"}`: "minSeverity",
		`{"enabled":true,"url":"https://x/y","kind":"ntfy","extra":1}`:         "unknown field",
	} {
		if _, err := ParseOutboundConfig([]byte(in)); err == nil || !strings.Contains(err.Error(), want) {
			t.Errorf("%s: %v, want %q", in, err, want)
		}
	}
	raw := "https://" + "user" + ":" + "pw" + "@" + "ntfy.example/topic?auth=abc"
	c, err := ParseOutboundConfig([]byte(`{"enabled":true,"kind":"ntfy","url":"` + raw + `"}`))
	if err != nil {
		t.Fatal(err)
	}
	line := c.Consent()
	if !strings.HasPrefix(line, "Sends data to https://ntfy.example/topic:") || strings.Contains(line, "pw") || strings.Contains(line, "abc") {
		t.Fatalf("consent line: %s", line)
	}
	if (OutboundConfig{}).Consent() != "Outbound alerts are off. Nothing leaves this machine." {
		t.Fatal("off consent line")
	}
}

func TestWebhookSenderSignsAndDoesNotRetry(t *testing.T) {
	srv, hits, mu := recorder(500)
	defer srv.Close()
	cfg := OutboundConfig{Enabled: true, Kind: "webhook", URL: srv.URL, Secret: "s3cret", MinSeverity: "good"}
	r, eng := alertsRig(t, "Paladin", &AlertsConfig{Outbound: cfg, Client: srv.Client()})
	if _, err := eng.Buy("alerts", okSpender{}); err != nil {
		t.Fatal(err)
	}
	r.feed(death(1), goals.Event{Seq: 2, Type: goals.EvLevelUp, Level: 8}, drop(3, "Shako", "unique"))
	r.fun.Alerts.Flush()
	mu.Lock()
	defer mu.Unlock()
	// death (alarm) and the unique (good) pass the minimum "good"; the level-up (info) does not; one attempt each, no retry
	if len(*hits) != 2 {
		t.Fatalf("%d requests, want 2 (no retry, min severity)", len(*hits))
	}
	for _, h := range *hits {
		if h.hdr.Get("X-D2-Signature") != Sign("s3cret", h.body) || !strings.HasPrefix(h.hdr.Get("X-D2-Signature"), "sha256=") {
			t.Errorf("signature %q for %s", h.hdr.Get("X-D2-Signature"), h.body)
		}
		var p WebhookPayload
		if err := json.Unmarshal(h.body, &p); err != nil || p.Schema != "watch.alert/1" || p.Text == "" {
			t.Errorf("payload %s: %v", h.body, err)
		}
	}
	if errs := r.fun.Alerts.Errors(); len(errs) != 2 || !strings.Contains(errs[0], "HTTP 500") {
		t.Errorf("errors: %v", errs)
	}
	if v := alertsOf(r, 0); !v.Outbound.Enabled || v.Outbound.Target != srv.URL || strings.Contains(v.Outbound.Consent, "s3cret") ||
		!strings.HasPrefix(v.Outbound.Consent, "Sends data to "+srv.URL) {
		t.Errorf("view: %+v", v.Outbound)
	}
}

func TestWebhookWithoutSecretIsUnsigned(t *testing.T) {
	srv, hits, mu := recorder(204)
	defer srv.Close()
	s := &WebhookSender{URL: srv.URL, Client: srv.Client()}
	if err := s.Send(context.Background(), Alert{Seq: 1, Rule: "death", Kind: "death", Severity: "alarm", Text: "x"}); err != nil {
		t.Fatal(err)
	}
	mu.Lock()
	defer mu.Unlock()
	if len(*hits) != 1 || (*hits)[0].hdr.Get("X-D2-Signature") != "" || (*hits)[0].hdr.Get("Content-Type") != "application/json" {
		t.Fatalf("%+v", *hits)
	}
}

func TestNtfySender(t *testing.T) {
	srv, hits, mu := recorder(200)
	defer srv.Close()
	cfg := OutboundConfig{Enabled: true, Kind: "ntfy", URL: srv.URL + "/topic", MinSeverity: "info"}
	s, err := NewSender(cfg, srv.Client())
	if err != nil || s.Name() != "ntfy" {
		t.Fatalf("%v %v", s, err)
	}
	if err := s.Send(context.Background(), Alert{Rule: "death", Severity: "alarm", Text: "The hero has died."}); err != nil {
		t.Fatal(err)
	}
	mu.Lock()
	defer mu.Unlock()
	h := (*hits)[0]
	if string(h.body) != "The hero has died." || h.hdr.Get("Priority") != "5" || h.hdr.Get("Title") != "d.idle" || h.hdr.Get("Tags") != "death" {
		t.Fatalf("%q %v", h.body, h.hdr)
	}
}

func TestSenderInterfaceAndRedirectNotFollowed(t *testing.T) {
	var _ Sender = (*WebhookSender)(nil)
	var _ Sender = (*NtfySender)(nil)
	var n atomic.Int32
	target := httptest.NewServer(http.HandlerFunc(func(http.ResponseWriter, *http.Request) { n.Add(1) }))
	defer target.Close()
	redir := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) { http.Redirect(w, r, target.URL, http.StatusFound) }))
	defer redir.Close()
	s, err := NewSender(OutboundConfig{Enabled: true, Kind: "webhook", URL: redir.URL}, nil) // the default client
	if err != nil {
		t.Fatal(err)
	}
	if err := s.Send(context.Background(), Alert{Text: "x", Severity: "alarm"}); err == nil || n.Load() != 0 {
		t.Fatalf("redirect followed (%d hits) or no error: %v", n.Load(), err)
	}
}

func TestShippedLadderHasTheAlertsRow(t *testing.T) {
	l, err := unlock.Shipped()
	if err != nil {
		t.Fatal(err)
	}
	e, err := l.Entry(AlertsID)
	if err != nil {
		t.Fatal(err)
	}
	if e.Trigger.Kind != "tally_spend" || e.Price <= 0 || e.Group != "ui" {
		t.Fatalf("%+v", e)
	}
	// every shipped rule is reachable from a free path or the row; the free ones never name the row
	rt, _ := ShippedAlertRules()
	for _, r := range rt.Rules {
		if r.Free && r.Event != "death" && r.Event != "low_life" {
			t.Errorf("rule %s: only death and low life may be free", r.ID)
		}
	}
}
