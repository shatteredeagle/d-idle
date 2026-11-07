package currency

import (
	"context"
	"errors"
	"fmt"
)

// Observer is what the bot side gives the bank operation (the hook is job B's). Both reads are read-only.
type Observer interface {
	// PackCount is how many units of the item currency the character carries now (pack, not stash).
	PackCount(ctx context.Context, character, currency string) (int, error)
	// Cheated says whether the run has gameplay writes or the stack was set up by a cheat write
	// (the cheat ledger: internal/cheat Ledger.CheatedKey since the run start). true refuses the credit.
	Cheated(ctx context.Context, character, run string) (bool, error)
}

// BankRequest is one bank action. Key is the idempotency key of the observation, e.g. "<character>/<n>"; the same key
// never credits twice, whatever the pack looks like on a replay.
type BankRequest struct {
	Character, Run, Instance string
	Recipe                   string // converts_to.via of the registry row, e.g. cube_bank
	Key                      string
}

// BankResult is the outcome of a bank action.
type BankResult struct {
	Key      string
	Seq      int   // seq of the closing entry
	Before   int   // pack count before the action
	After    int   // pack count observed after it
	Loss     int   // Before - After, the observed loss (never the intended number)
	Credited int64 // units of the target currency credited (0 when voided)
	Sealed   int64 // part of Credited above the daily soft cap
	Currency string
	Replayed bool // the key was already closed; nothing was written
	Voided   bool
}

// Ticket is an open bank action (the write-ahead entry exists).
type Ticket struct {
	Req     BankRequest
	From    *Currency
	PendSeq int
	Before  int
	done    *BankResult
}

// BeginBank checks the rules and writes the pending entry with the pack count before the action. A key that is
// already closed returns a ticket that Commit answers from the ledger; a key that is pending resumes it.
func (l *Ledger) BeginBank(ctx context.Context, req BankRequest, obs Observer) (*Ticket, error) {
	if l.vanilla() {
		return nil, ErrVanilla
	}
	if req.Key == "" || req.Character == "" {
		return nil, errors.New("currency: a bank request needs a character and a key")
	}
	from, err := l.opt.Registry.ByRecipe(req.Recipe)
	if err != nil {
		return nil, err
	}
	if _, err := l.ledgerCurrency(from.Converts.Currency); err != nil {
		return nil, err
	}
	un, err := l.lock()
	if err != nil {
		return nil, err
	}
	st, err := l.load(true)
	un()
	if err != nil {
		return nil, err
	}
	if e, ok := st.byKey(req.Key); ok {
		return &Ticket{Req: req, From: from, done: resultOf(e)}, nil
	}
	if p, ok := st.pendingByKey(req.Key); ok {
		return &Ticket{Req: req, From: from, PendSeq: p.Seq, Before: *p.Before}, nil
	}
	if cheated, err := obs.Cheated(ctx, req.Character, req.Run); err != nil {
		return nil, err
	} else if cheated {
		return nil, ErrCheated
	}
	before, err := obs.PackCount(ctx, req.Character, from.ID)
	if err != nil {
		return nil, err
	}
	if before <= 0 {
		return nil, ErrNothingToBank
	}
	e, err := l.Append(Entry{Kind: KindPending, Currency: from.ID, Key: req.Key, Character: req.Character, Run: req.Run,
		Instance: req.Instance, Recipe: req.Recipe, Before: &before})
	if err != nil {
		return nil, err
	}
	return &Ticket{Req: req, From: from, PendSeq: e.Seq, Before: before}, nil
}

// resultOf rebuilds the result of an already closed key.
func resultOf(e Entry) *BankResult {
	r := &BankResult{Key: e.Key, Seq: e.Seq, Replayed: true, Credited: 0, Sealed: e.Sealed, Voided: e.Kind != KindCommit}
	if e.Kind == KindCommit {
		r.Credited, r.Currency = e.Amount, e.Currency
	}
	if e.Before != nil {
		r.Before = *e.Before
	}
	if e.After != nil {
		r.After = *e.After
	}
	r.Loss = r.Before - r.After
	return r
}

// CommitBank observes the pack again and books the outcome: the credit is the observed loss times the ratio. An
// unchanged pack voids the pending entry; a run that turned out cheated closes it as refused (ErrCheated).
func (l *Ledger) CommitBank(ctx context.Context, t *Ticket, obs Observer) (*BankResult, error) {
	if t.done != nil {
		return t.done, nil
	}
	after, err := obs.PackCount(ctx, t.Req.Character, t.From.ID)
	if err != nil {
		return nil, err
	}
	return l.settle(ctx, t.Req, t.From, t.PendSeq, t.Before, after, obs)
}

func (l *Ledger) settle(ctx context.Context, req BankRequest, from *Currency, pend, before, after int, obs Observer) (*BankResult, error) {
	if after < 0 {
		return nil, fmt.Errorf("currency: observed pack count %d", after)
	}
	base := Entry{Key: req.Key, Ref: pend, Character: req.Character, Run: req.Run, Instance: req.Instance, Recipe: req.Recipe,
		Before: &before, After: &after}
	res := &BankResult{Key: req.Key, Before: before, After: after, Loss: max(0, before-after), Currency: from.Converts.Currency}
	cheated, err := obs.Cheated(ctx, req.Character, req.Run)
	if err != nil {
		return nil, err
	}
	var e Entry
	switch {
	case cheated:
		e = base
		e.Kind, e.Note = KindRefused, "cheated run"
	case res.Loss == 0:
		e = base
		e.Kind, e.Note = KindVoid, "pack unchanged"
		res.Voided = true
	default:
		e = base
		e.Kind, e.Currency, e.Source = KindCommit, from.Converts.Currency, "bank"
		e.Amount = int64(res.Loss) * from.Converts.Ratio
	}
	got, err := l.Append(e)
	if errors.Is(err, ErrDuplicate) { // a concurrent or replayed settle: the first one stands
		return resultOf(got), nil
	}
	if err != nil {
		return nil, err
	}
	res.Seq = got.Seq
	if got.Kind == KindCommit {
		res.Credited, res.Sealed = got.Amount, got.Sealed
	}
	if cheated {
		return res, ErrCheated
	}
	return res, nil
}

// Bank runs a whole bank action: pending entry, act (the cube transmute, job B's), then the observed commit. The
// commit happens even when act fails, because the pack decides what was lost.
func (l *Ledger) Bank(ctx context.Context, req BankRequest, obs Observer, act func(context.Context) error) (*BankResult, error) {
	t, err := l.BeginBank(ctx, req, obs)
	if err != nil {
		return nil, err
	}
	var actErr error
	if t.done == nil && act != nil {
		actErr = act(ctx)
	}
	res, err := l.CommitBank(ctx, t, obs)
	if err != nil {
		return res, err
	}
	if actErr != nil {
		return res, fmt.Errorf("currency: bank action: %w", actErr)
	}
	return res, nil
}

// Recover settles every pending bank entry left by a crash: the pack is read again, the credit is the loss since the
// pending entry (never more than it recorded), and an unchanged pack voids it. Call it at start, in the new session.
func (l *Ledger) Recover(ctx context.Context, obs Observer) ([]*BankResult, error) {
	if l.vanilla() {
		return nil, nil
	}
	un, err := l.lock()
	if err != nil {
		return nil, err
	}
	st, err := l.load(true)
	un()
	if err != nil {
		return nil, err
	}
	var out []*BankResult
	for _, p := range st.entries {
		if p.Kind != KindPending || st.closed(p.Seq) {
			continue
		}
		from, err := l.opt.Registry.ByRecipe(p.Recipe)
		if err != nil {
			return out, err
		}
		after, err := obs.PackCount(ctx, p.Character, from.ID)
		if err != nil {
			return out, err
		}
		req := BankRequest{Character: p.Character, Run: p.Run, Instance: p.Instance, Recipe: p.Recipe, Key: p.Key}
		res, err := l.settle(ctx, req, from, p.Seq, *p.Before, min(after, *p.Before), obs)
		if res != nil {
			out = append(out, res)
		}
		if err != nil && !errors.Is(err, ErrCheated) {
			return out, err
		}
	}
	return out, nil
}

// Grant books a one-off credit from a grant source (tier unlock, first rule). The key makes a replay harmless.
func (l *Ledger) Grant(currency, source, key string, amount int64, note string) (Entry, error) {
	if key == "" {
		return Entry{}, errors.New("currency: a grant needs a key")
	}
	return l.Append(Entry{Kind: KindGrant, Currency: currency, Source: source, Key: key, Amount: amount, Note: note})
}

// Fake is an Observer for tests and drills.
type Fake struct {
	Packs      map[string]int  // "character/currency" -> count
	CheatedRun map[string]bool // run -> cheated
	Err        error
}

func fakeKey(character, currency string) string { return character + "/" + currency }

func (f *Fake) PackCount(_ context.Context, character, currency string) (int, error) {
	return f.Packs[fakeKey(character, currency)], f.Err
}

func (f *Fake) Cheated(_ context.Context, _, run string) (bool, error) {
	return f.CheatedRun[run], f.Err
}

// Set puts n units in a character's pack.
func (f *Fake) Set(character, currency string, n int) {
	if f.Packs == nil {
		f.Packs = map[string]int{}
	}
	f.Packs[fakeKey(character, currency)] = n
}
