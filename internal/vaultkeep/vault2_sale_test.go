package vaultkeep

import (
	"context"
	"errors"
	"path/filepath"
	"testing"

	"d2lod-bot/internal/d2dctl"
	"d2lod-bot/internal/vault"
)

// Card vault2: the instant sale reserves its records in the store before the credit. Every test here MUST FAIL when a
// record can be credited and still stay in the vault (or be kept, pinned or discarded after it was paid for).

// hookCredit runs a function while the credit is on the wire.
type hookCredit struct {
	d2dctl.GoldCreditor
	during func()
}

func (h hookCredit) CreditGold(ctx context.Context, amount int, order string, items int) (d2dctl.CreditResult, error) {
	if h.during != nil {
		h.during()
	}
	return h.GoldCreditor.CreditGold(ctx, amount, order, items)
}

var rivals = map[string]func(st *vault.Store, id string) error{
	"keep":    func(st *vault.Store, id string) error { return st.Keep(id) },
	"pin":     func(st *vault.Store, id string) error { return st.Pin(id) },
	"discard": func(st *vault.Store, id string) error { return st.Discard(id, "player") },
}

func soldGold(r *saleRig, ids ...string) (gold int) {
	for _, id := range ids {
		if rec := r.state(id); rec.State == vault.Sold {
			gold += rec.Gold
		}
	}
	return gold
}

// A rival lands between the quote and the reservation: it wins, and the credit is only for what is really sold.
func TestVault2RivalBeforeReserveIsNotPaidFor(t *testing.T) {
	for _, class := range classes {
		for name, rival := range rivals {
			t.Run(class+"/"+name, func(t *testing.T) {
				r := newSaleRig(t, class, 1000)
				a, b := r.item(1, 2, 7, 100), r.item(2, 3, 9, 200)
				once := false
				r.k.reserveHook = func() {
					if !once {
						once = true
						if err := rival(r.st, a); err != nil {
							t.Fatalf("rival before the reservation refused: %v", err)
						}
					}
				}
				res, v := r.sellNow(false, a, b)
				if v == nil || v.State != SaleDone || r.pay.Stash-1000 != soldGold(r, a, b) || r.pay.Stash != 1200 {
					t.Fatalf("credited %d, sold for %d: res %+v view %+v", r.pay.Stash-1000, soldGold(r, a, b), res, v)
				}
				if r.state(a).State == vault.Sold || res[0].OK || len(r.st.Held()) != 0 {
					t.Fatalf("the rival's record: %+v res %+v held %v", r.state(a), res[0], r.st.Held())
				}
			})
		}
	}
}

// A rival lands while the credit is on the wire: it is refused, the record is sold, gold and records agree.
func TestVault2RivalDuringCreditIsRefused(t *testing.T) {
	for _, class := range classes {
		for name, rival := range rivals {
			t.Run(class+"/"+name, func(t *testing.T) {
				r := newSaleRig(t, class, 1000)
				a, b := r.item(1, 2, 7, 100), r.item(2, 3, 9, 200)
				var got error
				r.k.Credit = hookCredit{r.pay, func() { got = rival(r.st, a) }}
				res, v := r.sellNow(false, a, b)
				if !errors.Is(got, vault.ErrHeld) {
					t.Fatalf("the rival moved a reserved record: %v", got)
				}
				if okCount(res) != 2 || v.Done != 2 || r.pay.Stash != 1300 || soldGold(r, a, b) != 300 || v.Note != "" {
					t.Fatalf("res %+v view %+v stash %d", res, v, r.pay.Stash)
				}
			})
		}
	}
}

// A refused credit gives the reservation back; an unanswered one keeps it across a restart and the same order finishes.
func TestVault2CreditFailureAfterReserve(t *testing.T) {
	for _, class := range classes {
		t.Run(class, func(t *testing.T) {
			r := newSaleRig(t, class, 1000)
			a := r.item(1, 2, 7, 100)
			r.pay.FlagOff = true
			if res, _ := r.sellNow(false, a); res[0].OK || len(r.st.Held()) != 0 || r.pay.Stash != 1000 {
				t.Fatalf("refused credit: %+v held %v", res, r.st.Held())
			}
			if err := r.st.Pin(a); err != nil {
				t.Fatalf("released record cannot be pinned: %v", err)
			}
			must(t, r.st.Unpin(a))
			r.pay.FlagOff, r.pay.Fail = false, errors.New("timeout")
			res, v := r.sellNow(false, a)
			if res[0].OK || v == nil || v.State != SaleQueued || len(r.st.Held()) != 1 {
				t.Fatalf("unanswered credit: %+v %+v held %v", res, v, r.st.Held())
			}
			for name, rival := range rivals {
				if err := rival(r.st, a); !errors.Is(err, vault.ErrHeld) {
					t.Fatalf("%s took a record whose credit is unanswered: %v", name, err)
				}
			}
			r.pay.Fail = nil
			r.open() // the bot restarts; the order file and the hold are on disk
			r.wireSale()
			if len(r.st.Held()) != 1 {
				t.Fatalf("the hold did not survive the restart: %v", r.st.Held())
			}
			r.k.Run(context.Background())
			if r.state(a).State != vault.Sold || r.pay.Stash != 1100 || len(r.st.Held()) != 0 {
				t.Fatalf("after the restart: %+v stash %d", r.state(a), r.pay.Stash)
			}
		})
	}
}

// The order file cannot be written: no credit, nothing held. A hold with no order (a stop between the two) is given back.
func TestVault2NoOrderFileNoCredit(t *testing.T) {
	for _, class := range classes {
		t.Run(class, func(t *testing.T) {
			r := newSaleRig(t, class, 1000)
			a := r.item(1, 2, 7, 100)
			r.k.SaleFile = func() string { return filepath.Join(r.dir, "nodir\x00", "sale.json") }
			res, _ := r.sellNow(false, a)
			if res[0].OK || res[0].Code != "journal" || len(r.pay.Calls) != 0 || len(r.st.Held()) != 0 {
				t.Fatalf("unsaved order ran: %+v calls %v held %v", res, r.pay.Calls, r.st.Held())
			}
			r.wireSale()
			if _, err := r.st.Reserve([]string{a}, "s-lost", 0); err != nil {
				t.Fatal(err)
			}
			r.k.Run(context.Background())
			if len(r.st.Held()) != 0 || r.state(a).State != vault.Recorded || len(r.pay.Calls) != 0 {
				t.Fatalf("orphan hold: %v %+v", r.st.Held(), r.state(a))
			}
			if res, v := r.sellNow(false, a); !res[0].OK || v.Gold != 100 {
				t.Fatalf("sale after the orphan was released: %+v", res)
			}
		})
	}
}

// Sell all reserves too, and a reserved record is left out of a second plan.
func TestVault2SellAllReserves(t *testing.T) {
	for _, class := range classes {
		t.Run(class, func(t *testing.T) {
			r := newSaleRig(t, class, 0)
			a, b := r.item(1, 2, 7, 100), r.item(2, 3, 9, 200)
			var got error
			r.k.Credit = hookCredit{r.pay, func() {
				got = r.st.Keep(a)
				if p := r.k.SellAllPlan(context.Background()); len(p.Quote.Lines) != 0 {
					t.Errorf("a reserved record is in a second plan: %+v", p.Quote.Lines)
				}
			}}
			out := r.k.SellAll(context.Background(), "tap1")
			if !errors.Is(got, vault.ErrHeld) || out.View == nil || out.View.Done != 2 || r.pay.Stash != soldGold(r, a, b) {
				t.Fatalf("rival %v out %+v stash %d", got, out, r.pay.Stash)
			}
		})
	}
}

func must(t *testing.T, err error) {
	t.Helper()
	if err != nil {
		t.Fatal(err)
	}
}
