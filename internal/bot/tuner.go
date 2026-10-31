package bot

import (
	"fmt"
	"sort"
	"sync"

	"d2lod-bot/internal/api"
	"d2lod-bot/internal/env"
	"d2lod-bot/internal/store"
)

// Tuner is the live, editable view of an instance's env.Tuning behind GET/PUT /knobs. An edit rebuilds the tuning
// from defaults + profile + overrides and replaces *Tune in place; behaviors that copy a group (t := e.Tune.Bot) pick it
// up at their next read. UNVERIFIED: behaviors read the shared struct without a lock, so a write can race a read; each
// field is a word-sized value, so the worst case is one tick with the old or new number.
type Tuner struct {
	tune *env.Tuning
	file *store.File[tunerState] // nil: not persisted

	mu        sync.Mutex
	custom    *Profile
	profile   string
	overrides map[string]float64
}

type tunerState struct {
	Profile   string             `json:"profile,omitempty"`
	Overrides map[string]float64 `json:"overrides,omitempty"`
}

// NewTuner wraps tune (the pointer in env.Env.Tune) with the profile and overrides it was built from (Config).
// path, if set, persists later edits (store.File) and restores them now.
func NewTuner(tune *env.Tuning, cfg Config, path string) (*Tuner, error) {
	t := &Tuner{tune: tune, custom: cfg.Custom, profile: cfg.Profile, overrides: map[string]float64{}}
	if cfg.Custom != nil {
		t.profile = t.customName()
	}
	if t.profile == "" {
		t.profile = "optimal"
	}
	for k, v := range cfg.Tuning {
		t.overrides[k] = v
	}
	if path != "" {
		t.file = &store.File[tunerState]{Path: path, Version: 1}
		st, info, err := t.file.Load()
		if err != nil {
			return nil, err
		}
		if info.Exists {
			if st.Profile != "" {
				t.profile = st.Profile
			}
			t.overrides = st.Overrides
			if t.overrides == nil {
				t.overrides = map[string]float64{}
			}
			if err := t.rebuild(); err != nil { // a saved edit that no longer resolves must not stop the bot
				return nil, fmt.Errorf("saved tuning %s: %w", path, err)
			}
		}
	}
	return t, nil
}

func (t *Tuner) customName() string {
	if t.custom != nil && t.custom.Name != "" {
		return t.custom.Name
	}
	return "custom"
}

func (t *Tuner) resolve(name string) (Profile, error) {
	if t.custom != nil && name == t.customName() {
		p := *t.custom
		p.Name = name
		return p, nil
	}
	return ProfileByName(name)
}

func (t *Tuner) build(profile string, over map[string]float64) (env.Tuning, error) {
	nt := env.DefaultTuning()
	p, err := t.resolve(profile)
	if err != nil {
		return nt, err
	}
	p.Apply(&nt)
	return nt, ApplyOverrides(&nt, over)
}

// rebuild applies profile and overrides to the live tuning. Callers hold mu (or are the constructor).
func (t *Tuner) rebuild() error {
	nt, err := t.build(t.profile, t.overrides)
	if err != nil {
		return err
	}
	*t.tune = nt
	return nil
}

// Snapshot implements api.Tuning.
func (t *Tuner) Snapshot() api.TuneBody {
	t.mu.Lock()
	defer t.mu.Unlock()
	b := api.TuneBody{Knobs: env.Knobs(), Values: map[string]float64{}, Profile: t.profile, Upgrades: []api.Upgrade{}, ReadOnly: false}
	for _, k := range b.Knobs {
		if v, err := GetKnob(t.tune, k); err == nil {
			b.Values[k.Key] = v
		}
	}
	profs := Profiles()
	if t.custom != nil {
		c := *t.custom
		c.Name = t.customName()
		profs = append(profs, c)
	}
	for _, p := range profs {
		b.Profiles = append(b.Profiles, api.Profile{Name: p.Name, Label: p.Name, Desc: p.Desc})
	}
	return b
}

// Edit implements api.Tuning. The edit is all or nothing: unknown knobs or profiles change nothing.
func (t *Tuner) Edit(e api.TuneEdit) error {
	t.mu.Lock()
	defer t.mu.Unlock()
	profile := t.profile
	if e.Profile != nil {
		profile = *e.Profile
	}
	over := make(map[string]float64, len(t.overrides)+len(e.Overrides))
	for k, v := range t.overrides {
		over[k] = v
	}
	keys := make([]string, 0, len(e.Overrides))
	for k := range e.Overrides {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	for _, k := range keys {
		if v := e.Overrides[k]; v == nil {
			delete(over, k)
		} else {
			over[k] = *v
		}
	}
	nt, err := t.build(profile, over)
	if err != nil {
		return fmt.Errorf("%w: %v", api.ErrInvalidTune, err)
	}
	t.profile, t.overrides = profile, over
	*t.tune = nt
	if t.file != nil {
		return t.file.Save(tunerState{Profile: profile, Overrides: over})
	}
	return nil
}

// Unlock implements api.Tuning. There is no upgrade data yet (docs/plan has the design, nothing backs it), so the
// list is empty and every unlock is refused.
func (t *Tuner) Unlock(id string) error { return api.ErrNoUpgrades }
