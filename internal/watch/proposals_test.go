package watch

import (
	"bytes"
	"encoding/json"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"d2lod-bot/internal/currency"
	"d2lod-bot/internal/decide"
)

var updateProposals = os.Getenv("UPDATE_PROPOSALS") != ""

type propRig struct {
	*funRig
	facts   ProposalFacts
	propose *decide.Proposer
	path    string
}

// tallyRanges is the one zone table: the tally strip's.
func tallyRanges(rows *currency.TallyRows) decide.RangeFn {
	return func(area string) (int, int, bool) {
		for _, z := range rows.Zones {
			if strings.EqualFold(z.Area, area) {
				return z.Min, z.Max, true
			}
		}
		return 0, 0, false
	}
}

func newPropRig(t *testing.T, class string, level int) *propRig {
	t.Helper()
	pr := &propRig{facts: ProposalFacts{Kit: "OK", Potions: 4, Ok: true}, path: filepath.Join(t.TempDir(), "proposals.json")}
	pr.funRig = newFunRigWith(t, class, level, func(r *funRig, fc *FunConfig) {
		tbl, err := decide.ShippedProposeTable()
		if err != nil {
			t.Fatal(err)
		}
		if pr.propose, err = decide.NewProposer(tbl, tallyRanges(currency.DefaultTallyRows()), r.clock, pr.path); err != nil {
			t.Fatal(err)
		}
		fc.Proposals = &ProposalConfig{Proposer: pr.propose, Clock: r.clock, Facts: func() ProposalFacts { return pr.facts }}
	})
	pr.hub.SetHero(HeroInfo{Class: class, Level: level, Area: "Blood Moor", AreaID: 2})
	return pr
}

func (r *propRig) post(token, path, body string) (int, map[string]any) {
	resp, b := r.do("POST", r.answer, path, token, body)
	var m map[string]any
	_ = json.Unmarshal(b, &m)
	return resp.StatusCode, m
}

func (r *propRig) section() SectionView {
	var d ReturnData
	r.getJSON("/api/return?hours=1", &d)
	for _, s := range d.Sections {
		if s.ID == "proposals" {
			return s
		}
	}
	r.t.Fatal("no proposals section")
	return SectionView{}
}

func (r *propRig) golden(name string, v any) {
	r.t.Helper()
	got, _ := json.MarshalIndent(v, "", " ")
	got = append(got, '\n')
	path := filepath.Join("testdata", "proposals", name)
	if updateProposals {
		os.MkdirAll(filepath.Dir(path), 0o755)
		if err := os.WriteFile(path, got, 0o644); err != nil {
			r.t.Fatal(err)
		}
	}
	want, err := os.ReadFile(path)
	if err != nil {
		r.t.Fatalf("%v (UPDATE_PROPOSALS=1 writes it)", err)
	}
	if !bytes.Equal(got, want) {
		r.t.Errorf("%s differs:\n%s", name, got)
	}
}

// The whole story for two builds: the hero is ready, the player is away, nothing advances, the proposal waits on the
// return screen, the player says yes and the order goes out.
func TestProposalAwayWaitsThenAccept(t *testing.T) {
	for _, tc := range []struct {
		class string
		level int
	}{{"Sorceress", 7}, {"Paladin", 6}} {
		t.Run(tc.class, func(t *testing.T) {
			r := newPropRig(t, tc.class, tc.level)
			r.do("GET", r.view, "/watch?live=1", funToken, "") // the last look
			r.fun.Settle()
			r.wall = r.wall.Add(25 * time.Minute) // away
			for i := 0; i < 20; i++ {
				r.clock.Advance(30 * time.Minute)
				r.fun.Settle()
			}
			if n := len(r.hub.orders); n != 0 {
				t.Fatalf("%d orders placed while away: the hero must not advance by itself", n)
			}
			if h := r.hub.heroNow(); h.Area != "Blood Moor" {
				t.Fatalf("hero moved to %s", h.Area)
			}
			var pv ProposalsView
			r.getJSON("/api/proposals", &pv)
			if len(pv.Pending) != 1 || !pv.Away || pv.Pending[0].To != "Cold Plains" || pv.Pending[0].FarmingMins < 9*60 {
				t.Fatalf("view: %+v", pv)
			}
			if !strings.HasPrefix(pv.Pending[0].Intro, "First time in Cold Plains: ") {
				t.Errorf("codex intro: %q", pv.Pending[0].Intro)
			}
			sec := r.section()
			if len(sec.Buttons) != 3 || sec.Buttons[0].Act != "proposal" || !strings.Contains(strings.Join(sec.Lines, "\n"), "farming Blood Moor for") {
				t.Fatalf("section: %+v", sec)
			}
			r.golden(strings.ToLower(tc.class)+"-section.json", sec)

			// the page carries the buttons
			if _, b := r.do("GET", r.view, "/watch/return", funToken, ""); !strings.Contains(string(b), `data-act="proposal"`) {
				t.Error("the return page has no proposal buttons")
			}

			// token gates
			if code, _ := r.post("", "/api/proposals/answer", `{"id":1,"answer":"go"}`); code != http.StatusUnauthorized {
				t.Errorf("no token: %d", code)
			}
			if resp, _ := r.do("GET", r.view, "/api/proposals", "", ""); resp.StatusCode != http.StatusUnauthorized {
				t.Errorf("viewer without token: %d", resp.StatusCode)
			}
			if resp, _ := r.do("POST", r.view, "/api/proposals/answer", funToken, `{"id":1,"answer":"go"}`); resp.StatusCode != http.StatusMethodNotAllowed && resp.StatusCode != http.StatusNotFound {
				t.Errorf("POST on the viewer: %d", resp.StatusCode)
			}
			if code, m := r.post(funToken, "/api/proposals/answer", `{"id":99,"answer":"go"}`); code != 404 {
				t.Errorf("unknown id: %d %v", code, m)
			}
			if code, m := r.post(funToken, "/api/proposals/answer", `{"id":1,"answer":"now"}`); code != 409 {
				t.Errorf("bad answer: %d %v", code, m)
			}
			if len(r.hub.orders) != 0 {
				t.Fatal("a refused answer placed an order")
			}

			// the yes places the order and closes the proposal
			code, m := r.post(funToken, "/api/proposals/answer", `{"id":1,"answer":"go"}`)
			if code != 200 || m["state"] != "accepted" || m["order"] == nil {
				t.Fatalf("accept: %d %v", code, m)
			}
			if len(r.hub.orders) != 1 || r.hub.orders[0].Area != "Cold Plains" || r.hub.orders[0].Kind != OrderGo {
				t.Fatalf("orders: %+v", r.hub.orders)
			}
			if sec := r.section(); len(sec.Lines) != 0 || sec.Placeholder == "" {
				t.Errorf("answered proposal still listed: %+v", sec)
			}
		})
	}
}

func TestProposalStayPersistsAndStaysQuiet(t *testing.T) {
	r := newPropRig(t, "Paladin", 6)
	r.fun.Settle()
	if code, m := r.post(funToken, "/api/proposals/answer", `{"id":1,"answer":"stay"}`); code != 200 || m["state"] != "stayed" {
		t.Fatalf("stay: %d %v", code, m)
	}
	r.clock.Advance(time.Minute)
	for i := 0; i < 5; i++ { // the hero looks every 2 s; a refused proposal is not asked again
		r.fun.Settle()
	}
	if n := len(r.propose.Pending()); n != 0 {
		t.Fatalf("re-asked right after a stay: %d pending", n)
	}
	if len(r.hub.orders) != 0 {
		t.Fatal("an order was placed")
	}
	// a restart keeps the answer
	again, err := decide.NewProposer(r.propose2Table(t), tallyRanges(currency.DefaultTallyRows()), r.clock, r.path)
	if err != nil {
		t.Fatal(err)
	}
	if pr, _ := again.Look(decide.ProposeInput{Class: "paladin", Clvl: 6, Area: "Blood Moor", Kit: "OK", Potions: 4}); pr != nil {
		t.Fatalf("a stay did not survive the reload: %+v", pr)
	}
}

func (r *propRig) propose2Table(t *testing.T) *decide.ProposeTable {
	tbl, err := decide.ShippedProposeTable()
	if err != nil {
		t.Fatal(err)
	}
	return tbl
}

func TestProposalKitGateAndFactsNotRead(t *testing.T) {
	r := newPropRig(t, "Paladin", 9)
	r.facts = ProposalFacts{Ok: false}
	r.fun.Settle()
	r.facts = ProposalFacts{Kit: "THIN", Potions: 4, Ok: true}
	r.fun.Settle()
	if n := len(r.propose.Pending()); n != 0 {
		t.Fatalf("proposed with a thin kit or unread facts: %d", n)
	}
	r.facts.Kit = "OK"
	r.fun.Settle()
	if n := len(r.propose.Pending()); n != 1 {
		t.Fatalf("kit ok: %d pending", n)
	}
}

// The order goes through the charter placer: a proposal that is no longer in range (the hero lost levels) is priced, the
// answer is refused with 402 and the proposal stays pending.
func TestProposalAcceptAboveRangeCostsACharter(t *testing.T) {
	r := newPropRig(t, "Paladin", 6)
	r.fun.Settle()
	r.hub.SetHero(HeroInfo{Class: "Paladin", Level: 2, Area: "Blood Moor", AreaID: 2})
	code, m := r.post(funToken, "/api/proposals/answer", `{"id":1,"answer":"go"}`)
	ch, _ := m["charter"].(map[string]any)
	if code != 402 || m["code"] != "charter-needed" || ch == nil || ch["price"].(float64) != 40 {
		t.Fatalf("above range: %d %v", code, m)
	}
	if len(r.hub.orders) != 0 || len(r.propose.Pending()) != 1 {
		t.Fatalf("a refused charter must change nothing: orders %d, pending %d", len(r.hub.orders), len(r.propose.Pending()))
	}
}

func TestProposalDeskNeedsItsParts(t *testing.T) {
	if _, err := NewProposalDesk(nil, ProposalConfig{}); err == nil {
		t.Fatal("an empty config must be refused")
	}
}

func TestSectionWithoutDeskKeepsThePlaceholder(t *testing.T) {
	r := newFunRig(t, "Paladin", 6)
	for _, s := range func() []SectionView { var d ReturnData; r.getJSON("/api/return", &d); return d.Sections }() {
		if s.ID == "proposals" && s.Placeholder == "" {
			t.Error("placeholder lost")
		}
	}
}
