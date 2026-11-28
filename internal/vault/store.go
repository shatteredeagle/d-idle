package vault

import (
	"bufio"
	"bytes"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"
)

// File names inside a hero's vault directory (docs/vault.md).
const (
	LogFile     = "log.jsonl"
	IndexFile   = "index.json"
	ArchiveFile = "archive.jsonl"
)

// event is one log line. Add carries the whole item; every other event names a record by id.
type event struct {
	Seq    int64     `json:"seq"`
	Ev     string    `json:"ev"`
	ID     string    `json:"id,omitempty"`
	T      time.Time `json:"t"`
	Item   *Item     `json:"item,omitempty"`
	Trim   string    `json:"trim,omitempty"` // add: the record this insert displaced (already archived)
	Reason string    `json:"reason,omitempty"`
	N      int       `json:"n,omitempty"`     // sold: gold
	Where  string    `json:"where,omitempty"` // keepq: the target, stash (default) or inv
	Stage  string    `json:"stage,omitempty"` // reveal and sale steps: the stage entered
}

// Options configure a Store. Zero values are usable except Capacity.
type Options struct {
	Capacity      func() int // read at every insert; nil means the table base
	Tiers         *Tiers
	Values        *ValueTable
	Now           func() time.Time
	NoSync        bool // skip fsync (benchmarks, tests); production leaves it off
	TrimProtected bool // with AutoTrim: allow trimming protected items when nothing else is left (needs a Needs-you confirm upstream)
	// AutoTrim reads loot.vaultAutoTrim, live at every insert. nil or false (the default, user decision 2026-11-21):
	// a full vault never removes anything and a new drop is NOT recorded (ErrFull; see Miss). true: the old behaviour,
	// an insert at capacity archives the least valuable unpinned record first.
	AutoTrim func() bool
}

// Store is one hero's vault. Safe for concurrent use.
type Store struct {
	dir  string
	hero string
	opt  Options

	mu       sync.Mutex
	f        *os.File
	size     int64
	lastLine [32]byte // sha256 of the last complete line (index check)
	seq      int64
	ver      uint64 // bumped by every state change (apply, snapshot load): read caches key on it
	recs     []*Rec
	byID     map[string]*Rec
	liveHash map[string]*Rec
	gens     map[string]int
	live     int
	pins     int
	ix       index
	missed   Missed
	anomaly  int  // log lines skipped on replay
	repaired bool // a torn tail was cut on open

	// test hook: called with the line about to be written; a non-nil error aborts before it reaches the file.
	// (A partial write is simulated by the hook returning the prefix to write plus an error.)
	hook func(line []byte) (write []byte, err error)
}

// Open opens (creating if needed) the vault in dir for hero. It loads the index snapshot when it matches the
// log, replays the rest, cuts a torn last line, and ends ready to append.
func Open(dir, hero string, opt Options) (*Store, error) {
	if err := ValidHero(hero); err != nil {
		return nil, err
	}
	var err error
	if opt.Tiers == nil {
		if opt.Tiers, err = DefaultTiers(); err != nil {
			return nil, err
		}
	}
	if opt.Values == nil {
		if opt.Values, err = DefaultValues(); err != nil {
			return nil, err
		}
	}
	if opt.Now == nil {
		opt.Now = time.Now
	}
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return nil, err
	}
	s := &Store{dir: dir, hero: hero, opt: opt}
	s.reset()
	if err := s.load(); err != nil {
		return nil, err
	}
	s.f, err = os.OpenFile(filepath.Join(dir, LogFile), os.O_WRONLY|os.O_APPEND|os.O_CREATE, 0o644)
	if err != nil {
		return nil, err
	}
	return s, nil
}

// HeroDir is the per-hero directory under root: root/<hero>/vault.
func HeroDir(root, hero string) string { return filepath.Join(root, hero, "vault") }

// OpenHero is Open(HeroDir(root, hero), hero, opt).
func OpenHero(root, hero string, opt Options) (*Store, error) {
	if err := ValidHero(hero); err != nil {
		return nil, err
	}
	return Open(HeroDir(root, hero), hero, opt)
}

// ValidHero rejects keys that could leave the store directory.
func ValidHero(h string) error {
	if h == "" || len(h) > 64 || h == "." || h == ".." || strings.ContainsAny(h, `/\:`+"\x00") {
		return fmt.Errorf("vault: bad hero key %q", h)
	}
	return nil
}

func (s *Store) reset() {
	s.recs, s.byID, s.liveHash, s.gens = nil, map[string]*Rec{}, map[string]*Rec{}, map[string]int{}
	s.live, s.pins, s.seq, s.anomaly = 0, 0, 0, 0
	s.missed = Missed{}
	s.ix = newIndex()
}

// Close writes the snapshot and closes the log.
func (s *Store) Close() error {
	s.mu.Lock()
	defer s.mu.Unlock()
	err := s.checkpointLocked()
	if cerr := s.f.Close(); err == nil {
		err = cerr
	}
	return err
}

// Hero is the key this store was opened for.
func (s *Store) Hero() string { return s.hero }

func (s *Store) capacity() int {
	if s.opt.Capacity != nil {
		return s.opt.Capacity()
	}
	return s.opt.Tiers.Base
}

// ---- appending ----

func (s *Store) appendEv(ev *event) error {
	s.seq++
	ev.Seq = s.seq
	b, err := json.Marshal(ev)
	if err != nil {
		s.seq--
		return err
	}
	line := append(b, '\n')
	w := line
	var herr error
	if s.hook != nil {
		w, herr = s.hook(line)
	}
	if len(w) > 0 {
		n, werr := s.f.Write(w)
		s.size += int64(n)
		if werr != nil && herr == nil {
			herr = werr
		}
	}
	if herr != nil {
		s.seq--
		return herr
	}
	if !s.opt.NoSync {
		if err := s.f.Sync(); err != nil {
			return err
		}
	}
	s.lastLine = sha256.Sum256(b)
	return nil
}

// commit appends ev and then applies it, so the in-memory state is never ahead of the log.
func (s *Store) commit(ev *event) error {
	ev.T = s.opt.Now().UTC()
	if err := s.appendEv(ev); err != nil {
		return err
	}
	if err := s.apply(ev); err != nil {
		return fmt.Errorf("vault: applying committed event: %w", err)
	}
	return nil
}

// ---- replay ----

func (s *Store) load() error {
	path := filepath.Join(s.dir, LogFile)
	fi, err := os.Stat(path)
	if os.IsNotExist(err) {
		return nil
	}
	if err != nil {
		return err
	}
	start := int64(0)
	if snap, ok := s.loadSnapshot(path, fi.Size()); ok {
		start = snap
	}
	fh, err := os.Open(path)
	if err != nil {
		return err
	}
	defer fh.Close()
	if _, err := fh.Seek(start, io.SeekStart); err != nil {
		return err
	}
	rd := bufio.NewReaderSize(fh, 1<<20)
	off := start
	for {
		line, err := rd.ReadBytes('\n')
		if len(line) > 0 && line[len(line)-1] != '\n' { // torn tail
			s.repaired = true
			break
		}
		if len(line) > 0 {
			body := bytes.TrimSuffix(line, []byte("\n"))
			var ev event
			if jerr := json.Unmarshal(body, &ev); jerr != nil || s.apply(&ev) != nil {
				s.anomaly++
			} else if ev.Seq > s.seq {
				s.seq = ev.Seq
			}
			s.lastLine = sha256.Sum256(body)
			off += int64(len(line))
		}
		if err != nil {
			break
		}
	}
	s.size = off
	if s.repaired {
		if err := os.Truncate(path, off); err != nil {
			return err
		}
	}
	return nil
}

// apply is the pure state machine: replay and live commits both go through it.
func (s *Store) apply(ev *event) error {
	s.ver++
	need := func(states ...State) (*Rec, error) {
		r := s.byID[ev.ID]
		if r == nil {
			return nil, ErrNotFound
		}
		for _, st := range states {
			if r.State == st {
				return r, nil
			}
		}
		return nil, fmt.Errorf("%w: %s is %s, event %s", ErrState, r.ID, r.State, ev.Ev)
	}
	switch ev.Ev {
	case "add":
		if ev.Item == nil {
			return ErrInvalid
		}
		it := *ev.Item
		h := hashOf(it.Raw)
		if s.liveHash[h] != nil {
			return ErrDuplicate
		}
		if ev.Trim != "" {
			v := s.byID[ev.Trim]
			if v == nil || !v.State.Live() {
				return fmt.Errorf("%w: trim target %s", ErrState, ev.Trim)
			}
			s.leave(v, Trimmed, ev.T)
			v.Reason = "trimmed"
		}
		r := &Rec{Item: it, Hash: h, State: Recorded, Updated: ev.T, seq: len(s.recs)}
		s.recs = append(s.recs, r)
		s.byID[r.ID] = r
		s.liveHash[h] = r
		s.gens[h]++
		s.live++
		s.ix.add(r)
	case "miss":
		if ev.Item == nil {
			return ErrInvalid
		}
		s.missed.add(*ev.Item, ev.T)
	case "missclear":
		s.missed = Missed{}
	case "pin":
		r, err := need(Recorded)
		if err != nil {
			return err
		}
		if !r.Pin {
			r.Pin = true
			s.pins++
		}
	case "unpin":
		r, err := need(Recorded)
		if err != nil {
			return err
		}
		if r.Pin {
			r.Pin = false
			s.pins--
		}
	case "keepq":
		r, err := need(Recorded)
		if err != nil {
			return err
		}
		r.State, r.Reason, r.Where, r.Updated = KeepQ, "", ev.Where, ev.T
	case "kbegin":
		r, err := need(KeepQ)
		if err != nil {
			return err
		}
		if r.InFlight {
			return fmt.Errorf("%w: %s create already in flight", ErrState, r.ID)
		}
		r.InFlight, r.Attempts, r.Updated = true, r.Attempts+1, ev.T
	case "kretry": // the create was not seen: queued again, no record of a create
		r, err := need(KeepQ)
		if err != nil {
			return err
		}
		r.InFlight, r.Reason, r.Updated = false, ev.Reason, ev.T
	case "kept":
		r, err := need(KeepQ)
		if err != nil {
			return err
		}
		r.InFlight = false
		if ev.Reason != "" { // where the item landed, e.g. "stash 3,5"
			r.Reason = ev.Reason
		}
		s.leave(r, Kept, ev.T)
	case "keepfail":
		r, err := need(KeepQ)
		if err != nil {
			return err
		}
		r.State, r.InFlight, r.Reason, r.Updated = Recorded, false, ev.Reason, ev.T
	case "keepcancel":
		r, err := need(KeepQ)
		if err != nil {
			return err
		}
		if r.InFlight {
			return fmt.Errorf("%w: %s create in flight", ErrState, r.ID)
		}
		r.State, r.Updated = Recorded, ev.T
	case "revq":
		r, err := need(Recorded, KeepQ)
		if err != nil {
			return err
		}
		r.RevQ, r.Reason, r.Updated = true, "", ev.T
	case "revcancel":
		r, err := need(Recorded, KeepQ)
		if err != nil {
			return err
		}
		r.RevQ, r.Updated = false, ev.T
	case "rstage": // reveal or sale step entered
		r, err := need(Recorded, KeepQ, SellQ)
		if err != nil {
			return err
		}
		r.Stage, r.Updated = ev.Stage, ev.T
		if ev.Stage == StageSellBegin {
			r.GoldPre, r.Attempts = ev.N, r.Attempts+1
		}
		if ev.Stage == StageRevBegin {
			r.Attempts++
		}
	case "rfail": // reveal ends without a result; the item stays as recorded
		r, err := need(Recorded, KeepQ)
		if err != nil {
			return err
		}
		r.RevQ, r.Stage, r.Reason, r.Updated = false, "", ev.Reason, ev.T
	case "reveal":
		r, err := need(Recorded, KeepQ)
		if err != nil {
			return err
		}
		if ev.Item == nil {
			return ErrInvalid
		}
		delete(s.liveHash, r.Hash)
		r.Raw, r.Name, r.Identified, r.Value = ev.Item.Raw, ev.Item.Name, true, ev.Item.Value
		r.Hash = hashOf(r.Raw)
		s.liveHash[r.Hash] = r
		r.Updated, r.RevQ, r.Reason = ev.T, false, ""
		if r.Stage == StageRevMade && r.State == Recorded {
			r.Stage = StageRevCleanup
		} else {
			r.Stage = ""
		}
	case "unseal": // the reveal was paid: the held affixes become the record's lines (the bytes stay as dropped)
		r, err := need(Recorded, KeepQ)
		if err != nil {
			return err
		}
		if r.Identified || r.Sealed == "" {
			return ErrState
		}
		r.Props, r.Sealed, r.Identified, r.Unlocked = r.Sealed, "", true, true
		r.Value = s.opt.Values.Of(&r.Item)
		r.RevQ, r.Reason, r.Updated = false, "", ev.T
	case "seal": // the affixes the item's own bytes carry, decoded by the fork (item decode): held, never shown until paid
		r, err := need(Recorded, KeepQ)
		if err != nil {
			return err
		}
		if r.Identified || r.Sealed != "" || ev.Reason == "" {
			return ErrState
		}
		r.Sealed, r.Updated = ev.Reason, ev.T
	case "props": // an identified record that carries no property text: the fork decoded its own bytes
		r, err := need(Recorded, KeepQ, SellQ)
		if err != nil {
			return err
		}
		if !r.Identified || r.Props != "" || ev.Reason == "" {
			return ErrState
		}
		r.Props, r.Updated = ev.Reason, ev.T
	case "reseal": // undo an unseal whose payment failed
		r, err := need(Recorded, KeepQ)
		if err != nil {
			return err
		}
		if !r.Unlocked {
			return ErrState
		}
		r.Sealed, r.Props, r.Identified, r.Unlocked = r.Props, "", false, false
		r.Value = s.opt.Values.Of(&r.Item)
		r.Updated = ev.T
	case "sellq":
		r, err := need(Recorded)
		if err != nil {
			return err
		}
		if r.Pin {
			return ErrPinned
		}
		r.State, r.Reason, r.Updated = SellQ, "", ev.T
		r.Ordered = ev.N == 1
	case "sellord":
		r, err := need(SellQ)
		if err != nil {
			return err
		}
		r.Ordered, r.Updated = true, ev.T
	case "sellfail", "sellcancel":
		r, err := need(SellQ)
		if err != nil {
			return err
		}
		r.State, r.Stage, r.GoldPre, r.Reason, r.Ordered, r.Updated = Recorded, "", 0, ev.Reason, false, ev.T
	case "sold":
		r, err := need(SellQ)
		if err != nil {
			return err
		}
		r.Gold, r.Stage, r.Reason = ev.N, "", ev.Reason
		s.leave(r, Sold, ev.T)
	case "soldnow": // the instant sale: the gold was credited to the stash by order ev.Reason
		r, err := need(Recorded, SellQ)
		if err != nil {
			return err
		}
		if r.Hold != "" && r.Hold != ev.Reason {
			return fmt.Errorf("%w: %w: %s by %s", ErrState, ErrHeld, r.ID, r.Hold)
		}
		r.Gold, r.Stage, r.Reason, r.Ordered, r.Hold = ev.N, "", ev.Reason, false, ""
		s.leave(r, Sold, ev.T)
	case "hold": // reserved by order ev.Reason before its gold is credited
		r, err := need(Recorded, SellQ)
		if err != nil {
			return err
		}
		if r.Pin || r.Stage != "" || r.InFlight || (r.Hold != "" && r.Hold != ev.Reason) || ev.Reason == "" {
			return fmt.Errorf("%w: %s cannot be reserved", ErrState, r.ID)
		}
		r.Hold, r.Updated = ev.Reason, ev.T
	case "release":
		r := s.byID[ev.ID]
		if r == nil {
			return ErrNotFound
		}
		r.Hold, r.Updated = "", ev.T
	case "discard":
		r, err := need(Recorded)
		if err != nil {
			return err
		}
		r.Reason = ev.Reason
		s.leave(r, Discarded, ev.T)
	default:
		return fmt.Errorf("vault: unknown event %q", ev.Ev)
	}
	return nil
}

// leave moves r to a final state and out of the live counters and the hash map.
func (s *Store) leave(r *Rec, to State, t time.Time) {
	if r.State.Live() {
		s.live--
	}
	if r.Pin {
		r.Pin = false
		s.pins--
	}
	if s.liveHash[r.Hash] == r {
		delete(s.liveHash, r.Hash)
	}
	r.State, r.Updated = to, t
}

func hashOf(raw string) string {
	b, err := hex.DecodeString(raw)
	if err != nil {
		b = []byte(raw)
	}
	sum := sha256.Sum256(b)
	return hex.EncodeToString(sum[:8])
}

// ---- public operations ----

// Add records a new drop. The ID is derived from the content (generation g > 0 when the same bytes were
// recorded and finished before). At exact capacity it returns ErrFull and removes nothing, unless Options.AutoTrim
// is on: then it trims one record first (design 2.4). Over capacity it always refuses; a live identical item is a duplicate.
func (s *Store) Add(it Item) (*Rec, error) {
	if err := s.check(&it); err != nil {
		return nil, err
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	h := hashOf(it.Raw)
	if s.liveHash[h] != nil {
		return nil, ErrDuplicate
	}
	it.Hero = s.hero
	it.ID = "v-" + h
	if g := s.gens[h]; g > 0 {
		it.ID += fmt.Sprintf("~%d", g)
	}
	if it.Value == 0 {
		it.Value = s.opt.Values.Of(&it)
	}
	if it.Found.Time.IsZero() {
		it.Found.Time = s.opt.Now().UTC()
	}
	capN := s.capacity()
	ev := &event{Ev: "add", ID: it.ID, Item: &it}
	switch {
	case s.live > capN:
		return nil, ErrOverCapacity
	case s.live == capN:
		if s.opt.AutoTrim == nil || !s.opt.AutoTrim() {
			return nil, ErrFull
		}
		v := s.trimCandidate()
		if v == nil {
			return nil, ErrFull
		}
		// Archive first: the record is never gone from disk before it is somewhere else.
		if err := s.archive(v, "trim"); err != nil {
			return nil, err
		}
		ev.Trim = v.ID
	}
	if err := s.commit(ev); err != nil {
		return nil, err
	}
	r := *s.byID[it.ID]
	return &r, nil
}

// Miss records a drop that could not be recorded because the vault is full (never an item removal). Only the facts
// the page lists are kept (name, code, quality, ilvl, value), not the bytes.
func (s *Store) Miss(it Item) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	m := Item{Name: it.Name, Code: it.Code, Quality: it.Quality, Ilvl: it.Ilvl, Value: it.Value, Found: it.Found}
	if m.Value == 0 {
		m.Value = s.opt.Values.Of(&it)
	}
	return s.commit(&event{Ev: "miss", Item: &m})
}

// ClearMissed resets the missed-drops counter and list (the player's dismiss button).
func (s *Store) ClearMissed() error {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.commit(&event{Ev: "missclear"})
}

// MissedDrops is a copy of the missed-drops counter.
func (s *Store) MissedDrops() Missed {
	s.mu.Lock()
	defer s.mu.Unlock()
	m := s.missed
	m.Best = append([]Item(nil), m.Best...)
	return m
}

func (s *Store) check(it *Item) error {
	raw, err := hex.DecodeString(it.Raw)
	if err != nil || len(raw) == 0 || len(raw) > 4096 {
		return fmt.Errorf("%w: raw must be 1..4096 bytes of hex", ErrInvalid)
	}
	if it.Code == "" || len(it.Code) > 8 || it.Quality < 0 || it.Quality > 9 || it.Ilvl < 0 || it.Ilvl > 110 ||
		it.Sockets < 0 || it.Sockets > 6 || len(it.Name) > 120 || len(it.Chain) > 8 {
		return fmt.Errorf("%w: code=%q q=%d ilvl=%d sockets=%d", ErrInvalid, it.Code, it.Quality, it.Ilvl, it.Sockets)
	}
	return nil
}

// trimCandidate: lowest value, then lowest ilvl, then oldest, among unpinned Recorded records; protected ones
// only when nothing else is left and Options.TrimProtected is set.
func (s *Store) trimCandidate() *Rec {
	var best, bestP *Rec
	less := func(a, b *Rec) bool {
		if a.Value != b.Value {
			return a.Value < b.Value
		}
		if a.Ilvl != b.Ilvl {
			return a.Ilvl < b.Ilvl
		}
		return a.seq < b.seq
	}
	for _, r := range s.recs {
		if r.State != Recorded || r.Pin || r.Hold != "" {
			continue
		}
		if s.opt.Values.Protected(&r.Item) {
			if bestP == nil || less(r, bestP) {
				bestP = r
			}
			continue
		}
		if best == nil || less(r, best) {
			best = r
		}
	}
	if best == nil && s.opt.TrimProtected {
		return bestP
	}
	return best
}

func (s *Store) op(id string, ev event) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.opLocked(id, ev)
}

func (s *Store) opLocked(id string, ev event) error {
	if s.byID[id] == nil {
		return ErrNotFound
	}
	ev.ID = id
	// Validate against the state machine on a dry copy of the rec before touching the log.
	if err := s.dry(&ev); err != nil {
		return err
	}
	return s.commit(&ev)
}

// dry checks that ev would apply, without applying it.
func (s *Store) dry(ev *event) error {
	r := s.byID[ev.ID]
	ok := func(states ...State) error {
		for _, st := range states {
			if r.State == st {
				return nil
			}
		}
		return fmt.Errorf("%w: %s is %s, event %s", ErrState, r.ID, r.State, ev.Ev)
	}
	// A reserved record moves only by its own order: sold by it, held again by it, or released.
	if r.Hold != "" && ev.Ev != "release" && !((ev.Ev == "soldnow" || ev.Ev == "hold") && ev.Reason == r.Hold) {
		return fmt.Errorf("%w: %w: %s by %s", ErrState, ErrHeld, r.ID, r.Hold)
	}
	switch ev.Ev {
	case "hold":
		if err := ok(Recorded, SellQ); err != nil {
			return err
		}
		switch {
		case ev.Reason == "":
			return ErrInvalid
		case r.Pin:
			return ErrPinned
		case r.Stage != "" || r.InFlight:
			return fmt.Errorf("%w: %s has a step in flight", ErrState, r.ID)
		}
	case "pin":
		if err := ok(Recorded); err != nil {
			return err
		}
		if !r.Pin && s.pins+1 > s.pinLimit() {
			return ErrPinCap
		}
	case "unpin":
		return ok(Recorded)
	case "keepq", "discard":
		if err := ok(Recorded); err != nil {
			return err
		}
		if r.Stage != "" { // a reveal has a copy in the hero: let it finish first
			return fmt.Errorf("%w: %s has a reveal in flight", ErrState, r.ID)
		}
	case "sellq":
		if err := ok(Recorded); err != nil {
			return err
		}
		if r.Pin {
			return ErrPinned
		}
		if r.Stage != "" {
			return fmt.Errorf("%w: %s has a reveal in flight", ErrState, r.ID)
		}
	case "revq", "revcancel", "rfail":
		return ok(Recorded, KeepQ)
	case "rstage":
		return ok(Recorded, KeepQ, SellQ)
	case "kbegin":
		if err := ok(KeepQ); err != nil {
			return err
		}
		if r.InFlight {
			return fmt.Errorf("%w: %s create already in flight", ErrState, r.ID)
		}
	case "kretry", "kept", "keepfail":
		return ok(KeepQ)
	case "keepcancel":
		if err := ok(KeepQ); err != nil {
			return err
		}
		if r.InFlight {
			return fmt.Errorf("%w: %s create in flight", ErrState, r.ID)
		}
	case "sellfail", "sellcancel", "sold":
		return ok(SellQ)
	case "soldnow":
		if err := ok(Recorded, SellQ); err != nil {
			return err
		}
		if r.Pin {
			return ErrPinned
		}
		if r.Stage != "" {
			return fmt.Errorf("%w: %s has a step in flight", ErrState, r.ID)
		}
	}
	return nil
}

func (s *Store) pinLimit() int {
	return max(s.capacity(), 0) * s.opt.Tiers.PinPercent / 100
}

func (s *Store) Pin(id string) error   { return s.op(id, event{Ev: "pin"}) }
func (s *Store) Unpin(id string) error { return s.op(id, event{Ev: "unpin"}) }

// Keep queues a record for creation at the next town visit.
func (s *Store) Keep(id string) error { return s.op(id, event{Ev: "keepq"}) }

// KeepTo queues a record for creation in where (WhereStash or WhereInv; "" is the stash).
func (s *Store) KeepTo(id, where string) error {
	if where != "" && where != WhereStash && where != WhereInv {
		return fmt.Errorf("%w: keep target %q", ErrInvalid, where)
	}
	return s.op(id, event{Ev: "keepq", Where: where})
}

// CompleteKeepAt marks the create confirmed and records where it landed (shown on the card).
func (s *Store) CompleteKeepAt(id, place string) error {
	return s.op(id, event{Ev: "kept", Reason: place})
}

// KeepQueued is the records waiting for a create (queued or in flight), oldest queued first (insertion order).
func (s *Store) KeepQueued() []Rec {
	s.mu.Lock()
	defer s.mu.Unlock()
	var out []Rec
	for _, r := range s.recs {
		if r.State == KeepQ {
			out = append(out, *r)
		}
	}
	return out
}

// CancelKeep returns a queued (not in flight) record to Recorded.
func (s *Store) CancelKeep(id string) error { return s.op(id, event{Ev: "keepcancel"}) }

// BeginKeep logs that the create is about to be sent. After a crash the record shows InFlight and Recover
// decides from the game state whether the create happened. Never call the game create before BeginKeep returns.
func (s *Store) BeginKeep(id string) error { return s.op(id, event{Ev: "kbegin"}) }

// CompleteKeep marks the create confirmed (Kept, final).
func (s *Store) CompleteKeep(id string) error { return s.op(id, event{Ev: "kept"}) }

// FailKeep ends a keep with a reason (no space, version, dead): the record goes back to Recorded.
func (s *Store) FailKeep(id, reason string) error {
	return s.op(id, event{Ev: "keepfail", Reason: reason})
}

// RetryKeep clears the in-flight mark when the create is known not to have happened; it stays queued.
func (s *Store) RetryKeep(id, reason string) error {
	return s.op(id, event{Ev: "kretry", Reason: reason})
}

// Reveal stores the identified item bytes (read back from the game after the real identify path).
func (s *Store) Reveal(id string, identified Item) error {
	if err := s.check(&identified); err != nil {
		return err
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	r := s.byID[id]
	if r == nil {
		return ErrNotFound
	}
	if r.State != Recorded && r.State != KeepQ {
		return fmt.Errorf("%w: %s is %s, event reveal", ErrState, id, r.State)
	}
	if r.Hold != "" {
		return fmt.Errorf("%w: %w: %s by %s", ErrState, ErrHeld, id, r.Hold)
	}
	h := hashOf(identified.Raw)
	if o := s.liveHash[h]; o != nil && o != r {
		return ErrDuplicate
	}
	if identified.Value == 0 {
		identified.Value = s.opt.Values.Of(&identified)
	}
	return s.commit(&event{Ev: "reveal", ID: id, Item: &identified})
}

// Unseal reveals an unidentified record from the affixes the vault already holds: no game step. It refuses an identified
// record (ErrState) and one that holds nothing (ErrNothingSealed). The caller charges for it afterwards (Reseal undoes it).
func (s *Store) Unseal(id string) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	r := s.byID[id]
	if r == nil {
		return ErrNotFound
	}
	if r.Identified {
		return fmt.Errorf("%w: %s is identified already", ErrState, id)
	}
	if r.Sealed == "" {
		return ErrNothingSealed
	}
	return s.opLocked(id, event{Ev: "unseal"})
}

// Seal stores the affixes decoded from an unidentified record's own bytes (`item decode`, read-only in the game) so a
// record that holds none can be revealed from the vault. props is "stat:param:value;..." as the fork words it. It refuses an
// identified record, one that holds affixes already, and an empty text (ErrState). Nothing changes in the game.
func (s *Store) Seal(id, props string) error {
	if props == "" {
		return ErrState
	}
	return s.op(id, event{Ev: "seal", Reason: props})
}

// SetProps gives an identified record with no property text the one decoded from its own bytes (`item decode`); ErrState
// when it has some, is not identified, or props is empty.
func (s *Store) SetProps(id, props string) error {
	if props == "" {
		return ErrState
	}
	return s.op(id, event{Ev: "props", Reason: props})
}

// Reseal takes an Unseal back (the Ether charge failed after it).
func (s *Store) Reseal(id string) error { return s.op(id, event{Ev: "reseal"}) }

// MarkSell queues a record for the sell errand; a pinned record is refused (ErrPinned).
func (s *Store) MarkSell(id string) error { return s.op(id, event{Ev: "sellq"}) }

// OrderSell queues a sale the player asked for (N=1 in the log). With loot.vaultAutoTrim off only ordered sales run;
// a record marked without an order is a flag the player confirms (ConfirmSell). Ordering an already marked record
// confirms it.
func (s *Store) OrderSell(id string) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	if r := s.byID[id]; r != nil && r.State == SellQ {
		if r.Ordered {
			return nil
		}
		return s.opLocked(id, event{Ev: "sellord"})
	}
	return s.opLocked(id, event{Ev: "sellq", N: 1})
}

// ConfirmSell turns marked-for-sale records into player orders; it returns how many it confirmed.
func (s *Store) ConfirmSell() (int, error) {
	n := 0
	for _, r := range s.SellMarked() {
		if err := s.OrderSell(r.ID); err != nil {
			return n, err
		}
		n++
	}
	return n, nil
}

// SellMarked lists sell-queued records the player has not ordered (flags from before the full-vault rule, or set
// without an order), oldest first.
func (s *Store) SellMarked() []Rec {
	var out []Rec
	for _, r := range s.SellQueued() {
		if !r.Ordered && r.Stage == "" {
			out = append(out, r)
		}
	}
	return out
}

// FailSell ends a sale with a reason; the record goes back to Recorded.
func (s *Store) FailSell(id, reason string) error {
	return s.op(id, event{Ev: "sellfail", Reason: reason})
}

// CancelSell takes a queued sale back (not while its create is in flight).
func (s *Store) CancelSell(id string) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	if r := s.byID[id]; r != nil && r.State == SellQ && r.Stage != "" {
		return fmt.Errorf("%w: %s sale in flight", ErrState, id)
	}
	return s.opLocked(id, event{Ev: "sellcancel"})
}

// Sold ends a sale with the gold the read-back saw.
func (s *Store) Sold(id string, gold int) error { return s.op(id, event{Ev: "sold", N: gold}) }

// SoldNote is Sold with a note (an unconfirmed gold read-back) kept in the record.
func (s *Store) SoldNote(id string, gold int, note string) error {
	return s.op(id, event{Ev: "sold", N: gold, Reason: note})
}

// RevealWant queues a reveal (identify) for an unidentified record in Recorded or KeepQ. Idempotent callers check RevQ.
func (s *Store) RevealWant(id string) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	r := s.byID[id]
	if r == nil {
		return ErrNotFound
	}
	if r.Identified {
		return fmt.Errorf("%w: %s is identified already", ErrState, id)
	}
	return s.opLocked(id, event{Ev: "revq"})
}

// RevealCancel takes a queued reveal back while no step of it ran.
func (s *Store) RevealCancel(id string) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	if r := s.byID[id]; r != nil && r.Stage != "" {
		return fmt.Errorf("%w: %s reveal in flight", ErrState, id)
	}
	return s.opLocked(id, event{Ev: "revcancel"})
}

// RevealFail ends a reveal with a reason; the item stays as it was recorded.
func (s *Store) RevealFail(id, reason string) error {
	return s.op(id, event{Ev: "rfail", Reason: reason})
}

// SetStage logs the step a reveal or sale is entering (Stage* constants; "" clears it). For StageSellBegin gold is the
// purse before the create. Log it BEFORE the game step.
func (s *Store) SetStage(id, stage string, gold int) error {
	return s.op(id, event{Ev: "rstage", Stage: stage, N: gold})
}

// RevealQueued lists records with a reveal wanted, oldest first: pure reveals (Recorded) and keep-combined (KeepQ).
func (s *Store) RevealQueued() []Rec {
	s.mu.Lock()
	defer s.mu.Unlock()
	var out []Rec
	for _, r := range s.recs {
		if (r.RevQ || r.Stage == StageRevCleanup) && (r.State == Recorded || r.State == KeepQ) {
			out = append(out, *r)
		}
	}
	return out
}

// SellQueued lists records marked for the sell errand, oldest first.
func (s *Store) SellQueued() []Rec {
	s.mu.Lock()
	defer s.mu.Unlock()
	var out []Rec
	for _, r := range s.recs {
		if r.State == SellQ {
			out = append(out, *r)
		}
	}
	return out
}

// Discard archives the record, then marks it discarded. It never deletes in the same step: the archive line is
// durable before the log line, and the archive is purged only by PurgeArchive.
func (s *Store) Discard(id, reason string) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	r := s.byID[id]
	if r == nil {
		return ErrNotFound
	}
	if err := s.dry(&event{Ev: "discard", ID: id}); err != nil {
		return err
	}
	if err := s.archive(r, "discard: "+reason); err != nil {
		return err
	}
	return s.commit(&event{Ev: "discard", ID: id, Reason: reason})
}

// SellNow ends a sale the player ordered with the gold already credited to the stash by order `order`: the record is archived
// first (the archive line is durable before the log line, as Discard), then marked sold with the gold. Pinned records and a
// record with a step in flight are refused. Recorded and queued (sellq) records are both accepted.
func (s *Store) SellNow(id string, gold int, order string) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	r := s.byID[id]
	if r == nil {
		return ErrNotFound
	}
	if err := s.dry(&event{Ev: "soldnow", ID: id, Reason: order}); err != nil {
		return err
	}
	if err := s.archive(r, fmt.Sprintf("sold: %d gold, order %s", gold, order)); err != nil {
		return err
	}
	return s.commit(&event{Ev: "soldnow", ID: id, N: gold, Reason: order})
}

// SoldLine is one record of a batch sale: the id and the gold the engine paid for it.
type SoldLine struct {
	ID   string
	Gold int
}

// SellNowBatch is SellNow for many records under one lock with ONE archive write, ONE log write and ONE fsync each (a
// per-record SellNow costs two fsyncs a record: seconds for a whole vault). A record that cannot be sold is skipped and
// counted in failed; err is the last such error. The order is the same as SellNow: archive first, then the log, then the
// in-memory state, so a crash in between leaves records that the same order id sells again.
func (s *Store) SellNowBatch(lines []SoldLine, order string) (done int, failed int, err error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	var evs []*event
	var arch []byte
	seen := map[string]bool{}
	for _, l := range lines {
		r := s.byID[l.ID]
		if r == nil || seen[l.ID] {
			failed++
			err = ErrNotFound
			continue
		}
		seen[l.ID] = true
		if e := s.dry(&event{Ev: "soldnow", ID: l.ID, Reason: order}); e != nil {
			failed++
			err = e
			continue
		}
		b, e := json.Marshal(archived{At: s.opt.Now().UTC(), Reason: fmt.Sprintf("sold: %d gold, order %s", l.Gold, order), Rec: *r})
		if e != nil {
			return 0, failed, e
		}
		arch = append(append(arch, b...), '\n')
		evs = append(evs, &event{Ev: "soldnow", ID: l.ID, N: l.Gold, Reason: order})
	}
	if len(evs) == 0 {
		return 0, failed, err
	}
	if e := s.appendArchive(arch); e != nil {
		return 0, failed, e
	}
	if e := s.commitBatch(evs); e != nil {
		return 0, failed, e
	}
	return len(evs), failed, err
}

// Reserve is the compare-and-set that starts an order on records: every id becomes held by order in ONE logged batch, or
// none does. ver is the Version the caller read the records at (0: no check); a store that changed since answers ErrStale
// and the caller reads again. A record that is pinned, not in the vault, busy with a step or held by another order refuses
// the whole reservation (ErrHeld, ErrPinned, ErrState, ErrNotFound; bad names the first such id). A held record leaves
// only through SellNow/SellNowBatch of the same order or Release, so no Keep, Pin, Discard, queue Sell or Reveal can take
// it between the reservation and the sale. Reserving again with the same order is a no-op for records it already holds.
func (s *Store) Reserve(ids []string, order string, ver uint64) (bad string, err error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if order == "" {
		return "", ErrInvalid
	}
	if ver != 0 && ver != s.ver {
		return "", ErrStale
	}
	var evs []*event
	seen := map[string]bool{}
	for _, id := range ids {
		r := s.byID[id]
		if r == nil {
			return id, ErrNotFound
		}
		if seen[id] || r.Hold == order {
			continue
		}
		seen[id] = true
		ev := &event{Ev: "hold", ID: id, Reason: order}
		if e := s.dry(ev); e != nil {
			return id, e
		}
		evs = append(evs, ev)
	}
	if len(evs) == 0 {
		return "", nil
	}
	if e := s.commitBatch(evs); e != nil {
		// A short write held only some: take those back so the reservation stays all or none (best effort; the log is
		// the truth and a restart shows the same state).
		s.releaseLocked(order)
		return "", e
	}
	return "", nil
}

// Release frees every record order holds (the order failed before it paid) and returns how many it freed.
func (s *Store) Release(order string) (int, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.releaseLocked(order)
}

func (s *Store) releaseLocked(order string) (int, error) {
	var evs []*event
	for _, r := range s.recs {
		if order != "" && r.Hold == order {
			evs = append(evs, &event{Ev: "release", ID: r.ID, Reason: order})
		}
	}
	if len(evs) == 0 {
		return 0, nil
	}
	return len(evs), s.commitBatch(evs)
}

// Held lists the records an order holds, by order id, in insertion order.
func (s *Store) Held() map[string][]string {
	s.mu.Lock()
	defer s.mu.Unlock()
	out := map[string][]string{}
	for _, r := range s.recs {
		if r.Hold != "" {
			out[r.Hold] = append(out[r.Hold], r.ID)
		}
	}
	return out
}

func (s *Store) appendArchive(b []byte) error {
	fh, err := os.OpenFile(filepath.Join(s.dir, ArchiveFile), os.O_WRONLY|os.O_APPEND|os.O_CREATE, 0o644)
	if err != nil {
		return err
	}
	_, err = fh.Write(b)
	if err == nil && !s.opt.NoSync {
		err = fh.Sync()
	}
	if cerr := fh.Close(); err == nil {
		err = cerr
	}
	return err
}

// commitBatch is commit for several events: one fsync for the lot, then each is applied. On a write error the events written
// so far stay in the log and are applied (the log is the truth), the rest are dropped.
func (s *Store) commitBatch(evs []*event) error {
	var werr error
	n := 0
	for _, ev := range evs {
		ev.T = s.opt.Now().UTC()
		s.seq++
		ev.Seq = s.seq
		b, err := json.Marshal(ev)
		if err == nil {
			w := append(b, '\n')
			var herr error
			if s.hook != nil {
				w, herr = s.hook(w)
			}
			if len(w) > 0 {
				k, e := s.f.Write(w)
				s.size += int64(k)
				if e != nil && herr == nil {
					herr = e
				}
			}
			err = herr
		}
		if err != nil {
			s.seq--
			werr = err
			break
		}
		s.lastLine = sha256.Sum256(b)
		n++
	}
	if n > 0 && !s.opt.NoSync {
		if err := s.f.Sync(); err != nil && werr == nil {
			werr = err
		}
	}
	for _, ev := range evs[:n] {
		if err := s.apply(ev); err != nil {
			return fmt.Errorf("vault: applying committed event: %w", err)
		}
	}
	return werr
}

// Recover resolves every InFlight keep after a restart. present reports whether the item (matched by Raw hash
// in the stash) exists in the game: true completes the keep, false clears the mark so it is queued again. An
// error leaves the record in flight. It returns the ids it completed and requeued.
func (s *Store) Recover(present func(Rec) (bool, error)) (done, requeued []string, err error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	for _, r := range append([]*Rec(nil), s.recs...) {
		if r.State != KeepQ || !r.InFlight {
			continue
		}
		ok, perr := present(*r)
		if perr != nil {
			return done, requeued, perr
		}
		if ok {
			err = s.commit(&event{Ev: "kept", ID: r.ID})
			done = append(done, r.ID)
		} else {
			err = s.commit(&event{Ev: "kretry", ID: r.ID, Reason: "create not seen after restart"})
			requeued = append(requeued, r.ID)
		}
		if err != nil {
			return done, requeued, err
		}
	}
	return done, requeued, nil
}

// ---- reads ----

// Get returns a copy of a record.
func (s *Store) Get(id string) (Rec, bool) {
	s.mu.Lock()
	defer s.mu.Unlock()
	r := s.byID[id]
	if r == nil {
		return Rec{}, false
	}
	return *r, true
}

// Status is the capacity meter and health line.
type Status struct {
	Count, Capacity, Pins, PinLimit int
	Over, Full                      bool
	Anomalies                       int
	Repaired                        bool
}

func (s *Store) Status() Status {
	s.mu.Lock()
	defer s.mu.Unlock()
	c := s.capacity()
	return Status{Count: s.live, Capacity: c, Pins: s.pins, PinLimit: s.pinLimit(),
		Over: s.live > c, Full: s.live == c, Anomalies: s.anomaly, Repaired: s.repaired}
}

// Version changes whenever the records, the index or the missed-drop list change; equal versions mean equal read results.
func (s *Store) Version() uint64 {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.ver
}

// Scan calls fn for every record in insertion order until fn returns false, without the copy Dump makes. fn runs under the
// store lock: it must not call the Store and must not keep or change the record.
func (s *Store) Scan(fn func(r *Rec) bool) {
	s.mu.Lock()
	defer s.mu.Unlock()
	for _, r := range s.recs {
		if !fn(r) {
			return
		}
	}
}

// Dump is every record in insertion order (tests, export of history).
func (s *Store) Dump() []Rec {
	s.mu.Lock()
	defer s.mu.Unlock()
	out := make([]Rec, len(s.recs))
	for i, r := range s.recs {
		out[i] = *r
	}
	return out
}

// ---- archive ----

type archived struct {
	At     time.Time `json:"at"`
	Reason string    `json:"reason"`
	Rec    Rec       `json:"rec"`
}

func (s *Store) archive(r *Rec, reason string) error {
	b, err := json.Marshal(archived{At: s.opt.Now().UTC(), Reason: reason, Rec: *r})
	if err != nil {
		return err
	}
	return s.appendArchive(append(b, '\n'))
}

// Archived lists the archive newest last; a record archived twice (a retry after a crash) appears once, the
// latest entry winning.
func (s *Store) Archived() ([]Rec, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	return readArchive(filepath.Join(s.dir, ArchiveFile), time.Time{}, false)
}

func readArchive(path string, before time.Time, keepNewer bool) ([]Rec, error) {
	fh, err := os.Open(path)
	if os.IsNotExist(err) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	defer fh.Close()
	var out []Rec
	idx := map[string]int{}
	sc := bufio.NewScanner(fh)
	sc.Buffer(make([]byte, 1<<20), 1<<24)
	for sc.Scan() {
		var a archived
		if json.Unmarshal(sc.Bytes(), &a) != nil {
			continue
		}
		if keepNewer && a.At.Before(before) {
			continue
		}
		if i, ok := idx[a.Rec.ID]; ok {
			out[i] = a.Rec
			continue
		}
		idx[a.Rec.ID] = len(out)
		out = append(out, a.Rec)
	}
	return out, sc.Err()
}

// PurgeArchive drops archive entries older than before (design: trimmed and discarded items stay recoverable
// for 14 days). It is the only place anything is deleted from disk, it touches only the archive, and it
// rewrites through a temp file and a rename.
func (s *Store) PurgeArchive(before time.Time) (removed int, err error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	path := filepath.Join(s.dir, ArchiveFile)
	raw, err := os.ReadFile(path)
	if os.IsNotExist(err) {
		return 0, nil
	}
	if err != nil {
		return 0, err
	}
	var keep bytes.Buffer
	for _, line := range bytes.Split(raw, []byte("\n")) {
		if len(line) == 0 {
			continue
		}
		var a archived
		if json.Unmarshal(line, &a) == nil && a.At.Before(before) {
			removed++
			continue
		}
		keep.Write(line)
		keep.WriteByte('\n')
	}
	return removed, writeAtomic(path, keep.Bytes(), !s.opt.NoSync)
}

func writeAtomic(path string, b []byte, sync bool) error {
	tmp := path + ".tmp"
	fh, err := os.Create(tmp)
	if err != nil {
		return err
	}
	_, err = fh.Write(b)
	if err == nil && sync {
		err = fh.Sync()
	}
	if cerr := fh.Close(); err == nil {
		err = cerr
	}
	if err != nil {
		os.Remove(tmp)
		return err
	}
	return os.Rename(tmp, path)
}

// ---- index snapshot ----

type snapshot struct {
	Version  int            `json:"version"`
	Hero     string         `json:"hero"`
	Offset   int64          `json:"offset"`
	LastLine string         `json:"lastLine"` // hex sha256 of the last line before Offset
	Seq      int64          `json:"seq"`
	Anomaly  int            `json:"anomaly"`
	Gens     map[string]int `json:"gens"`
	Recs     []*Rec         `json:"recs"`
	Missed   *Missed        `json:"missed,omitempty"`
}

// Checkpoint writes the index snapshot (also done by Close). The snapshot is only a cache: delete it and the
// next Open replays the whole log to the same state.
func (s *Store) Checkpoint() error {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.checkpointLocked()
}

func (s *Store) checkpointLocked() error {
	sn := snapshot{Version: 1, Hero: s.hero, Offset: s.size, LastLine: hex.EncodeToString(s.lastLine[:]),
		Seq: s.seq, Anomaly: s.anomaly, Gens: s.gens, Recs: s.recs, Missed: &s.missed}
	b, err := json.Marshal(sn)
	if err != nil {
		return err
	}
	return writeAtomic(filepath.Join(s.dir, IndexFile), b, !s.opt.NoSync)
}

// loadSnapshot returns the log offset to continue replay from, or ok=false (state reset) when the snapshot is
// missing, damaged, for another hero, or does not match the log bytes it claims to cover.
func (s *Store) loadSnapshot(logPath string, logSize int64) (int64, bool) {
	b, err := os.ReadFile(filepath.Join(s.dir, IndexFile))
	if err != nil {
		return 0, false
	}
	var sn snapshot
	if json.Unmarshal(b, &sn) != nil || sn.Version != 1 || sn.Hero != s.hero || sn.Offset <= 0 || sn.Offset > logSize {
		return 0, false
	}
	fh, err := os.Open(logPath)
	if err != nil {
		return 0, false
	}
	defer fh.Close()
	// Find the start of the last covered line: scan back from Offset-1 for the previous newline.
	back := int64(1 << 20)
	from := max(sn.Offset-back, 0)
	buf := make([]byte, sn.Offset-from)
	if _, err := fh.ReadAt(buf, from); err != nil || buf[len(buf)-1] != '\n' {
		return 0, false
	}
	body := buf[:len(buf)-1]
	i := bytes.LastIndexByte(body, '\n')
	if i < 0 && from > 0 {
		return 0, false
	}
	sum := sha256.Sum256(body[i+1:])
	if hex.EncodeToString(sum[:]) != sn.LastLine {
		return 0, false
	}
	s.reset()
	if lb, err := hex.DecodeString(sn.LastLine); err == nil {
		copy(s.lastLine[:], lb)
	}
	s.gens = sn.Gens
	if s.gens == nil {
		s.gens = map[string]int{}
	}
	for _, r := range sn.Recs {
		r.seq = len(s.recs)
		s.recs = append(s.recs, r)
		s.byID[r.ID] = r
		if r.State.Live() {
			s.live++
			s.liveHash[r.Hash] = r
		}
		if r.Pin {
			s.pins++
		}
		s.ix.add(r)
	}
	s.seq, s.anomaly = sn.Seq, sn.Anomaly
	if sn.Missed != nil {
		s.missed = *sn.Missed
	}
	return sn.Offset, true
}
