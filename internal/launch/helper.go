package launch

import (
	"bufio"
	"context"
	"d2lod-bot/internal/mem"
	"fmt"
	"net"
	"path/filepath"
	"strings"
	"time"
)

// Helper manages the in-prefix input helper (cmd/d2input, built with GOOS=windows GOARCH=386) that
// listens on Config.HelperAddr.
type Helper struct {
	Cfg Config
	m   Managed
}

// NewHelper returns a Helper for cfg (already Resolved). spawn may be nil.
func NewHelper(cfg Config, spawn *Spawner) *Helper {
	// Every prefix runs its own d2input.exe: Find/Stop by image name must never see another instance's helper.
	if cfg.Instance == "" {
		cfg.Instance = mem.DefaultInstance
	}
	return &Helper{Cfg: cfg, m: Managed{
		Cfg: cfg, Spawn: spawn, Exe: cfg.HelperExe, Cwd: filepath.Dir(cfg.HelperExe),
		Image: "d2input.exe", Log: "d2input.out",
		Args: helperArgs(cfg),
	}}
}

// Ping dials the helper and sends `info`. alive is true if anything answered with a protocol line;
// windowUp is true if the reply was "ok ..." (the game window exists). "err no Diablo II window" counts
// as alive, not up. VERIFIED protocol: tools/helper.sh and cmd/d2input.
func (h *Helper) Ping(timeout time.Duration) (alive, windowUp bool, reply string, err error) {
	return pingHelper(h.Cfg.HelperAddr, timeout)
}

func pingHelper(addr string, timeout time.Duration) (alive, windowUp bool, reply string, err error) {
	c, err := net.DialTimeout("tcp", addr, timeout)
	if err != nil {
		return false, false, "", err
	}
	defer c.Close()
	c.SetDeadline(time.Now().Add(timeout))
	if _, err = fmt.Fprintln(c, "info"); err != nil {
		return false, false, "", err
	}
	line, err := bufio.NewReader(c).ReadString('\n')
	if err != nil {
		return false, false, "", err
	}
	line = strings.TrimSpace(line)
	return strings.HasPrefix(line, "ok") || strings.HasPrefix(line, "err"), strings.HasPrefix(line, "ok"), line, nil
}

// Ensure makes sure a helper is listening: if one already answers it is adopted (it may have been
// started by tools/helper.sh), otherwise it is started and waited for (readiness = TCP answers `info`).
// Note an adopted helper is NOT our descendant; that does not matter, the helper is only talked to over TCP.
func (h *Helper) Ensure(ctx context.Context, timeout time.Duration) error {
	if alive, _, _, _ := h.Ping(time.Second); alive {
		return nil
	}
	if _, running := h.m.Find(); running {
		// Process exists but is not answering yet (just started) or is wedged: give it a moment, then kill.
		if h.waitReady(ctx, 5*time.Second) == nil {
			return nil
		}
		if err := h.m.Stop(3 * time.Second); err != nil {
			return err
		}
	}
	if err := h.m.Start(ctx); err != nil {
		return err
	}
	return h.waitReady(ctx, timeout)
}

func (h *Helper) waitReady(ctx context.Context, timeout time.Duration) error {
	deadline := time.Now().Add(timeout)
	for {
		if alive, _, _, _ := h.Ping(time.Second); alive {
			return nil
		}
		if time.Now().After(deadline) {
			return fmt.Errorf("helper not answering on %s after %s", h.Cfg.HelperAddr, timeout)
		}
		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-time.After(300 * time.Millisecond):
		}
	}
}

// Stop asks the helper to quit (`quit` command) and falls back to SIGTERM/SIGKILL.
func (h *Helper) Stop() error {
	if c, err := net.DialTimeout("tcp", h.Cfg.HelperAddr, time.Second); err == nil {
		fmt.Fprintln(c, "quit")
		c.Close()
		time.Sleep(500 * time.Millisecond)
	}
	return h.m.Stop(3 * time.Second)
}

// Supervise keeps the helper alive until ctx ends: every interval it pings and restarts a dead helper
// (backoff doubles from 1 s to 30 s while restarts keep failing). onEvent may be nil.
func (h *Helper) Supervise(ctx context.Context, interval time.Duration, onEvent func(string)) {
	say := func(s string) {
		if onEvent != nil {
			onEvent(s)
		}
	}
	backoff := time.Second
	for {
		if alive, _, _, _ := h.Ping(2 * time.Second); alive {
			backoff = time.Second
		} else {
			say("helper down, restarting")
			if err := h.Ensure(ctx, 30*time.Second); err != nil {
				say("helper restart failed: " + err.Error())
				select {
				case <-ctx.Done():
					return
				case <-time.After(backoff):
				}
				if backoff < 30*time.Second {
					backoff *= 2
				}
				continue
			}
			say("helper up")
		}
		select {
		case <-ctx.Done():
			return
		case <-time.After(interval):
		}
	}
}

// helperArgs tells the helper which port to listen on when it is not the default one.
func helperArgs(cfg Config) []string {
	if cfg.HelperAddr == "" || cfg.HelperAddr == "127.0.0.1:47012" {
		return nil
	}
	return []string{"-addr", cfg.HelperAddr}
}
