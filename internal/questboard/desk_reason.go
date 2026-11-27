package questboard

import (
	"strconv"
	"strings"
	"time"
)

// Reason codes in the API (card bsm6, design appendix B.1/B.2): every live chunk row says why it does not move, as a
// closed code plus the sentence the one table builds. The page picks buttons from the code, never from the sentence.

// ReasonView is the reason of a chunk row, offer or run.
type ReasonView struct {
	Code   string `json:"code"`
	Text   string `json:"text"`
	Remedy Remedy `json:"remedy,omitempty"`
	// Since and Clears fill in for a park (card questresume): when it began and, in words, what lifts it (the policy of
	// resume.go, or only the player's Try again). Both are empty for every other reason.
	Since  *time.Time `json:"since,omitempty"`
	Clears string     `json:"clears,omitempty"`
}

// ChangedRow is one run a verb touched, with its state and reason after the verb.
type ChangedRow struct {
	Quest  string      `json:"quest"`
	Chunk  string      `json:"chunk"`
	State  State       `json:"state"`
	Reason *ReasonView `json:"reason,omitempty"`
}

// CodeHandInReady is the offer code of a chunk whose field work the game already counts done (appendix B.1, C5).
const CodeHandInReady = "handin_ready"

// CodeHandInWaiting is the hold code of a hand-in tap queued for the bot loop that has not picked it up.
const CodeHandInWaiting = "handin_waiting"

// reasonView builds the view of a code; a placeholder the arguments leave open is replaced by the producer's own
// words (why), or, without them, the sentence stays as the table has it so the gap is visible.
func reasonView(code string, args map[string]string, why string) *ReasonView {
	info, ok := ReasonOf(code)
	if !ok {
		return nil
	}
	if args == nil {
		args = map[string]string{}
	}
	if _, has := args["why"]; !has && why != "" {
		args["why"] = why
	}
	text := Sentence(code, args)
	if strings.Contains(text, "{") && why != "" {
		text = why
	}
	return &ReasonView{Code: code, Text: text, Remedy: info.Remedy}
}

// chunkReason is the reason of one chunk row now: the wire state decides, the stored run supplies a park's code and
// the head's hold supplies a hold's. nil means the chunk moves (a running step) or has nothing live.
func (d *Desk) chunkReason(quest string, cv ChunkView, run *Run, a *Active, handing bool, titles map[string]string) *ReasonView {
	head := a != nil && a.Quest == quest && a.Chunk == cv.ID
	switch cv.State {
	case Accepted, InProgress:
		if !head {
			if cv.State == Accepted {
				return reasonView("waits_for", map[string]string{"part": titles["head"]}, "")
			}
			return nil
		}
		if a.Hold != "" {
			return reasonView(a.Hold, map[string]string{"quest": titles["quest"]}, a.HoldWhy)
		}
	case Awaiting:
		args := map[string]string{"quest": titles["quest"], "npc": titles["npc"]}
		if handing {
			if r := d.handInWaitReason(); r != nil {
				return r
			}
			return reasonView("handin_in_town", map[string]string{"quest": titles["quest"], "npc": titles["npc"]}, "")
		}
		if run != nil && run.Status == RunHandingIn {
			return reasonView("confirming", args, "")
		}
		return reasonView("handin_wanted", args, "")
	case Blocked:
		code := "gave_up"
		if run != nil && run.Status == RunParked && run.Code != "" {
			code = run.Code
		}
		args := map[string]string{"quest": titles["quest"], "n": "3", "why": cv.Why}
		if run != nil {
			args["n"] = strconv.Itoa(max(run.Failures, run.Deaths, 1))
		}
		rv := reasonView(code, args, cv.Why)
		if run != nil && run.Status == RunParked && rv != nil {
			if code == "died_too_often" {
				// the stored code stays (old pages, plan files); the API says what the player sees: a pause, since when, what clears it
				rv.Code = "paused_after_deaths"
				rv.Text = Sentence(rv.Code, args)
			}
			if !run.ParkedAt.IsZero() {
				t := run.ParkedAt
				rv.Since = &t
			}
			if x, _ := d.exec(); x != nil {
				rv.Clears = x.ResumeNote(*run)
			}
		}
		return rv
	case Done:
		if !cv.Confirmed && run != nil {
			return reasonView("confirming", nil, "")
		}
	}
	return nil
}

// SMRuns returns the live runs of the machine by chunk key for the current difficulty; nil when the machine is off.
func (e *Exec) SMRuns() map[string]Run {
	e.mu.Lock()
	defer e.mu.Unlock()
	if !e.o.SM || !e.o.Enabled {
		return nil
	}
	m, err := e.st.LoadSM(e.smDone())
	if err != nil {
		return nil
	}
	out := map[string]Run{}
	for _, r := range m.Live(e.b, Difficulties[e.env.Diff()]) {
		out[ChunkKey(r.Quest, r.Chunk)] = r
	}
	return out
}

// handInReady is the row reason of a chunk that reads awaiting with no run (appendix B.1 "why Start is not offered").
func handInReady() *ReasonView {
	return &ReasonView{Code: CodeHandInReady, Text: "The field work is already done: Start goes straight to the hand-in.", Remedy: RemedyPlayer}
}

// SMOn reports the knob quests.boardSM.
func (e *Exec) SMOn() bool { e.mu.Lock(); defer e.mu.Unlock(); return e.o.SM }

// handInWaitReason is the reason of a hand-in tap the bot loop has not served yet: the holder in words, from the
// bot side. nil when the loop is serving it (or no holder is known).
func (d *Desk) handInWaitReason() *ReasonView {
	if d.cfg.HandInWhy == nil {
		return nil
	}
	if why := d.cfg.HandInWhy(); why != "" {
		return reasonView(CodeHandInWaiting, nil, why)
	}
	return nil
}
