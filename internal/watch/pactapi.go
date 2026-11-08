package watch

import (
	"errors"
	"mime"
	"net/http"
	"sync"
)

// The Pact routes (TASK-169, docs/pact-away.md). GET on the viewer, POST on the answer listener (same gates as the other
// answer routes: origin list, token, JSON only, small body). The desk is late-bound: the Pact controller exists only
// once the hero's class is known, the routes are mounted before Serve.

// PactRoutes are the viewer GET routes; PactPostRoutes the answer listener's.
var (
	PactRoutes     = []string{"/api/pacts/offer"}
	PactPostRoutes = []string{"/api/pacts/consent", "/api/pacts/revoke", "/api/pacts/hardcore"}
)

// ErrPactInactive marks a refusal that is not the caller's mistake: the thing is not switched on (Hardcore today).
var ErrPactInactive = errors.New("not active yet")

// PactDesk is what the routes need from the bot side (cmd/d2bot/pactwire.go adapts pact.Controller to it).
type PactDesk interface {
	Offer() any // plain-language offer: what can be lost, with numbers, and the state of each consent
	Consent(pact string, rank, awayHours int, typed string) error
	Revoke(pact string) error
	Hardcore() any
	HardcoreConfirm(typed string) error
	HardcoreRevoke() error
	HardcoreActivate() error
}

// PactAPI holds the desk once it exists.
type PactAPI struct {
	mu sync.Mutex
	d  PactDesk
}

// NewPactAPI makes the empty holder.
func NewPactAPI() *PactAPI { return &PactAPI{} }

// Bind sets the desk.
func (a *PactAPI) Bind(d PactDesk) { a.mu.Lock(); a.d = d; a.mu.Unlock() }

func (a *PactAPI) desk() PactDesk { a.mu.Lock(); defer a.mu.Unlock(); return a.d }

// Handler serves GET /api/pacts/offer.
func (a *PactAPI) Handler() http.Handler {
	return readOnly(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		d := a.desk()
		if d == nil {
			writeJSON(w, http.StatusServiceUnavailable, policyReply{Code: "off", Error: "Pacts are off or the hero is not known yet"})
			return
		}
		writeJSON(w, 200, map[string]any{"offer": d.Offer(), "hardcore": d.Hardcore()})
	}))
}

type pactBody struct {
	Pact      string `json:"pact"`
	Rank      int    `json:"rank"`
	AwayHours int    `json:"awayHours"`
	Typed     string `json:"typed"`
	Action    string `json:"action"`
}

// PostHandler serves the three POST routes.
func (a *PactAPI) PostHandler(o AnswerOptions) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		origin := r.Header.Get("Origin")
		allowed := origin != "" && containsStr(o.Origins, origin)
		if allowed {
			w.Header().Set("Access-Control-Allow-Origin", origin)
			w.Header().Set("Access-Control-Allow-Credentials", "true")
			w.Header().Set("Vary", "Origin")
		}
		fail := func(code int, c, msg string) { writeJSON(w, code, policyReply{Code: c, Error: msg}) }
		switch {
		case r.Method == http.MethodOptions && allowed:
			w.Header().Set("Access-Control-Allow-Methods", "POST")
			w.Header().Set("Access-Control-Allow-Headers", "Content-Type, Authorization")
			w.WriteHeader(http.StatusNoContent)
			return
		case r.Method != http.MethodPost:
			w.Header().Set("Allow", "POST, OPTIONS")
			fail(http.StatusMethodNotAllowed, "method", "POST only")
			return
		case origin != "" && !allowed:
			fail(http.StatusForbidden, "origin", "origin not allowed")
			return
		case !o.tokenOK(r):
			fail(http.StatusUnauthorized, "token", "unauthorized")
			return
		}
		if mt, _, err := mime.ParseMediaType(r.Header.Get("Content-Type")); err != nil || mt != "application/json" {
			fail(http.StatusUnsupportedMediaType, "content-type", "send application/json")
			return
		}
		var b pactBody
		if err := decodeSmall(r, &b); err != nil {
			fail(http.StatusBadRequest, "body", "body must be one small JSON object with known fields")
			return
		}
		d := a.desk()
		if d == nil {
			fail(http.StatusServiceUnavailable, "off", "Pacts are off or the hero is not known yet")
			return
		}
		var err error
		switch r.URL.Path {
		case "/api/pacts/consent":
			err = d.Consent(b.Pact, b.Rank, b.AwayHours, b.Typed)
		case "/api/pacts/revoke":
			err = d.Revoke(b.Pact)
		case "/api/pacts/hardcore":
			switch b.Action {
			case "confirm":
				err = d.HardcoreConfirm(b.Typed)
			case "revoke":
				err = d.HardcoreRevoke()
			case "activate":
				err = d.HardcoreActivate()
			default:
				fail(http.StatusBadRequest, "action", `action is "confirm", "revoke" or "activate"`)
				return
			}
		default:
			http.NotFound(w, r)
			return
		}
		switch {
		case err == nil:
			writeJSON(w, 200, policyReply{OK: true})
		case errors.Is(err, ErrPactInactive):
			fail(http.StatusNotImplemented, "inactive", err.Error())
		default:
			fail(http.StatusBadRequest, "refused", err.Error())
		}
	})
}
