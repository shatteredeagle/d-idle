package combat

import (
	"context"
	"strings"
	"testing"
	"time"

	"d2lod-bot/internal/d2"
)

type runOut struct {
	keys  []string
	moves int
	casts int
	end   time.Time
}

func castLogRun(off bool) (runOut, string) {
	h := novaHarness(nil)
	h.f.clog.off = off
	for i := 0; i < 3; i++ {
		h.s.addMon(uint32(10+i), clsFallen, 101+float64(i), 101, 60, 1, 0)
	}
	buf := logTo(h)
	h.f.Clear(context.Background(), ClearOpts{Timeout: 6 * time.Second})
	return runOut{append([]string(nil), h.s.castKeys...), len(h.s.moves), h.s.casts, h.f.now()}, buf.String()
}

// INVARIANT: the cast log never changes what the bot does: same keys, moves, casts and virtual time.
func TestCastLogDoesNotChangeActions(t *testing.T) {
	on, logOn := castLogRun(false)
	off, logOff := castLogRun(true)
	if strings.Join(on.keys, ",") != strings.Join(off.keys, ",") || on.moves != off.moves || on.casts != off.casts || !on.end.Equal(off.end) {
		t.Fatalf("on %+v\noff %+v", on, off)
	}
	if len(on.keys) == 0 {
		t.Fatal("no casts")
	}
	if !strings.Contains(logOn, "msg=build") || !strings.Contains(logOn, `msg="cast result"`) || !strings.Contains(logOn, "last_skill=") {
		t.Errorf("log lines missing:\n%s", logOn)
	}
	if strings.Contains(logOff, "cast result") || strings.Contains(logOff, "last_skill") {
		t.Errorf("off still logs:\n%s", logOff)
	}
}

func TestArmorState(t *testing.T) {
	p := &d2.Player{}
	if ArmorState(p) != "?" || ArmorState(nil) != "?" {
		t.Fatal("empty state array is unreadable")
	}
	p.States[0] = 1 << 2 // poison only
	if ArmorState(p) != "none" {
		t.Fatalf("got %s", ArmorState(p))
	}
	p.States[d2.StateShiverArmor/32] |= 1 << (d2.StateShiverArmor % 32)
	if ArmorState(p) != "Shiver Armor" {
		t.Fatalf("got %s", ArmorState(p))
	}
}

func TestCastTicksLogged(t *testing.T) {
	// the Lightning family reads 19 on game.exe and 13 on d2d (castgap.go)
	if castTicks("Chain Lightning", 0, false) != 19 || castTicks("Chain Lightning", 0, true) != 13 || castTicks("Fire Bolt", 0, true) != 13 {
		t.Fatal("cast ticks table changed")
	}
}
