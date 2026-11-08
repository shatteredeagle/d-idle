package watch

import (
	"encoding/json"
	"errors"
	"net/http"
	"strings"
	"testing"

	voicedata "d2lod-bot/data/voice"
	"d2lod-bot/internal/decide"
	"d2lod-bot/internal/goals"
	"d2lod-bot/internal/unlock"
)

func voiceOf(r *funRig) VoiceView {
	var v VoiceView
	r.getJSON(VoiceRoute, &v)
	return v
}

func voiceStream() []goals.Event {
	evs := append(funStream(), goals.Event{Type: goals.EvDeath, Monster: "Fallen"})
	for i := range evs {
		evs[i].Seq = uint64(i + 1)
	}
	return evs
}

func idleLines(r *funRig) []string {
	var out []string
	for _, l := range r.hub.Journal(0, "").Lines {
		if l.Kind == "idle" {
			out = append(out, l.Text)
		}
	}
	return out
}

func TestVoiceGoldenTwoClasses(t *testing.T) {
	for _, class := range []string{"Sorceress", "Paladin"} {
		t.Run(class, func(t *testing.T) {
			r, eng := shippedRig(t, class)
			if v := voiceOf(r); v.Open || v.Teaser == "" || len(v.Diary) != 0 || v.Latest != nil {
				t.Fatalf("locked view: %+v", v)
			}
			r.feed(voiceStream()...)
			if !eng.Allowed(VoiceID) {
				t.Fatal("level-up did not open the voice")
			}
			b, _ := json.Marshal(voiceOf(r))
			golden(t, "voice-"+strings.ToLower(class), string(b)+"\n")
			// every milestone has a page and the hero said it in the feed
			v := voiceOf(r)
			kinds := map[string]bool{}
			for _, p := range v.Diary {
				kinds[p.Kind] = true
				if strings.Contains(p.Text, "{") || p.Milestone == "" {
					t.Errorf("unfilled page %+v", p)
				}
			}
			for _, k := range []string{"levelup_first", "death_first", "codex_monster", "codex_area"} {
				if !kinds[k] {
					t.Errorf("no page for %s: %v", k, kinds)
				}
			}
			// firsts met before the voice opened are backfilled as quiet pages (the first level-up opened it)
			backfilled := 0
			for backfilled < len(v.Diary) && v.Diary[backfilled].Kind != "levelup_first" {
				backfilled++
			}
			if backfilled != 4 || len(idleLines(r)) != len(v.Diary)-backfilled {
				t.Errorf("%d idle lines for %d pages, %d backfilled", len(idleLines(r)), len(v.Diary), backfilled)
			}
			found := false
			for _, l := range storyLines(r) {
				found = found || strings.Contains(l, "Unlocked: Hero voice.")
			}
			if !found {
				t.Fatalf("no reveal line: %v", storyLines(r))
			}
			if e := r.fun.Unlocks.Errors(); len(e) != 0 {
				t.Fatal(e)
			}
		})
	}
}

func TestVoiceReplayAndPersistence(t *testing.T) {
	r, eng := shippedRig(t, "Sorceress")
	r.feed(voiceStream()...)
	n := len(voiceOf(r).Diary)
	r.feed(voiceStream()...)
	if m := len(voiceOf(r).Diary); m != n {
		t.Fatalf("replay added pages: %d -> %d", n, m)
	}
	// a restart: a new Voice over the same state dir keeps the pages and says nothing twice
	v2, err := NewVoice(r.fun.Unlocks, r.hub, nil, func() string { return "Sorceress" }, nil)
	if err != nil {
		t.Fatal(err)
	}
	if got := v2.View(); len(got.Diary) != n || !eng.Allowed(VoiceID) {
		t.Fatalf("after reload: %d pages, want %d", len(got.Diary), n)
	}
	before := len(idleLines(r))
	v2.OnGoalEvent(goals.Event{Type: goals.EvLevelUp, Level: 7})
	v2.OnGoalEvent(goals.Event{Type: goals.EvDeath, Monster: "Fallen"})
	if len(idleLines(r)) != before {
		t.Fatal("the reloaded voice spoke an old milestone again")
	}
}

func TestVoiceSeededChoiceIsStable(t *testing.T) {
	pick := func(profile string) string {
		l, _ := unlock.Shipped()
		eng, _ := unlock.Open("", profile, l, nil)
		v, err := NewVoice(&Unlocks{Engine: eng}, nil, nil, nil, nil)
		if err != nil {
			t.Fatal(err)
		}
		s, err := v.pick("codex_monster", "codex_monster/fallen", "paladin")
		if err != nil {
			t.Fatal(err)
		}
		return s
	}
	if pick("hero-a") != pick("hero-a") {
		t.Fatal("same seed, different line")
	}
	differ := false
	for _, p := range []string{"b", "c", "d", "e", "f", "g", "h"} {
		differ = differ || pick(p) != pick("hero-a")
	}
	if !differ {
		t.Fatal("the seed does not change the choice")
	}
}

func TestVoiceMuteSwitch(t *testing.T) {
	r, _ := shippedRig(t, "Paladin")
	post := func(token, body string) (*http.Response, string) {
		resp, b := r.do("POST", r.answer, VoiceMuteRoute, token, body)
		return resp, string(b)
	}
	if resp, _ := post("", `{"muted":true}`); resp.StatusCode != http.StatusUnauthorized {
		t.Fatalf("no token: %d", resp.StatusCode)
	}
	if resp, _ := r.do("GET", r.answer, VoiceMuteRoute, funToken, ""); resp.StatusCode != http.StatusMethodNotAllowed {
		t.Fatalf("GET: %d", resp.StatusCode)
	}
	if resp, _ := r.do("POST", r.view, VoiceMuteRoute, funToken, `{"muted":true}`); resp.StatusCode == 200 {
		t.Fatal("the viewer must not take the mute route")
	}
	if resp, _ := post(funToken, `{}`); resp.StatusCode != http.StatusBadRequest {
		t.Fatalf("empty body: %d", resp.StatusCode)
	}
	if resp, b := post(funToken, `{"muted":true}`); resp.StatusCode != 200 {
		t.Fatalf("mute: %d %s", resp.StatusCode, b)
	}
	r.feed(voiceStream()...)
	v := voiceOf(r)
	if !v.Muted || len(v.Diary) == 0 {
		t.Fatalf("muted: %+v", v)
	}
	if n := len(idleLines(r)); n != 0 {
		t.Fatalf("a muted hero spoke %d lines", n)
	}
	post(funToken, `{"muted":false}`)
	r.fun.Voice.OnFired([]decide.Fired{{Entry: decide.CodexEntry{ID: "x_area", Kind: "area", Subject: "Cold Plains"}}})
	if n := len(idleLines(r)); n != 1 {
		t.Fatalf("unmuted: %d lines", n)
	}
}

func TestVoiceUnknownRowClassAndKind(t *testing.T) {
	l, _ := unlock.Parse([]byte(`{"schema":"unlock.ladder/1","entries":[
 {"id":"chronicle","group":"ui","trigger":{"kind":"kills_of_kind","args":{"kind":"*","min":1}},"reveal":{"title":"C","text":"c"}}]}`))
	eng, _ := unlock.Open("", "p", l, nil)
	if _, err := NewVoice(&Unlocks{Engine: eng}, nil, nil, nil, nil); err == nil || !strings.Contains(err.Error(), `"voice"`) || !errors.Is(err, unlock.ErrUnknownEntry) {
		t.Fatalf("missing row: %v", err)
	}
	u := mustUnlocks(t)
	u.Engine.Observe(unlock.Event{Type: unlock.EvLevel, N: 2})
	v, err := NewVoice(u, nil, nil, func() string { return "mage" }, nil)
	if err != nil {
		t.Fatal(err)
	}
	v.OnGoalEvent(goals.Event{Type: goals.EvLevelUp, Level: 2})
	v.OnFired([]decide.Fired{{Entry: decide.CodexEntry{ID: "q", Kind: "quest", Subject: "x"}}})
	errs := strings.Join(u.Errors(), "|")
	if !strings.Contains(errs, `unknown class "mage"`) || !strings.Contains(errs, `Codex kind "quest"`) {
		t.Fatalf("errors: %v", u.Errors())
	}
	if len(v.View().Diary) != 1 {
		t.Fatal("an unknown class must still get the general line")
	}
}

func TestParseVoiceLines(t *testing.T) {
	good, err := ParseVoiceLines(voicedata.JSON)
	if err != nil {
		t.Fatal(err)
	}
	var tbl map[string]any
	json.Unmarshal(voicedata.JSON, &tbl)
	for k, vk := range good.Kinds {
		if len(vk.Any) < 6 {
			t.Errorf("%s: %d variants", k, len(vk.Any))
		}
	}
	mutate := func(f func(*VoiceLines)) error {
		var c VoiceLines
		b, _ := json.Marshal(good)
		json.Unmarshal(b, &c)
		f(&c)
		b, _ = json.Marshal(c)
		_, err := ParseVoiceLines(b)
		return err
	}
	cases := []struct {
		name string
		f    func(*VoiceLines)
		want string
	}{
		{"few", func(c *VoiceLines) { k := c.Kinds["codex_area"]; k.Any = k.Any[:5]; c.Kinds["codex_area"] = k }, `"codex_area" needs 6`},
		{"unknown kind", func(c *VoiceLines) { c.Kinds["boss_kill"] = VoiceKind{} }, `unknown kind "boss_kill"`},
		{"missing kind", func(c *VoiceLines) { delete(c.Kinds, "codex_shrine") }, `"codex_shrine" is missing`},
		{"class", func(c *VoiceLines) {
			k := c.Kinds["death_first"]
			k.Classes = map[string][]string{"mage": {"x"}}
			c.Kinds["death_first"] = k
		}, `unknown class "mage"`},
		{"placeholder", func(c *VoiceLines) { k := c.Kinds["death_first"]; k.Any[0] = "Hello {who}"; c.Kinds["death_first"] = k }, `unknown placeholder {who}`},
	}
	for _, c := range cases {
		if err := mutate(c.f); err == nil || !strings.Contains(err.Error(), c.want) {
			t.Errorf("%s: %v, want %q", c.name, err, c.want)
		}
	}
}

// Spoilers: a line may use {name} for what the hero just met, but never name content of the Act 1 table itself.
func TestVoiceLinesNameNothingUnmet(t *testing.T) {
	good, err := ParseVoiceLines(voicedata.JSON)
	if err != nil {
		t.Fatal(err)
	}
	tbl, err := decide.ShippedFirsts()
	if err != nil {
		t.Fatal(err)
	}
	var names []string
	for _, f := range tbl.Firsts {
		switch f.Kind {
		case "monster", "area", "waypoint", "shrine":
			for _, m := range f.Match {
				if len(m) >= 4 {
					names = append(names, strings.ToLower(m))
				}
			}
		}
	}
	if len(names) == 0 {
		t.Fatal("no names to check against")
	}
	for k, vk := range good.Kinds {
		all := append([]string(nil), vk.Any...)
		for _, ls := range vk.Classes {
			all = append(all, ls...)
		}
		for _, l := range all {
			low := strings.ToLower(l)
			for _, n := range names {
				if strings.Contains(low, n) {
					t.Errorf("%s: line %q names %q", k, l, n)
				}
			}
		}
	}
}
