// Command d2firsthour computes the headline "fresh hero, first 60 realtime minutes" numbers from a bot log (slog text
// or JSON, as `docker logs` of the sandbox prints it) and appends one row to docs/probes/firsthour.md (TASK-245,
// docs/first-hour.md). Read-only on the log; it fails instead of guessing when the log does not cover the window.
package main

import (
	"bufio"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"sort"
	"strconv"
	"strings"
	"time"
)

// townAreas are the act towns (S: D2 area ids 1, 40, 75, 103, 109); a shuttle is not counted through them.
var townAreas = map[int]bool{1: true, 40: true, 75: true, 103: true, 109: true}

// PovertyStallMin is the stall length that counts (project review R2: more than 10 minutes).
const PovertyStallMin = 10

// Result is one first-hour row. Every field says in its doc how it is derived: V = read straight from a log line the
// bot writes (docs/lifelog.md), S = derived by a rule stated here, U = the log may not carry it.
type Result struct {
	Class      string
	Start, End time.Time
	Versions   []string // V: `d2bot start` version values in the window, in order; more than one = a build swap
	Clvl       int      // V: `minute` line clvl, the last one at or before End
	Kills      int      // S: `minute` kills, summed across counter resets (a bot restart zeroes it)
	Gold       int      // S: `minute` gold (carried) plus stashGold when the line has it
	Chickens   int      // S: `safety decision` action=chicken; a chicken inside combat.Fighter is not logged (U how often)
	Entries    int      // V: `zone entry` lines
	Shuttles   int      // S: A,B,A runs of three consecutive non-town zone entries (the cave shuttle)
	Stalls     []int    // S: minutes of each `poverty mode: on` .. `off` span longer than PovertyStallMin
	Restarts   int      // S: larger of the `minute` restarts growth and the extra `d2bot start` lines
	Deaths     int      // S: `minute` deaths, summed across resets
}

type line struct {
	at time.Time
	kv map[string]string
}

func parse(r io.Reader) ([]line, error) {
	var out []line
	sc := bufio.NewScanner(r)
	sc.Buffer(make([]byte, 1<<20), 1<<20)
	for sc.Scan() {
		s := strings.TrimSpace(sc.Text())
		var kv map[string]string
		switch {
		case strings.HasPrefix(s, "{"):
			kv = jsonKV(s)
		case strings.Contains(s, "msg="):
			kv = textKV(s)
		}
		at, err := time.Parse(time.RFC3339Nano, kv["time"])
		if kv == nil || err != nil {
			continue
		}
		out = append(out, line{at, kv})
	}
	return out, sc.Err()
}

func jsonKV(s string) map[string]string {
	var raw map[string]any
	if json.Unmarshal([]byte(s), &raw) != nil {
		return nil
	}
	kv := make(map[string]string, len(raw))
	for k, v := range raw {
		if f, ok := v.(float64); ok && f == float64(int64(f)) {
			kv[k] = strconv.FormatInt(int64(f), 10)
			continue
		}
		kv[k] = fmt.Sprint(v)
	}
	return kv
}

func textKV(s string) map[string]string {
	kv := map[string]string{}
	for s != "" {
		s = strings.TrimLeft(s, " ")
		k, rest, ok := strings.Cut(s, "=")
		if !ok {
			break
		}
		var v string
		if strings.HasPrefix(rest, `"`) {
			end := 1
			for end < len(rest) && (rest[end] != '"' || rest[end-1] == '\\') {
				end++
			}
			if end >= len(rest) {
				break
			}
			v, _ = strconv.Unquote(rest[:end+1])
			s = rest[end+1:]
		} else {
			v, s, _ = strings.Cut(rest, " ")
		}
		kv[k] = v
	}
	return kv
}

func num(kv map[string]string, k string) (int, bool) {
	n, err := strconv.Atoi(kv[k])
	return n, err == nil
}

// counter sums a monotone counter across resets: a drop means the process restarted and counted from zero again.
type counter struct{ total, last int }

func (c *counter) add(v int) {
	if v >= c.last {
		c.total += v - c.last
	} else {
		c.total += v
	}
	c.last = v
}

// Compute derives the row. start zero means the first `d2bot start` line (else the first line) of the log. window is
// the hour; the log must reach window-slack past start or Compute errors.
func Compute(lines []line, class string, start time.Time, window, slack time.Duration) (Result, error) {
	if len(lines) == 0 {
		return Result{}, errors.New("no parseable log lines (need slog text or JSON with a time key)")
	}
	if start.IsZero() {
		start = lines[0].at
		for _, l := range lines {
			if l.kv["msg"] == "d2bot start" {
				start = l.at
				break
			}
		}
	}
	res := Result{Class: class, Start: start, End: start.Add(window)}
	if last := lines[len(lines)-1].at; last.Before(res.End.Add(-slack)) {
		return res, fmt.Errorf("log ends %s, %s after the start; the window needs %s: the run is not finished or the log was rotated",
			last.Format(time.RFC3339), last.Sub(start).Round(time.Second), window)
	}
	var kills, deaths, restarts counter
	var minutes, starts int
	povertyOn := time.Time{}
	var prev []int // last two non-town areas entered, oldest first
	lastArea := -1
	for _, l := range lines {
		if l.at.Before(start) || l.at.After(res.End) {
			continue
		}
		switch l.kv["msg"] {
		case "d2bot start":
			starts++
			if v := l.kv["version"]; v != "" && (len(res.Versions) == 0 || res.Versions[len(res.Versions)-1] != v) {
				res.Versions = append(res.Versions, v)
			}
		case "minute":
			cl, ok1 := num(l.kv, "clvl")
			k, ok2 := num(l.kv, "kills")
			g, ok3 := num(l.kv, "gold")
			if !ok1 || !ok2 || !ok3 {
				return res, fmt.Errorf("minute line at %s lacks clvl, kills or gold: %v", l.at.Format(time.RFC3339), l.kv)
			}
			if minutes == 0 && cl > 2 {
				return res, fmt.Errorf("first minute line has clvl %d: not a fresh hero (pass -start of the fresh hero's first line)", cl)
			}
			minutes++
			res.Clvl = cl
			res.Gold = g
			if sg, ok := num(l.kv, "stashGold"); ok {
				res.Gold += sg
			}
			kills.add(k)
			d, _ := num(l.kv, "deaths")
			deaths.add(d)
			rs, _ := num(l.kv, "restarts")
			restarts.add(rs)
		case "safety decision":
			if l.kv["action"] == "chicken" {
				res.Chickens++
			}
		case "zone entry":
			a, ok := num(l.kv, "area")
			if !ok {
				return res, fmt.Errorf("zone entry line at %s has no area", l.at.Format(time.RFC3339))
			}
			if a == lastArea {
				break // the first snapshot repeats the area: not an entry
			}
			lastArea = a
			res.Entries++
			if townAreas[a] {
				prev = nil
				break
			}
			if len(prev) == 2 && prev[0] == a && prev[1] != a {
				res.Shuttles++
			}
			prev = append(prev, a)
			if len(prev) > 2 {
				prev = prev[1:]
			}
		case "poverty mode: on":
			if povertyOn.IsZero() {
				povertyOn = l.at
			}
		case "poverty mode: off":
			res.stall(povertyOn, l.at)
			povertyOn = time.Time{}
		}
	}
	res.stall(povertyOn, res.End) // still poor at the end of the hour
	if minutes == 0 {
		return res, errors.New("no `minute` lines in the window: the bot did not run or logs the wrong level")
	}
	if len(res.Versions) == 0 {
		return res, errors.New("no `d2bot start` line with a version in the window: pass -deploy to name the build")
	}
	res.Kills, res.Deaths = kills.total, deaths.total
	res.Restarts = max(restarts.total, starts-1) // the two count the same restarts
	return res, nil
}

func (r *Result) stall(from, to time.Time) {
	if from.IsZero() {
		return
	}
	if m := int(to.Sub(from) / time.Minute); m > PovertyStallMin {
		r.Stalls = append(r.Stalls, m)
	}
}

// StallText is "0" or the list of stall lengths.
func (r Result) StallText() string {
	if len(r.Stalls) == 0 {
		return "0"
	}
	s := append([]int(nil), r.Stalls...)
	sort.Ints(s)
	p := make([]string, len(s))
	for i, m := range s {
		p[i] = strconv.Itoa(m)
	}
	return strings.Join(p, "+")
}

// Header is written once when the table file does not exist.
const Header = `# First hour: fresh hero, 60 realtime minutes at 1x

Headline metric of the project review (docs/research/project-review-2026-11-10.md, R2; TASK-245). One row per run,
appended by ` + "`tools/firsthour.sh`" + ` (cmd/d2firsthour). Tags: V read from a bot log line (docs/lifelog.md), S derived
by the rule below, U the log may not carry it.

| column | tag | derivation |
|---|---|---|
| date, start | V | log time of the fresh hero's first line (` + "`d2bot start`" + `) or the -start given |
| build | V | ` + "`d2bot start` version value(s) in the window; ` > ` joins a swap" + ` |
| clvl@60, gold | V | last ` + "`minute`" + ` line in the window; gold = carried + stash when logged |
| kills, deaths | S | ` + "`minute`" + ` counters summed across process restarts (a drop = a reset) |
| chickens | S | ` + "`safety decision action=chicken`" + `; a chicken inside the fight loop is not logged (U how often) |
| entries, shuttles | V, S | ` + "`zone entry`" + ` lines; shuttle = A,B,A over three consecutive non-town entries (U the cap) |
| poverty stalls | S | minutes of each ` + "`poverty mode: on`" + ` to ` + "`off`" + ` span over 10 min (open at the hour's end counts) |
| restarts | S | the larger of the ` + "`minute`" + ` restarts counter growth and the extra ` + "`d2bot start`" + ` lines |

A row is only written when the log covers the whole hour of a fresh hero (clvl at most 2 on the first minute line).

| date | build | class | start | clvl@60 | kills | deaths | gold | chickens | entries | shuttles | poverty stalls (min) | restarts |
|---|---|---|---|---|---|---|---|---|---|---|---|---|
`

// Row is the table line for the result.
func (r Result) Row() string {
	return fmt.Sprintf("| %s | %s | %s | %s | %d | %d | %d | %d | %d | %d | %d | %s | %d |\n",
		r.Start.UTC().Format("2006-01-02"), strings.Join(r.Versions, " > "), r.Class, r.Start.UTC().Format("15:04Z"),
		r.Clvl, r.Kills, r.Deaths, r.Gold, r.Chickens, r.Entries, r.Shuttles, r.StallText(), r.Restarts)
}
