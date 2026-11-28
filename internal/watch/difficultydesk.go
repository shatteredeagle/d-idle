package watch

import (
	"errors"
	"fmt"
	"mime"
	"net/http"
	"strings"
	"sync"
	"time"

	"d2lod-bot/internal/questboard"
)

// The owner's go for the difficulty hop (bot.diffReadyV2, docs/diffready.md): a Return screen section that shows the
// readiness line and, only when the hero is ready, one button "Ready for Nightmare: Go"; POST /api/difficulty/go on
// the answer listener records the go. The bot never hops alone (user decision 2026-11-27).

const (
	DifficultySectionID = "difficulty"
	DifficultyPostRoute = "/api/difficulty/go"
)

// DifficultyOffer is the bot's view of the next difficulty (bot.DifficultyOffer, copied so watch does not import bot).
type DifficultyOffer struct {
	Target  string // "" hides the section
	Ready   bool
	Pending bool // the go was given and the hop has not happened yet
	Line    string
	Missing []string
}

// DifficultyConfig wires the section and the POST. Offer reads the bot; Go records the go or refuses with the reason.
type DifficultyConfig struct {
	Offer func() DifficultyOffer
	Go    func(target string) (string, error)
	Now   func() time.Time // nil: time.Now (the hourly budget of the POST)

	mu   sync.Mutex
	gone []time.Time // POSTs that reached Go in the last hour
}

// difficultyGoPerHour is the budget of the page door, the same number as the operator verb's (ops.PerQuestHour).
const difficultyGoPerHour = 6

// admit counts one go against the hourly budget.
func (c *DifficultyConfig) admit() bool {
	now := time.Now()
	if c.Now != nil {
		now = c.Now()
	}
	c.mu.Lock()
	defer c.mu.Unlock()
	i := 0
	for i < len(c.gone) && now.Sub(c.gone[i]) >= time.Hour {
		i++
	}
	c.gone = c.gone[i:]
	if len(c.gone) >= difficultyGoPerHour {
		return false
	}
	c.gone = append(c.gone, now)
	return true
}

func titleCase(s string) string {
	if s == "" {
		return s
	}
	return strings.ToUpper(s[:1]) + s[1:]
}

// DifficultySection is the section; hidden while there is nothing to offer (knob off, Hell, no snapshot).
func DifficultySection(c *DifficultyConfig) ReturnSection {
	return SectionFunc(func(*ReturnData) *SectionView {
		if c == nil || c.Offer == nil {
			return nil
		}
		o := c.Offer()
		if o.Target == "" {
			return nil
		}
		sv := &SectionView{ID: DifficultySectionID, Title: "Next difficulty: " + titleCase(o.Target), After: true}
		name := titleCase(o.Target)
		switch {
		case o.Pending:
			sv.Lines = []string{"Your go for " + name + " is recorded. The hero hops at its next safe moment."}
		case o.Ready:
			sv.Lines = []string{o.Line, "The hero never hops on its own. Press the button when you want it to."}
			sv.Buttons = []SectionButton{{Label: "Ready for " + name + ": Go", Row: o.Target, Act: "difficulty"}}
		default:
			sv.Lines = append([]string{"Not ready for " + name + " yet. Missing:"}, o.Missing...)
			if len(o.Missing) == 0 {
				sv.Lines = []string{o.Line}
			}
		}
		return sv
	})
}

type difficultyBody struct {
	Target string `json:"target"`
}

// DifficultyPostHandler is POST /api/difficulty/go {target} on the answer listener: origin list, token, JSON only.
// A refusal (not ready, measured run) is 409 with the reason in error; more than difficultyGoPerHour is 429. Every
// request that names a target writes one audit line (client class and agent family, never an address), like the
// operator verb go-difficulty.
func DifficultyPostHandler(c *DifficultyConfig, o AnswerOptions) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		origin := r.Header.Get("Origin")
		allowed := origin != "" && containsStr(o.Origins, origin)
		if allowed {
			w.Header().Set("Access-Control-Allow-Origin", origin)
			w.Header().Set("Access-Control-Allow-Credentials", "true")
			w.Header().Set("Vary", "Origin")
		}
		fail := func(code int, c, msg string) { writeJSON(w, code, hallReply{Code: c, Error: msg}) }
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
		var b difficultyBody
		if err := decodeSmall(r, &b); err != nil || (b.Target != "nightmare" && b.Target != "hell") {
			fail(http.StatusBadRequest, "body", "send {\"target\":\"nightmare\"} or {\"target\":\"hell\"}")
			return
		}
		if c == nil || c.Go == nil {
			fail(http.StatusServiceUnavailable, "off", errors.New("the difficulty hop is not available").Error())
			return
		}
		ev := questboard.AuditEvent{Action: "go-difficulty", Quest: b.Target, Path: safePath(r.URL.Path),
			Remote: remoteClass(r.RemoteAddr), Agent: agentFamily(r.UserAgent()), Status: http.StatusOK, OK: true}
		defer func() { logQuestAction(ev) }()
		refuse := func(code int, c, msg string) {
			ev.OK, ev.Status, ev.Code, ev.Reason = false, code, c, msg
			fail(code, c, msg)
		}
		if !c.admit() {
			refuse(http.StatusTooManyRequests, "rate_limited", fmt.Sprintf("the difficulty go was pressed %d times in the last hour; wait", difficultyGoPerHour))
			return
		}
		if _, err := c.Go(b.Target); err != nil {
			refuse(http.StatusConflict, "not-ready", err.Error())
			return
		}
		writeJSON(w, 200, hallReply{OK: true, Changed: true})
	})
}
