package levelsim

import (
	"bufio"
	"encoding/json"
	"io"
	"math"
	"strings"
	"time"
)

// Extract reduces a bot log (JSON lines: `minute`, `arrival`, `d2d: Save and Exit ...`) to a Run entry for
// data/runs.json. t0 is the hero's first moment; zero means the first line of the log. Raw lines are never
// kept: only minute rows, arrivals and restart times (docs/levelsim.md section 3, "re-calibrate").
func Extract(r io.Reader, id int, build string, t0 time.Time, endMin float64) (Run, error) {
	run := Run{ID: id, Build: build, EndMin: endMin, Tag: "extracted by levelsim -extract"}
	sc := bufio.NewScanner(r)
	sc.Buffer(make([]byte, 1<<20), 1<<24)
	for sc.Scan() {
		line := sc.Text()
		if !strings.HasPrefix(line, "{") {
			continue
		}
		var l struct {
			Time  string  `json:"time"`
			Msg   string  `json:"msg"`
			Area  float64 `json:"area"`
			Clvl  float64 `json:"clvl"`
			XP    float64 `json:"xp"`
			Kills float64 `json:"kills"`
		}
		if json.Unmarshal([]byte(line), &l) != nil {
			continue
		}
		t, err := time.Parse(time.RFC3339Nano, l.Time)
		if err != nil {
			continue
		}
		if t0.IsZero() {
			t0 = t
		}
		m := math.Round(t.Sub(t0).Minutes()*100) / 100
		if m < 0 || (endMin > 0 && m > endMin+1) {
			continue
		}
		switch {
		case l.Msg == "minute":
			run.Minutes = append(run.Minutes, []float64{m, l.Area, l.Clvl, l.XP, l.Kills})
			run.Observed.Clvl, run.Observed.XP, run.Observed.Kills = int(l.Clvl), l.XP, l.Kills
		case l.Msg == "arrival":
			run.Arrivals = append(run.Arrivals, []float64{m, l.Area})
		case strings.HasPrefix(l.Msg, "d2d: Save and Exit"):
			run.Restarts = append(run.Restarts, m)
		}
	}
	return run, sc.Err()
}
