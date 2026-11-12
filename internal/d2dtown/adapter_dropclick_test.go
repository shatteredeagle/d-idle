package d2dtown

import (
	"testing"

	"d2lod-bot/internal/input"
)

type dropRec struct {
	input.Controller
	walked, dropped int
}

func (d *dropRec) ClickWorld(input.Point) error { d.walked++; return nil }
func (d *dropRec) DropClick(input.Point) error  { d.dropped++; return nil }

type plainRec struct {
	input.Controller
	walked int
}

func (p *plainRec) ClickWorld(input.Point) error { p.walked++; return nil }

// The hover wrapper must not hide DropClick: town.Hand finds it through input.WorldDropper on the town's controller.
func TestTownInputKeepsDropClick(t *testing.T) {
	d := &dropRec{}
	var in input.Controller = Input(d, nil)
	wd, ok := in.(input.WorldDropper)
	if !ok {
		t.Fatal("the town input does not offer DropClick")
	}
	if err := wd.DropClick(input.Point{X: 1, Y: 2}); err != nil || d.dropped != 1 || d.walked != 0 {
		t.Fatalf("dropped %d walked %d err %v", d.dropped, d.walked, err)
	}
	p := &plainRec{}
	if err := Input(p, nil).(input.WorldDropper).DropClick(input.Point{}); err != nil || p.walked != 1 {
		t.Fatalf("fallback: walked %d err %v", p.walked, err)
	}
}
