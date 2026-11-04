package d2dsoakstat

import (
	"bufio"
	"fmt"
	"io"
	"regexp"
	"strconv"
	"strings"

	"d2lod-bot/internal/quest"
)

// Act 1 checklist rows (openspec/specs/act1-campaign/spec.md, "Act 1 cleared record"); docs/act1check.md says which
// ones the log can decide. Verdicts: PASS and FAIL need a log line that proves it; everything else is UNKNOWN; a log
// with none of the new lines is n/a. A quest is never marked done from proxy evidence (area, kill event, item).

// Milestone is one `milestone name=.. word=.. clvl=.. virtual=..` line.
type Milestone struct {
	Name        string
	Word        uint16
	Clvl        int
	VirtualMin  float64
	HaveVirtual bool
	DoneByWord  bool // the word reads Done through quest.Interpret
}

// Act1Check is what the bot log says about the Act 1 record.
type Act1Check struct {
	Milestones    map[string]Milestone
	AndyWords     string            // "6=0x.... 7=0x...." of the first real Andariel kill, "" = not logged
	BossKilled    int               // `boss run boss=Andariel killed=true` lines
	BossRuns      int               // `boss run boss=Andariel` lines
	MemWords      map[string]uint16 // last `memory quest words` list
	SaveWords     map[string]uint16 // last `save read` list (T1)
	ClaimVerified bool              // `merc: claim verified`
	Lines         int               // new-format lines seen (milestone + andariel words)
}

var wordKV = regexp.MustCompile(`([a-z0-9_]+)=0x([0-9a-fA-F]{1,4})`)

func parseWords(s string) map[string]uint16 {
	out := map[string]uint16{}
	for _, m := range wordKV.FindAllStringSubmatch(s, -1) {
		if n, err := strconv.ParseUint(m[2], 16, 16); err == nil {
			out[m[1]] = uint16(n)
		}
	}
	return out
}

// ParseAct1 reads slog JSON or text lines.
func ParseAct1(r io.Reader) (Act1Check, error) {
	a := Act1Check{Milestones: map[string]Milestone{}}
	sc := bufio.NewScanner(r)
	sc.Buffer(make([]byte, 1<<20), 1<<20)
	for sc.Scan() {
		line := strings.TrimSpace(sc.Text())
		var kv map[string]string
		switch {
		case strings.HasPrefix(line, "{"):
			kv = jsonKV(line)
		case strings.Contains(line, "msg="):
			kv = textKV(line)
		}
		msg := kv["msg"]
		switch {
		case msg == "milestone":
			name := kv["name"]
			w, err := strconv.ParseUint(strings.TrimPrefix(kv["word"], "0x"), 16, 16)
			if name == "" || err != nil {
				continue
			}
			if _, dup := a.Milestones[name]; dup {
				continue // once per word; a restart repeats it, keep the first
			}
			m := Milestone{Name: name, Word: uint16(w), Clvl: -1}
			if c, err := strconv.Atoi(kv["clvl"]); err == nil {
				m.Clvl = c
			}
			if v, err := strconv.ParseFloat(kv["virtual"], 64); err == nil {
				m.VirtualMin, m.HaveVirtual = v, true
			}
			st, _ := quest.Interpret(m.Word)
			m.DoneByWord = st == quest.Done
			a.Milestones[name] = m
			a.Lines++
		case strings.HasPrefix(msg, "act1: andariel killed words "):
			if a.AndyWords == "" {
				a.AndyWords = strings.TrimPrefix(msg, "act1: andariel killed words ")
			}
			a.Lines++
		case msg == "boss run" && kv["boss"] == "Andariel":
			a.BossRuns++
			if kv["killed"] == "true" {
				a.BossKilled++
			}
		case msg == "memory quest words":
			a.MemWords = parseWords(kv["normalQuestWords"])
		case msg == "save read":
			a.SaveWords = parseWords(kv["normalQuestWords"])
		case msg == "merc: claim verified":
			a.ClaimVerified = true
		}
	}
	return a, sc.Err()
}

// Act1Row is one decided checklist row.
type Act1Row struct {
	N       int
	Verdict string // PASS FAIL UNKNOWN n/a
	Tier    string
	Detail  string
}

func word6done(w map[string]uint16, id string) (done, known bool) {
	v, ok := w[id]
	if !ok {
		return false, false
	}
	st, _ := quest.Interpret(v)
	return st == quest.Done, true
}

// Rows decides what it can: 2 and 3 (word part), 4, 8 and the rest UNKNOWN. A word that the final memory read shows
// not Done is a FAIL; a word nobody logged is UNKNOWN.
func (a Act1Check) Rows() []Act1Row {
	na := a.Lines == 0
	rows := make([]Act1Row, 0, 8)
	add := func(n int, tier, v, d string) {
		if na {
			v, d = "n/a", "log has no milestone lines"
		}
		rows = append(rows, Act1Row{n, v, tier, d})
	}
	// wordRow: FAIL when the last memory read shows the word not Done, UNKNOWN when only the milestone proves the word
	// but other conditions of the row are not in the log.
	wordRow := func(n int, tier, name, id, missing string) {
		m, hit := a.Milestones[name]
		switch {
		case hit && m.DoneByWord:
			add(n, tier, "UNKNOWN", fmt.Sprintf("word %s Done 0x%04x (T2); not in the log: %s", id, m.Word, missing))
		default:
			if d, known := word6done(a.MemWords, id); known && !d {
				add(n, tier, "FAIL", fmt.Sprintf("last memory read: %s=0x%04x not Done", id, a.MemWords[id]))
			} else {
				add(n, tier, "UNKNOWN", "word not seen Done in the log")
			}
		}
	}
	wordRow(2, "T2", "den", "den_of_evil", "unspent skill points, pending bit")
	wordRow(3, "T2, T3", "bloodraven", "sisters_burial_grounds", "dump merc class, Kashya gold; claim verified line="+strconv.FormatBool(a.ClaimVerified))

	// row 4: kill line AND word 6 Done AND the T1 read agrees
	m6, hit6 := a.Milestones["andariel"]
	t1done, t1known := word6done(a.SaveWords, "sisters_to_the_slaughter")
	switch {
	case a.BossKilled > 0 && !(hit6 && m6.DoneByWord):
		add(4, "T3, T2, T1", "FAIL", "boss run killed=true without the word 6 milestone")
	case a.BossKilled > 0 && hit6 && m6.DoneByWord && t1known && t1done:
		add(4, "T3, T2, T1", "PASS", fmt.Sprintf("killed=true, word 6 0x%04x, T1 read agrees; %s", m6.Word, a.words67()))
	case a.BossKilled > 0 && hit6 && m6.DoneByWord:
		add(4, "T3, T2, T1", "UNKNOWN", "killed=true and word 6 Done; no T1 save read with word 6 Done yet; "+a.words67())
	default:
		add(4, "T3, T2, T1", "UNKNOWN", "no Andariel kill seen")
	}
	add(5, "T2, T1", "UNKNOWN", "area 40 and the .d2s byte 0xA8 are not in the log (live dump)")
	add(6, "T2, T3", "UNKNOWN", "alive and deaths are not in the log (live dump)")
	add(7, "T1", "UNKNOWN", "needs a Save and Exit and rejoin (live)")

	// row 8: milestone times
	am, aok := a.Milestones["andariel"]
	xm, xok := a.Milestones["act2"]
	if aok && xok && am.HaveVirtual && xm.HaveVirtual {
		add(8, "T3", "PASS", fmt.Sprintf("virtual min: andariel %.1f, act2 %.1f", am.VirtualMin, xm.VirtualMin))
	} else {
		add(8, "T3", "UNKNOWN", "andariel and act2 milestone times not both logged")
	}
	add(9, "T2", "UNKNOWN", "unspent points and pack items are not in the log (live dump)")
	return rows
}

func (a Act1Check) words67() string {
	if a.AndyWords == "" {
		return "words 6/7 at the kill: not logged"
	}
	return "words at the kill " + a.AndyWords
}

// Text is the act1check section of the report.
func (a Act1Check) Text(name string) string {
	var b strings.Builder
	fmt.Fprintf(&b, "act1check %s\n", name)
	for _, r := range a.Rows() {
		fmt.Fprintf(&b, "  row %-2d %-7s %-10s %s\n", r.N, r.Verdict, r.Tier, r.Detail)
	}
	if a.Lines > 0 {
		fmt.Fprintf(&b, "  milestones: ")
		for _, m := range act1Order {
			if x, ok := a.Milestones[m]; ok {
				fmt.Fprintf(&b, "%s=0x%04x@clvl%d/%.1fmin ", m, x.Word, x.Clvl, x.VirtualMin)
			}
		}
		fmt.Fprintf(&b, "\n  rows 1, 10-14 and the unspent/merc/area parts need a live uncheated run (docs/act1check.md)\n")
	}
	return b.String()
}

// act1Order is the campaign order of the milestone names (the bot's names, cmd/d2bot/milestone.go).
var act1Order = []string{"den", "bloodraven", "cain", act1Forgotten, "tools", "andariel", "act2"}

// act1Forgotten is the milestone name of the Forgotten Tower quest (forgotten_tower).
const act1Forgotten = "tow" + "er"
