package launch

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"sort"
	"strconv"
	"strings"
	"sync"
	"time"

	"d2lod-bot/internal/mem"
	"d2lod-bot/internal/paths"
)

// Diablo II refuses a second copy inside one Wine prefix ("Only one copy of Diablo II may run at a
// time", a per-wineserver guard; VERIFIED LIVE 2026-10-28). So every instance has its OWN prefix
// (own wineserver) and its own d2input helper on its own port. Prefixes are hardlink clones of the
// main one (tools/game.sh mkprefix), so they cost almost no disk. See docs/multi.md.

// Placement positions a game window on a real display (Xvfb instances need none).
type Placement struct {
	Set  bool
	X, Y int
}

// Instance describes one game instance.
type Instance struct {
	ID        string   // "i1": stable key; also the D2BOT_INSTANCE marker
	Character string   // character this instance is logged into (one instance per character)
	Class     string   // class key used when the character has to be created
	Args      []string // extra Game.exe arguments (default "-w -ns")
	Window    Placement
	Display   string // X display for the game, e.g. ":101" (a private Xvfb); "" inherits DISPLAY
}

// Num is the number in the id ("i3" -> 3), 0 for i0 or an odd id.
func (i Instance) Num() int {
	n, _ := strconv.Atoi(strings.TrimPrefix(i.ID, "i"))
	return n
}

// HelperAddr is this instance's d2input address (47012+N; the original game keeps 47012).
func (i Instance) HelperAddr() string { return "127.0.0.1:" + strconv.Itoa(47012+i.Num()) }

// Config derives the launch Config of the instance from base: own Root/prefix/logs/helper port, shared
// helper exe. i0 keeps base unchanged apart from the marker.
func (i Instance) Config(base Config) Config {
	c := base
	c.Instance = i.ID
	if i.Num() == 0 {
		return c
	}
	c.Root = paths.SiblingRoot(base.Root, i.ID)
	c.PrefixDir, c.GameDir, c.LogDir = "", "", ""
	c.HelperExe = base.HelperExe // one helper binary, one process per prefix
	if c.HelperExe == "" {
		c.HelperExe = os.Getenv("D2INPUT_EXE") // test builds of the helper without replacing the shared exe
	}
	if c.HelperExe == "" {
		c.HelperExe = filepath.Join(base.Root, "helper", "d2input.exe")
	}
	c.HelperAddr = i.HelperAddr()
	if len(i.Args) > 0 {
		c.GameArgs = i.Args
	} else {
		c.GameArgs = nil
	}
	c.ExtraEnv = append([]string(nil), base.ExtraEnv...)
	if i.Display != "" {
		c.ExtraEnv = append(c.ExtraEnv, "DISPLAY="+i.Display)
	}
	return c
}

// Place moves the instance's game window with xdotool (real displays only; best effort).
func (i Instance) Place(display string) error {
	if !i.Window.Set {
		return nil
	}
	out, err := exec.Command("xdotool", "search", "--name", "^Diablo II$").Output()
	if err != nil {
		return err
	}
	for _, w := range strings.Fields(string(out)) {
		cmd := exec.Command("xdotool", "windowmove", w, strconv.Itoa(i.Window.X), strconv.Itoa(i.Window.Y))
		if display != "" {
			cmd.Env = append(cmd.Environ(), "DISPLAY="+display)
		}
		if err := cmd.Run(); err != nil {
			return err
		}
	}
	return nil
}

var startMu sync.Mutex // instances are started one at a time

// StartInstance launches the game of inst (and its helper) and verifies that exactly one new Game.exe
// process appeared and that the instance's helper reports exactly one window. It serializes with other
// starts and waits stagger first (let the previous instance settle; start order = CPU spike order).
func StartInstance(ctx context.Context, base Config, inst Instance, spawn *Spawner, stagger, timeout time.Duration) (Proc, error) {
	startMu.Lock()
	defer startMu.Unlock()
	select {
	case <-ctx.Done():
		return Proc{}, ctx.Err()
	case <-time.After(stagger):
	}
	cfg := inst.Config(base)
	if err := cfg.Resolve(); err != nil {
		return Proc{}, err
	}
	g := NewGame(cfg, spawn)
	if p, ok := g.Find(); ok {
		return p, fmt.Errorf("instance %s already running (pid %d)", inst.ID, p.PID)
	}
	before := gamePIDs()
	h := NewHelper(cfg, spawn)
	if err := h.Ensure(ctx, timeout); err != nil {
		return Proc{}, fmt.Errorf("helper for %s: %w", inst.ID, err)
	}
	if _, err := g.Launch(ctx, timeout); err != nil {
		return Proc{}, err
	}
	// Window check: the per-prefix helper sees only this game's window.
	deadline := time.Now().Add(timeout)
	for {
		if _, up, _, _ := h.Ping(time.Second); up {
			break
		}
		if time.Now().After(deadline) {
			return Proc{}, fmt.Errorf("instance %s: no game window within %s", inst.ID, timeout)
		}
		time.Sleep(300 * time.Millisecond)
	}
	added := newPIDs(before, gamePIDs())
	if len(added) != 1 {
		return Proc{}, fmt.Errorf("instance %s: expected exactly one new Game.exe, got %v", inst.ID, added)
	}
	p, ok := g.Find()
	if !ok || p.PID != added[0] {
		return Proc{}, fmt.Errorf("instance %s: new Game.exe %d is not marked %s", inst.ID, added[0], inst.ID)
	}
	_ = inst.Place(inst.Display)
	return p, nil
}

func gamePIDs() map[int]bool {
	out := map[int]bool{}
	for _, p := range mem.FindGamesFor("") {
		out[p] = true
	}
	return out
}

func newPIDs(before, after map[int]bool) []int {
	var out []int
	for p := range after {
		if !before[p] {
			out = append(out, p)
		}
	}
	sort.Ints(out)
	return out
}
