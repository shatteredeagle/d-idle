package town_test

import (
	"bytes"
	"context"
	"d2lod-bot/internal/gear/geartest"
	"log/slog"
	"strings"
	"testing"

	"d2lod-bot/internal/town"
)

// Every purchase writes a greppable "shop: bought" line and every shop visit one "shop: visit" summary with the kit
// state (G1 attempt 6: the belt criterion could not be judged from the log).
func TestShopLogsEveryPurchase(t *testing.T) {
	geartest.Need(t)
	r := newRig(t)
	var buf bytes.Buffer
	r.Env.Log = slog.New(slog.NewTextHandler(&buf, nil))
	rep, err := r.T.Chores(context.Background(), town.ChorePlan{Potions: true, Scrolls: true})
	if err != nil {
		t.Fatal(err)
	}
	n := 0
	for _, c := range rep.Bought {
		n += c
	}
	if n == 0 {
		t.Fatalf("nothing bought with 3000 gold: %v", rep.Bought)
	}
	out := buf.String()
	if got := strings.Count(out, "shop: bought"); got != n {
		t.Errorf("%d purchase lines for %d purchases:\n%s", got, n, out)
	}
	for _, want := range []string{"shop: visit", "beltHP=", "tp="} {
		if !strings.Contains(out, want) {
			t.Errorf("log lacks %q:\n%s", want, out)
		}
	}
}

// A visit with the gold for a tome and no tome in stock says so instead of staying silent.
func TestShopWarnsWhenNoTomeInStock(t *testing.T) {
	geartest.Need(t)
	r := newRig(t)
	r.W.Stock(r.Akara, "hp1", "mp1", "tsc")
	var buf bytes.Buffer
	r.Env.Log = slog.New(slog.NewTextHandler(&buf, nil))
	if _, err := r.T.Chores(context.Background(), town.ChorePlan{Scrolls: true}); err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(buf.String(), "shop: no tome in stock") {
		t.Errorf("no tome warning:\n%s", buf.String())
	}
}
