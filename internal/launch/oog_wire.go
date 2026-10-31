package launch

import (
	"context"
	"fmt"
	"os/exec"
	"syscall"
	"time"
)

// Wire builds a Keeper for the real system. newInput returns the menu input (an input.OOG adapter that
// also has TypeChar); the launch package does not import internal/input. muteCmd, if set, is run after
// every launch (tools/mute-d2.sh), its failure is only logged.
func Wire(cfg Config, g *Game, h *Helper, newInput func() Input, muteCmd []string, ch CharSpec, log func(string)) *Keeper {
	k := &Keeper{Mon: &Monitor{P: DefaultProbes(g)}, Char: ch, Log: log}
	k.Kill = func() error {
		if ps, err := ListProcs(); err == nil {
			for _, p := range FindImage(ps, "blizzarderror.exe") {
				_ = syscall.Kill(p.PID, syscall.SIGKILL)
			}
		}
		return g.Shutdown(5 * time.Second)
	}
	k.Launch = func(ctx context.Context) error {
		_, err := g.Launch(ctx, 60*time.Second)
		return err
	}
	k.Ready = func(ctx context.Context) (*Flow, error) {
		if err := h.Ensure(ctx, 30*time.Second); err != nil {
			return nil, fmt.Errorf("helper: %w", err)
		}
		if len(muteCmd) > 0 {
			go func() { // the audio stream appears a few seconds after the process
				if out, err := exec.Command(muteCmd[0], muteCmd[1:]...).CombinedOutput(); err != nil && log != nil {
					log(fmt.Sprintf("mute: %v %s", err, out))
				}
			}()
		}
		deadline := time.Now().Add(40 * time.Second)
		for { // window up (the helper finds it on every command)
			if _, up, _, _ := h.Ping(time.Second); up {
				break
			}
			if time.Now().After(deadline) {
				return nil, fmt.Errorf("%w: game window never appeared", ErrTimeout)
			}
			select {
			case <-ctx.Done():
				return nil, ctx.Err()
			case <-time.After(500 * time.Millisecond):
			}
		}
		var lastErr error
		for time.Now().Before(deadline.Add(20 * time.Second)) {
			_, _, src, err := g.Attach()
			if err == nil {
				return &Flow{Mem: src, In: newInput(), Log: log}, nil
			}
			lastErr = err
			time.Sleep(500 * time.Millisecond)
		}
		return nil, fmt.Errorf("attach: %w", lastErr)
	}
	return k
}
