package replay

import (
	"bytes"
	"context"
	"path/filepath"
	"testing"
	"time"

	"d2lod-bot/internal/input"
	"d2lod-bot/internal/mem"
)

func page(v byte) []byte { return bytes.Repeat([]byte{v}, mem.PageSize) }

func TestStreamRoundTripAndSource(t *testing.T) {
	st := &Stream{
		Header: mem.NewHeader(mem.KindStream),
		Frames: []Frame{
			{0, map[uint32][]byte{0x403000: page(1), 0x404000: page(2)}},
			{100 * time.Millisecond, map[uint32][]byte{0x403000: page(3)}},
		},
		Calls: []Call{{50 * time.Millisecond, "key i"}},
	}
	p := filepath.Join(t.TempDir(), "s.d2fx")
	if err := st.Save(p); err != nil {
		t.Fatal(err)
	}
	fx, err := Load(p)
	if err != nil || fx.Stream == nil || fx.Header.Schema != mem.EnvSchema || fx.Header.Kind != mem.KindStream {
		t.Fatalf("%+v %v", fx, err)
	}
	got := fx.Stream
	if len(got.Frames) != 2 || len(got.Calls) != 1 || got.Calls[0].Text != "key i" || got.Duration() != 100*time.Millisecond {
		t.Fatalf("%+v", got)
	}
	s := NewSource(got)
	b, err := mem.U8(s, 0x403010)
	if err != nil || b != 1 {
		t.Fatalf("%d %v", b, err)
	}
	if !s.Step() {
		t.Fatal("step")
	}
	if b, _ := mem.U8(s, 0x403010); b != 3 {
		t.Fatalf("after step %d", b)
	}
	if b, _ := mem.U8(s, 0x404010); b != 2 {
		t.Fatalf("delta must keep unchanged pages, got %d", b)
	}
	if s.Step() {
		t.Fatal("end")
	}
	s.Seek(10 * time.Millisecond) // backwards
	if b, _ := mem.U8(s, 0x403010); b != 1 || s.Time() != 0 {
		t.Fatalf("seek back %d", b)
	}
	if _, err := mem.U8(s, 0x500000); err == nil {
		t.Fatal("want unmapped")
	}
}

type fakeSrc struct{ v byte }

func (f *fakeSrc) ReadAt(p []byte, addr uint32) error {
	if addr&^(mem.PageSize-1) != 0x403000 {
		return mem.ErrUnmapped
	}
	for i := range p {
		p[i] = f.v
	}
	return nil
}

func TestRecorderDeltas(t *testing.T) {
	src := &fakeSrc{v: 1}
	clock := time.Unix(0, 0)
	r := NewRecorder(src, mem.NewHeader(mem.KindStream))
	r.Now = func() time.Time { return clock }
	var b [4]byte
	r.ReadAt(b[:], 0x403010)
	if err := r.ReadAt(b[:], 0x700000); err == nil {
		t.Fatal("want error")
	}
	if n := r.Tick(); n != 1 {
		t.Fatalf("first tick %d", n)
	}
	clock = clock.Add(100 * time.Millisecond)
	if n := r.Tick(); n != 0 {
		t.Fatalf("unchanged tick %d", n)
	}
	src.v = 2
	clock = clock.Add(100 * time.Millisecond)
	in := r.WrapInput(input.NewFake(input.Point{}))
	in.Key("i")
	if n := r.Tick(); n != 1 {
		t.Fatalf("changed tick %d", n)
	}
	st := r.Stream()
	if len(st.Frames) != 2 || st.Frames[1].T != 200*time.Millisecond || len(st.Calls) != 1 || st.Calls[0].T != 200*time.Millisecond {
		t.Fatalf("%+v", st)
	}
	s := NewSource(st)
	s.SeekFrame(2)
	if v, _ := mem.U8(s, 0x403100); v != 2 {
		t.Fatal(v)
	}
	// Run stops at the deadline and records at least one tick.
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Millisecond)
	defer cancel()
	r2 := NewRecorder(src, mem.NewHeader(mem.KindStream))
	calls := 0
	r2.Run(ctx, 10*time.Millisecond, func() { calls++; r2.ReadAt(b[:], 0x403000) })
	if calls == 0 || len(r2.Stream().Frames) == 0 {
		t.Fatal("run recorded nothing")
	}
}

func TestInputSimulatesAndRecords(t *testing.T) {
	in := NewInput(input.Point{X: 1, Y: 1})
	in.ClickWorld(input.Point{X: 5, Y: 6})
	in.Click(input.Right, 10, 20, input.ModShift)
	in.CastAt("F", input.Point{X: 7, Y: 8})
	if p, _ := in.Pos(); p != (input.Point{X: 5, Y: 6}) {
		t.Fatal(p)
	}
	want := []string{"clickworld 5 6", "click R 10 20 shift", "castat F 7 8"}
	got := in.Calls()
	for i := range want {
		if got[i] != want[i] {
			t.Fatalf("%v", got)
		}
	}
	in.Err = context.Canceled
	if err := in.CloseAll(); err != context.Canceled {
		t.Fatal(err)
	}
	if err := in.CloseAll(); err != nil {
		t.Fatal("error must be one-shot")
	}
}

func TestLegacyFixturesLoadUnchanged(t *testing.T) {
	cases := map[string]string{
		"../d2/testdata/town.snap":                mem.KindPages,
		"../d2map/testdata/rogue_encampment.d2mi": mem.KindMapImage,
		"../combat/testdata/death/death1.jsonl":   mem.KindTrace,
	}
	for f, kind := range cases {
		fx, err := Load(f)
		if err != nil {
			t.Fatalf("%s: %v", f, err)
		}
		if fx.Header.Schema != 0 || fx.Header.Kind != kind {
			t.Fatalf("%s: %+v", f, fx.Header)
		}
		if kind == mem.KindTrace && len(fx.Trace) < 10 {
			t.Fatalf("trace records %d", len(fx.Trace))
		}
		if kind != mem.KindTrace && fx.Mem() == nil {
			t.Fatalf("%s: no memory", f)
		}
	}
}

func TestRunGolden(t *testing.T) {
	Run(t, "../d2/testdata/town.snap", func(src *Source, in *Input) {
		if v, err := mem.U32(src, mem.Base+0x3A6A70); err != nil || v == 0 {
			t.Fatalf("player var %x %v", v, err)
		}
	})
}

func TestCheckBuild(t *testing.T) {
	h := mem.NewHeader(mem.KindStream)
	if err := CheckBuild(h); err != nil {
		t.Fatal(err)
	}
	h.Build.Timestamp++
	if CheckBuild(h) == nil {
		t.Fatal("want mismatch")
	}
	if CheckBuild(mem.Header{}) != nil {
		t.Fatal("legacy passes")
	}
}
