package fleet

import (
	"slices"
	"testing"
	"time"

	"d2lod-bot/internal/paths"
)

// managedSetup: the d2bot of a slot starts the game (rt.alive) and its SIGTERM stops it, like --stop-game.
func managedSetup(t *testing.T) (*Fleet, *fakeRT, *fakeExec) {
	t.Helper()
	rt, ex := newRT(), newExec()
	ex.onStart = func(BotCmd) { rt.mu.Lock(); rt.alive["i1"] = true; rt.mu.Unlock() }
	pb := &ProcBots{Exec: ex, Binary: "/bin/d2bot", Layout: paths.Layout{Root: t.TempDir()}, Sleep: fastSleep,
		Backoff: Backoff{Min: time.Millisecond, Max: 4 * time.Millisecond, StableAfter: time.Hour}, StopGrace: 20 * time.Millisecond}
	f, _ := mk(t, rt, bigProbe(), func(c *Config) { c.Bots = pb })
	return f, rt, ex
}

func TestManagedBotLaunchesGame(t *testing.T) {
	f, rt, ex := managedSetup(t)
	v, err := f.Add(t.Context(), Spec{Character: "Wiz", Goals: "levelto:8"})
	if err != nil {
		t.Fatal(err)
	}
	wait(t, f, v.ID, Running)
	if rt.count("prepare i1") != 1 || rt.count("start i1") != 0 || rt.count("login i1 Wiz") != 0 {
		t.Fatalf("fleet must only prepare: %v", rt.log)
	}
	if !slices.Contains(ex.cmd(0).Args, "--stop-game") || !slices.Contains(ex.cmd(0).Args, "--managed") {
		t.Fatalf("args %v", ex.cmd(0).Args)
	}
	if b := waitBot(t, f, v.ID, BotRunning); b.GamePid != 4242 {
		t.Fatalf("game pid %+v", b)
	}
}

func TestManagedRemoveAndRestartGoThroughBot(t *testing.T) {
	f, rt, ex := managedSetup(t)
	v, _ := f.Add(t.Context(), Spec{Character: "Wiz"})
	wait(t, f, v.ID, Running)
	if err := f.Restart(t.Context(), v.ID, "test"); err != nil {
		t.Fatal(err)
	}
	// the d2bot got SIGTERM, then a new one launches the game again
	deadline := time.Now().Add(5 * time.Second)
	for ex.n() < 2 {
		if time.Now().After(deadline) {
			t.Fatal("no second d2bot")
		}
		time.Sleep(time.Millisecond)
	}
	if len(ex.proc(0).sigs) == 0 {
		t.Fatal("first d2bot never got a signal")
	}
	if rt.count("start i1") != 0 {
		t.Fatal("fleet started the game")
	}
	wait(t, f, v.ID, Running)
	if err := f.Remove(t.Context(), v.ID, RemoveOpts{}); err != nil {
		t.Fatal(err)
	}
	if rt.count("leave i1") != 0 {
		t.Fatal("OOG leave must go through the bot")
	}
	if rt.count("stop i1") == 0 {
		t.Fatal("cleanup stop missing")
	}
	if len(ex.proc(ex.n()-1).sigs) == 0 {
		t.Fatal("remove did not SIGTERM d2bot")
	}
}

func TestManagedNeedsPreparer(t *testing.T) {
	type bare struct{ Runtime }
	pb := &ProcBots{}
	if _, err := New(Config{Runtime: bare{newRT()}, Bots: pb}); err == nil {
		t.Fatal("want error")
	}
	pb.Attach = true
	if _, err := New(Config{Runtime: bare{newRT()}, Bots: pb}); err != nil {
		t.Fatal(err)
	}
}

func TestManagedGameGapIsNotARestart(t *testing.T) {
	f, rt, ex := managedSetup(t)
	v, _ := f.Add(t.Context(), Spec{Character: "Wiz"})
	wait(t, f, v.ID, Running)
	rt.kill("i1") // d2bot's own recycle: the game is gone for a while, the fake clock does not advance
	time.Sleep(50 * time.Millisecond)
	rt.mu.Lock()
	rt.alive["i1"] = true // d2bot brought it back
	rt.mu.Unlock()
	time.Sleep(20 * time.Millisecond)
	if ex.n() != 1 {
		t.Fatalf("a short game gap restarted the slot (%d d2bot starts)", ex.n())
	}
}

func TestManagedGameGoneForGameWaitRestarts(t *testing.T) {
	rt, ex := newRT(), newExec()
	ex.onStart = func(BotCmd) { rt.mu.Lock(); rt.alive["i1"] = true; rt.mu.Unlock() }
	pb := &ProcBots{Exec: ex, Binary: "/bin/d2bot", Layout: paths.Layout{Root: t.TempDir()}, Sleep: fastSleep,
		Backoff: Backoff{Min: time.Millisecond, Max: 4 * time.Millisecond, StableAfter: time.Hour}, StopGrace: 20 * time.Millisecond}
	f, clk := mk(t, rt, bigProbe(), func(c *Config) { c.Bots = pb })
	v, _ := f.Add(t.Context(), Spec{Character: "Wiz"})
	wait(t, f, v.ID, Running)
	rt.kill("i1")
	time.Sleep(20 * time.Millisecond)
	clk.add(10 * time.Minute) // past GameWait
	deadline := time.Now().Add(5 * time.Second)
	for ex.n() < 2 {
		if time.Now().After(deadline) {
			t.Fatal("slot not restarted although the game stayed gone")
		}
		time.Sleep(time.Millisecond)
	}
}
