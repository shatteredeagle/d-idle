package setup

import (
	"path/filepath"
	"regexp"
	"strings"
	"testing"

	"d2lod-bot/internal/bot"
	"d2lod-bot/internal/fleet"
	"d2lod-bot/web"
)

// The wizard page must ship the goal and handicap pickers and every option must pass the server-side validation.
func TestSetupUIPickers(t *testing.T) {
	b, err := web.FS.ReadFile("setup.js")
	if err != nil {
		t.Fatal(err)
	}
	js := string(b)
	for _, id := range []string{"f-profile", "f-pValue", "f-handicap", "f-charName", "f-charClass", "body.goals = pv", "body.profile = form.handicap"} {
		if !strings.Contains(js, id) {
			t.Errorf("setup.js lacks %q", id)
		}
	}
	ok := func(s fleet.Spec) bool { s.Character = "Wizbot"; return s.Validate() == nil }
	for _, m := range regexp.MustCompile(`\['(campaign:[a-z0-9_]+)'`).FindAllStringSubmatch(js, -1) {
		if !ok(fleet.Spec{Goals: m[1]}) {
			t.Errorf("goal preset %q rejected", m[1])
		}
	}
	for _, v := range []string{"levelto:99", "farm:mephisto", "farm:a_b.c-d"} {
		if !ok(fleet.Spec{Goals: v}) {
			t.Errorf("%q rejected", v)
		}
	}
	for _, m := range regexp.MustCompile(`HANDICAPS = \[(.*)\];`).FindStringSubmatch(js)[1:] {
		for _, n := range regexp.MustCompile(`\['([a-z]*)',`).FindAllStringSubmatch(m, -1) {
			if _, err := bot.ProfileByName(n[1]); err != nil {
				t.Errorf("handicap %q: %v", n[1], err)
			}
		}
	}
}

// Goals and handicap are separate inputs; a goal preset is not a handicap name and the reverse.
func TestWizardInputsGoalsVsProfile(t *testing.T) {
	w, err := New(filepath.Join(t.TempDir(), "setup.json"), nil)
	if err != nil {
		t.Fatal(err)
	}
	if err := w.SetInputs(Inputs{Profile: "reach:8"}); err == nil {
		t.Error("goal string accepted as handicap profile")
	}
	if err := w.SetInputs(Inputs{Goals: "rookie"}); err == nil {
		t.Error("handicap name accepted as goals")
	}
	if err := w.SetInputs(Inputs{Profile: "rookie", Goals: "campaign:normal_act1,levelto:20"}); err != nil {
		t.Fatal(err)
	}
	if in := w.Inputs(); in.Profile != "rookie" || in.Goals != "campaign:normal_act1,levelto:20" {
		t.Errorf("inputs %+v", in)
	}
}
