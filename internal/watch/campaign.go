package watch

import (
	"net/http"
	"sync"

	"d2lod-bot/internal/campaign"
)

// The guided journey routes (docs/campaign.md section 6): GET /api/campaign (schema campaign/1: the chapters with band,
// intent and state, the decision, the level meter) on the viewer side; POST /api/campaign/intent on the answer listener
// with the same gates as the policy and repair POSTs. The desk (*campaign.Controller) is late-bound.

const (
	CampaignRoute       = "/api/campaign"
	CampaignIntentRoute = "/api/campaign/intent"
)

// CampaignRoutes are the GET routes of the journey.
var CampaignRoutes = []string{CampaignRoute}

// CampaignDesk is what the routes need: campaign.Controller.
type CampaignDesk interface {
	View() campaign.View
	Apply(campaign.Request) error
}

// CampaignAPI holds the desk once it exists.
type CampaignAPI struct {
	mu sync.Mutex
	d  CampaignDesk
}

func NewCampaignAPI() *CampaignAPI { return &CampaignAPI{} }

func (a *CampaignAPI) Bind(d CampaignDesk) { a.mu.Lock(); a.d = d; a.mu.Unlock() }

func (a *CampaignAPI) desk() CampaignDesk { a.mu.Lock(); defer a.mu.Unlock(); return a.d }

func campaignOff(w http.ResponseWriter) {
	writeJSON(w, http.StatusServiceUnavailable, policyReply{Code: "off", Error: "the journey is not available yet"})
}

// Handler serves the viewer GET.
func (a *CampaignAPI) Handler() http.Handler {
	m := http.NewServeMux()
	m.HandleFunc(CampaignRoute, func(w http.ResponseWriter, r *http.Request) {
		d := a.desk()
		if d == nil {
			campaignOff(w)
			return
		}
		w.Header().Set("Cache-Control", "no-store")
		writeJSON(w, 200, d.View())
	})
	return readOnly(m)
}

// PostHandler serves POST /api/campaign/intent with one campaign.Request: {chapter, intent} (auto, ask, skip, hold),
// {chapter, goNow}, {chapter, answer}, {goAt} (recommended, floor) or {mode} (guided, free, orders).
func (a *CampaignAPI) PostHandler(o AnswerOptions) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var b campaign.Request
		if !policyGate(o, w, r, &b) {
			return
		}
		d := a.desk()
		if d == nil {
			campaignOff(w)
			return
		}
		if err := d.Apply(b); err != nil {
			writeJSON(w, http.StatusBadRequest, policyReply{Code: "body", Error: short(err.Error(), 200)})
			return
		}
		writeJSON(w, 200, map[string]any{"ok": true, "campaign": d.View()})
	})
}

func short(s string, n int) string {
	if len(s) > n {
		return s[:n]
	}
	return s
}

// ChapterDone tells the watch screen that a journey chapter is done (docs/campaign.md section 6): a Chronicle line of
// kind "chapter", a story line in the feed and journal ("Chapter done: Blood Raven (Kashya's free Rogue mercenary)"),
// and the unlock ladder's "seen" event "chapter:<id>", so a ladder entry can open on a chapter through data.
func (f *Fun) ChapterDone(id, title, reward string) {
	if f == nil {
		return
	}
	text := "Chapter done: " + title
	if reward != "" {
		text += " (" + reward + ")"
	}
	if f.Chronicle != nil {
		f.Chronicle.chapter(text)
	}
	if f.Hub != nil {
		f.Hub.Note("story", text)
	}
	if f.Unlocks != nil {
		f.Unlocks.Seen("chapter:" + id)
	}
}

func (c *Chronicle) chapter(text string) {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.n++
	c.lines = append(c.lines, ChronicleLine{N: c.n, T: c.now(), Kind: "chapter", Text: safeText(text, 200)})
	if len(c.lines) > chronicleMax {
		c.lines = append([]ChronicleLine(nil), c.lines[len(c.lines)-chronicleMax:]...)
	}
}
