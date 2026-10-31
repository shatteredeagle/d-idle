package main

import (
	"context"
	"encoding/json"
	"log/slog"
	"net/http"
	"strings"
	"sync/atomic"
	"time"

	"d2lod-bot/internal/event"
	"d2lod-bot/internal/goals"
	"d2lod-bot/internal/metrics"
	"d2lod-bot/internal/server"
	"d2lod-bot/web"
)

// health is the state behind the dry mode /healthz (the other modes use the server's).
type health struct {
	started time.Time
	mode    string
	ready   atomic.Bool
	status  func() goals.Status // nil in dry mode
	game    func() bool         // nil: not applicable
}

func (h *health) handler() http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		out := map[string]any{"ok": true, "mode": h.mode, "uptimeSec": int(time.Since(h.started).Seconds()), "ready": h.ready.Load()}
		code := http.StatusOK
		if h.game != nil && !h.game() {
			out["ok"], out["game"] = false, "gone"
			code = http.StatusServiceUnavailable
		}
		if h.status != nil {
			st := h.status()
			out["state"], out["activity"] = st.State, st.Activity
			if st.State == goals.RunnerError {
				out["ok"], out["message"] = false, st.Message
				code = http.StatusServiceUnavailable
			}
		}
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(code)
		json.NewEncoder(w).Encode(out)
	})
}

// serve runs the one HTTP server (internal/server) with a single instance until ctx ends, then shuts it down
// gracefully. The flat /api/v1 routes keep working through the server's compat shim; /metrics is mounted
// behind the same token.
func serve(ctx context.Context, log *slog.Logger, o options, inst *server.Instance, reg *metrics.Registry, version string) error {
	cfg := server.Config{
		Addr: o.addr, Token: o.token, AllowedHosts: splitNonEmpty(o.allowHost), Assets: web.FS, Version: version,
		Instances: &server.StaticRegistry{}, SoakDir: o.soakDir,
	}
	if err := cfg.Instances.(*server.StaticRegistry).Add(inst); err != nil {
		return err
	}
	if reg != nil {
		cfg.Mounts = []server.Mount{{Path: "/metrics/", Handler: reg.Handler()}}
	}
	srv, err := server.New(cfg)
	if err != nil {
		return err
	}
	ln, err := srv.Listen()
	if err != nil {
		return err
	}
	log.Info("listening", "addr", ln.Addr().String(), "dashboard", "http://"+ln.Addr().String()+"/", "instance", inst.ID)
	return srv.Serve(ctx, ln)
}

// instanceID is the server-side instance id: D2BOT_INSTANCE / --instance, "main" by default.
func instanceID(o options) string {
	if o.instance == "" {
		return "main"
	}
	return o.instance
}

// newRegistry returns the metrics registry with the runtime gauges registered.
func newRegistry() *metrics.Registry {
	r := metrics.NewRegistry()
	r.RegisterRuntime()
	return r
}

// busBridge forwards the manager's own events (goal_*, control, info) onto the bus so SSE and the fleet stream see
// them; runner events are already published by the bot and must not be duplicated. It stops when ctx ends.
func busBridge(ctx context.Context, m *goals.Manager, bus *event.Bus) {
	ch, cancel := m.Subscribe()
	defer cancel()
	for {
		select {
		case <-ctx.Done():
			return
		case ev, ok := <-ch:
			if !ok {
				return
			}
			switch ev.Type {
			case goals.EvKill, goals.EvDrop, goals.EvDeath, goals.EvLevelUp, goals.EvAreaChange, goals.EvPotion, goals.EvError:
				continue
			}
			ev.Seq = 0
			bus.Publish(event.FromGoals(ev, event.SrcManager))
		}
	}
}

func splitNonEmpty(s string) []string {
	var out []string
	for _, p := range strings.Split(s, ",") {
		if p = strings.TrimSpace(p); p != "" {
			out = append(out, p)
		}
	}
	return out
}
