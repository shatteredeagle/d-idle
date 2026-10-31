package d2dinput

import (
	"testing"
	"time"

	"d2lod-bot/internal/d2dctl"
	"d2lod-bot/internal/d2dctl/fake"
)

func TestPacer(t *testing.T) {
	path, cleanup, err := fake.TempSocket()
	if err != nil {
		t.Fatal(err)
	}
	defer cleanup()
	srv, err := fake.New(path, fake.Sample(), fake.Options{})
	if err != nil {
		t.Fatal(err)
	}
	defer srv.Close()
	c := d2dctl.New(path, d2dctl.Options{Timeout: 2 * time.Second})
	defer c.Close()
	p := &Pacer{C: c, Chunk: 4}
	t0 := p.Now()
	p.Sleep(100 * time.Millisecond) // 3 ticks
	p.Sleep(400 * time.Millisecond) // 10 ticks in 3 chunks
	p.Sleep(0)
	if got := p.Now().Sub(t0); got != 520*time.Millisecond {
		t.Fatalf("virtual %v", got)
	}
	steps, ticks, _ := p.Stats()
	if steps != 4 || ticks != 13 || p.Err() != nil {
		t.Fatalf("steps %d ticks %d err %v", steps, ticks, p.Err())
	}
	c.Close()
	srv.Close()
	p.Sleep(80 * time.Millisecond)
	if p.Err() == nil || p.Now().Sub(t0) != 600*time.Millisecond {
		t.Fatalf("failed step must still advance: %v %v", p.Err(), p.Now().Sub(t0))
	}
}

func TestPacerFollowSurvivesGameRestart(t *testing.T) {
	p := &Pacer{}
	p.start, p.last, p.synced = 5000, 5000, true
	p.follow(8000)
	before := p.cur
	if before != 3000 {
		t.Fatalf("cur %d", before)
	}
	p.follow(200) // d2d restarted: game_ms counts from 0 again
	if p.cur < before || p.cur != 3200 {
		t.Fatalf("virtual time went back or lost the restart: %d", p.cur)
	}
	p.follow(1000)
	if p.cur != 4000 {
		t.Fatalf("cur %d", p.cur)
	}
}

func TestPacerNowIsNotZeroTime(t *testing.T) {
	p := &Pacer{synced: true}
	if p.Now().IsZero() {
		t.Fatal("virtual time 0 must not read as the zero Time")
	}
}
