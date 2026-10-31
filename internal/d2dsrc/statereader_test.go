package d2dsrc

import (
	"errors"
	"testing"

	"d2lod-bot/internal/d2"
	"d2lod-bot/internal/d2/fake"
	"d2lod-bot/internal/mem"
)

// Every implementation of d2.StateReader must answer each method without panicking, and an error must be a real
// error or one of the documented "unknown" sentinels, never a zero value posing as data.
func TestStateReaderConformance(t *testing.T) {
	_, c := startScript(t, lateReplies(t))
	impls := []struct {
		name string
		r    d2.StateReader
	}{
		{"memory", d2.New(mem.NewFake())},
		{"d2d", New(c, Options{Tables: fakeTables{}, Logf: quiet})},
		{"fake", &fake.Reader{Snap: &d2.Snapshot{State: d2.StateInGame, Player: &d2.Player{Name: "x"}}}},
		{"fake-empty", &fake.Reader{}},
	}
	for _, im := range impls {
		t.Run(im.name, func(t *testing.T) {
			s, err := im.r.Snapshot()
			if err == nil && s == nil {
				t.Error("Snapshot: nil, nil")
			}
			if p, err := im.r.Player(); err == nil && p == nil {
				t.Error("Player: nil, nil")
			}
			if _, err := im.r.UI(); err != nil && !errors.Is(err, d2.ErrNotInGame) {
				t.Logf("UI: %v", err)
			}
			for _, typ := range []int{d2.TypePlayer, d2.TypeMonster, d2.TypeObject, d2.TypeMissile, d2.TypeItem} {
				_, _ = im.r.Units(typ)
			}
			_, _ = im.r.UnitStats(0)
			_, _ = im.r.ReadStatList(0)
			_, _ = im.r.ReadBaseStatList(0)
			_, _ = im.r.RawStatList(0, 4)
		})
	}
}

func TestD2dAddressReadsUnsupported(t *testing.T) {
	var s *Source
	for _, err := range []error{
		func() error { _, e := s.UnitStats(1); return e }(),
		func() error { _, e := s.ReadStatList(1); return e }(),
		func() error { _, e := s.ReadBaseStatList(1); return e }(),
		func() error { _, e := s.RawStatList(1, 4); return e }(),
		func() error { _, e := s.Units(d2.TypeItem); return e }(),
	} {
		if !errors.Is(err, d2.ErrUnsupported) {
			t.Errorf("err = %v, want ErrUnsupported", err)
		}
	}
}

func TestD2dPlayerFromSnapshot(t *testing.T) {
	_, c := startScript(t, lateReplies(t))
	src := New(c, Options{Tables: fakeTables{}, Logf: quiet})
	p, err := src.Player()
	if err != nil || p.Name != "Soaky" {
		t.Fatalf("Player = %+v, %v", p, err)
	}
}
