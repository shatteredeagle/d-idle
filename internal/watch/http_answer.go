package watch

import (
	"crypto/subtle"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"mime"
	"net/http"
	"strings"

	"d2lod-bot/internal/currency"
)

// AnswerOptions guards the answer route, which carries every write of the screen.
type AnswerOptions struct {
	// Token, when set, must arrive as "Authorization: Bearer <token>" or in the d2dview cookie.
	Token string
	// Origins are the page origins allowed to post (the viewer's, e.g. "http://host:8891"). A request with an Origin
	// header outside the list is refused; one without (a tool) passes the other checks.
	Origins []string
}

const maxBody = 4096

type answerBody struct {
	Thought *int   `json:"thought"`
	Answer  string `json:"answer"`
	Yes     *bool  `json:"yes"`
	Key     string `json:"key"`
	Confirm bool   `json:"confirm"` // revoke-all must say so
	Kind    string `json:"kind"`
	Area    string `json:"area"`
	Minutes int    `json:"minutes"`
	Until   string `json:"until"`   // orders: "" or "minutes" (timed) or "forever"; "level" arrives with slice 2
	Replace bool   `json:"replace"` // orders: cancel the open orders first ("Go now")
	ID      *int   `json:"id"`
	Approve bool   `json:"approve"` // orders above the zone range: pay the charter (docs/tally-strip.md)
}

type reply struct {
	OK      bool     `json:"ok"`
	Code    string   `json:"code,omitempty"`
	Error   string   `json:"error,omitempty"`
	Allowed []string `json:"allowed,omitempty"`
	Revoked int      `json:"revoked,omitempty"`
	Order   *Order   `json:"order,omitempty"`
	Charter *Charter `json:"charter,omitempty"` // the price and the hero's warning of an order above the zone range
}

// AnswerHandler serves POST /api/answer, /api/confirm, /api/revoke, /api/revoke-all, /api/orders and /api/orders/cancel.
// It is meant for its own listener, apart from the read-only viewer.
func (h *Hub) AnswerHandler(o AnswerOptions) http.Handler {
	routes := map[string]func(b answerBody) (int, reply){
		"/api/answer":        h.postAnswer,
		"/api/confirm":       h.postConfirm,
		"/api/revoke":        h.postRevoke,
		"/api/revoke-all":    h.postRevokeAll,
		"/api/orders":        h.postOrder,
		"/api/orders/cancel": h.postCancel,
	}
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		origin := r.Header.Get("Origin")
		allowed := origin != "" && containsStr(o.Origins, origin)
		if allowed {
			w.Header().Set("Access-Control-Allow-Origin", origin)
			w.Header().Set("Access-Control-Allow-Credentials", "true")
			w.Header().Set("Vary", "Origin")
		}
		fail := func(code int, c, msg string) { writeJSON(w, code, reply{Code: c, Error: msg}) }
		if r.Method == http.MethodOptions {
			if !allowed {
				fail(http.StatusForbidden, "origin", "origin not allowed")
				return
			}
			w.Header().Set("Access-Control-Allow-Methods", "POST")
			w.Header().Set("Access-Control-Allow-Headers", "Content-Type, Authorization")
			w.Header().Set("Access-Control-Max-Age", "600")
			w.WriteHeader(http.StatusNoContent)
			return
		}
		if r.Method != http.MethodPost {
			w.Header().Set("Allow", "POST, OPTIONS")
			fail(http.StatusMethodNotAllowed, "method", "POST only")
			return
		}
		if origin != "" && !allowed {
			fail(http.StatusForbidden, "origin", "origin not allowed")
			return
		}
		if !o.tokenOK(r) {
			fail(http.StatusUnauthorized, "token", "unauthorized")
			return
		}
		fn := routes[r.URL.Path]
		if fn == nil {
			fail(http.StatusNotFound, "route", "no such route")
			return
		}
		// A JSON content type forces a CORS preflight on a cross-origin form or fetch, which the origin check then judges.
		if mt, _, err := mime.ParseMediaType(r.Header.Get("Content-Type")); err != nil || mt != "application/json" {
			fail(http.StatusUnsupportedMediaType, "content-type", "send application/json")
			return
		}
		var b answerBody
		dec := json.NewDecoder(io.LimitReader(r.Body, maxBody+1))
		dec.DisallowUnknownFields()
		if err := dec.Decode(&b); err != nil || dec.More() {
			fail(http.StatusBadRequest, "body", "body must be one small JSON object with known fields")
			return
		}
		code, rep := fn(b)
		writeJSON(w, code, rep)
	})
}

func (o AnswerOptions) tokenOK(r *http.Request) bool {
	if o.Token == "" {
		return true
	}
	got := strings.TrimPrefix(r.Header.Get("Authorization"), "Bearer ")
	if got == "" {
		if c, err := r.Cookie("d2dview"); err == nil {
			got = c.Value
		}
	}
	return subtle.ConstantTimeCompare([]byte(got), []byte(o.Token)) == 1
}

func containsStr(s []string, v string) bool {
	for _, x := range s {
		if x == v {
			return true
		}
	}
	return false
}

// status maps the Hub's errors to an HTTP status and a stable code.
func status(err error) (int, reply) {
	r := reply{Error: err.Error()}
	switch {
	case errors.Is(err, ErrUnknownThought), errors.Is(err, ErrNoRule), errors.Is(err, ErrNoOrder):
		return 404, withCode(r, "not-found")
	case errors.Is(err, ErrAlreadyAnswered):
		return 409, withCode(r, "already-answered")
	case errors.Is(err, ErrThoughtFaded):
		return 410, withCode(r, "faded")
	case errors.Is(err, ErrBadChoice):
		return 400, withCode(r, "bad-choice")
	case errors.Is(err, ErrWrongThought):
		return 400, withCode(r, "wrong-kind")
	case errors.Is(err, ErrBadOrder):
		return 400, withCode(r, "bad-order")
	case errors.Is(err, currency.ErrCharterNeeded):
		return 402, withCode(r, "charter-needed")
	case errors.Is(err, currency.ErrInsufficient):
		return 402, withCode(r, "insufficient")
	case errors.Is(err, ErrOrdersFull), errors.Is(err, errOrderClosed):
		return 409, withCode(r, "order-state")
	}
	return 500, withCode(r, "internal")
}

func withCode(r reply, c string) reply { r.Code = c; return r }

func (h *Hub) postAnswer(b answerBody) (int, reply) {
	if b.Thought == nil {
		return 400, reply{Code: "bad-request", Error: "thought is required"}
	}
	if err := h.Answer(*b.Thought, b.Answer); err != nil {
		code, r := status(err)
		if code == 400 && r.Code == "bad-choice" {
			if info, ok := h.cfg.Store.Thought(*b.Thought); ok {
				for _, c := range info.Choices {
					r.Allowed = append(r.Allowed, c.ID)
				}
				r.Allowed = append(r.Allowed, "notsure")
			}
		}
		return code, r
	}
	return 200, reply{OK: true}
}

func (h *Hub) postConfirm(b answerBody) (int, reply) {
	if b.Thought == nil || b.Yes == nil {
		return 400, reply{Code: "bad-request", Error: "thought and yes are required"}
	}
	if err := h.Confirm(*b.Thought, *b.Yes); err != nil {
		return status(err)
	}
	return 200, reply{OK: true}
}

func (h *Hub) postRevoke(b answerBody) (int, reply) {
	if b.Key == "" {
		return 400, reply{Code: "bad-request", Error: "key is required"}
	}
	if err := h.Revoke(b.Key); err != nil {
		return status(err)
	}
	return 200, reply{OK: true, Revoked: 1}
}

func (h *Hub) postRevokeAll(b answerBody) (int, reply) {
	if !b.Confirm {
		return 400, reply{Code: "bad-request", Error: "send {\"confirm\":true} to revoke every rule"}
	}
	n, err := h.RevokeAll()
	if err != nil {
		return status(err)
	}
	return 200, reply{OK: true, Revoked: n}
}

func (h *Hub) postOrder(b answerBody) (int, reply) {
	r := OrderRequest{Kind: b.Kind, Area: b.Area, Minutes: b.Minutes, Replace: b.Replace}
	switch b.Until {
	case "", "minutes":
	case "forever":
		r.Forever = true
	default:
		return status(fmt.Errorf("%w: until must be minutes or forever (until-level comes later)", ErrBadOrder))
	}
	o, ch, err := h.place(r, b.Approve)
	if err != nil {
		code, rep := status(err)
		rep.Charter = ch
		return code, rep
	}
	return 200, reply{OK: true, Order: &o, Charter: ch}
}

func (h *Hub) postCancel(b answerBody) (int, reply) {
	if b.ID == nil {
		return 400, reply{Code: "bad-request", Error: "id is required"}
	}
	if err := h.CancelOrder(*b.ID); err != nil {
		return status(err)
	}
	return 200, reply{OK: true}
}
