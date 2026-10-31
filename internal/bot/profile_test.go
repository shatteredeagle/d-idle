package bot

import (
	"reflect"
	"strings"
	"testing"

	"d2lod-bot/internal/env"
)

func TestEveryKnobResolvesToAField(t *testing.T) {
	var tu env.Tuning
	for _, k := range env.Knobs() {
		if _, err := GetKnob(&tu, k); err != nil {
			// Canary for knob naming: a key must be group.fieldName. Only bot knobs are ours to fail on; others are reported.
			if strings.HasPrefix(k.Key, "bot.") {
				t.Errorf("%s: %v", k.Key, err)
			} else {
				t.Logf("KNOB MISMATCH (owner must fix): %s: %v", k.Key, err)
			}
			continue
		}
		if k.Min > k.Default || k.Default > k.Max {
			t.Errorf("%s: default %v outside [%v, %v]", k.Key, k.Default, k.Min, k.Max)
		}
		if k.Label == "" || k.Desc == "" || k.Unit == "" {
			t.Errorf("%s: incomplete metadata", k.Key)
		}
	}
}

func TestDefaultTuningMatchesKnobDefaults(t *testing.T) {
	tu := env.DefaultTuning()
	for _, k := range env.Knobs() {
		if v, err := GetKnob(&tu, k); err == nil && v != k.Default {
			t.Errorf("%s: field %v (%v), knob default %v", k.Key, v, err, k.Default)
		}
	}
}

// A handicap profile must never leave a knob outside its documented range, and never be better than the default.
func TestProfilesStayInRangeAndNeverBeatTheDefault(t *testing.T) {
	for _, p := range Profiles() {
		tu := env.DefaultTuning()
		p.Apply(&tu)
		for _, k := range env.Knobs() {
			v, err := GetKnob(&tu, k)
			if err != nil {
				continue // unresolved knobs are warned about by TestEveryKnobResolvesToAField
			}
			if v < k.Min || v > k.Max {
				t.Errorf("%s: %s = %v outside [%v, %v]", p.Name, k.Key, v, k.Min, k.Max)
			}
			if k.HigherIsBetter && v > k.Default || !k.HigherIsBetter && v < k.Default {
				t.Errorf("%s: %s = %v is better than the default %v", p.Name, k.Key, v, k.Default)
			}
		}
	}
}

func TestProfilesAreMonotonic(t *testing.T) {
	prev := map[string]float64{}
	for i, p := range Profiles() {
		tu := env.DefaultTuning()
		p.Apply(&tu)
		for _, k := range env.Knobs() {
			v, _ := GetKnob(&tu, k)
			if i > 0 {
				worse := v - prev[k.Key]
				if k.HigherIsBetter {
					worse = -worse
				}
				if worse < 0 {
					t.Errorf("%s: %s got better than the previous profile", p.Name, k.Key)
				}
			}
			prev[k.Key] = v
		}
	}
}

func TestOptimalIsTheDefaultAndRookieIsTheWorstEnd(t *testing.T) {
	def := env.DefaultTuning()
	opt := env.DefaultTuning()
	p, _ := ProfileByName("optimal")
	p.Apply(&opt)
	if !reflect.DeepEqual(def, opt) {
		t.Fatal("optimal changed the tuning")
	}
	rk := env.DefaultTuning()
	p, _ = ProfileByName("ROOKIE")
	p.Apply(&rk)
	for _, k := range env.Knobs() {
		v, err := GetKnob(&rk, k)
		if err != nil {
			continue
		}
		if k.NoHandicap {
			if v != k.Default {
				t.Errorf("rookie moved no-handicap knob %s to %v", k.Key, v)
			}
			continue
		}
		if v != worstEnd(k) {
			t.Errorf("rookie %s = %v, want the worst end %v", k.Key, v, worstEnd(k))
		}
	}
	if _, err := ProfileByName("nope"); err == nil {
		t.Fatal("unknown profile accepted")
	}
}

func TestPerKnobAndGroupStrength(t *testing.T) {
	p := Profile{Name: "x", Level: 0.5, Per: map[string]float64{"bot.": 0, "bot.reactionMs": 1}}
	tu := env.DefaultTuning()
	p.Apply(&tu)
	if tu.Bot.ReactionMs != 300 || tu.Bot.TickMs != 100 || tu.Bot.ClearRadius != 30 {
		t.Fatalf("per-knob strengths ignored: %+v", tu.Bot)
	}
	if p.Strength("bot.tickMs") != 0 || p.Strength("nav.x") != 0.5 || (Profile{Level: 7}).Strength("a.b") != 1 || (Profile{Level: -1}).Strength("a.b") != 0 {
		t.Fatal("strength lookup wrong")
	}
}

func TestOverridesAreClamped(t *testing.T) {
	tu := env.DefaultTuning()
	if err := ApplyOverrides(&tu, map[string]float64{"bot.clearRadius": 5, "bot.reactionMs": 9999, "bot.tickMs": 250}); err != nil {
		t.Fatal(err)
	}
	if tu.Bot.ClearRadius != 12 || tu.Bot.ReactionMs != 300 || tu.Bot.TickMs != 250 {
		t.Fatalf("%+v", tu.Bot)
	}
	if ApplyOverrides(&tu, map[string]float64{"bot.nope": 1}) == nil {
		t.Fatal("unknown knob accepted")
	}
	if err := ApplyOverrides(&tu, map[string]float64{"bot.starveGold": 99}); err != nil {
		t.Fatalf("retired knob must still load: %v", err)
	}
}

func TestConfigBuildTuning(t *testing.T) {
	c, err := ParseConfig([]byte(`{"profile":"casual","tuning":{"bot.clearRadius":20}}`))
	if err != nil {
		t.Fatal(err)
	}
	tu, _, err := c.BuildTuning()
	if err != nil || tu.Bot.ClearRadius != 20 || tu.Bot.WatchMs <= 50 {
		t.Fatalf("%+v %v", tu.Bot, err)
	}
	if _, err := ParseConfig([]byte(`{"profil":"x"}`)); err == nil {
		t.Fatal("typo accepted")
	}
	if _, _, err := (Config{Profile: "zzz"}).BuildTuning(); err == nil {
		t.Fatal("unknown profile accepted")
	}
	c2, _ := ParseConfig([]byte(`{"custom":{"Name":"mine","Level":0.5}}`))
	if p, _ := c2.ResolveProfile(); p.Name != "mine" {
		t.Fatal("custom profile ignored")
	}
}

// Degrading a knob must make the bot less efficient, not suicidal: the safety-relevant knobs keep their survival range.
func TestSurvivalKnobsKeepTheirFloor(t *testing.T) {
	tu := env.DefaultTuning()
	p, _ := ProfileByName("rookie")
	p.Apply(&tu)
	if tu.Bot.WatchMs > 300 || tu.Bot.ReactionMs > 300 || tu.Bot.ResupplyPotions < 2 || tu.Bot.PauseDeathsPerHour > 20 {
		t.Fatalf("rookie left the survival range: %+v", tu.Bot)
	}
}
