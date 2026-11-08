package watch

import (
	"encoding/json"
	"errors"
	"fmt"
	"hash/fnv"
	"net/http"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"sync"

	voicedata "d2lod-bot/data/voice"
	"d2lod-bot/internal/decide"
	"d2lod-bot/internal/goals"
)

// The hero's voice and journal (TASK-155, docs/unlock-voice.md): a short in-character line at the first level-up, the
// first death and every Codex first, kept as one diary page per milestone. The first level-up opens it (ladder row
// "voice"; a first death opens it too). Lines are data (data/voice/lines.json), the choice is seeded so a replay says the
// same thing, and a mute switch silences the spoken line without losing the diary.

const (
	// VoiceRoute is GET (viewer); VoiceMuteRoute is POST (answer listener). VoiceID is the ladder row.
	VoiceRoute     = "/api/voice"
	VoiceMuteRoute = "/api/voice/mute"
	VoiceID        = "voice"

	voiceSchema      = "watch.voice/1"
	voiceFileVersion = 1
	voiceMinVariants = 6
)

// voiceKinds are the line kinds the code speaks; the table must carry every one with 6 or more lines.
var voiceKinds = []string{"levelup_first", "death_first", "codex_monster", "codex_area", "codex_waypoint", "codex_shrine",
	"codex_drop", "codex_champion", "codex_death"}

var voicePlaceholder = regexp.MustCompile(`\{[^}]*\}`)

// VoiceKind is one kind of line: the diary milestone title and the lines, general and per class.
type VoiceKind struct {
	Milestone string              `json:"milestone"`
	Any       []string            `json:"any"`
	Classes   map[string][]string `json:"classes,omitempty"`
}

// VoiceLines is the parsed table.
type VoiceLines struct {
	Schema  string               `json:"schema"`
	Doc     string               `json:"doc"`
	Note    string               `json:"note"`
	Classes []string             `json:"classes"`
	Kinds   map[string]VoiceKind `json:"kinds"`
}

// ParseVoiceLines checks the table: every kind of voiceKinds with 6 or more general lines and a milestone title, no
// kind the code does not speak, class flavour only for a class listed in "classes", only the placeholders {name} and
// {n}. Each error names the kind or class.
func ParseVoiceLines(b []byte) (*VoiceLines, error) {
	var v VoiceLines
	dec := json.NewDecoder(strings.NewReader(string(b)))
	dec.DisallowUnknownFields()
	if err := dec.Decode(&v); err != nil {
		return nil, fmt.Errorf("voice lines: %w", err)
	}
	if v.Schema != "voice.lines/1" {
		return nil, fmt.Errorf("voice lines: schema %q", v.Schema)
	}
	for k := range v.Kinds {
		if !containsStr(voiceKinds, k) {
			return nil, fmt.Errorf("voice lines: unknown kind %q", k)
		}
	}
	check := func(kind string, lines []string) error {
		for _, l := range lines {
			if strings.TrimSpace(l) == "" {
				return fmt.Errorf("voice lines: kind %q has an empty line", kind)
			}
			for _, p := range voicePlaceholder.FindAllString(l, -1) {
				if p != "{name}" && p != "{n}" {
					return fmt.Errorf("voice lines: kind %q: unknown placeholder %s", kind, p)
				}
			}
		}
		return nil
	}
	for _, k := range voiceKinds {
		vk, ok := v.Kinds[k]
		switch {
		case !ok:
			return nil, fmt.Errorf("voice lines: kind %q is missing", k)
		case len(vk.Any) < voiceMinVariants:
			return nil, fmt.Errorf("voice lines: kind %q needs %d or more lines, has %d", k, voiceMinVariants, len(vk.Any))
		case strings.TrimSpace(vk.Milestone) == "":
			return nil, fmt.Errorf("voice lines: kind %q has no milestone title", k)
		}
		if err := check(k, vk.Any); err != nil {
			return nil, err
		}
		for c, ls := range vk.Classes {
			if !containsStr(v.Classes, c) {
				return nil, fmt.Errorf("voice lines: kind %q: unknown class %q", k, c)
			}
			if err := check(k, ls); err != nil {
				return nil, err
			}
		}
	}
	return &v, nil
}

// DiaryPage is one milestone: what happened and what the hero said.
type DiaryPage struct {
	N         int     `json:"n"`
	T         float64 `json:"t"`
	Kind      string  `json:"kind"`
	Milestone string  `json:"milestone"`
	Text      string  `json:"text"`
}

type voiceFile struct {
	Version int         `json:"version"`
	Muted   bool        `json:"muted,omitempty"`
	Spoken  []string    `json:"spoken"`
	Diary   []DiaryPage `json:"diary"`
}

// Voice speaks the milestones. Safe for concurrent use.
type Voice struct {
	u       *Unlocks
	hub     *Hub // nil: no spoken line in the feed (the diary still fills)
	lines   *VoiceLines
	profile string
	class   func() string
	now     func() float64
	path    string

	mu      sync.Mutex
	f       voiceFile
	spoken  map[string]bool
	waiting []waitingFirst // Codex firsts met before the voice opened; they become quiet diary pages when it opens
}

type waitingFirst struct {
	kind, key, name string
}

const maxWaitingFirsts = 100

// NewVoice builds the voice over the ladder. dir "" keeps the state in memory; otherwise it lives in voice-<profile>.json
// beside the ladder's file. A ladder without the row "voice" is an error naming it; lines nil reads the embedded table.
func NewVoice(u *Unlocks, hub *Hub, lines *VoiceLines, class func() string, now func() float64) (*Voice, error) {
	if _, err := u.Engine.Check(VoiceID); err != nil {
		return nil, err
	}
	if lines == nil {
		var err error
		if lines, err = ParseVoiceLines(voicedata.JSON); err != nil {
			return nil, err
		}
	}
	if class == nil {
		class = func() string { return "" }
	}
	if now == nil {
		now = func() float64 { return 0 }
	}
	v := &Voice{u: u, hub: hub, lines: lines, profile: u.Engine.Profile(), class: class, now: now, spoken: map[string]bool{}}
	v.f = voiceFile{Version: voiceFileVersion, Spoken: []string{}, Diary: []DiaryPage{}}
	if dir := u.Engine.Dir(); dir != "" {
		v.path = filepath.Join(dir, "voice-"+v.profile+".json")
		b, err := os.ReadFile(v.path)
		switch {
		case errors.Is(err, os.ErrNotExist):
		case err != nil:
			return nil, err
		default:
			if err := json.Unmarshal(b, &v.f); err != nil {
				return nil, fmt.Errorf("voice: %s: %w", v.path, err)
			}
			if v.f.Version != voiceFileVersion {
				return nil, fmt.Errorf("voice: %s: version %d, want %d", v.path, v.f.Version, voiceFileVersion)
			}
		}
	}
	for _, s := range v.f.Spoken {
		v.spoken[s] = true
	}
	return v, nil
}

func (v *Voice) saveLocked() error {
	if v.path == "" {
		return nil
	}
	b, err := json.MarshalIndent(v.f, "", " ")
	if err != nil {
		return err
	}
	tmp := v.path + ".tmp"
	if err := os.WriteFile(tmp, append(b, '\n'), 0o644); err != nil {
		return err
	}
	return os.Rename(tmp, v.path)
}

// pick chooses the line: the general lines plus the hero class's flavour, by a hash of profile, kind and key.
func (v *Voice) pick(kind, key, class string) (string, error) {
	vk := v.lines.Kinds[kind]
	pool := append([]string(nil), vk.Any...)
	var err error
	if class != "" {
		if !containsStr(v.lines.Classes, class) {
			err = fmt.Errorf("voice: unknown class %q (general lines only)", class)
		}
		pool = append(pool, vk.Classes[class]...)
	}
	h := fnv.New32a()
	h.Write([]byte(v.profile + "/" + kind + "/" + key))
	return pool[int(h.Sum32())%len(pool)], err
}

// speak says the line of a milestone once. Closed (the row "voice" is not open) or already spoken: nothing.
func (v *Voice) speak(kind, key, name string, n int) {
	if !v.u.Engine.Allowed(VoiceID) {
		if strings.HasPrefix(kind, "codex_") {
			v.mu.Lock()
			if len(v.waiting) < maxWaitingFirsts {
				v.waiting = append(v.waiting, waitingFirst{kind, key, name})
			}
			v.mu.Unlock()
		}
		return
	}
	v.mu.Lock()
	queued := v.waiting
	v.waiting = nil
	v.mu.Unlock()
	for _, w := range queued {
		v.say(w.kind, w.key, w.name, 0, true)
	}
	v.say(kind, key, name, n, false)
}

// say writes the page and, unless quiet or muted, puts the line in the feed.
func (v *Voice) say(kind, key, name string, n int, quiet bool) {
	sk := kind + "/" + strings.ToLower(key)
	v.mu.Lock()
	if v.spoken[sk] {
		v.mu.Unlock()
		return
	}
	text, err := v.pick(kind, sk, strings.ToLower(v.class()))
	if err != nil {
		v.u.fail(err)
	}
	text = safeText(strings.NewReplacer("{name}", name, "{n}", fmt.Sprint(n)).Replace(text), 200)
	ms := safeText(strings.NewReplacer("{name}", name).Replace(v.lines.Kinds[kind].Milestone), 80)
	page := DiaryPage{N: len(v.f.Diary) + 1, T: v.now(), Kind: kind, Milestone: ms, Text: text}
	v.spoken[sk] = true
	v.f.Spoken = append(v.f.Spoken, sk)
	v.f.Diary = append(v.f.Diary, page)
	if err := v.saveLocked(); err != nil {
		v.u.fail(err)
	}
	muted := v.f.Muted
	v.mu.Unlock()
	if v.hub != nil && !muted && !quiet {
		v.hub.Say(text)
	}
}

// OnGoalEvent speaks the first level-up and the first death. Call it after Unlocks.OnGoalEvent, so the level-up that
// opens the voice is also its first line.
func (v *Voice) OnGoalEvent(ev goals.Event) {
	switch ev.Type {
	case goals.EvLevelUp:
		if ev.Level >= 2 {
			v.speak("levelup_first", "1", "", ev.Level)
		}
	case goals.EvDeath:
		v.speak("death_first", "1", ev.Monster, 0)
	}
}

// codexKinds maps a Codex kind to its line kind.
var codexKinds = map[string]string{"monster": "codex_monster", "area": "codex_area", "waypoint": "codex_waypoint",
	"shrine": "codex_shrine", "drop": "codex_drop", "champion": "codex_champion", "death": "codex_death"}

// OnFired speaks every new Codex entry. A kind the table has no line for is an error naming it (nothing is skipped
// silently). A first death by a Codex entry also gives the first-death page when the death event came too early.
func (v *Voice) OnFired(fired []decide.Fired) {
	for _, f := range fired {
		kind, ok := codexKinds[f.Entry.Kind]
		if !ok {
			v.u.fail(fmt.Errorf("voice: no line kind for Codex kind %q", f.Entry.Kind))
			continue
		}
		if f.Entry.Kind == "death" {
			v.speak("death_first", "1", f.Entry.Subject, 0)
		}
		v.speak(kind, f.Entry.ID, f.Entry.Subject, 0)
	}
}

// VoiceView is GET /api/voice. Locked: the teaser only.
type VoiceView struct {
	Schema string      `json:"schema"`
	Open   bool        `json:"open"`
	Title  string      `json:"title"`
	Teaser string      `json:"teaser,omitempty"`
	Muted  bool        `json:"muted"`
	Latest *DiaryPage  `json:"latest,omitempty"`
	Diary  []DiaryPage `json:"diary"`
}

// View is the payload.
func (v *Voice) View() VoiceView {
	st, _ := v.u.state(VoiceID)
	out := VoiceView{Schema: voiceSchema, Open: st.Open, Title: st.Title, Diary: []DiaryPage{}}
	if !st.Open {
		out.Teaser = st.Teaser
		return out
	}
	v.mu.Lock()
	defer v.mu.Unlock()
	out.Muted = v.f.Muted
	out.Diary = append(out.Diary, v.f.Diary...)
	if n := len(v.f.Diary); n > 0 {
		out.Latest = &v.f.Diary[n-1]
	}
	return out
}

// SetMuted is the mute switch: a muted voice still writes the diary but says nothing in the feed.
func (v *Voice) SetMuted(m bool) error {
	v.mu.Lock()
	defer v.mu.Unlock()
	prev := v.f.Muted
	v.f.Muted = m
	if err := v.saveLocked(); err != nil {
		v.f.Muted = prev
		return err
	}
	return nil
}

// Handler serves GET /api/voice.
func (v *Voice) Handler() http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet && r.Method != http.MethodHead {
			http.Error(w, "GET only", http.StatusMethodNotAllowed)
			return
		}
		writeJSON(w, 200, v.View())
	})
}

type voiceMuteBody struct {
	Muted *bool `json:"muted"`
}

// MuteHandler is POST /api/voice/mute {muted: bool}, on the answer listener with the same gate as the other answer
// routes. It works while the voice is locked too (a viewer who does not want it can say so first).
func (v *Voice) MuteHandler(o AnswerOptions) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if !unlockPostGate(o, w, r) {
			return
		}
		var b voiceMuteBody
		if err := decodeSmall(r, &b); err != nil || b.Muted == nil {
			writeJSON(w, http.StatusBadRequest, unlockReply{Code: "body", Error: `body must be {"muted": true|false}`})
			return
		}
		if err := v.SetMuted(*b.Muted); err != nil {
			writeJSON(w, http.StatusInternalServerError, unlockReply{Code: "internal", Error: err.Error()})
			return
		}
		writeJSON(w, 200, unlockReply{OK: true})
	})
}
