// Package server is the one HTTP server of a host. It replaces the three separate servers (d2bot API on 8420,
// d2watch on 8890, d2sentinel on 8895) with one listener, one token and one route tree:
//
//	GET  /healthz, /readyz                              open: process alive / all instances ready
//	GET  /api/v1/instances                              instance summaries
//	     /api/v1/instances/{id}/{status,goals,settings,stats,catalog,control/*,...}   internal/api routes
//	GET  /api/v1/instances/{id}/events[?since=&limit=]  event envelopes (internal/event), oldest first
//	GET  /api/v1/instances/{id}/events/stream           SSE: log (envelope, id=seq), status, goals
//	GET  /api/v1/instances/{id}/{health,deaths,state}   Instance callbacks
//	     /api/v1/instances/{id}/<mount>/...             Instance.Mounts (viewport, ...), prefix stripped
//	GET  /api/v1/fleet, /api/v1/fleet/events/stream     all instances at once; every event carries "instance"
//	     /api/v1/fleet-admin/...                        Config.FleetAdmin (fleet.Handler), prefix stripped
//	GET  /api/v1/sentinel                               Config.Sentinel snapshot
//	GET  /api/v1/soak                                   newest soak run alerts + status (Config.SoakDir)
//	GET  /instances/{id}/live/                          the live display of one instance (status.Hub)
//	GET  /live/                                         Config.Hub, else the first instance's hub
//	     /api/v1/<anything else>                        compat shim: the first instance, old flat JSON shapes
//	     /api/setup/...                                 Config.Setup (setup.API.Handler), prefix stripped, behind the token
//	GET  /setup.html                                    the first-run wizard page (part of Config.Assets, web/setup.html)
//	     Config.Mounts                                  other handlers (d2watch, sentinel pages), prefix stripped;
//	                                                    Mount.Exact: one exact path, request unchanged
//	     /                                              Config.Assets (the dashboard)
//
// The compat shim keeps docs/api.md working unchanged, including /api/v1/events (flat goals.Event) and
// /api/v1/bot/{deaths,health,state}. New clients use the instance routes and the event envelope.
package server

import (
	"context"
	"encoding/json"
	"errors"
	"io/fs"
	"net"
	"net/http"
	"regexp"
	"sort"
	"strings"
	"sync"
	"time"

	"d2lod-bot/internal/api"
	"d2lod-bot/internal/event"
	"d2lod-bot/internal/goals"
	"d2lod-bot/internal/status"
)

const (
	apiBase   = "/api/v1"
	instBase  = apiBase + "/instances"
	fleetBase = apiBase + "/fleet"
)

var idRE = regexp.MustCompile(`^[a-z0-9-]{1,16}$`)

// Instance is everything the server needs from one bot instance. Only ID and Manager are required.
type Instance struct {
	ID    string
	Title string // display name, for example the character
	// Manager is the goal manager of this instance (status, goals, settings, stats).
	Manager *goals.Manager
	// Bus feeds events/stream. Nil: events are derived from the manager's own flat log.
	Bus *event.Bus
	// Hub is the live display (status.Hub); the owner runs hub.Run.
	Hub *status.Hub
	// Health returns the /health body and whether the instance is healthy (false answers 503); bot.Runner.Health.
	// Nil: derived from the runner state.
	Health func() (body any, ok bool)
	// Ready reports whether the instance is in a known state (for /readyz). Nil: always ready.
	Ready func() bool
	// Deaths is the /deaths body (bot.Runner.DeathReport). Nil: 404.
	Deaths func() any
	// State is the /state body. Nil: derived from the manager snapshot.
	State func() any
	// Mounts are extra handlers under /api/v1/instances/{id}/<name>/ (key "viewport", ...). The prefix is
	// stripped; the server's guard has already authenticated the request.
	Mounts map[string]http.Handler
}

// Registry lists the instances. The first is the default instance of the compat shim.
type Registry interface {
	List() []*Instance
	Get(id string) *Instance
}

// StaticRegistry is a Registry that is filled by Add.
type StaticRegistry struct {
	mu   sync.RWMutex
	list []*Instance
}

// Add registers an instance; ids must match [a-z0-9-]{1,16} and be unique.
func (r *StaticRegistry) Add(i *Instance) error {
	if i == nil || i.Manager == nil {
		return errors.New("server: instance needs a Manager")
	}
	if !idRE.MatchString(i.ID) {
		return errors.New("server: instance id must match [a-z0-9-]{1,16}: " + i.ID)
	}
	r.mu.Lock()
	defer r.mu.Unlock()
	for _, o := range r.list {
		if o.ID == i.ID {
			return errors.New("server: duplicate instance id " + i.ID)
		}
	}
	r.list = append(r.list, i)
	return nil
}

func (r *StaticRegistry) List() []*Instance {
	r.mu.RLock()
	defer r.mu.RUnlock()
	return append([]*Instance(nil), r.list...)
}

func (r *StaticRegistry) Get(id string) *Instance {
	r.mu.RLock()
	defer r.mu.RUnlock()
	for _, i := range r.list {
		if i.ID == id {
			return i
		}
	}
	return nil
}

// Mount serves Handler under Path (for example "/watch/"), prefix stripped. Path must not be "/".
// With Exact set it serves only that one path (for example "/api/sentinel"), the request unchanged.
type Mount struct {
	Path    string
	Handler http.Handler
	Exact   bool
}

// Slot is one fleet slot (internal/fleet.View) as the server shows it: Config.Slots feeds /api/v1/fleet.
type Slot struct {
	ID    string `json:"id"`
	Title string `json:"title,omitempty"`
	State string `json:"state"`           // starting, running, restarting, draining, failed
	Error string `json:"error,omitempty"` // last error
}

type Config struct {
	// Addr defaults to 127.0.0.1:8420. A non-loopback address requires Token.
	Addr string
	// Token: one token for everything (Bearer, X-Token, d2token cookie, ?token=). See api.Security.
	Token        string
	AllowedHosts []string
	Instances    Registry
	// Assets is the dashboard served at / (index.html at its root). Nil: API only.
	Assets fs.FS
	// Hub is a host-level display served at /live/ (the sentinel panel). Nil: the first instance's Hub.
	Hub *status.Hub
	// Sentinel returns the sentinel snapshot for /api/v1/sentinel and the fleet view. Nil: 404 / omitted.
	Sentinel func() any
	Mounts   []Mount
	// FleetAdmin is the fleet admin API (fleet.Handler(f, "")), mounted at /api/v1/fleet-admin/ behind the
	// server's own token check. Nil: not served.
	FleetAdmin http.Handler
	// Slots lists the fleet slots for /api/v1/fleet: each slot without a registered instance appears in
	// "instances" (summary with "slot" set) and all of them in "slots". Nil: omitted.
	Slots func() []Slot
	// Remotes lists bots that run as their own process (fleet slots): shown as instances, proxied, see Remote.
	// Called per request, so it must be cheap and follow restarts. Nil: none.
	Remotes func() []Remote
	// Setup is the first-run wizard API (setup.API.Handler()), mounted at /api/setup/. Nil: not served.
	Setup http.Handler
	// SoakDir is the directory holding soak run directories (~/d2lod-live/soak); /api/v1/soak serves the newest
	// run's alerts.json and status.txt, read-only. Empty: off (404).
	SoakDir string
	Version string
}

type Server struct {
	cfg  Config
	sec  api.Security
	root http.Handler

	mu    sync.Mutex
	apis  map[string]*api.Server // per instance id
	rings map[string]*ring
	flat  *api.Server // legacy flat handler, rebuilt when the first instance changes
	flatM *goals.Manager
	start time.Time
	rc    remoteCache
}

func New(cfg Config) (*Server, error) {
	if cfg.Instances == nil {
		cfg.Instances = &StaticRegistry{}
	}
	if cfg.Addr == "" {
		cfg.Addr = "127.0.0.1:8420"
	}
	sec := api.Security{Addr: cfg.Addr, Token: cfg.Token, AllowedHosts: cfg.AllowedHosts}
	if err := sec.Validate(); err != nil {
		return nil, errors.New(strings.Replace(err.Error(), "api:", "server:", 1))
	}
	for _, m := range cfg.Mounts {
		if m.Handler == nil || !strings.HasPrefix(m.Path, "/") || strings.Trim(m.Path, "/") == "" {
			return nil, errors.New("server: bad mount " + m.Path)
		}
	}
	if cfg.FleetAdmin != nil {
		cfg.Mounts = append(append([]Mount(nil), cfg.Mounts...), Mount{Path: fleetBase + "-admin/", Handler: cfg.FleetAdmin})
	}
	if cfg.Setup != nil {
		cfg.Mounts = append(append([]Mount(nil), cfg.Mounts...), Mount{Path: "/api/setup/", Handler: cfg.Setup})
	}
	s := &Server{cfg: cfg, sec: sec, apis: map[string]*api.Server{}, rings: map[string]*ring{}, start: time.Now()}
	s.root = s.top()
	for _, i := range cfg.Instances.List() {
		s.ringFor(i) // start recording history now, not at the first client
	}
	return s, nil
}

func (s *Server) Handler() http.Handler { return s.root }

// Listen opens the configured address (port 0 picks a free one).
func (s *Server) Listen() (net.Listener, error) { return net.Listen("tcp", s.cfg.Addr) }

// Serve runs until ctx is done, then shuts down gracefully and closes SSE streams and event rings.
func (s *Server) Serve(ctx context.Context, ln net.Listener) error {
	base, stop := context.WithCancel(context.Background())
	defer stop()
	srv := &http.Server{
		Handler:           s.root,
		ReadHeaderTimeout: 10 * time.Second,
		IdleTimeout:       2 * time.Minute,
		BaseContext:       func(net.Listener) context.Context { return base },
	}
	errc := make(chan error, 1)
	go func() { errc <- srv.Serve(ln) }()
	var err error
	select {
	case err = <-errc:
	case <-ctx.Done():
		stop()
		sctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
		defer cancel()
		if e := srv.Shutdown(sctx); e != nil {
			srv.Close()
		}
	}
	s.Close()
	return err
}

// Close releases the event rings.
func (s *Server) Close() {
	s.mu.Lock()
	defer s.mu.Unlock()
	for id, r := range s.rings {
		r.close()
		delete(s.rings, id)
	}
}

// ---- dispatch ------------------------------------------------------------------------------------

type errBody struct {
	Error string `json:"error"`
	Code  string `json:"code"`
}

func writeJSON(w http.ResponseWriter, status int, v any) {
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.Header().Set("Cache-Control", "no-store")
	w.WriteHeader(status)
	json.NewEncoder(w).Encode(v)
}

func notFound(w http.ResponseWriter, msg string) {
	writeJSON(w, http.StatusNotFound, errBody{msg, "not_found"})
}

func getOnly(w http.ResponseWriter, r *http.Request) bool {
	if r.Method == http.MethodGet || r.Method == http.MethodHead {
		return true
	}
	w.Header().Set("Allow", "GET, HEAD")
	writeJSON(w, http.StatusMethodNotAllowed, errBody{"method not allowed", "method_not_allowed"})
	return false
}

// top answers /healthz and /readyz without a token (process supervision), everything else is guarded.
func (s *Server) top() http.Handler {
	guarded := s.sec.Wrap(http.HandlerFunc(s.route))
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/healthz":
			w.Header().Set("Cache-Control", "no-store")
			writeJSON(w, 200, map[string]any{"ok": true, "uptimeSec": int(time.Since(s.start).Seconds()), "version": s.cfg.Version})
		case "/readyz":
			s.readyz(w)
		default:
			guarded.ServeHTTP(w, r)
		}
	})
}

func (s *Server) readyz(w http.ResponseWriter) {
	out := map[string]bool{}
	all := true
	for _, i := range s.cfg.Instances.List() {
		ok := i.Ready == nil || i.Ready()
		out[i.ID] = ok
		all = all && ok
	}
	code := 200
	if !all {
		code = http.StatusServiceUnavailable
	}
	writeJSON(w, code, map[string]any{"ready": all, "instances": out})
}

func (s *Server) route(w http.ResponseWriter, r *http.Request) {
	p := r.URL.Path
	for _, m := range s.mountsByLength() {
		if m.Exact {
			if p == m.Path {
				m.Handler.ServeHTTP(w, r)
				return
			}
			continue
		}
		if strings.HasPrefix(p, m.Path) || p+"/" == m.Path {
			if p+"/" == m.Path {
				http.Redirect(w, r, m.Path, http.StatusMovedPermanently)
				return
			}
			http.StripPrefix(strings.TrimSuffix(m.Path, "/"), m.Handler).ServeHTTP(w, r)
			return
		}
	}
	switch {
	case p == instBase || p == instBase+"/":
		if getOnly(w, r) {
			s.listInstances(r.Context(), w)
		}
	case strings.HasPrefix(p, instBase+"/"):
		id, rest, _ := strings.Cut(strings.TrimPrefix(p, instBase+"/"), "/")
		s.instanceRoute(w, r, id, rest)
	case p == fleetBase:
		if getOnly(w, r) {
			writeJSON(w, 200, s.fleet(r.Context()))
		}
	case p == fleetBase+"/events/stream":
		if getOnly(w, r) {
			s.fleetStream(w, r)
		}
	case p == apiBase+"/sentinel":
		if !getOnly(w, r) {
			return
		}
		if s.cfg.Sentinel == nil {
			notFound(w, "no sentinel on this server")
			return
		}
		w.Header().Set("Access-Control-Allow-Origin", "*")
		writeJSON(w, 200, s.cfg.Sentinel())
	case p == apiBase+"/soak":
		if !getOnly(w, r) {
			return
		}
		if s.cfg.SoakDir == "" {
			notFound(w, "soak strip is off on this server")
			return
		}
		v, ok := LoadSoak(s.cfg.SoakDir, time.Now())
		if !ok {
			notFound(w, "no soak run found")
			return
		}
		writeJSON(w, 200, v)
	case strings.HasPrefix(p, "/instances/"):
		id, rest, _ := strings.Cut(strings.TrimPrefix(p, "/instances/"), "/")
		if rest != "live" && !strings.HasPrefix(rest, "live/") {
			http.NotFound(w, r)
			return
		}
		i := s.cfg.Instances.Get(id)
		if i == nil || i.Hub == nil {
			http.NotFound(w, r)
			return
		}
		serveHub(w, r, i.Hub, "/instances/"+id+"/live")
	case p == "/live" || strings.HasPrefix(p, "/live/"):
		h := s.cfg.Hub
		if h == nil {
			if f := s.first(); f != nil {
				h = f.Hub
			}
		}
		if h == nil {
			http.NotFound(w, r)
			return
		}
		serveHub(w, r, h, "/live")
	case strings.HasPrefix(p, apiBase+"/"):
		s.flatRoute(w, r)
	case s.cfg.Assets != nil:
		w.Header().Set("Cache-Control", "no-cache")
		http.FileServerFS(s.cfg.Assets).ServeHTTP(w, r)
	default:
		http.NotFound(w, r)
	}
}

func (s *Server) mountsByLength() []Mount {
	ms := make([]Mount, 0, len(s.cfg.Mounts))
	for _, m := range s.cfg.Mounts {
		if !m.Exact && !strings.HasSuffix(m.Path, "/") {
			m.Path += "/"
		}
		ms = append(ms, m)
	}
	sort.Slice(ms, func(i, j int) bool { return len(ms[i].Path) > len(ms[j].Path) })
	return ms
}

func serveHub(w http.ResponseWriter, r *http.Request, h *status.Hub, prefix string) {
	if r.URL.Path == prefix {
		http.Redirect(w, r, prefix+"/", http.StatusMovedPermanently)
		return
	}
	http.StripPrefix(prefix, h.Handler()).ServeHTTP(w, r)
}

func (s *Server) first() *Instance {
	if l := s.cfg.Instances.List(); len(l) > 0 {
		return l[0]
	}
	return nil
}

// apiFor returns the (cached) per-instance route handler.
func (s *Server) apiFor(i *Instance) *api.Server {
	s.mu.Lock()
	defer s.mu.Unlock()
	if a := s.apis[i.ID]; a != nil {
		return a
	}
	a := api.NewInstance(i.Manager, instBase+"/"+i.ID, s.cfg.Version)
	s.apis[i.ID] = a
	return a
}

func (s *Server) instanceRoute(w http.ResponseWriter, r *http.Request, id, rest string) {
	i := s.cfg.Instances.Get(id)
	if i == nil {
		if rm, ok := s.remote(id); ok {
			s.remoteRoute(w, r, rm, rest)
			return
		}
		notFound(w, "no such instance")
		return
	}
	name, _, _ := strings.Cut(rest, "/")
	if h := i.Mounts[name]; h != nil && name != "" {
		http.StripPrefix(instBase+"/"+id+"/"+name, h).ServeHTTP(w, r)
		return
	}
	switch rest {
	case "":
		if getOnly(w, r) {
			writeJSON(w, 200, s.summary(i))
		}
	case "events":
		if getOnly(w, r) {
			s.events(w, r, i)
		}
	case "events/stream":
		if getOnly(w, r) {
			s.stream(w, r, i)
		}
	case "health", "deaths", "state":
		if getOnly(w, r) {
			s.instanceData(w, i, rest)
		}
	default:
		s.apiFor(i).Routes().ServeHTTP(w, r)
	}
}

func (s *Server) instanceData(w http.ResponseWriter, i *Instance, what string) {
	switch what {
	case "health":
		body, ok := s.health(i)
		code := 200
		if !ok {
			code = http.StatusServiceUnavailable
		}
		writeJSON(w, code, body)
	case "deaths":
		if i.Deaths == nil {
			notFound(w, "no death log for this instance")
			return
		}
		writeJSON(w, 200, i.Deaths())
	case "state":
		if i.State != nil {
			writeJSON(w, 200, i.State())
			return
		}
		sn := i.Manager.Snapshot()
		writeJSON(w, 200, map[string]any{"state": sn.Runner.State, "status": sn.Runner})
	}
}

func (s *Server) health(i *Instance) (any, bool) {
	if i.Health != nil {
		return i.Health()
	}
	rs := i.Manager.Snapshot().Runner
	ok := rs.State != goals.RunnerError
	return map[string]any{"ok": ok, "state": rs.State, "activity": rs.Activity, "message": rs.Message}, ok
}

// flatRoute is the compat shim: /api/v1/<x> behaves like the first instance's /api/v1/instances/<id>/<x>,
// with the old flat shapes (events are goals.Event, not envelopes). The first instance's Mounts are reachable
// there too: /api/v1/knobs, /api/v1/upgrades, ...
func (s *Server) flatRoute(w http.ResponseWriter, r *http.Request) {
	f := s.first()
	if f == nil {
		notFound(w, "no instances")
		return
	}
	rest := strings.TrimPrefix(r.URL.Path, apiBase+"/")
	if name, _, _ := strings.Cut(rest, "/"); name != "" && f.Mounts[name] != nil {
		http.StripPrefix(apiBase+"/"+name, f.Mounts[name]).ServeHTTP(w, r)
		return
	}
	if what, ok := strings.CutPrefix(rest, "bot/"); ok && (what == "health" || what == "deaths" || what == "state") {
		if getOnly(w, r) {
			s.instanceData(w, f, what)
		}
		return
	}
	s.mu.Lock()
	if s.flat == nil || s.flatM != f.Manager {
		s.flat, s.flatM = api.NewInstance(f.Manager, apiBase, s.cfg.Version), f.Manager
	}
	h := s.flat
	s.mu.Unlock()
	h.Routes().ServeHTTP(w, r)
}

// ---- summaries -----------------------------------------------------------------------------------

type summary struct {
	ID        string         `json:"id"`
	Title     string         `json:"title,omitempty"`
	Mode      goals.Mode     `json:"mode"`
	Runner    goals.Status   `json:"runner"`
	ActiveID  string         `json:"activeGoalId,omitempty"`
	XPPerHour float64        `json:"xpPerHour"`
	Totals    goals.Totals   `json:"totals"`
	OK        bool           `json:"ok"`
	Ready     bool           `json:"ready"`
	LastSeq   uint64         `json:"lastSeq"`
	Slot      string         `json:"slot,omitempty"`   // fleet slot state; set when the slot has no instance here yet
	Live      bool           `json:"live"`             // has a /instances/{id}/live/ display
	Mounts    []string       `json:"mounts,omitempty"` // names under /instances/{id}/
	Now       time.Time      `json:"now"`
	Extra     map[string]any `json:"-"`
}

func (s *Server) summary(i *Instance) summary {
	sn := i.Manager.Snapshot()
	_, ok := s.health(i)
	out := summary{ID: i.ID, Title: i.Title, Mode: sn.Mode, Runner: sn.Runner, ActiveID: sn.ActiveGoalID, XPPerHour: sn.XPPerHour,
		Totals: sn.Totals, OK: ok, Ready: i.Ready == nil || i.Ready(), LastSeq: sn.LastSeq, Live: i.Hub != nil, Now: sn.Now}
	for n := range i.Mounts {
		out.Mounts = append(out.Mounts, n)
	}
	sort.Strings(out.Mounts)
	return out
}

func (s *Server) listInstances(ctx context.Context, w http.ResponseWriter) {
	out := []summary{}
	for _, i := range s.cfg.Instances.List() {
		out = append(out, s.summary(i))
	}
	for _, sm := range s.remoteSummaries(ctx) {
		if s.cfg.Instances.Get(sm.ID) == nil {
			out = append(out, sm)
		}
	}
	writeJSON(w, 200, out)
}

type fleetBody struct {
	Now       time.Time `json:"now"`
	Instances []summary `json:"instances"`
	Sentinel  any       `json:"sentinel,omitempty"`
	// Failsafe is true when any instance is unhealthy; the dashboard shows the red banner from it.
	Failsafe bool   `json:"failsafe"`
	Slots    []Slot `json:"slots,omitempty"`
}

func (s *Server) fleet(ctx context.Context) fleetBody {
	f := fleetBody{Now: time.Now(), Instances: []summary{}}
	for _, i := range s.cfg.Instances.List() {
		sm := s.summary(i)
		f.Failsafe = f.Failsafe || !sm.OK
		f.Instances = append(f.Instances, sm)
	}
	have := map[string]bool{}
	for _, sm := range s.remoteSummaries(ctx) {
		if s.cfg.Instances.Get(sm.ID) != nil {
			continue
		}
		have[sm.ID] = true
		f.Failsafe = f.Failsafe || !sm.OK
		f.Instances = append(f.Instances, sm)
	}
	if s.cfg.Slots != nil {
		f.Slots = s.cfg.Slots()
		for _, sl := range f.Slots {
			if s.cfg.Instances.Get(sl.ID) != nil || have[sl.ID] {
				continue
			}
			bad := sl.State == "failed"
			f.Failsafe = f.Failsafe || bad
			f.Instances = append(f.Instances, summary{ID: sl.ID, Title: sl.Title, Slot: sl.State, OK: !bad, Ready: sl.State == "running", Now: f.Now,
				Runner: goals.Status{State: goals.RunnerState(sl.State), Activity: sl.State, Message: sl.Error}})
		}
	}
	if s.cfg.Sentinel != nil {
		f.Sentinel = s.cfg.Sentinel()
	}
	return f
}

// SentinelPanel is the status.Panel d2sentinel registers on its hub (snapshot is Runner.Snapshot).
func SentinelPanel(snapshot func() any) status.Panel {
	return status.Panel{ID: "sentinel", Title: "Sentinel", Order: 5, Every: 2 * time.Second, Snapshot: snapshot}
}
