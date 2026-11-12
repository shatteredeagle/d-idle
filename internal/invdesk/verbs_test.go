package invdesk

import (
	"context"
	"errors"
	"strings"
	"testing"
	"time"

	"d2lod-bot/internal/d2"
)

// fakeVerbs is a backend with (or without) the equip verb. It records what it was asked and can refuse or fail.
type fakeVerbs struct {
	caps      []string
	capsErr   error
	capsCalls int
	equips    []uint32
	err       error // returned by ItemEquip
	noop      bool
}

func (f *fakeVerbs) Caps(context.Context) ([]string, error) { f.capsCalls++; return f.caps, f.capsErr }
func (f *fakeVerbs) ItemEquip(_ context.Context, id uint32, _ int) (bool, error) {
	f.equips = append(f.equips, id)
	return f.noop, f.err
}

type refusal struct{ code string }

func (r refusal) Error() string      { return "item verb refused (" + r.code + ")" }
func (r refusal) RuleRefusal() bool { return true }

func on() bool  { return true }
func off() bool { return false }

// verbRig is the desk rig with the click equipper wrapped by a VerbEquipper.
func verbRig(t *testing.T, class d2.Class, fv *fakeVerbs, knob func() bool) (*Desk, *fakeEq, *VerbEquipper) {
	t.Helper()
	d, eq, _ := rig(t, class, town1)
	ve := NewVerbEquipper(eq, fv, knob)
	d.SetEquipper(ve)
	return d, eq, ve
}

func TestVerbFirstAndClickFallback(t *testing.T) {
	for _, c := range []struct {
		class d2.Class
		good  uint32
	}{{d2.Paladin, 1}, {d2.Sorceress, 2}} {
		t.Run(c.class.String(), func(t *testing.T) {
			// Knob on, capability present: the verb does it, the click path is untouched, the request says how=verb.
			fv := &fakeVerbs{caps: []string{"state.caps", CapItemEquip}}
			d, eq, _ := verbRig(t, c.class, fv, on)
			if _, err := d.RequestEquip(c.good, "rhand"); err != nil {
				t.Fatal(err)
			}
			lines := d.RunPending(context.Background())
			if len(fv.equips) != 1 || fv.equips[0] != c.good || len(eq.worn) != 0 {
				t.Fatalf("verb %v click %v", fv.equips, eq.worn)
			}
			if len(lines) != 1 || !strings.Contains(lines[0], "(how=verb)") || !d.Pinned(d2.BodyRHand) {
				t.Fatalf("lines %v pinned %v", lines, d.Pinned(d2.BodyRHand))
			}

			// Knob off (the default): never asks the backend, even with the capability.
			fv = &fakeVerbs{caps: []string{CapItemEquip}}
			d, eq, _ = verbRig(t, c.class, fv, off)
			d.RequestEquip(c.good, "rhand")
			lines = d.RunPending(context.Background())
			if fv.capsCalls != 0 || len(fv.equips) != 0 || len(eq.worn) != 1 || !strings.Contains(lines[0], "(how=click)") {
				t.Fatalf("knob off: caps %d verb %v click %v %v", fv.capsCalls, fv.equips, eq.worn, lines)
			}

			// Knob on but the backend has no verb (Game.exe, stock d2d, an older fork): click path.
			fv = &fakeVerbs{}
			d, eq, _ = verbRig(t, c.class, fv, on)
			d.RequestEquip(c.good, "rhand")
			lines = d.RunPending(context.Background())
			if len(fv.equips) != 0 || len(eq.worn) != 1 || !strings.Contains(lines[0], "(how=click)") {
				t.Fatalf("no cap: %v %v %v", fv.equips, eq.worn, lines)
			}
		})
	}
}

func TestVerbFailureModes(t *testing.T) {
	// A rule refusal is final: no click retry, the request fails with the reason.
	fv := &fakeVerbs{caps: []string{CapItemEquip}, err: refusal{"twohand"}}
	d, eq, ve := verbRig(t, d2.Paladin, fv, on)
	d.RequestEquip(1, "rhand")
	lines := d.RunPending(context.Background())
	if len(eq.worn) != 0 || len(lines) != 1 || !strings.Contains(lines[0], "failed") || !strings.Contains(lines[0], "twohand") || ve.LastHow() != HowVerb {
		t.Fatalf("refusal: click %v %v how %s", eq.worn, lines, ve.LastHow())
	}
	// A transport failure falls back to the click path and drops the cached capability (a minute later it asks again).
	fv = &fakeVerbs{caps: []string{CapItemEquip}, err: errors.New("d2d: connection reset")}
	now := time.Unix(1000, 0)
	d, eq, ve = verbRig(t, d2.Sorceress, fv, on)
	ve.Now = func() time.Time { return now }
	d.RequestEquip(2, "rhand")
	lines = d.RunPending(context.Background())
	if len(fv.equips) != 1 || len(eq.worn) != 1 || ve.LastHow() != HowClick || !strings.Contains(lines[0], "(how=click)") {
		t.Fatalf("fallback: verb %v click %v %v", fv.equips, eq.worn, lines)
	}
	calls := fv.capsCalls
	if ve.hasCap(context.Background()) || fv.capsCalls != calls {
		t.Fatal("capability should be dropped and not re-read within the minute")
	}
	now = now.Add(2 * time.Minute)
	fv.err = nil
	if !ve.hasCap(context.Background()) || fv.capsCalls != calls+1 {
		t.Fatal("capability should be re-read after the TTL")
	}
	// A caps transport error is not cached and means click.
	fv = &fakeVerbs{capsErr: errors.New("boom")}
	_, eq, ve = verbRig(t, d2.Paladin, fv, on)
	if err := ve.PlayerEquip(context.Background(), 1, d2.BodyRHand); err != nil || len(eq.worn) != 1 || ve.LastHow() != HowClick {
		t.Fatalf("caps error: %v %v", err, eq.worn)
	}
	// Verify: an ok reply that the snapshot does not confirm is an error, never a silent click retry.
	fv = &fakeVerbs{caps: []string{CapItemEquip}}
	_, eq, ve = verbRig(t, d2.Paladin, fv, on)
	ve.Verify = func(context.Context, uint32, int) error { return errors.New("not worn") }
	if err := ve.PlayerEquip(context.Background(), 1, d2.BodyRHand); err == nil || len(eq.worn) != 0 {
		t.Fatalf("verify: %v %v", err, eq.worn)
	}
	// noop (already worn there) needs no verification.
	fv = &fakeVerbs{caps: []string{CapItemEquip}, noop: true}
	_, _, ve = verbRig(t, d2.Paladin, fv, on)
	ve.Verify = func(context.Context, uint32, int) error { return errors.New("must not run") }
	if err := ve.PlayerEquip(context.Background(), 1, d2.BodyRHand); err != nil {
		t.Fatalf("noop: %v", err)
	}
}

func TestVerbEquipperChecksStayOnTheBotSide(t *testing.T) {
	fv := &fakeVerbs{caps: []string{CapItemEquip}}
	d, _, _ := verbRig(t, d2.Sorceress, fv, on)
	var r *Refusal
	if _, err := d.RequestEquip(1, "rhand"); !errors.As(err, &r) || r.Kind() != "requirements" { // a Paladin's sword
		t.Fatalf("pre-check lost: %v", err)
	}
	if len(fv.equips) != 0 {
		t.Fatal("the verb ran for a refused request")
	}
}
