// Package api serves the goal manager over HTTP: JSON under /api/v1, Server-Sent Events for live updates,
// and the embedded dashboard at /. See docs/api.md for the contract.
package api

import (
	"context"
	"crypto/subtle"
	"encoding/json"
	"errors"
	"io"
	"io/fs"
	"net"
	"net/http"
	"net/url"
	"strings"
	"time"

	"d2lod-bot/internal/goals"
)

const Prefix = "/api/v1"

type Config struct {
	Manager *goals.Manager
	// Addr is the listen address, default 127.0.0.1:8420. A non-loopback address requires Token.
	Addr string
	// Token, when set, is required on every request (Authorization: Bearer, the d2token cookie, or ?token=).
	Token string
	// Assets is the dashboard (index.html at its root). Nil serves the API only.
	Assets fs.FS
	// AllowedHosts are extra Host header names accepted on a loopback listener (reverse proxies, ssh tunnels).
	AllowedHosts []string
	Version      string
}

type Server struct {
	cfg  Config
	base string // URL prefix the routes are mounted under (Prefix, or an instance prefix)
	mux  http.Handler
}

// NewInstance returns the routes of one instance without any security wrapper, for mounting under base
// (for example /api/v1/instances/i1) by internal/server, which owns the one auth scheme. Version may be empty.
func NewInstance(m *goals.Manager, base, version string) *Server {
	return &Server{cfg: Config{Manager: m, Version: version}, base: strings.TrimRight(base, "/")}
}

// Routes serves the instance routes (status, goals, settings, stats, ...) with no guard.
func (s *Server) Routes() http.Handler { return http.HandlerFunc(s.apiRoute) }

func New(cfg Config) (*Server, error) {
	if cfg.Manager == nil {
		return nil, errors.New("api: Config.Manager is required")
	}
	if cfg.Addr == "" {
		cfg.Addr = "127.0.0.1:8420"
	}
	if !IsLoopbackAddr(cfg.Addr) && cfg.Token == "" {
		return nil, errors.New("api: listening on a non-loopback address requires a token")
	}
	s := &Server{cfg: cfg, base: Prefix}
	s.mux = s.routes()
	return s, nil
}

// IsLoopbackAddr reports whether a listen address is loopback-only.
func IsLoopbackAddr(addr string) bool {
	host, _, err := net.SplitHostPort(addr)
	if err != nil {
		return false
	}
	if host == "localhost" {
		return true
	}
	ip := net.ParseIP(host)
	return ip != nil && ip.IsLoopback()
}

func (s *Server) Handler() http.Handler { return s.mux }

// Listen opens the configured address; pass the listener to Serve (port 0 picks a free port).
func (s *Server) Listen() (net.Listener, error) { return net.Listen("tcp", s.cfg.Addr) }

// Serve runs until ctx is done, then shuts down gracefully. SSE streams are closed.
func (s *Server) Serve(ctx context.Context, ln net.Listener) error {
	base, stop := context.WithCancel(context.Background())
	defer stop()
	srv := &http.Server{
		Handler:           s.mux,
		ReadHeaderTimeout: 10 * time.Second,
		ReadTimeout:       30 * time.Second,
		IdleTimeout:       2 * time.Minute,
		BaseContext:       func(net.Listener) context.Context { return base },
	}
	errc := make(chan error, 1)
	go func() { errc <- srv.Serve(ln) }()
	select {
	case err := <-errc:
		return err
	case <-ctx.Done():
	}
	stop() // ends SSE handlers
	sctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()
	if err := srv.Shutdown(sctx); err != nil {
		srv.Close()
	}
	return nil
}

// ---- helpers -------------------------------------------------------------------------------------

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

func writeErr(w http.ResponseWriter, err error) {
	status, code := http.StatusInternalServerError, "internal"
	switch {
	case errors.Is(err, goals.ErrNotFound):
		status, code = http.StatusNotFound, "not_found"
	case errors.Is(err, goals.ErrInvalid):
		status, code = http.StatusBadRequest, "invalid"
	case errors.Is(err, goals.ErrState):
		status, code = http.StatusConflict, "conflict"
	}
	msg := strings.TrimPrefix(err.Error(), "goals: ")
	writeJSON(w, status, errBody{msg, code})
}

func badRequest(w http.ResponseWriter, msg string) {
	writeJSON(w, http.StatusBadRequest, errBody{msg, "invalid"})
}

func decode(w http.ResponseWriter, r *http.Request, v any) bool {
	dec := json.NewDecoder(http.MaxBytesReader(w, r.Body, 64<<10))
	dec.DisallowUnknownFields()
	if err := dec.Decode(v); err != nil {
		if errors.Is(err, io.EOF) {
			badRequest(w, "request body is empty")
		} else {
			badRequest(w, "bad JSON: "+err.Error())
		}
		return false
	}
	if dec.More() {
		badRequest(w, "unexpected data after the JSON body")
		return false
	}
	return true
}

// ---- middleware ----------------------------------------------------------------------------------

// Security is the one auth scheme of the product: Host check on loopback listeners, token (Bearer, X-Token,
// d2token cookie, ?token=), Origin and Content-Type checks on mutations, and the response headers.
type Security struct {
	Addr         string
	Token        string
	AllowedHosts []string
}

// Validate refuses a non-loopback listen address without a token.
func (x Security) Validate() error {
	if !IsLoopbackAddr(x.Addr) && x.Token == "" {
		return errors.New("api: listening on a non-loopback address requires a token")
	}
	return nil
}

func (s *Server) guard(next http.Handler) http.Handler {
	return Security{Addr: s.cfg.Addr, Token: s.cfg.Token, AllowedHosts: s.cfg.AllowedHosts}.Wrap(next)
}

func (x Security) hostOK(r *http.Request) bool {
	if !IsLoopbackAddr(x.Addr) {
		return true // token-protected; any Host is fine
	}
	h := r.Host
	if host, _, err := net.SplitHostPort(h); err == nil {
		h = host
	}
	h = strings.Trim(h, "[]")
	if h == "localhost" || h == "127.0.0.1" || h == "::1" {
		return true
	}
	for _, a := range x.AllowedHosts {
		if strings.EqualFold(a, h) {
			return true
		}
	}
	return false
}

func (x Security) tokenOK(r *http.Request) bool {
	if x.Token == "" {
		return true
	}
	eq := func(got string) bool {
		return got != "" && subtle.ConstantTimeCompare([]byte(got), []byte(x.Token)) == 1
	}
	if a := r.Header.Get("Authorization"); strings.HasPrefix(a, "Bearer ") && eq(a[7:]) {
		return true
	}
	if eq(r.Header.Get("X-Token")) { // d2watch's header, kept for its clients
		return true
	}
	if c, err := r.Cookie("d2token"); err == nil && eq(c.Value) {
		return true
	}
	return eq(r.URL.Query().Get("token"))
}

func (x Security) Wrap(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		h := w.Header()
		h.Set("X-Content-Type-Options", "nosniff")
		h.Set("Referrer-Policy", "no-referrer")
		h.Set("X-Frame-Options", "DENY")
		h.Set("Content-Security-Policy", "default-src 'self'; img-src 'self' data:; frame-ancestors 'none'; base-uri 'none'; form-action 'none'")
		if !x.hostOK(r) {
			writeJSON(w, http.StatusForbidden, errBody{"host not allowed; use localhost or pass -allow-host", "forbidden"})
			return
		}
		if !x.tokenOK(r) {
			if isAPIPath(r.URL.Path) {
				w.Header().Set("WWW-Authenticate", `Bearer realm="d2lod"`)
				writeJSON(w, http.StatusUnauthorized, errBody{"missing or wrong token", "unauthorized"})
			} else {
				http.Error(w, "Unauthorized. Open the dashboard with ?token=YOUR_TOKEN once.", http.StatusUnauthorized)
			}
			return
		}
		// A valid ?token= on a page load becomes a cookie, and the token leaves the address bar.
		if t := r.URL.Query().Get("token"); t != "" && x.Token != "" && r.Method == http.MethodGet && !isAPIPath(r.URL.Path) {
			http.SetCookie(w, &http.Cookie{Name: "d2token", Value: t, Path: "/", HttpOnly: true, SameSite: http.SameSiteStrictMode, MaxAge: 365 * 86400})
			q := r.URL.Query()
			q.Del("token")
			u := url.URL{Path: r.URL.Path, RawQuery: q.Encode()}
			http.Redirect(w, r, u.String(), http.StatusSeeOther)
			return
		}
		if r.Method != http.MethodGet && r.Method != http.MethodHead {
			if o := r.Header.Get("Origin"); o != "" {
				if u, err := url.Parse(o); err != nil || u.Host != r.Host {
					writeJSON(w, http.StatusForbidden, errBody{"cross-origin request refused", "forbidden"})
					return
				}
			}
			if ct := r.Header.Get("Content-Type"); !strings.HasPrefix(ct, "application/json") {
				writeJSON(w, http.StatusUnsupportedMediaType, errBody{"send Content-Type: application/json", "unsupported"})
				return
			}
		}
		next.ServeHTTP(w, r)
	})
}

// isAPIPath is true for JSON endpoints (and the /live/ data feeds): they get 401 JSON, never a cookie redirect.
func isAPIPath(p string) bool         { return strings.HasPrefix(p, "/api/") || strings.Contains(p, "/api/") }
func isLoopbackAddr(addr string) bool { return IsLoopbackAddr(addr) }
