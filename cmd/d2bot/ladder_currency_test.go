package main

import (
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"d2lod-bot/internal/goals"
	"d2lod-bot/internal/watch"
)

func startCurrencyRig(t *testing.T, lo ladderOptions) *ladderRig {
	t.Helper()
	ctx, cancel := context.WithCancel(context.Background())
	t.Cleanup(cancel)
	called := 0
	lo.on, lo.watch, lo.viewAddr, lo.answerAddr = true, true, "127.0.0.1:0", "127.0.0.1:0"
	rig, err := startLadder(ctx, lo, "", "", fakeEnable(t, &watch.SimClock{}, &called), goalManager(t),
		func() goals.Status { return goals.Status{State: goals.RunnerRunning} }, func() string { return "Rogue Encampment" }, nil)
	if err != nil || rig == nil || rig.Hub == nil {
		t.Fatalf("%v %v", rig, err)
	}
	return rig
}

// Without --overlay-dir the screen has no currency route and no config flag; with it the strip is served.
func TestLadderCurrencyOnlyInDidleMode(t *testing.T) {
	rig := startCurrencyRig(t, ladderOptions{})
	view := "http://" + rig.Served.View.String()
	if rig.Wallet != nil {
		t.Fatal("vanilla run has a wallet")
	}
	if code := httpDo(t, "GET", view+"/api/currency", nil, nil); code != 404 {
		t.Errorf("vanilla /api/currency: %d", code)
	}
	var cfg watch.ConfigView
	httpDo(t, "GET", view+"/api/config", nil, &cfg)
	if cfg.Currency {
		t.Error("vanilla config announces currency")
	}

	ov := t.TempDir()
	if err := os.MkdirAll(filepath.Join(ov, "excel"), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(ov, "excel", "Misc.txt"), []byte("code\tname\nzmk\tMark\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	rig = startCurrencyRig(t, ladderOptions{overlayDir: ov, currencyDir: filepath.Join(t.TempDir(), "currency")})
	view = "http://" + rig.Served.View.String()
	if rig.Wallet == nil || !strings.HasPrefix(rig.Wallet.Mode, "didle:") {
		t.Fatalf("wallet %+v", rig.Wallet)
	}
	rig.Wallet.SetCarried("Booti", "mark", 3)
	var cv watch.CurrencyView
	if code := httpDo(t, "GET", view+"/api/currency", nil, &cv); code != 200 || !cv.Enabled || !strings.Contains(cv.Strip, "Raven's Mark 3 in pack") {
		t.Fatalf("d.idle /api/currency: %d %+v", code, cv)
	}
	httpDo(t, "GET", view+"/api/config", nil, &cfg)
	if !cfg.Currency {
		t.Error("d.idle config does not announce currency")
	}
	if _, err := newWallet(ladderOptions{overlayDir: filepath.Join(ov, "missing"), currencyDir: t.TempDir()}); err == nil {
		t.Error("a bad overlay dir was accepted")
	}
}
