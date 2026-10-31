// Package d2dsoakstat turns the bot's per-minute summary lines into the soak metrics of docs/soak-analysis.md,
// aggregates them over seeds and compares them with stored original-game baselines (docs/d2d-soak.md).
// Pure functions, no I/O beyond the reader passed in.
package d2dsoakstat

import (
	"bufio"
	"encoding/json"
	"fmt"
	"io"
	"strconv"
	"strings"
	"time"
)

// Minute is one `msg=minute` line of the bot log. Field names follow internal/bot minuteLine (read from
// cmd/d2soak/minutelines.go; V: parses the real a14/a16/a18 logs in testdata).
type Minute struct {
	At                        time.Time
	Area, Clvl, Kills, Deaths int
	Gold, Restarts            int
	PotionsUsed               int // drunk counter; in old logs (no potionsUsed key) the `potions` field
	XP                        uint64
	State, RState             string
	Goal, Step                string
	// DmgTaken (running counter, life points lost) and HPMinPct (lowest life percent seen in the minute) are
	// V (by unit test, not live): the bot minute line emits both since change minutefields; HasHP is false for older logs that lack both keys.
	DmgTaken, HPMinPct int
	HasHP              bool
}

// ParseMinutes reads slog JSON or text lines and returns the minute summaries in file order; other and
// malformed lines are skipped.
func ParseMinutes(r io.Reader) ([]Minute, error) {
	out, _, err := ParseMinutesCount(r)
	return out, err
}

// ParseMinutesCount is ParseMinutes plus the number of minute lines that were dropped because a field (time, xp or
// a counter) did not parse; Run.NoteDropped turns that into an invalid run.
func ParseMinutesCount(r io.Reader) ([]Minute, int, error) {
	var out []Minute
	dropped := 0
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
		m := Minute{Goal: kv["goal"], Step: kv["step"], State: kv["state"], RState: kv["rstate"]}
		var terr error
		m.At, terr = time.Parse(time.RFC3339Nano, kv["time"])
		ok := terr == nil
		for k, p := range map[string]*int{"area": &m.Area, "clvl": &m.Clvl, "kills": &m.Kills, "deaths": &m.Deaths,
			"gold": &m.Gold, "restarts": &m.Restarts} {
			n, err := strconv.Atoi(kv[k])
			if err != nil {
				ok = false
			}
			*p = n
		}
		if v, has := kv["potionsUsed"]; has {
			m.PotionsUsed, _ = strconv.Atoi(v)
		} else {
			m.PotionsUsed, _ = strconv.Atoi(kv["potions"])
		}
		if a, ok1 := kv["dmgTaken"]; ok1 {
			if b, ok2 := kv["hpMinPct"]; ok2 {
				m.DmgTaken, _ = strconv.Atoi(a)
				m.HPMinPct, _ = strconv.Atoi(b)
				m.HasHP = true
			}
		}
		if x, has := kv["xp"]; has {
			var xerr error
			if m.XP, xerr = strconv.ParseUint(x, 10, 64); xerr != nil {
				ok = false
			}
		}
		if ok {
			out = append(out, m)
		} else {
			dropped++
		}
	}
	return out, dropped, sc.Err()
}

func jsonKV(line string) map[string]string {
	var raw map[string]any
	if json.Unmarshal([]byte(line), &raw) != nil {
		return nil
	}
	kv := make(map[string]string, len(raw))
	for k, v := range raw {
		if f, isNum := v.(float64); isNum && f == float64(int64(f)) {
			kv[k] = strconv.FormatInt(int64(f), 10) // fmt.Sprint would print 1.19175e+05
			continue
		}
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
