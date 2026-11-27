package bot

import (
	"bytes"
	"errors"
	"log/slog"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"d2lod-bot/internal/api"
	"d2lod-bot/internal/env"
)

func f64(v float64) *float64 { return &v }

func TestTunerEditProfileOverridesAndPersist(t *testing.T) {
	tune := env.DefaultTuning()
	path := filepath.Join(t.TempDir(), "state", "tuning.json")
	tn, err := NewTuner(&tune, Config{}, path)
	if err != nil {
		t.Fatal(err)
	}
	k := env.Knobs()[0]
	snap := tn.Snapshot()
	if snap.Profile != "optimal" || len(snap.Knobs) != len(env.Knobs()) || snap.Values[k.Key] != k.Default || len(snap.Profiles) < 2 {
		t.Fatalf("snapshot %+v", snap.Profile)
	}
	mid := (k.Min + k.Max) / 2
	if err := tn.Edit(api.TuneEdit{Overrides: map[string]*float64{k.Key: f64(mid)}}); err != nil {
		t.Fatal(err)
	}
	if got := tn.Snapshot().Values[k.Key]; got != mid {
		t.Fatalf("override not applied: %v != %v", got, mid)
	}
	if err := tn.Edit(api.TuneEdit{Overrides: map[string]*float64{"nope.knob": f64(1)}}); !errors.Is(err, api.ErrInvalidTune) {
		t.Fatalf("unknown knob: %v", err)
	}
	if err := tn.Edit(api.TuneEdit{Profile: ptr("bogus")}); !errors.Is(err, api.ErrInvalidTune) {
		t.Fatalf("unknown profile: %v", err)
	}
	if tn.Snapshot().Values[k.Key] != mid {
		t.Fatal("a rejected edit changed the tuning")
	}
	// a new Tuner on the same file restores the edit
	tune2 := env.DefaultTuning()
	tn2, err := NewTuner(&tune2, Config{}, path)
	if err != nil {
		t.Fatal(err)
	}
	if tn2.Snapshot().Values[k.Key] != mid {
		t.Fatal("edit not restored from disk")
	}
	// null resets to the profile value; a profile switch moves the knobs
	if err := tn2.Edit(api.TuneEdit{Overrides: map[string]*float64{k.Key: nil}, Profile: ptr("rookie")}); err != nil {
		t.Fatal(err)
	}
	if tn2.Snapshot().Profile != "rookie" {
		t.Fatal("profile not switched")
	}
	if err := tn2.Unlock("x"); !errors.Is(err, api.ErrNoUpgrades) {
		t.Fatal(err)
	}
}

func ptr(s string) *string { return &s }

// A tuning rebuilt from disk or by an edit keeps what the command line forced (--ladder), and without a hook the
// saved file stays authoritative (T0 live drill 2026-11-05, defect 2).
func TestTunerPostSurvivesReloadAndEdit(t *testing.T) {
	path := filepath.Join(t.TempDir(), "state", "tuning.json")
	seed := env.DefaultTuning()
	tn, err := NewTuner(&seed, Config{}, path)
	if err != nil {
		t.Fatal(err)
	}
	if err := tn.Edit(api.TuneEdit{Overrides: map[string]*float64{"ladder.enabled": f64(0), "combat.fightRadius": f64(20)}}); err != nil {
		t.Fatal(err)
	}
	force := func(x *env.Tuning) { x.Ladder.Enabled = 1 }
	for _, prof := range []string{"optimal", "rookie"} {
		on := env.DefaultTuning()
		tn, err := NewTuner(&on, Config{Profile: prof}, path, force)
		if err != nil {
			t.Fatal(err)
		}
		if on.Ladder.Enabled != 1 {
			t.Fatalf("%s: a saved tuning.json dropped the forced ladder switch", prof)
		}
		if err := tn.Edit(api.TuneEdit{Overrides: map[string]*float64{"combat.fightRadius": f64(25)}}); err != nil {
			t.Fatal(err)
		}
		if on.Ladder.Enabled != 1 || on.Combat.FightRadius != 25 {
			t.Fatalf("%s: an edit dropped the switch or the edit: ladder %d radius %d", prof, on.Ladder.Enabled, on.Combat.FightRadius)
		}
	}
	off := env.DefaultTuning()
	if _, err := NewTuner(&off, Config{}, path); err != nil {
		t.Fatal(err)
	}
	if off.Ladder.Enabled != 0 {
		t.Fatal("without a hook the saved ladder.enabled=0 must stay")
	}
}

// The zone clear settings: a change applies to the live tuning at once (no restart), its source is "dashboard",
// it survives a restart through tuning.json, and a reset goes back to "default".
func TestTunerZoneClearLiveSourcePersist(t *testing.T) {
	path := filepath.Join(t.TempDir(), "tuning.json")
	var tune env.Tuning
	cfg := Config{Tuning: map[string]float64{"bot.exploredPct": 90}}
	tn, err := NewTuner(&tune, cfg, path)
	if err != nil {
		t.Fatal(err)
	}
	if _, src, _ := tn.Value("bot.exploredPct"); src != "config" {
		t.Fatalf("source %q, want config", src)
	}
	if _, src, _ := tn.Value("bot.farmResetPct"); src != "default" {
		t.Fatalf("source %q, want default", src)
	}
	x, on := 75.0, 1.0
	if err := tn.Edit(api.TuneEdit{Overrides: map[string]*float64{"bot.exploredPct": &x, "bot.farmReset": &on}}); err != nil {
		t.Fatal(err)
	}
	if tune.Bot.ExploredPct != 75 || tune.Bot.FarmReset != 1 {
		t.Fatalf("the live tuning did not change: %+v", tune.Bot)
	}
	if v, src, _ := tn.Value("bot.exploredPct"); v != 75 || src != "dashboard" {
		t.Fatalf("%v %q", v, src)
	}
	var tune2 env.Tuning
	tn2, err := NewTuner(&tune2, Config{}, path)
	if err != nil {
		t.Fatal(err)
	}
	if v, src, _ := tn2.Value("bot.exploredPct"); v != 75 || src != "dashboard" || tune2.Bot.FarmReset != 1 {
		t.Fatalf("after a restart: %v %q", v, src)
	}
	if err := tn2.Edit(api.TuneEdit{Overrides: map[string]*float64{"bot.exploredPct": nil}}); err != nil {
		t.Fatal(err)
	}
	if v, src, _ := tn2.Value("bot.exploredPct"); v != 100 || src != "default" {
		t.Fatalf("reset: %v %q", v, src)
	}
	if _, _, err := tn2.Value("bot.nope"); err == nil {
		t.Fatal("an unknown knob must be an error")
	}
}

func TestTunerZeroByteFileRestoresFromBakWithOneWarn(t *testing.T) {
	for _, prof := range []string{"optimal", Profiles()[len(Profiles())-1].Name} {
		t.Run(prof, func(t *testing.T) {
			path := filepath.Join(t.TempDir(), "state", "tuning.json")
			tune := env.DefaultTuning()
			tn, err := NewTuner(&tune, Config{Profile: prof}, path)
			if err != nil {
				t.Fatal(err)
			}
			k := env.Knobs()[0]
			mid := (k.Min + k.Max) / 2
			for _, v := range []float64{mid, mid} { // two saves so .bak holds a good copy
				if err := tn.Edit(api.TuneEdit{Overrides: map[string]*float64{k.Key: f64(v)}}); err != nil {
					t.Fatal(err)
				}
			}
			if err := os.WriteFile(path, nil, 0o644); err != nil { // the Doofer failure
				t.Fatal(err)
			}
			var buf bytes.Buffer
			log := slog.New(slog.NewTextHandler(&buf, nil))
			tune2 := env.DefaultTuning()
			tn2, err := newTuner(log, &tune2, Config{}, path)
			if err != nil {
				t.Fatal(err)
			}
			if v, src, err := tn2.Value(k.Key); err != nil || v != mid || src != "dashboard" {
				t.Fatalf("restored value %v %s %v", v, src, err)
			}
			out := buf.String()
			if strings.Count(out, "level=WARN") != 1 || !strings.Contains(out, "knobsRestored=1") || !strings.Contains(out, "restoredFromBackup=true") {
				t.Fatalf("log %q", out)
			}
		})
	}
}

func TestTunerZeroByteFileNoBackupWarnsKnobsLost(t *testing.T) {
	path := filepath.Join(t.TempDir(), "tuning.json")
	os.WriteFile(path, nil, 0o644)
	var buf bytes.Buffer
	tune := env.DefaultTuning()
	if _, err := newTuner(slog.New(slog.NewTextHandler(&buf, nil)), &tune, Config{}, path); err != nil {
		t.Fatal(err)
	}
	if strings.Count(buf.String(), "level=WARN") != 1 || !strings.Contains(buf.String(), "knobsRestored=0") {
		t.Fatalf("log %q", buf.String())
	}
}
