package main

import (
	"bytes"
	"context"
	"encoding/json"
	"strings"
	"testing"
	"time"

	"d2lod-bot/internal/d2"
	"d2lod-bot/internal/d2/fake"
	"d2lod-bot/internal/input"
)

// skWorld is a fake reader with one zombie (class 7 in the fake only) whose life drops by dmg[i] on cast i.
type skWorld struct {
	r      *fake.Reader
	in     *input.Fake
	casts  int
	dmg    []float64
	zombie d2.Unit
}

type castCtl struct {
	*input.Fake
	w *skWorld
}

func (c castCtl) CastAt(k string, p input.Point) error {
	w := c.w
	err := c.Fake.CastAt(k, p)
	d := w.dmg[w.casts%len(w.dmg)]
	w.casts++
	st := w.r.Stats[w.zombie.Addr]
	st[d2.StatLife] -= int32(d * 256)
	if st[d2.StatLife] <= 0 {
		st[d2.StatLife] = 0
		w.zombie.Mode = d2.MonModeDeath
		w.r.Snap.Monsters[0] = w.zombie
	}
	return err
}

func newWorld(life float64, dmg ...float64) (*skWorld, skRig) {
	z := d2.Unit{Addr: 0x1000, Type: d2.TypeMonster, Class: 7, ID: 9, Mode: 1, X: 105, Y: 100}
	w := &skWorld{dmg: dmg, zombie: z}
	w.r = &fake.Reader{
		Snap:  &d2.Snapshot{State: d2.StateInGame, Player: &d2.Player{X: 100, Y: 100}, Monsters: []d2.Unit{z}},
		Stats: map[uint32]d2.Stats{z.Addr: {d2.StatLife: int32(life * 256)}},
	}
	w.in = input.NewFake(input.Point{})
	now := time.Unix(0, 0)
	return w, skRig{R: w.r, In: castCtl{w.in, w}, Sleep: func(d time.Duration) { now = now.Add(d) }, Now: func() time.Time { return now }}
}

func TestSKFireBoltLevel4Pass(t *testing.T) {
	w, g := newWorld(10000, 8, 9, 10, 9)
	o := skOpts{Skill: "Fire Bolt", Key: "f1", Level: 4, Hits: 40, Casts: 100, Wait: time.Second, Timeout: time.Hour}
	var out bytes.Buffer
	if err := skCmd(context.Background(), g, o, &out); err != nil {
		t.Fatal(err)
	}
	var r SKResult
	if err := json.Unmarshal(out.Bytes(), &r); err != nil {
		t.Fatalf("%v: %s", err, out.String())
	}
	if r.Hits != 40 || r.Mean != 9 || r.Min != 8 || r.Max != 10 || r.Pass == nil || !*r.Pass {
		t.Fatalf("%+v", r)
	}
	if r.ExpMean != 9 || w.in.Calls[0] != "cast f1 {105 100}" {
		t.Fatalf("exp %v call %q", r.ExpMean, w.in.Calls[0])
	}
}

func TestSKFailsOutOfRangeAndShortRun(t *testing.T) {
	_, g := newWorld(10000, 20) // level 1 expects 3-6
	o := skOpts{Skill: "Fire Bolt", Key: "f1", Level: 1, Hits: 40, Casts: 100, Wait: time.Second, Timeout: time.Hour}
	r, err := g.sample(context.Background(), o)
	if err != nil || r.Pass == nil || *r.Pass || r.OutRange != 40 {
		t.Fatalf("%+v %v", r, err)
	}
	_, g = newWorld(10000, 4.5)
	o.Hits, o.Casts = 5, 5
	r, _ = g.sample(context.Background(), o)
	if r.Pass == nil || *r.Pass || r.Note == "" {
		t.Fatalf("short run must not pass: %+v", r)
	}
}

func TestSKKillIsDiscardedAndNoTarget(t *testing.T) {
	_, g := newWorld(20, 8, 8, 8) // 20 life: two hits, the third kills
	o := skOpts{Skill: "Fire Bolt", Key: "f1", Level: 8, Hits: 10, Casts: 10, Wait: time.Second, Timeout: time.Hour}
	r, _ := g.sample(context.Background(), o)
	if r.Hits != 2 || r.Kills != 1 || !strings.HasPrefix(r.Note, "no live target") {
		t.Fatalf("%+v", r)
	}
	if r.Pass != nil && *r.Pass {
		t.Fatal("pass on two hits")
	}
}

func TestSKFlagsAndGuards(t *testing.T) {
	cls := func(n string) (int, bool) { return 7, n == "zombie1" }
	o, err := parseSK([]string{"--skill", "Charged Bolt", "--key", "f2", "--monster", "zombie1", "--level", "8"}, cls)
	if err != nil || o.Class != 7 || o.Level != 8 || o.Hits != 40 {
		t.Fatalf("%+v %v", o, err)
	}
	if _, err := parseSK([]string{"--skill", "x", "--key", "f2", "--monster", "nope"}, cls); err == nil {
		t.Fatal("unknown monster accepted")
	}
	if _, err := parseSK([]string{"--key", "f2"}, cls); err == nil {
		t.Fatal("missing skill accepted")
	}
	_, g := newWorld(100, 5)
	if _, err := g.sample(context.Background(), skOpts{Skill: "x"}); err == nil {
		t.Fatal("missing key accepted")
	}
}
