package main

import (
	"context"
	"log/slog"
	"path/filepath"
	"testing"

	"d2lod-bot/internal/bot"
	"d2lod-bot/internal/d2"
	"d2lod-bot/internal/d2dctl/fake"
	"d2lod-bot/internal/d2denv"
	"d2lod-bot/internal/data"
	"d2lod-bot/internal/env"
	"d2lod-bot/internal/invdesk"
	"d2lod-bot/internal/status"
)

// The production wiring (buildD2DStack -> wireInv) installs the tidy tooltip source on the inventory desk, so the
// Tidy hook is reachable in production and not only in the town package's tests.
func TestTidyHookIsWiredIntoTheDesk(t *testing.T) {
	dir := t.TempDir()
	sock := filepath.Join(dir, "d.sock")
	srv, err := fake.New(sock, fake.Sample(), fake.Options{})
	if err != nil {
		t.Fatal(err)
	}
	defer srv.Close()
	db, err := data.Load(synthTables(t))
	if err != nil {
		t.Fatal(err)
	}
	tune := env.DefaultTuning()
	be, err := d2denv.Build(context.Background(), d2denv.Config{ID: "t", Log: slog.New(slog.DiscardHandler), Tune: &tune, Data: db, Sock: sock})
	if err != nil {
		t.Fatal(err)
	}
	defer be.Close()
	desk, err := invdesk.New(invdesk.Config{Inventory: func() status.InventoryData { return status.InventoryData{} },
		Snap: func() *d2.Snapshot { return nil }})
	if err != nil {
		t.Fatal(err)
	}
	if desk.HasTidy() {
		t.Fatal("the desk starts without a tidy source")
	}
	buildD2DStack(context.Background(), options{char: "Fake"}, be, be.Navigator(), db, bot.Config{}, &runnerRef{inv: desk}, slog.New(slog.DiscardHandler))
	if !desk.HasTidy() {
		t.Error("buildD2DStack did not install the tidy source on the desk")
	}
}
