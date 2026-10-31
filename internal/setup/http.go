package setup

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"strings"
)

// API serves the wizard and doctor as JSON. Mount it in internal/server at "/api/setup/" (prefix
// stripped). Contract: docs/setup.md.
type API struct {
	W    *Wizard
	Env  *Env
	Opts func() Options // doctor options from the current inputs
	// BaseCtx is the context background runs live in (server lifetime).
	BaseCtx context.Context
}

type errBody struct {
	Error string `json:"error"`
}

func writeJSON(w http.ResponseWriter, code int, v any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(code)
	_ = json.NewEncoder(w).Encode(v)
}

// Handler returns the routes: GET /state, GET /checks[?quick=1&pre=1], PUT /inputs, POST /start,
// POST /cancel, POST /reset[?step=ID].
func (a *API) Handler() http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		path := "/" + strings.Trim(r.URL.Path, "/")
		want := map[string]string{"/state": "GET", "/checks": "GET", "/inputs": "PUT", "/start": "POST", "/cancel": "POST", "/reset": "POST"}
		m, ok := want[path]
		if !ok {
			writeJSON(w, 404, errBody{"not found"})
			return
		}
		if r.Method != m {
			w.Header().Set("Allow", m)
			writeJSON(w, 405, errBody{"method not allowed"})
			return
		}
		switch path {
		case "/state":
			writeJSON(w, 200, a.W.View())
		case "/checks":
			o := a.Opts()
			o.Quick = r.URL.Query().Get("quick") == "1"
			o.GameExpected = r.URL.Query().Get("pre") != "1"
			writeJSON(w, 200, NewReport(RunChecks(r.Context(), a.Env, o, Checks())))
		case "/inputs":
			var in Inputs
			dec := json.NewDecoder(http.MaxBytesReader(w, r.Body, 64<<10))
			dec.DisallowUnknownFields() // a stray "cdKey" field is a 400, never stored
			if err := dec.Decode(&in); err != nil {
				writeJSON(w, 400, errBody{"bad inputs: unknown or malformed field (CD keys are never accepted here)"})
				return
			}
			if err := a.W.SetInputs(in); err != nil {
				writeJSON(w, busyCode(err), errBody{err.Error()})
				return
			}
			writeJSON(w, 200, a.W.View())
		case "/start":
			ctx := a.BaseCtx
			if ctx == nil {
				ctx = context.Background()
			}
			if err := a.W.Start(ctx); err != nil {
				writeJSON(w, busyCode(err), errBody{err.Error()})
				return
			}
			writeJSON(w, 202, a.W.View())
		case "/cancel":
			a.W.Cancel()
			writeJSON(w, 202, a.W.View())
		case "/reset":
			if err := a.W.Reset(r.URL.Query().Get("step")); err != nil {
				writeJSON(w, busyCode(err), errBody{err.Error()})
				return
			}
			writeJSON(w, 200, a.W.View())
		}
	})
}

func busyCode(err error) int {
	if errors.Is(err, ErrBusy) {
		return 409
	}
	return 400
}
