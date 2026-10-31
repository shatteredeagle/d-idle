package goals

import (
	"fmt"
	"strconv"
	"strings"
)

// ParseRuns turns the goal text "runs:NAME:N" (also "runs:NAME" = one run, "runs:auto:N") into a Runs goal. NAME may
// use "_" for spaces ("runs:Eldritch_Shenk:5"). cmd/d2bot's --goals parser calls it for the "runs" kind.
func ParseRuns(text string) (Goal, error) {
	rest, ok := strings.CutPrefix(text, "runs:")
	if !ok {
		return Goal{}, fmt.Errorf("runs goal %q must start with runs:", text)
	}
	name, num, hasN := strings.Cut(rest, ":")
	name = strings.TrimSpace(strings.ReplaceAll(name, "_", " "))
	if name == "" {
		return Goal{}, fmt.Errorf("runs goal %q: missing run name (use runs:NAME:N)", text)
	}
	n := 1
	if hasN {
		v, err := strconv.Atoi(strings.TrimSpace(num))
		if err != nil || v < 1 {
			return Goal{}, fmt.Errorf("runs goal %q: N must be a positive number", text)
		}
		n = v
	}
	return Goal{Type: Runs, Params: Params{Area: name, Count: n}}, nil
}
