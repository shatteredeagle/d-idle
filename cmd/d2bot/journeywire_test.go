package main

import (
	"testing"
	"time"
)

func TestJourneyPaceMeterAndETA(t *testing.T) {
	now := time.Date(2026, 11, 11, 12, 0, 0, 0, time.UTC)
	exp := uint32(72144) // clvl 11 exactly
	pace := journeyPace(func() (int, uint32, bool) { return 11, exp, true }, func() time.Time { return now })
	if pct, eta, ok := pace(13); !ok || pct != 0 || eta != 0 {
		t.Fatalf("start: %v %v %v", pct, eta, ok)
	}
	now, exp = now.Add(10*time.Minute), 90180 // reached clvl 12 worth of XP in 10 min
	pct, eta, ok := pace(13)
	if !ok || pct < 0.44 || pct > 0.45 || eta < 12*time.Minute || eta > 13*time.Minute {
		t.Errorf("after 10 min: %v %v %v", pct, eta, ok)
	}
	if _, _, ok := pace(100); ok {
		t.Error("level 100 has no table row")
	}
}
