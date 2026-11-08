package main

import (
	"context"
	"crypto/ed25519"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"syscall"
	"time"

	"d2lod-bot/internal/selfupdate"
	"d2lod-bot/internal/watch"
)

// The in-app updater (TASK-122, docs/self-update-release.md). It exists only where an update root is present (the
// container's /data/didle, or D2BOT_UPDATE_DIR) and a trusted public key is baked in; anywhere else the watch
// screen shows no update UI.
//
// Env: D2BOT_UPDATE_CHECK=off|0|false|no switches every outbound call off (the inbox still works);
// D2BOT_UPDATE_DIR (default /data/didle), D2BOT_UPDATE_KEYS (default /opt/didle/pubkeys, *.pub), D2BOT_IMAGE_ABI.
const (
	defaultUpdateDir  = "/data/didle"
	defaultUpdateKeys = "/opt/didle/pubkeys"
)

func updateCheckEnabled() bool {
	switch strings.ToLower(os.Getenv("D2BOT_UPDATE_CHECK")) {
	case "off", "0", "false", "no":
		return false
	}
	return true
}

// newUpdater builds the engine, or returns nil when this is not an updatable install.
func newUpdater(shutdown func() error) *selfupdate.Engine {
	root := envOr("D2BOT_UPDATE_DIR", defaultUpdateDir)
	if st, err := os.Stat(root); err != nil || !st.IsDir() {
		return nil
	}
	keys := loadPubKeys(envOr("D2BOT_UPDATE_KEYS", defaultUpdateKeys))
	if len(keys) == 0 {
		return nil
	}
	abi, _ := strconv.Atoi(os.Getenv("D2BOT_IMAGE_ABI"))
	return selfupdate.New(selfupdate.Config{
		Layout:       selfupdate.Layout{Root: root},
		Client:       &selfupdate.Client{UserAgent: "didle-updater/" + version},
		Keys:         keys,
		Running:      version,
		ImageABI:     abi,
		Local:        selfupdate.BuildCompat,
		CheckEnabled: updateCheckEnabled(),
		Shutdown:     shutdown,
	})
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
func startUpdater(ctx context.Context) watch.Updater {
	e := newUpdater(func() error { return syscall.Kill(os.Getpid(), syscall.SIGTERM) })
	if e == nil {
		return nil
	}
	go e.Run(ctx, 2*time.Minute)
	return e
}
