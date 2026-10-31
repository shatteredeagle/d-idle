package sentinel

import (
	"encoding/json"
	"net/http"
	"time"

	"d2lod-bot/internal/server"
	"d2lod-bot/internal/status"
)

// NewServer builds the sentinel's HTTP side on internal/server: /api/v1/sentinel, /live/ (the hub with the sentinel
// panel), /healthz, /readyz, plus the deprecated /api/sentinel alias (kept one release) as an exact mount, so it passes
// the same token and Host checks. A non-loopback addr without a token is refused here.
// The caller runs hub.Run, serves the handler and calls Server.Close at the end.
func (r *Runner) NewServer(hub *status.Hub, addr, token string, allowHosts []string) (*server.Server, http.Handler, error) {
	if hub != nil {
		hub.Register(server.SentinelPanel(func() any { return r.Snapshot() }))
	}
	old := http.HandlerFunc(func(w http.ResponseWriter, req *http.Request) {
		if req.Method != http.MethodGet && req.Method != http.MethodHead {
			http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
			return
		}
		w.Header().Set("Deprecation", "true")
		w.Header().Set("Link", `</api/v1/sentinel>; rel="successor-version"`)
		w.Header().Set("Content-Type", "application/json; charset=utf-8")
		w.Header().Set("Cache-Control", "no-store")
		w.Header().Set("Access-Control-Allow-Origin", "*")
		json.NewEncoder(w).Encode(r.Snapshot())
	})
	srv, err := server.New(server.Config{
		Addr: addr, Token: token, AllowedHosts: allowHosts, Hub: hub,
		Sentinel: func() any { return r.Snapshot() },
		Mounts:   []server.Mount{{Path: "/api/sentinel", Exact: true, Handler: old}},
		Version:  "d2sentinel",
	})
	if err != nil {
		return nil, nil, err
	}
	return srv, srv.Handler(), nil
}

// Mux is the pre-server standalone mux (tests, embedding); d2sentinel itself uses NewServer. It serves both /api/v1/sentinel and
// the deprecated /api/sentinel.
func (r *Runner) Mux(hub *status.Hub) *http.ServeMux {
	mux := http.NewServeMux()
	snap := func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.Header().Set("Access-Control-Allow-Origin", "*")
		json.NewEncoder(w).Encode(r.Snapshot())
	}
	mux.HandleFunc("/api/sentinel", snap)
	mux.HandleFunc("/api/v1/sentinel", snap)
	mux.HandleFunc("/healthz", func(w http.ResponseWriter, _ *http.Request) { w.Write([]byte("ok\n")) })
	if hub != nil {
		hub.Register(status.Panel{ID: "sentinel", Title: "Sentinel", Order: 5, Every: 2 * time.Second,
			Snapshot: func() any { return r.Snapshot() }})
		hub.Mount(mux, "/live/")
	}
	return mux
}
