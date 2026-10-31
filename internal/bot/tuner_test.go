package bot

import (
	"errors"
	"path/filepath"
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
