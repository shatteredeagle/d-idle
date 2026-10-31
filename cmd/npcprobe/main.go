// npcprobe measures how fast each greeting-skip strategy (town.greetingSkip, docs/npc-skip.md) gets an NPC menu on a
// character's FIRST contact with that NPC. Run it on a brand-new character with the bot NOT running:
//
//	tools/live.sh <agent> npcprobe [-plan npc:148,esc:154,ground:147,wait:150] [-again] [-selflock]
//
// tools/live.sh already holds flock /tmp/d2lod-game-<id>.lock; a direct run adds -selflock to take it itself. The
// probe refuses to run while a bot or soak process of the same instance is alive. It only walks next to each NPC,
// opens its dialog with the strategy, prints the measurement and presses Esc.
package main

import (
	"context"
	"flag"
	"fmt"
	"log/slog"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"syscall"
	"time"

	"d2lod-bot/internal/d2"
	"d2lod-bot/internal/d2map"
	"d2lod-bot/internal/data"
	"d2lod-bot/internal/env"
	"d2lod-bot/internal/goals"
	"d2lod-bot/internal/input"
	"d2lod-bot/internal/mem"
	"d2lod-bot/internal/nav"
	"d2lod-bot/internal/town"
)

// One strategy per NPC: first contact happens once per NPC and character, so strategies cannot share an NPC.
// Act 1 classes (V, docs/map.md): Akara 148, Charsi 154, Gheed 147, Kashya 150, Warriv 155.
const defaultPlan = "npc:148,esc:154,ground:147,wait:150"

var strategies = map[string]int{"wait": town.SkipWait, "npc": town.SkipNPC, "esc": town.SkipEsc, "ground": town.SkipGround}

type step struct {
	strat int
	class int
}

func main() {
	plan := flag.String("plan", defaultPlan, "strategy:npcClass pairs, in order")
	again := flag.Bool("again", false, "open each NPC a second time (not first contact) as the baseline")
	selflock := flag.Bool("selflock", false, "take the instance lock itself (direct runs; tools/live.sh holds it already)")
	budget := flag.Duration("budget", 90*time.Second, "limit per NPC")
	flag.Parse()
	steps, err := parsePlan(*plan)
	die(err)
	inst := os.Getenv("D2BOT_INSTANCE")
	if inst == "" {
		inst = "i0"
	}
	die(refuseIfBusy(inst))
	if *selflock {
		f, err := takeLock(inst)
		die(err)
		defer f.Close()
	}

	src, err := mem.OpenGame()
	die(err)
	defer src.Close()
	home, _ := os.UserHomeDir()
	db, err := data.Open(home+"/Games/d2lod/proton/pfx/drive_c/Program Files (x86)/Diablo II", "/tmp/d2lod-town-cache")
	die(err)
	tune := env.DefaultTuning()
	game := input.NewGame(input.NewClient(""), input.Reader{Src: src})
	e := &env.Env{ID: "npcprobe", Log: slog.New(slog.NewTextHandler(os.Stderr, &slog.HandlerOptions{Level: slog.LevelInfo})), Mem: src,
		Read: d2.New(src), World: d2map.NewTracker(src), In: game, Data: db, Tune: &tune, Clock: env.RealClock{}, Emit: func(goals.Event) {}}
	t := town.New(e, nav.New(e))

	fmt.Printf("%-8s %-6s %-5s %9s %7s %12s  %s\n", "strategy", "class", "visit", "ms", "clicks", "first-contact", "result")
	for _, st := range steps {
		for visit := 1; visit <= 1+b2i(*again); visit++ {
			e.Tune.Town.GreetingSkip = st.strat
			ctx, cancel := context.WithTimeout(context.Background(), *budget)
			start := time.Now()
			_, err := t.OpenForTest(ctx, st.class)
			cancel()
			v := t.LastVisit()
			res := "menu open"
			if err != nil {
				res = "FAILED: " + err.Error()
			}
			fmt.Printf("%-8s %-6d %-5d %9d %7d %12v  %s (wall %s)\n", town.SkipName(st.strat), st.class, visit, v.Ms, v.Clicks, v.FirstContact, res, time.Since(start).Round(time.Millisecond))
			_ = game.CloseAll()
			time.Sleep(500 * time.Millisecond)
		}
	}
}

func parsePlan(s string) ([]step, error) {
	var out []step
	for _, p := range strings.Split(s, ",") {
		name, cls, ok := strings.Cut(strings.TrimSpace(p), ":")
		st, known := strategies[name]
		c, err := strconv.Atoi(cls)
		if !ok || !known || err != nil {
			return nil, fmt.Errorf("bad plan element %q (want strategy:class, strategy one of wait npc esc ground)", p)
		}
		out = append(out, step{st, c})
	}
	return out, nil
}

// refuseIfBusy fails when a bot or soak process of this instance is running: they would fight the probe over the
// mouse. It reads /proc/*/environ for D2BOT_INSTANCE (same user only), so it sees what tools/live.sh started.
func refuseIfBusy(inst string) error {
	self := os.Getpid()
	ents, _ := filepath.Glob("/proc/[0-9]*")
	for _, d := range ents {
		pid, _ := strconv.Atoi(filepath.Base(d))
		if pid == self {
			continue
		}
		cmd, err := os.ReadFile(d + "/cmdline")
		if err != nil {
			continue
		}
		c := string(cmd)
		if !(strings.Contains(c, "d2bot") || strings.Contains(c, "d2soak") || strings.Contains(c, "d2dsoak") || strings.Contains(c, "soak24")) || strings.Contains(c, "npcprobe") {
			continue
		}
		env, err := os.ReadFile(d + "/environ")
		if err != nil {
			continue
		}
		// The worker daemon name for an instance-less run is i0
		if strings.Contains("\x00"+string(env), "\x00D2BOT_INSTANCE="+inst+"\x00") || (inst == "i0" && !strings.Contains(string(env), "D2BOT_INSTANCE=")) {
			return fmt.Errorf("npcprobe: refusing, pid %d (%s) drives instance %s", pid, strings.ReplaceAll(strings.TrimRight(c, "\x00"), "\x00", " "), inst)
		}
	}
	return nil
}

// takeLock takes the legacy instance lock without waiting (tools/live.sh and tools/helper.sh use the same file).
func takeLock(inst string) (*os.File, error) {
	path := "/tmp/d2lod-game-" + inst + ".lock"
	if inst == "i0" {
		path = "/tmp/d2lod-game.lock"
	}
	f, err := os.OpenFile(path, os.O_CREATE|os.O_RDWR, 0o666)
	if err != nil {
		return nil, err
	}
	if err := syscall.Flock(int(f.Fd()), syscall.LOCK_EX|syscall.LOCK_NB); err != nil {
		f.Close()
		return nil, fmt.Errorf("npcprobe: refusing, %s is held (a bot or soak owns the instance)", path)
	}
	return f, nil
}

func b2i(b bool) int {
	if b {
		return 1
	}
	return 0
}

func die(err error) {
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}
