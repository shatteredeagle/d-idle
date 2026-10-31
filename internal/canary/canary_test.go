package canary

import (
	"path/filepath"
	"testing"

	"d2lod-bot/internal/mem"
	"d2lod-bot/internal/replay"
)

func find(rs []Result, name string) Result {
	for _, r := range rs {
		if r.Name == name {
			return r
		}
	}
	return Result{}
}

// Every committed memory fixture must satisfy the invariants (rotted offsets show up here).
func TestCommittedFixtures(t *testing.T) {
	var files []string
	for _, pat := range []string{"../d2/testdata/*.snap", "../d2map/testdata/*.d2mi"} {
		m, _ := filepath.Glob(pat)
		files = append(files, m...)
	}
	if len(files) == 0 {
		t.Fatal("no fixtures found")
	}
	for _, f := range files {
		t.Run(filepath.Base(f), func(t *testing.T) {
			fx, err := replay.Load(f)
			if err != nil {
				t.Fatal(err)
			}
			rs := Check(fx.Mem())
			for _, r := range rs {
				t.Log(r)
			}
			if f := Failed(rs); len(f) > 0 {
				t.Fatalf("failed: %v", f)
			}
			if err := replay.CheckBuild(fx.Header); err != nil {
				t.Fatal(err)
			}
		})
	}
	// town.snap is a full in-game capture: the game checks must really run, not skip.
	fx, err := replay.Load("../d2/testdata/town.snap")
	if err != nil {
		t.Fatal(err)
	}
	rs := Check(fx.Mem())
	for _, n := range []string{"menu-or-game", "player-type", "player-area", "inventory-owner", "server-table-chain"} {
		if r := find(rs, n); r.Status != Pass {
			t.Errorf("%s: %v", n, r)
		}
	}
	if r := find(rs, "build"); r.Status != Skip {
		t.Errorf("legacy fixture without header page should skip build, got %v", r)
	}
}

func TestDetectsCorruption(t *testing.T) {
	s, err := mem.LoadSnapshot("../d2/testdata/town.snap")
	if err != nil {
		t.Fatal(err)
	}
	player, _ := mem.U32(s, mem.Base+rvaPlayerUnit)
	// wrong unit type
	s.Put(player, []byte{7, 0, 0, 0})
	rs := Check(s)
	if find(rs, "player-type").Status != Fail {
		t.Fatalf("%v", rs)
	}
	if err := Gate(s); err == nil {
		t.Fatal("gate should fail")
	}
	// both menu and game variables set
	s2, _ := mem.LoadSnapshot("../d2/testdata/town.snap")
	s2.Put(mem.Base+rvaFirstControl, []byte{1, 0, 0, 0})
	if find(Check(s2), "menu-or-game").Status != Fail {
		t.Fatal("want menu-or-game fail")
	}
}

func TestEmptySourceSkips(t *testing.T) {
	rs := Check(mem.NewFake())
	if len(Failed(rs)) != 0 {
		t.Fatalf("%v", rs)
	}
	if find(rs, "build").Status != Skip {
		t.Fatalf("%v", rs)
	}
}

func TestWrongBuildFails(t *testing.T) {
	f := mem.NewFake()
	hdr := make([]byte, 0x400)
	hdr[0], hdr[1] = 'M', 'Z'
	f.Put(mem.Base, hdr)
	if find(Check(f), "build").Status != Fail {
		t.Fatal("bad header must fail")
	}
}

func TestEnvelopedFixtureEqualsLegacy(t *testing.T) {
	_, s, err := mem.LoadSnapshotHeader("../d2/testdata/town.snap")
	if err != nil {
		t.Fatal(err)
	}
	p := filepath.Join(t.TempDir(), "town.d2fx")
	if err := s.SaveEnvelope(p, mem.NewHeader(mem.KindPages)); err != nil {
		t.Fatal(err)
	}
	fx, err := replay.Load(p)
	if err != nil || fx.Header.Schema != mem.EnvSchema {
		t.Fatalf("%+v %v", fx, err)
	}
	legacy, _ := replay.Load("../d2/testdata/town.snap")
	a, b := Check(fx.Mem()), Check(legacy.Mem())
	for i := range a {
		if a[i] != b[i] {
			t.Fatalf("%v != %v", a[i], b[i])
		}
	}
}
