package fleet

import (
	"context"
	"crypto/subtle"
	"encoding/json"
	"errors"
	"net/http"
	"strings"
	"time"
)

// Handler is the admin API of a Fleet, for cmd/d2fleet and as a mount in internal/server (the server strips
// the mount prefix, for example Mount{Path: "/api/v1/fleet-admin/", Handler: fleet.Handler(f, "")}). Auth is
// the server's job when mounted; token != "" adds a bearer check for the standalone d2fleet.
//
//	GET    /status               HostStatus: capacity checks and all slots
//	GET    /slots                []View
//	POST   /slots                body Spec; 201 View | 409 duplicate | 422 invalid | 503 admission refused (reason in the body)
//	GET    /slots/{id}           View
//	DELETE /slots/{id}[?force=1] graceful drain, then 204
//	POST   /slots/{id}/restart   body {"reason": "..."}
//	POST   /slots/{id}/lease     body {"owner","ttlSeconds"}; 200 {"owner","expires"} | 409 holder
//	DELETE /slots/{id}/lease     release (only the holder's token would be stronger; U: owner is advisory over HTTP)
func Handler(f *Fleet, token string) http.Handler {
	h := &api{f: f}
	mux := http.NewServeMux()
	mux.HandleFunc("GET /status", h.status)
	mux.HandleFunc("GET /slots", func(w http.ResponseWriter, r *http.Request) { writeJSON(w, 200, f.List()) })
	mux.HandleFunc("POST /slots", h.add)
	mux.HandleFunc("GET /slots/{id}", h.get)
	mux.HandleFunc("DELETE /slots/{id}", h.remove)
	mux.HandleFunc("POST /slots/{id}/restart", h.restart)
	mux.HandleFunc("POST /slots/{id}/lease", h.lease)
	mux.HandleFunc("DELETE /slots/{id}/lease", h.unlease)
	if token == "" {
		return mux
	}
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		got := strings.TrimPrefix(r.Header.Get("Authorization"), "Bearer ")
		if subtle.ConstantTimeCompare([]byte(got), []byte(token)) != 1 {
			writeJSON(w, 401, map[string]string{"error": "unauthorized"})
			return
		}
		mux.ServeHTTP(w, r)
	})
}

type api struct {
	f *Fleet
}

func writeJSON(w http.ResponseWriter, code int, v any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(code)
	_ = json.NewEncoder(w).Encode(v)
}

func fail(w http.ResponseWriter, err error) {
	code := 500
	var ae *AdmissionError
	var le *LeasedError
	switch {
	case errors.As(err, &ae):
		code = 503
		writeJSON(w, code, map[string]any{"error": err.Error(), "check": ae.Check, "need": ae.Need, "have": ae.Have})
		return
	case errors.As(err, &le):
		code = 409
		writeJSON(w, code, map[string]any{"error": err.Error(), "holder": le.Holder, "until": le.Until})
		return
	case errors.Is(err, ErrNoSlot):
		code = 404
	case errors.Is(err, ErrDuplicate), errors.Is(err, ErrLeased):
		code = 409
	}
	writeJSON(w, code, map[string]string{"error": err.Error()})
}

func (a *api) status(w http.ResponseWriter, r *http.Request) { writeJSON(w, 200, a.f.Status()) }

func (a *api) add(w http.ResponseWriter, r *http.Request) {
	var sp Spec
	if err := json.NewDecoder(http.MaxBytesReader(w, r.Body, 1<<16)).Decode(&sp); err != nil {
		writeJSON(w, 400, map[string]string{"error": "bad json: " + err.Error()})
		return
	}
	if err := sp.Validate(); err != nil {
		writeJSON(w, 422, map[string]string{"error": err.Error()})
		return
	}
	v, err := a.f.Add(r.Context(), sp)
	if err != nil {
		fail(w, err)
		return
	}
	writeJSON(w, 201, v)
}

func (a *api) get(w http.ResponseWriter, r *http.Request) {
	v, err := a.f.Get(r.PathValue("id"))
	if err != nil {
		fail(w, err)
		return
	}
	writeJSON(w, 200, v)
}

func (a *api) remove(w http.ResponseWriter, r *http.Request) {
	// A drain outlives the HTTP request: a client that disconnects must not abort the save-and-exit.
	ctx, cancel := context.WithTimeout(context.WithoutCancel(r.Context()), 2*time.Minute)
	defer cancel()
	if err := a.f.Remove(ctx, r.PathValue("id"), RemoveOpts{Force: r.URL.Query().Get("force") == "1"}); err != nil {
		fail(w, err)
		return
	}
	w.WriteHeader(204)
}

func (a *api) restart(w http.ResponseWriter, r *http.Request) {
	var b struct{ Reason string }
	_ = json.NewDecoder(http.MaxBytesReader(w, r.Body, 1<<12)).Decode(&b)
	if err := a.f.Restart(r.Context(), r.PathValue("id"), b.Reason); err != nil {
		fail(w, err)
		return
	}
	w.WriteHeader(202)
}

func (a *api) lease(w http.ResponseWriter, r *http.Request) {
	var b struct {
		Owner      string
		TTLSeconds int `json:"ttlSeconds"`
	}
	if err := json.NewDecoder(http.MaxBytesReader(w, r.Body, 1<<12)).Decode(&b); err != nil {
		writeJSON(w, 400, map[string]string{"error": "bad json"})
		return
	}
	l, err := a.f.Lease(r.PathValue("id"), b.Owner, time.Duration(b.TTLSeconds)*time.Second)
	if err != nil {
		fail(w, err)
		return
	}
	writeJSON(w, 200, map[string]any{"owner": l.Owner, "expires": l.Expires})
}

func (a *api) unlease(w http.ResponseWriter, r *http.Request) {
	a.f.mu.Lock()
	if s, ok := a.f.slots[r.PathValue("id")]; ok {
		s.lease = nil
	}
	a.f.mu.Unlock()
	w.WriteHeader(204)
}
