package setup

import (
	"context"
	"net/http"
	"path/filepath"
	"time"
)

// WebConfig configures the first-run wizard served by a binary's web UI.
type WebConfig struct {
	DataDir string // wizard state is DataDir/setup.json; also the default data-dir input
	Display string // X display the installer windows are shown on (viewport/noVNC), "" = none
	// Deps overrides the host effects (tests); nil is the real thing.
	Deps *Deps
	// BaseCtx bounds background runs (server lifetime); nil is Background.
	BaseCtx context.Context
}

// Web is the wizard wired for the web UI: HoldForInput is on, so a run stops before the character step
// until the user has typed a name. Mount Handler at server.Config.Setup.
type Web struct {
	W   *Wizard
	API *API
}

// NewWeb builds the wizard on DataDir/setup.json. The process that runs the character step must have
// called launch.BecomeSubreaper (the binary's main does that), and main must call MaybeProbe first.
func NewWeb(c WebConfig) (*Web, error) {
	d := c.Deps
	if d == nil {
		d = NewDeps(NewEnv())
	}
	w, err := New(filepath.Join(c.DataDir, "setup.json"), NewSteps(d))
	if err != nil {
		return nil, err
	}
	w.HoldForInput, w.Display = true, c.Display
	if w.Inputs().DataDir == "" {
		if err := w.SetInputs(Inputs{DataDir: c.DataDir}); err != nil {
			return nil, err
		}
	}
	env := d.Env
	if env == nil {
		env = NewEnv()
	}
	return &Web{W: w, API: &API{W: w, Env: env, BaseCtx: c.BaseCtx, Opts: func() Options {
		return Options{DataDir: w.Inputs().DataDir, Slots: w.Inputs().Slots}
	}}}, nil
}

// Handler is the setup API for server.Config.Setup.
func (x *Web) Handler() http.Handler { return x.API.Handler() }

// Ready reports whether every step including the character is done, i.e. a bot may start.
func (x *Web) Ready() bool { return x.W.View().Done }

// WaitReady blocks until Ready or ctx ends. The bot (managed mode) waits here, so it does not touch the
// game while the wizard is still installing or creating the character.
func (x *Web) WaitReady(ctx context.Context) error {
	t := time.NewTicker(time.Second)
	defer t.Stop()
	for !x.Ready() {
		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-t.C:
		}
	}
	return nil
}
