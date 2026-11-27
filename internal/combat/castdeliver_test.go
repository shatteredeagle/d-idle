package combat

import (
	"testing"
	"time"

	"d2lod-bot/internal/d2"
	"d2lod-bot/internal/delivery"
	"d2lod-bot/internal/input"
)

// deliverRec records how one cast left the Fighter: through Deliver, or through the old calls.
type deliverRec struct {
	input.Controller
	on                    bool
	uses                  []delivery.Use
	holds                 []time.Duration
	swings, casts, clicks int
}

func (d *deliverRec) Delivers() bool { return d.on }
func (d *deliverRec) Deliver(id int, use *delivery.Use, at input.Point, hold time.Duration) error {
	d.uses, d.holds = append(d.uses, *use), append(d.holds, hold)
	return nil
}
func (d *deliverRec) Swing(int, input.Point) error                     { d.swings++; return nil }
func (d *deliverRec) CastAt(string, input.Point) error                 { d.casts++; return nil }
func (d *deliverRec) Click(input.Button, int, int, ...input.Mod) error { d.clicks++; return nil }

// Two builds: a Sorceress bolt (tap, right hand, bound key) and the Paladin's Zeal (repeat, left hand) and Smite (tap,
// left hand). Knob on: exactly one Deliver per cast with the row's use and a hold only for a repeat press, no old call.
// Knob off: no Deliver, exactly one old call, as before.
func TestFighterCastsThroughDeliver(t *testing.T) {
	type build struct {
		name   string
		skills []string
		setup  func(*testing.T) (*harness, func() (*d2.Snapshot, error))
	}
	builds := []build{
		{"sorceress", []string{"Fire Bolt"}, func(t *testing.T) (*harness, func() (*d2.Snapshot, error)) {
			h := newHarness(t)
			return h, h.s.snapshot
		}},
		{"paladin", []string{"Zeal", "Smite"}, func(t *testing.T) (*harness, func() (*d2.Snapshot, error)) {
			h, g := newHandGame(t)
			return h, g.snap
		}},
	}
	for _, b := range builds {
		for _, on := range []bool{false, true} {
			h, snapFn := b.setup(t)
			snap, err := snapFn()
			if err != nil {
				t.Fatal(err)
			}
			known := KnownSkills(snap.Player, h.f.cat, h.f.keys)
			for _, name := range b.skills {
				sp, ok := known[name]
				if !ok {
					t.Fatalf("%s: %s not known", b.name, name)
				}
				rec := &deliverRec{Controller: h.f.e.In, on: on}
				h.f.e.In = rec
				if err := h.f.cast(snap, sp, input.Point{X: snap.Player.X + 3, Y: snap.Player.Y}); err != nil {
					t.Fatalf("%s %s on=%v: %v", b.name, name, on, err)
				}
				h.f.e.In = rec.Controller
				old := rec.swings + rec.casts + rec.clicks
				if !on {
					if len(rec.uses) != 0 || old != 1 {
						t.Fatalf("%s %s knob off: delivers %d, old calls %d", b.name, name, len(rec.uses), old)
					}
					continue
				}
				row := useRow(name)
				if len(rec.uses) != 1 || old != 0 || rec.uses[0] != *row {
					t.Fatalf("%s %s knob on: uses %+v (row %+v), old calls %d", b.name, name, rec.uses, row, old)
				}
				if hold := rec.holds[0]; (row.Press != "tap") != (hold > 0) {
					t.Fatalf("%s %s: press %s, hold %v", b.name, name, row.Press, hold)
				}
				if (row.Hand == "left") != (sp.Hand == HandLeft) {
					t.Fatalf("%s %s: row hand %q, spec hand %v", b.name, name, row.Hand, sp.Hand)
				}
			}
		}
	}
}

// A skill with no row in the capability table keeps the old calls even with the knob on.
func TestFighterDeliverNeedsARow(t *testing.T) {
	h := newHarness(t)
	snap, _ := h.s.snapshot()
	sp := KnownSkills(snap.Player, h.f.cat, h.f.keys)["Fire Bolt"]
	sp.Name = "no such skill"
	rec := &deliverRec{Controller: h.f.e.In, on: true}
	h.f.e.In = rec
	if err := h.f.cast(snap, sp, input.Point{X: snap.Player.X + 3, Y: snap.Player.Y}); err != nil {
		t.Fatal(err)
	}
	if len(rec.uses) != 0 || rec.casts != 1 {
		t.Fatalf("delivers %d, casts %d", len(rec.uses), rec.casts)
	}
}
