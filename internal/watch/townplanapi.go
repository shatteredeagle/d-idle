package watch

import (
	"net/http"
	"sync"

	"d2lod-bot/internal/supply"
)

// GET /api/townplan (watch.townplan/1, additive; docs/design/fable-review-2026-11-21/town-planner.md section 8): the
// town plan of the last visit as the Town plan card shows it. Read-only; the plan is solved on the bot side and only
// reported here. Late-bound like the other desks: 503 until cmd/d2bot binds the source.

const TownPlanRoute = "/api/townplan"

type TownPlanAPI struct {
	mu  sync.Mutex
	src func() supply.View
}

func NewTownPlanAPI() *TownPlanAPI { return &TownPlanAPI{} }

// Bind sets the source (cmd/d2bot: the last plan the town solved, through supply.ViewOf; nil view before any visit is NoView).
func (a *TownPlanAPI) Bind(src func() supply.View) { a.mu.Lock(); a.src = src; a.mu.Unlock() }

func (a *TownPlanAPI) source() func() supply.View { a.mu.Lock(); defer a.mu.Unlock(); return a.src }

func (a *TownPlanAPI) Handler() http.Handler {
	m := http.NewServeMux()
	m.HandleFunc(TownPlanRoute, func(w http.ResponseWriter, r *http.Request) {
		src := a.source()
		if src == nil {
			writeJSON(w, http.StatusServiceUnavailable, policyReply{Code: "off", Error: "the town plan is not available yet"})
			return
		}
		w.Header().Set("Cache-Control", "no-store")
		writeJSON(w, 200, src())
	})
	return readOnly(m)
}
