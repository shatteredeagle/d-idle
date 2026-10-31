// Package stepgroup maps the runner's step text to a coarse activity group. cmd/d2soak (progress report) and the bot's
// minute log line share it, so the live "state" field and the offline time-in-state report agree.
package stepgroup

import (
	"regexp"
	"strings"
)

// groups, first match wins, on the step text after digits became N. Unknown texts fall into "other".
var groups = []struct{ prefix, group string }{
	{"Quest: fighting (Level up", "farming"},
	{"Quest: fighting", "quest fight"},
	{"Quest:", "quest travel"}, // travelling, in town (talk to an NPC)
	{"Recovery: farming", "recovery"},
	{"Game lost", "recovery"},
	{"Failsafe", "recovery"},
	{"Re-entering", "recovery"},
	{"Restocking after death", "dead-recovering"},
	{"Retrieving the corpse", "dead-recovering"},
	{"Chores in town", "town chores"},
	{"Gear:", "town chores"},
	{"Going to town", "town chores"},
	{"Returning to", "town chores"},
	{"Settling in town", "town chores"},
	{"Waiting", "waiting"},
	{"Paused", "waiting"},
}

// Order is the display order of the groups.
var Order = []string{"farming", "quest travel", "quest fight", "town chores", "recovery", "waiting", "dead-recovering", "other"}

var digits = regexp.MustCompile(`[0-9]+`)

// Norm replaces digit runs by N.
func Norm(s string) string { return digits.ReplaceAllString(s, "N") }

// Of returns the group of a step text and whether a table entry matched.
func Of(step string) (group string, known bool) {
	n := Norm(step)
	for _, g := range groups {
		if strings.HasPrefix(n, g.prefix) {
			return g.group, true
		}
	}
	return "other", false
}
