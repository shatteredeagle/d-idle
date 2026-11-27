package arbiter

import (
	"strings"
	"testing"
)

// A log with the bot's tick lines: the minute lines (numeric area, lagging counters) are skipped and vtMs is the clock.
func TestReplayPrefersTickLines(t *testing.T) {
	log := strings.Join([]string{
		`{"time":"2026-11-26T01:00:00Z","msg":"minute","area":3,"kills":0,"xp":0,"potions":9,"gold":1}`,
		`{"time":"2026-11-26T01:00:00Z","msg":"tick","vtMs":1000,"area":"Cold Plains","kills":1,"xp":10,"chain":"level","facts":{"potions":4,"potionsMin":2}}`,
		`{"time":"2026-11-26T01:00:00Z","msg":"other","area":"Dark Wood"}`,
		`{"time":"2026-11-26T01:00:00Z","msg":"tick","vtMs":2000,"area":"Cold Plains","kills":2,"xp":20,"chain":"level"}`,
		`{"time":"2026-11-26T01:01:00Z","msg":"minute","area":4,"kills":2,"xp":20}`,
		`{"time":"2026-11-26T01:00:00Z","msg":"tick","vtMs":3000,"area":"Cold Plains","kills":3,"xp":30,"chain":"level"}`,
	}, "\n")
	res, err := Replay(strings.NewReader(log), DefaultConfig(), ShadowConfig{LogGapSec: 60}, true)
	if err != nil {
		t.Fatal(err)
	}
	if res.Ticks != 3 || res.AreaChanges != 0 {
		t.Fatalf("ticks %d area changes %d, want 3 and 0 (minute lines must be skipped)", res.Ticks, res.AreaChanges)
	}
}

// Without any tick line a minute-only log replays as before.
func TestReplayMinuteOnly(t *testing.T) {
	log := `{"time":"2026-11-26T01:00:00Z","msg":"minute","area":3}` + "\n" + `{"time":"2026-11-26T01:01:00Z","msg":"minute","area":4}`
	res, err := Replay(strings.NewReader(log), DefaultConfig(), ShadowConfig{LogGapSec: 60}, false)
	if err != nil || res.Ticks != 2 || res.AreaChanges != 1 {
		t.Fatalf("res %+v err %v", res, err)
	}
}
