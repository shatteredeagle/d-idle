// Package shadowlog reads the shadow lines of a recorded bot log (JSON lines) and digests them per family, so a
// dark-launch row whose enable criteria name "a night of shadow lines" has something to cite (docs/shadowlog.md).
// It reads only; it never talks to a game or a bot.
package shadowlog

import (
	"bufio"
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"sort"
	"strings"
)

// Schema is the version of the JSON report.
const Schema = "shadowlog/1"

// Families, in report order.
const (
	Admission = "admission"
	Planner   = "planner"
	Upkeep    = "upkeep"
	Loot      = "loot"
)

// Verdicts. NoLines is not agreement: three of the four shadows log only when they disagree, so an empty family
// proves agreement only when the shadow ran (the knob named in the digest).
const (
	NoLines = "no-lines"
	Clean   = "clean"
	Review  = "review"
	Worse   = "worse"
)

// Row is one group of lines.
type Row struct {
	Key     string `json:"key"`
	Count   int    `json:"count"`
	Example string `json:"example,omitempty"`
}

// Digest is one family.
type Digest struct {
	Family   string `json:"family"`
	Knob     string `json:"knob"`     // the knob whose shadow writes these lines
	Lines    int    `json:"lines"`    // shadow lines, folded repeats counted
	Differ   int    `json:"differ"`   // lines that report a disagreement
	Unparsed int    `json:"unparsed"` // lines of the family this reader could not take apart
	Verdict  string `json:"verdict"`
	Why      string `json:"why"`
	Totals   []Row  `json:"totals,omitempty"` // counts over every line of the family
	Rows     []Row  `json:"rows,omitempty"`   // the disagreements, most frequent first
}

// Report is the digest of one or more logs.
type Report struct {
	Schema   string   `json:"schema"`
	Lines    int      `json:"lines"`
	NotJSON  int      `json:"notJson"`
	First    string   `json:"first,omitempty"` // time of the first and last record that carries one
	Last     string   `json:"last,omitempty"`
	Families []Digest `json:"families"`
}

// Failing reports whether any family needs a look.
func (r Report) Failing() bool {
	for _, d := range r.Families {
		if d.Verdict == Review || d.Verdict == Worse {
			return true
		}
	}
	return false
}

type family struct {
	name, knob string
	match      func(msg string) bool
	// add notes one line n times; false means the line did not parse.
	add     func(a *acc, msg string, rec map[string]any, n int) bool
	verdict func(a *acc) (string, string)
}

type acc struct {
	lines, differ, unparsed int
	totals, rows            map[string]*Row
	lastMsg                 string
	lastRec                 map[string]any
}

func (a *acc) bump(m *map[string]*Row, key, example string, n int) {
	if *m == nil {
		*m = map[string]*Row{}
	}
	r := (*m)[key]
	if r == nil {
		r = &Row{Key: key, Example: example}
		(*m)[key] = r
	}
	r.Count += n
}

func (a *acc) total(key string, n int)        { a.bump(&a.totals, key, "", n) }
func (a *acc) row(key, example string, n int) { a.bump(&a.rows, key, example, n); a.differ += n }

var families = []family{
	{Admission, "bot.admission", func(m string) bool { return m == "admission shadow" }, addAdmission, admissionVerdict},
	{Planner, "agency.planner", func(m string) bool { return strings.HasPrefix(m, "planner shadow: clvl=") }, addPlanner, plannerVerdict},
	{Upkeep, "combat.deliveryUpkeep", func(m string) bool { return m == "combat upkeep shadow" }, addUpkeep, upkeepVerdict},
	{Loot, "loot.rulesEngine", func(m string) bool { return strings.HasPrefix(m, "loot shadow: ") }, addLoot, lootVerdict},
}

const repeated = "last line repeated" // the counter line of internal/logx.Dedup

// Read digests the logs in order. A line that is not JSON is counted and skipped.
func Read(logs ...io.Reader) (Report, error) {
	rep := Report{Schema: Schema}
	accs := make([]acc, len(families))
	var first, last []byte
	for _, in := range logs {
		br := bufio.NewReaderSize(in, 1<<16)
		for {
			line, err := br.ReadBytes('\n')
			if len(bytes.TrimSpace(line)) > 0 {
				rep.Lines++
				if first == nil {
					first = bytes.Clone(line)
				}
				last = append(last[:0], line...)
				if bytes.Contains(line, []byte("shadow")) {
					if !take(accs, line) {
						rep.NotJSON++
					}
				} else if line[0] != '{' {
					rep.NotJSON++
				}
			}
			if err == io.EOF {
				break
			}
			if err != nil {
				return rep, err
			}
		}
	}
	rep.First, rep.Last = timeOf(first), timeOf(last)
	for i, f := range families {
		a := &accs[i]
		d := Digest{Family: f.name, Knob: f.knob, Lines: a.lines, Differ: a.differ, Unparsed: a.unparsed, Totals: sorted(a.totals), Rows: sorted(a.rows)}
		d.Verdict, d.Why = f.verdict(a)
		if a.unparsed > 0 {
			d.Verdict = Review
			d.Why = fmt.Sprintf("%d line(s) of this family did not parse: the reader is behind the bot's line format; %s", a.unparsed, d.Why)
		}
		rep.Families = append(rep.Families, d)
	}
	return rep, nil
}

// take feeds one candidate line to its family; false means the line is not a JSON record.
func take(accs []acc, line []byte) bool {
	var rec map[string]any
	if json.Unmarshal(line, &rec) != nil {
		return false
	}
	msg, _ := rec["msg"].(string)
	n := 1
	if msg == repeated {
		msg, _ = rec["line"].(string)
		t, _ := rec["times"].(float64)
		if n = int(t); n < 1 {
			return true
		}
		rec = nil
	}
	for i, f := range families {
		if !f.match(msg) {
			continue
		}
		a := &accs[i]
		if rec == nil && a.lastMsg == msg { // a repeat carries the message only: the attributes are those of the line it repeats
			rec = a.lastRec
		}
		a.lines += n
		if f.add(a, msg, rec, n) {
			a.lastMsg, a.lastRec = msg, rec
		} else {
			a.unparsed += n
		}
		return true
	}
	return true
}

func timeOf(line []byte) string {
	var rec struct {
		Time string `json:"time"`
	}
	_ = json.Unmarshal(line, &rec)
	return rec.Time
}

func sorted(m map[string]*Row) []Row {
	out := make([]Row, 0, len(m))
	for _, r := range m {
		out = append(out, *r)
	}
	sort.Slice(out, func(i, j int) bool {
		if out[i].Count != out[j].Count {
			return out[i].Count > out[j].Count
		}
		return out[i].Key < out[j].Key
	})
	return out
}

// fields takes "k1=v1 k2=v2 ..." apart by the known keys in order; a value may hold spaces (skill and item names do).
// ok is false when a key is missing or out of order.
func fields(s string, keys ...string) (map[string]string, bool) {
	out := make(map[string]string, len(keys))
	s = " " + s
	at := make([]int, len(keys)+1)
	from := 0
	for i, k := range keys {
		j := strings.Index(s[from:], " "+k+"=")
		if j < 0 {
			return nil, false
		}
		at[i] = from + j
		from = at[i] + len(k) + 2
	}
	at[len(keys)] = len(s)
	for i, k := range keys {
		out[k] = s[at[i]+len(k)+2 : at[i+1]]
	}
	return out, true
}

func str(rec map[string]any, k string) (string, bool) {
	switch v := rec[k].(type) {
	case string:
		return v, true
	case float64:
		return fmt.Sprintf("%g", v), true
	case bool:
		return fmt.Sprintf("%t", v), true
	}
	return "", false
}

// plural-free count list "a 2, b 1" of the totals whose key starts with prefix, in the order given.
func tally(a *acc, prefix string, keys ...string) string {
	var parts []string
	for _, k := range keys {
		n := 0
		if r := a.totals[prefix+k]; r != nil {
			n = r.Count
		}
		parts = append(parts, fmt.Sprintf("%s %d", k, n))
	}
	return strings.Join(parts, ", ")
}
