package watch

import (
	"context"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"testing"
	"time"

	"d2lod-bot/internal/currency"
)

const testMode = "didle:0123abcd"

// currencyRig attaches a wallet with a fixed ledger: 30 tally granted, 7 marks banked (tally 37), the hero carries 5.
func currencyRig(t *testing.T, build, mode string) *rig {
	t.Helper()
	r := newRig(t, build)
	var l *currency.Ledger
	if mode != currency.VanillaMode {
		now := time.Date(2026, 11, 6, 12, 0, 0, 0, time.UTC)
		var err error
		if l, err = currency.Open(t.TempDir(), currency.Options{Mode: mode, Instance: "i4", Now: func() time.Time { return now }}); err != nil {
			t.Fatal(err)
		}
		if _, err := l.Grant("tally", "grant", "tier-t0", 30, "T0 unlock"); err != nil {
			t.Fatal(err)
		}
		obs := &currency.Fake{}
		obs.Set("Hero", "mark", 7)
		if _, err := l.Bank(context.Background(), currency.BankRequest{Character: "Hero", Run: "r1", Recipe: "cube_bank", Key: "Hero/1"}, obs,
			func(context.Context) error { obs.Set("Hero", "mark", 0); return nil }); err != nil {
			t.Fatal(err)
		}
	}
	w := currency.NewWallet(mode, l)
	w.SetCarried("Hero", "mark", 5)
	r.sim.Hub.SetCurrency(w)
	return r
}

func TestCurrencyGoldenViews(t *testing.T) {
	for _, build := range []string{"sorceress", "paladin"} {
		r := currencyRig(t, build, testMode)
		w := r.get("/api/currency", nil)
		if w.Code != 200 {
			t.Fatalf("%s: %d %s", build, w.Code, w.Body)
		}
		golden(t, "currency-"+build, w.Body.String())
		var cfg ConfigView
		r.get("/api/config", &cfg)
		if !cfg.Currency {
			t.Errorf("%s: config does not announce the strip", build)
		}
	}
	r := currencyRig(t, "sorceress", currency.VanillaMode)
	golden(t, "currency-vanilla", r.get("/api/currency", nil).Body.String())
}

func TestCurrencyViewContent(t *testing.T) {
	r := currencyRig(t, "sorceress", testMode)
	var v CurrencyView
	r.get("/api/currency", &v)
	if !v.Enabled || v.Schema != "watch.currency/1" || v.Strip != "Raven's Mark 5 in pack · Tally 37 banked" {
		t.Fatalf("view %+v", v)
	}
	// 37 tally: level 1 (30) is affordable but not wired, so every Buy button stays off
	for _, u := range v.Spend {
		if u.Enabled || !u.Affordable || u.Why != "Not available yet." {
			t.Errorf("row %+v", u)
		}
	}
	// a short balance says how much is missing
	r2 := currencyRig(t, "sorceress", testMode)
	w := r2.sim.Hub.cur.Load()
	w.Ledger.Spend(currency.SpendRequest{Upgrade: "fade_faster", Key: "s1"}) // 37 - 30 = 7
	var v2 CurrencyView
	r2.get("/api/currency", &v2)
	for _, u := range v2.Spend {
		if u.ID == "fade_faster" && (u.Level != 1 || u.Affordable || u.Why != "Needs 68 more tally.") {
			t.Errorf("short row %+v", u)
		}
	}
	// a pack the bot has not reported yet is null, not zero
	r3 := newRig(t, "sorceress")
	l, _ := currency.Open(t.TempDir(), currency.Options{Mode: testMode})
	r3.sim.Hub.SetCurrency(currency.NewWallet(testMode, l))
	var v3 CurrencyView
	r3.get("/api/currency", &v3)
	if v3.Currencies[0].Amount != nil || v3.Currencies[1].Amount == nil || *v3.Currencies[1].Amount != 0 {
		t.Errorf("unknown pack: %+v", v3.Currencies)
	}
}

func TestCurrencyTamperedLedgerHidesBalances(t *testing.T) {
	r := currencyRig(t, "sorceress", testMode)
	w := r.sim.Hub.cur.Load()
	if es, _ := w.Ledger.Entries(); len(es) == 0 {
		t.Fatal("rig has no entries")
	}
	// cut the head file's view of the ledger: remove it
	if err := os.Remove(filepath.Join(w.Ledger.Dir(), "ledger.head")); err != nil {
		t.Fatal(err)
	}
	var v CurrencyView
	r.get("/api/currency", &v)
	if v.Error == "" || v.Currencies[1].Amount != nil {
		t.Fatalf("tampered ledger shown as fine: %+v", v)
	}
}

func TestCurrencyOffMeansUnchanged(t *testing.T) {
	r := newRig(t, "sorceress")
	if w := r.get("/api/currency", nil); w.Code != 404 {
		t.Errorf("no wallet: %d", w.Code)
	}
	if strings.Contains(r.get("/api/config", nil).Body.String(), "currency") {
		t.Error("config mentions currency without a wallet")
	}
	r = currencyRig(t, "sorceress", testMode)
	for _, m := range []string{"POST", "PUT", "DELETE"} {
		w := httptest.NewRecorder()
		r.view.ServeHTTP(w, httptest.NewRequest(m, "/api/currency", strings.NewReader("{}")))
		if w.Code != http.StatusMethodNotAllowed {
			t.Errorf("%s /api/currency: %d", m, w.Code)
		}
	}
}

func TestCurrencyPageHasNoThirdPartyUrl(t *testing.T) {
	r := newRig(t, "sorceress")
	url := regexp.MustCompile(`(?i)(https?:)?//[a-z0-9.-]+\.[a-z]{2,}`)
	for _, p := range []string{"/watch", "/watch/app.js", "/watch/app.css"} {
		body := r.get(p, nil).Body.String()
		if m := url.FindString(body); m != "" && !strings.HasPrefix(m, "//") { // "//" alone starts a JS comment
			t.Errorf("%s has an external reference: %s", p, m)
		}
		if strings.Contains(body, "https://") || strings.Contains(body, "http://") {
			t.Errorf("%s has a URL", p)
		}
	}
	html := r.get("/watch", nil).Body.String()
	if strings.Contains(html, "style=") || strings.Contains(html, "<style") {
		t.Error("inline style")
	}
	app := r.get("/watch/app.js", nil).Body.String()
	if !strings.Contains(app, "/api/currency") || strings.Contains(app, "innerHTML") {
		t.Error("app.js does not read /api/currency through textContent")
	}
}
