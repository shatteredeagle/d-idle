package currency

import (
	"context"
	"errors"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"sync"
	"testing"
	"time"
)

type clock struct{ t time.Time }

func (c *clock) now() time.Time { return c.t }

func newLedger(t *testing.T, mode string) (*Ledger, *clock) {
	t.Helper()
	c := &clock{time.Date(2026, 11, 6, 12, 0, 0, 0, time.UTC)}
	l, err := Open(t.TempDir(), Options{Mode: mode, Instance: "i4", Now: c.now})
	if err != nil {
		t.Fatal(err)
	}
	return l, c
}

const didle = "didle:0123abcd"

func bankReq(key string) BankRequest {
	return BankRequest{Character: "Booti", Run: "run1", Recipe: "cube_bank", Key: key}
}

func TestBankCreditsObservedLossOnly(t *testing.T) {
	l, _ := newLedger(t, didle)
	obs := &Fake{}
	obs.Set("Booti", "mark", 7)
	// the cube takes 5 of the 7 marks (two stay in the pack): the credit follows the observation, not the intent
	res, err := l.Bank(context.Background(), bankReq("Booti/1"), obs, func(context.Context) error { obs.Set("Booti", "mark", 2); return nil })
	if err != nil || res.Before != 7 || res.After != 2 || res.Loss != 5 || res.Credited != 5 {
		t.Fatalf("res %+v err %v", res, err)
	}
	b, _ := l.Balance("tally")
	if b.Spendable != 5 {
		t.Fatalf("balance %+v", b)
	}
	if err := l.Verify(); err != nil {
		t.Fatal(err)
	}
	es, _ := l.Entries()
	if len(es) != 2 || es[0].Kind != KindPending || es[1].Kind != KindCommit || es[1].Ref != 1 || es[1].Mode != didle || es[1].Instance != "i4" {
		t.Fatalf("entries %+v", es)
	}
}

func TestBankReplayNeverDoubleCredits(t *testing.T) {
	l, _ := newLedger(t, didle)
	obs := &Fake{}
	obs.Set("Booti", "mark", 4)
	act := func(context.Context) error { obs.Set("Booti", "mark", 0); return nil }
	if _, err := l.Bank(context.Background(), bankReq("Booti/1"), obs, act); err != nil {
		t.Fatal(err)
	}
	// the same observation key again, even with fresh marks in the pack
	obs.Set("Booti", "mark", 9)
	res, err := l.Bank(context.Background(), bankReq("Booti/1"), obs, act)
	if err != nil || !res.Replayed || res.Credited != 4 {
		t.Fatalf("replay %+v %v", res, err)
	}
	if b, _ := l.Balance("tally"); b.Spendable != 4 {
		t.Fatalf("double credit: %+v", b)
	}
	if es, _ := l.Entries(); len(es) != 2 {
		t.Fatalf("replay wrote %d entries", len(es))
	}
}

func TestBankUnchangedPackVoids(t *testing.T) {
	l, _ := newLedger(t, didle)
	obs := &Fake{}
	obs.Set("Booti", "mark", 3)
	res, err := l.Bank(context.Background(), bankReq("k"), obs, func(context.Context) error { return errors.New("cube busy") })
	if err == nil || res == nil || !res.Voided || res.Credited != 0 {
		t.Fatalf("res %+v err %v", res, err)
	}
	if b, _ := l.Balance("tally"); b.Spendable != 0 {
		t.Fatal("voided bank credited")
	}
}

func TestBankRefusals(t *testing.T) {
	ctx := context.Background()
	obs := &Fake{CheatedRun: map[string]bool{"cheat": true}}
	obs.Set("Booti", "mark", 3)

	van, _ := newLedger(t, "vanilla")
	if _, err := van.BeginBank(ctx, bankReq("k"), obs); !errors.Is(err, ErrVanilla) {
		t.Fatalf("vanilla: %v", err)
	}
	if _, err := van.Append(Entry{Kind: KindGrant, Currency: "tally", Source: "grant", Key: "g", Amount: 5}); !errors.Is(err, ErrVanilla) {
		t.Fatalf("vanilla append: %v", err)
	}
	if _, err := os.Stat(filepath.Join(van.dir, "ledger.jsonl")); err == nil {
		t.Fatal("vanilla created a ledger file")
	}

	l, _ := newLedger(t, didle)
	req := bankReq("k2")
	req.Run = "cheat"
	if _, err := l.BeginBank(ctx, req, obs); !errors.Is(err, ErrCheated) {
		t.Fatalf("cheated begin: %v", err)
	}
	// a run that becomes cheated between the pending entry and the commit is refused and closed
	tk, err := l.BeginBank(ctx, bankReq("k3"), obs)
	if err != nil {
		t.Fatal(err)
	}
	obs.CheatedRun["run1"] = true
	obs.Set("Booti", "mark", 0)
	if _, err := l.CommitBank(ctx, tk, obs); !errors.Is(err, ErrCheated) {
		t.Fatalf("cheated commit: %v", err)
	}
	if b, _ := l.Balance("tally"); b.Spendable != 0 {
		t.Fatal("cheated stack credited")
	}
	es, _ := l.Entries()
	if es[len(es)-1].Kind != KindRefused {
		t.Fatalf("last entry %+v", es[len(es)-1])
	}

	if _, err := l.BeginBank(ctx, BankRequest{Character: "Booti", Recipe: "nope", Key: "x"}, obs); !errors.Is(err, ErrUnknownRecipe) {
		t.Fatalf("recipe: %v", err)
	}
	if _, err := l.Append(Entry{Kind: KindGrant, Currency: "gold", Source: "grant", Key: "g", Amount: 1}); !errors.Is(err, ErrUnknownCurrency) {
		t.Fatalf("currency: %v", err)
	}
	if _, err := l.Append(Entry{Kind: KindGrant, Currency: "mark", Source: "mark_field", Key: "g", Amount: 1}); !errors.Is(err, ErrNotLedger) {
		t.Fatalf("item currency in the ledger: %v", err)
	}
	if _, err := l.Append(Entry{Kind: KindGrant, Currency: "tally", Source: "mark_field", Key: "g", Amount: 1}); !errors.Is(err, ErrUnknownSource) {
		t.Fatalf("foreign source: %v", err)
	}
	obs.Set("Nobody", "mark", 0)
	if _, err := l.BeginBank(ctx, BankRequest{Character: "Nobody", Recipe: "cube_bank", Key: "e"}, obs); !errors.Is(err, ErrNothingToBank) {
		t.Fatalf("empty pack: %v", err)
	}
}

func TestCrashBetweenPendingAndCommitRecovers(t *testing.T) {
	ctx := context.Background()
	l, _ := newLedger(t, didle)
	obs := &Fake{}
	obs.Set("Booti", "mark", 6)
	if _, err := l.BeginBank(ctx, bankReq("Booti/9"), obs); err != nil {
		t.Fatal(err)
	}
	// the process dies here; the cube had taken 4 marks
	obs.Set("Booti", "mark", 2)
	l2, err := Open(l.dir, Options{Mode: didle, Now: l.opt.Now})
	if err != nil {
		t.Fatal(err)
	}
	res, err := l2.Recover(ctx, obs)
	if err != nil || len(res) != 1 || res[0].Credited != 4 {
		t.Fatalf("recover %+v %v", res, err)
	}
	if res, _ := l2.Recover(ctx, obs); len(res) != 0 {
		t.Fatal("second recover settled again")
	}
	if b, _ := l2.Balance("tally"); b.Spendable != 4 {
		t.Fatalf("balance %+v", b)
	}
	// a pending whose pack never changed is voided
	obs.Set("Booti", "mark", 5)
	l2.BeginBank(ctx, bankReq("Booti/10"), obs)
	res, _ = l2.Recover(ctx, obs)
	if len(res) != 1 || !res[0].Voided {
		t.Fatalf("void: %+v", res)
	}
	// more marks in the pack than before never credit more than the pending recorded
	obs.Set("Booti", "mark", 5)
	l2.BeginBank(ctx, bankReq("Booti/11"), obs)
	obs.Set("Booti", "mark", 12)
	if res, _ = l2.Recover(ctx, obs); res[0].Credited != 0 {
		t.Fatalf("growth credited: %+v", res)
	}
	if err := l2.Verify(); err != nil {
		t.Fatal(err)
	}
}

func TestTornTailIsRepairedOnAppend(t *testing.T) {
	l, _ := newLedger(t, didle)
	l.Grant("tally", "grant", "g1", 5, "")
	f, _ := os.OpenFile(l.path("ledger.jsonl"), os.O_APPEND|os.O_WRONLY, 0)
	f.WriteString(`{"seq":2,"ts":"2026`)
	f.Close()
	if err := l.Verify(); !errors.Is(err, ErrCorrupt) {
		t.Fatalf("verify torn: %v", err)
	}
	if _, err := l.Grant("tally", "grant", "g2", 3, ""); err != nil {
		t.Fatal(err)
	}
	if l.Repaired() == 0 {
		t.Error("repair not reported")
	}
	if err := l.Verify(); err != nil {
		t.Fatal(err)
	}
	if b, _ := l.Balance("tally"); b.Spendable != 8 {
		t.Fatalf("balance %+v", b)
	}
}

func TestVerifyDetectsTamperAndTruncation(t *testing.T) {
	for _, tc := range []struct {
		name string
		edit func(path string, b []byte) []byte
	}{
		{"amount edited", func(_ string, b []byte) []byte {
			return []byte(strings.Replace(string(b), `"amount":5`, `"amount":50`, 1))
		}},
		{"line removed from the middle", func(_ string, b []byte) []byte {
			ls := strings.SplitAfter(string(b), "\n")
			return []byte(ls[0] + ls[2] + ls[3])
		}},
		{"tail truncated", func(_ string, b []byte) []byte {
			ls := strings.SplitAfter(string(b), "\n")
			return []byte(ls[0] + ls[1])
		}},
		{"reordered", func(_ string, b []byte) []byte {
			ls := strings.SplitAfter(string(b), "\n")
			return []byte(ls[1] + ls[0] + ls[2])
		}},
	} {
		l, _ := newLedger(t, didle)
		for i, k := range []string{"a", "b", "c"} {
			if _, err := l.Grant("tally", "grant", k, int64(5+i), ""); err != nil {
				t.Fatal(err)
			}
		}
		if err := l.Verify(); err != nil {
			t.Fatalf("%s: clean ledger: %v", tc.name, err)
		}
		p := l.path("ledger.jsonl")
		b, _ := os.ReadFile(p)
		os.WriteFile(p, tc.edit(p, b), 0o600)
		if err := l.Verify(); !errors.Is(err, ErrCorrupt) {
			t.Errorf("%s: Verify = %v", tc.name, err)
		}
		if _, err := l.Balance("tally"); err == nil {
			t.Errorf("%s: Balance read a corrupt ledger", tc.name)
		}
		if _, err := l.Grant("tally", "grant", "z", 1, ""); err == nil {
			t.Errorf("%s: Append extended a corrupt ledger", tc.name)
		}
	}
}

func TestHeadBehindByOneIsFine(t *testing.T) {
	l, _ := newLedger(t, didle)
	l.Grant("tally", "grant", "a", 5, "")
	old, _ := os.ReadFile(l.path("ledger.head"))
	l.Grant("tally", "grant", "b", 5, "")
	os.WriteFile(l.path("ledger.head"), old, 0o600) // a crash after the line, before the head
	if err := l.Verify(); err != nil {
		t.Fatalf("head one behind: %v", err)
	}
}

func TestSoftCapSealsTheExcess(t *testing.T) {
	l, c := newLedger(t, didle) // bank daily_cap is 2000
	if e, err := l.Grant("tally", "bank", "d1a", 1500, ""); err != nil || e.Sealed != 0 {
		t.Fatalf("%+v %v", e, err)
	}
	e, err := l.Grant("tally", "bank", "d1b", 800, "")
	if err != nil || e.Sealed != 300 {
		t.Fatalf("over cap: %+v %v", e, err)
	}
	b, _ := l.Balance("tally")
	if b.Spendable != 2000 || b.Sealed != 300 || b.Credited != 2300 {
		t.Fatalf("balance %+v", b)
	}
	if e, _ := l.Grant("tally", "bank", "d1c", 10, ""); e.Sealed != 10 {
		t.Fatalf("cap already full: %+v", e)
	}
	c.t = c.t.Add(24 * time.Hour) // the next UTC day starts fresh
	if e, _ := l.Grant("tally", "bank", "d2", 10, ""); e.Sealed != 0 {
		t.Fatalf("next day: %+v", e)
	}
}

func TestSpendRefusesShortBalanceAndTracksLevels(t *testing.T) {
	l, _ := newLedger(t, didle)
	l.Grant("tally", "grant", "g", 100, "")
	if e, err := l.Spend(SpendRequest{Upgrade: "fade_faster", Key: "s1"}); err != nil || e.Amount != 30 || e.Level != 1 {
		t.Fatalf("level 1: %+v %v", e, err)
	}
	if _, err := l.Spend(SpendRequest{Upgrade: "fade_faster", Key: "s1"}); !errors.Is(err, ErrDuplicate) {
		t.Fatalf("replayed spend: %v", err)
	}
	if _, err := l.Spend(SpendRequest{Upgrade: "fade_faster", Key: "s2"}); !errors.Is(err, ErrInsufficient) { // 70 left, level 2 costs 75
		t.Fatalf("short balance: %v", err)
	}
	if b, _ := l.Balance("tally"); b.Spendable != 70 {
		t.Fatalf("clamped or lost: %+v", b)
	}
	l.Grant("tally", "grant", "g2", 10, "")
	if e, err := l.Spend(SpendRequest{Upgrade: "fade_faster", Key: "s2"}); err != nil || e.Level != 2 {
		t.Fatalf("level 2: %+v %v", e, err)
	}
	if lv, _ := l.Levels(); lv["fade_faster"] != 2 {
		t.Fatalf("levels %v", lv)
	}
	if _, err := l.Spend(SpendRequest{Upgrade: "nope", Key: "x"}); err == nil {
		t.Fatal("unknown upgrade")
	}
}

func TestConcurrentAppendsStayChained(t *testing.T) {
	l, _ := newLedger(t, didle)
	var wg sync.WaitGroup
	for g := 0; g < 8; g++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			// each goroutine has its own Ledger value on the same directory, as separate instances would
			l2, _ := Open(l.dir, Options{Mode: didle, Now: l.opt.Now})
			for i := 0; i < 10; i++ {
				if _, err := l2.Append(Entry{Kind: KindCredit, Currency: "tally", Source: "grant", Amount: 1}); err != nil {
					t.Error(err)
				}
			}
		}()
	}
	wg.Wait()
	if err := l.Verify(); err != nil {
		t.Fatal(err)
	}
	if b, _ := l.Balance("tally"); b.Credited != 80 {
		t.Fatalf("lost appends: %+v", b)
	}
}

// TestConcurrentProcesses re-runs the test binary as helper processes that append to the same ledger (flock).
func TestConcurrentProcesses(t *testing.T) {
	if dir := os.Getenv("CURRENCY_HELPER_DIR"); dir != "" {
		l, _ := Open(dir, Options{Mode: didle})
		for i := 0; i < 15; i++ {
			if _, err := l.Append(Entry{Kind: KindCredit, Currency: "tally", Source: "grant", Amount: 1}); err != nil {
				t.Fatal(err)
			}
		}
		return
	}
	l, _ := newLedger(t, didle)
	var cmds []*exec.Cmd
	for i := 0; i < 4; i++ {
		c := exec.Command(os.Args[0], "-test.run=^TestConcurrentProcesses$")
		c.Env = append(os.Environ(), "CURRENCY_HELPER_DIR="+l.dir)
		if err := c.Start(); err != nil {
			t.Fatal(err)
		}
		cmds = append(cmds, c)
	}
	for _, c := range cmds {
		if err := c.Wait(); err != nil {
			t.Fatal(err)
		}
	}
	if err := l.Verify(); err != nil {
		t.Fatal(err)
	}
	if b, _ := l.Balance("tally"); b.Credited != 60 {
		t.Fatalf("lost appends across processes: %+v", b)
	}
}

func TestLedgerRefusesUnreadableEntryKinds(t *testing.T) {
	l, _ := newLedger(t, didle)
	if _, err := l.Append(Entry{Kind: "mint", Currency: "tally", Amount: 9}); err == nil {
		t.Fatal("unknown kind accepted")
	}
	if _, err := l.Append(Entry{Kind: KindCredit, Currency: "tally", Source: "grant", Amount: -3}); err == nil {
		t.Fatal("negative credit accepted")
	}
}
