package watch

import (
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"sync"
	"time"
)

// invViewTTL is how long one built /api/inventory body is served again. Building the view judges every carried item
// and checks ten slots per backpack item, so several open pages (or a page and the overlay) must not each repeat it
// (TASK-268). A POST on the inventory routes drops it at once, so the page sees its own change.
var invViewTTL = 1500 * time.Millisecond

// The inventory page (docs/inventory-ui.md): GET on the viewer, POST on the answer listener with the same gates as the
// other answer routes (origin list, token or passcode cookie, JSON only, small body). The desk is late-bound.

var (
	InvRoutes     = []string{"/api/inventory", "/watch/inventory", "/watch/inventory.js"}
	InvPostRoutes = []string{"/api/inventory/mark", "/api/inventory/equip", "/api/inventory/unpin"}
)

// InvDesk is what the routes need from the bot side (internal/invdesk.Desk through invAdapter in cmd/d2bot).
type InvDesk interface {
	View() any
	SetMark(id uint32, mark string) error
	RequestEquip(id uint32, slot string) (any, error)
	Unpin(slot string) error
}

// invRefusal is an error that carries a short code; the routes reply 409 with it and the reason line.
type invRefusal interface {
	error
	Kind() string
}

// InvAPI holds the desk once it exists.
type InvAPI struct {
	mu sync.Mutex
	d  InvDesk

	cmu    sync.Mutex // held while the view is built, so concurrent polls share one build
	cached []byte
	at     time.Time
}

func (a *InvAPI) drop() { a.cmu.Lock(); a.cached = nil; a.cmu.Unlock() }

func (a *InvAPI) viewJSON(d InvDesk) ([]byte, error) {
	a.cmu.Lock()
	defer a.cmu.Unlock()
	if a.cached != nil && time.Since(a.at) < invViewTTL {
		return a.cached, nil
	}
	b, err := json.Marshal(d.View())
	if err != nil {
		return nil, err
	}
	a.cached, a.at = append(b, '\n'), time.Now()
	return a.cached, nil
}

func NewInvAPI() *InvAPI { return &InvAPI{} }

func (a *InvAPI) Bind(d InvDesk) { a.mu.Lock(); a.d = d; a.mu.Unlock(); a.drop() }

func (a *InvAPI) desk() InvDesk { a.mu.Lock(); defer a.mu.Unlock(); return a.d }

// Handler serves the viewer GET routes.
func (a *InvAPI) Handler() http.Handler {
	m := http.NewServeMux()
	m.HandleFunc("/api/inventory", func(w http.ResponseWriter, r *http.Request) {
		d := a.desk()
		if d == nil {
			writeJSON(w, http.StatusServiceUnavailable, policyReply{Code: "off", Error: "the inventory page is off or the hero is not known yet"})
			return
		}
		b, err := a.viewJSON(d)
		if err != nil {
			writeJSON(w, http.StatusInternalServerError, policyReply{Code: "internal", Error: "could not build the inventory"})
			return
		}
		h := w.Header()
		h.Set("Content-Type", "application/json; charset=utf-8")
		h.Set("Cache-Control", "no-store")
		h.Set("X-Content-Type-Options", "nosniff")
		w.Write(b)
	})
	m.HandleFunc("/watch/inventory", func(w http.ResponseWriter, r *http.Request) {
		pactHdr(w, "text/html; charset=utf-8")
		io.WriteString(w, invPage)
	})
	m.HandleFunc("/watch/inventory.js", func(w http.ResponseWriter, r *http.Request) {
		b, err := webFS.ReadFile("web/inventory.js")
		if err != nil {
			http.NotFound(w, r)
			return
		}
		pactHdr(w, "text/javascript; charset=utf-8")
		w.Write(b)
	})
	return readOnly(m)
}

type invBody struct {
	ID   uint32 `json:"id"`
	Mark string `json:"mark"`
	Slot string `json:"slot"`
}

// PostHandler serves the three POST routes.
func (a *InvAPI) PostHandler(o AnswerOptions) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var b invBody
		if !policyGate(o, w, r, &b) {
			return
		}
		d := a.desk()
		if d == nil {
			writeJSON(w, http.StatusServiceUnavailable, policyReply{Code: "off", Error: "the inventory page is off or the hero is not known yet"})
			return
		}
		defer a.drop()
		var (
			err error
			res any
		)
		switch r.URL.Path {
		case "/api/inventory/mark":
			err = d.SetMark(b.ID, b.Mark)
		case "/api/inventory/equip":
			res, err = d.RequestEquip(b.ID, b.Slot)
		case "/api/inventory/unpin":
			err = d.Unpin(b.Slot)
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
			writeJSON(w, http.StatusInternalServerError, policyReply{Code: "internal", Error: "could not save the change"})
		}
	})
}
