package watch

import (
	"net/http"
	"strings"
	"testing"
	"time"

	"d2lod-bot/internal/currency"
	"d2lod-bot/internal/unlock"
)

const unlockFixture = `{"schema":"unlock.ladder/1","entries":[
 {"id":"map","group":"ui","trigger":{"kind":"codex_first","args":{"sub":"area"}},"alt":[{"kind":"level","args":{"min":2}}],
  "reveal":{"title":"Minimap","text":"You can see the map.","teaser":"Meet a new place."}},
 {"id":"dots","group":"ui","trigger":{"kind":"codex_first","args":{"sub":"monster"}},"reveal":{"title":"Enemy dots","text":"Red dots.","teaser":"Meet a monster."}},
 {"id":"elite","group":"ui","trigger":{"kind":"codex_first","args":{"sub":"champion"}},"reveal":{"title":"Elites","text":"Gold rings.","teaser":"Meet a champion."}},
 {"id":"qlog","group":"ui","trigger":{"kind":"quest","args":{"quest":"den_of_evil"}},"reveal":{"title":"Quest log","text":"Quests.","teaser":"Finish a quest."}},
 {"id":"loot","group":"ui","trigger":{"kind":"tally_spend"},"price":20,"reveal":{"title":"Eye for loot","text":"Loot markers.","teaser":"Costs 20 tally."}},
 {"id":"danger","group":"ui","trigger":{"kind":"tally_spend"},"price":1000,"reveal":{"title":"Danger","text":"Rating.","teaser":"Costs 1000 tally."}}
]}`

func unlockRig(t *testing.T, class string) (*funRig, *unlock.Engine) {
	t.Helper()
	l, err := unlock.Parse([]byte(unlockFixture))
	if err != nil {
		t.Fatal(err)
	}
	eng, err := unlock.Open(t.TempDir(), strings.ToLower(class), l, func() time.Time { return time.Unix(1_800_000_000, 0) })
	if err != nil {
		t.Fatal(err)
	}
	r := newFunRigWith(t, class, 1, func(_ *funRig, fc *FunConfig) { fc.Unlock = eng })
	return r, eng
}

func storyLines(r *funRig) []string {
	var out []string
	for _, l := range r.hub.Journal(0, "").Lines {
		if l.Kind == "story" && strings.HasPrefix(l.Text, "Unlocked: ") {
			out = append(out, l.Text)
		}
	}
	return out
}

func TestUnlocksRevealAndReturnSection(t *testing.T) {
	for _, class := range []string{"Sorceress", "Paladin"} { // monster and area firsts are world data: same for both
		t.Run(class, func(t *testing.T) {
			r, eng := unlockRig(t, class)
			var v UnlocksView
			r.getJSON(UnlocksRoute, &v)
			if len(v.Open) != 0 || len(v.Locked) != 6 {
				t.Fatalf("start: %+v", v)
			}
			r.feed(funStream()...)
			r.wall = r.wall.Add(21 * time.Minute)
			v = UnlocksView{}
			r.getJSON(UnlocksRoute, &v)
			var open []string
			for _, s := range v.Open {
				open = append(open, s.ID)
				if s.Text == "" || s.Teaser != "" {
					t.Errorf("open %+v", s)
				}
			}
			if strings.Join(open, ",") != "map,dots,qlog" || !eng.Allowed("map") || eng.Allowed("elite") {
				t.Fatalf("open %v", open)
			}
			for _, s := range v.Locked {
				if s.Text != "" || s.Teaser == "" {
					t.Errorf("locked %+v leaks or has no teaser", s)
				}
			}
			if got := storyLines(r); len(got) != 3 || !strings.Contains(got[0], "Minimap") {
				t.Fatalf("chronicle: %v", got)
			}
			var d ReturnData
			r.getJSON("/api/return", &d)
			var lines []string
			for _, s := range d.Sections {
				if s.ID == UnlocksSectionID {
					lines = s.Lines
					if s.Title != "Unlocked since you left" {
						t.Errorf("title %q", s.Title)
					}
				}
			}
			if len(lines) != 3 || !strings.Contains(strings.Join(lines, "|"), "Enemy dots: Red dots.") {
				t.Fatalf("return section: %v", lines)
			}
			// a replay of the same stream reveals nothing twice
			r.feed(funStream()...)
			if got := storyLines(r); len(got) != 3 {
				t.Fatalf("replay revealed again: %v", got)
			}
			if e := r.fun.Unlocks.Errors(); len(e) != 0 {
				t.Fatal(e)
			}
		})
	}
}

func TestUnlocksNoSectionWhenNothingOpened(t *testing.T) {
	r, _ := unlockRig(t, "Sorceress")
	r.wall = r.wall.Add(time.Minute)
	var d ReturnData
	r.getJSON("/api/return?hours=1", &d)
	for _, s := range d.Sections {
		if s.ID == UnlocksSectionID {
			t.Fatalf("empty section shown: %+v", s)
		}
	}
}

func TestUnlocksBuy(t *testing.T) {
	r, eng := unlockRig(t, "Sorceress")
	r.feed(funStream()...) // tally 53 (TestFunWiringEndToEnd)
	post := func(token, body string) (*http.Response, string) {
		resp, b := r.do("POST", r.answer, UnlocksBuyRoute, token, body)
		return resp, string(b)
	}
	if resp, _ := post("", `{"id":"loot"}`); resp.StatusCode != http.StatusUnauthorized {
		t.Fatalf("no token: %d", resp.StatusCode)
	}
	if resp, _ := r.do("GET", r.answer, UnlocksBuyRoute, funToken, ""); resp.StatusCode != http.StatusMethodNotAllowed {
		t.Fatalf("GET: %d", resp.StatusCode)
	}
	if resp, _ := r.do("POST", r.view, UnlocksBuyRoute, funToken, `{"id":"loot"}`); resp.StatusCode == 200 {
		t.Fatal("the viewer must not take the buy route")
	}
	req, _ := http.NewRequest("POST", r.answer+UnlocksBuyRoute, strings.NewReader(`{"id":"loot"}`))
	req.Header.Set("Authorization", "Bearer "+funToken)
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Origin", "http://evil.example")
	if resp, err := r.client.Do(req); err != nil || resp.StatusCode != http.StatusForbidden {
		t.Fatalf("origin: %v %v", resp, err)
	}
	if resp, _ := post(funToken, `{"id":"loot","x":1}`); resp.StatusCode != http.StatusBadRequest {
		t.Fatalf("unknown field: %d", resp.StatusCode)
	}
	if resp, b := post(funToken, `{"id":"ghost"}`); resp.StatusCode != http.StatusNotFound || !strings.Contains(b, "ghost") {
		t.Fatalf("unknown entry: %d %s", resp.StatusCode, b)
	}
	if resp, _ := post(funToken, `{"id":"qlog"}`); resp.StatusCode != http.StatusBadRequest {
		t.Fatalf("not for sale: %d", resp.StatusCode)
	}
	before, _ := r.ledger.Balance("tally")
	if resp, b := post(funToken, `{"id":"danger"}`); resp.StatusCode != http.StatusPaymentRequired || !strings.Contains(b, `"insufficient"`) {
		t.Fatalf("short: %d %s", resp.StatusCode, b)
	}
	if after, _ := r.ledger.Balance("tally"); after != before || eng.Allowed("danger") {
		t.Fatalf("a refused buy changed state: %+v -> %+v", before, after)
	}
	resp, b := post(funToken, `{"id":"loot"}`)
	if resp.StatusCode != 200 || !strings.Contains(b, `"open":true`) {
		t.Fatalf("buy: %d %s", resp.StatusCode, b)
	}
	after, _ := r.ledger.Balance("tally")
	if after.Spendable != before.Spendable-20 || !eng.Allowed("loot") {
		t.Fatalf("balance %d -> %d", before.Spendable, after.Spendable)
	}
	es, _ := r.ledger.Entries()
	last := es[len(es)-1]
	if last.Kind != currency.KindSpend || last.Amount != 20 || last.Key != "unlock/sorceress/loot" {
		t.Fatalf("ledger entry %+v", last)
	}
	if resp, _ := post(funToken, `{"id":"loot"}`); resp.StatusCode != http.StatusConflict {
		t.Fatalf("again: %d", resp.StatusCode)
	}
	if again, _ := r.ledger.Balance("tally"); again != after {
		t.Fatal("a second buy charged again")
	}
	if got := storyLines(r); len(got) == 0 || !strings.Contains(got[len(got)-1], "Eye for loot") {
		t.Fatalf("no reveal for a bought entry: %v", got)
	}
	var v UnlocksView
	r.getJSON(UnlocksRoute, &v)
	if v.Balance == nil || *v.Balance != after.Spendable {
		t.Fatalf("balance in view: %+v", v.Balance)
	}
}

func TestUnlocksBuyWithoutTally(t *testing.T) {
	l, _ := unlock.Parse([]byte(unlockFixture))
	eng, _ := unlock.Open("", "p", l, nil)
	r := newFunRigWith(t, "Sorceress", 1, func(_ *funRig, fc *FunConfig) { fc.Unlock = eng; fc.Strip = nil })
	resp, b := r.do("POST", r.answer, UnlocksBuyRoute, funToken, `{"id":"loot"}`)
	if resp.StatusCode != http.StatusServiceUnavailable || !strings.Contains(string(b), "tally-off") {
		t.Fatalf("%d %s", resp.StatusCode, b)
	}
}

func TestUnlocksOffWithoutEngine(t *testing.T) {
	r := newFunRig(t, "Sorceress", 1)
	if resp, _ := r.do("GET", r.view, UnlocksRoute, funToken, ""); resp.StatusCode != http.StatusNotFound {
		t.Fatalf("%d", resp.StatusCode)
	}
}
