package d2denv

import (
	"bufio"
	"context"
	"errors"
	"fmt"
	"io"
	"log/slog"
	"net"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"testing"
	"time"

	"d2lod-bot/internal/bot"
	"d2lod-bot/internal/d2"
	"d2lod-bot/internal/d2dctl"
	"d2lod-bot/internal/d2dctl/fake"
	"d2lod-bot/internal/d2dinput"
	"d2lod-bot/internal/d2dsrc"
	"d2lod-bot/internal/env"
	"d2lod-bot/internal/input"
	"d2lod-bot/internal/nav"
	"d2lod-bot/internal/questmem"
)

func sockPath(t *testing.T) string {
	t.Helper()
	dir, err := os.MkdirTemp("", "d2denv")
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { os.RemoveAll(dir) })
	return filepath.Join(dir, "d.sock")
}

func startFake(t *testing.T) (*fake.Server, string) {
	t.Helper()
	p := sockPath(t)
	s, err := fake.New(p, fake.Sample(), fake.Options{})
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(s.Close)
	return s, p
}

func quiet() *slog.Logger { return slog.New(slog.NewTextHandler(io.Discard, nil)) }

func TestBuildReadsStateAndAddsAddr(t *testing.T) {
	_, p := startFake(t)
	be, err := Build(context.Background(), Config{Sock: p, Log: quiet(), ID: "t"})
	if err != nil {
		t.Fatal(err)
	}
	defer be.Close()
	if err := be.Start(context.Background(), 0); err != nil {
		t.Fatal(err)
	}
	if !be.Clock.Lockstep() {
		t.Fatal("the fake reports lockstep: the clock must be a pacer")
	}
	s, err := be.Env.Snapshot()
	if err != nil || s.State != d2.StateInGame || s.Player == nil || s.Player.Name != "Spike" {
		t.Fatalf("snapshot %+v %v", s, err)
	}
	if len(s.Monsters) == 0 {
		t.Fatal("no monsters in the fixture snapshot")
	}
	for _, m := range s.Monsters {
		if m.ID == 0 || m.Addr != m.ID {
			t.Fatalf("Addr must be the unit id: %+v", m)
		}
	}
	// the clock is virtual: Sleep steps the fake instead of waiting
	t0 := be.Env.Clock.Now()
	w := time.Now()
	be.Env.Clock.Sleep(10 * time.Second)
	if time.Since(w) > 2*time.Second || be.Env.Clock.Now().Sub(t0) < 9*time.Second {
		t.Fatalf("virtual sleep: wall %v virtual %v", time.Since(w), be.Env.Clock.Now().Sub(t0))
	}
	if _, err := be.Env.Mem.(interface{ ReadAt([]byte, uint32) error }); !err {
		t.Fatal("Mem")
	}
	if be.Env.Mem.ReadAt(make([]byte, 4), 0x400000) == nil {
		t.Fatal("noMem must fail every read")
	}
	if _, ok := input.AsSemantic(be.Env.In); !ok {
		t.Fatal("d2d input must be semantic")
	}
}

func TestUnitStatsByID(t *testing.T) {
	_, p := startFake(t)
	c := d2dctl.New(p, d2dctl.Options{})
	defer c.Close()
	r := NewReader(d2dsrc.New(c, d2dsrc.Options{Logf: func(string, ...any) {}}))
	if _, err := r.UnitStats(1); !errors.Is(err, d2.ErrUnsupported) {
		t.Fatalf("before any snapshot: %v", err)
	}
	s, err := r.Snapshot()
	if err != nil {
		t.Fatal(err)
	}
	if _, err := r.UnitStats(s.Player.ID); !errors.Is(err, d2.ErrUnsupported) {
		t.Fatalf("player stats without a dump must be unknown: %v", err)
	}
	r.last.Snap.Player.ID, r.last.Snap.Player.Stats = 77, d2.Stats{d2.StatLevel: 5}
	if st, err := r.UnitStats(77); err != nil || st.Get(d2.StatLevel) != 5 {
		t.Fatalf("player stats by id: %v %v", st, err)
	}
	r.last.Extras.Items[900] = d2dsrc.ItemExtra{Qty: 3, Dur: 10, MaxDur: 12, Sockets: 2}
	if st, err := r.UnitStats(900); err != nil || st.Get(statQuantity) != 3 || st.Get(statDurability) != 10 || st.Get(statMaxDur) != 12 || st.Get(statSockets) != 2 {
		t.Fatalf("item stats by id: %v %v", st, err)
	}
	if _, err := r.UnitStats(s.Monsters[0].ID); !errors.Is(err, d2.ErrUnsupported) {
		t.Fatalf("monster stats must stay unsupported, got %v", err)
	}
	if _, err := r.Units(d2.TypeItem); !errors.Is(err, d2.ErrUnsupported) {
		t.Fatalf("item units: %v", err)
	}
}

// scripted is a d2d stand-in that answers exact lines (the fake has no dump verb).
func scripted(t *testing.T, replies map[string]string) string {
	t.Helper()
	p := sockPath(t)
	ln, err := net.Listen("unix", p)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { ln.Close() })
	var mu sync.Mutex
	go func() {
		for {
			c, err := ln.Accept()
			if err != nil {
				return
			}
			go func() {
				defer c.Close()
				rd := bufio.NewReader(c)
				for {
					l, err := rd.ReadString('\n')
					if err != nil {
						return
					}
					l = strings.TrimSpace(l)
					mu.Lock()
					r, ok := replies[l]
					mu.Unlock()
					if !ok {
						r = "err unknown verb '" + strings.Fields(l)[0] + "'\n"
					}
					c.Write([]byte(r))
				}
			}()
		}
	}()
	return p
}

func TestQuestSource(t *testing.T) {
	idx, ok := questmem.QuestIndex("den_of_evil")
	if !ok {
		t.Fatal("no den index")
	}
	diff := func(set int, w int) string {
		a := make([]string, d2.QuestWords)
		for i := range a {
			a[i] = "0"
		}
		if set >= 0 {
			a[set] = fmt.Sprint(w)
		}
		return "[" + strings.Join(a, ",") + "]"
	}
	dump := `{"v":1,"quests":{"diff":[` + diff(idx, 0x1234) + "," + diff(-1, 0) + "," + diff(-1, 0) + `],"waypoints":[3,0,0]}}` + "\nok\n"
	p := scripted(t, map[string]string{"dump quests": dump})
	c := d2dctl.New(p, d2dctl.Options{})
	defer c.Close()
	q := &QuestSource{Src: d2dsrc.New(c, d2dsrc.Options{Logf: func(string, ...any) {}})}
	if w, ok := q.Word(0, "den_of_evil"); !ok || w != 0x1234 {
		t.Fatalf("word %x %v", w, ok)
	}
	if _, ok := q.Word(0, "no_such_quest"); ok {
		t.Fatal("unknown id answered")
	}
	if bit, ok := questmem.WaypointBit(1); !ok {
		t.Fatal("no waypoint bit for area 1")
	} else if a, known := q.Waypoint(0, 1); !known || a != (3>>uint(bit)&1 == 1) {
		t.Fatalf("waypoint area 1 (bit %d of mask 3): active=%v known=%v", bit, a, known)
	}
	// stock d2d: no dump verb, every answer is unknown
	p2 := scripted(t, map[string]string{})
	c2 := d2dctl.New(p2, d2dctl.Options{})
	defer c2.Close()
	q2 := &QuestSource{Src: d2dsrc.New(c2, d2dsrc.Options{Logf: func(string, ...any) {}})}
	if _, ok := q2.Word(0, "den_of_evil"); ok {
		t.Fatal("stock d2d must be unknown, not zero")
	}
}

func TestCyclerAttach(t *testing.T) {
	srv, p := startFake(t)
	be, err := Build(context.Background(), Config{Sock: p, Log: quiet(), Spec: d2dinput.Spec{Name: "Spike", Class: 1}})
	if err != nil {
		t.Fatal(err)
	}
	defer be.Close()
	ctx := context.Background()
	if err := be.Cycler.EnterGame(ctx, ""); err != nil {
		t.Fatal(err)
	}
	if err := be.Cycler.EnterGame(ctx, "hell"); !errors.Is(err, input.ErrUnsupported) {
		t.Fatalf("hell: %v", err)
	}
	if err := be.Cycler.NewGame(ctx, ""); !errors.Is(err, input.ErrUnsupported) {
		t.Fatalf("attached d2d must not be restarted: %v", err)
	}
	if err := be.Cycler.LeaveGame(ctx); err != nil {
		t.Fatal(err)
	}
	if srv.Quit() {
		t.Fatal("attach mode must never send quit")
	}
	for _, l := range srv.Lines() {
		if l == "save" {
			return
		}
	}
	t.Fatal("no save sent")
}

func TestSeamsDefaultToNotYet(t *testing.T) {
	_, p := startFake(t)
	be, err := Build(context.Background(), Config{Sock: p, Log: quiet()})
	if err != nil {
		t.Fatal(err)
	}
	defer be.Close()
	f, tw, l := be.Behaviors(Seams{}, nil)
	if l != nil {
		t.Fatal("loot has no stub: nil keeps the bot's no-pickup path")
	}
	if _, err := f.Clear(context.Background(), bot.ClearOpts{}); !errors.Is(err, ErrNotYet) {
		t.Fatalf("clear: %v", err)
	}
	if err := tw.ToTown(context.Background()); !errors.Is(err, ErrNotYet) {
		t.Fatalf("totown: %v", err)
	}
	if f.Safety(&d2.Snapshot{}) != bot.SafetyNone {
		t.Fatal("stub safety must be none")
	}
	// a plugged seam wins
	got := false
	be.Behaviors(Seams{Fighter: func(*env.Env, *nav.Navigator) bot.Fighter { got = true; return f }}, nil)
	if !got {
		t.Fatal("seam not called")
	}
}
