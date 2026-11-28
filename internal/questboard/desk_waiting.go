package questboard

import "strings"

// The auto-start words of a quest (card questauto-visible): on is the player's opt-in in the plan file, never is the
// quest data's autoStart: never (a hazard), off is allowed but not opted in.
const (
	AutoOn    = "on"
	AutoOff   = "off"
	AutoNever = "never"
)

// What a ready part waits for. Only the Start tap, the operator start verb and AutoTick (opted in, readiness safe)
// accept a chunk; a part that AutoTick will take reports "".
const (
	WaitStartTap  = "start_tap"
	WaitNeverAuto = "never_auto"
)

func autoWord(on, ok bool) string {
	switch {
	case !ok:
		return AutoNever
	case on:
		return AutoOn
	}
	return AutoOff
}

func waitingFor(on, ok bool, word Word) string {
	switch {
	case !ok:
		return WaitNeverAuto
	case !on || word != ReadySafe:
		return WaitStartTap // an opted-in part that is not safe is never started by itself either
	}
	return ""
}

// WaitingLines is the Return screen's "Waiting for you to start" list: ready parts nothing will start, while the board
// runs nothing. nil when there are none.
func (d *Desk) WaitingLines() []string {
	v := d.View()
	if !v.Enabled || v.Running != nil || v.Awaiting != nil || v.Blocked != nil || v.Paused {
		return nil
	}
	var names []string
	for _, q := range v.Quests {
		for _, c := range q.Chunks {
			if c.WaitingFor != "" {
				names = append(names, q.Title+": "+c.Title)
			}
		}
	}
	if len(names) == 0 {
		return nil
	}
	return []string{"Waiting for you to start: " + strings.Join(names, ", ") + "."}
}
