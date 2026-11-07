package currency

import (
	"bufio"
	"bytes"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"sync"
	"sync/atomic"
	"time"
)

// Entry kinds. Only commit, credit, grant and spend change a balance; pending, void and refused are the audit trail
// of a bank action.
const (
	KindPending = "pending" // write-ahead record of a bank action; closed by commit, void or refused
	KindCommit  = "commit"  // the observed outcome of a bank action: credits Amount
	KindCredit  = "credit"  // a plain source credit
	KindGrant   = "grant"   // a one-off grant (tier unlock, first rule)
	KindSpend   = "spend"   // pays Amount for an upgrade
	KindVoid    = "void"    // closes a pending whose pack did not change
	KindRefused = "refused" // closes a pending whose credit was refused after the fact (cheated)

	VanillaMode = "vanilla"
	genesis     = "genesis"
)

var (
	ErrVanilla         = errors.New("currency: refused, vanilla mode has no currencies")
	ErrCheated         = errors.New("currency: refused, the run has gameplay writes (cheated)")
	ErrUnknownCurrency = errors.New("currency: unknown currency")
	ErrUnknownRecipe   = errors.New("currency: unknown bank recipe")
	ErrUnknownSource   = errors.New("currency: unknown source")
	ErrNotLedger       = errors.New("currency: not a ledger currency")
	ErrDuplicate       = errors.New("currency: key already booked")
	ErrInsufficient    = errors.New("currency: balance too low")
	ErrCorrupt         = errors.New("currency: ledger corrupt")
	ErrNothingToBank   = errors.New("currency: no marks in the pack to bank")
)

// Entry is one ledger line. The field order is the canonical JSON order (encoding/json writes struct fields in
// order), so the line bytes are what the next entry hashes.
type Entry struct {
	Seq       int    `json:"seq"`
	TS        string `json:"ts"`
	Prev      string `json:"prev"`
	Kind      string `json:"kind"`
	Currency  string `json:"currency,omitempty"`
	Amount    int64  `json:"amount,omitempty"`
	Sealed    int64  `json:"sealed,omitempty"` // part of Amount above the daily soft cap: booked, not spendable
	Source    string `json:"source,omitempty"`
	Key       string `json:"key,omitempty"` // idempotency key of the observation or grant
	Ref       int    `json:"ref,omitempty"` // seq of the pending this entry closes
	Mode      string `json:"mode"`
	Instance  string `json:"instance,omitempty"`
	Character string `json:"character,omitempty"`
	Run       string `json:"run,omitempty"`
	Recipe    string `json:"recipe,omitempty"`
	Before    *int   `json:"before,omitempty"` // pack count before / after the bank action
	After     *int   `json:"after,omitempty"`
	Upgrade   string `json:"upgrade,omitempty"`
	Level     int    `json:"level,omitempty"` // level reached by a spend
	Note      string `json:"note,omitempty"`
}

// Options configure a Ledger.
type Options struct {
	Registry *Registry        // default: the shipped registry
	Mode     string           // overlay.Mode value; "" or "vanilla" refuses every write
	Instance string           // stamped on entries
	Now      func() time.Time // default time.Now
}

// Ledger is the append-only, hash-chained account ledger of one profile: <dir>/ledger.jsonl, a head file
// <dir>/ledger.head (seq and hash of the last entry, so a cut tail is detected) and <dir>/ledger.lock.
type Ledger struct {
	dir  string
	opt  Options
	mu   sync.Mutex   // flock is per open file description; this keeps goroutines of one process apart too
	torn atomic.Int64 // bytes of an unfinished last line dropped by the most recent load
}

// Open prepares the directory. Nothing is written until the first entry.
func Open(dir string, opt Options) (*Ledger, error) {
	if opt.Registry == nil {
		opt.Registry = Default()
	}
	if opt.Now == nil {
		opt.Now = time.Now
	}
	if opt.Mode == "" {
		opt.Mode = VanillaMode
	}
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return nil, err
	}
	return &Ledger{dir: dir, opt: opt}, nil
}

func (l *Ledger) path(name string) string { return filepath.Join(l.dir, name) }

// Dir is the ledger directory.
func (l *Ledger) Dir() string { return l.dir }

// Registry returns the registry the ledger validates against.
func (l *Ledger) Registry() *Registry { return l.opt.Registry }

// Mode is the stamp entries carry.
func (l *Ledger) Mode() string { return l.opt.Mode }

func (l *Ledger) vanilla() bool { return l.opt.Mode == VanillaMode }

func hashLine(b []byte) string { s := sha256.Sum256(b); return hex.EncodeToString(s[:]) }

type head struct {
	Seq  int    `json:"seq"`
	Hash string `json:"hash"`
}

// state is the ledger replayed.
type state struct {
	entries []Entry
	lines   [][]byte
}

// load reads, parses and verifies the file; with repair it drops an unfinished last line (a crash mid-write).
// The caller holds the lock.
func (l *Ledger) load(repair bool) (*state, error) {
	b, err := os.ReadFile(l.path("ledger.jsonl"))
	if errors.Is(err, os.ErrNotExist) {
		b = nil
	} else if err != nil {
		return nil, err
	}
	l.torn.Store(0)
	if len(b) > 0 && b[len(b)-1] != '\n' {
		cut := bytes.LastIndexByte(b, '\n') + 1
		l.torn.Store(int64(len(b) - cut))
		if !repair {
			return nil, fmt.Errorf("%w: unfinished last line (%d bytes)", ErrCorrupt, len(b)-cut)
		}
		if err := os.Truncate(l.path("ledger.jsonl"), int64(cut)); err != nil {
			return nil, err
		}
		b = b[:cut]
	}
	st := &state{}
	prev := genesis
	sc := bufio.NewScanner(bytes.NewReader(b))
	sc.Buffer(make([]byte, 0, 64*1024), 1<<20)
	for sc.Scan() {
		line := append([]byte(nil), sc.Bytes()...)
		var e Entry
		if err := json.Unmarshal(line, &e); err != nil {
			return nil, fmt.Errorf("%w: line %d: %v", ErrCorrupt, len(st.entries)+1, err)
		}
		if e.Seq != len(st.entries)+1 {
			return nil, fmt.Errorf("%w: line %d has seq %d", ErrCorrupt, len(st.entries)+1, e.Seq)
		}
		if e.Prev != prev {
			return nil, fmt.Errorf("%w: seq %d does not chain to the entry before it", ErrCorrupt, e.Seq)
		}
		if re, _ := json.Marshal(e); !bytes.Equal(re, line) { // an edit that kept the JSON valid but not canonical
			return nil, fmt.Errorf("%w: seq %d is not in canonical form", ErrCorrupt, e.Seq)
		}
		prev = hashLine(line)
		st.entries = append(st.entries, e)
		st.lines = append(st.lines, line)
	}
	if err := sc.Err(); err != nil {
		return nil, err
	}
	if hb, err := os.ReadFile(l.path("ledger.head")); err == nil {
		var h head
		if json.Unmarshal(hb, &h) != nil {
			return nil, fmt.Errorf("%w: ledger.head unreadable", ErrCorrupt)
		}
		if h.Seq > len(st.entries) {
			return nil, fmt.Errorf("%w: truncated, the head names seq %d and the file ends at %d", ErrCorrupt, h.Seq, len(st.entries))
		}
		if h.Seq > 0 && hashLine(st.lines[h.Seq-1]) != h.Hash {
			return nil, fmt.Errorf("%w: seq %d was rewritten (head hash differs)", ErrCorrupt, h.Seq)
		}
	} else if !errors.Is(err, os.ErrNotExist) {
		return nil, err
	} else if len(st.entries) > 0 {
		return nil, fmt.Errorf("%w: ledger.head is missing", ErrCorrupt)
	}
	return st, nil
}

func (l *Ledger) lock() (func(), error) {
	l.mu.Lock()
	rel, err := lockFile(l.path("ledger.lock"))
	if err != nil {
		l.mu.Unlock()
		return nil, err
	}
	return func() { rel(); l.mu.Unlock() }, nil
}

// Verify reads the whole ledger and checks the chain, the canonical form of every line, the head file and the tail.
// A torn last line is reported, not repaired (Append repairs it).
func (l *Ledger) Verify() error {
	un, err := l.lock()
	if err != nil {
		return err
	}
	defer un()
	_, err = l.load(false)
	return err
}

// Entries returns a verified copy of the log.
func (l *Ledger) Entries() ([]Entry, error) {
	un, err := l.lock()
	if err != nil {
		return nil, err
	}
	defer un()
	st, err := l.load(false)
	if err != nil {
		return nil, err
	}
	return st.entries, nil
}

// ledgerCurrency checks that id is a ledger-stored currency.
func (l *Ledger) ledgerCurrency(id string) (*Currency, error) {
	c, err := l.opt.Registry.Get(id)
	if err != nil {
		return nil, err
	}
	if c.Storage != StorageLedger {
		return nil, fmt.Errorf("%w: %q is stored as %s", ErrNotLedger, id, c.Storage)
	}
	return c, nil
}

// day is the UTC day of an entry timestamp.
func day(ts string) string { return ts[:10] }

func (st *state) closed(seq int) bool {
	for _, e := range st.entries {
		if e.Ref == seq && e.Kind != KindPending {
			return true
		}
	}
	return false
}

// byKey finds the entry that booked key (commit, credit, grant, spend, void or refused).
func (st *state) byKey(key string) (Entry, bool) {
	if key == "" {
		return Entry{}, false
	}
	for _, e := range st.entries {
		if e.Key == key && e.Kind != KindPending {
			return e, true
		}
	}
	return Entry{}, false
}

func (st *state) pendingByKey(key string) (Entry, bool) {
	for _, e := range st.entries {
		if e.Kind == KindPending && e.Key == key && !st.closed(e.Seq) {
			return e, true
		}
	}
	return Entry{}, false
}

// creditedToday sums the credited amounts (sealed part included) of one source on one UTC day.
func (st *state) creditedToday(source, d string) int64 {
	var n int64
	for _, e := range st.entries {
		if e.Source == source && day(e.TS) == d && (e.Kind == KindCommit || e.Kind == KindCredit || e.Kind == KindGrant) {
			n += e.Amount
		}
	}
	return n
}

// Balance is a ledger currency replayed.
type Balance struct {
	Currency  string `json:"currency"`
	Spendable int64  `json:"spendable"`
	Sealed    int64  `json:"sealed"`
	Credited  int64  `json:"credited"`
	Spent     int64  `json:"spent"`
}

func (st *state) balance(id string) Balance {
	b := Balance{Currency: id}
	for _, e := range st.entries {
		if e.Currency != id {
			continue
		}
		switch e.Kind {
		case KindCommit, KindCredit, KindGrant:
			b.Credited += e.Amount
			b.Sealed += e.Sealed
		case KindSpend:
			b.Spent += e.Amount
		}
	}
	b.Spendable = b.Credited - b.Sealed - b.Spent
	return b
}

// Balance replays the ledger for one currency.
func (l *Ledger) Balance(currency string) (Balance, error) {
	if _, err := l.ledgerCurrency(currency); err != nil {
		return Balance{}, err
	}
	un, err := l.lock()
	if err != nil {
		return Balance{}, err
	}
	defer un()
	st, err := l.load(false)
	if err != nil {
		return Balance{}, err
	}
	return st.balance(currency), nil
}

// Levels returns how many levels of each upgrade were bought.
func (l *Ledger) Levels() (map[string]int, error) {
	es, err := l.Entries()
	if err != nil {
		return nil, err
	}
	out := map[string]int{}
	for _, e := range es {
		if e.Kind == KindSpend {
			out[e.Upgrade]++
		}
	}
	return out, nil
}

// Append validates e and adds it under the lock. Seq, TS, Prev and Mode are set here. Rules, in order: vanilla mode
// refuses; a credit names a known ledger currency and a source of that currency; a key already booked returns the
// earlier entry with ErrDuplicate (never books twice); a credit above the source's daily soft cap is split into the
// credited part and the sealed part; a spend cannot take the spendable balance below zero.
func (l *Ledger) Append(e Entry) (Entry, error) {
	un, err := l.lock()
	if err != nil {
		return Entry{}, err
	}
	defer un()
	st, err := l.load(true)
	if err != nil {
		return Entry{}, err
	}
	return l.appendLocked(st, e)
}

func (l *Ledger) appendLocked(st *state, e Entry) (Entry, error) {
	if l.vanilla() {
		return Entry{}, ErrVanilla
	}
	switch e.Kind {
	case KindPending, KindVoid, KindRefused, KindCommit, KindCredit, KindGrant, KindSpend:
	default:
		return Entry{}, fmt.Errorf("currency: unknown entry kind %q", e.Kind)
	}
	now := l.opt.Now().UTC()
	e.TS = now.Format(time.RFC3339)
	e.Mode = l.opt.Mode
	if e.Instance == "" {
		e.Instance = l.opt.Instance
	}
	switch e.Kind {
	case KindCommit, KindCredit, KindGrant, KindSpend:
		if _, err := l.ledgerCurrency(e.Currency); err != nil {
			return Entry{}, err
		}
		if e.Amount <= 0 {
			return Entry{}, fmt.Errorf("currency: %s of %d", e.Kind, e.Amount)
		}
	}
	if e.Kind == KindPending {
		if p, ok := st.pendingByKey(e.Key); ok {
			return p, fmt.Errorf("%w: %q is already pending (seq %d)", ErrDuplicate, e.Key, p.Seq)
		}
	}
	if e.Key != "" && e.Kind != KindPending {
		if prev, ok := st.byKey(e.Key); ok {
			return prev, fmt.Errorf("%w: %q (seq %d)", ErrDuplicate, e.Key, prev.Seq)
		}
	}
	switch e.Kind {
	case KindCommit, KindCredit, KindGrant:
		s, ok := l.opt.Registry.Source(e.Source)
		if !ok {
			return Entry{}, fmt.Errorf("%w: %q", ErrUnknownSource, e.Source)
		}
		if s.Currency != e.Currency {
			return Entry{}, fmt.Errorf("%w: %q pays %q, not %q", ErrUnknownSource, s.ID, s.Currency, e.Currency)
		}
		if s.DailyCap > 0 {
			room := max(0, s.DailyCap-st.creditedToday(s.ID, day(e.TS)))
			e.Sealed = max(0, e.Amount-room)
		}
	case KindSpend:
		if bal := st.balance(e.Currency); bal.Spendable < e.Amount {
			return Entry{}, fmt.Errorf("%w: have %d, need %d", ErrInsufficient, bal.Spendable, e.Amount)
		}
	}
	e.Seq = len(st.entries) + 1
	e.Prev = genesis
	if n := len(st.lines); n > 0 {
		e.Prev = hashLine(st.lines[n-1])
	}
	line, err := json.Marshal(e)
	if err != nil {
		return Entry{}, err
	}
	if err := l.write(line); err != nil {
		return Entry{}, err
	}
	st.entries = append(st.entries, e)
	st.lines = append(st.lines, line)
	hb, _ := json.Marshal(head{Seq: e.Seq, Hash: hashLine(line)})
	if err := atomicWrite(l.path("ledger.head"), hb); err != nil {
		return Entry{}, err
	}
	return e, nil
}

// write appends one line with its newline and fsyncs the file.
func (l *Ledger) write(line []byte) error {
	f, err := os.OpenFile(l.path("ledger.jsonl"), os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0o600)
	if err != nil {
		return err
	}
	if _, err := f.Write(append(line, '\n')); err != nil {
		f.Close()
		return err
	}
	if err := f.Sync(); err != nil {
		f.Close()
		return err
	}
	return f.Close()
}

func atomicWrite(path string, b []byte) error {
	tmp := path + ".tmp"
	f, err := os.OpenFile(tmp, os.O_CREATE|os.O_WRONLY|os.O_TRUNC, 0o600)
	if err != nil {
		return err
	}
	if _, err := f.Write(b); err != nil {
		f.Close()
		return err
	}
	if err := f.Sync(); err != nil {
		f.Close()
		return err
	}
	if err := f.Close(); err != nil {
		return err
	}
	return os.Rename(tmp, path)
}

// SpendRequest buys the next level of an upgrade.
type SpendRequest struct {
	Upgrade, Key, Character, Run string
}

// Spend books the cost of the next level of an upgrade. The balance is never clamped: a short balance is refused.
func (l *Ledger) Spend(r SpendRequest) (Entry, error) {
	u, ok := l.opt.Registry.Upgrade(r.Upgrade)
	if !ok {
		return Entry{}, fmt.Errorf("currency: unknown upgrade %q", r.Upgrade)
	}
	un, err := l.lock()
	if err != nil {
		return Entry{}, err
	}
	defer un()
	st, err := l.load(true)
	if err != nil {
		return Entry{}, err
	}
	level := 0
	for _, e := range st.entries {
		if e.Kind == KindSpend && e.Upgrade == u.ID {
			level++
		}
	}
	if level >= len(u.Costs) {
		return Entry{}, fmt.Errorf("currency: upgrade %q is at its top level %d", u.ID, level)
	}
	return l.appendLocked(st, Entry{Kind: KindSpend, Currency: u.Currency, Amount: u.Costs[level], Key: r.Key,
		Upgrade: u.ID, Level: level + 1, Character: r.Character, Run: r.Run})
}

// Repaired reports the bytes of an unfinished last line the latest write dropped.
func (l *Ledger) Repaired() int { return int(l.torn.Load()) }
