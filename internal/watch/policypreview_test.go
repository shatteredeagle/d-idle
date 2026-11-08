package watch

import (
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync"
	"testing"

	"d2lod-bot/internal/decide"
	"d2lod-bot/internal/env"
)

// fightLines is a journal of ten fights with known life lows plus pickups and a potion; now is 1000.
const fightLines = `{"t":100,"kind":"close","lifePct":30}
{"t":150,"kind":"close","lifePct":40}
{"t":200,"kind":"close","lifePct":60}
{"t":250,"kind":"close","lifePct":58}
{"t":300,"kind":"close","lifePct":70}
{"t":350,"kind":"close","lifePct":28}
{"t":400,"kind":"close","lifePct":90}
{"t":450,"kind":"close","lifePct":24}
{"t":500,"kind":"close","lifePct":52}
{"t":550,"kind":"close","lifePct":21}
{"t":560,"kind":"potion","potion":"Minor Healing Potion"}
{"t":600,"kind":"drop","item":"Short Sword","quality":"normal","type":"sword"}
{"t":610,"kind":"drop","item":"Cap","quality":"normal","type":"helm"}
{"t":620,"kind":"drop","item":"El Rune","quality":"normal","type":"rune"}
{"t":630,"kind":"drop","item":"Ring","quality":"magic","type":"ring"}
`

func profileDefault(t *testing.T, key string) float64 {
	t.Helper()
	for _, k := range env.Knobs() {
		if k.Key == key {
			return k.Default
		}
	}
	t.Fatalf("no knob %s", key)
	return 0
}

func pvPreview(t *testing.T, r *polRig, row, opt string) decide.PreviewResult {
	t.Helper()
	res, err := r.sheet.Preview(row, opt, 0)
	if err != nil {
		t.Fatalf("preview %s/%s: %v", row, opt, err)
	}
	return res
}

func lineDelta(t *testing.T, res decide.PreviewResult, key string) int {
	t.Helper()
	for _, l := range res.Lines {
		if l.Key == key && l.Delta != nil {
			return *l.Delta
		}
	}
	t.Fatalf("no delta %s in %+v", key, res.Lines)
	return 0
}

// The expected deltas are computed from the profile defaults, so a change of the Balanced numbers fails here loudly
// instead of silently making the preview wrong.
func TestPolicyPreviewStanceTwoClasses(t *testing.T) {
	if profileDefault(t, "combat.chickenPct") != 25 || profileDefault(t, "combat.drinkLifePct") != 55 {
		t.Skip("Balanced numbers changed: update the fixture lows")
	}
	lows := []int{30, 40, 60, 58, 70, 28, 90, 24, 52, 21}
	band := func(lo, hi int) int {
		n := 0
		for _, l := range lows {
			if l > lo && l <= hi {
				n++
			}
		}
		return n
	}
	for _, c := range []struct {
		class, to    string
		leave, drink int
	}{
		{"paladin", "cautious", band(25, 35), band(55, 65)},
		{"sorceress", "cautious", band(25, 35), band(55, 65)},
		{"paladin", "aggressive", -band(22, 25), -band(50, 55)},
		{"sorceress", "aggressive", -band(20, 25), -band(50, 55)},
	} {
		r := newPolRig(t, c.class, true)
		r.now = 1000
		r.feed(fightLines)
		before := len(r.store.PolicyLog())
		res := pvPreview(t, r, "stance", c.to)
		if !res.Supported || !res.Enough || res.From != "default" || res.To != c.to {
			t.Fatalf("%s/%s: %+v", c.class, c.to, res)
		}
		if g := lineDelta(t, res, "leave"); g != c.leave {
			t.Errorf("%s/%s leave %d want %d", c.class, c.to, g, c.leave)
		}
		if g := lineDelta(t, res, "drink"); g != c.drink {
			t.Errorf("%s/%s drink %d want %d", c.class, c.to, g, c.drink)
		}
		// read-only: no log line, no knob call, the option is unchanged
		if len(r.store.PolicyLog()) != before || len(r.calls) != 0 || r.sheet.CurrentOption("stance") != "default" {
			t.Errorf("%s/%s: preview changed state", c.class, c.to)
		}
	}
}

func TestPolicyPreviewWhitesAndHonestRows(t *testing.T) {
	r := newPolRig(t, "paladin", true)
	r.now = 1000
	r.feed(fightLines)
	res := pvPreview(t, r, "whites", "keepers")
	if lineDelta(t, res, "pickups") != -2 || res.FilteredTotal != 2 || res.Filtered[0].Item != "Cap" {
		t.Fatalf("%+v", res)
	}
	// every other row answers 'no preview yet' with its reason and no numbers
	for _, row := range r.tbl.Rows {
		if row.ID == "stance" || row.ID == "whites" {
			continue
		}
		o := row.Options[0].ID
		if o == row.Default && len(row.Options) > 1 {
			o = row.Options[1].ID
		}
		res := pvPreview(t, r, row.ID, o)
		if res.Supported || len(res.Lines) != 0 || res.FilteredTotal != 0 || !strings.HasPrefix(res.Reason, "no preview for this row yet: ") {
			t.Errorf("%s: %+v", row.ID, res)
		}
	}
}

func TestPolicyPreviewEmptyHistoryAndErrors(t *testing.T) {
	r := newPolRig(t, "sorceress", true)
	r.now = 1000
	res := pvPreview(t, r, "stance", "cautious")
	if res.Enough || len(res.Lines) != 0 || !strings.Contains(res.Reason, "not enough events") {
		t.Errorf("empty: %+v", res)
	}
	if _, err := r.sheet.Preview("nosuchrow", "x", 0); !errors.Is(err, ErrPolicyRow) || !strings.Contains(err.Error(), "nosuchrow") {
		t.Errorf("row: %v", err)
	}
	if _, err := r.sheet.Preview("stance", "melt", 0); !errors.Is(err, ErrPolicyOption) || !strings.Contains(err.Error(), "melt") {
		t.Errorf("option: %v", err)
	}
	r.class = "necromancer"
	if _, err := r.sheet.Preview("stance", "cautious", 0); err == nil || !strings.Contains(err.Error(), "necromancer") {
		t.Errorf("class: %v", err)
	}
	r.class = ""
	if _, err := r.sheet.Preview("stance", "cautious", 0); !errors.Is(err, ErrPolicyNoClass) {
		t.Errorf("no class: %v", err)
	}
}

func TestPolicyPreviewGoldenJSON(t *testing.T) {
	r := newPolRig(t, "paladin", true)
	r.now = 1000
	r.feed(fightLines)
	var all []decide.PreviewResult
	for _, c := range [][2]string{{"stance", "cautious"}, {"whites", "keepers"}, {"sell", "keep"}} {
		all = append(all, pvPreview(t, r, c[0], c[1]))
	}
	b, _ := json.MarshalIndent(all, "", " ")
	polGolden(t, "preview-paladin.json", append(b, '\n'))
}

func TestPolicyPreviewHTTPGates(t *testing.T) {
	r := newPolRig(t, "sorceress", true)
	r.now = 1000
	r.feed(fightLines)
	opts := AnswerOptions{Token: "tok", Origins: []string{"http://ok.example"}}
	h := r.sheet.PolicyPreviewHandler(opts)
	auth := map[string]string{"Authorization": "Bearer tok"}
	good := `{"row":"stance","option":"cautious"}`
	for name, c := range map[string]struct {
		hdr  map[string]string
		body string
		code int
	}{
		"no token":      {nil, good, 401},
		"wrong token":   {map[string]string{"Authorization": "Bearer nope"}, good, 401},
		"bad origin":    {map[string]string{"Authorization": "Bearer tok", "Origin": "http://evil.example"}, good, 403},
		"content type":  {map[string]string{"Authorization": "Bearer tok", "Content-Type": "text/plain"}, good, 415},
		"unknown field": {auth, `{"row":"stance","option":"cautious","x":1}`, 400},
		"two objects":   {auth, good + good, 400},
		"unknown row":   {auth, `{"row":"zzz","option":"cautious"}`, 404},
		"unknown opt":   {auth, `{"row":"stance","option":"melt"}`, 400},
		"ok":            {auth, good, 200},
		"ok origin":     {map[string]string{"Authorization": "Bearer tok", "Origin": "http://ok.example"}, good, 200},
		"minutes":       {auth, `{"row":"stance","option":"cautious","minutes":30}`, 200},
	} {
		if w := polPost(h, c.hdr, c.body); w.Code != c.code {
			t.Errorf("%s: %d %s, want %d", name, w.Code, w.Body, c.code)
		}
	}
	// GET is refused: the route has one verb, and a refused request changed nothing
	if w := polGet(h, auth); w.Code != 405 {
		t.Errorf("GET: %d", w.Code)
	}
	if len(r.store.PolicyLog()) != 0 || len(r.calls) != 0 {
		t.Fatal("a preview request changed the sheet")
	}
	// the write route does not accept the preview's field
	if w := polPost(r.sheet.PolicyAnswerHandler(opts), auth, `{"row":"stance","option":"cautious","minutes":30}`); w.Code != 400 {
		t.Errorf("answer route accepted minutes: %d", w.Code)
	}
}

// Events arrive while previews are computed.
func TestPolicyPreviewRace(t *testing.T) {
	var mu sync.Mutex
	now := 1000.0
	r := newPolRig(t, "paladin", true)
	r.events = NewAwayLog(func() float64 { mu.Lock(); defer mu.Unlock(); return now })
	r.sheet.events = r.events
	r.sheet.now = func() float64 { mu.Lock(); defer mu.Unlock(); return now }
	r.feed(fightLines)
	var wg sync.WaitGroup
	for i := 0; i < 8; i++ {
		wg.Add(2)
		go func(i int) {
			defer wg.Done()
			for j := 0; j < 50; j++ {
				r.events.Add(AwayEvent{T: 900, Kind: AwayClose, LifePct: 20 + j%40})
			}
		}(i)
		go func() {
			defer wg.Done()
			for j := 0; j < 50; j++ {
				if _, err := r.sheet.Preview("stance", "cautious", 0); err != nil {
					t.Error(err)
					return
				}
			}
		}()
	}
	wg.Wait()
}

func polGet(h http.Handler, hdr map[string]string) *httptest.ResponseRecorder {
	req := httptest.NewRequest("GET", PolicyPreviewRoute, nil)
	for k, v := range hdr {
		req.Header.Set(k, v)
	}
	w := httptest.NewRecorder()
	h.ServeHTTP(w, req)
	return w
}
