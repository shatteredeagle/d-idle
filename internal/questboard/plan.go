package questboard

import (
	"fmt"
	"path/filepath"
	"regexp"
	"sync"
	"time"

	"d2lod-bot/internal/store"
)

// PlanVersion is the on-disk version of the plan-record file.
const PlanVersion = 1

// Record is an accepted Start tap. It is a plan record, not handler progress: it says the player consented to
// this chunk, never how far the handler got (design 1.5).
type Record struct {
	Quest string    `json:"quest"`
	Chunk string    `json:"chunk"`
	Diff  string    `json:"difficulty"`
	At    time.Time `json:"at"`
	Mode  string    `json:"mode,omitempty"` // the mode that was active (train, idle, farm)
	Auto  bool      `json:"auto,omitempty"` // the board started it by itself (slice S8): the only chunk it may hand in itself
}

// SkipRecord is a player Skip. Chunk "" skips the whole quest.
type SkipRecord struct {
	Quest string    `json:"quest"`
	Chunk string    `json:"chunk,omitempty"`
	Diff  string    `json:"difficulty"`
	At    time.Time `json:"at"`
}

// AutoRecord is the opt-in to auto-start, per quest and difficulty (default off).
type AutoRecord struct {
	Quest string    `json:"quest"`
	Diff  string    `json:"difficulty"`
	Since time.Time `json:"since"`
}

// AutoEvent is one thing the auto-start controller did (slice S8): the rate limits count its starts and the "while you
// were away" card lists those with Away set. Kind is autoStart, autoHandIn or autoHandInFailed.
type AutoEvent struct {
	At    time.Time `json:"at"`
	Quest string    `json:"quest"`
	Chunk string    `json:"chunk"`
	Kind  string    `json:"kind"`
	Away  bool      `json:"away,omitempty"` // no watch client had been seen for quests.awayMin
	Note  string    `json:"note,omitempty"`
	// Awarded is the Ether the auto hand-in booked ("while you were away" shows it).
	Awarded []Pay `json:"awarded,omitempty"`
}

// Auto event kinds.
const (
	EvAutoStart        = "autoStart"
	EvAutoHandIn       = "autoHandIn"
	EvAutoHandInFailed = "autoHandInFailed"
	EvAutoResume       = "autoResume" // the pause policy lifted a park (resume.go)
)

// autoLogMax bounds the log: enough for the per-day cap and one away stretch.
const autoLogMax = 100

// Plan is the whole per-hero store: only the player's decisions, never derived progress.
type Plan struct {
	Accepted []Record     `json:"accepted,omitempty"`
	Skipped  []SkipRecord `json:"skipped,omitempty"`
	Auto     []AutoRecord `json:"autoStart,omitempty"`
	Seen     []string     `json:"seen,omitempty"` // payoff toasts already shown
	Mode     string       `json:"mode,omitempty"` // the between-chunks mode the player picked (S4 stores it, S6 acts on it)
	AutoLog  []AutoEvent  `json:"autoLog,omitempty"`
	AwaySeen time.Time    `json:"awaySeen,omitempty"` // the player dismissed the "while you were away" card at this time
	// Based lists the difficulties whose done quests were recorded when the board first saw the hero; Baseline holds
	// those award keys. A key in Baseline never pays (user decision 2026-11-16: no backfill for old saves).
	Based    []string `json:"based,omitempty"`
	Baseline []string `json:"baseline,omitempty"`
	// Runs and Latches are the lifecycle of the quest board state machine (sm.go, knob quests.boardSM). They are additive
	// inside file version 1: an older build ignores them and keeps working from Accepted, which stays written from the
	// live and finished runs for one release (store_sm.go).
	Runs    []Run   `json:"runs,omitempty"`
	Latches []Latch `json:"latches,omitempty"`
	// Legacy is LegacySchema once the retired Journey's decisions were imported (ImportLegacy).
	Legacy string `json:"legacy,omitempty"`
}

func (p Plan) based(diff int) bool {
	for _, d := range p.Based {
		if d == Difficulties[diff] {
			return true
		}
	}
	return false
}

func (p Plan) inBaseline(key string) bool {
	for _, k := range p.Baseline {
		if k == key {
			return true
		}
	}
	return false
}

// Modes are the between-chunks modes (user decision 2026-11-16); the default is Train.
var Modes = []string{"train", "idle", "farm"}

// DefaultMode is used while the player never picked one.
const DefaultMode = "train"

// ValidMode reports whether m is one of Modes.
func ValidMode(m string) bool {
	for _, x := range Modes {
		if x == m {
			return true
		}
	}
	return false
}

func (p Plan) accepted(diff int, quest, chunk string) bool {
	for _, r := range p.Accepted {
		if r.Quest == quest && r.Chunk == chunk && r.Diff == Difficulties[diff] {
			return true
		}
	}
	return false
}

func (p Plan) skipped(diff int, quest, chunk string) bool {
	for _, r := range p.Skipped {
		if r.Quest == quest && r.Diff == Difficulties[diff] && (r.Chunk == "" || r.Chunk == chunk) {
			return true
		}
	}
	return false
}

// touched reports whether the player ever started anything of the quest in this difficulty (the board ran it).
func (p Plan) touched(diff int, quest string) bool {
	for _, r := range p.Accepted {
		if r.Quest == quest && r.Diff == Difficulties[diff] {
			return true
		}
	}
	return false
}

// AutoStart reports the opt-in for a quest in a difficulty.
func (p Plan) AutoStart(diff int, quest string) bool {
	for _, r := range p.Auto {
		if r.Quest == quest && r.Diff == Difficulties[diff] {
			return true
		}
	}
	return false
}

// check rejects records that name an unknown quest, chunk or difficulty (an old record after a data change).
func (p Plan) check(b *Board) error {
	for _, r := range p.Accepted {
		if _, err := b.Chunk(r.Quest, r.Chunk); err != nil {
			return fmt.Errorf("plan record: %w", err)
		}
		if _, err := DiffIndex(r.Diff); err != nil {
			return fmt.Errorf("plan record: %w", err)
		}
	}
	for _, r := range p.Skipped {
		if r.Chunk == "" {
			if _, err := b.Quest(r.Quest); err != nil {
				return fmt.Errorf("plan skip: %w", err)
			}
		} else if _, err := b.Chunk(r.Quest, r.Chunk); err != nil {
			return fmt.Errorf("plan skip: %w", err)
		}
		if _, err := DiffIndex(r.Diff); err != nil {
			return fmt.Errorf("plan skip: %w", err)
		}
	}
	for _, r := range p.Auto {
		if _, err := b.Quest(r.Quest); err != nil {
			return fmt.Errorf("plan autoStart: %w", err)
		}
		if _, err := DiffIndex(r.Diff); err != nil {
			return fmt.Errorf("plan autoStart: %w", err)
		}
	}
	for _, e := range p.AutoLog {
		if _, err := b.Chunk(e.Quest, e.Chunk); err != nil {
			return fmt.Errorf("plan autoLog: %w", err)
		}
	}
	return nil
}

var heroName = regexp.MustCompile(`^[A-Za-z][A-Za-z0-9_-]{1,23}$`)

// Store keeps one hero's plan records in <dir>/<hero>.quests.json, written atomically with a .bak
// (internal/store.File). Calls are serialised.
type Store struct {
	b    *Board
	file store.File[Plan]
	now  func() time.Time
	mu   sync.Mutex
}

// Open returns the store of one hero. The hero name is a Diablo II character name (2 to 24 characters), so
// it can never escape dir.
func Open(b *Board, dir, hero string) (*Store, error) {
	if !heroName.MatchString(hero) {
		return nil, fmt.Errorf("questboard: bad hero name %q", hero)
	}
	return &Store{b: b, now: time.Now, file: store.File[Plan]{
		Path: filepath.Join(dir, hero+".quests.json"), Version: PlanVersion,
	}}, nil
}

// Load returns the records and what the load did (recovery from .bak is reported, a future version is an
// error). A record naming an id the data no longer has is an error too, naming it.
func (s *Store) Load() (Plan, store.Info, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.load()
}

func (s *Store) load() (Plan, store.Info, error) {
	p, info, err := s.file.Load()
	if err != nil {
		return Plan{}, info, err
	}
	return p, info, p.check(s.b)
}

func (s *Store) update(f func(*Plan) error) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	p, _, err := s.load()
	if err != nil {
		return err
	}
	if err := f(&p); err != nil {
		return err
	}
	return s.file.Save(p)
}

// Accept records the player's Start tap. Callers check CanStart against the derived views first; here the ids
// are validated and a second accepted chunk in the same difficulty is refused ("never two at once", design
// 2.6). Repeating the same tap is a no-op.
func (s *Store) Accept(quest, chunk string, diff int, mode string) error {
	return s.AcceptIf(quest, chunk, diff, mode, nil)
}

// AcceptIf is Accept where a record for which finished returns true (a chunk that is done, kept as the
// proof that the board ran it) no longer counts as running. A nil finished counts every record.
func (s *Store) AcceptIf(quest, chunk string, diff int, mode string, finished func(Record) bool) error {
	return s.accept(quest, chunk, diff, mode, finished, false)
}

// AcceptChain records several chunks of one quest in one save, in the order given (a chunk and the prerequisite
// steps it waits for). Chunks that already have a record are kept; any other unfinished record refuses.
func (s *Store) AcceptChain(quest string, chunks []string, diff int, mode string, finished func(Record) bool, events ...AutoEvent) error {
	if diff < 0 || diff > 2 {
		return fmt.Errorf("questboard: difficulty index %d out of range", diff)
	}
	for _, c := range chunks {
		if _, err := s.b.Chunk(quest, c); err != nil {
			return err
		}
	}
	return s.update(func(p *Plan) error {
		in := map[string]bool{}
		for _, c := range chunks {
			in[c] = true
		}
		have := map[string]bool{}
		for _, r := range p.Accepted {
			if r.Diff != Difficulties[diff] {
				continue
			}
			if r.Quest == quest && in[r.Chunk] {
				have[r.Chunk] = true
				continue
			}
			if finished != nil && finished(r) {
				continue
			}
			return fmt.Errorf("questboard: %s/%s is already accepted in %s", r.Quest, r.Chunk, r.Diff)
		}
		for _, c := range chunks {
			if !have[c] {
				p.Accepted = append(p.Accepted, Record{Quest: quest, Chunk: c, Diff: Difficulties[diff], At: s.now().UTC(), Mode: mode})
			}
		}
		for _, e := range events {
			p.addEvent(e)
		}
		return nil
	})
}

// AcceptAuto is AcceptIf for the auto-start controller: the record carries Auto, and the start event is written in
// the same save so a crash cannot leave a started chunk without its log line.
func (s *Store) AcceptAuto(quest, chunk string, diff int, mode string, finished func(Record) bool, ev AutoEvent) error {
	return s.accept(quest, chunk, diff, mode, finished, true, ev)
}

func (s *Store) accept(quest, chunk string, diff int, mode string, finished func(Record) bool, auto bool, ev ...AutoEvent) error {
	if diff < 0 || diff > 2 {
		return fmt.Errorf("questboard: difficulty index %d out of range", diff)
	}
	if _, err := s.b.Chunk(quest, chunk); err != nil {
		return err
	}
	return s.update(func(p *Plan) error {
		for _, r := range p.Accepted {
			if r.Diff != Difficulties[diff] {
				continue
			}
			if r.Quest == quest && r.Chunk == chunk {
				return nil
			}
			if finished != nil && finished(r) {
				continue
			}
			return fmt.Errorf("questboard: %s/%s is already accepted in %s", r.Quest, r.Chunk, r.Diff)
		}
		p.Accepted = append(p.Accepted, Record{Quest: quest, Chunk: chunk, Diff: Difficulties[diff], At: s.now().UTC(), Mode: mode, Auto: auto})
		for _, e := range ev {
			p.addEvent(e)
		}
		return nil
	})
}

// Cancel withdraws an accepted chunk ("Stop this quest", back to ready). Unknown ids are errors; cancelling a
// chunk that is not accepted is a no-op.
func (s *Store) Cancel(quest, chunk string, diff int) error {
	if _, err := s.b.Chunk(quest, chunk); err != nil {
		return err
	}
	return s.update(func(p *Plan) error {
		out := p.Accepted[:0]
		for _, r := range p.Accepted {
			if !(r.Quest == quest && r.Chunk == chunk && r.Diff == Difficulties[diff]) {
				out = append(out, r)
			}
		}
		p.Accepted = out
		return nil
	})
}

// Skip records a player Skip (chunk "" for the whole quest); Unskip removes it.
func (s *Store) Skip(quest, chunk string, diff int) error {
	if err := s.checkIDs(quest, chunk, diff); err != nil {
		return err
	}
	return s.update(func(p *Plan) error {
		for _, r := range p.Skipped {
			if r.Quest == quest && r.Chunk == chunk && r.Diff == Difficulties[diff] {
				return nil
			}
		}
		p.Skipped = append(p.Skipped, SkipRecord{quest, chunk, Difficulties[diff], s.now().UTC()})
		return nil
	})
}

func (s *Store) Unskip(quest, chunk string, diff int) error {
	if err := s.checkIDs(quest, chunk, diff); err != nil {
		return err
	}
	return s.update(func(p *Plan) error {
		out := p.Skipped[:0]
		for _, r := range p.Skipped {
			if !(r.Quest == quest && r.Chunk == chunk && r.Diff == Difficulties[diff]) {
				out = append(out, r)
			}
		}
		p.Skipped = out
		return nil
	})
}

// SetAutoStart turns the opt-in on or off for a quest in one difficulty. It refuses a quest whose data says
// autoStart never (design 1.1).
func (s *Store) SetAutoStart(quest string, diff int, on bool) error {
	q, err := s.b.Quest(quest)
	if err != nil {
		return err
	}
	if diff < 0 || diff > 2 {
		return fmt.Errorf("questboard: difficulty index %d out of range", diff)
	}
	if on && q.AutoStart != "allowed" {
		return fmt.Errorf("questboard: quest %q does not allow auto-start", quest)
	}
	return s.update(func(p *Plan) error {
		out := p.Auto[:0]
		for _, r := range p.Auto {
			if !(r.Quest == quest && r.Diff == Difficulties[diff]) {
				out = append(out, r)
			}
		}
		p.Auto = out
		if on {
			p.Auto = append(p.Auto, AutoRecord{quest, Difficulties[diff], s.now().UTC()})
		}
		return nil
	})
}

func (p *Plan) addEvent(e AutoEvent) {
	p.AutoLog = append(p.AutoLog, e)
	if n := len(p.AutoLog) - autoLogMax; n > 0 {
		p.AutoLog = append([]AutoEvent(nil), p.AutoLog[n:]...)
	}
}

// AddAutoEvent appends one controller event to the log.
func (s *Store) AddAutoEvent(e AutoEvent) error {
	return s.update(func(p *Plan) error { p.addEvent(e); return nil })
}

// DismissAway marks the "while you were away" card as read at t.
func (s *Store) DismissAway(t time.Time) error {
	return s.update(func(p *Plan) error { p.AwaySeen = t.UTC(); return nil })
}

// MarkSeen records that a payoff toast key was shown (so it is not shown again after a restart).
func (s *Store) MarkSeen(key string) error {
	return s.update(func(p *Plan) error {
		for _, k := range p.Seen {
			if k == key {
				return nil
			}
		}
		p.Seen = append(p.Seen, key)
		return nil
	})
}

// RecordBaseline stores the award keys of everything already done in the difficulty, once. A second call for the same
// difficulty changes nothing.
func (s *Store) RecordBaseline(diff int, keys []string) error {
	return s.update(func(p *Plan) error {
		if p.based(diff) {
			return nil
		}
		p.Based = append(p.Based, Difficulties[diff])
		for _, k := range keys {
			if !p.inBaseline(k) {
				p.Baseline = append(p.Baseline, k)
			}
		}
		return nil
	})
}

func (s *Store) checkIDs(quest, chunk string, diff int) error {
	if diff < 0 || diff > 2 {
		return fmt.Errorf("questboard: difficulty index %d out of range", diff)
	}
	if chunk == "" {
		_, err := s.b.Quest(quest)
		return err
	}
	_, err := s.b.Chunk(quest, chunk)
	return err
}

// Mode returns the stored mode, DefaultMode when none was picked.
func (s *Store) Mode() (string, error) {
	p, _, err := s.Load()
	if err != nil {
		return "", err
	}
	if p.Mode == "" {
		return DefaultMode, nil
	}
	return p.Mode, nil
}

// SetMode stores the mode; an unknown one is an error naming it. Setting the stored mode again is a no-op write.
func (s *Store) SetMode(m string) error {
	if !ValidMode(m) {
		return fmt.Errorf("questboard: unknown mode %q (train, idle or farm)", m)
	}
	return s.update(func(p *Plan) error { p.Mode = m; return nil })
}

// LegacySchema marks a plan file that already took in the retired Journey's decisions (slice S9).
const LegacySchema = "campaign/1"

// Legacy is what the board keeps of a hero's retired Journey state (state/campaign.json): the quests the player skipped
// (Intent skip on a chapter that names a quest) and whether the hero played "orders" mode. Derived progress (done
// chapters, asked questions, Go now, holds) is not imported: the board derives progress from the game's own quest
// bits, so a hero mid-Journey loses nothing and is simply offered the same next quest.
type Legacy struct {
	SkippedQuests []string
	OrdersMode    bool
}

// ImportLegacy takes the retired Journey's decisions into the plan once (the marker is stored in the plan, so a
// restart or a second call changes nothing). Quests the board has no data for are ignored. Never touches the
// Journey's own file. It returns how many skips were added.
func (s *Store) ImportLegacy(l Legacy, diff int) (added int, err error) {
	if diff < 0 || diff > 2 {
		return 0, fmt.Errorf("questboard: difficulty index %d out of range", diff)
	}
	err = s.update(func(p *Plan) error {
		if p.Legacy == LegacySchema {
			return nil
		}
		for _, name := range l.SkippedQuests {
			q := s.legacyQuest(name)
			if q == "" || p.skipped(diff, q, "") {
				continue
			}
			p.Skipped = append(p.Skipped, SkipRecord{Quest: q, Diff: Difficulties[diff], At: s.now().UTC()})
			added++
		}
		if l.OrdersMode && p.Mode == "" {
			p.Mode = "farm" // "orders": between the player's orders the hero farms its current safe zone
		}
		p.Legacy = LegacySchema
		return nil
	})
	return added, err
}

// legacyQuest resolves a quest name of the old Journey (a task-layer id such as sisters_burial) to the board's quest
// id ("" when the board has no such quest).
func (s *Store) legacyQuest(name string) string {
	for _, q := range s.b.Quests() {
		if q.ID == name || q.TaskID == name {
			return q.ID
		}
	}
	return ""
}
