package main

import (
	"encoding/binary"
	"os"
	"path/filepath"
	"testing"

	"d2lod-bot/internal/cheat/d2s"
)

// clvl 5 starts at 7875 (U table): in-level XP rises 100 -> 3000 (counter +3000), the game dies and falls back to the
// save at in-level 500, then the same 2500 XP is earned again. The counter says 5500 gained, the character 3000.
func lossRun() []Sample {
	cx := []int64{100, 1000, 2000, 3000, 500, 1500, 2500, 3000}
	var xp int64
	return synth(len(cx), func(i int, s *Sample) {
		s.Level, s.CharXP = 5, cx[i]
		if i > 0 {
			xp += max(cx[i]-cx[i-1], 0)
		}
		s.XP = xp
	})
}

func TestXPFromMemoryIsMonotoneAndFlagsLoss(t *testing.T) {
	r := Analyze(lossRun(), DefaultLimits())
	if r.XPSource != "memory" || r.XP != 2900 || r.XPCounter != 5400 {
		t.Fatalf("source %q xp %d counter %d, want memory 2900 5400", r.XPSource, r.XP, r.XPCounter)
	}
	if r.XPLost != 2500 || r.XPLostEvents != 1 {
		t.Fatalf("lost %d in %d events, want 2500 in 1", r.XPLost, r.XPLostEvents)
	}
	found := false
	for _, f := range r.Findings {
		found = found || f.Code == "xp_lost"
	}
	if !found {
		t.Fatalf("no xp_lost finding: %+v", r.Findings)
	}
}

func TestXPFromSaveWins(t *testing.T) {
	ss := lossRun()
	for i := range ss {
		ss[i].SaveXP = 9000 + int64(i/4)*2000 // two saves: 9000, then 11000
	}
	r := Analyze(ss, DefaultLimits())
	if r.XPSource != "save" || r.XP != 2000 {
		t.Fatalf("source %q xp %d, want save 2000", r.XPSource, r.XP)
	}
}

func TestNoCharXPFallsBackToCounter(t *testing.T) {
	r := Analyze(synth(10, nil), DefaultLimits())
	if r.XPSource != "counter" || r.XPLostEvents != 0 {
		t.Fatalf("source %q lost %d", r.XPSource, r.XPLostEvents)
	}
}

func TestXPAtRiskAlert(t *testing.T) {
	ss := synth(4, func(i int, s *Sample) { s.Level, s.CharXP, s.SaveXP = 5, 3000, 7875+100 })
	if a := EvalAlerts(ss, DefaultLimits(), DefaultAlertRules(), ss[3].T); hasAlert(a, "xp_at_risk") {
		t.Fatalf("2900 unsaved at clvl 5 (worth 6300) must not alert: %+v", a.Alerts)
	}
	ss = synth(4, func(i int, s *Sample) { s.Level, s.CharXP, s.SaveXP = 5, 6000, 7875 })
	if a := EvalAlerts(ss, DefaultLimits(), DefaultAlertRules(), ss[3].T); hasAlert(a, "xp_at_risk") {
		t.Fatalf("6000 unsaved of 6300 must not alert")
	}
	ss = synth(4, func(i int, s *Sample) { s.Level, s.CharXP, s.SaveXP = 5, 6300+10, 7875 })
	if a := EvalAlerts(ss, DefaultLimits(), DefaultAlertRules(), ss[3].T); !hasAlert(a, "xp_at_risk") {
		t.Fatalf("a level of unsaved XP must alert: %+v", a.Alerts)
	}
}

func hasAlert(a Alerts, code string) bool {
	for _, x := range a.Alerts {
		if x.Code == code {
			return true
		}
	}
	return false
}

func TestReadSave(t *testing.T) {
	h := make([]byte, 0x2FD)
	binary.LittleEndian.PutUint32(h[0:], d2s.Signature)
	binary.LittleEndian.PutUint32(h[4:], d2s.Version)
	h[0x2B] = 6
	copy(h[0x14F:], "Woo!")
	copy(h[0x279:], "WS")
	copy(h[0x2CA:], "w4")
	b := append(h, 'g', 'f', 0xFF, 0x01)
	b = append(b, 'i', 'f')
	b = append(b, make([]byte, 30)...)
	b = append(b, 'J', 'M', 0, 0)
	binary.LittleEndian.PutUint32(b[8:], uint32(len(b)))
	binary.LittleEndian.PutUint32(b[0xC:], d2s.Checksum(b))
	f, err := d2s.Parse(b)
	if err != nil {
		t.Fatal(err)
	}
	f.SetStat(d2s.StatExp, 23300)
	p := filepath.Join(t.TempDir(), "Soaky.d2s")
	os.WriteFile(p, f.Bytes(), 0o600)
	var s Sample
	readSave(p, &s)
	if s.SaveXP != 23300 || s.SaveLevel != 6 {
		t.Fatalf("save read %d lvl %d", s.SaveXP, s.SaveLevel)
	}
	var none Sample
	readSave(filepath.Join(t.TempDir(), "missing"), &none)
	readSave("", &none)
	if none.SaveXP != 0 {
		t.Fatal("missing file must leave the sample empty")
	}
}
