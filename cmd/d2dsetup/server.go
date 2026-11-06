package main

import (
	"crypto/subtle"
	"embed"
	"encoding/json"
	"errors"
	"io/fs"
	"net/http"
	"net/url"
	"strings"

	"d2lod-bot/internal/access"
	"d2lod-bot/web"
)

//go:embed page
var pageFS embed.FS

// cookieName is the watch screen's own cookie (internal/watch tokenGate), so the browser that passed this gate is
// already signed in on the watch screen that takes over port 8891.
const cookieName = "d2dview"

type Server struct {
	W     *Wizard
	Token string // the watch token; there is no tokenless mode
	mux   *http.ServeMux
}

func NewServer(w *Wizard, token string) (*Server, error) {
	if len(token) < 16 {
		return nil, errors.New("the watch token must be at least 16 characters")
	}
	s := &Server{W: w, Token: token, mux: http.NewServeMux()}
	page, _ := fs.Sub(pageFS, "page")
	static := http.FileServerFS(page)
	index := func(rw http.ResponseWriter, r *http.Request) {
		b, _ := fs.ReadFile(page, "index.html")
		rw.Header().Set("Content-Type", "text/html; charset=utf-8")
		rw.Write(b)
	}
	s.mux.HandleFunc("GET /{$}", index)
	s.mux.HandleFunc("GET /setup.html", index)
	s.mux.Handle("GET /app.js", static)
	s.mux.HandleFunc("GET /style.css", func(rw http.ResponseWriter, r *http.Request) {
		b, err := web.FS.ReadFile("style.css") // the wizard's own look, not forked
		if err != nil {
			http.NotFound(rw, r)
			return
		}
		rw.Header().Set("Content-Type", "text/css; charset=utf-8")
		rw.Write(b)
	})
	s.mux.HandleFunc("GET /api/state", func(rw http.ResponseWriter, r *http.Request) { writeJSON(rw, 200, s.W.State(s.Token)) })
	s.mux.HandleFunc("POST /api/check", func(rw http.ResponseWriter, r *http.Request) {
		s.W.Check()
		writeJSON(rw, 200, s.W.State(s.Token))
	})
	s.mux.HandleFunc("PUT /api/inputs", s.putInputs)
	s.mux.HandleFunc("PUT /api/ahead", s.putAhead)
	s.mux.HandleFunc("PUT /api/access", s.putAccess)
	s.mux.HandleFunc("POST /api/start", func(rw http.ResponseWriter, r *http.Request) {
		if err := s.W.Start(); err != nil {
			fail(rw, err)
			return
		}
		writeJSON(rw, 202, s.W.State(s.Token))
	})
	return s, nil
}

type errBody struct {
	Error string `json:"error"`
	Fix   string `json:"fix,omitempty"`
}

// fail writes err as JSON with its fix hint when it has one.
func fail(rw http.ResponseWriter, err error) {
	b := errBody{Error: err.Error()}
	var ue *userError
	if errors.As(err, &ue) {
		b.Fix = ue.Fix
	}
	writeJSON(rw, errCode(err), b)
}

func errCode(err error) int {
	if errors.Is(err, errBusy) {
		return 409
	}
	return 400
}

func writeJSON(w http.ResponseWriter, code int, v any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(code)
	_ = json.NewEncoder(w).Encode(v)
}

func (s *Server) putInputs(rw http.ResponseWriter, r *http.Request) {
	var in Inputs
	dec := json.NewDecoder(http.MaxBytesReader(rw, r.Body, 4<<10))
	dec.DisallowUnknownFields() // a stray "cdKey" field is a 400 and is never stored
	if err := dec.Decode(&in); err != nil {
		writeJSON(rw, 400, errBody{Error: "bad input: unknown or malformed field (CD keys are never accepted here)"})
		return
	}
	if err := s.W.SetInputs(in); err != nil {
		fail(rw, err)
		return
	}
	writeJSON(rw, 200, s.W.State(s.Token))
}

func decodeBody(rw http.ResponseWriter, r *http.Request, v any) bool {
	dec := json.NewDecoder(http.MaxBytesReader(rw, r.Body, 4<<10))
	dec.DisallowUnknownFields()
	if err := dec.Decode(v); err != nil || dec.More() {
		writeJSON(rw, 400, errBody{Error: "bad input: unknown or malformed field", Fix: "Reload the page and try again."})
		return false
	}
	return true
}

func (s *Server) putAhead(rw http.ResponseWriter, r *http.Request) {
	var b struct {
		OK bool `json:"ok"`
	}
	if !decodeBody(rw, r, &b) {
		return
	}
	if err := s.W.SetAhead(b.OK); err != nil {
		fail(rw, err)
		return
	}
	writeJSON(rw, 200, s.W.State(s.Token))
}

// putAccess stores the choice. For a passcode it also signs this browser in on the watch screen that takes over
// the port (the d2dpass cookie, SameSite=Strict), so the tester is not asked for the passcode they just typed.
func (s *Server) putAccess(rw http.ResponseWriter, r *http.Request) {
	var c AccessChoice
	if !decodeBody(rw, r, &c) {
		return
	}
	rec, err := s.W.SetAccess(c)
	if err != nil {
		fail(rw, err)
		return
	}
	if rec.Kind == access.KindPasscode {
		http.SetCookie(rw, rec.Cookie())
	}
	writeJSON(rw, 200, s.W.State(s.Token))
}

// ServeHTTP gates everything: the token (bearer header, or ?t= which sets the cookie and is stripped by a
// redirect), or the passcode cookie once one is chosen, is required for every request, loopback or not. Mutating requests also need a same-origin Origin
// (when the browser sends one) and a JSON content type.
func (s *Server) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	h := w.Header()
	h.Set("Cache-Control", "no-store")
	h.Set("Referrer-Policy", "no-referrer")
	h.Set("X-Content-Type-Options", "nosniff")
	h.Set("Content-Security-Policy", "default-src 'self'; frame-ancestors 'none'")
	got := strings.TrimPrefix(r.Header.Get("Authorization"), "Bearer ")
	fromQuery := false
	if q := r.URL.Query().Get("t"); q != "" {
		got, fromQuery = q, true
	} else if c, err := r.Cookie(cookieName); err == nil && got == "" {
		got = c.Value
	}
	signedIn := false
	if c, err := r.Cookie(access.SessionCookie); err == nil {
		signedIn = s.W.Access.Current().SessionOK(c.Value) // the passcode just chosen also opens this page again
	}
	if !signedIn && subtle.ConstantTimeCompare([]byte(got), []byte(s.Token)) != 1 {
		http.Error(w, "unauthorized: open the page with ?t=<the watch token> (see /data/watch.token)", http.StatusUnauthorized)
		return
	}
	if fromQuery && subtle.ConstantTimeCompare([]byte(got), []byte(s.Token)) == 1 {
		http.SetCookie(w, &http.Cookie{Name: cookieName, Value: got, Path: "/", HttpOnly: true, SameSite: http.SameSiteLaxMode})
		u := *r.URL
		q := u.Query()
		q.Del("t")
		u.RawQuery = q.Encode()
		http.Redirect(w, r, u.RequestURI(), http.StatusSeeOther)
		return
	}
	if r.Method != http.MethodGet && r.Method != http.MethodHead {
		if o := r.Header.Get("Origin"); o != "" {
			if u, err := url.Parse(o); err != nil || u.Host != r.Host {
				http.Error(w, "forbidden origin", http.StatusForbidden)
				return
			}
		}
		if ct := r.Header.Get("Content-Type"); r.ContentLength != 0 && !strings.HasPrefix(ct, "application/json") {
			http.Error(w, "content type must be application/json", http.StatusUnsupportedMediaType)
			return
		}
	}
	s.mux.ServeHTTP(w, r)
}
