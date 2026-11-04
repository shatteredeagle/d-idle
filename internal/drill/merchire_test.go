package drill

import (
	"context"
	"testing"
	"time"
)

type hireGame struct {
	fakeGame
	merc  MercInfo
	quote int
	code  string
	want  string
}

func (h *hireGame) MercState(context.Context) (MercInfo, error) { return h.merc, nil }
func (h *hireGame) PaidHire(context.Context, *Run) (HireInfo, error) {
	if h.gold < h.quote {
		return HireInfo{Quote: h.quote}, ErrHireCantAfford
	}
	h.gold -= h.quote
	h.merc = MercInfo{Present: true, Code: h.code, Life: 40, Max: 40}
	return HireInfo{Quote: h.quote, Row: "row 0", WantCode: h.want}, nil
}

func run8h(h *hireGame, i int) Outcome {
	h.w = &fakeWorld{}
	h.fakeGame.GameLab = h // MercState/PaidHire come from hireGame itself
	return D8h(h, time.Minute)[i].Do(context.Background(), &Run{W: h.w, Drill: "D8.h"})
}

func TestD8hPaid(t *testing.T) {
	if o := run8h(&hireGame{quote: 450, code: "X1", want: "X1"}, 0); o.Verdict != Pass {
		t.Fatalf("%v %s", o.Verdict, o.Reason)
	}
	if o := run8h(&hireGame{quote: 450, code: "X1", want: "X2"}, 0); o.Verdict != Fail {
		t.Fatalf("wrong code passed: %v", o.Verdict)
	}
}

func TestD8hCantAfford(t *testing.T) {
	if o := run8h(&hireGame{quote: 450, code: "X1"}, 1); o.Verdict != Pass {
		t.Fatalf("%v %s", o.Verdict, o.Reason)
	}
}

func TestD8hOwnedBlocked(t *testing.T) {
	if o := run8h(&hireGame{quote: 450, merc: MercInfo{Present: true, Code: "X1"}}, 0); o.Verdict != Blocked {
		t.Fatalf("%v", o.Verdict)
	}
}

func TestD8hNoLab(t *testing.T) {
	if d := D8h(&fakeGame{}, time.Minute); len(d) != 1 || len(d[0].Needs) == 0 {
		t.Fatal("expected a blocked drill")
	}
}
