package watch

import (
	"encoding/json"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"testing"
	"time"

	onboardingdata "d2lod-bot/data/onboarding"
	"d2lod-bot/internal/goals"
)

func onboardingRig(t *testing.T, path string) *funRig {
	return newFunRigWith(t, "Sorceress", 1, func(r *funRig, c *FunConfig) { c.OnboardingPath = path })
}

func (r *funRig) obGet() OnboardingView {
	r.t.Helper()
	var v OnboardingView
	r.getJSON(OnboardingRoute, &v)
	return v
}

func (r *funRig) obPost(body string) int {
	r.t.Helper()
	resp, _ := r.do("POST", r.answer, OnboardingRoute, funToken, body)
	return resp.StatusCode
}

func TestOnboardingTextsLoad(t *testing.T) {
	r := newFunRig(t, "Paladin", 1)
	v := r.obGet()
	if v.Schema != onboardingSchema || len(v.Steps) != 5 || !v.ShowWelcome || !v.ShowList || v.Done != 0 {
		t.Fatalf("fresh view: %+v", v)
	}
	var w struct {
		Lines   []string
		Buttons []struct{ ID, Label, Href string }
	}
	if err := json.Unmarshal(v.Texts.Welcome, &w); err != nil || len(w.Lines) < 3 || len(w.Lines) > 5 || len(w.Buttons) != 3 {
		t.Fatalf("welcome texts: %v %+v", err, w)
	}
	for _, s := range v.Steps {
		if s.Done || s.Label == "" {
			t.Errorf("step %+v", s)
		}
	}
}

// Dismissal, collapse and ticks are kept per hero in one file and survive a restart; a second hero starts fresh.
func TestOnboardingStatePersistsPerHero(t *testing.T) {
	path := filepath.Join(t.TempDir(), "state", "onboarding.json")
	r := onboardingRig(t, path)
	r.hub.SetHero(HeroInfo{Class: "Sorceress", Level: 1, Name: "Alpha"})
	for _, b := range []string{`{"action":"dismiss_welcome"}`, `{"action":"collapse"}`, `{"action":"tick","step":"journey"}`} {
		if c := r.obPost(b); c != 200 {
			t.Fatalf("%s: %d", b, c)
		}
	}
	v := r.obGet()
	if v.ShowWelcome || !v.Collapsed || v.Done != 1 || !v.Steps[3].Done {
		t.Fatalf("after posts: %+v", v)
	}
	if fi, err := os.Stat(path); err != nil || fi.Mode().Perm() != 0o600 {
		t.Fatalf("state file: %v %v", fi, err)
	}
	r2 := onboardingRig(t, path)
	r2.hub.SetHero(HeroInfo{Class: "Sorceress", Level: 1, Name: "Alpha"})
	if v := r2.obGet(); v.ShowWelcome || v.Done != 1 || v.Boot == "" {
		t.Fatalf("after restart: %+v", v)
	}
	r2.hub.SetHero(HeroInfo{Class: "Sorceress", Level: 1, Name: "Beta"})
	if v := r2.obGet(); !v.ShowWelcome || v.Done != 0 || v.Hero != "Beta" {
		t.Fatalf("second hero: %+v", v)
	}
	r2.hub.SetHero(HeroInfo{Class: "Sorceress", Level: 1, Name: "Alpha"})
	if c := r2.obPost(`{"action":"show_welcome"}`); c != 200 {
		t.Fatal(c)
	}
	if v := r2.obGet(); !v.ShowWelcome {
		t.Errorf("show again did not stick: %v", v.ShowWelcome)
	}
}

// The first kill ticks "fight" by itself; equip, junk and the policy sheet tick from what the player did; the list
// hides itself once every step is done and Help can bring it back.
func TestOnboardingAutoTicksAndHides(t *testing.T) {
	r := onboardingRig(t, "")
	r.feed(goals.Event{Type: goals.EvKill, Monster: "Fallen", XP: 10, Seq: 1})
	if v := r.obGet(); !v.Steps[2].Done || v.Steps[2].ID != "fight" {
		t.Fatalf("fight did not tick: %+v", v.Steps)
	}
	r.fun.Onboarding.Latch("equip")
	r.fun.Onboarding.Latch("junk")
	r.fun.Onboarding.Latch("nope") // unknown ids are ignored
	if resp, _ := r.do("GET", r.view, "/watch/policy", funToken, ""); resp.StatusCode == 200 {
		if v := r.obGet(); !v.Steps[1].Done {
			t.Errorf("opening the policy sheet did not tick: %+v", v.Steps)
		}
	} else {
		r.fun.Onboarding.Latch("policy") // the sheet is off in this rig
	}
	r.obPost(`{"action":"tick","step":"journey"}`)
	v := r.obGet()
	if !v.AllDone || v.ShowList || v.Done != 5 {
		t.Fatalf("done list must hide: %+v", v)
	}
	r.obPost(`{"action":"show_list"}`)
	if v := r.obGet(); !v.ShowList {
		t.Errorf("Help did not bring the list back: %+v", v)
	}
}

// The POST route has the same gates as the other answer routes and refuses unknown fields, actions and steps.
func TestOnboardingPostGates(t *testing.T) {
	r := onboardingRig(t, "")
	for name, tc := range map[string]struct {
		tok, body string
		want      int
	}{
		"no token":       {"", `{"action":"collapse"}`, 401},
		"unknown action": {funToken, `{"action":"burn"}`, 400},
		"unknown step":   {funToken, `{"action":"tick","step":"x"}`, 400},
		"unknown field":  {funToken, `{"action":"collapse","x":1}`, 400},
		"ok":             {funToken, `{"action":"collapse"}`, 200},
	} {
		if resp, _ := r.do("POST", r.answer, OnboardingRoute, tc.tok, tc.body); resp.StatusCode != tc.want {
			t.Errorf("%s: %d want %d", name, resp.StatusCode, tc.want)
		}
	}
	if resp, _ := r.do("POST", r.view, OnboardingRoute, funToken, `{"action":"collapse"}`); resp.StatusCode != 405 {
		t.Errorf("the viewer must stay GET only: %d", resp.StatusCode)
	}
}

// A long absence shows in the reply, from the same tracker as the return screen; a poll of it is not a look.
func TestOnboardingAway(t *testing.T) {
	r := onboardingRig(t, "")
	r.do("GET", r.view, "/watch?live=1", funToken, "")
	r.wall = r.wall.Add(2*time.Hour + 14*time.Minute)
	for i := 0; i < 2; i++ {
		v := r.obGet()
		if !v.Away.Absent || v.Away.Seconds < 7200 {
			t.Fatalf("away %d: %+v", i, v.Away)
		}
	}
}

func TestOnboardingTextsRejectBad(t *testing.T) {
	for _, s := range []string{`{`, `{"schema":"x"}`, `{"schema":"watch.onboarding.texts/1","steps":[{"id":"a","label":"A"},{"id":"a","label":"B"}]}`} {
		if _, err := LoadOnboardingTexts([]byte(s)); err == nil {
			t.Errorf("accepted %s", s)
		}
	}
}

// The texts hold no markup and no private data (they are shown through textContent, but the file is edited by hand).
func TestOnboardingTextsPlain(t *testing.T) {
	r := newFunRig(t, "Paladin", 1)
	b, _ := json.Marshal(r.obGet().Texts)
	for _, bad := range []string{"<", "http://", "https://"} {
		if strings.Contains(string(b), bad) {
			t.Errorf("texts contain %q", bad)
		}
	}
}

// The panel against a stub DOM (node, skipped without it): welcome, checklist states, Help, reconnect banners.
// UPDATE=1 rewrites testdata/onboarding_dom.txt.
func TestOnboardingPanelDOMGolden(t *testing.T) {
	runNode(t, "testdata/onboarding_dom.js", "web/panels/onboarding/panel.js")
}

// Page patterns of the panel: text only, CSP safe (no inline handler, style or script), and no focus or scroll jump on a
// poll. The only scroll is the one a click on Help asks for.
func TestOnboardingPanelPatterns(t *testing.T) {
	b, err := webFS.ReadFile("web/panels/onboarding/panel.js")
	if err != nil {
		t.Fatal(err)
	}
	src := string(b)
	for _, bad := range []string{"innerHTML", "outerHTML", "insertAdjacentHTML", "document.write", "eval(", "new Function", "onclick", "javascript:",
		"setAttribute('style'", ".style.", "cssText", "scrollTo(", "location.", "window.open", "fetch(", "setTimeout(", "setInterval(", "document.getElementById", "document.querySelector"} {
		if strings.Contains(src, bad) {
			t.Errorf("panel.js uses %q", bad)
		}
	}
	for _, m := range regexp.MustCompile(`\.focus\([^)]*\)`).FindAllString(src, -1) {
		if !strings.Contains(m, "preventScroll") {
			t.Errorf("focus without preventScroll: %s", m)
		}
	}
	if n := strings.Count(src, "scrollIntoView"); n != 2 { // the guard and the call, both inside the d2help handler
		t.Errorf("scrollIntoView appears %d times", n)
	}
	i := strings.Index(src, "scrollIntoView")
	h := strings.Index(src, "'d2help'")
	if h < 0 || i < h || i-h > 400 {
		t.Error("scrollIntoView must sit inside the d2help handler")
	}
	// every box is built through the key helper so the shell can give the focus back after a poll
	if strings.Count(src, "dataset.key") < 2 {
		t.Error("buttons and links need data-key for the shell's focus keeping")
	}
	css, _ := webFS.ReadFile("web/panels/onboarding/panel.css")
	if regexp.MustCompile(`#[0-9a-fA-F]{3,8}\b|rgb\(`).Match(css) {
		t.Error("panel.css uses a colour literal; use tokens")
	}
	shell, _ := webFS.ReadFile("web/shell.js")
	if !strings.Contains(string(shell), "'btn-help'") || !strings.Contains(string(shell), "'d2help'") {
		t.Error("shell.js does not wire the Help button to the d2help event")
	}
	page, _ := webFS.ReadFile("web/watch.html")
	if !strings.Contains(string(page), `id="btn-help"`) || strings.Contains(string(page), "onclick") {
		t.Error("watch.html needs the Help button and no inline handlers")
	}
}

// The step ids the Go side latches and the texts name stay the same set, and every link goes to a page of this server.
func TestOnboardingStepsMatchLatches(t *testing.T) {
	txt, err := LoadOnboardingTexts(onboardingdata.JSON)
	if err != nil {
		t.Fatal(err)
	}
	var ids []string
	for _, s := range txt.Steps {
		ids = append(ids, s.ID)
		if s.Href != "" && !strings.HasPrefix(s.Href, "/watch") && !strings.HasPrefix(s.Href, "#") {
			t.Errorf("step %s links to %s", s.ID, s.Href)
		}
	}
	if got := strings.Join(ids, ","); got != "equip,policy,fight,journey,junk" {
		t.Errorf("steps %s: onboardingSignals, Observe and opened latch these ids", got)
	}
}
