package scenario

import (
	"encoding/json"
	"fmt"
	"math"
	"strings"

	"d2lod-bot/internal/questboard"
)

// TapScript is the file behind d2bot --scripted-taps: the taps of the person on the quest board, decided inside the bot
// on the stepping goroutine (design docs/design/fable-review-2026-11-21/proving-ground-gate.md section 2.8 step 3). The
// HTTP tapper of cmd/d2scenario stays for sandbox runs; this one has no wall clock, so its taps are a function of the
// stepped ticks.
type TapScript struct {
	// Quest is one quest id, a comma list, or "*" (every quest, in board order), like the HTTP tapper.
	Quest string `json:"quest"`
	// Chunks limits the taps to these chunk ids; empty: every chunk.
	Chunks []string `json:"chunks,omitempty"`
	// EveryS is the virtual-second grid the board is evaluated on (0: 30, the HTTP tapper's grid).
	EveryS float64 `json:"every_s,omitempty"`
}

// ParseTapScript reads and checks a tap script.
func ParseTapScript(b []byte) (TapScript, error) {
	var s TapScript
	dec := json.NewDecoder(strings.NewReader(string(b)))
	dec.DisallowUnknownFields()
	if err := dec.Decode(&s); err != nil {
		return s, fmt.Errorf("scripted taps: %w", err)
	}
	if strings.TrimSpace(s.Quest) == "" {
		return s, fmt.Errorf("scripted taps: quest is empty (an id, a comma list or \"*\")")
	}
	if s.EveryS < 0 {
		return s, fmt.Errorf("scripted taps: every_s %v is negative", s.EveryS)
	}
	if s.EveryS == 0 {
		s.EveryS = 30
	}
	return s, nil
}

// TapRec is one tap made: the verb and chunk, and the virtual second it was made at (the grid point's crossing).
type TapRec struct {
	Tap string  `json:"tap"` // "start scroll", "handin read"
	At  float64 `json:"at"`
}

// TapEngine decides the taps. It holds no clock and no desk: Poll is given the board view and the bot's virtual second and
// calls press for the tap it wants, so the caller owns when it runs (the stepping goroutine) and what pressing does.
type TapEngine struct {
	S    TapScript
	next float64
	// handing is "quest/chunk" of a hand-in tap whose button the board still shows: no Start tap meanwhile (a Start now
	// cancels the hand-in dialogue, necro.act1-stage1 2026-11-23).
	handing string
	Taps    []TapRec
}

// NewTapEngine starts the grid at the first multiple of EveryS.
func NewTapEngine(s TapScript) *TapEngine { return &TapEngine{S: s, next: s.EveryS} }

func (e *TapEngine) wants(id string) bool {
	if e.S.Quest == "*" {
		return true
	}
	for _, q := range strings.Split(e.S.Quest, ",") {
		if strings.TrimSpace(q) == id {
			return true
		}
	}
	return false
}

func (e *TapEngine) allowed(chunk string) bool {
	if len(e.S.Chunks) == 0 {
		return true
	}
	for _, c := range e.S.Chunks {
		if c == chunk {
			return true
		}
	}
	return false
}

// Due reports whether the virtual second v is at or past the next grid point (cheap: the caller reads the board only then).
func (e *TapEngine) Due(v float64) bool { return v >= e.next }

// Poll evaluates the board once at virtual second v, when v is on or past the grid point, and returns the tap it pressed ("" for
// none). press does the tap (verb "start" or "handin") and may be the desk's Start or HandIn. Same rules as the HTTP tapper's
// Step: the button the row renders is the tap, nothing is pressed while a hand-in dialogue still shows its button, and a
// Start waits for the running chunk.
func (e *TapEngine) Poll(v questboard.View, at float64, press func(verb, quest, chunk string)) string {
	if !e.Due(at) {
		return ""
	}
	e.next = (math.Floor(at/e.S.EveryS) + 1) * e.S.EveryS
	if e.handing != "" && e.handInPending(v) {
		return ""
	}
	if v.HandIn != nil && (v.Running != nil || v.Awaiting != nil) {
		return ""
	}
	for _, q := range v.Quests {
		if !e.wants(q.ID) {
			continue
		}
		for _, c := range q.Chunks {
			a := c.Action
			if a == nil || a.Disabled || !e.allowed(c.ID) || a.Verb != "start" && a.Verb != "handin" {
				continue
			}
			if a.Verb == "handin" && v.HandIn != nil || a.Verb == "start" && v.Running != nil {
				continue
			}
			if a.Verb == "handin" {
				e.handing = q.ID + "/" + c.ID
			}
			tap := a.Verb + " " + c.ID
			e.Taps = append(e.Taps, TapRec{Tap: tap, At: at})
			press(a.Verb, q.ID, c.ID)
			return tap
		}
	}
	return ""
}

func (e *TapEngine) handInPending(v questboard.View) bool {
	for _, q := range v.Quests {
		for _, c := range q.Chunks {
			if q.ID+"/"+c.ID != e.handing {
				continue
			}
			if c.State == "awaiting_handin" || c.Action != nil && c.Action.Verb == "handin" && !c.Action.Disabled {
				return true
			}
		}
	}
	e.handing = ""
	return false
}
