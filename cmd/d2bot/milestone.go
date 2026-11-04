package main

import (
	"fmt"
	"time"

	"d2lod-bot/internal/quest"
)

// act1Milestones are the Act 1 quest words whose rising edge to Done is logged (docs/act1check.md). Order is the
// campaign order; act2 is the "able to go to Act 2" word (act1_done), the one the Go East dialog row gates on.
var act1Milestones = []struct{ Name, Quest string }{
	{"den", "den_of_evil"}, {"bloodraven", "sisters_burial_grounds"}, {"cain", "search_for_cain"},
	{"tower", "forgotten_tower"}, {"tools", "tools_of_the_trade"}, {"andariel", "sisters_to_the_slaughter"}, {"act2", "act1_done"},
}

// logMilestones logs `milestone name=.. word=.. clvl=.. virtual=..` once per Act 1 word when it first reads Done, and
// `act1: andariel killed words 6=.. 7=..` once when the Andariel word first reads Done. Logging only: it reads the
// same memory words logMem does and never changes what the reader returns. clvl is the save level (-1 when no save
// was read; the save lags the game until Save and Exit); virtual is minutes since the first read on the reader's clock.
func (r *saveQuestReader) logMilestones(diff int) {
	if r.mem == nil || r.log == nil || diff != 0 {
		return
	}
	words := map[string]uint16{}
	for _, m := range act1Milestones {
		if w, ok := r.mem.Word(diff, m.Quest); ok {
			words[m.Quest] = w
		}
	}
	now := time.Now()
	if r.now != nil {
		now = r.now()
	}
	type line struct {
		msg  string
		args []any
	}
	var lines []line
	r.mu.Lock()
	if r.t0.IsZero() {
		r.t0 = now
	}
	clvl := -1
	if r.have {
		clvl = int(r.prog.Level)
	}
	virt := now.Sub(r.t0).Minutes()
	andy := false
	for _, m := range act1Milestones {
		w, ok := words[m.Quest]
		if !ok || r.msDone[m.Name] {
			continue
		}
		if st, _ := quest.Interpret(w); st != quest.Done {
			continue
		}
		if r.msDone == nil {
			r.msDone = map[string]bool{}
		}
		r.msDone[m.Name] = true
		lines = append(lines, line{"milestone", []any{"name", m.Name, "word", fmt.Sprintf("0x%04x", w), "clvl", clvl, "virtual", fmt.Sprintf("%.1f", virt)}})
		andy = andy || m.Name == "andariel"
	}
	if andy && !r.andyLogged {
		r.andyLogged = true
		lines = append(lines, line{fmt.Sprintf("act1: andariel killed words 6=0x%04x 7=0x%04x", words["sisters_to_the_slaughter"], words["act1_done"]), nil})
	}
	r.mu.Unlock()
	for _, l := range lines {
		r.log(l.msg, l.args...)
	}
}
