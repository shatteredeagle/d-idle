package bot

import (
	"context"
	"testing"
	"time"

	"d2lod-bot/internal/d2"
)

type beltFillFake struct {
	*fakeTown
	needs bool
	calls int
}

func (b *beltFillFake) NeedsBeltFill(*d2.Snapshot) bool { return b.needs }
func (b *beltFillFake) FillBelt(context.Context) (int, error) {
	b.calls++
	return 2, nil
}

func TestBeltFillStep(t *testing.T) {
	rg := newRig(t)
	bf := &beltFillFake{fakeTown: rg.town, needs: true}
	rg.r.d.Town = bf
	r := rg.r
	s := townSnap(2, 100)
	s.Inventory = &d2.Inventory{}
	if !r.beltFillStep(context.Background(), s) || bf.calls != 1 {
		t.Fatalf("no fill: calls %d", bf.calls)
	}
	if r.beltFillStep(context.Background(), s) || bf.calls != 1 {
		t.Error("filled again inside the backoff")
	}
	r.e.Clock.Sleep(beltFillBackoff + time.Second)
	bf.needs = false
	if r.beltFillStep(context.Background(), s) || bf.calls != 1 {
		t.Error("filled without a need")
	}
}
