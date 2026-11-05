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

// Profiles named in the T0 drill. They mirror the profiles of internal/decide's tests (those live in a test file and
// cannot be imported): cautious replies the default, bold the last answer, inconsistent anything, and a holdout is
// generated from a seed (never a live person).
var KnownProfiles = []string{"cautious", "bold", "inconsistent"}

// ProfileSource replies the way a named player profile would. It is deterministic per seed.
type ProfileSource struct {
	mu                           sync.Mutex
	name                         string
	rng, g                       *rand.Rand
	flip, ignore, unsure, yesPct float64
	pref                         map[string]int
}

// NewProfileSource makes cautious, bold, inconsistent or holdout (seeded).
func NewProfileSource(name string, seed int64) (*ProfileSource, error) {
	p := &ProfileSource{name: name, rng: rand.New(rand.NewSource(seed)), pref: map[string]int{}}
	switch name {
	case "cautious", "bold", "inconsistent":
	case "holdout":
		p.g = rand.New(rand.NewSource(seed ^ 0x5eed))
		p.flip, p.ignore, p.unsure, p.yesPct = p.g.Float64()*0.5, p.g.Float64()*0.6, p.g.Float64()*0.2, p.g.Float64()
	default:
		return nil, fmt.Errorf("ladder: unknown answer profile %q", name)
	}
	return p, nil
}

func (p *ProfileSource) Reply(v ThoughtView) Reply {
	p.mu.Lock()
	defer p.mu.Unlock()
	r := Reply{Delay: time.Duration(1+p.rng.Intn(30)) * time.Second}
	if v.Confirm {
		switch p.name {
		case "cautious":
			r.Answer = "no"
		case "bold":
			r.Answer = "yes"
		case "inconsistent":
			r.Answer = []string{"yes", "no", ""}[p.rng.Intn(3)]
		default:
			r.Answer = "no"
			if p.rng.Float64() < p.yesPct {
				r.Answer = "yes"
			}
		}
		return r
	}
	n := len(v.Choices)
	if n == 0 {
		return r
	}
	switch p.name {
	case "cautious":
		r.Answer = v.Default
	case "bold":
		if p.rng.Intn(10) != 0 {
			r.Answer = v.Choices[n-1]
		}
	case "inconsistent":
		if p.rng.Intn(5) == 0 {
			r.Answer = decide.NotSure
		} else {
			r.Answer = v.Choices[p.rng.Intn(n)]
		}
	default:
		i, ok := p.pref[v.Point]
		if !ok {
			i = p.g.Intn(n)
			p.pref[v.Point] = i
		}
		switch x := p.rng.Float64(); {
		case x < p.ignore:
		case x < p.ignore+p.unsure:
			r.Answer = decide.NotSure
		case x < p.ignore+p.unsure+p.flip:
			r.Answer = v.Choices[p.rng.Intn(n)]
		default:
			r.Answer = v.Choices[i]
		}
	}
	return r
}
