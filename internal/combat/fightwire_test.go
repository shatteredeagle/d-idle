package combat

import (
	"bytes"
	"context"
	"log/slog"
	"strings"
	"testing"
)

func TestFightEmitsTelemetryLine(t *testing.T) {
	h := newHarness(t)
	var buf bytes.Buffer
	h.f.e.Log = slog.New(slog.NewTextHandler(&buf, nil))
	h.s.stock(itHP1, 0, 2)
	h.s.addMon(10, clsFallen, 120, 100, 30, 5, 5)
	if _, err := h.f.Clear(context.Background(), ClearOpts{}); err != nil {
		t.Fatal(err)
	}
	out := buf.String()
	if n := strings.Count(out, "fight area="); n != 1 || !strings.Contains(out, "kills=1") {
		t.Fatalf("want one fight line with kills=1:\n%s", out)
	}
}

func TestSafetyCurseRetreatDrinks(t *testing.T) {
	h := newHarness(t)
	h.s.stock(itHP1, 0, 2)
	h.s.addMon(10, clsFallen, 110, 100, 30, 0, 0)
	snap, _ := h.s.snapshot()
	p := snap.Player
	p.MaxLife, p.Life = 100, 55 // above the plain drink line, below the curse line
	id := curseIDs.Amplify
	p.States[id>>5] |= 1 << (uint(id) & 31)
	if a := h.f.Safety(snap); a != DrinkLife {
		t.Fatalf("cursed at 55%%: %v", a)
	}
	p.States = [6]uint32{}
	if a := h.f.Safety(snap); a != SafetyNone {
		t.Fatalf("uncursed at 55%%: %v", a)
	}
}
