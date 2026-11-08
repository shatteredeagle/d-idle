// Package smoke judges the log of a short lockstep run of the bot (TASK-111, docs/smoke.md): one verdict line per bot
// change. It only reads text; running the game is tools/smoke.sh. The rules are data (data/smoke/rules.json).
package smoke

import (
	"bufio"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"sort"
	"strings"
)

// ClvlPoint says: at the given virtual minute the character must be at least level Min. The values are the
// docs/soaks/classpacing.md baseline minus a tolerance of one level.
type ClvlPoint struct {
	Minute int `json:"minute"`
	Min    int `json:"min"`
}

// ClassRule is one class: its d2d class index (cmd/d2dsetup/tables.json), its leveler build and its pace floor.
type ClassRule struct {
	D2DIndex int         `json:"d2d_index"`
	Build    string      `json:"build"`
	Clvl     []ClvlPoint `json:"clvl"`
}

// Rules is data/smoke/rules.json.
type Rules struct {
	Version           int                  `json:"version"`
	Minutes           int                  `json:"minutes"`
	CrashPatterns     []string             `json:"crash_patterns"`
	ErrorLevels       []string             `json:"error_levels"`
	ErrorAllow        []string             `json:"error_allow"`
	MinKills          int                  `json:"min_kills"`
	EquipPatterns     []string             `json:"equip_patterns"`
	EquipMax          int                  `json:"equip_max"`
	AuraPattern       string               `json:"aura_pattern"`
	AuraMax           int                  `json:"aura_max"`
	QuestionMsg       string               `json:"question_msg"`
	QuestionKeyAttrs  []string             `json:"question_key_attrs"`
	QuestionsMax      int                  `json:"questions_max"`
	QuestionRepeatMax int                  `json:"question_repeat_max"`
	SessionFiles      []string             `json:"session_files"`
	WritesRequired    bool                 `json:"writes_required"`
	Classes           map[string]ClassRule `json:"classes"`
}

// LoadRules reads and checks a rules file.
func LoadRules(path string) (*Rules, error) {
	b, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}
	var r Rules
	dec := json.NewDecoder(strings.NewReader(string(b)))
	dec.DisallowUnknownFields()
	if err := dec.Decode(&r); err != nil {
		return nil, fmt.Errorf("smoke: %s: %w", path, err)
	}
	if r.Minutes <= 0 || len(r.Classes) == 0 {
		return nil, fmt.Errorf("smoke: %s: minutes and classes are required", path)
	}
	return &r, nil
}

// Class returns the rule of a class; an unknown class is an error that names it and the known ones.
func (r *Rules) Class(name string) (ClassRule, error) {
	c, ok := r.Classes[strings.ToLower(name)]
	if !ok {
		known := make([]string, 0, len(r.Classes))
		for k := range r.Classes {
			known = append(known, k)
		}
		sort.Strings(known)
		return ClassRule{}, fmt.Errorf("smoke: unknown class %q (known: %s)", name, strings.Join(known, ", "))
	}
	return c, nil
}

// Input is what one run left behind.
type Input struct {
	Class      string
	Commit     string
	Log        io.Reader // the bot log, JSON lines (slog); other lines are matched as text
	SessionDir string    // the session record directory; "" when the run had no ladder
	Secrets    []string  // strings that must not appear in the session files
	Writes     int       // `d2cheat ledger --since-run` count; -1: unknown
	Minutes    int       // 0: the rules' default
}

// Result is the verdict.
type Result struct {
	Pass    bool
	Class   string
	Commit  string
	Clvl    int // at the judged minute
	Minute  int
	Kills   int
	Deaths  int
	Fails   []string
	Notes   []string
	Samples []string // a few offending log lines for the record
}

type logLine struct {
	Msg   string
	Level string
	Raw   string
	Attrs map[string]any
}

func parse(line string) logLine {
	l := logLine{Raw: line, Msg: line}
	if strings.HasPrefix(line, "{") {
		var m map[string]any
		if json.Unmarshal([]byte(line), &m) == nil {
			l.Attrs = m
			l.Msg, _ = m["msg"].(string)
			l.Level, _ = m["level"].(string)
		}
	}
	return l
}

func num(v any) int {
	f, _ := v.(float64)
	return int(f)
}

// Judge applies the rules. Every failed rule adds one line to Fails; a run passes with none.
func Judge(r *Rules, in Input) (*Result, error) {
	cr, err := r.Class(in.Class)
	if err != nil {
		return nil, err
	}
	mins := in.Minutes
	if mins <= 0 {
		mins = r.Minutes
	}
	res := &Result{Class: strings.ToLower(in.Class), Commit: in.Commit, Minute: mins}
	fail := func(f string, a ...any) { res.Fails = append(res.Fails, fmt.Sprintf(f, a...)) }
	sample := func(s string) {
		if len(res.Samples) < 12 {
			if len(s) > 240 {
				s = s[:240]
			}
			res.Samples = append(res.Samples, s)
		}
	}

	type minute struct{ clvl, kills, deaths int }
	var minutes []minute
	var equip, aura, questions int
	keyCount := map[string]int{}
	allowed := func(s string) bool {
		for _, a := range r.ErrorAllow {
			if strings.Contains(s, a) {
				return true
			}
		}
		return false
	}
	var crashes, errs int
	sc := bufio.NewScanner(in.Log)
	sc.Buffer(make([]byte, 1<<20), 16<<20)
	for sc.Scan() {
		l := parse(sc.Text())
		if l.Msg == "minute" && l.Attrs != nil {
			minutes = append(minutes, minute{num(l.Attrs["clvl"]), num(l.Attrs["kills"]), num(l.Attrs["deaths"])})
			continue
		}
		for _, p := range r.CrashPatterns {
			if strings.Contains(l.Raw, p) {
				crashes++
				sample(l.Raw)
				break
			}
		}
		if hasLevel(r.ErrorLevels, l.Level) && !allowed(l.Raw) {
			errs++
			sample(l.Raw)
		}
		for _, p := range r.EquipPatterns {
			if strings.Contains(l.Raw, p) {
				equip++
				break
			}
		}
		if r.AuraPattern != "" && strings.Contains(l.Raw, r.AuraPattern) {
			aura++
		}
		if r.QuestionMsg != "" && l.Msg == r.QuestionMsg {
			questions++
			for _, a := range r.QuestionKeyAttrs {
				if k, _ := l.Attrs[a].(string); k != "" {
					keyCount[k]++
					break
				}
			}
		}
	}
	if err := sc.Err(); err != nil {
		return nil, fmt.Errorf("smoke: reading the log: %w", err)
	}

	if crashes > 0 {
		fail("%d crash or panic line(s)", crashes)
	}
	if errs > 0 {
		fail("%d error event(s) not on the allow list", errs)
	}
	if len(minutes) < mins {
		fail("short run: %d minute lines, need %d", len(minutes), mins)
	}
	if n := min(len(minutes), mins); n > 0 {
		last := minutes[n-1]
		res.Clvl, res.Kills, res.Deaths = last.clvl, last.kills, last.deaths
		for _, p := range cr.Clvl {
			if p.Minute > len(minutes) {
				continue // already reported as a short run
			}
			if got := minutes[p.Minute-1].clvl; got < p.Min {
				fail("clvl %d at minute %d, need %d", got, p.Minute, p.Min)
			}
		}
	}
	if res.Kills < r.MinKills {
		fail("kills %d, need %d", res.Kills, r.MinKills)
	}
	if equip > r.EquipMax {
		fail("%d equip refusals, max %d", equip, r.EquipMax)
	}
	if aura > r.AuraMax {
		fail("%d aura \"%s\" lines, max %d", aura, r.AuraPattern, r.AuraMax)
	}
	if questions > r.QuestionsMax {
		fail("%d questions, max %d", questions, r.QuestionsMax)
	}
	for _, k := range sortedKeys(keyCount) {
		if keyCount[k] > r.QuestionRepeatMax {
			fail("question key %q asked %d times, max %d", k, keyCount[k], r.QuestionRepeatMax)
		}
	}
	if in.SessionDir != "" {
		checkSession(r, in, fail)
	} else {
		res.Notes = append(res.Notes, "no-session-record")
	}
	switch {
	case in.Writes > 0:
		fail("writes after start: %d", in.Writes)
	case in.Writes < 0 && r.WritesRequired:
		fail("writes after start unknown (no ledger check)")
	case in.Writes < 0:
		res.Notes = append(res.Notes, "writes-unchecked")
	}
	if equip > 0 {
		res.Notes = append(res.Notes, fmt.Sprintf("equip=%d", equip))
	}
	if questions > 0 {
		res.Notes = append(res.Notes, fmt.Sprintf("questions=%d", questions))
	}
	res.Pass = len(res.Fails) == 0
	return res, nil
}

func hasLevel(levels []string, l string) bool {
	for _, x := range levels {
		if strings.EqualFold(x, l) {
			return true
		}
	}
	return false
}

func sortedKeys(m map[string]int) []string {
	k := make([]string, 0, len(m))
	for s := range m {
		k = append(k, s)
	}
	sort.Strings(k)
	return k
}

// checkSession wants every session file and no secret in any of them. Files are read whole (the record is small).
func checkSession(r *Rules, in Input, fail func(string, ...any)) {
	dir := in.SessionDir
	// The record sits in <dir>/<id>/; accept <dir> itself or its single newest subdirectory.
	if ents, err := os.ReadDir(dir); err == nil {
		for i := len(ents) - 1; i >= 0; i-- {
			if ents[i].IsDir() {
				if _, err := os.Stat(filepath.Join(dir, r.SessionFiles[0])); err != nil {
					dir = filepath.Join(dir, ents[i].Name())
				}
				break
			}
		}
	}
	for _, f := range r.SessionFiles {
		b, err := os.ReadFile(filepath.Join(dir, f))
		if err != nil {
			fail("session record file missing: %s", f)
			continue
		}
		for _, s := range in.Secrets {
			if s != "" && strings.Contains(string(b), s) {
				fail("a secret appears in session file %s", f)
			}
		}
	}
}

// Line is the one-line verdict.
func (r *Result) Line() string {
	v := "PASS"
	if !r.Pass {
		v = "FAIL"
	}
	notes := append([]string(nil), r.Notes...)
	for _, f := range r.Fails {
		notes = append(notes, strings.ReplaceAll(f, " ", "_"))
	}
	n := strings.Join(notes, ",")
	if n == "" {
		n = "none"
	}
	c := r.Commit
	if len(c) > 10 {
		c = c[:10]
	}
	return fmt.Sprintf("SMOKE %s %s %s clvl@%d=%d kills=%d deaths=%d notes=%s", v, r.Class, c, r.Minute, r.Clvl, r.Kills, r.Deaths, n)
}

// Record is the small markdown record of a run (docs/soaks/smoke-<stamp>.md).
func (r *Result) Record(stamp, profile string) string {
	var b strings.Builder
	fmt.Fprintf(&b, "# smoke %s %s\n\n`%s`\n\n", r.Class, stamp, r.Line())
	fmt.Fprintf(&b, "Profile: %s. Commit: %s. Judged at virtual minute %d (rules: data/smoke/rules.json).\n\n", profile, r.Commit, r.Minute)
	if len(r.Fails) == 0 {
		b.WriteString("No rule failed.\n")
	} else {
		b.WriteString("Failed rules:\n")
		for _, f := range r.Fails {
			fmt.Fprintf(&b, "- %s\n", f)
		}
	}
	if len(r.Samples) > 0 {
		b.WriteString("\nSample log lines (cut to 240 characters):\n\n```\n")
		for _, s := range r.Samples {
			b.WriteString(s + "\n")
		}
		b.WriteString("```\n")
	}
	return b.String()
}
