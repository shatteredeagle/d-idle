package ladder

import (
	"encoding/json"
	"fmt"
	"math/rand"
	"os"
	"strings"
	"sync"
	"time"

	"d2lod-bot/internal/decide"
)

// ScriptFile is the scripted-answer file of the T0 drill: rows are matched top to bottom on the thought's point and an
// optional key fragment; Times limits how often a row answers (0 = always). Anything unmatched falls to Else.
//
//	{"profile": "cautious", "rows": [{"point": "t1.repair", "answer": "repair", "delaySec": 5}], "else": {"answer": ""}}
type ScriptFile struct {
	Profile string      `json:"profile,omitempty"`
	Rows    []ScriptRow `json:"rows"`
	Else    ScriptRow   `json:"else"`
}

// ScriptRow answers one thought. Confirm is the reply to a confirm thought ("yes", "no" or "" to let it fade).
type ScriptRow struct {
	Point    string  `json:"point,omitempty"`
	KeyHas   string  `json:"keyHas,omitempty"`
	Answer   string  `json:"answer"`
	Confirm  string  `json:"confirm,omitempty"`
	DelaySec float64 `json:"delaySec,omitempty"`
	Times    int     `json:"times,omitempty"`
}

// FileSource serves a ScriptFile. Reading the file is the only input it has.
type FileSource struct {
	mu   sync.Mutex
	f    ScriptFile
	used []int
}

// LoadScript reads a script file; it never reads stdin or the network.
func LoadScript(path string) (*FileSource, error) {
	b, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("ladder: answer script: %w", err)
	}
	return ParseScript(b)
}

func ParseScript(b []byte) (*FileSource, error) {
	var f ScriptFile
	if err := json.Unmarshal(b, &f); err != nil {
		return nil, fmt.Errorf("ladder: answer script: %w", err)
	}
	return &FileSource{f: f, used: make([]int, len(f.Rows))}, nil
}

func (s *FileSource) Profile() string { return s.f.Profile }

func (s *FileSource) Reply(v ThoughtView) Reply {
	s.mu.Lock()
	defer s.mu.Unlock()
	row := s.f.Else
	for i, r := range s.f.Rows {
		if (r.Point == "" || r.Point == v.Point) && strings.Contains(v.Key, r.KeyHas) && (r.Times == 0 || s.used[i] < r.Times) {
			s.used[i]++
			row = r
			break
		}
	}
	ans := row.Answer
	if v.Confirm {
		ans = row.Confirm
	}
	return Reply{Answer: ans, Delay: time.Duration(row.DelaySec * float64(time.Second))}
}

// ProfileSource replies the way a named simulated player (decide.Profile: cautious, bold, inconsistent, or a seeded
// holdout) would. It is deterministic per seed and never a live person.
type ProfileSource struct {
	mu   sync.Mutex
	prof decide.Profile
	cat  *decide.Catalogue
	rng  *rand.Rand
}

// NewProfileSource makes cautious, bold, inconsistent or holdout (seeded) from internal/decide's profiles.
func NewProfileSource(name string, seed int64) (*ProfileSource, error) {
	prof, err := decide.NewProfile(name, seed)
	if err != nil {
		return nil, err
	}
	cat, err := decide.Shipped()
	if err != nil {
		return nil, err
	}
	return &ProfileSource{prof: prof, cat: cat, rng: rand.New(rand.NewSource(seed))}, nil
}

func (p *ProfileSource) Reply(v ThoughtView) Reply {
	p.mu.Lock()
	defer p.mu.Unlock()
	r := Reply{Delay: time.Duration(1+p.rng.Intn(30)) * time.Second}
	if v.Confirm {
		r.Answer = p.prof.Confirm(p.rng)
		return r
	}
	if pt, err := p.cat.Point(v.Point); err == nil && len(pt.Answers) > 0 {
		r.Answer = p.prof.Reply(p.rng, pt, v.Default)
	}
	return r
}
