package main

import (
	"bufio"
	"fmt"
	"io"
	"sort"
	"strconv"
	"strings"
	"time"
)

// FightHour sums the per-fight telemetry lines ("fight area= diff= danger= kills= damage= minlife= potions=
// dodges= sec=", emitted by combat.FightTelemetry) of one clock hour.
type FightHour struct {
	Hour    time.Time // truncated to the hour; zero for lines without a time= field
	Fights  int
	Kills   int
	Damage  int
	Potions int
	Dodges  int
	Sec     int
	MinLife int // lowest minlife of the hour, 100 when none
}

// ParseFightLines reads a bot log (JSON lines, or slog text lines; the fight line is inside msg="combat fight ...") and returns
// one FightHour per hour, oldest first. Unrelated and malformed lines are skipped.
func ParseFightLines(r io.Reader) ([]FightHour, error) {
	by := map[time.Time]*FightHour{}
	sc := bufio.NewScanner(r)
	sc.Buffer(make([]byte, 1<<20), 1<<20)
	for sc.Scan() {
		body, ts, ok := telemetryBody(sc.Text(), "fight area=")
		if !ok {
			continue
		}
		kv := map[string]int{}
		for _, f := range strings.Fields(body) {
			if k, v, ok := strings.Cut(f, "="); ok {
				if n, err := strconv.Atoi(strings.Trim(v, `"`)); err == nil {
					kv[k] = n
				}
			}
		}
		if _, ok := kv["kills"]; !ok {
			continue
		}
		var hr time.Time
		if t, err := time.Parse(time.RFC3339Nano, ts); err == nil {
			hr = t.Truncate(time.Hour)
		}
		h := by[hr]
		if h == nil {
			h = &FightHour{Hour: hr, MinLife: 100}
			by[hr] = h
		}
		h.Fights++
		h.Kills += kv["kills"]
		h.Damage += kv["damage"]
		h.Potions += kv["potions"]
		h.Dodges += kv["dodges"]
		h.Sec += kv["sec"]
		h.MinLife = min(h.MinLife, kv["minlife"])
	}
	out := make([]FightHour, 0, len(by))
	for _, h := range by {
		out = append(out, *h)
	}
	sort.Slice(out, func(a, b int) bool { return out[a].Hour.Before(out[b].Hour) })
	return out, sc.Err()
}

// FightText renders the hourly table for the soak report.
func FightText(hs []FightHour) string {
	if len(hs) == 0 {
		return "fights: no telemetry lines\n"
	}
	var b strings.Builder
	b.WriteString("fights per hour: hour fights kills damage potions dodges minlife%\n")
	for _, h := range hs {
		when := "(no time)"
		if !h.Hour.IsZero() {
			when = h.Hour.Format("2006-01-02 15:00")
		}
		fmt.Fprintf(&b, "  %s %d %d %d %d %d %d\n", when, h.Fights, h.Kills, h.Damage, h.Potions, h.Dodges, h.MinLife)
	}
	return b.String()
}
