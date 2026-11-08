package watch

import (
	"encoding/json"
	"net/http"
	"strings"
	"testing"

	idrevealdata "d2lod-bot/data/idreveal"
	"d2lod-bot/internal/goals"
	"d2lod-bot/internal/unlock"
)

type fakeAffixes map[string][]string

func (f fakeAffixes) Affixes(name string) []string { return f[name] }

func revealOf(r *funRig) RevealView {
	var v RevealView
	r.getJSON(RevealRoute, &v)
	return v
}

func dropEv(name, quality string, seq uint64) goals.Event {
	return goals.Event{Type: goals.EvDrop, Item: &goals.Item{Name: name, Quality: quality}, Seq: seq}
}

func revealRig(t *testing.T, class string, aff AffixSource) (*funRig, *unlock.Engine) {
	t.Helper()
	l, err := unlock.Shipped()
	if err != nil {
		t.Fatal(err)
	}
	eng, err := unlock.Open(t.TempDir(), strings.ToLower(class), l, nil)
	if err != nil {
		t.Fatal(err)
	}
	return newFunRigWith(t, class, 1, func(_ *funRig, fc *FunConfig) { fc.Unlock, fc.Affixes = eng, aff }), eng
}

func TestRevealFaceDownThenFlipTwoClasses(t *testing.T) {
	for _, class := range []string{"Sorceress", "Paladin"} {
		t.Run(class, func(t *testing.T) {
			r, eng := revealRig(t, class, fakeAffixes{"Short Sword": {"+12 Defense", "of the Fox"}})
			if v := revealOf(r); v.Open || v.Teaser == "" || len(v.Cards) != 0 {
				t.Fatalf("locked: %+v", v)
			}
			r.feed(dropEv("Wirt's Leg", "normal", 1)) // plain items never make a card
			if eng.Allowed(IDRevealID) {
				t.Fatal("a normal item opened the reveal")
			}
			r.feed(dropEv("Short Sword", "magic", 2))
			if !eng.Allowed(IDRevealID) {
				t.Fatal("the first magic item did not open the reveal")
			}
			// face down: nothing about the item in the payload, not even its name
			_, raw := r.do("GET", r.view, RevealRoute, funToken, "")
			v := revealOf(r)
			if len(v.Cards) != 1 || v.Cards[0].Revealed {
				t.Fatalf("%+v", v)
			}
			for _, leak := range []string{"magic", "q-magic", "Short Sword", "Magic", "blue"} {
				if strings.Contains(string(raw), leak) {
					t.Errorf("a face-down card leaks %q: %s", leak, raw)
				}
			}
			b, _ := json.Marshal(v)
			golden(t, "reveal-facedown-"+strings.ToLower(class), string(b)+"\n")
			// flip
			resp, body := r.do("POST", r.answer, RevealFlipRoute, funToken, `{"id":"c1"}`)
			if resp.StatusCode != 200 {
				t.Fatalf("flip: %d %s", resp.StatusCode, body)
			}
			var fr revealFlipReply
			json.Unmarshal(body, &fr)
			c := fr.Card
			if c == nil || c.Quality != "magic" || c.Colour != "q-magic" || c.Name != "Short Sword" || len(c.Cue) == 0 || len(c.Affixes) != 2 || c.Text == "" {
				t.Fatalf("card %+v", c)
			}
			b, _ = json.Marshal(revealOf(r))
			golden(t, "reveal-flipped-"+strings.ToLower(class), string(b)+"\n")
			if resp, _ := r.do("POST", r.answer, RevealFlipRoute, funToken, `{"id":"c1"}`); resp.StatusCode != http.StatusConflict {
				t.Fatalf("flip again: %d", resp.StatusCode)
			}
			found := false
			for _, l := range storyLines(r) {
				found = found || strings.Contains(l, "Unlocked: Identify reveal.")
			}
			if !found {
				t.Fatalf("no reveal line: %v", storyLines(r))
			}
		})
	}
}

func TestRevealNoAffixReaderSaysSo(t *testing.T) {
	r, _ := revealRig(t, "Sorceress", nil)
	r.feed(dropEv("Ring", "rare", 1))
	_, body := r.do("POST", r.answer, RevealFlipRoute, funToken, `{"id":"c1"}`)
	var fr revealFlipReply
	json.Unmarshal(body, &fr)
	if fr.Card == nil || fr.Card.Quality != "rare" || len(fr.Card.Affixes) != 0 || !strings.Contains(fr.Card.AffixNote, "not read") {
		t.Fatalf("%s", body)
	}
}

func TestRevealUniquesWaitForTheCodexFirst(t *testing.T) {
	r, eng := revealRig(t, "Paladin", nil)
	r.feed(dropEv("Cap", "magic", 1))
	// the unique is stored but not shown until the Codex has seen the first unique; the fun stream's drop does that
	r.feed(goals.Event{Type: goals.EvDrop, Item: &goals.Item{Name: "Tyrael's Might", Quality: "unique"}, Seq: 2})
	v := revealOf(r)
	if !eng.Allowed(IDRevealLaterID) {
		t.Fatal("the first unique in the Codex did not open the later cards")
	}
	if !v.Later || len(v.Cards) != 2 {
		t.Fatalf("%+v", v)
	}
	// before that, the same unique id could not be probed
	r2, _ := revealRig(t, "Paladin", nil)
	r2.fun.Reveal.OnGoalEvent(dropEv("Cap", "magic", 1))
	r2.fun.Unlocks.Observe(unlock.Event{Type: unlock.EvItem, Name: "magic"})
	r2.fun.Reveal.OnGoalEvent(dropEv("Griswold's Edge", "unique", 2))
	if v := revealOf(r2); len(v.Cards) != 1 {
		t.Fatalf("a unique showed before its Codex first: %+v", v)
	}
	if resp, body := r2.do("POST", r2.answer, RevealFlipRoute, funToken, `{"id":"c2"}`); resp.StatusCode != http.StatusNotFound || !strings.Contains(string(body), `c2`) {
		t.Fatalf("probe: %d %s", resp.StatusCode, body)
	}
}

func TestRevealReplayAndGates(t *testing.T) {
	r, _ := revealRig(t, "Sorceress", nil)
	r.feed(dropEv("Cap", "magic", 1), dropEv("Ring", "rare", 2))
	r.feed(dropEv("Cap", "magic", 1), dropEv("Ring", "rare", 2)) // a replay
	if v := revealOf(r); len(v.Cards) != 2 {
		t.Fatalf("replay added cards: %+v", v)
	}
	post := func(token, body string) int {
		resp, _ := r.do("POST", r.answer, RevealFlipRoute, token, body)
		return resp.StatusCode
	}
	if c := post("", `{"id":"c1"}`); c != http.StatusUnauthorized {
		t.Fatalf("no token: %d", c)
	}
	if resp, _ := r.do("GET", r.answer, RevealFlipRoute, funToken, ""); resp.StatusCode != http.StatusMethodNotAllowed {
		t.Fatalf("GET: %d", resp.StatusCode)
	}
	if resp, _ := r.do("POST", r.view, RevealFlipRoute, funToken, `{"id":"c1"}`); resp.StatusCode == 200 {
		t.Fatal("the viewer must not take the flip route")
	}
	if c := post(funToken, `{"id":"c1","x":1}`); c != http.StatusBadRequest {
		t.Fatalf("unknown field: %d", c)
	}
	if c := post(funToken, `{"id":"ghost"}`); c != http.StatusNotFound {
		t.Fatalf("unknown card: %d", c)
	}
	if got := revealOf(r).Cards; got[0].Revealed || got[1].Revealed {
		t.Fatal("a refused flip changed a card")
	}
}

func TestRevealUnknownRowAndTable(t *testing.T) {
	l, _ := unlock.Parse([]byte(`{"schema":"unlock.ladder/1","entries":[
 {"id":"idreveal","group":"ui","trigger":{"kind":"item_quality","args":{"qualities":["magic"]}},"reveal":{"title":"R","text":"r"}}]}`))
	eng, _ := unlock.Open("", "p", l, nil)
	if _, err := NewReveal(&Unlocks{Engine: eng}, nil, nil, nil); err == nil || !strings.Contains(err.Error(), `"idreveal_later"`) {
		t.Fatalf("missing row: %v", err)
	}
	good, err := ParseRevealTable(idrevealdata.JSON)
	if err != nil {
		t.Fatal(err)
	}
	mut := func(f func(*RevealTable)) error {
		var c RevealTable
		b, _ := json.Marshal(good)
		json.Unmarshal(b, &c)
		f(&c)
		b, _ = json.Marshal(c)
		_, err := ParseRevealTable(b)
		return err
	}
	if err := mut(func(c *RevealTable) { q := c.Qualities["rare"]; q.Text = q.Text[:2]; c.Qualities["rare"] = q }); err == nil || !strings.Contains(err.Error(), `"rare"`) {
		t.Errorf("few texts: %v", err)
	}
	if err := mut(func(c *RevealTable) { c.First = append(c.First, "ethereal") }); err == nil || !strings.Contains(err.Error(), `"ethereal"`) {
		t.Errorf("unknown quality: %v", err)
	}
	if err := mut(func(c *RevealTable) { c.Later = append(c.Later, "magic") }); err == nil || !strings.Contains(err.Error(), `"magic"`) {
		t.Errorf("overlap: %v", err)
	}
	if _, err := unlock.Parse([]byte(`{"schema":"unlock.ladder/1","entries":[{"id":"x","group":"ui","trigger":{"kind":"item_quality","args":{"qualities":[]}},"reveal":{"title":"t","text":"x"}}]}`)); err == nil {
		t.Error("empty qualities accepted")
	}
}
