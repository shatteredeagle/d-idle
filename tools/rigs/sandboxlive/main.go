//go:build rigs

// sandboxlive proves the Wine mount sandbox end to end on ONE instance at the main menu (never loads a character):
// managed launch (the app is the subreaper), memory attach from outside the namespace, helper TCP, the window, the
// namespace facts (mount/user ns differ, no network mount visible), and the stop path. Results: docs/sandbox.md.
//
//	tools/live.sh --nocap sandbox sandboxlive        (D2_INSTANCE=i4; Xvfb :104 must be up: tools/game.sh helper i4)
package main

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"

	"d2lod-bot/internal/launch"
)

func main() {
	if err := run(); err != nil {
		fmt.Println("FAIL:", err)
		os.Exit(1)
	}
}

func run() error {
	id := os.Getenv("D2BOT_INSTANCE")
	in := launch.Instance{ID: id, Display: fmt.Sprintf(":%d", 100+launch.Instance{ID: id}.Num())}
	if !launch.ValidID(id) || in.Num() == 0 {
		return fmt.Errorf("D2BOT_INSTANCE must be i1..i9, got %q", id)
	}
	cfg := in.Config(launch.Defaults())
	cfg.ExtraEnv = append(cfg.ExtraEnv, "WAYLAND_DISPLAY=", "XAUTHORITY=")
	if err := cfg.Resolve(); err != nil {
		return err
	}
	fmt.Println("sandbox:", cfg.SandboxStatus())
	if err := writeCheck(cfg); err != nil {
		return err
	}
	if err := launch.BecomeSubreaper(); err != nil {
		return err
	}
	ctx := context.Background()
	sp := launch.NewSpawner()
	helper := launch.NewHelper(cfg, sp)
	game := launch.NewGame(cfg, sp)
	game.WindowUp = func() bool { _, up, _, _ := helper.Ping(time.Second); return up }
	if _, ok := game.Find(); ok {
		return fmt.Errorf("a game of %s already runs; stop it first", id)
	}
	if err := game.PrepareLaunch(); err != nil {
		fmt.Println("prepare:", err)
	}
	t0 := time.Now()
	if err := helper.Ensure(ctx, 60*time.Second); err != nil {
		return fmt.Errorf("helper: %w", err)
	}
	fmt.Printf("helper up after %s\n", time.Since(t0).Round(time.Second))
	p, err := game.Launch(ctx, 90*time.Second)
	if err != nil {
		return fmt.Errorf("launch: %w", err)
	}
	fmt.Printf("(c) window up after %s, Game.exe pid %d\n", time.Since(t0).Round(time.Second), p.PID)
	pid, base, rd, err := game.Attach()
	if err != nil {
		return fmt.Errorf("(a) attach: %w", err)
	}
	var b [2]byte
	if err := rd.ReadAt(b[:], base); err != nil {
		return fmt.Errorf("(a) read: %w", err)
	}
	fmt.Printf("(a) attached pid %d base %#x read %q as a descendant, no capability\n", pid, base, b[:])
	_, up, reply, err := helper.Ping(2 * time.Second)
	fmt.Printf("(b) helper %s -> %q up=%v err=%v\n", cfg.HelperAddr, reply, up, err)
	self, _ := os.Readlink("/proc/self/ns/mnt")
	gm, _ := os.Readlink(fmt.Sprintf("/proc/%d/ns/mnt", pid))
	gu, _ := os.Readlink(fmt.Sprintf("/proc/%d/ns/user", pid))
	su, _ := os.Readlink("/proc/self/ns/user")
	fmt.Printf("namespaces: mnt self=%s game=%s ; user self=%s game=%s\n", self, gm, su, gu)
	mi, _ := os.ReadFile(fmt.Sprintf("/proc/%d/mountinfo", pid))
	var leaked []string
	for _, l := range strings.Split(string(mi), "\n") {
		for _, bad := range []string{"cifs", "/mnt/", "/media", "gvfs", "fuse"} {
			if strings.Contains(l, bad) {
				leaked = append(leaked, l)
			}
		}
	}
	fmt.Printf("game sees %d mounts, network/fuse lines: %d\n", strings.Count(string(mi), "\n"), len(leaked))
	if p2, ok := game.Find(); !ok || p2.PID != pid {
		return fmt.Errorf("(d) Find lost the game")
	}
	fmt.Println("(d) Find sees the pid; stopping")
	if err := game.Shutdown(8 * time.Second); err != nil {
		return fmt.Errorf("(d) shutdown: %w", err)
	}
	if err := cfg.WineserverKill(); err != nil {
		fmt.Println("wineserver -k:", err)
	}
	time.Sleep(2 * time.Second)
	_, still := game.Find()
	fmt.Println("(d) after Shutdown + wineserver -k: Game.exe still found =", still)
	return nil
}

// writeCheck is (e) without a character: a program inside the sandbox writes into the save dir of the real prefix and
// the file shows up on the host; the same program sees no /mnt, /media or /run/media.
func writeCheck(cfg launch.Config) error {
	dir := cfg.SaveDir()
	probe := filepath.Join(dir, ".sandbox-probe")
	os.Remove(probe)
	home, _ := os.UserHomeDir()
	dl := filepath.Join(home, "Downloads")
	name, args, err := cfg.Wrap(dir, "/usr/bin/sh", "-c", `echo ok > .sandbox-probe && [ ! -e /mnt ] && [ ! -e /media ] && [ ! -e /run/media ] && [ ! -e "$1" ] && echo hidden`, "sandbox-probe", dl)
	if err != nil {
		return err
	}
	out, err := exec.Command(name, args...).CombinedOutput()
	b, rerr := os.ReadFile(probe)
	os.Remove(probe)
	if err != nil || rerr != nil || strings.TrimSpace(string(b)) != "ok" || !strings.Contains(string(out), "hidden") {
		return fmt.Errorf("(e) write check: err=%v read=%v out=%q", err, rerr, out)
	}
	fmt.Println("(e) wrote into the real save dir from inside the sandbox; /mnt /media /run/media and ~/Downloads are absent")
	return nil
}
