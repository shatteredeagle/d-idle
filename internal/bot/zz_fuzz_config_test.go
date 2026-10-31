package bot

import (
	"math"

	"d2lod-bot/internal/env"
	"math/rand"
	"os"
	"path/filepath"
	"testing"
)

// loadAll runs a config through every step the bot takes at startup; it must return errors, never panic.
func loadAll(t testing.TB, path string) {
	t.Helper()
	defer func() {
		if x := recover(); x != nil {
			t.Fatalf("panic loading config: %v", x)
		}
	}()
	c, err := LoadConfig(path)
	if err != nil {
		return
	}
	c.BuildTuning()
	c.ResolveProfile()
	c.observes("kill")
	c.WithDefaults()
	_ = sec(c.StuckAfterSec)
}

func writeTmp(t testing.TB, b []byte) string {
	t.Helper()
	p := filepath.Join(t.TempDir(), "cfg.json")
	if err := os.WriteFile(p, b, 0o644); err != nil {
		t.Fatal(err)
	}
	return p
}

var configSeeds = []string{
	``, `{}`, `null`, `[]`, `"x"`, `0`, `{"bot":null}`, `{"bot":[]}`, `{"bot":{}}`, `{"bot":{"profile":"nope"}}`,
	`{"profile":"optimal","acts":[-1,0,99,1],"tuning":{"bot.clearRadius":1e308}}`,
	`{"tuning":{"":1,"x":-1e308,"combat.chickenLifePct":1e999}}`,
	`{"custom":{}}`, `{"custom":{"name":"x","knobs":{"a":1}}}`, `{"custom":null}`, `{"custom":{"Level":2,"Per":{"nav.":-5,"":3,".":1}}}`, `{"custom":{"Level":1e999}}`,
	`{"stuckAfterSec":9223372036854775807,"travelTimeoutSec":-9223372036854775808}`,
	`{"stateStuckSec":{"":0,"clearing":-5},"disableObserve":[null]}`,
	`{"acts":[1,1,1],"deathLogPath":"/nonexistent/\u0000/x"}`,
	`{"bot":{"bot":{}}}`, `{"bot":{"acts":"1"}}`, "\xff\xfe{", `{"a":` + string(make([]byte, 0)) + `}`,
}

func TestConfigGarbageTable(t *testing.T) {
	for _, s := range configSeeds {
		loadAll(t, writeTmp(t, []byte(s)))
	}
	rng := rand.New(rand.NewSource(3))
	good := []byte(`{"bot":{"profile":"optimal","acts":[1,2],"tuning":{"bot.clearRadius":20},"stuckAfterSec":90,"disableObserve":["kill"]}}`)
	// every truncation and 5000 byte-level mutations of a valid file
	for n := 0; n <= len(good); n++ {
		loadAll(t, writeTmp(t, good[:n]))
	}
	for i := 0; i < 3000; i++ {
		b := append([]byte(nil), good...)
		for k := rng.Intn(4); k >= 0; k-- {
			b[rng.Intn(len(b))] = byte(rng.Intn(256))
		}
		loadAll(t, writeTmp(t, b))
	}
	// directory, empty path, missing file
	loadAll(t, "")
	loadAll(t, t.TempDir())
	loadAll(t, filepath.Join(t.TempDir(), "missing.json"))
	// a deeply nested document must not blow the stack
	deep := make([]byte, 0, 200000)
	for i := 0; i < 100000; i++ {
		deep = append(deep, '[')
	}
	loadAll(t, writeTmp(t, deep))
}

func FuzzParseConfig(f *testing.F) {
	for _, s := range configSeeds {
		f.Add([]byte(s))
	}
	f.Fuzz(func(t *testing.T, b []byte) {
		defer func() {
			if x := recover(); x != nil {
				t.Fatalf("panic: %v", x)
			}
		}()
		c, err := ParseConfig(b)
		if err != nil {
			return
		}
		c.BuildTuning()
		c.ResolveProfile()
	})
}

// Extreme and non-finite knob values (a config cannot carry NaN, but a UI or an API might) must clamp, not panic.
func TestKnobExtremes(t *testing.T) {
	vals := []float64{0, -1, 1, 1e308, -1e308, math.Inf(1), math.Inf(-1), math.NaN(), math.MaxInt64, math.SmallestNonzeroFloat64}
	for _, k := range env.Knobs() {
		for _, v := range vals {
			tu := env.DefaultTuning()
			func() {
				defer func() {
					if x := recover(); x != nil {
						t.Errorf("knob %s value %v: panic %v", k.Key, v, x)
					}
				}()
				SetKnob(&tu, k, v)
				GetKnob(&tu, k)
			}()
		}
	}
}
