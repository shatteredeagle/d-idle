package scenario

import (
	"bytes"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strings"
	"time"
)

// Earned saves (card pgearned, design 2026-11-21 section 6.3, ADR 0027 item 8). A nightly act run from a fresh hero
// exports its save at milestones; a step scenario may then declare `pre.from_save: <milestone>` instead of setup verbs
// and is EARNED. The save files are binaries and live on the builder under $PG_EARNED_DIR, never in the repo; this file
// is the manifest, the milestone catalogue and the rules that keep a CHEATED hero out of the chain.

// Format versions of the two data files.
const (
	EarnedVersion     = "didle.pg-earned/1"
	MilestonesVersion = "didle.pg-milestones/1"
)

// Milestone is one export point of the earned chain (tools/provingground/earned/milestones.json).
type Milestone struct {
	ID    string `json:"id"`
	Title string `json:"title"`
	// MinLevel is the character level the hero must have reached when the save is exported (a save below it is refused).
	MinLevel int `json:"min_level"`
}

// MilestoneFile is milestones.json.
type MilestoneFile struct {
	Version    string      `json:"version"`
	Milestones []Milestone `json:"milestones"`
}

// LoadMilestones reads the catalogue; unknown fields, a duplicate or malformed id are errors.
func LoadMilestones(path string) (MilestoneFile, error) {
	b, err := os.ReadFile(path)
	if err != nil {
		return MilestoneFile{}, err
	}
	var f MilestoneFile
	dec := json.NewDecoder(bytes.NewReader(b))
	dec.DisallowUnknownFields()
	if err := dec.Decode(&f); err != nil {
		return MilestoneFile{}, fmt.Errorf("%s: %w", path, err)
	}
	if f.Version != MilestonesVersion {
		return MilestoneFile{}, fmt.Errorf("%s: version %q, want %q", path, f.Version, MilestonesVersion)
	}
	seen := map[string]bool{}
	for _, m := range f.Milestones {
		if !validID(m.ID) || seen[m.ID] || m.Title == "" || m.MinLevel < 1 || m.MinLevel > 99 {
			return MilestoneFile{}, fmt.Errorf("%s: milestone %+v: id, title and min_level 1..99 are required, ids are unique", path, m)
		}
		seen[m.ID] = true
	}
	return f, nil
}

// Find returns the milestone or false.
func (f MilestoneFile) Find(id string) (Milestone, bool) {
	for _, m := range f.Milestones {
		if m.ID == id {
			return m, true
		}
	}
	return Milestone{}, false
}

// SaveFile is one file of an exported save with its hash (so a copy that changed is noticed).
type SaveFile struct {
	Name   string `json:"name"`
	SHA256 string `json:"sha256"`
}

// SaveLoad is one portability observation: the save loaded (or was refused) on a d2d build.
type SaveLoad struct {
	D2DSHA256 string `json:"d2d_sha256"`
	Result    string `json:"result"` // ok, refused, changed (loaded but the hero read back differently)
	Run       string `json:"run"`    // ledger run id of the check
}

// SaveManifest describes one exported save: which EARNED run made it, with which binaries, and where it has been
// loaded. A manifest is refused unless the run behind it was EARNED with zero writes after the start (the no-crutch rule
// and the provenance rule of the private agent rules: a CHEATED hero never founds an earned chain).
type SaveManifest struct {
	Version   string `json:"version"`
	Milestone string `json:"milestone"`
	Class     string `json:"class"`
	Hero      string `json:"hero"`
	Level     int    `json:"level"`
	Run       string `json:"run"` // ledger run id of the EARNED run that exported it
	Scenario  string `json:"scenario"`
	// Parent is the run id of the earlier save this run started from (empty: a fresh hero). The chain is unbroken only
	// while every link is EARNED, so a save exported by a run that itself used from_save is allowed and says so here.
	Parent           string     `json:"parent,omitempty"`
	BotCommit        string     `json:"bot_commit"`
	BotDirty         bool       `json:"bot_dirty,omitempty"`
	D2DSHA256        string     `json:"d2d_sha256"`
	Setup            string     `json:"setup"`
	SetupTags        []string   `json:"setup_tags,omitempty"`
	WritesAfterStart *int       `json:"writes_after_start"`
	Exported         string     `json:"exported"` // RFC3339 UTC
	Files            []SaveFile `json:"files"`
	LoadedOn         []SaveLoad `json:"loaded_on,omitempty"`
}

// Validate refuses a manifest that could carry a hero play did not make.
func (m SaveManifest) Validate(ms MilestoneFile) error {
	at := fmt.Sprintf("earned save %s/%s", m.Milestone, m.Class)
	ml, ok := ms.Find(m.Milestone)
	switch {
	case m.Version != EarnedVersion:
		return fmt.Errorf("%s: version %q, want %q", at, m.Version, EarnedVersion)
	case !ok:
		return fmt.Errorf("%s: unknown milestone", at)
	case m.Level < ml.MinLevel:
		return fmt.Errorf("%s: level %d is below the milestone's %d", at, m.Level, ml.MinLevel)
	case m.Setup != SetupEarned:
		return fmt.Errorf("%s: setup %q: only an EARNED run exports a save (CHEATED and IMPOSSIBLE heroes are mechanics evidence only)", at, m.Setup)
	case m.WritesAfterStart == nil || *m.WritesAfterStart != 0:
		return fmt.Errorf("%s: writes_after_start must be recorded and 0", at)
	case m.BotDirty:
		return fmt.Errorf("%s: bot_dirty: a run of an unclean tree is not evidence", at)
	case m.Run == "" || m.Scenario == "" || m.Hero == "" || m.BotCommit == "" || m.D2DSHA256 == "" || m.Exported == "" || len(m.Files) == 0:
		return fmt.Errorf("%s: run, scenario, hero, bot_commit, d2d_sha256, exported and files are required", at)
	}
	if _, err := ClassIndex(m.Class); err != nil {
		return fmt.Errorf("%s: %w", at, err)
	}
	for _, t := range m.SetupTags {
		if u := strings.ToUpper(t); u == SetupCheated || u == SetupImpossible {
			return fmt.Errorf("%s: setup tag %s in an earned chain", at, t)
		}
	}
	for _, f := range m.Files {
		if f.Name == "" || strings.ContainsAny(f.Name, `/\`) || len(f.SHA256) != 64 {
			return fmt.Errorf("%s: file %+v: a bare name and a sha256 are required", at, f)
		}
	}
	return nil
}

// NewManifest builds the manifest of an exported save from the ledger line of the run that made it. It refuses a run that
// was not EARNED, wrote anything after the start, ran on a dirty tree, or did not end GREEN or reach the milestone level.
// parent is the run id of the save that run started from ("" for a fresh hero).
func NewManifest(rec LedgerRecord, ms MilestoneFile, milestone, hero string, level int, files []SaveFile, parent string, now time.Time) (SaveManifest, error) {
	m := SaveManifest{Version: EarnedVersion, Milestone: milestone, Class: rec.Class, Hero: hero, Level: level, Run: rec.Run,
		Scenario: rec.Scenario, Parent: parent, BotCommit: rec.BotCommit, BotDirty: rec.BotDirty, D2DSHA256: rec.D2DSHA256,
		Setup: rec.Setup, SetupTags: rec.SetupTags, WritesAfterStart: rec.WritesN, Exported: now.UTC().Format(time.RFC3339), Files: files}
	if rec.Verdict != Green {
		return SaveManifest{}, fmt.Errorf("earned save: run %s is %s, only a GREEN run exports a save", rec.Run, rec.Verdict)
	}
	return m, m.Validate(ms)
}

// Portability of a save on a d2d build (the U of design 6.3).
const (
	PortSame       = "same-build" // exported by this very d2d binary
	PortVerified   = "verified"   // an earlier check loaded it unchanged on this binary
	PortUnverified = "unverified" // never loaded on this binary: the runner must not trust it
	PortRefused    = "refused"    // a check on this binary refused it or read the hero back changed
)

// Portability says whether the save is known to load on the d2d build with this hash. A recorded refusal wins over any
// earlier success on the same binary.
func (m SaveManifest) Portability(d2dSHA256 string) string {
	state := PortUnverified
	if m.D2DSHA256 == d2dSHA256 {
		state = PortSame
	}
	for _, l := range m.LoadedOn {
		if l.D2DSHA256 != d2dSHA256 {
			continue
		}
		if l.Result == "ok" && state == PortUnverified {
			state = PortVerified
		} else if l.Result != "ok" {
			return PortRefused
		}
	}
	return state
}

// ManifestPath is where the manifest of a milestone and class sits under the earned store.
func ManifestPath(dir, milestone, class string) string {
	return filepath.Join(dir, milestone, strings.ToLower(class), "manifest.json")
}

// ReadManifest loads and validates one manifest.
func ReadManifest(path string, ms MilestoneFile) (SaveManifest, error) {
	b, err := os.ReadFile(path)
	if err != nil {
		return SaveManifest{}, err
	}
	var m SaveManifest
	dec := json.NewDecoder(bytes.NewReader(b))
	dec.DisallowUnknownFields()
	if err := dec.Decode(&m); err != nil {
		return SaveManifest{}, fmt.Errorf("%s: %w", path, err)
	}
	if err := m.Validate(ms); err != nil {
		return SaveManifest{}, fmt.Errorf("%s: %w", path, err)
	}
	return m, nil
}

// ResolveSave finds the manifest a scenario's from_save names for one class. A missing save is an error naming the
// milestone and class (the run is BLOCKED with it, never run from a made-up hero).
func ResolveSave(dir, milestone, class string, ms MilestoneFile) (SaveManifest, error) {
	m, err := ReadManifest(ManifestPath(dir, milestone, class), ms)
	if err != nil {
		return SaveManifest{}, fmt.Errorf("no earned save for %s/%s: %w", milestone, class, err)
	}
	if !strings.EqualFold(m.Class, class) {
		return SaveManifest{}, fmt.Errorf("earned save %s/%s holds a %s hero", milestone, class, m.Class)
	}
	return m, nil
}

var cheatedWord = regexp.MustCompile(`CHEATED|IMPOSSIBLE`)

// validateFromSave is the static half of the earned rules, run by Scenario.Validate: a scenario that starts from an
// earned save declares nothing the save does not hold, so its hero stays EARNED. Taps, board acceptance and bot knobs are
// the player's own side and stay allowed.
func (s Scenario) validateFromSave() error {
	p := s.Pre
	if p.FromSave == "" {
		return nil
	}
	at := "scenario " + s.ID + ": pre.from_save"
	if !validID(p.FromSave) {
		return fmt.Errorf("%s %q: not a milestone id", at, p.FromSave)
	}
	var writes []string
	for name, set := range map[string]bool{
		"level": p.Level > 0, "gold": p.Gold > 0, "stash": p.Stash > 0, "difficulty": p.Difficulty != "", "waypoints": len(p.Waypoints) > 0,
		"quests": len(p.Quests) > 0, "items": len(p.Items) > 0, "belt": len(p.Belt) > 0, "merc": p.Merc != nil, "merc_level": p.MercLevel > 0,
		"rooms_up": p.RoomsUp, "life_pct": p.LifePct > 0, "pack": len(p.Pack) > 0, "worn": len(p.Worn) > 0, "vault_records": p.Vault != nil,
		"spawn": len(p.Spawn) > 0, "warp": p.Warp != nil, "objects": len(p.Objects) > 0, "stats": len(p.Stats) > 0, "ether": p.Ether > 0,
	} {
		if set {
			writes = append(writes, name)
		}
	}
	if len(writes) > 0 {
		sort.Strings(writes)
		return fmt.Errorf("%s with setup writes (%s): the hero is whatever the save holds; a changed hero is CHEATED, so drop from_save or the writes", at, strings.Join(writes, ", "))
	}
	if p.Start == StartField {
		return fmt.Errorf("%s with start field: a field start needs the warp, which makes the run CHEATED (design 6.3)", at)
	}
	if cheatedWord.MatchString(s.Provenance) {
		return fmt.Errorf("%s: provenance names CHEATED or IMPOSSIBLE; an earned-save scenario is EARNED", at)
	}
	return nil
}

// CheckProvenance is the provenance rule of the private agent rules as a static check: a scenario whose setup is
// CHEATED must say so in its provenance text (the row of docs/probes/test-characters.md comes from it), and one that
// starts from an earned save must build an EARNED setup in every matrix class.
func (s Scenario) CheckProvenance() error {
	for _, c := range s.Matrix.Classes {
		su, err := BuildSetup(s, c, s.Matrix.Seeds[0])
		if err != nil {
			return err
		}
		tag := SetupTag(su.Tags)
		if tag == SetupCheated && !strings.Contains(s.Provenance, "CHEATED") {
			return fmt.Errorf("scenario %s (%s): the setup is CHEATED (%s) but provenance does not say CHEATED: a cheated hero must be tagged (docs/probes/test-characters.md)", s.ID, c, strings.Join(su.Tags[1:], ", "))
		}
		if s.Pre.FromSave != "" && tag != SetupEarned {
			return fmt.Errorf("scenario %s (%s): from_save but the setup is %s", s.ID, c, tag)
		}
	}
	return nil
}
