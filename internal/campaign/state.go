package campaign

import (
	"encoding/json"
	"errors"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"time"
)

// Mode is how the bot picks its work.
type Mode string

const (
	Guided Mode = "guided" // default: follow the spine
	Free   Mode = "free"   // today's planner (quest floors from internal/task), the spine only advises
	Orders Mode = "orders" // only the player's orders; between orders the hero farms its current safe zone
)

func validMode(m Mode) bool { return m == Guided || m == Free || m == Orders }

// Answer is the player's reply to a "go at the floor?" question, with the level it was given at: a no holds until
// the next level-up, then the question may come back.
type Answer struct {
	Yes   bool `json:"yes"`
	Level int  `json:"level"`
}

// Farming is the level step the hero is in for a chapter (its start feeds the offer to go at the floor).
type Farming struct {
	Chapter string    `json:"chapter"`
	Since   time.Time `json:"since"`
}

// HeroState is the per-hero campaign state the player changes (persisted in state/campaign.json).
type HeroState struct {
	Mode    Mode                 `json:"mode"`
	GoAt    GoAt                 `json:"goAt"`
	Intents map[string]Intent    `json:"intents,omitempty"` // the player's per-chapter intents (override the defaults)
	GoNow   string               `json:"goNow,omitempty"`   // the chapter the player sent the hero to now
	Asked   map[string]time.Time `json:"asked,omitempty"`   // when the floor question for a chapter was queued
	Answers map[string]Answer    `json:"answers,omitempty"`
	Done    map[string]time.Time `json:"done,omitempty"` // when a chapter was first seen done
	Skipped map[string]time.Time `json:"skipped,omitempty"`
	Farm    *Farming             `json:"farm,omitempty"`
}

// NewHeroState is the default: Guided, Auto goes at the recommended level.
func NewHeroState() *HeroState { return &HeroState{Mode: Guided, GoAt: GoRecommended} }

// Intent is the effective intent of c for class: the player's, else the chapter default.
func (h *HeroState) Intent(c *Chapter, class string) Intent {
	if i, ok := h.Intents[c.ID]; ok {
		return i
	}
	return c.DefaultIntent(class)
}

// SetIntent records the player's intent for chapter id (Skip also stamps Skipped for the Chronicle).
func (h *HeroState) SetIntent(s *Spine, id string, i Intent, now time.Time) error {
	if s.Chapter(id) == nil {
		return fmt.Errorf("campaign: unknown chapter %q", id)
	}
	if !validIntent(i) {
		return fmt.Errorf("campaign: unknown intent %q", i)
	}
	if h.Intents == nil {
		h.Intents = map[string]Intent{}
	}
	h.Intents[id] = i
	if i == Skip {
		if h.Skipped == nil {
			h.Skipped = map[string]time.Time{}
		}
		h.Skipped[id] = now
	}
	if h.GoNow == id && (i == Skip || i == Hold) {
		h.GoNow = ""
	}
	return nil
}

// SetGoNow sends the hero to chapter id now, whatever its band ("" clears it).
func (h *HeroState) SetGoNow(s *Spine, id string) error {
	if id != "" && s.Chapter(id) == nil {
		return fmt.Errorf("campaign: unknown chapter %q", id)
	}
	h.GoNow = id
	return nil
}

// SetAnswer records the reply to the floor question of chapter id at level lvl.
func (h *HeroState) SetAnswer(id string, yes bool, lvl int) {
	if h.Answers == nil {
		h.Answers = map[string]Answer{}
	}
	h.Answers[id] = Answer{Yes: yes, Level: lvl}
	delete(h.Asked, id)
}

// SetMode switches the mode.
func (h *HeroState) SetMode(m Mode) error {
	if !validMode(m) {
		return fmt.Errorf("campaign: unknown mode %q", m)
	}
	h.Mode = m
	return nil
}

// SetGoAt switches the policy row.
func (h *HeroState) SetGoAt(g GoAt) error {
	if g != GoRecommended && g != GoFloor {
		return fmt.Errorf("campaign: unknown goAt %q", g)
	}
	h.GoAt = g
	return nil
}

// StateSchema tags state/campaign.json.
const StateSchema = "campaign.state/1"

// Store is the state file: one HeroState per hero name.
type Store struct {
	Schema string                `json:"schema"`
	Heroes map[string]*HeroState `json:"heroes"`
}

// Hero returns the state of name, created with the defaults when absent.
func (s *Store) Hero(name string) *HeroState {
	if s.Heroes == nil {
		s.Heroes = map[string]*HeroState{}
	}
	h := s.Heroes[name]
	if h == nil {
		h = NewHeroState()
		s.Heroes[name] = h
	}
	if h.Mode == "" {
		h.Mode = Guided
	}
	return h // GoAt "" = the controller's default policy (the knob), see Controller.State
}

// LoadStore reads path; a missing file is an empty store.
func LoadStore(path string) (*Store, error) {
	b, err := os.ReadFile(path)
	if errors.Is(err, fs.ErrNotExist) {
		return &Store{Schema: StateSchema}, nil
	}
	if err != nil {
		return nil, err
	}
	var s Store
	if err := json.Unmarshal(b, &s); err != nil {
		return nil, fmt.Errorf("campaign: %s: %w", path, err)
	}
	if s.Schema != StateSchema {
		return nil, fmt.Errorf("campaign: %s: schema %q, want %q", path, s.Schema, StateSchema)
	}
	return &s, nil
}

// Save writes the store atomically (temp file and rename).
func (s *Store) Save(path string) error {
	s.Schema = StateSchema
	b, err := json.MarshalIndent(s, "", " ")
	if err != nil {
		return err
	}
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return err
	}
	tmp := path + ".tmp"
	if err := os.WriteFile(tmp, b, 0o644); err != nil {
		return err
	}
	return os.Rename(tmp, path)
}
