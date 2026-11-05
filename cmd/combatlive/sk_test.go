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

// skWorld is a fake reader with one zombie (class 7 in the fake only) of maxLife points. Like the real client it
// reports life on a 0..128 percent scale; the nth cast deals dmg[n] points. A killed zombie is replaced by a new one.
type skWorld struct {
	r       *fake.Reader
	in      *input.Fake
	casts   int
	dmg     []float64
	maxLife float64
	life    float64
	zombie  d2.Unit
}

type castCtl struct {
	*input.Fake
	w *skWorld
}

func (w *skWorld) setPct() {
	pct := w.life / w.maxLife * 128
	if pct < 0 {
		pct = 0
	}
	w.r.Stats[w.zombie.Addr] = d2.Stats{d2.StatLife: int32(pct * 256)}
}

func (c castCtl) CastAt(k string, p input.Point) error {
	w := c.w
	err := c.Fake.CastAt(k, p)
	w.life -= w.dmg[w.casts%len(w.dmg)]
	w.casts++
	w.setPct()
	if w.life <= 0 {
		w.zombie.Mode = d2.MonModeDeath
		w.r.Snap.Monsters[len(w.r.Snap.Monsters)-1] = w.zombie
		w.zombie.Addr += 0x100
		w.zombie.ID++
		w.zombie.Mode = 1
		w.life = w.maxLife
		w.r.Snap.Monsters = append(w.r.Snap.Monsters, w.zombie)
		w.setPct()
	}
	return err
}

func newWorld(maxLife float64, dmg ...float64) (*skWorld, skRig) {
	z := d2.Unit{Addr: 0x1000, Type: d2.TypeMonster, Class: 7, ID: 9, Mode: 1, X: 105, Y: 100}
	w := &skWorld{dmg: dmg, zombie: z, maxLife: maxLife, life: maxLife}
	w.r = &fake.Reader{
		Snap:  &d2.Snapshot{State: d2.StateInGame, Player: &d2.Player{X: 100, Y: 100}, Monsters: []d2.Unit{z}},
		Stats: map[uint32]d2.Stats{},
	}
	w.setPct()
	w.in = input.NewFake(input.Point{})
	now := time.Unix(0, 0)
	return w, skRig{
		R: w.r, In: castCtl{w.in, w}, Sleep: func(d time.Duration) { now = now.Add(d) }, Now: func() time.Time { return now },
		MaxLife: func(class, diff int) (int, int, bool) { return int(maxLife), int(maxLife), class == 7 },
	}
}

func TestSKBoltsToKillBracketsTable(t *testing.T) {
	// 40 life, 9 per hit: 4 hurting casts then the kill; Lo = 40/5 = 8, Hi = 40/4 = 10, table mean (level 4) = 9.
	w, g := newWorld(40, 9)
	var log bytes.Buffer
	g.Log = &log
	o := skOpts{Skill: "Fire Bolt", Key: "f1", Level: 4, Kills: 10, Casts: 100, Wait: time.Second, Timeout: time.Hour}
	var out bytes.Buffer
	if err := skCmd(context.Background(), g, o, &out); err != nil {
		t.Fatal(err)
	}
	var r SKResult
	if err := json.Unmarshal(out.Bytes(), &r); err != nil {
		t.Fatalf("%v: %s", err, out.String())
	}
	if len(r.Used) != 10 || r.MeanLo != 8 || r.MeanHi != 10 || r.Pass == nil || !*r.Pass || r.ExpMean != 9 {
		t.Fatalf("%+v", r)
	}
	if w.in.Calls[0] != "cast f1 {105 100}" || strings.Count(log.String(), "\n") < 50 || !strings.Contains(log.String(), "kill") {
		t.Fatalf("call %q log %q", w.in.Calls[0], log.String())
	}
}

func TestSKFailsWhenTableOutsideBracket(t *testing.T) {
	_, g := newWorld(40, 20) // 2 hits per kill: Lo 20, Hi 40; level 1 expects 4.5
	o := skOpts{Skill: "Fire Bolt", Key: "f1", Level: 1, Kills: 10, Casts: 100, Wait: time.Second, Timeout: time.Hour}
	r, err := g.sample(context.Background(), o)
	if err != nil || r.Pass == nil || *r.Pass {
		t.Fatalf("%+v %v", r, err)
	}
	_, g = newWorld(40, 9)
	o.Kills, o.Level = 3, 4
	r, _ = g.sample(context.Background(), o)
	if r.Pass == nil || *r.Pass || !strings.Contains(r.Note, "fewer than 10") {
		t.Fatalf("short run must not pass: %+v", r)
	}
}

func TestSKOneShotAndMissAndNoTarget(t *testing.T) {
	_, g := newWorld(10, 0, 50, 50) // miss, then one-shot kills
	o := skOpts{Skill: "Fire Bolt", Key: "f1", Level: 8, Kills: 10, Casts: 9, Wait: time.Second, Timeout: time.Hour}
	r, _ := g.sample(context.Background(), o)
	if r.Misses != 3 || r.OneShot != 6 || len(r.Used) != 0 || r.Pass != nil {
		t.Fatalf("%+v", r)
	}
	_, g = newWorld(10, 5)
	g.R.(*fake.Reader).Snap.Monsters = nil
	r, _ = g.sample(context.Background(), o)
	if !strings.HasPrefix(r.Note, "no live target") {
		t.Fatalf("%+v", r)
	}
}

func TestSKFlagsAndGuards(t *testing.T) {
	cls := func(n string) (int, bool) { return 7, n == "zombie1" }
	o, err := parseSK([]string{"--skill", "Charged Bolt", "--key", "f2", "--monster", "zombie1", "--level", "8"}, cls)
	if err != nil || o.Class != 7 || o.Level != 8 || o.Kills != 10 {
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
