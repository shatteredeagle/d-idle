// Command didleboot is the container's boot shim: it selects the release to run (the baked base or a downloaded one
// below /data/didle), supervises it, confirms or rolls back an update and hands over to the release's launcher. It is
// part of the image and cannot be replaced by an update, so it stays small. Design: docs/research/self-update-design.md
// 3.1, 3.8, 3.9; drills: docs/self-update-drill.md.
//
//	didleboot [--launcher PATH] [launcher args...]   run (the image ENTRYPOINT is tini -- didleboot --launcher /opt/d2lod/entrypoint.sh)
//	didleboot -select                print the release that would run and exit (drills, support)
//
// Env: DIDLE_ROOT (/data/didle), DIDLE_BASE_BIN (/opt/d2lod), DIDLE_BASE_D2D (/opt/d2d), DIDLE_BASE_VERSION (0.0.0),
// D2BOT_IMAGE_ABI (0), DIDLE_KEYS (/opt/didle/pubkeys), D2BOT_ADDR (127.0.0.1:8420), D2D_SOCK, TESTER=1 (the watch
// screen must answer too), D2BOT_WATCH_ADDR.
package main

import (
	"crypto/ed25519"
	"fmt"
	"os"
	"os/signal"
	"path/filepath"
	"strconv"
	"strings"
	"sync/atomic"
	"syscall"

	"d2lod-bot/internal/selfupdate"
)

func envOr(name, def string) string {
	if v := os.Getenv(name); v != "" {
		return v
	}
	return def
}

func loadKeys(dir string) []ed25519.PublicKey {
	files, _ := filepath.Glob(filepath.Join(dir, "*.pub"))
	var keys []ed25519.PublicKey
	for _, f := range files {
		if b, err := os.ReadFile(f); err == nil {
			if k, err := selfupdate.DecodePublicKey(b); err == nil {
				keys = append(keys, k)
			}
		}
	}
	return keys
}

func imageFromEnv() Image {
	abi, _ := strconv.Atoi(os.Getenv("D2BOT_IMAGE_ABI"))
	bin := envOr("DIDLE_BASE_BIN", "/opt/d2lod")
	return Image{
		ABI:         abi,
		BaseVersion: envOr("DIDLE_BASE_VERSION", "0.0.0"),
		Base:        Release{Base: true, BinDir: bin, D2DDir: envOr("DIDLE_BASE_D2D", "/opt/d2d"), Entry: filepath.Join(bin, "entrypoint.sh")},
		Local:       selfupdate.BuildCompat,
		Keys:        loadKeys(envOr("DIDLE_KEYS", "/opt/didle/pubkeys")),
		WatchWanted: os.Getenv("TESTER") == "1",
	}
}

func main() { os.Exit(realMain(os.Args[1:])) }

func realMain(args []string) int {
	img := imageFromEnv()
	if len(args) >= 2 && args[0] == "--launcher" {
		img.Base.Entry, args = args[1], args[2:]
	}
	b := &Boot{
		Layout: selfupdate.Layout{Root: envOr("DIDLE_ROOT", "/data/didle")},
		Image:  img,
		Policy: DefaultPolicy(),
		Clock:  realClock{},
		Log:    func(f string, a ...any) { fmt.Fprintf(os.Stderr, "didleboot: "+f+"\n", a...) },
	}
	if len(args) > 0 && args[0] == "-select" {
		st, _ := b.Layout.LoadState()
		rel, skipped := b.Select(st)
		fmt.Printf("release=%s base=%v bin=%s d2d=%s\n", rel.Version, rel.Base, rel.BinDir, rel.D2DDir)
		for _, s := range skipped {
			fmt.Printf("skipped=%s reason=%s\n", s.Version, s.Reason)
		}
		return 0
	}
	watchAddr := envOr("D2BOT_WATCH_ADDR", "127.0.0.1:8891")
	if strings.HasPrefix(watchAddr, "0.0.0.0") {
		watchAddr = "127.0.0.1" + strings.TrimPrefix(watchAddr, "0.0.0.0")
	}
	ping := filepath.Join(b.Image.Base.BinDir, "d2dping")
	b.Probe = newProbes(envOr("D2BOT_ADDR", "127.0.0.1:8420"), watchAddr, ping, envOr("D2D_SOCK", "/run/d2d/d2d.sock"))
	b.Runner = osRunner{args: args, env: os.Environ()}
	var stop atomic.Bool
	b.Stop = stop.Load
	sig := make(chan os.Signal, 2)
	signal.Notify(sig, syscall.SIGTERM, syscall.SIGINT)
	go func() { <-sig; stop.Store(true) }()
	return b.Run()
}
