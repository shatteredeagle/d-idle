package d2dctl_test

import (
	"context"
	"errors"
	"net"
	"strings"
	"sync"
	"testing"
	"time"

	"d2lod-bot/internal/d2dctl"
	"d2lod-bot/internal/d2dctl/fake"
)

func start(t *testing.T, opts fake.Options) (*fake.Server, *d2dctl.Client) {
	t.Helper()
	path, cleanup, err := fake.TempSocket()
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(cleanup)
	srv, err := fake.New(path, fake.Sample(), opts)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(srv.Close)
	c := d2dctl.New(path, d2dctl.Options{Timeout: 2 * time.Second})
	t.Cleanup(func() { c.Close() })
	return srv, c
}

func ctx() context.Context { return context.Background() }

func TestBasicVerbs(t *testing.T) {
	_, c := start(t, fake.Options{})
	if err := c.Ping(ctx()); err != nil {
		t.Fatal(err)
	}
	in, err := c.Info(ctx())
	if err != nil || in.W != 800 || in.H != 600 || in.Frame == 0 {
		t.Fatalf("info %+v %v", in, err)
	}
	st, _, err := c.State(ctx())
	if err != nil || st.Class != "Necromancer" || st.Life != (d2dctl.Pair{45, 45}) || st.GameMenuOpen() {
		t.Fatalf("state %+v %v", st, err)
	}
	n, err := c.Screenshot(ctx(), "/tmp/x.png")
	if err != nil || n != 12345 {
		t.Fatalf("screenshot %d %v", n, err)
	}
}

func TestLists(t *testing.T) {
	_, c := start(t, fake.Options{})
	npcs, err := c.NPCs(ctx())
	if err != nil || len(npcs.Items) != 2 || npcs.Items[1].Name != "Deckard Cain" || npcs.Items[0].Index != 0 || !npcs.Items[0].HasMenu || npcs.Items[0].CellX != 35.7 {
		t.Fatalf("npcs %+v %v", npcs.Items, err)
	}
	mons, err := c.Monsters(ctx())
	if err != nil || len(mons.Items) != 2 {
		t.Fatalf("mons %+v %v", mons.Items, err)
	}
	m := mons.Items[1]
	if m.Name != "Fallen Shaman" || m.Unit != 5 || !m.IsBoss() || len(m.Mods) != 2 || m.Mods[1] != 26 || m.HP != (d2dctl.Pair{7, 9}) || m.Level != 2 || m.Row != "A1:3" || m.SX != 760 {
		t.Fatalf("monster %+v", m)
	}
	if mons.Items[0].IsBoss() || mons.Items[0].Mods != nil || mons.Items[0].Dead() {
		t.Fatalf("monster0 %+v", mons.Items[0])
	}
	g, err := c.Ground(ctx())
	if err != nil || len(g.Items) != 1 || !g.Items[0].IsGold() || g.Items[0].Label != "8800 Gold" || g.Items[0].Unit != 1 {
		t.Fatalf("ground %+v %v", g.Items, err)
	}
	if x, y, ok := g.Items[0].Cell(); !ok || x != 40.9 || y != 11.5 {
		t.Fatalf("ground cell %v %v %v", x, y, ok)
	}
	mn, err := c.Menu(ctx())
	if err != nil || len(mn.Items) != 3 || mn.Items[1].Text != "talk" || mn.Items[1].Y != 468 {
		t.Fatalf("menu %+v %v", mn.Items, err)
	}
	items, _, err := c.Items(ctx())
	if err != nil || len(items) != 1 || items[0].Code != "wnd" || items[0].Loc() != 1 || len(items[0].Tooltip) != 2 {
		t.Fatalf("items %+v %v", items, err)
	}
}

func TestDebugFamily(t *testing.T) {
	_, c := start(t, fake.Options{})
	pl, err := c.Players(ctx())
	if err != nil || len(pl.Items) != 1 || pl.Items[0].Name != "Spike" || pl.Items[0].Seat != -1 {
		t.Fatalf("players %+v %v", pl.Items, err)
	}
	lv, err := c.Level(ctx())
	if err != nil || lv.ID != 1 || lv.W != 57 || lv.OriginY != 912 || lv.X != 30.7 {
		t.Fatalf("level %+v %v", lv, err)
	}
	q, err := c.QuestShow(ctx(), 1)
	if err != nil || q.Bits != 0x14 || len(q.States) != 6 || q.States[0] != 3 {
		t.Fatalf("quest %+v %v", q, err)
	}
	co, err := c.Corpses(ctx())
	if err != nil || len(co.Items) != 1 || co.Items[0].Items != 3 {
		t.Fatalf("corpses %+v %v", co.Items, err)
	}
	w, err := c.Warps(ctx())
	if err != nil || len(w.Items) != 1 || w.Items[0].To != 8 {
		t.Fatalf("warps %+v %v", w.Items, err)
	}
	pr, err := c.Presets(ctx())
	if err != nil || len(pr.Items) != 2 || !pr.Items[1].Hidden || pr.Items[0].Name != "Waypoint" {
		t.Fatalf("presets %+v %v", pr.Items, err)
	}
	ob, err := c.Objects(ctx())
	if err != nil || len(ob.Items) != 1 || !ob.Items[0].Locked {
		t.Fatalf("objects %+v %v", ob.Items, err)
	}
	po, err := c.Portals(ctx())
	if err != nil || len(po.Items) != 2 || po.Items[1].Which != 1 {
		t.Fatalf("portals %+v %v", po.Items, err)
	}
	at, err := c.Attack(ctx())
	if v, _ := at.Get("res"); err != nil || v != "0/0/0/0" {
		t.Fatalf("attack %+v %v", at, err)
	}
	if v, err := c.Blocked(ctx(), 1, 1); err != nil || v {
		t.Fatalf("blocked %v %v", v, err)
	}
	if g, err := c.Grid(ctx(), 5); err != nil || len(g) != 2 {
		t.Fatalf("grid %v %v", g, err)
	}
	for _, f := range []func() error{
		func() error { return c.Die(ctx()) },
		func() error { return c.Kill(ctx(), 2, "diablo") },
		func() error { return c.SetStat(ctx(), 12, 8) },
		func() error { return c.StatPts(ctx(), 5) },
		func() error { return c.SkillPts(ctx(), 5) },
		func() error { return c.ClearInv(ctx()) },
		func() error { return c.Points(ctx(), 70, 3) },
		func() error { return c.QuestDone(ctx(), 1) },
		func() error { return c.QuestReset(ctx(), 1) },
		func() error { _, err := c.Equip(ctx(), "wnd", 4); return err },
	} {
		if err := f(); err != nil {
			t.Fatal(err)
		}
	}
}

func TestCmdLines(t *testing.T) {
	srv, c := start(t, fake.Options{})
	c.CmdMove(ctx(), 31.5, 14)
	c.CmdInteract(ctx(), 4)
	c.CorpseInteract(ctx(), 0)
	c.CmdSkill(ctx(), 72, 12.1, 28.7, d2dctl.SkillOpts{Unit: 2, Left: true})
	c.CmdSkill(ctx(), 72, 12.1, 28.7, d2dctl.SkillOpts{})
	c.CmdPickup(ctx(), 1)
	c.CmdBelt(ctx(), 1, true)
	c.CmdBelt(ctx(), 0, false)
	c.CmdSelect(ctx(), 72, true)
	if err := c.CmdSkill(ctx(), 1, 1, 1, d2dctl.SkillOpts{Item: 9}); err == nil {
		t.Fatal("item without left should be rejected")
	}
	want := []string{
		"cmd move 31.5 14", "cmd interact 4", "cmd interact -3000",
		"cmd skill 72 12.1 28.7 2 left", "cmd skill 72 12.1 28.7", "cmd pickup 1",
		"cmd belt 1 merc", "cmd belt 0", "cmd select 72 1",
	}
	got := srv.Lines()
	if strings.Join(got, "|") != strings.Join(want, "|") {
		t.Fatalf("got %q", got)
	}
}

func TestErrors(t *testing.T) {
	_, c := start(t, fake.Options{})
	r, err := c.Do(ctx(), "frobnicate 1")
	var ce *d2dctl.CmdError
	if !errors.As(err, &ce) || ce.Msg != "unknown verb 'frobnicate'" || r == nil {
		t.Fatalf("%v %v", r, err)
	}
	if !c.Connected() {
		t.Fatal("plain err must keep the connection")
	}
	if err := c.Ping(ctx()); err != nil {
		t.Fatal(err)
	}
	if err := c.Key(ctx(), "shift"); err == nil {
		t.Fatal("key shift should err")
	}
	if _, err := c.Do(ctx(), "a\nb"); err == nil {
		t.Fatal("newline accepted")
	}
	if _, err := c.Do(ctx(), strings.Repeat("x", d2dctl.MaxLine)); !errors.Is(err, d2dctl.ErrLineTooLong) {
		t.Fatal(err)
	}
}

func TestBusy(t *testing.T) {
	srv, c1 := start(t, fake.Options{})
	if err := c1.Ping(ctx()); err != nil {
		t.Fatal(err)
	}
	c2 := d2dctl.New(srv.Path(), d2dctl.Options{Timeout: time.Second})
	defer c2.Close()
	err := c2.Ping(ctx())
	if !errors.Is(err, d2dctl.ErrBusy) {
		t.Fatalf("want busy, got %v", err)
	}
	if c2.Connected() {
		t.Fatal("busy connection should be dropped")
	}
	// first client still fine, and once it leaves the second can connect (reconnect path)
	if err := c1.Ping(ctx()); err != nil {
		t.Fatal(err)
	}
	c1.Close()
	deadline := time.Now().Add(2 * time.Second)
	for {
		if err = c2.Ping(ctx()); err == nil {
			break
		}
		if time.Now().After(deadline) {
			t.Fatal(err)
		}
		time.Sleep(10 * time.Millisecond)
	}
}

func TestTimeoutAndContext(t *testing.T) {
	_, c := start(t, fake.Options{SlowVerbs: map[string]time.Duration{"npcs": 400 * time.Millisecond}})
	cx, cancel := context.WithTimeout(ctx(), 80*time.Millisecond)
	defer cancel()
	t0 := time.Now()
	_, err := c.NPCs(cx)
	if !errors.Is(err, context.DeadlineExceeded) || time.Since(t0) > 350*time.Millisecond {
		t.Fatalf("err %v after %v", err, time.Since(t0))
	}
	if c.Connected() {
		t.Fatal("timed out connection must be dropped (stream out of step)")
	}
	// next call redials and gets a clean stream, not the stale npcs answer
	time.Sleep(450 * time.Millisecond)
	if err := c.Ping(ctx()); err != nil {
		t.Fatal(err)
	}

	// per-client timeout
	srv2, c2 := start(t, fake.Options{Delay: 300 * time.Millisecond})
	_ = srv2
	c3 := d2dctl.New(c2.Path(), d2dctl.Options{Timeout: 50 * time.Millisecond})
	defer c3.Close()
	if err := c3.Ping(ctx()); !errors.Is(err, context.DeadlineExceeded) {
		t.Fatalf("want deadline, got %v", err)
	}
}

func TestDroppedConnectionAndReconnect(t *testing.T) {
	srv, c := start(t, fake.Options{})
	if err := c.Ping(ctx()); err != nil {
		t.Fatal(err)
	}
	srv.DropClient()
	time.Sleep(20 * time.Millisecond)
	if err := c.Ping(ctx()); err == nil {
		t.Fatal("expected failure on the dead connection")
	}
	if err := c.Ping(ctx()); err != nil {
		t.Fatalf("should have redialed: %v", err)
	}

	// server drops mid-command: no reply, error, and the command is not resent
	srv2, c2 := start(t, fake.Options{DropAfter: 2})
	if err := c2.Ping(ctx()); err != nil {
		t.Fatal(err)
	}
	if err := c2.Click(ctx(), 1, 2); err == nil {
		t.Fatal("expected EOF")
	}
	if n := len(srv2.Lines()); n != 1 {
		t.Fatalf("dropped command must not be logged/resent, log=%v", srv2.Lines())
	}

	// no-reconnect option
	srv3, _ := start(t, fake.Options{})
	c3 := d2dctl.New(srv3.Path(), d2dctl.Options{NoReconnect: true})
	defer c3.Close()
	c3.Ping(ctx())
	srv3.DropClient()
	time.Sleep(20 * time.Millisecond)
	c3.Ping(ctx())
	if err := c3.Ping(ctx()); !errors.Is(err, d2dctl.ErrClosed) {
		t.Fatalf("want ErrClosed, got %v", err)
	}
}

func TestQuitClosesAndClosedClient(t *testing.T) {
	srv, c := start(t, fake.Options{})
	if err := c.Quit(ctx()); err != nil || !srv.Quit() {
		t.Fatal(err)
	}
	c.Close()
	if err := c.Ping(ctx()); !errors.Is(err, d2dctl.ErrClosed) {
		t.Fatal(err)
	}
}

func TestKickQuirk(t *testing.T) {
	srv, c := start(t, fake.Options{})
	if err := c.Goto(ctx(), 2); err != nil {
		t.Fatal(err)
	}
	if !c.NeedsKick() {
		t.Fatal("goto should mark kick")
	}
	c.CmdMove(ctx(), 10, 10) // auto kick first
	if c.NeedsKick() {
		t.Fatal("kick not sent")
	}
	for _, e := range srv.Log() {
		if e.Ignored {
			t.Fatalf("command ignored despite auto kick: %+v", srv.Log())
		}
	}
	// without auto kick the fake swallows it, like the real game
	srv2, c2 := start(t, fake.Options{})
	c2n := d2dctl.New(c2.Path(), d2dctl.Options{NoAutoKick: true})
	defer c2n.Close()
	c2n.Enter(ctx(), 0)
	c2n.CmdMove(ctx(), 1, 1)
	if err := c2n.Kick(ctx()); err != nil {
		t.Fatal(err)
	}
	c2n.CmdInteract(ctx(), 0)
	lg := srv2.Log()
	if !lg[1].Ignored || lg[3].Ignored || lg[1].Line != "cmd move 1 1" {
		t.Fatalf("%+v", lg)
	}
}

func TestConcurrentCallersStaySynchronised(t *testing.T) {
	_, c := start(t, fake.Options{})
	var wg sync.WaitGroup
	for i := 0; i < 8; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for j := 0; j < 25; j++ {
				if m, err := c.Monsters(ctx()); err != nil || len(m.Items) != 2 {
					t.Errorf("monsters %v %v", m.Items, err)
					return
				}
				if err := c.Ping(ctx()); err != nil {
					t.Error(err)
					return
				}
			}
		}()
	}
	wg.Wait()
}

func TestLongLineFromServer(t *testing.T) {
	path, cleanup, _ := fake.TempSocket()
	defer cleanup()
	ln, err := net.Listen("unix", path)
	if err != nil {
		t.Fatal(err)
	}
	defer ln.Close()
	go func() {
		conn, _ := ln.Accept()
		buf := make([]byte, 64)
		conn.Read(buf)
		conn.Write([]byte(strings.Repeat("a", 70<<10) + "\nok\n"))
	}()
	c := d2dctl.New(path, d2dctl.Options{Timeout: time.Second})
	defer c.Close()
	if err := c.Ping(ctx()); !errors.Is(err, d2dctl.ErrLineTooLong) {
		t.Fatalf("got %v", err)
	}
}

func TestScript(t *testing.T) {
	_, c := start(t, fake.Options{})
	script := `
# comment
ping
sleep 10ms
npcs
expect Kashya.*35\.7
-frobnicate
expect err unknown verb
debug level
expect ^1 30\.70
`
	steps, err := d2dctl.RunScript(ctx(), c, strings.NewReader(script))
	if err != nil || len(steps) != 8 {
		t.Fatalf("%v %+v", err, steps)
	}
	steps, err = d2dctl.RunScript(ctx(), c, strings.NewReader("ping\nexpect nothing-here\nping\n"))
	if err == nil || len(steps) != 2 || steps[1].OK {
		t.Fatalf("%v %+v", err, steps)
	}
	if _, err = d2dctl.RunScript(ctx(), c, strings.NewReader("frobnicate\n")); err == nil {
		t.Fatal("err reply without '-' must fail")
	}
}

func TestStepClock(t *testing.T) {
	_, c := start(t, fake.Options{})
	ci, err := c.Step(ctx(), 5)
	if err != nil || ci.Tick != 5 || ci.GameMS != 200 {
		t.Fatalf("step: %+v %v", ci, err)
	}
	ci, err = c.Clock(ctx())
	if err != nil || !ci.Lockstep || ci.Stepped != 5 || ci.GameMS != 200 {
		t.Fatalf("clock: %+v %v", ci, err)
	}
}

func TestPrefetchServesOnceThenAsks(t *testing.T) {
	srv, c := start(t, fake.Options{})
	if err := c.Prefetch(ctx(), "monsters", "npcs", "ground"); err != nil {
		t.Fatal(err)
	}
	m1, err := c.Monsters(ctx())
	if err != nil || len(m1.Items) == 0 {
		t.Fatalf("prefetched monsters %+v %v", m1.Items, err)
	}
	n, err := c.NPCs(ctx())
	if err != nil || len(n.Items) != 2 {
		t.Fatalf("prefetched npcs %+v %v", n.Items, err)
	}
	c.DropPrefetch()
	if _, err := c.Ground(ctx()); err != nil { // dropped: a real round trip
		t.Fatal(err)
	}
	if _, err := c.Monsters(ctx()); err != nil { // consumed: a real round trip
		t.Fatal(err)
	}
	seen := srv.Lines()
	if got := strings.Count(strings.Join(seen, "|"), "monsters"); got != 2 {
		t.Fatalf("monsters reached the server %d times, want 2: %v", got, seen)
	}
	if tr := c.Trace(); len(tr) == 0 {
		t.Fatal("no trace")
	}
}

func TestBusyWhileCommandInFlight(t *testing.T) {
	_, c := start(t, fake.Options{})
	if c.Busy() {
		t.Fatal("idle client reported busy")
	}
	c.DropPrefetch()
}
