package main

import (
	"bufio"
	"encoding/json"
	"fmt"
	"io"
	"strconv"
	"strings"
	"time"
)

// Minute is one "minute" summary line of the bot log (internal/bot minuteLine). Kills (per-character, survives bot
// restarts), Deaths, PotionsUsed and Restarts are running totals; Potions and Mana are the belt counts at that minute
// (docs/soak-analysis.md section 11). Old logs have neither the new fields nor a belt count: there `potions` was the
// drunk counter, which is what PotionsUsed then holds and Potions is 0.
type Minute struct {
	At                                 time.Time
	Area, Clvl, Kills, Deaths, Potions int
	Gold, Restarts, Rung               int
	XP                                 uint64
	Goal, Step                         string
	Mana, PotionsUsed, KillsGame       int
	State, RState                      string // activity group (internal/stepgroup) and runner state; "" in old logs
	HasBelt                            bool   // the line carried potionsUsed, so potions is a belt count
}

// ParseMinuteLines reads slog text or JSON lines and returns the minute summaries in file order; other and
// malformed lines are skipped.
func ParseMinuteLines(r io.Reader) ([]Minute, error) {
	var out []Minute
	sc := bufio.NewScanner(r)
	sc.Buffer(make([]byte, 1<<20), 1<<20)
	for sc.Scan() {
		line := strings.TrimSpace(sc.Text())
		var kv map[string]string
		switch {
		case strings.HasPrefix(line, "{"):
			kv = jsonKV(line)
		case strings.Contains(line, "msg=minute "):
			kv = textKV(line)
		}
		if kv == nil || kv["msg"] != "minute" {
			continue
		}
		m := Minute{Goal: kv["goal"], Step: kv["step"]}
		m.At, _ = time.Parse(time.RFC3339Nano, kv["time"])
		ints := map[string]*int{"area": &m.Area, "clvl": &m.Clvl, "kills": &m.Kills, "deaths": &m.Deaths,
			"potions": &m.Potions, "gold": &m.Gold, "restarts": &m.Restarts, "rung": &m.Rung}
		ok := true
		for k, p := range ints {
			n, err := strconv.Atoi(kv[k])
			if err != nil {
				ok = false
			}
			*p = n
		}
		m.State, m.RState = kv["state"], kv["rstate"]
		m.Mana, m.KillsGame = atoiOr(kv["mana"], 0), atoiOr(kv["killsGame"], m.Kills)
		if v, has := kv["potionsUsed"]; has {
			m.HasBelt, m.PotionsUsed = true, atoiOr(v, 0)
		} else {
			m.PotionsUsed, m.Potions = m.Potions, 0
		}
		m.XP, _ = strconv.ParseUint(kv["xp"], 10, 64)
		if ok {
			out = append(out, m)
		}
	}
	return out, sc.Err()
}

func atoiOr(s string, def int) int {
	if n, err := strconv.Atoi(s); err == nil {
		return n
	}
	return def
}

func jsonKV(line string) map[string]string {
	var raw map[string]any
	if json.Unmarshal([]byte(line), &raw) != nil {
		return nil
	}
	kv := make(map[string]string, len(raw))
	for k, v := range raw {
		kv[k] = fmt.Sprint(v)
	}
	return kv
}

// textKV splits slog text output (key=value, values optionally quoted).
func textKV(line string) map[string]string {
	kv := map[string]string{}
	for len(line) > 0 {
		line = strings.TrimLeft(line, " ")
		k, rest, ok := strings.Cut(line, "=")
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
			line = rest[end+1:]
		} else {
			v, line, _ = strings.Cut(rest, " ")
		}
		kv[k] = v
	}
	return kv
}

// MinuteText renders the run summary from the minute lines: span, progress deltas and the busiest goal/step.
func MinuteText(ms []Minute) string {
	if len(ms) == 0 {
		return "minutes: no summary lines\n"
	}
	a, z := ms[0], ms[len(ms)-1]
	var b strings.Builder
	fmt.Fprintf(&b, "minutes: %d lines, clvl %d -> %d, xp %d -> %d, kills %d, deaths %d, potions used %d, gold %d -> %d, restarts %d\n",
		len(ms), a.Clvl, z.Clvl, a.XP, z.XP, z.Kills-a.Kills, z.Deaths-a.Deaths, z.PotionsUsed-a.PotionsUsed, a.Gold, z.Gold, z.Restarts-a.Restarts)
	rung, goals := 0, map[string]int{}
	for _, m := range ms {
		rung = max(rung, m.Rung)
		goals[m.Goal+"/"+m.Step]++
	}
	best, n := "", 0
	for g, c := range goals {
		if c > n || (c == n && g < best) {
			best, n = g, c
		}
	}
	fmt.Fprintf(&b, "  highest failsafe rung %d, most common goal/step %s (%d min)\n", rung, best, n)
	return b.String()
}

// telemetryBody finds a telemetry record in one bot log line, in either format the bot writes: a JSON line (the
// record is inside the "msg" field) or a slog text line (inside msg="..."). marker is the record's first key
// ("fight area="); the returned body starts there and holds only the key=value fields. ts is the line's time.
func telemetryBody(line, marker string) (body, ts string, ok bool) {
	line = strings.TrimSpace(line)
	if strings.HasPrefix(line, "{") {
		kv := jsonKV(line)
		msg := kv["msg"]
		i := strings.Index(msg, marker)
		if i < 0 {
			return "", "", false
		}
		return msg[i:], kv["time"], true
	}
	i := strings.Index(line, marker)
	if i < 0 {
		return "", "", false
	}
	if j := strings.Index(line, "time="); j >= 0 {
		ts, _, _ = strings.Cut(line[j+5:], " ")
	}
	return strings.Trim(line[i:], `"`), ts, true
}
