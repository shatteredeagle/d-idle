package watch

import (
	"errors"
	"net/http"
	"sync"
)

// The repair routes (docs/repair.md): GET /api/repair (gear condition, mode, request state), POST request and cancel on
// the answer listener with the same gates as the inventory POSTs, and the script of the compact gear strip on the watch
// screen. The desk is late-bound, like the inventory's.

var (
	RepairRoutes     = []string{"/api/repair", "/watch/gear.js"}
	RepairPostRoutes = []string{"/api/repair/request", "/api/repair/cancel"}
)

// RepairDesk is what the routes need from the bot side (internal/repair.Controller through repairAdapter in cmd/d2bot).
type RepairDesk interface {
	View() any
	Request() (any, error)
	Cancel() error
}

// RepairAPI holds the desk once it exists.
type RepairAPI struct {
	mu sync.Mutex
	d  RepairDesk
}

func NewRepairAPI() *RepairAPI { return &RepairAPI{} }

func (a *RepairAPI) Bind(d RepairDesk) { a.mu.Lock(); a.d = d; a.mu.Unlock() }

func (a *RepairAPI) desk() RepairDesk { a.mu.Lock(); defer a.mu.Unlock(); return a.d }

// Handler serves the viewer GET routes.
func (a *RepairAPI) Handler() http.Handler {
	m := http.NewServeMux()
	m.HandleFunc("/api/repair", func(w http.ResponseWriter, r *http.Request) {
		d := a.desk()
		if d == nil {
			writeJSON(w, http.StatusServiceUnavailable, policyReply{Code: "off", Error: "gear condition is not available yet"})
			return
		}
		w.Header().Set("Cache-Control", "no-store")
		writeJSON(w, 200, d.View())
	})
	m.HandleFunc("/watch/gear.js", func(w http.ResponseWriter, r *http.Request) {
		b, err := webFS.ReadFile("web/gear.js")
		if err != nil {
			http.NotFound(w, r)
			return
		}
		w.Header().Set("Content-Type", "text/javascript; charset=utf-8")
		w.Header().Set("X-Content-Type-Options", "nosniff")
		w.Header().Set("Content-Security-Policy", pageCSP)
		w.Header().Set("Cache-Control", "no-store")
		w.Write(b)
	})
	return readOnly(m)
}

// PostHandler serves the two POST routes.
func (a *RepairAPI) PostHandler(o AnswerOptions) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var b struct{}
		if !policyGate(o, w, r, &b) {
			return
		}
		d := a.desk()
		if d == nil {
			writeJSON(w, http.StatusServiceUnavailable, policyReply{Code: "off", Error: "gear condition is not available yet"})
			return
		}
		var (
			err error
			res any
		)
		switch r.URL.Path {
		case "/api/repair/request":
			res, err = d.Request()
		case "/api/repair/cancel":
			err = d.Cancel()
		default:
			http.NotFound(w, r)
			return
		}
		var ref invRefusal
		switch {
		case err == nil:
			writeJSON(w, 200, map[string]any{"ok": true, "request": res})
		case errors.As(err, &ref):
			writeJSON(w, http.StatusConflict, policyReply{Code: ref.Kind(), Error: ref.Error()})
		default:
			writeJSON(w, http.StatusInternalServerError, policyReply{Code: "internal", Error: "could not save the request"})
		}
	})
}
