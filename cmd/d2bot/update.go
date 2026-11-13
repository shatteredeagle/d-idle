package main

import (
	"context"
	"crypto/ed25519"
	"fmt"
	"log/slog"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"d2lod-bot/internal/selfupdate"
	"d2lod-bot/internal/watch"
)

// The in-app updater (TASK-122, docs/self-update-release.md). It exists only where an update root is present (the
// container's /data/didle, or D2BOT_UPDATE_DIR) and a trusted public key is baked in; anywhere else the watch
// screen shows no update UI.
//
// Env: D2BOT_UPDATE_CHECK=off|0|false|no switches every outbound call off (the inbox still works), manual keeps the
// buttons but starts no daily check (internal/selfupdate/dev.go);
// D2BOT_UPDATE_FEED (https feed of your own, keys stay the baked ones), D2BOT_UPDATE_DIR (default /data/didle), D2BOT_UPDATE_KEYS (default /opt/didle/pubkeys, *.pub), D2BOT_IMAGE_ABI.
const (
	defaultUpdateDir  = "/data/didle"
	defaultUpdateKeys = "/opt/didle/pubkeys"
)

func updateCheckMode() selfupdate.CheckMode {
	return selfupdate.ParseCheckMode(os.Getenv("D2BOT_UPDATE_CHECK"))
}

// newUpdater builds the engine, or returns nil when this is not an updatable install.
func newUpdater(shutdown func() error, idleSafe func() bool) *selfupdate.Engine {
	root := envOr("D2BOT_UPDATE_DIR", defaultUpdateDir)
	if st, err := os.Stat(root); err != nil || !st.IsDir() {
		return nil
	}
	keys := loadPubKeys(envOr("D2BOT_UPDATE_KEYS", defaultUpdateKeys))
	client := &selfupdate.Client{UserAgent: "didle-updater/" + version}
	devSource := false
	// DEV images only (DIDLE_DEV=1): a LAN source and one more trusted key (docs/dev-update.md)
	if dev, err := selfupdate.DevFromEnv(os.Getenv); err != nil {
		slog.Warn("update: dev source ignored", "err", err)
	} else if dev != nil {
		keys, client, devSource = append(keys, dev.Keys...), dev.Client("didle-updater/"+version), true
	}
	// Any image: an optional https feed of the user's own. Replaces the GitHub host, adds no key (internal/selfupdate/feed.go).
	if feed, err := selfupdate.FeedFromEnv(os.Getenv); err != nil {
		slog.Warn("update: feed ignored", "err", err)
	} else if feed != nil && !devSource {
		client = feed.Client("didle-updater/" + version)
	}
	if len(keys) == 0 {
		return nil
	}
	abi, _ := strconv.Atoi(os.Getenv("D2BOT_IMAGE_ABI"))
	return selfupdate.New(updaterCfg{
		Layout:       selfupdate.Layout{Root: root},
		Client:       client,
		Keys:         keys,
		Running:      version,
		ImageABI:     abi,
		Local:        selfupdate.BuildCompat,
		CheckEnabled: updateCheckMode() != selfupdate.CheckOff,
		Shutdown:     shutdown,
	}.withGuards(root, idleSafe))
}

// updaterGuards are the three safety hooks of the engine; nil in any of them silently skips that check
// (docs/wiring.md finding 6), so TestUpdaterGuardsAreSet pins them.
type updaterCfg selfupdate.Config

func (c updaterCfg) withGuards(root string, idleSafe func() bool) selfupdate.Config {
	c.IdleSafe, c.SelfTest = idleSafe, releaseSelfTest
	c.DiskFree = func() (uint64, error) { return diskFreeBytes(root) }
	return selfupdate.Config(c)
}

// releaseSelfTest runs the unpacked release's own d2bot with --version before it can become current, so a binary that
// does not start on this image (wrong arch, missing library, truncated file) is refused while the old release still
// runs. A release without bin/d2bot (data-only) has nothing to run. The boot shim's crash rollback stays the second net.
func releaseSelfTest(dir string) error {
	bin := filepath.Join(dir, "bin", "d2bot")
	if _, err := os.Stat(bin); err != nil {
		if os.IsNotExist(err) {
			return nil
		}
		return err
	}
	ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer cancel()
	out, err := exec.CommandContext(ctx, bin, "--version").Output()
	if err != nil {
		return fmt.Errorf("d2bot --version: %w", err)
	}
	if !strings.HasPrefix(string(out), "d2bot ") {
		return fmt.Errorf("d2bot --version printed %q", strings.TrimSpace(string(out)))
	}
	return nil
}

func loadPubKeys(dir string) []ed25519.PublicKey {
	files, _ := filepath.Glob(filepath.Join(dir, "*.pub"))
	var keys []ed25519.PublicKey
	for _, f := range files {
		b, err := os.ReadFile(f)
		if err != nil {
			continue
		}
		if k, err := selfupdate.DecodePublicKey(b); err == nil {
			keys = append(keys, k)
		}
	}
	return keys
}

// startUpdater returns the Updater for the watch screen (nil: none) and starts the daily version check. Apply asks
// the process to stop with SIGTERM, the same path `docker stop` takes (hero saved, d2d quit in managed mode); the
// container's restart policy then brings the new release up.
func startUpdater(ctx context.Context, idleSafe func() bool) watch.Updater {
	e := newUpdater(signalSelfTerm, idleSafe)
	if e == nil {
		return nil
	}
	if updateCheckMode() == selfupdate.CheckOn { // manual: the buttons and check-now ask, nothing runs by itself
		go e.Run(ctx, 2*time.Minute)
	}
	return e
}
