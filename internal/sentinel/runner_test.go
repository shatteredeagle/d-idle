package sentinel

import (
	"context"
	"os"
	"path/filepath"
	"testing"
	"time"
)

type fakeActor struct{ calls []Action }

func (f *fakeActor) Do(_ context.Context, _ InstanceConf, a Action, _ Decision) Result {
	f.calls = append(f.calls, a)
	return Result{OK: true, Verified: true}
}

func drive(r *Runner, secs int, mut func(*Observation)) {
	in := r.Cfg.Instances[0]
	for i := 0; i <= secs; i++ {
		o := healthy()
		o.Time = t0.Add(time.Duration(i) * time.Second)
		o.CPUTicks, o.FrameHash = uint64(i), string(rune('a'+i%9))
		o.X = 50 + i%2*10
		mut(&o)
		r.Now = func() time.Time { return o.Time }
		r.Tick(context.Background(), in, o)
	}
}

func newRunner(m Mode) (*Runner, *fakeActor, *[]Record) {
	cfg := DefaultConfig()
	cfg.Instances = []InstanceConf{{ID: "i5", Character: "Multi"}}
	fa := &fakeActor{}
	r := NewRunner(cfg, m, fa, nil)
	var recs []Record
	r.LogW = func(rec Record) { recs = append(recs, rec) }
	r.Engine.State("i5")
	r.status["i5"] = &InstStatus{ID: "i5"}
	return r, fa, &recs
}

func TestObserveOnlyNeverActs(t *testing.T) {
	r, fa, recs := newRunner(Mode{})
	drive(r, 30, func(o *Observation) { o.Life = 0 })
	if len(fa.calls) != 0 {
		t.Fatalf("acted in observe-only: %v", fa.calls)
	}
	n := 0
	for _, rec := range *recs {
		if rec.Kind == "would" && rec.Rule == "death-overlay" {
			n++
		}
	}
	if n == 0 {
		t.Fatal("no would-act record")
	}
	if s := r.Snapshot(); s.Mode != "observe-only" || len(s.Recent) == 0 {
		t.Fatalf("%+v", s)
	}
}

func TestActRulesSelects(t *testing.T) {
	r, fa, recs := newRunner(Mode{ActRules: map[string]bool{"esc-menu-stuck": true}})
	drive(r, 30, func(o *Observation) { o.EscMenu = true })
	if len(fa.calls) != 1 || fa.calls[0] != ActEsc {
		t.Fatalf("%v", fa.calls)
	}
	drive2, _, recs2 := newRunner(Mode{ActRules: map[string]bool{"esc-menu-stuck": true}})
	drive(drive2, 30, func(o *Observation) { o.Life = 0 })
	for _, rec := range *recs2 {
		if rec.Kind == "acted" {
			t.Fatal("death rule acted though not selected")
		}
	}
	found := false
	for _, rec := range *recs {
		if rec.Kind == "acted" && rec.Result != nil && rec.Result.Verified {
			found = true
		}
	}
	if !found {
		t.Fatal("no acted record")
	}
}

func TestCharactersFromInstancesConf(t *testing.T) {
	dir := t.TempDir()
	f := filepath.Join(dir, "instances.conf")
	os.WriteFile(f, []byte("# agents\nnav i1\n#character i1 Palli paladin\n#character i7 Zed Necromancer\nbogus line\n"), 0o644)
	c := DefaultConfig()
	c.InstancesConf = f
	c.WithInstances([]string{"i1", "i2", "i7"})
	got := map[string][2]string{}
	for _, in := range c.Instances {
		got[in.ID] = [2]string{in.Character, in.Class}
	}
	if got["i1"] != [2]string{"Palli", "paladin"} || got["i2"] != [2]string{"Combi", "sorceress"} || got["i7"] != [2]string{"Zed", "necromancer"} {
		t.Fatalf("%v", got)
	}
	// a character named in the JSON config wins
	c = DefaultConfig()
	c.InstancesConf = f
	c.Instances = []InstanceConf{{ID: "i1", Character: "Keep"}}
	c.WithInstances(nil)
	if c.Instances[0].Character != "Keep" {
		t.Fatalf("%+v", c.Instances[0])
	}
	if len(CharactersFromConf(filepath.Join(dir, "missing"))) != 0 {
		t.Fatal("missing file must give an empty map")
	}
}
