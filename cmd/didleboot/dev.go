package main

import (
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strconv"
	"strings"

	"d2lod-bot/internal/selfupdate"
)

// The dev loop (TASK-192, docs/dev-update.md). Only on a DEV image (DIDLE_DEV=1, build arg DEV=1); a normal image has
// none of this and `check-now` says so.
//
//	didleboot check-now      ask the dev source, verify, download, install, then make the shim restart the bot process
//	didleboot rollback-now   go back to the previous release (or the baked base) and restart the bot process
//
// Output is one line for the script tools/dev-update.sh: `UPDATE old new`, `CURRENT old`, `ROLLBACK from to`, or
// `REJECTED reason` (exit 3) / `ERROR reason` (exit 1).

func devOn() bool { return os.Getenv("DIDLE_DEV") == "1" }

func pidFile(layout selfupdate.Layout) string { return filepath.Join(layout.Root, "boot.pid") }

// signalShim sends SIGHUP to the running shim named by boot.pid (after checking that pid is a didleboot).
func signalShim(layout selfupdate.Layout) error {
	raw, err := os.ReadFile(pidFile(layout))
	if err != nil {
		return fmt.Errorf("no running shim (%v)", err)
	}
	pid, err := strconv.Atoi(strings.TrimSpace(string(raw)))
	if err != nil || pid <= 1 {
		return errors.New("boot.pid is not a pid")
	}
	cmdline, err := os.ReadFile(fmt.Sprintf("/proc/%d/cmdline", pid))
	if err != nil || !strings.Contains(string(cmdline), "boot") {
		return fmt.Errorf("pid %d is not the boot shim", pid)
	}
	return sendHUP(pid)
}

func devEngine(img Image, layout selfupdate.Layout) (*selfupdate.Engine, string, error) {
	src, err := selfupdate.DevFromEnv(os.Getenv)
	if err != nil {
		return nil, "", err
	}
	if src == nil {
		return nil, "", errors.New("no dev source: set D2BOT_UPDATE_URL (and D2BOT_UPDATE_PUBKEY) on a DEV image")
	}
	b := &Boot{Layout: layout, Image: img}
	st, _ := layout.LoadState()
	rel, _ := b.Select(st)
	e := selfupdate.New(selfupdate.Config{
		Layout:       layout,
		Client:       src.Client("didle-updater/dev"),
		Keys:         img.Keys,
		Running:      rel.Version,
		ImageABI:     img.ABI,
		Local:        img.Local,
		CheckEnabled: selfupdate.ParseCheckMode(os.Getenv("D2BOT_UPDATE_CHECK")) != selfupdate.CheckOff,
		Shutdown:     func() error { return signalShim(layout) },
	})
	return e, rel.Version, nil
}

// devCommand runs check-now or rollback-now and returns the exit code.
func devCommand(cmd string, img Image, layout selfupdate.Layout, out io.Writer) int {
	if !devOn() {
		fmt.Fprintln(out, "ERROR this is not a DEV image (build with DEV=1, docs/dev-update.md)")
		return 1
	}
	e, running, err := devEngine(img, layout)
	if err != nil {
		fmt.Fprintln(out, "ERROR", err)
		return 1
	}
	ctx := ctxBackground()
	switch cmd {
	case "rollback-now":
		to := layout.CurrentVersion()
		st, _ := layout.LoadState()
		if err := e.Rollback(); err != nil {
			fmt.Fprintln(out, "ERROR", err)
			return 1
		}
		if st.Previous == "" {
			st.Previous = img.BaseVersion
		}
		fmt.Fprintln(out, "ROLLBACK", to, st.Previous)
		return 0
	default: // check-now
		if err := e.Check(ctx); err != nil {
			fmt.Fprintln(out, "REJECTED", oneLine(err))
			return 3
		}
		a := e.Status().Available
		if a == nil {
			fmt.Fprintln(out, "CURRENT", running)
			return 0
		}
		if a.NeedsImage {
			fmt.Fprintln(out, "ERROR release", a.Version, "needs a newer image: rebuild")
			return 1
		}
		if err := e.Download(ctx); err != nil {
			fmt.Fprintln(out, "REJECTED", oneLine(err))
			return 3
		}
		if err := e.Apply(a.Version, true); err != nil {
			fmt.Fprintln(out, "ERROR", oneLine(err))
			return 1
		}
		fmt.Fprintln(out, "UPDATE", running, a.Version)
		return 0
	}
}

func oneLine(err error) string { return strings.Join(strings.Fields(err.Error()), " ") }
