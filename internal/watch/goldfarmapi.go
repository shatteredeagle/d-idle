package watch

import (
	"net/http"
	"sync"
)

// The Farm gold routes (docs/goldfarm.md): GET /api/farmgold (status) on the viewer side, POST /api/farmgold {on} on the
// answer listener with the same gates as the repair and inventory POSTs. The desk is late-bound.

const GoldFarmRoute = "/api/farmgold"

// GoldFarmDesk is what the routes need from the bot side (internal/goldfarm.Controller through an adapter in cmd/d2bot).
type GoldFarmDesk interface {
	View() any
	Set(on bool) error
}

// GoldFarmAPI holds the desk once it exists.
type GoldFarmAPI struct {
	mu sync.Mutex
	d  GoldFarmDesk
}

func NewGoldFarmAPI() *GoldFarmAPI { return &GoldFarmAPI{} }

func (a *GoldFarmAPI) Bind(d GoldFarmDesk) { a.mu.Lock(); a.d = d; a.mu.Unlock() }

func (a *GoldFarmAPI) desk() GoldFarmDesk { a.mu.Lock(); defer a.mu.Unlock(); return a.d }

// Handler serves the viewer GET.
func (a *GoldFarmAPI) Handler() http.Handler {
	m := http.NewServeMux()
	m.HandleFunc(GoldFarmRoute, func(w http.ResponseWriter, r *http.Request) {
		d := a.desk()
		if d == nil {
			writeJSON(w, http.StatusServiceUnavailable, policyReply{Code: "off", Error: "gold farming is not available yet"})
			return
		}
		w.Header().Set("Cache-Control", "no-store")
		writeJSON(w, 200, d.View())
	})
	return readOnly(m)
}

// PostHandler serves POST /api/farmgold {"on": true|false}.
func (a *GoldFarmAPI) PostHandler(o AnswerOptions) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var b struct {
			On *bool `json:"on"`
		}
		if !policyGate(o, w, r, &b) {
			return
		}
		if b.On == nil {
			writeJSON(w, http.StatusBadRequest, policyReply{Code: "body", Error: `send {"on": true} or {"on": false}`})
			return
		}
		d := a.desk()
		if d == nil {
			writeJSON(w, http.StatusServiceUnavailable, policyReply{Code: "off", Error: "gold farming is not available yet"})
			return
		}
		if err := d.Set(*b.On); err != nil {
			writeJSON(w, http.StatusInternalServerError, policyReply{Code: "internal", Error: "could not save the order"})
			return
		}
		writeJSON(w, 200, map[string]any{"ok": true, "farm": d.View()})
	})
}
